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

		// update the notification status to inflight
		txApp.DB().NewQuery(fmt.Sprintf("UPDATE notifications SET status = 'inflight' WHERE id = '%s'", notification.Id)).Execute()

		return nil
	})

	// TODO: This next segment of code until the actual sending of the email could
	// potentially be moved into the transaction above so that any errors from the
	// block prevent the status from being updated to inflight. Actually it would
	// be even better to have everything in the transaction but we can't do that
	// because sending the email is a blocking operation. For this reason we will
	// execute the transaction and then send the email in a second step. In fact
	// later we will move the sending of the email to a background job so that we
	// can return immediately from this function.

	if err != nil {
		// we do not decrement the count here because we want to return the total
		// number of pending notifications, and any error from the transaction will
		// result in the status change to 'inflight' being rolled back
		// TODO: confirm this
		return countResult.Count, err
	}

	// populate the text template
	textTemplate, err := template.New("text_email").Parse(notification.Template)
	if err != nil {
		return countResult.Count - 1, fmt.Errorf("error parsing text template for notification %s: %s", notification.Id, err)
	}

	// execute the text template
	var text bytes.Buffer
	err = textTemplate.Execute(&text, notification)
	if err != nil {
		return countResult.Count - 1, fmt.Errorf("error executing text template for notification %s: %s", notification.Id, err)
	}

	// create the message
	message := &mailer.Message{
		From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
		To:      []mail.Address{{Name: notification.RecipientName, Address: notification.RecipientEmail}},
		Subject: notification.Subject,
		Text:    text.String(),
	}

	// send the notification
	err = app.NewMailClient().Send(message)
	if err != nil {
		// update the notification status to error and set the error message
		app.DB().NewQuery(fmt.Sprintf("UPDATE notifications SET status = 'error', error = '%s' WHERE id = '%s'", err.Error(), notification.Id)).Execute()
		return countResult.Count - 1, fmt.Errorf("error sending notification %s: %s", notification.Id, err)
	}

	// update the notification status to sent
	_, err = app.DB().NewQuery(fmt.Sprintf("UPDATE notifications SET status = 'sent' WHERE id = '%s'", notification.Id)).Execute()

	if err != nil {
		return countResult.Count - 1, fmt.Errorf("error setting status to sent for notification %s: %s", notification.Id, err)
	}

	return countResult.Count - 1, nil
}
