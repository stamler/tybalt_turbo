// notifications_test.go

package main

import (
	"strings"
	"testing"
	"time"

	"tybalt/internal/testutils"
	"tybalt/notifications"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// This is the test file for the notifications package.

func upsertNotificationsConfigRawValue(t *testing.T, app core.App, rawValue string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "notifications")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "notifications")
	}

	record.Set("value", rawValue)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save notifications config: %v", err)
	}
}

// SendNextPendingNotification()

//  1. one email is sent when there are one or more pending notifications
//     notifications. The pending count is returned as 1 less than the original
//     count, no error is returned, and there is one email in the
//     TestMailer.Messages() inbox.
func TestSendNextPendingNotification_SendsOneEmail(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Call SendNextPendingNotification
	remaining, err := notifications.SendNextPendingNotification(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify remaining count is one less than original 4 pending notifications
	// in the test data.
	if remaining != 4 {
		t.Errorf("Expected remaining count to be 4, got %d", remaining)
	}

	// Sleep for 100ms to allow the goroutine called by
	// SendNextPendingNotification to complete. This is a bit of a hack, but it's
	// necessary since the email sending is async and we need to wait for it to
	// complete before checking the TestMailer.Messages() inbox. TODO: Find a
	// better way to do this, perhaps by using a channel to communicate the
	// completion of the email sending.
	time.Sleep(100 * time.Millisecond)

	messageCount := len(app.TestMailer.Messages())
	if messageCount != 1 {
		t.Errorf("Expected 1 email to be sent, got %d", messageCount)
	}
}

//  2. no emails are sent when there are no pending notifications. The pending
//     count is returned as 0 and a nil error is returned. There are no emails
//     in the TestMailer.Messages() inbox.
func TestSendNextPendingNotification_NoEmailsWhenNoPendingNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// First, clear all pending notifications
	_, err := notifications.SendNotifications(app)
	if err != nil {
		t.Fatalf("Failed to clear pending notifications: %v", err)
	}

	// Now try to send another notification when there are none pending
	remaining, err := notifications.SendNextPendingNotification(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify remaining count is 0 since there are no pending notifications
	if remaining != 0 {
		t.Errorf("Expected remaining count to be 0, got %d", remaining)
	}

	// Sleep briefly to allow any potential async operations to complete
	time.Sleep(20 * time.Millisecond)

	// Get the initial message count
	initialMessageCount := len(app.TestMailer.Messages())

	// Try to send another notification
	_, err = notifications.SendNextPendingNotification(app)
	if err != nil {
		t.Fatalf("Expected no error on second attempt, got %v", err)
	}

	// Sleep briefly to allow any potential async operations to complete
	time.Sleep(20 * time.Millisecond)

	// Verify no new messages were added to the TestMailer
	finalMessageCount := len(app.TestMailer.Messages())
	if finalMessageCount != initialMessageCount {
		t.Errorf("Expected no new messages to be sent, but message count changed from %d to %d",
			initialMessageCount, finalMessageCount)
	}
}

//  3. pending count of 0 and an error are returned if the CountResult query
//     fails
func TestSendNextPendingNotification_ErrorOnCountQuery(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Break the notifications table to force query errors
	_, err := app.NonconcurrentDB().NewQuery("ALTER TABLE notifications RENAME TO notifications_broken").Execute()
	if err != nil {
		t.Fatalf("Failed to rename notifications table: %v", err)
	}

	// Attempt to send notification with broken table
	remaining, err := notifications.SendNextPendingNotification(app)

	// Verify we got an error that contains the string "error counting pending notifications"
	if err == nil {
		t.Error("Expected an error when notifications table is missing, got nil")
	} else if !strings.Contains(err.Error(), "error counting pending notifications") {
		t.Errorf("Expected an error containing 'error counting pending notifications', got %v", err)
	}

	// Verify remaining count is 0
	if remaining != 0 {
		t.Errorf("Expected remaining count to be 0 when query fails, got %d", remaining)
	}
}

//  4. the pending count and an error are returned if fetching one notification
//     fails for a reason other than there being no pending notifications
func TestSendNextPendingNotification_ErrorOnFetch(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Break the notification_templates table to force a JOIN error
	_, err := app.NonconcurrentDB().NewQuery("ALTER TABLE notification_templates RENAME TO notification_templates_broken").Execute()
	if err != nil {
		t.Fatalf("Failed to rename notification_templates table: %v", err)
	}

	// Get initial count of pending notifications
	var countResult struct {
		Count int64 `db:"count"`
	}
	err = app.DB().NewQuery("SELECT COUNT(*) AS count FROM notifications WHERE status = 'pending'").One(&countResult)
	if err != nil {
		t.Fatalf("Failed to get initial count: %v", err)
	}
	initialCount := countResult.Count

	// Attempt to send notification with broken join
	remaining, err := notifications.SendNextPendingNotification(app)

	// Verify we got an error
	if err == nil {
		t.Error("Expected an error when notification_templates table is missing, got nil")
	} else if !strings.Contains(err.Error(), "error fetching notification") {
		t.Errorf("Expected an error containing 'error fetching notification', got %v", err)
	}

	// Verify remaining count matches initial count since the operation failed
	if remaining != initialCount {
		t.Errorf("Expected remaining count to be %d when fetch fails, got %d", initialCount, remaining)
	}
}

//  5. the pending count and an error are returned if the text template cannot
//     be parsed
func TestSendNextPendingNotification_ErrorOnInvalidTemplate(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Get initial count of pending notifications
	var countResult struct {
		Count int64 `db:"count"`
	}
	err := app.DB().NewQuery("SELECT COUNT(*) AS count FROM notifications WHERE status = 'pending'").One(&countResult)
	if err != nil {
		t.Fatalf("Failed to get initial count: %v", err)
	}
	initialCount := countResult.Count

	// Update a notification template to have invalid template syntax
	_, err = app.NonconcurrentDB().NewQuery("UPDATE notification_templates SET text_email = '{{.InvalidSyntax}'").Execute()
	if err != nil {
		t.Fatalf("Failed to update template: %v", err)
	}

	// Attempt to send notification with invalid template
	remaining, err := notifications.SendNextPendingNotification(app)

	// Verify we got an error
	if err == nil {
		t.Error("Expected an error when template is invalid, got nil")
	} else if !strings.Contains(err.Error(), "error parsing text template for notification") {
		t.Errorf("Expected an error containing 'error parsing text template for notification', got %v", err)
	}

	// Verify remaining count matches initial count since the operation failed
	if remaining != initialCount {
		t.Errorf("Expected remaining count to be %d when template parsing fails, got %d", initialCount, remaining)
	}
}

//  6. the pending count and an error are returned if template execution
//     references a missing key.
func TestSendNextPendingNotification_ErrorOnMissingTemplateKey(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	var target struct {
		ID         string `db:"id"`
		TemplateID string `db:"template"`
	}
	if err := app.DB().NewQuery(`
		SELECT id, template
		FROM notifications
		LIMIT 1
	`).One(&target); err != nil {
		t.Fatalf("Failed to load target notification: %v", err)
	}

	_, err := app.NonconcurrentDB().NewQuery("UPDATE notifications SET status = 'sent'").Execute()
	if err != nil {
		t.Fatalf("Failed to normalize notification statuses: %v", err)
	}
	_, err = app.NonconcurrentDB().NewQuery(`
		UPDATE notifications
		SET status = 'pending'
		WHERE id = {:id}
	`).Bind(dbx.Params{
		"id": target.ID,
	}).Execute()
	if err != nil {
		t.Fatalf("Failed to mark target notification as pending: %v", err)
	}

	_, err = app.NonconcurrentDB().NewQuery(`
		UPDATE notification_templates
		SET text_email = {:text}
		WHERE id = {:id}
	`).Bind(dbx.Params{
		"id":   target.TemplateID,
		"text": "Hello {{.DefinitelyMissingTemplateKey}}",
	}).Execute()
	if err != nil {
		t.Fatalf("Failed to update template: %v", err)
	}

	var countResult struct {
		Count int64 `db:"count"`
	}
	err = app.DB().NewQuery("SELECT COUNT(*) AS count FROM notifications WHERE status = 'pending'").One(&countResult)
	if err != nil {
		t.Fatalf("Failed to get initial count: %v", err)
	}
	initialCount := countResult.Count

	if initialCount != 1 {
		t.Fatalf("Expected exactly one pending notification for this test, got %d", initialCount)
	}

	remaining, err := notifications.SendNextPendingNotification(app)
	if err == nil {
		t.Fatal("Expected an error when template key is missing, got nil")
	}
	if !strings.Contains(err.Error(), "error executing text template for notification") {
		t.Fatalf("Expected missing key execution error, got %v", err)
	}
	if remaining != initialCount {
		t.Errorf("Expected remaining count to be %d when template execution fails, got %d", initialCount, remaining)
	}
}

//  7. the pending count and an error are returned if legacy placeholders remain
//     unresolved after rendering.
func TestSendNextPendingNotification_ErrorOnLegacyPlaceholder(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	var target struct {
		ID         string `db:"id"`
		TemplateID string `db:"template"`
	}
	if err := app.DB().NewQuery(`
		SELECT id, template
		FROM notifications
		LIMIT 1
	`).One(&target); err != nil {
		t.Fatalf("Failed to load target notification: %v", err)
	}

	_, err := app.NonconcurrentDB().NewQuery("UPDATE notifications SET status = 'sent'").Execute()
	if err != nil {
		t.Fatalf("Failed to normalize notification statuses: %v", err)
	}
	_, err = app.NonconcurrentDB().NewQuery(`
		UPDATE notifications
		SET status = 'pending'
		WHERE id = {:id}
	`).Bind(dbx.Params{
		"id": target.ID,
	}).Execute()
	if err != nil {
		t.Fatalf("Failed to mark target notification as pending: %v", err)
	}

	_, err = app.NonconcurrentDB().NewQuery(`
		UPDATE notification_templates
		SET text_email = {:text}
		WHERE id = {:id}
	`).Bind(dbx.Params{
		"id":   target.TemplateID,
		"text": "{APP_URL}/time/entries/list",
	}).Execute()
	if err != nil {
		t.Fatalf("Failed to update template: %v", err)
	}

	var countResult struct {
		Count int64 `db:"count"`
	}
	err = app.DB().NewQuery("SELECT COUNT(*) AS count FROM notifications WHERE status = 'pending'").One(&countResult)
	if err != nil {
		t.Fatalf("Failed to get initial count: %v", err)
	}
	initialCount := countResult.Count

	if initialCount != 1 {
		t.Fatalf("Expected exactly one pending notification for this test, got %d", initialCount)
	}

	remaining, err := notifications.SendNextPendingNotification(app)
	if err == nil {
		t.Fatal("Expected an error when legacy placeholder remains unresolved, got nil")
	}
	if !strings.Contains(err.Error(), "unresolved legacy placeholder") {
		t.Fatalf("Expected unresolved legacy placeholder error, got %v", err)
	}
	if remaining != initialCount {
		t.Errorf("Expected remaining count to be %d when legacy placeholder check fails, got %d", initialCount, remaining)
	}
}

// SendNotifications()

//  1. on success, sentCount matches the number of emails in the TestMailer
//     messages parameter.
func TestSendNotifications_SendsAllPendingNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Call SendNotifications
	sentCount, err := notifications.SendNotifications(app)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	time.Sleep(20 * time.Millisecond)
	// Verify sentCount matches the number of emails in the TestMailer messages
	// parameter
	if sentCount != int64(len(app.TestMailer.Messages())) {
		t.Errorf("Expected sentCount to be %d, got %d", len(app.TestMailer.Messages()), sentCount)
	}

	// Verify that there are 5 messages in the TestMailer, matching the 5 pending
	// notifications in the test data.
	if len(app.TestMailer.Messages()) != 5 {
		t.Errorf("Expected 5 messages in the TestMailer, got %d", len(app.TestMailer.Messages()))
	}

	for i, msg := range app.TestMailer.Messages() {
		if strings.Contains(msg.Text, "{APP_URL}") || strings.Contains(msg.Text, "{:RECORD_ID}") || strings.Contains(msg.Text, "<no value>") {
			t.Fatalf("message %d contains unresolved placeholders or missing values: %q", i, msg.Text)
		}
	}
}

//  2. on failure, sentCount matches the number of emails in the TestMailer
//     messages parameter but an error is returned
func TestSendNotifications_ErrorHandling(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Break the notifications table to force query errors
	_, err := app.NonconcurrentDB().NewQuery("ALTER TABLE notifications RENAME TO notifications_broken").Execute()
	if err != nil {
		t.Fatalf("Failed to rename notifications table: %v", err)
	}

	// Call SendNotifications
	sentCount, err := notifications.SendNotifications(app)

	// Sleep briefly to allow any async operations to complete
	time.Sleep(20 * time.Millisecond)

	// Verify we got an error
	if err == nil {
		t.Error("Expected an error when email sending fails, got nil")
	}

	// Verify sentCount matches the number of emails in the TestMailer messages
	messageCount := len(app.TestMailer.Messages())
	if sentCount != int64(messageCount) {
		t.Errorf("Expected sentCount to be %d, got %d", messageCount, sentCount)
	}
}

func TestCreateNotification_SkipsWhenFeatureDisabled(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	upsertNotificationsConfigRawValue(t, app, `{"timesheet_shared":false}`)

	var userRow struct {
		UID string `db:"uid"`
	}
	if err := app.DB().NewQuery(`SELECT id AS uid FROM users LIMIT 1`).One(&userRow); err != nil {
		t.Fatalf("failed to find recipient user: %v", err)
	}

	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("timesheet_shared")

	if err := notifications.CreateNotification(app, "timesheet_shared", userRow.UID, map[string]any{
		"WeekEnding": "2026-02-14",
	}, true); err != nil {
		t.Fatalf("expected no error when disabled notification is skipped, got %v", err)
	}

	afterCount := countForTemplate("timesheet_shared")
	if afterCount != beforeCount {
		t.Fatalf("expected notification count to remain unchanged when feature is disabled, before=%d after=%d", beforeCount, afterCount)
	}
}

// SendNotificationByID()

// 1. sends a specific pending notification identified by record ID
func TestSendNotificationByID_SendsTargetedNotification(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Get the first pending notification ID
	var row struct {
		ID string `db:"id"`
	}
	if err := app.DB().NewQuery("SELECT id FROM notifications WHERE status = 'pending' LIMIT 1").One(&row); err != nil {
		t.Fatalf("failed to find pending notification: %v", err)
	}

	// Send just that notification
	if err := notifications.SendNotificationByID(app, row.ID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for async goroutine
	time.Sleep(100 * time.Millisecond)

	// Verify exactly one email was sent
	if count := len(app.TestMailer.Messages()); count != 1 {
		t.Errorf("expected 1 email sent, got %d", count)
	}

	// Verify the notification status transitioned out of pending
	var status struct {
		Status string `db:"status"`
	}
	if err := app.DB().NewQuery("SELECT status FROM notifications WHERE id = {:id}").Bind(dbx.Params{
		"id": row.ID,
	}).One(&status); err != nil {
		t.Fatalf("failed to query notification status: %v", err)
	}
	if status.Status == "pending" {
		t.Error("expected notification status to no longer be pending")
	}
}

// 2. is a no-op for a non-existent notification ID
func TestSendNotificationByID_NoOpForNonExistentID(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	if err := notifications.SendNotificationByID(app, "nonexistent_id_12345"); err != nil {
		t.Fatalf("expected no error for non-existent ID, got %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	if count := len(app.TestMailer.Messages()); count != 0 {
		t.Errorf("expected 0 emails sent for non-existent ID, got %d", count)
	}
}

// CreateAndSendNotification()

// 1. creates a notification record and immediately sends it
func TestCreateAndSendNotification_CreatesAndSends(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Clear all existing pending notifications first
	if _, err := notifications.SendNotifications(app); err != nil {
		t.Fatalf("failed to clear pending notifications: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	initialMessages := len(app.TestMailer.Messages())

	// Find a user to send to
	var userRow struct {
		UID string `db:"uid"`
	}
	if err := app.DB().NewQuery(`SELECT id AS uid FROM users LIMIT 1`).One(&userRow); err != nil {
		t.Fatalf("failed to find recipient user: %v", err)
	}

	// Create and send a notification
	err := notifications.CreateAndSendNotification(app, "po_approval_required", userRow.UID, map[string]any{
		"POId":      "test_po_id",
		"ActionURL": "https://example.com/pos/test_po_id/edit",
	}, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Wait for async send
	time.Sleep(200 * time.Millisecond)

	// Verify exactly one new email was sent
	newMessages := len(app.TestMailer.Messages()) - initialMessages
	if newMessages != 1 {
		t.Errorf("expected 1 new email sent, got %d", newMessages)
	}

	// Verify notification record exists and is no longer pending
	var result struct {
		Count int64 `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM notifications n
		JOIN notification_templates t ON n.template = t.id
		WHERE t.code = 'po_approval_required'
		  AND n.recipient = {:uid}
		  AND n.status != 'pending'
	`).Bind(dbx.Params{
		"uid": userRow.UID,
	}).One(&result); err != nil {
		t.Fatalf("failed to query notification: %v", err)
	}
	if result.Count == 0 {
		t.Error("expected notification to be created and no longer pending")
	}
}

// 2. skips send when feature is disabled
func TestCreateAndSendNotification_SkipsWhenFeatureDisabled(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	upsertNotificationsConfigRawValue(t, app, `{"timesheet_shared":false}`)

	var userRow struct {
		UID string `db:"uid"`
	}
	if err := app.DB().NewQuery(`SELECT id AS uid FROM users LIMIT 1`).One(&userRow); err != nil {
		t.Fatalf("failed to find recipient user: %v", err)
	}

	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("timesheet_shared")

	err := notifications.CreateAndSendNotification(app, "timesheet_shared", userRow.UID, map[string]any{
		"WeekEnding": "2026-02-14",
	}, true)
	if err != nil {
		t.Fatalf("expected no error when disabled notification is skipped, got %v", err)
	}

	afterCount := countForTemplate("timesheet_shared")
	if afterCount != beforeCount {
		t.Fatalf("expected notification count to remain unchanged when feature is disabled, before=%d after=%d", beforeCount, afterCount)
	}
}

// QueueTimesheetSubmissionReminders()
//
//  1. creates one or more notifications with the timesheet_submission_reminder template
//     for users who are missing a timesheet for the previous week.
//
// NOTE: This test is currently skipped because the test database (test_pb_data/data.db)
// does not contain any users who are expected to submit a timesheet but haven't
// for any given week. To enable this test, seed such data in data.db.
func TestQueueTimesheetSubmissionReminders_CreatesNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Find a week_ending value that has at least one user who is expected to have
	// a timesheet but is missing it. This mirrors the logic in
	// QueueTimesheetSubmissionRemindersForWeek.
	var weeks []struct {
		WeekEnding string `db:"week_ending"`
	}
	if err := app.DB().NewQuery(`
		SELECT DISTINCT week_ending
		FROM time_sheets
		WHERE week_ending != ''
	`).All(&weeks); err != nil {
		t.Fatalf("failed to load distinct week_endings: %v", err)
	}

	var targetWeek string
	for _, w := range weeks {
		var res struct {
			Count int `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT
				COUNT(*) AS count
			FROM users u
			LEFT JOIN time_sheets ts ON ts.uid = u.id AND ts.week_ending = {:week_ending} AND ts.submitted = 1
			LEFT JOIN admin_profiles ap ON ap.uid = u.id
			WHERE ts.id IS NULL
			  AND COALESCE(ap.time_sheet_expected, 0) = 1
		`).Bind(dbx.Params{
			"week_ending": w.WeekEnding,
		}).One(&res)
		if err != nil {
			t.Fatalf("failed to check missing timesheets for week %s: %v", w.WeekEnding, err)
		}
		if res.Count > 0 {
			targetWeek = w.WeekEnding
			break
		}
	}

	if targetWeek == "" {
		t.Skip("no week in test data where expected users are missing timesheets")
	}

	// Helper to count notifications for a given template code
	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("timesheet_submission_reminder")

	// Call the queue function without sending emails for the specific target week
	if err := notifications.QueueTimesheetSubmissionRemindersForWeek(app, targetWeek, false); err != nil {
		t.Fatalf("expected no error from QueueTimesheetSubmissionReminders, got %v", err)
	}

	afterCount := countForTemplate("timesheet_submission_reminder")

	if afterCount <= beforeCount {
		t.Fatalf("expected timesheet_submission_reminder notifications to be created, before=%d after=%d", beforeCount, afterCount)
	}
}

// QueueTimesheetApprovalReminders()
//
//  1. creates one or more notifications with the timesheet_approval_reminder template
//     for managers who have pending timesheets awaiting approval.
func TestQueueTimesheetApprovalReminders_CreatesNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// First, ensure there is at least one manager with pending timesheets in the test data.
	var managers []struct {
		ManagerUID string `db:"manager_uid"`
	}
	if err := app.DB().NewQuery(`
		SELECT DISTINCT
			ts.approver AS manager_uid
		FROM time_sheets ts
		WHERE ts.submitted = 1
		  AND ts.approved = ''
		  AND ts.committed = ''
		  AND ts.rejected = ''
		  AND ts.approver != ''
	`).All(&managers); err != nil {
		t.Fatalf("failed to query managers with pending timesheets: %v", err)
	}
	if len(managers) == 0 {
		t.Skip("no managers with pending timesheets in test data")
	}

	// Helper to count notifications for a given template code
	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("timesheet_approval_reminder")

	// Call the queue function without sending emails
	if err := notifications.QueueTimesheetApprovalReminders(app, false); err != nil {
		t.Fatalf("expected no error from QueueTimesheetApprovalReminders, got %v", err)
	}

	afterCount := countForTemplate("timesheet_approval_reminder")

	if afterCount <= beforeCount {
		t.Fatalf("expected timesheet_approval_reminder notifications to be created, before=%d after=%d", beforeCount, afterCount)
	}
}

// QueueExpenseApprovalReminders()
//
//  1. creates one or more notifications with the expense_approval_reminder template
//     for managers who have pending expenses awaiting approval.
//
// NOTE: This test is currently skipped because the test database (test_pb_data/data.db)
// does not contain any managers with pending expenses (submitted=1, approved=”,
// committed=”, rejected=”). To enable this test, seed such data in data.db.
func TestQueueExpenseApprovalReminders_CreatesNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// First, ensure there is at least one manager with pending expenses in the test data.
	var managers []struct {
		ManagerUID string `db:"manager_uid"`
	}
	if err := app.DB().NewQuery(`
		SELECT DISTINCT
			e.approver AS manager_uid
		FROM expenses e
		WHERE e.submitted = 1
		  AND e.approved = ''
		  AND e.committed = ''
		  AND e.rejected = ''
		  AND e.approver != ''
	`).All(&managers); err != nil {
		t.Fatalf("failed to query managers with pending expenses: %v", err)
	}
	if len(managers) == 0 {
		t.Skip("no managers with pending expenses in test data")
	}

	// Helper to count notifications for a given template code
	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("expense_approval_reminder")

	// Call the queue function without sending emails
	if err := notifications.QueueExpenseApprovalReminders(app, false); err != nil {
		t.Fatalf("expected no error from QueueExpenseApprovalReminders, got %v", err)
	}

	afterCount := countForTemplate("expense_approval_reminder")

	if afterCount <= beforeCount {
		t.Fatalf("expected expense_approval_reminder notifications to be created, before=%d after=%d", beforeCount, afterCount)
	}
}

// QueueTimesheetRejectedNotifications()
//
//  1. creates notifications with the timesheet_rejected template for the employee and
//     the rejector (and manager when different), using data from the timesheet and profiles.
func TestQueueTimesheetRejectedNotifications_CreatesNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Find a timesheet whose employee has a manager so we exercise the manager path.
	var tsRow struct {
		TimesheetID string `db:"tsid"`
		EmployeeUID string `db:"employee_uid"`
		ManagerUID  string `db:"manager_uid"`
	}
	if err := app.DB().NewQuery(`
		SELECT
			ts.id           AS tsid,
			ts.uid          AS employee_uid,
			p.manager       AS manager_uid
		FROM time_sheets ts
		JOIN profiles p ON p.uid = ts.uid
		WHERE COALESCE(p.manager, '') != ''
		LIMIT 1
	`).One(&tsRow); err != nil {
		t.Skipf("no suitable time_sheets record with manager in test data: %v", err)
	}

	// Use the manager as the rejector for this happy-path test.
	rejectorUID := tsRow.ManagerUID

	// Load the timesheet record to pass to the helper.
	timesheet, err := app.FindRecordById("time_sheets", tsRow.TimesheetID)
	if err != nil {
		t.Fatalf("failed to load timesheet %s: %v", tsRow.TimesheetID, err)
	}

	// Helper to count notifications for a given template code
	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("timesheet_rejected")

	const rejectionReason = "Test rejection reason from unit test"

	// Call the helper to queue notifications
	if err := notifications.QueueTimesheetRejectedNotifications(app, timesheet, rejectorUID, rejectionReason); err != nil {
		t.Fatalf("expected no error from QueueTimesheetRejectedNotifications, got %v", err)
	}

	afterCount := countForTemplate("timesheet_rejected")

	if afterCount <= beforeCount {
		t.Fatalf("expected timesheet_rejected notifications to be created, before=%d after=%d", beforeCount, afterCount)
	}
}

// QueueExpenseRejectedNotifications()
//
//  1. creates notifications with the expense_rejected template for the employee and
//     the rejector (and manager when different), using data from the expense and profiles.
func TestQueueExpenseRejectedNotifications_CreatesNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Find an expense whose employee has a manager so we exercise the manager path.
	var expRow struct {
		ExpenseID   string `db:"eid"`
		EmployeeUID string `db:"employee_uid"`
		ManagerUID  string `db:"manager_uid"`
	}
	if err := app.DB().NewQuery(`
		SELECT
			e.id      AS eid,
			e.uid     AS employee_uid,
			p.manager AS manager_uid
		FROM expenses e
		JOIN profiles p ON p.uid = e.uid
		WHERE COALESCE(p.manager, '') != ''
		LIMIT 1
	`).One(&expRow); err != nil {
		t.Skipf("no suitable expenses record with manager in test data: %v", err)
	}

	// Use the manager as the rejector for this happy-path test.
	rejectorUID := expRow.ManagerUID

	// Load the expense record to pass to the helper.
	expense, err := app.FindRecordById("expenses", expRow.ExpenseID)
	if err != nil {
		t.Fatalf("failed to load expense %s: %v", expRow.ExpenseID, err)
	}

	// Helper to count notifications for a given template code
	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("expense_rejected")

	const rejectionReason = "Test expense rejection reason from unit test"

	// Call the helper to queue notifications
	if err := notifications.QueueExpenseRejectedNotifications(app, expense, rejectorUID, rejectionReason); err != nil {
		t.Fatalf("expected no error from QueueExpenseRejectedNotifications, got %v", err)
	}

	afterCount := countForTemplate("expense_rejected")

	if afterCount <= beforeCount {
		t.Fatalf("expected expense_rejected notifications to be created, before=%d after=%d", beforeCount, afterCount)
	}
}

// QueueTimesheetSharedNotifications()
//
//  1. creates notifications with the timesheet_shared template for newly added viewers,
//     using data from the timesheet and profiles.
func TestQueueTimesheetSharedNotifications_CreatesNotifications(t *testing.T) {
	// Set up test app
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Find a timesheet with a non-empty approver so we can use them as the sharer.
	var tsRow struct {
		TimesheetID string `db:"tsid"`
		ApproverUID string `db:"approver_uid"`
	}
	if err := app.DB().NewQuery(`
		SELECT
			ts.id        AS tsid,
			ts.approver  AS approver_uid
		FROM time_sheets ts
		WHERE COALESCE(ts.approver, '') != ''
		LIMIT 1
	`).One(&tsRow); err != nil {
		t.Skipf("no suitable time_sheets record with approver in test data: %v", err)
	}

	// Load the timesheet record to pass to the helper.
	timesheet, err := app.FindRecordById("time_sheets", tsRow.TimesheetID)
	if err != nil {
		t.Fatalf("failed to load timesheet %s: %v", tsRow.TimesheetID, err)
	}

	sharerUID := tsRow.ApproverUID

	// Find a viewer UID that is different from the sharer (and ideally exists as a user).
	var viewerRow struct {
		ViewerUID string `db:"viewer_uid"`
	}
	if err := app.DB().NewQuery(`
		SELECT id AS viewer_uid
		FROM users
		WHERE id != {:sharer}
		LIMIT 1
	`).Bind(dbx.Params{
		"sharer": sharerUID,
	}).One(&viewerRow); err != nil {
		t.Skipf("no suitable viewer user found in test data: %v", err)
	}

	// Helper to count notifications for a given template code
	countForTemplate := func(code string) int64 {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": code,
		}).One(&result)
		if err != nil {
			t.Fatalf("failed to count notifications for code %s: %v", code, err)
		}
		return result.Count
	}

	beforeCount := countForTemplate("timesheet_shared")

	// Call the helper to queue notifications for a single new viewer
	if err := notifications.QueueTimesheetSharedNotifications(app, timesheet, sharerUID, []string{viewerRow.ViewerUID}); err != nil {
		t.Fatalf("expected no error from QueueTimesheetSharedNotifications, got %v", err)
	}

	afterCount := countForTemplate("timesheet_shared")

	if afterCount <= beforeCount {
		t.Fatalf("expected timesheet_shared notifications to be created, before=%d after=%d", beforeCount, afterCount)
	}
}
