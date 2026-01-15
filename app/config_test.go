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
				"allocations": []
			}`),
			Headers: map[string]string{"Authorization": recordToken},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				// Disable job editing by updating the config
				record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatalf("failed to find app_config record: %v", err)
				}
				record.Set("value", `{"create_edit_absorb": false}`)
				if err := app.Save(record); err != nil {
					t.Fatalf("failed to update app_config: %v", err)
				}
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
				`"job editing is currently disabled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
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
			Headers: map[string]string{"Authorization": recordToken},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				// Disable job editing by updating the config
				record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatalf("failed to find app_config record: %v", err)
				}
				record.Set("value", `{"create_edit_absorb": false}`)
				if err := app.Save(record); err != nil {
					t.Fatalf("failed to update app_config: %v", err)
				}
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
				`"job editing is currently disabled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
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

	scenarios := []tests.ApiScenario{
		{
			Name:   "job update via custom API blocked when editing disabled",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + jobID,
			Body: strings.NewReader(`{
				"job": {
					"description": "Updated description when disabled"
				},
				"allocations": []
			}`),
			Headers: map[string]string{"Authorization": recordToken},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				// Disable job editing
				record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatalf("failed to find app_config record: %v", err)
				}
				record.Set("value", `{"create_edit_absorb": false}`)
				if err := app.Save(record); err != nil {
					t.Fatalf("failed to update app_config: %v", err)
				}
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "job update via PocketBase API blocked when editing disabled",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/" + jobID,
			Body: strings.NewReader(`{
				"description": "Updated description when disabled"
			}`),
			Headers: map[string]string{"Authorization": recordToken},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				// Disable job editing
				record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatalf("failed to find app_config record: %v", err)
				}
				record.Set("value", `{"create_edit_absorb": false}`)
				if err := app.Save(record); err != nil {
					t.Fatalf("failed to update app_config: %v", err)
				}
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"jobs_editing_disabled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
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
			Headers: map[string]string{"Authorization": recordToken},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				// Disable job editing
				record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatalf("failed to find app_config record: %v", err)
				}
				record.Set("value", `{"create_edit_absorb": false}`)
				if err := app.Save(record); err != nil {
					t.Fatalf("failed to update app_config: %v", err)
				}
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`Job editing is disabled`,
			},
			TestAppFactory: testutils.SetupTestApp,
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
					"authorizing_document": "PA"
				},
				"allocations": []
			}`),
			Headers: map[string]string{"Authorization": recordToken},
			BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
				// Ensure job editing is enabled
				record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
				if err != nil {
					t.Fatalf("failed to find app_config record: %v", err)
				}
				record.Set("value", `{"create_edit_absorb": true}`)
				if err := app.Save(record); err != nil {
					t.Fatalf("failed to update app_config: %v", err)
				}
			},
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
