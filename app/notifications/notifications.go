package notifications

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/mail"
	"strings"
	"text/template"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
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
	Data               []byte `db:"data"`
	parsedData         map[string]any
}

func appURL(app core.App) string {
	return strings.TrimRight(strings.TrimSpace(app.Settings().Meta.AppURL), "/")
}

// BuildActionURL converts an absolute app-relative path into an absolute URL
// rooted at app.Settings().Meta.AppURL.
func BuildActionURL(app core.App, path string) string {
	base := appURL(app)
	if base == "" {
		return ""
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return base
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	return base + cleanPath
}

func unresolvedLegacyPlaceholder(text string) string {
	if strings.Contains(text, "{APP_URL}") {
		return "{APP_URL}"
	}
	if strings.Contains(text, "{:RECORD_ID}") {
		return "{:RECORD_ID}"
	}
	return ""
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

// updateNotificationStatus handles updating the notification status after sending.
// It uses NonconcurrentDB() with raw SQL to avoid PocketBase hook side-effects
// (this runs in an async goroutine and must not trigger model/record events) while
// still using the serialized writer that prevents "database is locked" errors.
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

// sendNotificationByID fetches a single pending notification by ID, renders its
// template, transitions it to inflight inside a transaction, and sends the email
// asynchronously in a goroutine. It is the shared workhorse used by both
// SendNextPendingNotification (which picks the next pending row) and
// SendNotificationByID (which targets a specific row).
func sendNotificationByID(app core.App, notificationID string) error {
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
				return nil
			}
			return fmt.Errorf("error fetching notification %s: %v", notificationID, err)
		}

		// unmarshal the json data if it exists
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

		// populate the text template
		textTemplate, err := template.New("text_email").Option("missingkey=error").Parse(notification.Template)
		if err != nil {
			return fmt.Errorf("error parsing text template for notification %s: %s", notification.Id, err)
		}

		// Create a map to hold all template data
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

		// Merge the custom data from the 'Data' field using maps.Copy
		if notification.parsedData != nil {
			maps.Copy(templateData, notification.parsedData)
		}

		// execute the text template
		var text bytes.Buffer
		err = textTemplate.Execute(&text, templateData)
		if err != nil {
			return fmt.Errorf("error executing text template for notification %s: %s", notification.Id, err)
		}
		if unresolved := unresolvedLegacyPlaceholder(text.String()); unresolved != "" {
			return fmt.Errorf("notification %s rendered with unresolved legacy placeholder %s", notification.Id, unresolved)
		}

		// create the message
		message = &mailer.Message{
			From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
			To:      []mail.Address{{Name: notification.RecipientName, Address: notification.RecipientEmail}},
			Subject: notification.Subject,
			Text:    text.String(),
		}

		// update the notification status to inflight
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

	// If the notification was found and processed, send the email asynchronously.
	// notification.Id is empty when the query returned sql.ErrNoRows (already
	// sent or does not exist), so we skip silently.
	if notification.Id != "" {
		go func(app core.App, message *mailer.Message, notification Notification) {
			// Recover from panics that can occur when the app shuts down
			// (or a test cleans up) while this goroutine is still running.
			defer func() {
				if r := recover(); r != nil {
					// Best-effort log; the logger itself may be torn down.
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

// SendNotificationByID sends a single pending notification identified by its
// record ID. It is safe to call even if the notification has already been sent
// (the call is a no-op in that case).
func SendNotificationByID(app core.App, notificationID string) error {
	return sendNotificationByID(app, notificationID)
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
		return 0, fmt.Errorf("error counting pending notifications: %v", err)
	}
	if countResult.Count == 0 {
		return 0, nil
	}

	// Find the next pending notification ID.
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

	if err := sendNotificationByID(app, idResult.ID); err != nil {
		return countResult.Count, err
	}

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

// CreateNotification creates a notification record with the given template code,
// recipient, and optional data. The notification is left in pending status for
// later delivery by a cron job or explicit SendNotifications call.
func CreateNotification(app core.App, templateCode string, recipientUID string, data map[string]any, system bool) error {
	_, err := createNotificationWithUser(app, templateCode, recipientUID, data, system, "")
	return err
}

// CreateNotificationWithUser creates a notification record with the given
// template code, recipient, optional data, system flag, and optional actor user
// ID. The notification is left in pending status. Returns the notification
// record ID on success (empty string when skipped because the feature is
// disabled).
func CreateNotificationWithUser(app core.App, templateCode string, recipientUID string, data map[string]any, system bool, actorUID string) (string, error) {
	return createNotificationWithUser(app, templateCode, recipientUID, data, system, actorUID)
}

// CreateAndSendNotification creates a notification and immediately sends it.
// The send is asynchronous (fire-and-forget goroutine) so the caller is never
// blocked by email delivery. If the notification feature is disabled or the
// record cannot be created, the error is returned before any send attempt.
func CreateAndSendNotification(app core.App, templateCode string, recipientUID string, data map[string]any, system bool) error {
	return CreateAndSendNotificationWithUser(app, templateCode, recipientUID, data, system, "")
}

// CreateAndSendNotificationWithUser creates a notification and immediately
// sends it. The record creation and template rendering are synchronous but the
// actual email delivery happens asynchronously in a background goroutine, so
// the caller is never blocked by SMTP.
func CreateAndSendNotificationWithUser(app core.App, templateCode string, recipientUID string, data map[string]any, system bool, actorUID string) error {
	notificationID, err := createNotificationWithUser(app, templateCode, recipientUID, data, system, actorUID)
	if err != nil {
		return err
	}
	if notificationID == "" {
		// Notification was skipped (feature disabled or config error).
		return nil
	}
	// sendNotificationByID runs the DB transaction synchronously (fast) and
	// spawns a goroutine only for the actual SMTP send, so we call it inline.
	if err := sendNotificationByID(app, notificationID); err != nil {
		app.Logger().Error(
			"failed to send notification immediately after creation",
			"notification_id", notificationID,
			"template_code", templateCode,
			"error", err,
		)
		// The notification was still created; log the send error but don't
		// propagate it so that the business operation (PO creation, etc.)
		// is not blocked by email delivery failures.
	}
	return nil
}

// createNotificationWithUser is the internal implementation shared by all Create
// variants. It returns the notification record ID on success (empty string when
// the notification was skipped because the feature is disabled).
func createNotificationWithUser(app core.App, templateCode string, recipientUID string, data map[string]any, system bool, actorUID string) (string, error) {
	enabled, err := utilities.IsNotificationFeatureEnabled(app, templateCode)
	if err != nil {
		// Intentionally fail closed for notification features:
		// if config cannot be read (for example DB/query error), we skip notification
		// creation and return (id="", err=nil). This keeps business workflows
		// non-blocking (PO approval/rejection, etc.) while ensuring we never send a
		// notification unless explicitly enabled.
		//
		// Important: callers cannot distinguish "disabled by config" from
		// "config read failure" via return error; use logs to investigate.
		app.Logger().Error(
			"error reading notifications feature config; skipping notification (fail-closed)",
			"template_code", templateCode,
			"error", err,
		)
		return "", nil
	}
	if !enabled {
		app.Logger().Info(
			"notification creation skipped because feature is disabled",
			"template_code", templateCode,
			"recipient_uid", recipientUID,
		)
		return "", nil
	}

	notificationCollection, err := app.FindCollectionByNameOrId("notifications")
	if err != nil {
		app.Logger().Error(
			"error finding notifications collection",
			"error", err,
		)
		return "", fmt.Errorf("error finding notifications collection: %v", err)
	}

	notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
		"code": templateCode,
	})
	if err != nil {
		app.Logger().Error(
			"error finding notification template",
			"template_code", templateCode,
			"error", err,
		)
		return "", fmt.Errorf("error finding notification template %s: %v", templateCode, err)
	}

	notificationRecord := core.NewRecord(notificationCollection)
	notificationRecord.Set("recipient", recipientUID)
	notificationRecord.Set("template", notificationTemplate.Get("id"))
	notificationRecord.Set("subject", notificationTemplate.Get("subject"))
	notificationRecord.Set("text_email", notificationTemplate.Get("text_email"))
	notificationRecord.Set("status", "pending")
	notificationRecord.Set("system_notification", system)
	if actorUID != "" {
		notificationRecord.Set("user", actorUID)
	}

	// If data is provided, marshal it to JSON and store it
	if len(data) > 0 {
		dataJSON, err := json.Marshal(data)
		if err != nil {
			app.Logger().Error(
				"error marshaling notification data",
				"error", err,
			)
			return "", fmt.Errorf("error marshaling notification data: %v", err)
		}
		notificationRecord.Set("data", string(dataJSON))
	}

	err = app.Save(notificationRecord)
	if err != nil {
		app.Logger().Error(
			"error saving notification",
			"template_code", templateCode,
			"recipient_uid", recipientUID,
			"error", err,
		)
		return "", fmt.Errorf("error saving notification: %v", err)
	}

	return notificationRecord.Id, nil
}

// QueueSecondApproverNotifications will create a notification for each user (id
// column) in `pending_items_for_qualified_po_second_approvers` who has a
// `num_pos_qualified` > 0.
func QueuePoSecondApproverNotifications(app core.App, send bool) error {
	// query the `pending_items_for_qualified_po_second_approvers` view
	records, err := app.FindRecordsByFilter(
		"pending_items_for_qualified_po_second_approvers",
		"num_pos_qualified > 0",
		"",
		0,
		0,
	)
	if err != nil {
		app.Logger().Error(
			"error querying pending_items_for_qualified_po_second_approvers",
			"error", err,
		)
		return fmt.Errorf("error querying pending_items_for_qualified_po_second_approvers: %v", err)
	}

	// create a notification for each user
	createdCount := 0
	for _, record := range records {
		recipientID := record.GetString("id")
		notificationID, err := createNotificationWithUser(app, "po_second_approval_required", recipientID, map[string]any{
			"ActionURL": BuildActionURL(app, "/pos/list"),
		}, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating second approval notification",
				"error", err,
				"recipient_id", recipientID,
			)
			return fmt.Errorf("error saving second approval notification: %v", err)
		}
		if notificationID != "" {
			createdCount++
		}
	}

	app.Logger().Info(
		"queued second approval notifications",
		"candidate_count", len(records),
		"created_count", createdCount,
	)

	// Send the notifications if the send flag is true
	if send {
		sentCount, err := SendNotifications(app)
		if err != nil {
			return fmt.Errorf("error sending notifications: %v", err)
		}
		app.Logger().Info(
			"sent all notifications",
			"sent_count", sentCount,
		)
	}
	return nil
}

// QueueTimesheetSubmissionReminders creates notifications for users who haven't submitted
// their timesheet for the previous week. The previous week ending is calculated as the
// Saturday that was 7 days before today's week ending.
func QueueTimesheetSubmissionReminders(app core.App, send bool) error {
	// Calculate the previous week ending (7 days before today's week ending)
	today := time.Now()
	todayStr := today.Format(time.DateOnly)
	weekEnding, err := utilities.GenerateWeekEnding(todayStr)
	if err != nil {
		return fmt.Errorf("error calculating week ending: %v", err)
	}

	weekEndingTime, err := time.Parse(time.DateOnly, weekEnding)
	if err != nil {
		return fmt.Errorf("error parsing week ending: %v", err)
	}

	previousWeekEnding := weekEndingTime.AddDate(0, 0, -7).Format(time.DateOnly)

	return QueueTimesheetSubmissionRemindersForWeek(app, previousWeekEnding, send)
}

// QueueTimesheetSubmissionRemindersForWeek creates notifications for users who haven't submitted
// their timesheet for the specified week ending.
func QueueTimesheetSubmissionRemindersForWeek(app core.App, weekEnding string, send bool) error {
	// Find users who should have submitted a timesheet but haven't
	// This query is based on the createTimesheetMissingHandler logic
	query := `
		SELECT DISTINCT
			u.id AS uid
		FROM users u
		LEFT JOIN time_sheets ts ON ts.uid = u.id AND ts.week_ending = {:week_ending} AND ts.submitted = 1
		LEFT JOIN admin_profiles ap ON ap.uid = u.id
		WHERE ts.id IS NULL
		  AND COALESCE(ap.time_sheet_expected, 0) = 1
	`

	type UserRow struct {
		UID string `db:"uid"`
	}

	var users []UserRow
	err := app.DB().NewQuery(query).Bind(dbx.Params{
		"week_ending": weekEnding,
	}).All(&users)
	if err != nil {
		app.Logger().Error(
			"error querying users missing timesheets",
			"week_ending", weekEnding,
			"error", err,
		)
		return fmt.Errorf("error querying users missing timesheets: %v", err)
	}

	// Check for existing notifications to avoid duplicates
	notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
		"code": "timesheet_submission_reminder",
	})
	if err != nil {
		return fmt.Errorf("error finding notification template: %v", err)
	}

	createdCount := 0
	for _, user := range users {
		// Check if a notification already exists for this recipient, template, and week ending
		existingQuery := `
			SELECT COUNT(*) AS count
			FROM notifications n
			WHERE n.recipient = {:recipient}
			  AND n.template = {:template}
			  AND n.status IN ('pending', 'inflight')
			  AND json_extract(n.data, '$.WeekEnding') = {:week_ending}
		`

		type CountResult struct {
			Count int `db:"count"`
		}
		var countResult CountResult
		err = app.DB().NewQuery(existingQuery).Bind(dbx.Params{
			"recipient":   user.UID,
			"template":    notificationTemplate.Get("id"),
			"week_ending": weekEnding,
		}).One(&countResult)
		if err != nil {
			app.Logger().Error(
				"error checking for existing notification",
				"recipient", user.UID,
				"error", err,
			)
			continue
		}

		if countResult.Count > 0 {
			// Skip if notification already exists
			continue
		}

		// Create notification with week ending in data
		data := map[string]any{
			"WeekEnding": weekEnding,
			"ActionURL":  BuildActionURL(app, "/time/entries/list"),
		}
		notificationID, err := createNotificationWithUser(app, "timesheet_submission_reminder", user.UID, data, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating timesheet submission reminder",
				"recipient", user.UID,
				"error", err,
			)
			// Continue with other users even if one fails
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
	}

	app.Logger().Info(
		"queued timesheet submission reminders",
		"week_ending", weekEnding,
		"created_count", createdCount,
	)

	// Send the notifications if the send flag is true
	if send {
		sentCount, err := SendNotifications(app)
		if err != nil {
			return fmt.Errorf("error sending notifications: %v", err)
		}
		app.Logger().Info(
			"sent timesheet submission reminder notifications",
			"sent_count", sentCount,
		)
	}
	return nil
}

// QueueTimesheetApprovalReminders creates notifications for managers who have pending
// timesheets awaiting approval.
func QueueTimesheetApprovalReminders(app core.App, send bool) error {
	// Find managers with pending timesheets
	query := `
		SELECT DISTINCT
			ts.approver AS manager_uid
		FROM time_sheets ts
		WHERE ts.submitted = 1
		  AND ts.approved = ''
		  AND ts.committed = ''
		  AND ts.rejected = ''
		  AND ts.approver != ''
	`

	type ManagerRow struct {
		ManagerUID string `db:"manager_uid"`
	}

	var managers []ManagerRow
	err := app.DB().NewQuery(query).All(&managers)
	if err != nil {
		app.Logger().Error(
			"error querying managers with pending timesheets",
			"error", err,
		)
		return fmt.Errorf("error querying managers with pending timesheets: %v", err)
	}

	// Check for existing notifications to avoid duplicates (within last 24 hours)
	notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
		"code": "timesheet_approval_reminder",
	})
	if err != nil {
		return fmt.Errorf("error finding notification template: %v", err)
	}

	createdCount := 0
	for _, manager := range managers {
		// Check if a notification already exists for this manager today
		existingQuery := `
			SELECT COUNT(*) AS count
			FROM notifications n
			WHERE n.recipient = {:recipient}
			  AND n.template = {:template}
			  AND n.status IN ('pending', 'inflight')
			  AND datetime(n.created) > datetime('now', '-1 day')
		`

		type CountResult struct {
			Count int `db:"count"`
		}
		var countResult CountResult
		err = app.DB().NewQuery(existingQuery).Bind(dbx.Params{
			"recipient": manager.ManagerUID,
			"template":  notificationTemplate.Get("id"),
		}).One(&countResult)
		if err != nil {
			app.Logger().Error(
				"error checking for existing notification",
				"recipient", manager.ManagerUID,
				"error", err,
			)
			continue
		}

		if countResult.Count > 0 {
			// Skip if notification already exists within last 24 hours
			continue
		}

		notificationID, err := createNotificationWithUser(app, "timesheet_approval_reminder", manager.ManagerUID, map[string]any{
			"ActionURL": BuildActionURL(app, "/time/sheets/pending"),
		}, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating timesheet approval reminder",
				"recipient", manager.ManagerUID,
				"error", err,
			)
			// Continue with other managers even if one fails
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
	}

	app.Logger().Info(
		"queued timesheet approval reminders",
		"created_count", createdCount,
	)

	// Send the notifications if the send flag is true
	if send {
		sentCount, err := SendNotifications(app)
		if err != nil {
			return fmt.Errorf("error sending notifications: %v", err)
		}
		app.Logger().Info(
			"sent timesheet approval reminder notifications",
			"sent_count", sentCount,
		)
	}
	return nil
}

// QueueExpenseApprovalReminders creates notifications for managers who have pending
// expenses awaiting approval.
func QueueExpenseApprovalReminders(app core.App, send bool) error {
	// Find managers with pending expenses
	query := `
		SELECT DISTINCT
			e.approver AS manager_uid
		FROM expenses e
		WHERE e.submitted = 1
		  AND e.approved = ''
		  AND e.committed = ''
		  AND e.rejected = ''
		  AND e.approver != ''
	`

	type ManagerRow struct {
		ManagerUID string `db:"manager_uid"`
	}

	var managers []ManagerRow
	err := app.DB().NewQuery(query).All(&managers)
	if err != nil {
		app.Logger().Error(
			"error querying managers with pending expenses",
			"error", err,
		)
		return fmt.Errorf("error querying managers with pending expenses: %v", err)
	}

	// Check for existing notifications to avoid duplicates (within last 24 hours)
	notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
		"code": "expense_approval_reminder",
	})
	if err != nil {
		return fmt.Errorf("error finding notification template: %v", err)
	}

	createdCount := 0
	for _, manager := range managers {
		// Check if a notification already exists for this manager today
		existingQuery := `
			SELECT COUNT(*) AS count
			FROM notifications n
			WHERE n.recipient = {:recipient}
			  AND n.template = {:template}
			  AND n.status IN ('pending', 'inflight')
			  AND datetime(n.created) > datetime('now', '-1 day')
		`

		type CountResult struct {
			Count int `db:"count"`
		}
		var countResult CountResult
		err = app.DB().NewQuery(existingQuery).Bind(dbx.Params{
			"recipient": manager.ManagerUID,
			"template":  notificationTemplate.Get("id"),
		}).One(&countResult)
		if err != nil {
			app.Logger().Error(
				"error checking for existing notification",
				"recipient", manager.ManagerUID,
				"error", err,
			)
			continue
		}

		if countResult.Count > 0 {
			// Skip if notification already exists within last 24 hours
			continue
		}

		notificationID, err := createNotificationWithUser(app, "expense_approval_reminder", manager.ManagerUID, map[string]any{
			"ActionURL": BuildActionURL(app, "/expenses/pending"),
		}, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating expense approval reminder",
				"recipient", manager.ManagerUID,
				"error", err,
			)
			// Continue with other managers even if one fails
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
	}

	app.Logger().Info(
		"queued expense approval reminders",
		"created_count", createdCount,
	)

	// Send the notifications if the send flag is true
	if send {
		sentCount, err := SendNotifications(app)
		if err != nil {
			return fmt.Errorf("error sending notifications: %v", err)
		}
		app.Logger().Info(
			"sent expense approval reminder notifications",
			"sent_count", sentCount,
		)
	}
	return nil
}

// QueueTimesheetRejectedNotifications creates notifications for timesheet rejection.
// Recipients include: the employee, the rejector, and the employee's manager (if different from rejector).
func QueueTimesheetRejectedNotifications(app core.App, timesheet *core.Record, rejectorUID, reason string) error {
	employeeUID := timesheet.GetString("uid")
	weekEnding := timesheet.GetString("week_ending")

	// Load employee profile
	employeeProfile, err := app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": employeeUID,
	})
	if err != nil {
		app.Logger().Error(
			"error finding employee profile",
			"employee_uid", employeeUID,
			"error", err,
		)
		return fmt.Errorf("error finding employee profile: %v", err)
	}
	employeeName := employeeProfile.GetString("given_name") + " " + employeeProfile.GetString("surname")

	// Load rejector profile
	rejectorProfile, err := app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": rejectorUID,
	})
	if err != nil {
		app.Logger().Error(
			"error finding rejector profile",
			"rejector_uid", rejectorUID,
			"error", err,
		)
		return fmt.Errorf("error finding rejector profile: %v", err)
	}
	rejectorName := rejectorProfile.GetString("given_name") + " " + rejectorProfile.GetString("surname")

	// Load employee's manager
	managerUID := employeeProfile.GetString("manager")

	// Build notification data
	data := map[string]any{
		"EmployeeName":    employeeName,
		"WeekEnding":      weekEnding,
		"RejectorName":    rejectorName,
		"RejectionReason": reason,
		"ActionURL":       BuildActionURL(app, fmt.Sprintf("/time/sheets/%s/details", timesheet.Id)),
	}

	// Determine recipients: employee, rejector, and manager (if different from rejector)
	recipients := []string{employeeUID}
	if rejectorUID != employeeUID {
		recipients = append(recipients, rejectorUID)
	}
	if managerUID != "" && managerUID != rejectorUID && managerUID != employeeUID {
		recipients = append(recipients, managerUID)
	}

	// Create and immediately send notifications for each recipient
	createdCount := 0
	for _, recipientUID := range recipients {
		notificationID, err := createNotificationWithUser(app, "timesheet_rejected", recipientUID, data, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating timesheet rejection notification",
				"recipient", recipientUID,
				"error", err,
			)
			// Continue with other recipients even if one fails
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
		if err := sendNotificationByID(app, notificationID); err != nil {
			app.Logger().Error(
				"failed to send timesheet rejection notification immediately after creation",
				"notification_id", notificationID,
				"error", err,
			)
		}
	}

	app.Logger().Info(
		"created timesheet rejection notifications",
		"timesheet_id", timesheet.Id,
		"created_count", createdCount,
		"recipient_count", len(recipients),
	)

	return nil
}

// QueueExpenseRejectedNotifications creates notifications for expense rejection.
// Recipients include: the employee, the rejector, and the employee's manager (if different from rejector).
func QueueExpenseRejectedNotifications(app core.App, expense *core.Record, rejectorUID, reason string) error {
	employeeUID := expense.GetString("uid")
	expenseDate := expense.GetString("date")
	expenseTotal := expense.GetFloat("total")

	// Format expense amount as currency string (2 decimal places)
	expenseAmount := fmt.Sprintf("$%.2f", expenseTotal)

	// Load employee profile
	employeeProfile, err := app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": employeeUID,
	})
	if err != nil {
		app.Logger().Error(
			"error finding employee profile",
			"employee_uid", employeeUID,
			"error", err,
		)
		return fmt.Errorf("error finding employee profile: %v", err)
	}
	employeeName := employeeProfile.GetString("given_name") + " " + employeeProfile.GetString("surname")

	// Load rejector profile
	rejectorProfile, err := app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": rejectorUID,
	})
	if err != nil {
		app.Logger().Error(
			"error finding rejector profile",
			"rejector_uid", rejectorUID,
			"error", err,
		)
		return fmt.Errorf("error finding rejector profile: %v", err)
	}
	rejectorName := rejectorProfile.GetString("given_name") + " " + rejectorProfile.GetString("surname")

	// Load employee's manager
	managerUID := employeeProfile.GetString("manager")

	// Build notification data
	data := map[string]any{
		"EmployeeName":    employeeName,
		"ExpenseDate":     expenseDate,
		"ExpenseAmount":   expenseAmount,
		"RejectorName":    rejectorName,
		"RejectionReason": reason,
		"ActionURL":       BuildActionURL(app, fmt.Sprintf("/expenses/%s/details", expense.Id)),
	}

	// Determine recipients: employee, rejector, and manager (if different from rejector)
	recipients := []string{employeeUID}
	if rejectorUID != employeeUID {
		recipients = append(recipients, rejectorUID)
	}
	if managerUID != "" && managerUID != rejectorUID && managerUID != employeeUID {
		recipients = append(recipients, managerUID)
	}

	// Create and immediately send notifications for each recipient
	createdCount := 0
	for _, recipientUID := range recipients {
		notificationID, err := createNotificationWithUser(app, "expense_rejected", recipientUID, data, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating expense rejection notification",
				"recipient", recipientUID,
				"error", err,
			)
			// Continue with other recipients even if one fails
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
		if err := sendNotificationByID(app, notificationID); err != nil {
			app.Logger().Error(
				"failed to send expense rejection notification immediately after creation",
				"notification_id", notificationID,
				"error", err,
			)
		}
	}

	app.Logger().Info(
		"created expense rejection notifications",
		"expense_id", expense.Id,
		"created_count", createdCount,
		"recipient_count", len(recipients),
	)

	return nil
}

// QueueTimesheetSharedNotifications creates notifications for newly added timesheet reviewers.
func QueueTimesheetSharedNotifications(app core.App, timesheet *core.Record, sharerUID string, newViewerUIDs []string) error {
	if len(newViewerUIDs) == 0 {
		return nil
	}

	employeeUID := timesheet.GetString("uid")
	weekEnding := timesheet.GetString("week_ending")

	// Load sharer profile
	sharerProfile, err := app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": sharerUID,
	})
	if err != nil {
		app.Logger().Error(
			"error finding sharer profile",
			"sharer_uid", sharerUID,
			"error", err,
		)
		return fmt.Errorf("error finding sharer profile: %v", err)
	}
	sharerName := sharerProfile.GetString("given_name") + " " + sharerProfile.GetString("surname")

	// Load employee profile
	employeeProfile, err := app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": employeeUID,
	})
	if err != nil {
		app.Logger().Error(
			"error finding employee profile",
			"employee_uid", employeeUID,
			"error", err,
		)
		return fmt.Errorf("error finding employee profile: %v", err)
	}
	employeeName := employeeProfile.GetString("given_name") + " " + employeeProfile.GetString("surname")

	// Build notification data
	data := map[string]any{
		"UserName":     sharerName,
		"EmployeeName": employeeName,
		"WeekEnding":   weekEnding,
		"ActionURL":    BuildActionURL(app, fmt.Sprintf("/time/sheets/%s/details", timesheet.Id)),
	}

	// Create and immediately send notifications for each new viewer
	createdCount := 0
	for _, viewerUID := range newViewerUIDs {
		notificationID, err := createNotificationWithUser(app, "timesheet_shared", viewerUID, data, true, "")
		if err != nil {
			app.Logger().Error(
				"error creating timesheet shared notification",
				"viewer_uid", viewerUID,
				"error", err,
			)
			// Continue with other viewers even if one fails
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
		if err := sendNotificationByID(app, notificationID); err != nil {
			app.Logger().Error(
				"failed to send timesheet shared notification immediately after creation",
				"notification_id", notificationID,
				"error", err,
			)
		}
	}

	app.Logger().Info(
		"created timesheet shared notifications",
		"timesheet_id", timesheet.Id,
		"created_count", createdCount,
	)

	return nil
}
