// types.go defines shared internal types for the notifications package.
//
// It centralizes transport/persistence models and config structs used by
// dispatch and reminder queue orchestration.
package notifications

import "github.com/pocketbase/dbx"

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
	Data               []byte `db:"data"`
	parsedData         map[string]any
}

type DeliveryMode string

const (
	DeliveryDeferred  DeliveryMode = "deferred"
	DeliveryImmediate DeliveryMode = "immediate"
)

type DispatchArgs struct {
	TemplateCode string
	RecipientUID string
	Data         map[string]any
	System       bool
	ActorUID     string
	Mode         DeliveryMode
}

type DedupeSpec struct {
	Where  string
	Params func(row dbx.NullStringMap) dbx.Params
}

type ReminderJob struct {
	Name         string
	TemplateCode string
	Query        string
	QueryParams  dbx.Params
	RecipientCol string

	Dedupe DedupeSpec

	BuildData func(row dbx.NullStringMap) map[string]any
	LogFields map[string]any
}
