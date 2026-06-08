package hooks

import (
	"context"
	"errors"
	"strings"
	"testing"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	paGateMissingProjectID  = "pafixmissing01"
	paGatePendingProjectID  = "pafixpending01"
	paGateApprovedProjectID = "pafixapprove01"
	paGateLegacyPOProjectID = "pafixlegacy01"
	paGateProposalID        = "kyed8fha9uaha27"
)

func TestEnsureProjectAuthorizationApprovedForJob(t *testing.T) {
	t.Run("disabled enforcement allows unapproved project", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, false)

		if err := EnsureProjectAuthorizationApprovedForJob(app, paGateMissingProjectID, "job"); err != nil {
			t.Fatalf("expected disabled enforcement to allow project, got %v", err)
		}
	})

	t.Run("enabled enforcement blocks unapproved project", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, true)

		err := EnsureProjectAuthorizationApprovedForJob(app, paGateMissingProjectID, "job")
		if err == nil {
			t.Fatal("expected unapproved project to be blocked")
		}
		errs, ok := err.(validation.Errors)
		if !ok {
			t.Fatalf("expected validation.Errors, got %T: %v", err, err)
		}
		jobErr, ok := errs["job"]
		if !ok {
			t.Fatalf("expected job field project authorization error, got %v", errs)
		}
		codeErr, ok := jobErr.(validation.Error)
		if !ok || codeErr.Code() != ProjectAuthorizationNotApprovedCode {
			t.Fatalf("expected project authorization code, got %T %v", jobErr, jobErr)
		}
	})

	t.Run("enabled enforcement allows approved project", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, true)

		if err := EnsureProjectAuthorizationApprovedForJob(app, paGateApprovedProjectID, "job"); err != nil {
			t.Fatalf("expected approved project to be allowed, got %v", err)
		}
	})

	t.Run("enabled enforcement blocks legacy PO project", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, true)

		err := EnsureProjectAuthorizationApprovedForJob(app, paGateLegacyPOProjectID, "job")
		assertProjectAuthorizationGateValidationError(t, err, "job")
	})

	t.Run("enabled enforcement ignores proposals", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, true)

		if err := EnsureProjectAuthorizationApprovedForJob(app, paGateProposalID, "job"); err != nil {
			t.Fatalf("expected proposal to bypass project authorization gate, got %v", err)
		}
	})
}

func TestProjectAuthorizationSaveInvariantBlocksUntrustedModelSaves(t *testing.T) {
	protectedFields := map[string]any{
		"project_authorization_doc_hash": strings.Repeat("b", 64),
		"pa_uploader":                    "f2j5a8vk006baub",
		"pa_uploaded":                    "2026-06-03 10:00:00.000Z",
		"pa_rejector":                    "f2j5a8vk006baub",
		"pa_rejected":                    "2026-06-03 11:00:00.000Z",
		"pa_rejection_reason":            "Missing signature",
	}
	for field, value := range protectedFields {
		t.Run(field, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			AddHooks(app)

			job, err := app.FindRecordById("jobs", paGateMissingProjectID)
			if err != nil {
				t.Fatalf("failed to load job: %v", err)
			}
			job.Set(field, value)

			err = app.Save(job)
			assertProjectAuthorizationHookError(t, err, field, "not_editable")

			reloaded, err := app.FindRecordById("jobs", paGateMissingProjectID)
			if err != nil {
				t.Fatalf("failed to reload job: %v", err)
			}
			if reloaded.GetString(field) != "" {
				t.Fatalf("untrusted save mutated %s to %q", field, reloaded.GetString(field))
			}
		})
	}
}

func TestProjectAuthorizationSaveInvariantAllowsTrustedApprovalAndRevocation(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddHooks(app)

	job, err := app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	job.Set("pa_reviewed", "2026-06-03 12:00:00.000Z")
	job.Set("pa_reviewer", "f2j5a8vk006baub")
	err = app.Save(job)
	assertProjectAuthorizationHookError(t, err, "pa_reviewer", "not_editable")

	job, err = app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	job.Set("pa_reviewed", "2026-06-03 12:00:00.000Z")
	job.Set("pa_reviewer", "f2j5a8vk006baub")
	if err := app.SaveWithContext(WithProjectAuthorizationMutation(context.Background(), ProjectAuthorizationMutationApprove), job); err != nil {
		t.Fatalf("trusted approval save failed: %v", err)
	}

	job, err = app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to reload approved job: %v", err)
	}
	if job.GetString("pa_reviewed") == "" || job.GetString("pa_reviewer") != "f2j5a8vk006baub" {
		t.Fatalf("trusted approval did not persist review fields: reviewed=%q reviewer=%q", job.GetString("pa_reviewed"), job.GetString("pa_reviewer"))
	}

	job.Set("pa_reviewed", "")
	job.Set("pa_reviewer", "")
	if err := app.SaveWithContext(WithProjectAuthorizationMutation(context.Background(), ProjectAuthorizationMutationRevoke), job); err != nil {
		t.Fatalf("trusted revocation save failed: %v", err)
	}
	job, err = app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to reload revoked job: %v", err)
	}
	if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
		t.Fatalf("trusted revocation did not clear review fields: reviewed=%q reviewer=%q", job.GetString("pa_reviewed"), job.GetString("pa_reviewer"))
	}
	if job.GetString("project_authorization_doc") == "" || job.GetString("project_authorization_doc_hash") == "" {
		t.Fatalf("trusted revocation should preserve document and hash")
	}
}

func TestProjectAuthorizationSaveInvariantAllowsTrustedDeleteOnlyWhenUnapproved(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddHooks(app)

	job, err := app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	job.Set("project_authorization_doc", "")
	job.Set("project_authorization_doc_hash", "")
	job.Set("pa_uploader", "")
	job.Set("pa_uploaded", "")
	job.Set("pa_rejector", "")
	job.Set("pa_rejected", "")
	job.Set("pa_rejection_reason", "")
	if err := app.SaveWithContext(WithProjectAuthorizationMutation(context.Background(), ProjectAuthorizationMutationDelete), job); err != nil {
		t.Fatalf("trusted delete save failed: %v", err)
	}
	job, err = app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to reload deleted job: %v", err)
	}
	if job.GetString("project_authorization_doc") != "" || job.GetString("project_authorization_doc_hash") != "" {
		t.Fatalf("trusted delete did not clear document and hash: doc=%q hash=%q", job.GetString("project_authorization_doc"), job.GetString("project_authorization_doc_hash"))
	}
	if job.GetString("pa_uploader") != "" || job.GetString("pa_uploaded") != "" {
		t.Fatalf("trusted delete did not clear upload metadata: uploader=%q uploaded=%q", job.GetString("pa_uploader"), job.GetString("pa_uploaded"))
	}
}

func TestProjectAuthorizationSaveInvariantAllowsTrustedRejection(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddHooks(app)

	job, err := app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	job.Set("pa_rejected", "2026-06-03 13:00:00.000Z")
	job.Set("pa_rejector", "f2j5a8vk006baub")
	job.Set("pa_rejection_reason", "Missing signature")
	if err := app.Save(job); err == nil {
		t.Fatal("expected untrusted rejection metadata save to fail")
	}

	job, err = app.FindRecordById("jobs", paGatePendingProjectID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	job.Set("pa_rejected", "2026-06-03 13:00:00.000Z")
	job.Set("pa_rejector", "f2j5a8vk006baub")
	job.Set("pa_rejection_reason", "Missing signature")
	if err := app.SaveWithContext(WithProjectAuthorizationMutation(context.Background(), ProjectAuthorizationMutationReject), job); err != nil {
		t.Fatalf("trusted rejection save failed: %v", err)
	}
}

func TestProjectAuthorizationSaveInvariantRejectsTrustedUploadForApprovedJob(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	AddHooks(app)

	job, err := app.FindRecordById("jobs", paGateApprovedProjectID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	job.Set("project_authorization_doc", "replacement-pa.pdf")
	job.Set("project_authorization_doc_hash", strings.Repeat("c", 64))
	job.Set("pa_reviewed", "")
	job.Set("pa_reviewer", "")
	err = app.SaveWithContext(WithProjectAuthorizationMutation(context.Background(), ProjectAuthorizationMutationUpload), job)
	assertProjectAuthorizationHookError(t, err, "project_authorization_doc", "project_authorization_approved_immutable")
}

func TestProjectAuthorizationPurchaseOrderGate(t *testing.T) {
	scenarios := []struct {
		name     string
		enforce  bool
		approved bool
		legacy   bool
		blocked  bool
	}{
		{name: "disabled enforcement allows unapproved project", enforce: false},
		{name: "enabled enforcement blocks unapproved project", enforce: true, blocked: true},
		{name: "enabled enforcement blocks legacy PO project", enforce: true, legacy: true, blocked: true},
		{name: "enabled enforcement allows approved project", enforce: true, approved: true},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
				t.Fatalf("failed to load expenditure kinds config: %v", err)
			}
			setProjectAuthorizationEnforcementForTest(t, app, scenario.enforce)
			jobID := paGateMissingProjectID
			if scenario.approved {
				jobID = paGateApprovedProjectID
			} else if scenario.legacy {
				jobID = paGateLegacyPOProjectID
			}

			record := buildRecordFromMap(poCollection, map[string]any{
				"uid":          "f2j5a8vk006baub",
				"date":         "2024-09-01",
				"branch":       "80875lm27v8wgi4",
				"division":     "vccd5fo56ctbigh",
				"description":  "PA gate purchase order",
				"payment_type": "OnAccount",
				"total":        100.0,
				"vendor":       "2zqxtsmymf670ha",
				"approver":     "f2j5a8vk006baub",
				"type":         "One-Time",
				"job":          jobID,
				"kind":         utilities.DefaultProjectExpenditureKindID(),
			})

			err := ValidatePurchaseOrder(app, record, true)
			if scenario.blocked {
				assertProjectAuthorizationGateValidationError(t, err, "job")
				return
			}
			if err != nil {
				t.Fatalf("expected purchase order to pass PA gate, got %v", err)
			}
		})
	}
}

func TestProjectAuthorizationExpenseGate(t *testing.T) {
	scenarios := []struct {
		name     string
		enforce  bool
		approved bool
		legacy   bool
		blocked  bool
	}{
		{name: "disabled enforcement allows unapproved project", enforce: false},
		{name: "enabled enforcement blocks unapproved project", enforce: true, blocked: true},
		{name: "enabled enforcement blocks legacy PO project", enforce: true, legacy: true, blocked: true},
		{name: "enabled enforcement allows approved project", enforce: true, approved: true},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()
			if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
				t.Fatalf("failed to load expenditure kinds config: %v", err)
			}
			setProjectAuthorizationEnforcementForTest(t, app, scenario.enforce)
			jobID := paGateMissingProjectID
			if scenario.approved {
				jobID = paGateApprovedProjectID
			} else if scenario.legacy {
				jobID = paGateLegacyPOProjectID
			}

			record := buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-09-01",
				"description":     "PA gate mileage expense",
				"division":        "vccd5fo56ctbigh",
				"job":             jobID,
				"payment_type":    "Mileage",
				"purchase_order":  "",
				"total":           constants.NO_PO_EXPENSE_LIMIT + 100,
				"distance":        100.0,
				"vendor":          "2zqxtsmymf670ha",
				"kind":            utilities.DefaultProjectExpenditureKindID(),
			})

			err := validateExpense(app, record, nil, 0, false)
			if scenario.blocked {
				assertProjectAuthorizationGateValidationError(t, err, "job")
				return
			}
			if err != nil {
				t.Fatalf("expected expense to pass PA gate, got %v", err)
			}
		})
	}
}

func assertProjectAuthorizationHookError(t *testing.T, err error, field string, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected PA hook error for %s", field)
	}
	var hookErr *errs.HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected HookError, got %T: %v", err, err)
	}
	fieldErr, ok := hookErr.Data[field]
	if !ok {
		t.Fatalf("expected hook error field %s, got %+v", field, hookErr.Data)
	}
	if fieldErr.Code != code {
		t.Fatalf("hook error code for %s = %s, want %s", field, fieldErr.Code, code)
	}
}

func setProjectAuthorizationEnforcementForTest(t *testing.T, app *tests.TestApp, enabled bool) {
	t.Helper()
	record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatalf("failed to load jobs app_config: %v", err)
	}
	if enabled {
		record.Set("value", `{"create_edit_absorb": true, "enforce_project_authorization": true}`)
	} else {
		record.Set("value", `{"create_edit_absorb": true, "enforce_project_authorization": false}`)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save jobs app_config: %v", err)
	}
}

func assertProjectAuthorizationGateValidationError(t *testing.T, err error, field string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected project authorization validation error, got nil")
	}
	errs, ok := err.(validation.Errors)
	if !ok {
		t.Fatalf("expected validation.Errors, got %T: %v", err, err)
	}
	fieldErr, ok := errs[field]
	if !ok {
		t.Fatalf("expected %s project authorization error, got %v", field, errs)
	}
	codeErr, ok := fieldErr.(validation.Error)
	if !ok || codeErr.Code() != ProjectAuthorizationNotApprovedCode {
		t.Fatalf("expected project authorization code, got %T %v", fieldErr, fieldErr)
	}
}
