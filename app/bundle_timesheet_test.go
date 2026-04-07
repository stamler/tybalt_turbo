package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

// TestBundleTimesheet_InactiveManagerFails verifies that bundling a timesheet
// fails when the user's manager (who becomes the approver) is inactive.
func TestBundleTimesheet_InactiveManagerFails(t *testing.T) {
	// User has_inactive_mgr@test.com has a profile with manager = u_inactive
	recordToken, err := testutils.GenerateRecordToken("users", "has_inactive_mgr@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "bundle timesheet fails when manager is inactive",
		Method:         http.MethodPost,
		URL:            "/api/time_sheets/2024-09-14/bundle", // A Saturday (week ending)
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"code":"approver_not_active"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
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
