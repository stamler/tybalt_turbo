package hooks

import (
	"context"
	"github.com/pocketbase/dbx"
	"testing"
	"tybalt/internal/testseed"
	"tybalt/utilities"
)

func TestProjectAuthorityCurrentFullValue(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	for _, tc := range []struct {
		name, uid string
		value     float64
		valid     bool
	}{
		{"exact", "6bq4j0eb26631dy", 1000000, true}, {"below", "6bq4j0eb26631dy", 100000, true}, {"above", "6bq4j0eb26631dy", 1000001, false}, {"fraction", "6bq4j0eb26631dy", 10.5, false}, {"zero", "6bq4j0eb26631dy", 0, false}, {"admin is not authority", "f2j5a8vk006baub", 10, false}, {"inactive", "inactpoappr0001", 10, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := utilities.ProjectAuthorizationLimit(app, tc.uid, tc.value)
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, got %v", tc.valid, err)
			}
		})
	}
	// Change existing fixture authority to test live revocation. No data is seeded here.
	props, err := app.FindRecordById("project_authorization_props", "paprops00000001")
	if err != nil {
		t.Fatal(err)
	}
	props.Set("max_value", 50)
	if err = app.Save(props); err != nil {
		t.Fatal(err)
	}
	if _, err = utilities.ProjectAuthorizationLimit(app, "6bq4j0eb26631dy", 100000); err == nil {
		t.Fatal("reduced authority still approved full value")
	}
}

func TestProjectWorkflowModelGuards(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	for _, table := range []string{"job_requests", "job_attachments", "job_request_allocations"} {
		id := map[string]string{"job_requests": "parequest000001", "job_attachments": "paattachment001", "job_request_allocations": "paallocation001"}[table]
		r, err := app.FindRecordById(table, id)
		if err != nil {
			t.Fatal(err)
		}
		if err = app.Save(r); err == nil {
			t.Fatalf("generic model save allowed for %s", table)
		}
		if err = app.Delete(r); err == nil {
			t.Fatalf("generic model delete allowed for %s", table)
		}
	}
	job, err := app.FindRecordById("jobs", "cjf0kt0defhq480")
	if err != nil {
		t.Fatal(err)
	}
	job.Set("project_value", 123456)
	if err = app.Save(job); err == nil {
		t.Fatal("direct project value bypass allowed")
	}
}

func TestProjectRequestDataAndSharedRelations(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	request, err := app.FindRecordById("job_requests", "parequest000001")
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateProjectRequestData(app, request); err != nil {
		t.Fatalf("complete request: %v", err)
	}
	request.Set("contact", "235g6k01xx3sdjk")
	if err = ValidateProjectRequestData(app, request); err == nil {
		t.Fatal("different client contact allowed")
	}
	request.Set("contact", "184qbpajciwrrp2")
	request.Set("fn_agreement_answer", "")
	if err = ValidateProjectRequestData(app, request); err == nil {
		t.Fatal("missing First Nations answer allowed")
	}
}

func TestAuthorityMissingAndInvalidProperties(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limit  float64
		remove bool
	}{{"missing properties", 0, true}, {"zero", 0, false}, {"fraction", 100.5, false}, {"excessive", 1e20, false}, {"above exact range", 9007199254740992, false}} {
		t.Run(tc.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			// Change the authority fixture to exercise corrupted or absent properties.
			var err error
			if tc.remove {
				_, err = app.DB().NewQuery("DELETE FROM project_authorization_props WHERE id='paprops00000001'").Execute()
			} else {
				_, err = app.DB().NewQuery("UPDATE project_authorization_props SET max_value={:limit} WHERE id='paprops00000001'").Bind(dbx.Params{"limit": tc.limit}).Execute()
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err = utilities.ProjectAuthorizationLimit(app, "6bq4j0eb26631dy", 10); err == nil {
				t.Fatal("invalid properties granted authority")
			}
		})
	}
}

func TestJobDeletionPreservesEvidence(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	// Transfer the existing PDF fixture to the legacy job to test deletion guards.
	attachment, err := app.FindRecordById("job_attachments", "paattachment001")
	if err != nil {
		t.Fatal(err)
	}
	attachment.Set("request", "")
	attachment.Set("job", "pawflegacy00001")
	ctx := WithProjectWorkflowMutation(context.Background(), ProjectWorkflowMutationValue)
	if err = app.SaveWithContext(ctx, attachment); err != nil {
		t.Fatal(err)
	}
	job, err := app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	if err = app.Delete(job); err == nil {
		t.Fatal("job deletion removed pending evidence")
	}
	if _, err = app.FindRecordById("job_attachments", attachment.Id); err != nil {
		t.Fatal("attachment lost after failed job delete")
	}
	attachment.Set("status", "Approved")
	attachment.Set("initial_approval", true)
	attachment.Set("approval_sequence", 1)
	if err = app.SaveWithContext(ctx, attachment); err != nil {
		t.Fatal(err)
	}
	if err = app.DeleteWithContext(WithProjectWorkflowMutation(ctx, ProjectWorkflowMutationDelete), attachment); err == nil {
		t.Fatal("initial approval deletion allowed")
	}
}

func TestHistoricalRestorationRejectsArbitraryValue(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	// Use the existing attachment as a later approval on the legacy job.
	attachment, err := app.FindRecordById("job_attachments", "paattachment001")
	if err != nil {
		t.Fatal(err)
	}
	attachment.Set("request", "")
	attachment.Set("job", "pawflegacy00001")
	attachment.Set("status", "Approved")
	attachment.Set("approval_sequence", 1)
	attachment.Set("previous_project_value", 0)
	ctx := WithProjectWorkflowMutation(context.Background(), ProjectWorkflowMutationApproveValue)
	if err = app.SaveWithContext(ctx, attachment); err != nil {
		t.Fatal(err)
	}
	job, err := app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	job.Set("project_value", 100000)
	if err = app.SaveWithContext(ctx, job); err != nil {
		t.Fatal(err)
	}
	restore := WithProjectWorkflowMutation(ctx, ProjectWorkflowMutationRestoreValue)
	job, err = app.FindRecordById("jobs", job.Id)
	if err != nil {
		t.Fatal(err)
	}
	job.Set("project_value", 12345)
	if err = app.SaveWithContext(restore, job); err == nil {
		t.Fatal("arbitrary restored value was accepted")
	}
	job, err = app.FindRecordById("jobs", job.Id)
	if err != nil {
		t.Fatal(err)
	}
	job.Set("project_value", 0)
	if err = app.SaveWithContext(restore, job); err != nil {
		t.Fatalf("exact stored zero: %v", err)
	}
}

func TestRequestCreationFlagRejectsInvalidConfiguration(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	enabled, err := utilities.IsJobRequestApprovalRequired(app)
	if err != nil || enabled {
		t.Fatalf("default flag: enabled=%v error=%v", enabled, err)
	}
	// Change the existing configuration fixture; invalid values must not allow direct creation.
	config, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatal(err)
	}
	config.Set("value", map[string]any{"require_job_request_approval": "invalid"})
	if err = app.Save(config); err != nil {
		t.Fatal(err)
	}
	if _, err = utilities.IsJobRequestApprovalRequired(app); err == nil {
		t.Fatal("invalid creation flag was treated as disabled")
	}
	config.Set("value", map[string]any{"require_job_request_approval": true})
	if err = app.Save(config); err != nil {
		t.Fatal(err)
	}
	if enabled, err = utilities.IsJobRequestApprovalRequired(app); err != nil || !enabled {
		t.Fatalf("enabled flag: %v %v", enabled, err)
	}
}

func TestAuthorityPropertiesModelRejectsExcessiveLimit(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	// Change the existing authority fixture to test the guard even when schema validation is skipped.
	props, err := app.FindRecordById("project_authorization_props", "paprops00000001")
	if err != nil {
		t.Fatal(err)
	}
	props.Set("max_value", 1e20)
	if err = app.SaveNoValidate(props); err == nil {
		t.Fatal("model accepted an excessive authority limit")
	}
	props, err = app.FindRecordById("project_authorization_props", "paprops00000001")
	if err != nil {
		t.Fatal(err)
	}
	props.Set("max_value", utilities.MaxProjectAuthorizationValue)
	if err = app.Save(props); err != nil {
		t.Fatalf("exact safe maximum: %v", err)
	}
	limit, err := utilities.ProjectAuthorizationLimit(app, "6bq4j0eb26631dy", utilities.MaxProjectAuthorizationValue)
	if err != nil || limit != utilities.MaxProjectAuthorizationValue {
		t.Fatalf("exact safe maximum authority: %d %v", limit, err)
	}
}

func TestJSONFieldsCannotSelectInternalWorkflowPolicy(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	actor, err := app.FindRecordById("users", "f2j5a8vk006baub")
	if err != nil {
		t.Fatal(err)
	}
	job, err := app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	// Simulate unknown JSON fields passed through the generic custom job writer.
	// Closing the zero-value project needs a value unless that policy applies.
	job.Set("@projectWorkflowOperation", "restore_value")
	job.Set("status", "Closed")
	job.Set("description", "Attempted zero-value validation bypass")
	if err = ProcessJobCore(app, job, actor); err == nil {
		t.Fatal("JSON field selected the historical restoration policy")
	}
	job, err = app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	job.Set("@projectWorkflowValidated", true)
	job.Set("description", "Attempted model validation bypass")
	if err = app.Save(job); err == nil {
		t.Fatal("JSON Boolean selected the validated-model policy")
	}
}

func TestValidatedJobCannotChangeBeforeModelSave(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	// Give the legacy fixture a positive effective value to test an ordinary validated edit.
	if _, err := app.DB().NewQuery("UPDATE jobs SET project_value=100000 WHERE id='pawflegacy00001'").Execute(); err != nil {
		t.Fatal(err)
	}
	actor, err := app.FindRecordById("users", "f2j5a8vk006baub")
	if err != nil {
		t.Fatal(err)
	}
	job, err := app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	if err = ProcessJobCore(app, job, actor); err != nil {
		t.Fatal(err)
	}
	job.Set("description", "Changed after validation")
	if err = app.Save(job); err == nil {
		t.Fatal("model saved data changed after validation")
	}
	job, err = app.FindRecordById("jobs", "pawflegacy00001")
	if err != nil {
		t.Fatal(err)
	}
	job.Set("description", "Changed before validation")
	if err = ProcessJobCore(app, job, actor); err != nil {
		t.Fatal(err)
	}
	if err = app.Save(job); err != nil {
		t.Fatalf("validated data failed to save: %v", err)
	}
}

// A model save cannot switch a job between project and proposal numbers, even
// within a workflow operation. Only the renumber action changes numbers.
func TestJobNumberCannotSwitchJobType(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddProjectWorkflowHooks(app)
	ctx := WithProjectWorkflowMutation(context.Background(), ProjectWorkflowMutationMerge)
	for id, number := range map[string]string{"pcactivedated01": "P86-9002", "fcpropimpinprog": "88-9105"} {
		job, err := app.FindRecordById("jobs", id)
		if err != nil {
			t.Fatal(err)
		}
		job.Set("number", number)
		if err = app.SaveWithContext(ctx, job); err == nil {
			t.Fatalf("%s changed type with number %s", id, number)
		}
	}
}
