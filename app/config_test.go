package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func setupJobsEditingDisabledApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app := testutils.SetupTestApp(t)
	record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatalf("failed to find app_config record: %v", err)
	}
	record.Set("value", `{"create_edit_absorb": false}`)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to update app_config: %v", err)
	}
	return app
}

func setupExpensesEditingDisabledApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app := testutils.SetupTestApp(t)
	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("key", "expenses")
	record.Set("value", `{"create_edit_absorb": false}`)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to create expenses app_config: %v", err)
	}
	return app
}

func setupTimeEditingDisabledApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app := testutils.SetupTestApp(t)
	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "time")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "time")
	}
	record.Set("value", `{"create_edit": false}`)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save time app_config: %v", err)
	}
	return app
}

// TestGetConfigValue verifies the GetConfigValue function retrieves config correctly
func TestGetConfigValue(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	t.Run("returns config for existing domain key", func(t *testing.T) {
		config, err := utilities.GetConfigValue(app, "jobs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config == nil {
			t.Fatal("expected config to be non-nil")
		}
		// Check that create_edit_absorb exists in the config
		if _, ok := config["create_edit_absorb"]; !ok {
			t.Error("expected create_edit_absorb to exist in config")
		}
	})

	t.Run("returns nil for non-existent domain key", func(t *testing.T) {
		config, err := utilities.GetConfigValue(app, "nonexistent_domain")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if config != nil {
			t.Errorf("expected config to be nil, got %v", config)
		}
	})
}

// TestGetConfigBool verifies the GetConfigBool function extracts booleans correctly
func TestGetConfigBool(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	t.Run("returns boolean value for existing property", func(t *testing.T) {
		value, err := utilities.GetConfigBool(app, "jobs", "create_edit_absorb", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The test database has create_edit_absorb set to true
		if !value {
			t.Error("expected create_edit_absorb to be true")
		}
	})

	t.Run("returns default for non-existent domain", func(t *testing.T) {
		value, err := utilities.GetConfigBool(app, "nonexistent", "some_property", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !value {
			t.Error("expected default value true")
		}

		value, err = utilities.GetConfigBool(app, "nonexistent", "some_property", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if value {
			t.Error("expected default value false")
		}
	})

	t.Run("returns default for non-existent property", func(t *testing.T) {
		value, err := utilities.GetConfigBool(app, "jobs", "nonexistent_property", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !value {
			t.Error("expected default value true")
		}
	})
}

// TestIsJobsEditingEnabled verifies the convenience function works correctly
func TestIsJobsEditingEnabled(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	t.Run("returns true when create_edit_absorb is true", func(t *testing.T) {
		enabled, err := utilities.IsJobsEditingEnabled(app)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The test database has create_edit_absorb set to true
		if !enabled {
			t.Error("expected job editing to be enabled")
		}
	})
}

// TestJobCreationBlockedWhenEditingDisabled verifies that job creation fails when
// job editing is disabled via app_config
func TestJobCreationBlockedWhenEditingDisabled(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const divisionID = "fy4i9poneukvq9u"

	scenarios := []tests.ApiScenario{
		{
			Name:   "job creation via custom API blocked when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job creation when disabled",
					"location": "8FW4V75J+QQ",
					"branch": "f2j5a8vk006baub",
					"client": "lb0fnenkeyitsny",
					"project_award_date": "2025-01-15",
					"authorizing_document": "Email"
				},
				"allocations": [
					{ "division": "` + divisionID + `", "hours": 10 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
				`"job editing is currently disabled"`,
			},
			TestAppFactory: setupJobsEditingDisabledApp,
		},
		{
			Name:   "job creation via PocketBase API blocked when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/collections/jobs/records",
			Body: strings.NewReader(`{
				"description": "Test job creation when disabled",
				"location": "8FW4V75J+QQ",
				"branch": "f2j5a8vk006baub",
				"client": "lb0fnenkeyitsny",
				"project_award_date": "2025-01-15",
				"authorizing_document": "Email"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
				`"job editing is currently disabled"`,
			},
			TestAppFactory: setupJobsEditingDisabledApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestJobUpdateBlockedWhenEditingDisabled verifies that job updates fail when
// job editing is disabled via app_config
func TestJobUpdateBlockedWhenEditingDisabled(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const jobID = "cjf0kt0defhq480"
	const divisionID = "fy4i9poneukvq9u"

	scenarios := []tests.ApiScenario{
		{
			Name:   "job update via custom API blocked when editing disabled",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + jobID,
			Body: strings.NewReader(`{
				"job": {
					"description": "Updated description when disabled"
				},
				"allocations": [
					{ "division": "` + divisionID + `", "hours": 10 }
				]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
			},
			TestAppFactory: setupJobsEditingDisabledApp,
		},
		{
			Name:   "job update via PocketBase API blocked when editing disabled",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/" + jobID,
			Body: strings.NewReader(`{
				"description": "Updated description when disabled"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
			},
			TestAppFactory: setupJobsEditingDisabledApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestClientAbsorbBlockedWhenJobEditingDisabled verifies that absorbing clients fails
// when job editing is disabled (since clients absorb modifies jobs.client and jobs.job_owner)
func TestClientAbsorbBlockedWhenJobEditingDisabled(t *testing.T) {
	// Use a user with the 'absorb' claim: book@keeper.com
	recordToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "clients absorb blocked when job editing disabled",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body: strings.NewReader(`{
				"ids_to_absorb": ["eldtxi3i4h00k8r"]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`Job editing is disabled`,
			},
			TestAppFactory: setupJobsEditingDisabledApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestJobCreationAllowedWhenEditingEnabled is a control test verifying that
// job creation works when editing is enabled
func TestJobCreationAllowedWhenEditingEnabled(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Use valid test data IDs from the test database
	const branchID = "1r7r6hyp681vi15"
	const clientID = "lb0fnenkeyitsny"
	const contactID = "nh5u9z3cyknjclv" // belongs to clientID
	const managerID = "f2j5a8vk006baub" // valid manager from existing jobs
	const activeRateSheet = "c41ofep525bcacj"
	const divisionID = "fy4i9poneukvq9u"

	scenarios := []tests.ApiScenario{
		{
			Name:   "job creation succeeds when editing enabled",
			Method: http.MethodPost,
			URL:    "/api/jobs",
			Body: strings.NewReader(`{
				"job": {
					"description": "Test job creation when enabled",
					"location": "8FW4V75J+QQ",
					"branch": "` + branchID + `",
					"client": "` + clientID + `",
					"contact": "` + contactID + `",
					"manager": "` + managerID + `",
					"project_award_date": "2025-01-15",
					"authorizing_document": "PA",
					"rate_sheet": "` + activeRateSheet + `"
				},
				"allocations": [
					{ "division": "` + divisionID + `", "hours": 10 }
				]
			}`),
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"id":"`},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestDefaultBehaviorWhenConfigMissing verifies that the system fails-open
// (allows editing) when the config record is missing
func TestDefaultBehaviorWhenConfigMissing(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Delete the config record
	record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatalf("failed to find app_config record: %v", err)
	}
	if err := app.Delete(record); err != nil {
		t.Fatalf("failed to delete app_config record: %v", err)
	}

	// Now check that IsJobsEditingEnabled returns true (fail-open)
	enabled, err := utilities.IsJobsEditingEnabled(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected job editing to be enabled (fail-open) when config is missing")
	}
}

// TestDefaultBehaviorWhenPropertyMissing verifies that the system fails-open
// when the property is missing from the config JSON
func TestDefaultBehaviorWhenPropertyMissing(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Update the config to remove the create_edit_absorb property
	record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		t.Fatalf("failed to find app_config record: %v", err)
	}
	record.Set("value", `{"some_other_property": true}`)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to update app_config: %v", err)
	}

	// Check that IsJobsEditingEnabled returns true (fail-open)
	enabled, err := utilities.IsJobsEditingEnabled(app)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !enabled {
		t.Error("expected job editing to be enabled (fail-open) when property is missing")
	}
}

// TestIsExpensesEditingEnabled verifies the convenience function works correctly
func TestIsExpensesEditingEnabled(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	t.Run("returns true (fail-open) when config missing", func(t *testing.T) {
		enabled, err := utilities.IsExpensesEditingEnabled(app)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !enabled {
			t.Error("expected expenses editing to be enabled when config is missing")
		}
	})
}

// TestExpensesEditingDisabledBlocks verifies that expense, PO, and vendor
// create/update/delete via PocketBase API all fail when expense editing is
// disabled via app_config.
func TestExpensesEditingDisabledBlocks(t *testing.T) {
	const vendorID = "2zqxtsmymf670ha"

	// Each sub-table defines a user, and the scenarios that user should exercise.
	type userScenarios struct {
		email     string
		scenarios []tests.ApiScenario
	}

	groups := []userScenarios{
		{
			email: "time@test.com",
			scenarios: []tests.ApiScenario{
				{
					Name:   "expense creation blocked when editing disabled",
					Method: http.MethodPost,
					URL:    "/api/collections/expenses/records",
					Body: strings.NewReader(`{
						"uid": "rzr98oadsp9qc11",
						"date": "2025-01-15",
						"division": "fy4i9poneukvq9u",
						"description": "test expense blocked",
						"payment_type": "Expense",
						"total": 50.00
					}`),
				},
				{
					Name:   "PO creation blocked when editing disabled",
					Method: http.MethodPost,
					URL:    "/api/collections/purchase_orders/records",
					Body: strings.NewReader(`{
						"date": "2025-01-15",
						"division": "fy4i9poneukvq9u",
						"vendor": "2zqxtsmymf670ha",
						"description": "test PO blocked",
						"payment_type": "OnAccount",
						"total": 100.00,
						"type": "One-Time",
						"status": "Unapproved",
						"approver": "f2j5a8vk006baub"
					}`),
				},
			},
		},
		{
			// book@keeper.com has the payables_admin claim required for vendor CRUD
			email: "book@keeper.com",
			scenarios: []tests.ApiScenario{
				{
					Name:   "vendor creation blocked when editing disabled",
					Method: http.MethodPost,
					URL:    "/api/collections/vendors/records",
					Body: strings.NewReader(`{
						"name": "New Vendor Blocked",
						"alias": "NVB",
						"status": "Active"
					}`),
				},
				{
					Name:   "vendor update blocked when editing disabled",
					Method: http.MethodPatch,
					URL:    "/api/collections/vendors/records/" + vendorID,
					Body:   strings.NewReader(`{"alias": "Updated"}`),
				},
				{
					Name:   "vendor delete blocked when editing disabled",
					Method: http.MethodDelete,
					URL:    "/api/collections/vendors/records/" + vendorID,
				},
			},
		},
	}

	for _, g := range groups {
		recordToken, err := testutils.GenerateRecordToken("users", g.email)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range g.scenarios {
			// Apply the shared defaults so each scenario entry stays minimal.
			s.Headers = map[string]string{"Authorization": recordToken}
			s.ExpectedStatus = 403
			s.ExpectedContent = []string{`"expenses_editing_disabled"`}
			s.TestAppFactory = setupExpensesEditingDisabledApp
			s.Test(t)
		}
	}

	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	expenseUncommitScenario := tests.ApiScenario{
		Name:           "expense uncommit blocked when editing disabled",
		Method:         http.MethodPost,
		URL:            "/api/expenses/xg2yeucklhgbs3n/uncommit",
		Headers:        map[string]string{"Authorization": adminToken},
		ExpectedStatus: 403,
		ExpectedContent: []string{
			`"Expense editing is currently disabled."`,
		},
		TestAppFactory: setupExpensesEditingDisabledApp,
	}
	expenseUncommitScenario.Test(t)
}

// TestVendorAbsorbBlockedWhenEditingDisabled verifies that absorbing vendors fails
// when expense editing is disabled
func TestVendorAbsorbBlockedWhenEditingDisabled(t *testing.T) {
	// Use a user with the 'absorb' claim: book@keeper.com
	recordToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "vendor absorb blocked when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/vendors/2zqxtsmymf670ha/absorb",
			Body: strings.NewReader(`{
				"ids_to_absorb": ["ctswqva5onxj75q"]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"Expense editing is currently disabled."`,
			},
			TestAppFactory: setupExpensesEditingDisabledApp,
		},
		{
			Name:           "vendor undo absorb blocked when editing disabled",
			Method:         http.MethodPost,
			URL:            "/api/vendors/undo_absorb",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"Expense editing is currently disabled."`,
			},
			TestAppFactory: setupExpensesEditingDisabledApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimeEditingDisabledBlocks(t *testing.T) {
	uNoClaimsToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	inactiveMgrToken, err := testutils.GenerateRecordToken("users", "has_inactive_mgr@test.com")
	if err != nil {
		t.Fatal(err)
	}
	approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	committerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry creation blocked when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "u_no_claims",
				"time_type": "d35auo4vawx7t9u",
				"date": "2024-01-08",
				"description": "blocked time entry create",
				"hours": 1
			}`),
			Headers:        map[string]string{"Authorization": uNoClaimsToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"time_editing_disabled"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:           "time entry update blocked when editing disabled",
			Method:         http.MethodPatch,
			URL:            "/api/collections/time_entries/records/r464ccf9b3527eb",
			Body:           strings.NewReader(`{"description":"blocked update"}`),
			Headers:        map[string]string{"Authorization": uNoClaimsToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"time_editing_disabled"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:           "time entry delete blocked when editing disabled",
			Method:         http.MethodDelete,
			URL:            "/api/collections/time_entries/records/r464ccf9b3527eb",
			Headers:        map[string]string{"Authorization": uNoClaimsToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"time_editing_disabled"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:           "copy to tomorrow blocked when editing disabled",
			Method:         http.MethodPost,
			URL:            "/api/time_entries/r464ccf9b3527eb/copy_to_tomorrow",
			Headers:        map[string]string{"Authorization": uNoClaimsToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"Time editing is currently disabled."`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:           "bundle timesheet blocked when editing disabled",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/2024-09-14/bundle",
			Headers:        map[string]string{"Authorization": inactiveMgrToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"Time editing is currently disabled."`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:           "timesheet approve blocked when editing disabled",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/aeyl94og4xmnpq4/approve",
			Headers:        map[string]string{"Authorization": approverToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"Time editing is currently disabled."`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:   "timesheet reject remains allowed when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/time_sheets/aeyl94og4xmnpq4/reject",
			Body:   strings.NewReader(`{"rejection_reason":"still allowed when disabled"}`),
			Headers: map[string]string{
				"Authorization": approverToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"record rejected successfully"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:   "timesheet commit remains allowed when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/time_sheets/aeyl94og4xmnpq4/commit",
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"record_not_approved"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:   "timesheet unbundle remains allowed when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/time_sheets/aeyl94og4xmnpq4/unbundle",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Time sheet unbundled successfully"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:           "timesheet uncommit blocked when editing disabled",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/j1lr2oddjongtoj/uncommit",
			Headers:        map[string]string{"Authorization": approverToken},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"Time editing is currently disabled."`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:   "time amendment create blocked when editing disabled",
			Method: http.MethodPost,
			URL:    "/api/collections/time_amendments/records",
			Body: strings.NewReader(`{
				"creator": "f2j5a8vk006baub",
				"time_type": "sdyfl3q7j7ap849",
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-02",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "time amendment blocked",
				"hours": 1,
				"skip_tsid_check": true,
				"week_ending": "2006-01-02"
			}`),
			Headers: map[string]string{
				"Authorization": approverToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"time_editing_disabled"`,
				`"time editing is currently disabled"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:   "time amendment update blocked when editing disabled",
			Method: http.MethodPatch,
			URL:    "/api/collections/time_amendments/records/qn4jyrkxp3pfjom",
			Body: strings.NewReader(`{
				"description": "time amendment update blocked"
			}`),
			Headers: map[string]string{
				"Authorization": approverToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"time_editing_disabled"`,
				`"time editing is currently disabled"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
		{
			Name:   "time amendment delete blocked when editing disabled",
			Method: http.MethodDelete,
			URL:    "/api/collections/time_amendments/records/qn4jyrkxp3pfjom",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"time_editing_disabled"`,
				`"time editing is currently disabled"`,
			},
			TestAppFactory: setupTimeEditingDisabledApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
