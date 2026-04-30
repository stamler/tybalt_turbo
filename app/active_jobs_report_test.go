package main

import (
	"encoding/csv"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestActiveJobsReport(t *testing.T) {
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
			Name:           "admin-only user cannot download active jobs csv",
			Method:         http.MethodGet,
			URL:            "/api/reports/active_jobs",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "report holder can download active jobs csv",
			Method:         http.MethodGet,
			URL:            "/api/reports/active_jobs",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`status,job_number,manager_name,branch_name,time_and_materials`,
				`Active,97-9902,Horace Silver,Toronto,true`,
				`Active,97-9901,Fakesy Manjor,Kitchener-Waterloo,false`,
			},
			NotExpectedContent: []string{
				`P97-9903`,
				`97-9900`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				records, err := csv.NewReader(res.Body).ReadAll()
				if err != nil {
					tb.Fatalf("failed to read active jobs csv: %v", err)
				}
				if len(records) < 3 {
					tb.Fatalf("expected at least header and two active job rows, got %d", len(records))
				}

				higherIndex := activeJobsReportRowIndex(records, "97-9902")
				lowerIndex := activeJobsReportRowIndex(records, "97-9901")
				if higherIndex == -1 {
					tb.Fatal("expected 97-9902 in active jobs csv")
				}
				if lowerIndex == -1 {
					tb.Fatal("expected 97-9901 in active jobs csv")
				}
				if higherIndex >= lowerIndex {
					tb.Fatalf("expected 97-9902 to sort before 97-9901, got indexes %d and %d", higherIndex, lowerIndex)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func activeJobsReportRowIndex(records [][]string, jobNumber string) int {
	for i, record := range records {
		if len(record) > 1 && record[1] == jobNumber {
			return i
		}
	}
	return -1
}
