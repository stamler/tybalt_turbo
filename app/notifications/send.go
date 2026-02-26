// send.go contains notification delivery internals.
//
// It implements the send engine for pending notifications, including record
// fetch/render, status transitions (pending -> inflight -> sent/error),
// targeted send by ID, and queue draining.
package notifications

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/mail"
	"text/template"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func updateNotificationStatus(app core.App, notification Notification, sendErr error) {
	status := "sent"
	errMsg := ""
	if sendErr != nil {
		status = "error"
		errMsg = sendErr.Error()
	}

	if _, err := app.NonconcurrentDB().NewQuery(
		"UPDATE notifications SET status = {:status}, error = {:error}, status_updated = {:status_updated} WHERE id = {:id}",
	).Bind(dbx.Params{
		"status":         status,
		"error":          errMsg,
		"status_updated": time.Now().UTC().Format("2006-01-02 15:04:05.000Z"),
		"id":             notification.Id,
	}).Execute(); err != nil {
		app.Logger().Error(
			"Failed to update notification status",
			"notification_id", notification.Id,
			"intended_status", status,
			"error", err,
		)
	}
}

// SendNotificationByID sends a single pending notification identified by its
// notification record ID.
//
// If the record does not exist or is no longer pending, this is a no-op and
// returns nil.
func SendNotificationByID(app core.App, notificationID string) error {
	notification := Notification{}
	message := &mailer.Message{}
	err := app.RunInTransaction(func(txApp core.App) error {
		err := txApp.DB().NewQuery(`SELECT
				n.*,
				(r_profile.given_name || ' ' || r_profile.surname) AS recipient_name,
				u.email,
				r_profile.notification_type,
				COALESCE(u_profile.given_name || ' ' || u_profile.surname, '') AS user_name,
				nt.subject,
				nt.text_email,
				n.data
			FROM notifications n
			LEFT JOIN profiles r_profile ON n.recipient = r_profile.uid
			LEFT JOIN profiles u_profile ON n.user = u_profile.uid
			LEFT JOIN notification_templates nt ON n.template = nt.id
			LEFT JOIN users u ON n.recipient = u.id
			WHERE n.id = {:id}
			  AND n.status = 'pending'`).Bind(dbx.Params{
			"id": notificationID,
		}).One(&notification)
		if err != nil {
			if err == sql.ErrNoRows {
				// Intentional no-op when record is missing or already not pending.
				return nil
			}
			return fmt.Errorf("error fetching notification %s: %v", notificationID, err)
		}

		if len(notification.Data) > 0 {
			err = json.Unmarshal(notification.Data, &notification.parsedData)
			if err != nil {
				app.Logger().Error(
					"Failed to unmarshal notification data",
					"notification_id", notification.Id,
					"error", err,
					"raw_data", string(notification.Data),
				)
				return fmt.Errorf("error unmarshalling notification data for %s: %w", notification.Id, err)
			}
		}

		textTemplate, err := template.New("text_email").Option("missingkey=error").Parse(notification.Template)
		if err != nil {
			return fmt.Errorf("error parsing text template for notification %s: %s", notification.Id, err)
		}

		templateData := map[string]any{
			"Id":                 notification.Id,
			"RecipientEmail":     notification.RecipientEmail,
			"RecipientName":      notification.RecipientName,
			"NotificationType":   notification.NotificationType,
			"UserName":           notification.UserName,
			"Subject":            notification.Subject,
			"Template":           notification.Template,
			"Status":             notification.Status,
			"StatusUpdated":      notification.StatusUpdated,
			"Error":              notification.Error,
			"UserId":             notification.UserId,
			"SystemNotification": notification.SystemNotification,
		}

		if notification.parsedData != nil {
			maps.Copy(templateData, notification.parsedData)
		}

		var text bytes.Buffer
		err = textTemplate.Execute(&text, templateData)
		if err != nil {
			return fmt.Errorf("error executing text template for notification %s: %s", notification.Id, err)
		}
		if unresolved := unresolvedLegacyPlaceholder(text.String()); unresolved != "" {
			return fmt.Errorf("notification %s rendered with unresolved legacy placeholder %s", notification.Id, unresolved)
		}

		message = &mailer.Message{
			From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
			To:      []mail.Address{{Name: notification.RecipientName, Address: notification.RecipientEmail}},
			Subject: notification.Subject,
			Text:    text.String(),
		}

		_, err = txApp.NonconcurrentDB().NewQuery(
			"UPDATE notifications SET status = 'inflight', status_updated = {:status_updated} WHERE id = {:id}",
		).Bind(dbx.Params{
			"status_updated": time.Now().UTC().Format("2006-01-02 15:04:05.000Z"),
			"id":             notification.Id,
		}).Execute()
		if err != nil {
			return fmt.Errorf("error updating notification status to inflight: %v", err)
		}

		return nil
	})

	if err != nil {
		return err
	}

	if notification.Id != "" {
		go func(app core.App, message *mailer.Message, notification Notification) {
			defer func() {
				if r := recover(); r != nil {
					app.Logger().Error(
						"recovered from panic while sending notification email",
						"notification_id", notification.Id,
						"panic", fmt.Sprintf("%v", r),
					)
				}
			}()
			err := app.NewMailClient().Send(message)
			if err != nil {
				app.Logger().Error(
					"Failed to send notification email",
					"notification_id", notification.Id,
					"error", err,
				)
			}
			updateNotificationStatus(app, notification, err)
		}(app, message, notification)
	}

	return nil
}

// SendNextPendingNotification sends one pending notification from the queue and
// returns the remaining pending count after the send attempt.
//
// It returns (0, nil) when there are no pending notifications. On errors, the
// returned remaining count reflects the pre-send pending count when available.
func SendNextPendingNotification(app core.App) (remaining int64, err error) {
	type CountResult struct {
		Count int64 `db:"count"`
	}
	var countResult CountResult
	err = app.DB().NewQuery("SELECT COUNT(*) AS count FROM notifications WHERE status = 'pending'").One(&countResult)
	if err != nil {
		return 0, fmt.Errorf("error counting pending notifications: %v", err)
	}
	if countResult.Count == 0 {
		return 0, nil
	}

	type IDResult struct {
		ID string `db:"id"`
	}
	var idResult IDResult
	err = app.DB().NewQuery("SELECT id FROM notifications WHERE status = 'pending' LIMIT 1").One(&idResult)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return countResult.Count, fmt.Errorf("error finding next pending notification: %v", err)
	}

	if err := SendNotificationByID(app, idResult.ID); err != nil {
		return countResult.Count, err
	}

	return countResult.Count - 1, nil
}

// SendNotifications drains the pending notification queue by repeatedly calling
// SendNextPendingNotification until none remain.
//
// It returns the number of loop iterations that completed without error. The
// actual SMTP delivery happens asynchronously inside SendNotificationByID.
func SendNotifications(app core.App) (int64, error) {
	sentCount := int64(0)
	remaining := int64(1)
	var err error

	for remaining > 0 {
		remaining, err = SendNextPendingNotification(app)
		if err != nil {
			return sentCount, err
		}
		sentCount++
	}
	return sentCount, nil
}
