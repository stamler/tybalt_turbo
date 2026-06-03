package main

import (
	"net/http"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

// TestBundleTimesheet_InactiveManagerFails verifies that bundling a timesheet
// fails when the user's manager (who becomes the approver) is inactive.
func TestBundleTimesheet_InactiveManagerFails(t *testing.T) {
	noClaimsToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	// User has_inactive_mgr@test.com has a profile with manager = u_inactive
	recordToken, err := testutils.GenerateRecordToken("users", "has_inactive_mgr@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "bundle timesheet requires time claim",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/2024-01-13/bundle",
			Headers:        map[string]string{"Authorization": noClaimsToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"Time claim required."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "bundle timesheet fails when manager is inactive",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/2024-09-14/bundle", // A Saturday (week ending)
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"approver_not_active"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestBundleTimesheet_SelfManagerWithTaprAutoApproves verifies that when a user
// is their own manager and holds the tapr claim, bundling their timesheet sets
// the `approved` timestamp automatically (no separate approve call required).
//
// Fixture: self_apv_yes@test.com — uid u_self_apv_yes whose profile.manager
// points at themselves and who holds the tapr claim.
func TestBundleTimesheet_SelfManagerWithTaprAutoApproves(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "self_apv_yes@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "self-managed user with tapr auto-approves on bundle",
		Method:         http.MethodPost,
		URL:            "/api/time_sheets/2024-09-14/bundle",
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"message":"Time sheet processed successfully"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			ts, err := app.FindFirstRecordByFilter(
				"time_sheets",
				"uid = {:uid} && week_ending = {:weekEnding}",
				dbx.Params{"uid": "u_self_apv_yes", "weekEnding": "2024-09-14"},
			)
			if err != nil {
				tb.Fatalf("failed to find newly bundled time sheet: %v", err)
			}
			if ts.GetString("approver") != "u_self_apv_yes" {
				tb.Fatalf("approver = %q, want %q", ts.GetString("approver"), "u_self_apv_yes")
			}
			if ts.GetDateTime("approved").IsZero() {
				tb.Fatalf("expected approved timestamp to be set on auto-approved time sheet")
			}
		},
	}

	scenario.Test(t)
}

// TestBundleTimesheet_SelfManagerWithoutTaprFails verifies that a user who is
// their own manager but does NOT hold the tapr claim cannot bundle (and thus
// cannot self-approve).
//
// Fixture: self_apv_no@test.com — uid u_self_apv_no whose profile.manager
// points at themselves but who does NOT hold the tapr claim.
func TestBundleTimesheet_SelfManagerWithoutTaprFails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "self_apv_no@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "self-managed user without tapr cannot bundle",
		Method:         http.MethodPost,
		URL:            "/api/time_sheets/2024-09-14/bundle",
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"code":"unqualified_approver"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestBundleTimesheet_ProjectAuthorizationGate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "self_apv_yes@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []struct {
		name            string
		enforce         bool
		approved        bool
		expectedStatus  int
		expectedContent []string
	}{
		{
			name:           "disabled enforcement allows unapproved project",
			expectedStatus: http.StatusOK,
			expectedContent: []string{
				`"message":"Time sheet processed successfully"`,
			},
		},
		{
			name:           "enabled enforcement blocks unapproved project",
			enforce:        true,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedContent: []string{
				`"code":"` + hooks.ProjectAuthorizationNotApprovedCode + `"`,
				`"blocking_jobs":[`,
				`"id":"pafixmissing01"`,
				`"manager_name":"Horace Silver"`,
			},
		},
		{
			name:           "enabled enforcement allows approved project",
			enforce:        true,
			approved:       true,
			expectedStatus: http.StatusOK,
			expectedContent: []string{
				`"message":"Time sheet processed successfully"`,
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			apiScenario := tests.ApiScenario{
				Name:            scenario.name,
				Method:          http.MethodPost,
				URL:             "/api/time_sheets/2024-09-14/bundle",
				Headers:         map[string]string{"Authorization": recordToken},
				ExpectedStatus:  scenario.expectedStatus,
				ExpectedContent: scenario.expectedContent,
				TestAppFactory: func(tb testing.TB) *tests.TestApp {
					return setupProjectAuthorizationBundleGateApp(tb, scenario.enforce, scenario.approved)
				},
			}
			apiScenario.Test(t)
		})
	}
}

func setupProjectAuthorizationBundleGateApp(tb testing.TB, enforce bool, approved bool) *tests.TestApp {
	tb.Helper()
	app := testutils.SetupTestApp(tb)
	setProjectAuthorizationBundleGateConfig(tb, app, enforce)
	jobID := "pafixmissing01"
	if approved {
		jobID = "pafixapprove01"
	}
	if _, err := app.DB().NewQuery(`
		UPDATE time_entries
		SET job = {:job},
		    role = 'tbgoiwwwfj8cvju',
		    description = 'PA gate project time',
		    tsid = ''
		WHERE id = 'te_self_apv_yes_001'
	`).Bind(map[string]any{"job": jobID}).Execute(); err != nil {
		tb.Fatalf("failed to point self-approver time entry at PA-gated project: %v", err)
	}
	return app
}

func setProjectAuthorizationBundleGateConfig(tb testing.TB, app *tests.TestApp, enabled bool) {
	tb.Helper()
	record, err := app.FindFirstRecordByData("app_config", "key", "jobs")
	if err != nil {
		tb.Fatalf("failed to load jobs app_config: %v", err)
	}
	if enabled {
		record.Set("value", `{"create_edit_absorb": true, "enforce_project_authorization": true}`)
	} else {
		record.Set("value", `{"create_edit_absorb": true, "enforce_project_authorization": false}`)
	}
	if err := app.Save(record); err != nil {
		tb.Fatalf("failed to save jobs app_config: %v", err)
	}
}
