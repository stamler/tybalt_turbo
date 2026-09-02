package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const employeeBranchHoursReportURL = "/api/kpi/reports/employee_branch_hours?start_date=2040-05-01&end_date=2040-05-31"

var expectedEmployeeBranchHoursColumns = []string{
	"payrollId",
	"surname",
	"givenName",
	"defaultBranch",
	"Collingwood",
	"CollingwoodNoJob",
	"Corporate",
	"CorporateNoJob",
	"Fort Frances",
	"Fort FrancesNoJob",
	"Kenora",
	"KenoraNoJob",
	"Kitchener-Waterloo",
	"Kitchener-WaterlooNoJob",
	"Ottawa",
	"OttawaNoJob",
	"Thunder Bay",
	"Thunder BayNoJob",
	"Toronto",
	"TorontoNoJob",
	"total",
}

type employeeBranchHoursAPIResponse struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

func TestEmployeeBranchHoursReportAuthorization(t *testing.T) {
	kpiToken := mustGenerateEmployeeBranchHoursToken(t, "time@test.com")
	reportToken := mustGenerateEmployeeBranchHoursToken(t, "fatt@mac.com")
	adminToken := mustGenerateEmployeeBranchHoursToken(t, "admin.only@example.com")

	scenarios := []tests.ApiScenario{
		{
			Name:           "employee branch hours report requires authentication",
			Method:         http.MethodGet,
			URL:            employeeBranchHoursReportURL,
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				"The request requires valid record authorization token.",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "report claim does not grant KPI report access",
			Method:         http.MethodGet,
			URL:            employeeBranchHoursReportURL,
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"You are not authorized to view KPI reports.",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "admin claim does not grant KPI report access",
			Method:         http.MethodGet,
			URL:            employeeBranchHoursReportURL,
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				"You are not authorized to view KPI reports.",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "KPI claim grants employee branch hours report access",
			Method:         http.MethodGet,
			URL:            employeeBranchHoursReportURL,
			Headers:        map[string]string{"Authorization": kpiToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"payrollId"`,
				`"Thunder BayNoJob"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestEmployeeBranchHoursReportData(t *testing.T) {
	kpiToken := mustGenerateEmployeeBranchHoursToken(t, "time@test.com")

	scenario := tests.ApiScenario{
		Name:   "employee branch hours report returns inclusive SQL totals and inactive employees",
		Method: http.MethodGet,
		URL:    employeeBranchHoursReportURL,
		Headers: map[string]string{
			"Authorization": kpiToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"columns"`,
			`"rows"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, response *http.Response) {
			defer response.Body.Close()

			var payload employeeBranchHoursAPIResponse
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				tb.Fatalf("failed to decode employee branch hours response: %v", err)
			}
			if !slices.Equal(payload.Columns, expectedEmployeeBranchHoursColumns) {
				tb.Fatalf("columns = %+v, want %+v", payload.Columns, expectedEmployeeBranchHoursColumns)
			}
			if len(payload.Rows) != 2 {
				tb.Fatalf("row count = %d, want 2: %+v", len(payload.Rows), payload.Rows)
			}

			rowsByPayrollID := employeeBranchHoursRowsByPayrollID(tb, payload.Rows)
			activeRow := requireEmployeeBranchHoursRow(tb, rowsByPayrollID, "900001")
			assertEmployeeBranchHoursText(tb, activeRow, "surname", "Time")
			assertEmployeeBranchHoursText(tb, activeRow, "givenName", "Tester")
			assertEmployeeBranchHoursText(tb, activeRow, "defaultBranch", "Thunder Bay")
			assertEmployeeBranchHoursNumber(tb, activeRow, "Thunder Bay", 4)
			assertEmployeeBranchHoursNumber(tb, activeRow, "Thunder BayNoJob", 1.5)
			assertEmployeeBranchHoursNumber(tb, activeRow, "Toronto", 2)
			assertEmployeeBranchHoursNumber(tb, activeRow, "TorontoNoJob", 0)
			assertEmployeeBranchHoursNumber(tb, activeRow, "total", 7.5)

			inactiveRow := requireEmployeeBranchHoursRow(tb, rowsByPayrollID, "FIX_INACTIVE")
			assertEmployeeBranchHoursText(tb, inactiveRow, "surname", "User")
			assertEmployeeBranchHoursText(tb, inactiveRow, "givenName", "Inactive")
			assertEmployeeBranchHoursText(tb, inactiveRow, "defaultBranch", "Thunder Bay")
			assertEmployeeBranchHoursNumber(tb, inactiveRow, "Thunder Bay", 0)
			assertEmployeeBranchHoursNumber(tb, inactiveRow, "Thunder BayNoJob", 3)
			assertEmployeeBranchHoursNumber(tb, inactiveRow, "Toronto", 2.5)
			assertEmployeeBranchHoursNumber(tb, inactiveRow, "total", 5.5)

			inactiveProfile, err := app.FindFirstRecordByFilter("admin_profiles", "payroll_id='FIX_INACTIVE'", nil)
			if err != nil {
				tb.Fatalf("failed to load inactive profile fixture: %v", err)
			}
			if inactiveProfile.GetBool("active") {
				tb.Fatal("inactive report fixture must remain inactive")
			}
			if _, exists := rowsByPayrollID["910000"]; exists {
				tb.Fatal("employee with no qualifying hours must not have a report row")
			}
		},
	}

	scenario.Test(t)
}

func TestEmployeeBranchHoursReportCSV(t *testing.T) {
	kpiToken := mustGenerateEmployeeBranchHoursToken(t, "time@test.com")

	scenario := tests.ApiScenario{
		Name:   "employee branch hours CSV uses the JSON column order and data",
		Method: http.MethodGet,
		URL:    employeeBranchHoursReportURL + "&format=csv",
		Headers: map[string]string{
			"Authorization": kpiToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"payrollId,surname,givenName,defaultBranch",
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, response *http.Response) {
			defer response.Body.Close()
			if got := response.Header.Get("Content-Disposition"); got != `attachment; filename="employee_branch_hours_2040-05-01_2040-05-31.csv"` {
				tb.Fatalf("Content-Disposition = %q", got)
			}

			records, err := csv.NewReader(response.Body).ReadAll()
			if err != nil {
				tb.Fatalf("failed to parse employee branch hours CSV: %v", err)
			}
			if len(records) != 3 {
				tb.Fatalf("CSV row count = %d, want 3", len(records))
			}
			if !slices.Equal(records[0], expectedEmployeeBranchHoursColumns) {
				tb.Fatalf("CSV columns = %+v, want %+v", records[0], expectedEmployeeBranchHoursColumns)
			}

			rowsByPayrollID := map[string]map[string]string{}
			for _, record := range records[1:] {
				if len(record) != len(records[0]) {
					tb.Fatalf("CSV row has %d columns, want %d", len(record), len(records[0]))
				}
				row := map[string]string{}
				for index, column := range records[0] {
					row[column] = record[index]
				}
				rowsByPayrollID[row["payrollId"]] = row
			}
			if got := rowsByPayrollID["900001"]["total"]; got != "7.5" {
				tb.Fatalf("active employee CSV total = %q, want 7.5", got)
			}
			if got := rowsByPayrollID["FIX_INACTIVE"]["total"]; got != "5.5" {
				tb.Fatalf("inactive employee CSV total = %q, want 5.5", got)
			}
		},
	}

	scenario.Test(t)
}

func TestEmployeeBranchHoursReportValidationAndEmptyRange(t *testing.T) {
	kpiToken := mustGenerateEmployeeBranchHoursToken(t, "time@test.com")
	scenarios := []tests.ApiScenario{
		{
			Name:            "employee branch hours report requires start date",
			Method:          http.MethodGet,
			URL:             "/api/kpi/reports/employee_branch_hours?end_date=2040-05-31",
			Headers:         map[string]string{"Authorization": kpiToken},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"Start_date is required"},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:            "employee branch hours report rejects invalid dates",
			Method:          http.MethodGet,
			URL:             "/api/kpi/reports/employee_branch_hours?start_date=2040-02-30&end_date=2040-05-31",
			Headers:         map[string]string{"Authorization": kpiToken},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"Start_date must be a valid date in YYYY-MM-DD format"},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:            "employee branch hours report rejects reversed range",
			Method:          http.MethodGet,
			URL:             "/api/kpi/reports/employee_branch_hours?start_date=2040-06-01&end_date=2040-05-31",
			Headers:         map[string]string{"Authorization": kpiToken},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"Start_date must be on or before end_date"},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:            "employee branch hours report rejects unknown format",
			Method:          http.MethodGet,
			URL:             employeeBranchHoursReportURL + "&format=pdf",
			Headers:         map[string]string{"Authorization": kpiToken},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{"Format must be json or csv"},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:           "employee branch hours report returns columns and no rows for an empty range",
			Method:         http.MethodGet,
			URL:            "/api/kpi/reports/employee_branch_hours?start_date=1900-01-01&end_date=1900-01-31",
			Headers:        map[string]string{"Authorization": kpiToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"columns"`,
				`"rows":[]`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestEmployeeBranchHoursReportIndex(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	claim, err := app.FindRecordById("claims", "kpihoursclaim01")
	if err != nil {
		t.Fatalf("failed to load kpi claim: %v", err)
	}
	if got := claim.GetString("name"); got != "kpi" {
		t.Fatalf("claim name = %q, want kpi", got)
	}

	var result struct {
		Count int `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM sqlite_master
		WHERE type = 'index'
		  AND name = 'idx_time_entries_time_type_date'
	`).One(&result); err != nil {
		t.Fatalf("failed to inspect employee branch hours index: %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("employee branch hours index count = %d, want 1", result.Count)
	}

	var planRows []struct {
		Detail string `db:"detail"`
	}
	if err := app.DB().NewQuery(`
		EXPLAIN QUERY PLAN
		SELECT te.uid
		FROM time_entries te
		INNER JOIN time_types tt ON tt.id = te.time_type
		INNER JOIN branches b ON b.id = te.branch
		WHERE tt.code IN ('R', 'RT')
		  AND te.date >= '2040-05-01'
		  AND te.date <= '2040-05-31'
	`).All(&planRows); err != nil {
		t.Fatalf("failed to explain employee branch hours query: %v", err)
	}
	usesReportIndex := false
	for _, row := range planRows {
		if strings.Contains(row.Detail, "idx_time_entries_time_type_date") {
			usesReportIndex = true
			break
		}
	}
	if !usesReportIndex {
		t.Fatalf("employee branch hours query plan does not use its range index: %+v", planRows)
	}
}

func mustGenerateEmployeeBranchHoursToken(tb testing.TB, email string) string {
	tb.Helper()
	token, err := testutils.GenerateRecordToken("users", email)
	if err != nil {
		tb.Fatalf("failed to create token for %s: %v", email, err)
	}
	return token
}

func employeeBranchHoursRowsByPayrollID(tb testing.TB, rows []map[string]interface{}) map[string]map[string]interface{} {
	tb.Helper()
	result := map[string]map[string]interface{}{}
	for _, row := range rows {
		payrollID, ok := row["payrollId"].(string)
		if !ok || payrollID == "" {
			tb.Fatalf("report row has invalid payrollId: %+v", row)
		}
		result[payrollID] = row
	}
	return result
}

func requireEmployeeBranchHoursRow(tb testing.TB, rows map[string]map[string]interface{}, payrollID string) map[string]interface{} {
	tb.Helper()
	row, ok := rows[payrollID]
	if !ok {
		tb.Fatalf("missing report row for payrollId %s", payrollID)
	}
	return row
}

func assertEmployeeBranchHoursText(tb testing.TB, row map[string]interface{}, column string, want string) {
	tb.Helper()
	got, ok := row[column].(string)
	if !ok || got != want {
		tb.Fatalf("%s = %#v, want %q", column, row[column], want)
	}
}

func assertEmployeeBranchHoursNumber(tb testing.TB, row map[string]interface{}, column string, want float64) {
	tb.Helper()
	got, ok := row[column].(float64)
	if !ok || got != want {
		tb.Fatalf("%s = %#v, want %v", column, row[column], want)
	}
}
