package notifications

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/mail"
	"text/template"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

type Notification struct {
	Id                 string `db:"id"`
	RecipientEmail     string `db:"email"`
	RecipientName      string `db:"recipient_name"`
	NotificationType   string `db:"notification_type"`
	UserName           string `db:"user_name"`
	Subject            string `db:"subject"`
	Template           string `db:"text_email"`
	Status             string `db:"status"`
	StatusUpdated      string `db:"status_updated"`
	Error              string `db:"error"`
	UserId             string `db:"user"`
	SystemNotification bool   `db:"system_notification"`
}

// WriteStatusUpdatedOnNotification is a hook that writes the current time to the status_updated field
// if the status field has changed or if the record is new.
func WriteStatusUpdated(app core.App, e *core.RecordEvent) error {
	// set the status_updated field to now if the value of status has changed or if the record is new
	if e.Record.Get("status") != e.Record.Original().Get("status") || e.Record.IsNew() {
		e.Record.Set("status_updated", time.Now())
	}
	return nil
}

// updateNotificationStatus handles updating the notification status after sending
func updateNotificationStatus(app core.App, notification Notification, sendErr error) {
	// Get the notification record. We must do this rather than running UPDATE
	// directly in an SQL statement because we depend on PocketBase's writer to
	// handle locking and busy-waiting. A previous version used update and was
	// causing race conditions (database was locked for writing during another
	// update). This could probably also be solved with a mutex for notification
	// status updates, but I think PocketBase's writer is somehow more
	// efficient/easier.
	record, err := app.FindRecordById("notifications", notification.Id)
	if err != nil {
		app.Logger().Error(
			"Failed to find notification record",
			"notification_id", notification.Id,
			"error", err,
		)
		return
	}

	// Update the status and error fields
	if sendErr != nil {
		record.Set("status", "error")
		record.Set("error", sendErr.Error())
	} else {
		record.Set("status", "sent")
		record.Set("error", "")
	}

	// Save the record back to the database
	if err := app.Save(record); err != nil {
		app.Logger().Error(
			"Failed to update notification status",
			"notification_id", notification.Id,
			"intended_status", map[bool]string{true: "error", false: "sent"}[sendErr != nil],
			"error", err,
		)
	}
}

// SendNextPendingNotification will send the next pending notification from the
// notifications collection in a transaction. For now, the notification_type
// will be always be "email_text" so we'll just use the text_email field of the
// notification_templates collection as the body template of the email.
func SendNextPendingNotification(app core.App) (remaining int64, err error) {
	// fast return if there are no pending notifications
	type CountResult struct {
		Count int64 `db:"count"`
	}
	var countResult CountResult
	err = app.DB().NewQuery("SELECT COUNT(*) AS count FROM notifications WHERE status = 'pending'").One(&countResult)
	if err != nil {
		return 0, err
	}
	if countResult.Count == 0 {
		return 0, nil
	}

	notification := Notification{}
	message := &mailer.Message{}
	err = app.RunInTransaction(func(txApp core.App) error {
		// get all notifications that are pending
		err := txApp.DB().NewQuery(`SELECT 
				n.*,
				(r_profile.given_name || ' ' || r_profile.surname) AS recipient_name,
				u.email,
				r_profile.notification_type,
				(u_profile.given_name || ' ' || u_profile.surname) AS user_name,
				nt.subject,
				nt.text_email
			FROM notifications n
			LEFT JOIN profiles r_profile ON n.recipient = r_profile.uid
			LEFT JOIN profiles u_profile ON n.user = u_profile.uid
			LEFT JOIN notification_templates nt ON n.template = nt.id
			LEFT JOIN users u ON n.recipient = u.id
			WHERE n.status = 'pending'
			LIMIT 1`).One(&notification)
		if err != nil {
			// if the error is that there are no more pending notifications, return nil
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}

		// populate the text template
		textTemplate, err := template.New("text_email").Parse(notification.Template)
		if err != nil {
			return fmt.Errorf("error parsing text template for notification %s: %s", notification.Id, err)
		}

		// execute the text template
		var text bytes.Buffer
		err = textTemplate.Execute(&text, notification)
		if err != nil {
			return fmt.Errorf("error executing text template for notification %s: %s", notification.Id, err)
		}

		// create the message
		message = &mailer.Message{
			From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
			To:      []mail.Address{{Name: notification.RecipientName, Address: notification.RecipientEmail}},
			Subject: notification.Subject,
			Text:    text.String(),
		}

		// update the notification status to inflight
		_, err = txApp.DB().NewQuery(fmt.Sprintf("UPDATE notifications SET status = 'inflight' WHERE id = '%s'", notification.Id)).Execute()
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		// return the total number of pending notifications, which won't change due
		// to the error since any error from the transaction will result in the
		// status change to 'inflight' being rolled back
		return countResult.Count, err
	}

	// sending the email is now non-blocking. We launch it in a goroutine and
	// update the status once it completes
	go func(app core.App, message *mailer.Message, notification Notification) {
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

	// return immediately with the decremented count since we've taken one notification
	return countResult.Count - 1, nil
}

// SendNotifications will send all notifications that are pending. It will call
// SendNextPendingNotification in a loop until there are no more pending
// notifications.
func SendNotifications(app core.App) (int64, error) {
	sentCount := int64(0)
	remaining := int64(1) // initialize greater than 0 to enter the loop
	var err error

	for remaining > 0 {
		remaining, err = SendNextPendingNotification(app)
		if err != nil {
			// if there was an error, return the remaining count and the error because
			// if remaining is greater than 0 and the next call continues to fail,
			// we'll never get out of this loop
			return sentCount, err
		}
		sentCount++
	}
	return sentCount, nil
}
