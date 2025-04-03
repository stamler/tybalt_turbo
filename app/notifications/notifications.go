package notifications

import (
	"log"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type Notification struct {
	Id                 string    `db:"id"`
	Recipient          string    `db:"recipient"`
	Template           string    `db:"template"`
	Status             string    `db:"status"`
	StatusUpdated      time.Time `db:"status_updated"`
	Error              string    `db:"error"`
	UserId             string    `db:"user"`
	SystemNotification bool      `db:"system_notification"`
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

// Send all pending notifications from the notifications collection in a
// transaction. First we'll get all entries from the notifications collection
// that are pending. Then for each one we'll send a notification to the
// recipient. We will aggregate the results and any errors then update the
// status of the notification to either sent or failed, along with any
// corresponding error message. A notification will be sent to each destination
// specified in the notification_types property on the user's profile. the
// notification_types property is an array of strings that refer to columns in
// the notification_templates collection. For now, options will be "email_text"
// and "email_html". Since a user will only want to receive one email at a time,
// if both email_text and email_html are specified, only send the email_text.
func SendNotifications(app core.App, e *core.RecordEvent) error {

	err := app.RunInTransaction(func(txApp core.App) error {
		// get all notifications that are pending
		notifications := []Notification{}
		err := txApp.DB().NewQuery(`SELECT 
    n.*,
    (r_profile.given_name || ' ' || r_profile.surname) AS RecipientName,
    (u_profile.given_name || ' ' || u_profile.surname) AS UserName,
    nt.subject,
    nt.text_email
FROM notifications n
LEFT JOIN profiles r_profile ON n.recipient = r_profile.uid
LEFT JOIN profiles u_profile ON n.user = u_profile.uid
LEFT JOIN notification_templates nt ON n.template = nt.id
WHERE n.status = 'pending'`).All(&notifications)
		if err != nil {
			return err
		}

		// for each notification, send a notification to the recipient
		for _, notification := range notifications {
			log.Println("sending notification to", notification.Recipient)
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
