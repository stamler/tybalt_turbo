package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/routes"

	"github.com/pocketbase/pocketbase/tests"
)

func TestJobWIPMissingRates(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "valuation1@example.com")
	if err != nil {
		t.Fatal(err)
	}
	sheet := &routes.WIPMissingRateSheet{ID: "rsvaluation0001", Name: "Summary valuation fixture", Revision: 0}
	// Existing CSV fixtures cover both fallback sources, blank roles, matched
	// rates, and future time. The assigned inactive revision must remain in use.
	for _, tc := range []struct {
		name, job, date string
		sheet           *routes.WIPMissingRateSheet
		roles           []routes.WIPMissingRateRole
	}{
		{"group missing rates and exclude blank roles and future time", "jobwipreason001", "2026-01-02", sheet, []routes.WIPMissingRateRole{{ID: "2bd6pxxof7p649x", Name: "Administration Staff", Hours: 7}}},
		{"honour the report date", "jobwipreason001", "2026-01-01", sheet, []routes.WIPMissingRateRole{}},
		{"blank entry role is not a missing sheet role", "jobwip000000001", "2026-01-02", sheet, []routes.WIPMissingRateRole{}},
		{"no sheet", "jobwip000000002", "2026-01-02", nil, []routes.WIPMissingRateRole{}},
		{"no time", "jobwip000000004", "2026-01-02", sheet, []routes.WIPMissingRateRole{}},
	} {
		(&tests.ApiScenario{
			Name: tc.name, Method: http.MethodGet,
			URL:     "/api/jobs/" + tc.job + "/wip/missing-rates?as_of=" + tc.date,
			Headers: map[string]string{"Authorization": token}, ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{"{"}, TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()
				var got routes.WIPMissingRates
				if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
					tb.Fatal(err)
				}
				want := routes.WIPMissingRates{AsOf: tc.date, RateSheet: tc.sheet, Roles: tc.roles}
				if !reflect.DeepEqual(got, want) {
					tb.Errorf("missing rates = %+v; want %+v", got, want)
				}
			},
		}).Test(t)
	}
	for _, tc := range []struct {
		name, job, query, token string
		status                  int
	}{
		{"authentication required", "jobwipreason001", "", "", http.StatusUnauthorized},
		{"unknown job", "missingjob00001", "", token, http.StatusNotFound},
		{"invalid date", "jobwipreason001", "?as_of=2026-13-01", token, http.StatusBadRequest},
		{"future date", "jobwipreason001", "?as_of=2999-01-01", token, http.StatusBadRequest},
	} {
		(&tests.ApiScenario{Name: tc.name, Method: http.MethodGet, URL: "/api/jobs/" + tc.job + "/wip/missing-rates" + tc.query, Headers: map[string]string{"Authorization": tc.token}, ExpectedStatus: tc.status, ExpectedContent: []string{"{"}, TestAppFactory: testutils.SetupTestApp}).Test(t)
	}
}
