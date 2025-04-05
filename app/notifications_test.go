// notifications_test.go

package main

import (
	"testing"
	"time"
	"tybalt/internal/testutils"
	"tybalt/notifications"
)

// This will be the test file for the notifications package. We're using mailpit
// for testing so we'll use the mailpit API to check for messages as expected in
// the test cases below. The mailbox should be emptied prior to each test so
// that each test is independent. The mailpit API is expected to be running
// at http://localhost:8025

// SendNextPendingNotification()

//  1. one email is sent when there are one or more pending notifications
//     notifications. The pending count is returned as 1 less than the original
//     count, no error is returned, and there is one email in the mailpit inbox.
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
	// complete before checking the mailpit inbox. TODO: Find a better way to do
	// this, perhaps by using a channel to communicate the completion of the email
	// sending.
	time.Sleep(100 * time.Millisecond)

	messageCount := len(app.TestMailer.Messages())
	if messageCount != 1 {
		t.Errorf("Expected 1 email to be sent, got %d", messageCount)
	}
}

// 2. no emails are sent when there are no pending notifications. The pending
//    count is returned as 0 and a nil error is returned. There are no emails
//    in the mailpit inbox.

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

// 1. on success, sentCount matches the number of emails in the mailpit inbox.

// 2. on failure, sentCount matches the number of emails in the mailpit inbox
//    but an error is returned
