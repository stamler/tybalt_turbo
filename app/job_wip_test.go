package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
	"tybalt/internal/testutils"
	"tybalt/routes"

	"github.com/pocketbase/pocketbase/tests"
)

func TestJobWIP(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "valuation1@example.com")
	if err != nil {
		t.Fatal(err)
	}
	// Dedicated append-only CSV fixtures include future dates and invalid legacy
	// monetary data to test exclusions and incomplete values without runtime edits.
	cases := []struct {
		name, job string
		want      routes.JobWIP
	}{
		{"all factors without duplicate PO spend", "jobwip000000001", routes.JobWIP{
			ProjectValue: 10000, TimeValue: 849.5, Hours: 3.5, EstimatedHours: .5,
			ExpenseValue: 800, POValue: 3565, EstimatedPOs: 1,
		}},
		{"no sheet uses defaults and retains unpriced hours", "jobwip000000002", routes.JobWIP{
			ProjectValue: 10000, NoRateSheet: true, TimeValue: 1998, Hours: 3, EstimatedHours: 2, UnpricedHours: 1,
		}},
		{"missing settlement exchange rate and recurring approval", "jobwip000000003", routes.JobWIP{
			ProjectValue: 10000, UnpricedExpenses: 1, UnpricedPOs: 2,
		}},
		{"empty job with zero project value", "jobwip000000004", routes.JobWIP{}},
	}
	for _, tc := range cases {
		(&tests.ApiScenario{
			Name: tc.name, Method: http.MethodGet, URL: "/api/jobs/" + tc.job + "/wip",
			Headers: map[string]string{"Authorization": token}, ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{"{"},
			TestAppFactory:  testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()
				var got routes.JobWIP
				if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
					tb.Fatal(err)
				}
				if _, err := time.Parse("2006-01-02", got.AsOf); err != nil {
					tb.Errorf("invalid report date: %q", got.AsOf)
				}
				got.AsOf = ""
				if got != tc.want {
					tb.Errorf("WIP = %+v; want %+v", got, tc.want)
				}
			},
		}).Test(t)
	}
	for _, tc := range []struct {
		name, job, token string
		status           int
	}{
		{"authentication required", "jobwip000000001", "", http.StatusUnauthorized},
		{"unknown job", "missingjob00001", token, http.StatusNotFound},
	} {
		(&tests.ApiScenario{Name: tc.name, Method: http.MethodGet, URL: "/api/jobs/" + tc.job + "/wip", Headers: map[string]string{"Authorization": tc.token}, ExpectedStatus: tc.status, ExpectedContent: []string{"{"}, TestAppFactory: testutils.SetupTestApp}).Test(t)
	}
}
