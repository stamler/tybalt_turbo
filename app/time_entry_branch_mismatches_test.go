package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	branchMismatchTimeEntryID = "tebrmismatch001"
	branchMatchTimeEntryID    = "tebranchokay001"
)

func seedTimeEntryBranchMismatch(tb testing.TB, app *tests.TestApp) {
	tb.Helper()

	collection, err := app.FindCollectionByNameOrId("time_entries")
	if err != nil {
		tb.Fatalf("failed to load time_entries collection: %v", err)
	}

	weekEnding, err := utilities.GenerateWeekEnding("2024-09-02")
	if err != nil {
		tb.Fatalf("failed to generate week ending: %v", err)
	}

	for _, seed := range []struct {
		id       string
		branchID string
		jobID    string
		desc     string
	}{
		{
			id:       branchMismatchTimeEntryID,
			branchID: "80875lm27v8wgi4",
			jobID:    "test_job_w_rs",
			desc:     "mismatched branch",
		},
		{
			id:       branchMatchTimeEntryID,
			branchID: "xeq9q81q5307f70",
			jobID:    "test_job_w_rs",
			desc:     "matching branch",
		},
	} {
		record := core.NewRecord(collection)
		record.Set("id", seed.id)
		record.Set("uid", "rzr98oadsp9qc11")
		record.Set("branch", seed.branchID)
		record.Set("date", "2024-09-02")
		record.Set("week_ending", weekEnding)
		record.Set("time_type", "sdyfl3q7j7ap849")
		record.Set("division", "fy4i9poneukvq9u")
		record.Set("description", seed.desc)
		record.Set("hours", 1)
		record.Set("job", seed.jobID)
		record.Set("role", "tbgoiwwwfj8cvju")
		if err := app.Save(record); err != nil {
			tb.Fatalf("failed to save seeded time entry %s: %v", seed.id, err)
		}
	}
}

func TestTimeEntryBranchMismatchesReport(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "admin-only user cannot download report-claim mismatch csv",
			Method:         http.MethodGet,
			URL:            "/api/reports/time_entry_branch_mismatches",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: setupAdminOnlyViewerApp,
		},
		{
			Name:           "report holder can download time entry branch mismatches csv",
			Method:         http.MethodGet,
			URL:            "/api/reports/time_entry_branch_mismatches",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`time_entry_id,date,week_ending,uid,employee_name,job_id,job_number,job_description,time_entry_branch_id,time_entry_branch_code,time_entry_branch_name,job_branch_id,job_branch_code,job_branch_name,description`,
				`tebrmismatch001,2024-09-02,2024-09-07,rzr98oadsp9qc11,Tester Time,test_job_w_rs,98-8001,Test Job With Rate Sheet,80875lm27v8wgi4,ThunderBay,Thunder Bay,xeq9q81q5307f70,Toronto,Toronto,mismatched branch`,
			},
			NotExpectedContent: []string{
				`tebranchokay001,`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(tb)
				setJobBranch(tb, app, "test_job_w_rs", "xeq9q81q5307f70")
				seedTimeEntryBranchMismatch(tb, app)
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
