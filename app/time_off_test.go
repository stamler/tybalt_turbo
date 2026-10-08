package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	// Salaried supervisor (fakemanager@fakesite.xyz) who manages themself.
	timeOffSalariedUID = "wegviunlyr2jjjv"
	// Hourly direct report (book@keeper.com) of timeOffSalariedUID.
	timeOffHourlyUID = "tqqf7q0f3378rvp"
)

// includeTimeOffFixtureUsers puts both fixture users into the time_off view,
// which only lists employees with tracked time off and expected timesheets.
func includeTimeOffFixtureUsers(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
	tb.Helper()
	_, err := app.NonconcurrentDB().NewQuery(
		"UPDATE admin_profiles SET time_sheet_expected = 1, untracked_time_off = 0 WHERE uid IN ({:a}, {:b})",
	).Bind(map[string]any{"a": timeOffSalariedUID, "b": timeOffHourlyUID}).Execute()
	if err != nil {
		tb.Fatalf("failed to include fixture users in time_off: %v", err)
	}
}

func TestTimeOffSalaryLabelAccess(t *testing.T) {
	supervisorToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	reportToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "supervisor sees salaried status on own row",
			Method:         http.MethodGet,
			URL:            "/api/collections/time_off/records/" + timeOffSalariedUID,
			Headers:        map[string]string{"Authorization": supervisorToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + timeOffSalariedUID + `"`,
				`"salary":true`,
			},
			BeforeTestFunc: includeTimeOffFixtureUsers,
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "supervisor sees hourly status on direct report row",
			Method:         http.MethodGet,
			URL:            "/api/collections/time_off/records/" + timeOffHourlyUID,
			Headers:        map[string]string{"Authorization": supervisorToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + timeOffHourlyUID + `"`,
				`"salary":false`,
			},
			BeforeTestFunc: includeTimeOffFixtureUsers,
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "supervisor list includes salary for self and direct report only",
			Method:         http.MethodGet,
			URL:            "/api/collections/time_off/records",
			Headers:        map[string]string{"Authorization": supervisorToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"totalItems":2`,
				`"salary":true`,
				`"salary":false`,
			},
			BeforeTestFunc: includeTimeOffFixtureUsers,
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "employee cannot view their supervisor's row",
			Method:          http.MethodGet,
			URL:             "/api/collections/time_off/records/" + timeOffSalariedUID,
			Headers:         map[string]string{"Authorization": reportToken},
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"data":{}`},
			BeforeTestFunc:  includeTimeOffFixtureUsers,
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:           "employee list excludes their supervisor's row",
			Method:         http.MethodGet,
			URL:            "/api/collections/time_off/records",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"totalItems":1`,
				`"id":"` + timeOffHourlyUID + `"`,
			},
			NotExpectedContent: []string{`"id":"` + timeOffSalariedUID + `"`},
			BeforeTestFunc:     includeTimeOffFixtureUsers,
			TestAppFactory:     testutils.SetupTestApp,
		},
		{
			Name:            "unauthenticated request cannot view time off",
			Method:          http.MethodGet,
			URL:             "/api/collections/time_off/records/" + timeOffHourlyUID,
			ExpectedStatus:  http.StatusNotFound,
			ExpectedContent: []string{`"data":{}`},
			BeforeTestFunc:  includeTimeOffFixtureUsers,
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
