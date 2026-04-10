package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const (
	branchMismatchTimeEntryID = "tebrmismatch001"
	branchMatchTimeEntryID    = "tebranchokay001"
)

func TestTimeEntryBranchMismatchesReport(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "admin.only@example.com")
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
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "report holder can download time entry branch mismatches csv",
			Method:         http.MethodGet,
			URL:            "/api/reports/time_entry_branch_mismatches",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`time_entry_id,date,week_ending,hours,uid,employee_name,job_id,job_number,job_description,time_entry_branch_id,time_entry_branch_code,time_entry_branch_name,job_branch_id,job_branch_code,job_branch_name,description`,
				`tebrmismatch001,2024-09-02,2024-09-07,1,rzr98oadsp9qc11,Tester Time,jobbrmatch0001,98-8101,Branch mismatch fixture job,80875lm27v8wgi4,ThunderBay,Thunder Bay,xeq9q81q5307f70,Toronto,Toronto,mismatched branch`,
			},
			NotExpectedContent: []string{
				`tebranchokay001,`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
