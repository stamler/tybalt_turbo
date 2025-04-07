// notifications_test.go

package main

import (
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

// 2. no emails are sent when there are no pending notifications. The pending
//    count is returned as 0 and a nil error is returned. There are no emails
//    in the TestMailer.Messages() inbox.

// 3. pending count of 0 and an error are returned if the CountResult query
//    fails

// 4. the pending count and an error are returned if fetching one notification
//    fails for a reason other than there being no pending notifications

// 5. the pending count and an error are returned if the text template cannot be
//    parsed

// 6. the pending count and an error are returned if the text template cannot be
//    executed

// 7. the pending count and an error are returned if updating the notification
//    status to inflight fails

// 8. the pending count and an error are returned if the email cannot be sent

// 9. the pending count and an error are returned if updating the notification
//    status to sent fails

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

// 2. on failure, sentCount matches the number of emails in the TestMailer
//    messages parameter but an error is returned
