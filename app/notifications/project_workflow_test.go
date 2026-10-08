package notifications_test

import (
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"
	"tybalt/notifications"
)

func TestProjectWorkflowNoticesAreQueuedByDefault(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	hooks.AddHooks(app)
	for _, template := range []string{"job_request_workflow", "job_value_workflow"} {
		id, err := notifications.DispatchNotification(app, notifications.DispatchArgs{TemplateCode: template, RecipientUID: "6bq4j0eb26631dy", ActorUID: "rzr98oadsp9qc11", System: true, Mode: notifications.DeliveryDeferred, Data: map[string]any{"RequestID": "parequest000001", "JobNumber": "26-9876", "Description": "Fixture project", "Action": "submitted", "ActionURL": "/job-requests/parequest000001"}})
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
