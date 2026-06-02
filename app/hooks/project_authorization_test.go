package hooks

import (
	"testing"
	"tybalt/constants"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	paGateProjectID  = "cjf0kt0defhq480"
	paGateProposalID = "kyed8fha9uaha27"
)

func TestEnsureProjectAuthorizationApprovedForJob(t *testing.T) {
	t.Run("disabled enforcement allows unapproved project", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, false)

		if err := EnsureProjectAuthorizationApprovedForJob(app, paGateProjectID, "job"); err != nil {
			t.Fatalf("expected disabled enforcement to allow project, got %v", err)
		}
	})

	t.Run("enabled enforcement blocks unapproved project", func(t *testing.T) {
		app := testseed.NewSeededTestApp(t)
		defer app.Cleanup()
		setProjectAuthorizationEnforcementForTest(t, app, true)

		err := EnsureProjectAuthorizationApprovedForJob(app, paGateProjectID, "job")
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
		setProjectAuthorizationApprovedForTest(t, app, paGateProjectID)

		if err := EnsureProjectAuthorizationApprovedForJob(app, paGateProjectID, "job"); err != nil {
			t.Fatalf("expected approved project to be allowed, got %v", err)
		}
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

func TestProjectAuthorizationPurchaseOrderGate(t *testing.T) {
	scenarios := []struct {
		name     string
		enforce  bool
		approved bool
		blocked  bool
	}{
		{name: "disabled enforcement allows unapproved project", enforce: false},
		{name: "enabled enforcement blocks unapproved project", enforce: true, blocked: true},
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
			if scenario.approved {
				setProjectAuthorizationApprovedForTest(t, app, paGateProjectID)
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
				"job":          paGateProjectID,
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
		blocked  bool
	}{
		{name: "disabled enforcement allows unapproved project", enforce: false},
		{name: "enabled enforcement blocks unapproved project", enforce: true, blocked: true},
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
			if scenario.approved {
				setProjectAuthorizationApprovedForTest(t, app, paGateProjectID)
			}

			record := buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-09-01",
				"description":     "PA gate mileage expense",
				"division":        "vccd5fo56ctbigh",
				"job":             paGateProjectID,
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

func setProjectAuthorizationApprovedForTest(t *testing.T, app *tests.TestApp, jobID string) {
	t.Helper()
	if _, err := app.DB().NewQuery(`
		UPDATE jobs
		SET project_authorization_doc = 'approved-pa.pdf',
				project_authorization_doc_hash = '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
				pa_reviewed = '2026-06-02 12:00:00.000Z',
				pa_reviewer = 'f2j5a8vk006baub'
		WHERE id = {:id}
	`).Bind(map[string]any{"id": jobID}).Execute(); err != nil {
		t.Fatalf("failed to approve PA fixture: %v", err)
	}
}
