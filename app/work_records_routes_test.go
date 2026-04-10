package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

type workRecordSearchRow struct {
	WorkRecord string `json:"work_record"`
	Prefix     string `json:"prefix"`
	EntryCount int    `json:"entry_count"`
	SearchText string `json:"search_text"`
}

type workRecordDetailRow struct {
	ID          string  `json:"id"`
	WorkRecord  string  `json:"work_record"`
	WeekEnding  string  `json:"week_ending"`
	UID         string  `json:"uid"`
	Hours       float64 `json:"hours"`
	JobNumber   string  `json:"job_number"`
	JobID       string  `json:"job_id"`
	Description string  `json:"description"`
	Surname     string  `json:"surname"`
	GivenName   string  `json:"given_name"`
	TimesheetID string  `json:"timesheet_id"`
}

func TestWorkRecordsRoutes(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}
	workRecordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "report holder can fetch work records list",
			Method: http.MethodGet,
			URL:    "/api/work_records",
			Headers: map[string]string{
				"Authorization": reportToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"work_record":"F34-142"`,
				`"work_record":"K12-314"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var rows []workRecordSearchRow
				if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
					t.Fatalf("failed decoding work records list response: %v", err)
				}

				if len(rows) != 2 {
					t.Fatalf("expected 2 work records, got %d: %#v", len(rows), rows)
				}
				if rows[0].WorkRecord != "F34-142" || rows[0].EntryCount != 1 || rows[0].Prefix != "F" {
					t.Fatalf("unexpected first row: %#v", rows[0])
				}
				if rows[1].WorkRecord != "K12-314" || rows[1].EntryCount != 2 || rows[1].Prefix != "K" {
					t.Fatalf("unexpected second row: %#v", rows[1])
				}
				if rows[1].SearchText == "" {
					t.Fatalf("expected populated search_text for %#v", rows[1])
				}
			},
		},
		{
			Name:   "work_record holder can fetch work records list",
			Method: http.MethodGet,
			URL:    "/api/work_records",
			Headers: map[string]string{
				"Authorization": workRecordToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"work_record":"K12-314"`,
				`"entry_count":2`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user cannot fetch work records list",
			Method: http.MethodGet,
			URL:    "/api/work_records",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"you are not authorized to view work records"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "report holder can fetch work record details with hidden link fields",
			Method: http.MethodGet,
			URL:    "/api/work_records/K12-314",
			Headers: map[string]string{
				"Authorization": reportToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"work_record":"K12-314"`,
				`"timesheet_id":"j1lr2oddjongtoj"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
				defer res.Body.Close()

				var rows []workRecordDetailRow
				if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
					t.Fatalf("failed decoding work record details response: %v", err)
				}

				if len(rows) != 2 {
					t.Fatalf("expected 2 detail rows, got %d: %#v", len(rows), rows)
				}
				if rows[0].WorkRecord != "K12-314" || rows[1].WorkRecord != "K12-314" {
					t.Fatalf("expected only K12-314 rows, got %#v", rows)
				}
				if rows[0].WeekEnding != "2024-09-28" || rows[1].WeekEnding != "2024-06-22" {
					t.Fatalf("expected default detail ordering by week ending desc, got %#v", rows)
				}
				if rows[0].JobID == "" || rows[0].UID == "" || rows[0].TimesheetID == "" {
					t.Fatalf("expected hidden link fields to be populated, got %#v", rows[0])
				}
			},
		},
		{
			Name:   "work_record holder can fetch a different work record detail",
			Method: http.MethodGet,
			URL:    "/api/work_records/F34-142",
			Headers: map[string]string{
				"Authorization": workRecordToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"work_record":"F34-142"`,
				`"job_id":"zke3cs3yipplwtu"`,
				`"timesheet_id":"av32qwch9xrcb5n"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user cannot fetch work record details",
			Method: http.MethodGet,
			URL:    "/api/work_records/K12-314",
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"you are not authorized to view work records"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
