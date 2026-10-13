package notifications_test

import (
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"
	"tybalt/notifications"
)

var projectWorkflowEmailCodes = []string{
	"job_value_submitted", "job_value_approved", "job_value_rejected",
	"job_request_submitted", "job_request_approved", "job_request_rejected", "job_request_created",
}

func TestProjectWorkflowNoticesAreQueuedByDefault(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	hooks.AddHooks(app)
	for _, template := range projectWorkflowEmailCodes {
		id, err := notifications.DispatchNotification(app, notifications.DispatchArgs{TemplateCode: template, RecipientUID: "6bq4j0eb26631dy", ActorUID: "rzr98oadsp9qc11", System: true, Mode: notifications.DeliveryDeferred, Data: map[string]any{"JobNumber": "26-9876", "Description": "Fixture project", "ClientName": "Fixture client", "ActionURL": "/jobs/fixture/details"}})
		if err != nil {
			t.Fatal(err)
		}
		if id == "" {
			t.Fatalf("required notice was skipped: %s", template)
		}
		record, err := app.FindRecordById("notifications", id)
		if err != nil {
			t.Fatal(err)
		}
		if record.GetString("status") != "pending" || record.GetString("recipient") != "6bq4j0eb26631dy" {
			t.Fatalf("incorrect queued notice: %v", record.PublicExport())
		}
	}
}

// Subjects are filled in from the notification data like the body.
func TestNotificationSubjectUsesData(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	hooks.AddHooks(app)
	notifications.SetSendNotificationAsyncForTest(app, false)
	id, err := notifications.DispatchNotification(app, notifications.DispatchArgs{TemplateCode: "job_value_rejected", RecipientUID: "f2j5a8vk006baub", ActorUID: "6bq4j0eb26631dy", System: true, Mode: notifications.DeliveryImmediate, Data: map[string]any{"JobNumber": "26-9876", "Description": "Fixture project", "ClientName": "Fixture client", "ProposedValue": "$10,000", "CurrentValue": "$0", "RejectionReason": "Attach the change order.", "ActionURL": "/jobs/fixture/details"}})
	if err != nil || id == "" {
		t.Fatalf("notice was not created: %q %v", id, err)
	}
	messages := app.TestMailer.Messages()
	if len(messages) != 1 || messages[0].Subject != "Rejected: project 26-9876 value change" {
		t.Fatalf("subject was not filled in: %v", messages)
	}
}
