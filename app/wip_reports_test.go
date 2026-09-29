package main

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/routes"

	"github.com/pocketbase/pocketbase/tests"
)

func TestWIPReportAccess(t *testing.T) {
	for _, tc := range []struct {
		name, email, path string
		status            int
	}{
		{"my requires authentication", "", "/my", 401},
		{"branch requires authentication", "", "/branch", 401},
		{"division requires authentication", "", "/division?division=0vqgq5fktoen3rr", 401},
		{"ordinary user can view own report", "valuation1@example.com", "/my", 200},
		{"ordinary user cannot view branches", "valuation1@example.com", "/branch", 403},
		{"ordinary user cannot view divisions", "valuation1@example.com", "/division?division=0vqgq5fktoen3rr", 403},
		{"report claim alone is insufficient", "fatt@mac.com", "/branch", 403},
		{"branch manager can view another branch", "u_no_claims@example.com", "/branch?branch=80875lm27v8wgi4", 200},
		{"branch manager can view divisions", "u_no_claims@example.com", "/division?division=0vqgq5fktoen3rr", 200},
		{"kpi can view branches", "time@test.com", "/branch", 200},
		{"kpi can view divisions", "time@test.com", "/division?division=0vqgq5fktoen3rr", 200},
		{"admin can view branches", "admin.only@example.com", "/branch", 200},
		{"admin can view divisions", "admin.only@example.com", "/division?division=0vqgq5fktoen3rr", 200},
		{"division is required", "time@test.com", "/division", 400},
		{"invalid division", "time@test.com", "/division?division=missing", 400},
		{"invalid branch", "time@test.com", "/branch?branch=missing", 400},
		{"invalid filter", "time@test.com", "/division?division=0vqgq5fktoen3rr&require_time=maybe", 400},
	} {
		token := ""
		if tc.email != "" {
			var err error
			token, err = testutils.GenerateRecordToken("users", tc.email)
			if err != nil {
				t.Fatal(err)
			}
		}
		(&tests.ApiScenario{Name: tc.name, Method: http.MethodGet, URL: "/api/wip" + tc.path,
			Headers: map[string]string{"Authorization": token}, ExpectedStatus: tc.status,
			ExpectedContent: []string{"{"}, TestAppFactory: testutils.SetupTestApp,
		}).Test(t)
	}
}

func TestWIPReportSelectionAndValues(t *testing.T) {
	// These append-only CSV fixtures include parent and child jobs, future time,
	// meal-only entries, allocations without time, and time without allocations.
	for _, tc := range []struct {
		name, email, path string
		ids               []string
		excluded          int
	}{
		{"personal scope ignores caller supplied manager", "valuation1@example.com", "/my?uid=uvaluation00003&manager=uvaluation00003", []string{"000001", "000002", "000003", "000005", "000006", "000007", "000008", "000009"}, 2},
		{"alternate manager is not job manager", "valuation3@example.com", "/my", []string{"000010"}, 0},
		{"no qualifying projects is empty", "valuation2@example.com", "/my", []string{}, 0},
		{"branch selection uses job branch", "u_no_claims@example.com", "/branch?branch=2b65d8161y8hx95", []string{"000002"}, -1},
		{"division defaults to recorded work hours", "time@test.com", "/division?division=0vqgq5fktoen3rr", []string{"000001", "000002", "000008", "000010"}, 2},
		{"division can include allocations without work hours", "time@test.com", "/division?division=0vqgq5fktoen3rr&require_time=false", []string{"000001", "000002", "000003", "000005", "000006", "000007", "000008", "000010"}, 2},
	} {
		token, err := testutils.GenerateRecordToken("users", tc.email)
		if err != nil {
			t.Fatal(err)
		}
		(&tests.ApiScenario{Name: tc.name, Method: http.MethodGet, URL: "/api/wip" + tc.path,
			Headers: map[string]string{"Authorization": token}, ExpectedStatus: 200,
			ExpectedContent: []string{"{"}, TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()
				var got routes.WIPReport
				if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
					tb.Fatal(err)
				}
				if got.Items == nil {
					tb.Fatal("items must be an array")
				}
				ids := []string{}
				for _, row := range got.Items {
					if row.ProjectValue <= 0 {
						tb.Errorf("nonpositive project value: %+v", row)
					}
					if !strings.HasPrefix(row.ID, "jobwiprpt") {
						tb.Errorf("unexpected job: %s", row.ID)
					}
					ids = append(ids, strings.TrimPrefix(row.ID, "jobwiprpt"))
					if row.AsOf != got.AsOf {
						tb.Error("inconsistent report date")
					}
					switch row.ID {
					case "jobwiprpt000001":
						// Whole-job totals include other divisions, even in the division report.
						want := routes.JobWIP{ProjectValue: 10000, AsOf: got.AsOf, TimeValue: 849.5, Hours: 3.5, EstimatedHours: .5, MissingRoleHours: .5, ExpenseValue: 800, POValue: 3565, EstimatedPOs: 1}
						if row.JobWIP != want {
							tb.Errorf("whole-job values = %+v; want %+v", row.JobWIP, want)
						}
					case "jobwiprpt000002":
						if !row.NoRateSheet || row.TimeValue != 1998 || row.UnpricedHours != 1 || row.EstimatedHours != 2 {
							tb.Errorf("missing-rate values = %+v", row.JobWIP)
						}
					case "jobwiprpt000003":
						if row.UnpricedExpenses != 1 || row.UnpricedPOs != 2 {
							tb.Errorf("missing monetary values = %+v", row.JobWIP)
						}
					case "jobwiprpt000006", "jobwiprpt000007":
						if row.TimeValue != 0 || row.Hours != 0 {
							tb.Errorf("future or meal-only time included: %+v", row.JobWIP)
						}
					case "jobwiprpt000008":
						if row.TimeValue != 100 || row.Hours != 1 {
							tb.Errorf("child job must remain separate: %+v", row.JobWIP)
						}
					}
				}
				if !reflect.DeepEqual(ids, tc.ids) {
					tb.Errorf("jobs = %v; want %v", ids, tc.ids)
				}
				if tc.excluded >= 0 && got.ExcludedProjectValue != tc.excluded {
					tb.Errorf("excluded = %d; want %d", got.ExcludedProjectValue, tc.excluded)
				}
			},
		}).Test(t)
	}
}
