// notifications_test.go

package main

import (
	"strings"
	"testing"
	"time"
	"tybalt/internal/testutils"
	"tybalt/notifications"
)

// This is the test file for the notifications package.

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
	if remaining != 3 {
		t.Errorf("Expected remaining count to be 3, got %d", remaining)
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
	} else if !strings.Contains(err.Error(), "error fetching pending notification") {
		t.Errorf("Expected an error containing 'error fetching pending notification', got %v", err)
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

	// Verify that there are 4 messages in the TestMailer, matching the 4 pending
	// notifications in the test data.
	if len(app.TestMailer.Messages()) != 4 {
		t.Errorf("Expected 4 messages in the TestMailer, got %d", len(app.TestMailer.Messages()))
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
