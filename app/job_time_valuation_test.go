package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

type valuationExpectedRow struct {
	key       string
	hours     float64
	value     float64
	unpriced  float64
	estimated float64
	percent   *float64
}

func TestJobTimeValuation(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "valuation1@example.com")
	if err != nil {
		t.Fatal(err)
	}
	percent := func(v float64) *float64 { return &v }

	// These cases use dedicated CSV fixtures. The assigned sheet is inactive and
	// has a newer active revision with different rates. One employee has a
	// different default charge-out rate; the other has no admin profile.
	cases := []struct {
		name      string
		job       string
		start     string
		end       string
		total     float64
		unpriced  float64
		estimated float64
		staff     []valuationExpectedRow
		division  []valuationExpectedRow
	}{
		{
			name: "assigned revision takes priority over employee defaults",
			job:  "jobvaluation001", start: "2034-01-02", end: "2034-01-03", total: 975,
			staff: []valuationExpectedRow{
				{"uvaluation00001", 8, 900, 0, 0, percent(92.3)},
				{"uvaluation00002", 0.5, 75, 0, 0, percent(7.7)},
			},
			division: []valuationExpectedRow{
				{"CB", 6.5, 675, 0, 0, percent(69.2)},
				{"E", 2, 300, 0, 0, percent(30.8)},
			},
		},
		{
			name: "mixed job rates, employee defaults and unpriced hours",
			job:  "jobvaluation001", start: "2034-01-02", end: "2034-01-05", total: 3972, unpriced: 4, estimated: 3,
			staff: []valuationExpectedRow{
				{"uvaluation00001", 11, 3897, 0, 3, nil},
				{"uvaluation00002", 4.5, 75, 4, 0, nil},
			},
			division: []valuationExpectedRow{
				{"CB", 9.5, 3672, 0, 3, nil},
				{"E", 6, 300, 4, 0, nil},
			},
		},
		{
			name: "missing role uses default and complete estimates have percentages",
			job:  "jobvaluation001", start: "2034-01-02", end: "2034-01-04", total: 3972, estimated: 3,
			staff: []valuationExpectedRow{
				{"uvaluation00001", 11, 3897, 0, 3, percent(98.1)},
				{"uvaluation00002", 0.5, 75, 0, 0, percent(1.9)},
			},
			division: []valuationExpectedRow{
				{"CB", 9.5, 3672, 0, 3, percent(92.4)},
				{"E", 2, 300, 0, 0, percent(7.6)},
			},
		},
		{
			name: "incomplete status takes priority over estimated total",
			job:  "jobvaluation001", start: "2034-01-04", end: "2034-01-05", total: 2997, unpriced: 4, estimated: 3,
			staff: []valuationExpectedRow{
				{"uvaluation00001", 3, 2997, 0, 3, nil},
				{"uvaluation00002", 4, 0, 4, 0, nil},
			},
			division: []valuationExpectedRow{{"CB", 3, 2997, 0, 3, nil}, {"E", 4, 0, 4, 0, nil}},
		},
		{
			name: "missing rate and admin profile leaves all hours unpriced",
			job:  "jobvaluation001", start: "2034-01-05", end: "2034-01-05", unpriced: 4,
			staff:    []valuationExpectedRow{{"uvaluation00002", 4, 0, 4, 0, nil}},
			division: []valuationExpectedRow{{"E", 4, 0, 4, 0, nil}},
		},
		{
			name: "job without rate sheet uses employee default",
			job:  "jobvaluation002", start: "2034-01-02", end: "2034-01-02", total: 4995, estimated: 5,
			staff:    []valuationExpectedRow{{"uvaluation00001", 5, 4995, 0, 5, percent(100)}},
			division: []valuationExpectedRow{{"CB", 5, 4995, 0, 5, percent(100)}},
		},
		{
			name: "job without rate sheet or employee default stays unpriced",
			job:  "jobvaluation002", start: "2034-01-03", end: "2034-01-03", unpriced: 2,
			staff:    []valuationExpectedRow{{"uvaluation00002", 2, 0, 2, 0, nil}},
			division: []valuationExpectedRow{{"E", 2, 0, 2, 0, nil}},
		},
		{
			name: "other job uses its own revision", job: "jobvaluation003", start: "2034-01-02", end: "2034-01-02", total: 800,
			staff:    []valuationExpectedRow{{"uvaluation00001", 1, 800, 0, 0, percent(100)}},
			division: []valuationExpectedRow{{"CB", 1, 800, 0, 0, percent(100)}},
		},
		{
			name: "zero hours have no percentage or estimate", job: "jobvaluation001", start: "2034-01-07", end: "2034-01-07",
			staff:    []valuationExpectedRow{{"uvaluation00001", 0, 0, 0, 0, nil}},
			division: []valuationExpectedRow{{"CB", 0, 0, 0, 0, nil}},
		},
		{
			name: "missing role rate uses default for fractional hours",
			job:  "jobvaluation001", start: "2034-01-08", end: "2034-01-08", total: 1498.5, estimated: 1.5,
			staff:    []valuationExpectedRow{{"uvaluation00001", 1.5, 1498.5, 0, 1.5, percent(100)}},
			division: []valuationExpectedRow{{"E", 1.5, 1498.5, 0, 1.5, percent(100)}},
		},
		{
			name: "zero employee default is not a usable fallback",
			job:  "jobvaluation001", start: "2034-01-09", end: "2034-01-09", unpriced: 2,
			staff:    []valuationExpectedRow{{"uvaluation00003", 2, 0, 2, 0, nil}},
			division: []valuationExpectedRow{{"E", 2, 0, 2, 0, nil}},
		},
		{
			name: "zero employee default does not prevent use of job rate",
			job:  "jobvaluation001", start: "2034-01-10", end: "2034-01-10", total: 200,
			staff:    []valuationExpectedRow{{"uvaluation00003", 2, 200, 0, 0, percent(100)}},
			division: []valuationExpectedRow{{"E", 2, 200, 0, 0, percent(100)}},
		},
		{
			name: "empty range returns an empty array", job: "jobvaluation001", start: "2034-02-01", end: "2034-02-02",
		},
	}

	for _, endpoint := range []string{"staff", "divisions"} {
		for _, tc := range cases {
			expected := tc.staff
			key := "uid"
			if endpoint == "divisions" {
				expected = tc.division
				key = "division_code"
			}
			scenario := tests.ApiScenario{
				Name: endpoint + "/" + tc.name, Method: http.MethodGet,
				URL:     fmt.Sprintf("/api/jobs/%s/%s/summary?start_date=%s&end_date=%s", tc.job, endpoint, tc.start, tc.end),
				Headers: map[string]string{"Authorization": token}, ExpectedStatus: http.StatusOK,
				ExpectedContent: []string{"["},
				TestAppFactory:  testutils.SetupTestApp,
				AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
					defer res.Body.Close()
					var rows []map[string]any
					if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
						tb.Fatal(err)
					}
					if rows == nil || len(rows) != len(expected) {
						tb.Fatalf("expected %d rows in a JSON array, got %#v", len(expected), rows)
					}
					for i, want := range expected {
						row := rows[i]
						sheetName := "Summary valuation fixture"
						var revision any = float64(0)
						if tc.job == "jobvaluation002" {
							sheetName, revision = "", nil
						} else if tc.job == "jobvaluation003" {
							revision = float64(1)
						}
						for field, value := range map[string]any{
							key: want.key, "hours": want.hours, "value": want.value,
							"unpriced_hours": want.unpriced, "total": tc.total, "total_unpriced_hours": tc.unpriced,
							"estimated_hours": want.estimated, "total_estimated_hours": tc.estimated,
							"rate_sheet_name": sheetName, "rate_sheet_revision": revision,
						} {
							if row[field] != value {
								tb.Errorf("row %d: %s = %v, want %v", i, field, row[field], value)
							}
						}
						valueStatus, totalStatus := "Rate sheet", "Rate sheet"
						if want.estimated > 0 {
							valueStatus = "Estimated"
						}
						if tc.estimated > 0 {
							totalStatus = "Estimated"
						}
						if want.unpriced > 0 {
							valueStatus = "Incomplete"
						}
						if tc.unpriced > 0 {
							totalStatus = "Incomplete"
						}
						if row["value_status"] != valueStatus || row["total_status"] != totalStatus {
							tb.Errorf("row %d: incorrect pricing status: %#v", i, row)
						}
						value, exists := row["percent"]
						if !exists || (want.percent == nil && value != nil) || (want.percent != nil && value != *want.percent) {
							tb.Errorf("row %d: unexpected percent: %v", i, value)
						}
					}
				},
			}
			scenario.Test(t)
		}
		for _, tc := range []struct {
			name   string
			query  string
			token  string
			status int
		}{
			{"authentication required", "?start_date=2034-01-02&end_date=2034-01-03", "", http.StatusUnauthorized},
			{"start date required", "?end_date=2034-01-03", token, http.StatusBadRequest},
			{"end date required", "?start_date=2034-01-02", token, http.StatusBadRequest},
		} {
			scenario := tests.ApiScenario{
				Name: endpoint + "/" + tc.name, Method: http.MethodGet,
				URL:     "/api/jobs/jobvaluation001/" + endpoint + "/summary" + tc.query,
				Headers: map[string]string{"Authorization": tc.token}, ExpectedStatus: tc.status,
				ExpectedContent: []string{`"message"`},
				TestAppFactory:  testutils.SetupTestApp,
			}
			scenario.Test(t)
		}
	}
}
