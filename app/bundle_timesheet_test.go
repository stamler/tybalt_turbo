package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

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
