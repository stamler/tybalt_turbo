package reports

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed employee_branch_hours.sql
var employeeBranchHoursQueryTemplate string

const employeeBranchHoursColumnsToken = "/* employee_branch_hours_columns */"

var employeeBranchHoursBaseColumns = []string{
	"payrollId",
	"surname",
	"givenName",
	"defaultBranch",
}

type employeeBranchHoursBranch struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

type EmployeeBranchHoursReport struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
}

// CreateEmployeeBranchHoursReportHandler returns an employee branch-hours KPI
// report for an inclusive date range. The route group is responsible for
// requiring the kpi claim.
func CreateEmployeeBranchHoursReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		queryValues := e.Request.URL.Query()
		startDate, err := parseEmployeeBranchHoursDate(e, queryValues.Get("start_date"), "start_date")
		if err != nil {
			return err
		}
		endDate, err := parseEmployeeBranchHoursDate(e, queryValues.Get("end_date"), "end_date")
		if err != nil {
			return err
		}
		if startDate.After(endDate) {
			return e.Error(http.StatusBadRequest, "start_date must be on or before end_date", nil)
		}

		format := strings.ToLower(strings.TrimSpace(queryValues.Get("format")))
		if format == "" {
			format = "json"
		}
		if format != "json" && format != "csv" {
			return e.Error(http.StatusBadRequest, "format must be json or csv", nil)
		}

		report, err := getEmployeeBranchHoursReport(
			app,
			startDate.Format(time.DateOnly),
			endDate.Format(time.DateOnly),
		)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to create employee branch hours report", err)
		}

		if format == "json" {
			return e.JSON(http.StatusOK, report)
		}

		csvText, err := employeeBranchHoursCSV(report)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to create employee branch hours CSV", err)
		}
		filename := fmt.Sprintf(
			"employee_branch_hours_%s_%s.csv",
			startDate.Format(time.DateOnly),
			endDate.Format(time.DateOnly),
		)
		e.Response.Header().Set("Content-Type", "text/csv; charset=utf-8")
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		return e.String(http.StatusOK, csvText)
	}
}

func parseEmployeeBranchHoursDate(e *core.RequestEvent, value string, name string) (time.Time, error) {
	if value == "" {
		return time.Time{}, e.Error(http.StatusBadRequest, name+" is required", nil)
	}
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Format(time.DateOnly) != value {
		return time.Time{}, e.Error(http.StatusBadRequest, name+" must be a valid date in YYYY-MM-DD format", err)
	}
	return parsed, nil
}

func getEmployeeBranchHoursReport(app core.App, startDate string, endDate string) (EmployeeBranchHoursReport, error) {
	branches := []employeeBranchHoursBranch{}
	if err := app.DB().NewQuery(`
		SELECT id, name
		FROM branches
		WHERE TRIM(name) != ''
		ORDER BY name, id
	`).All(&branches); err != nil {
		return EmployeeBranchHoursReport{}, err
	}

	columns := append([]string{}, employeeBranchHoursBaseColumns...)
	usedColumns := map[string]struct{}{}
	for _, column := range employeeBranchHoursBaseColumns {
		usedColumns[strings.ToLower(column)] = struct{}{}
	}
	usedColumns["total"] = struct{}{}

	params := dbx.Params{
		"start_date": startDate,
		"end_date":   endDate,
	}
	branchExpressions := make([]string, 0, len(branches)*2)
	for index, branch := range branches {
		jobColumn := branch.Name
		noJobColumn := branch.Name + "NoJob"
		for _, column := range []string{jobColumn, noJobColumn} {
			key := strings.ToLower(column)
			if _, exists := usedColumns[key]; exists {
				return EmployeeBranchHoursReport{}, fmt.Errorf("branch name creates duplicate report column %q", column)
			}
			usedColumns[key] = struct{}{}
			columns = append(columns, column)
		}

		parameterName := fmt.Sprintf("branch_%d", index)
		params[parameterName] = branch.ID
		branchExpressions = append(branchExpressions,
			fmt.Sprintf(
				"COALESCE(SUM(CASE WHEN qt.branch = {:%s} AND qt.job != '' THEN qt.hours ELSE 0 END), 0) AS %s",
				parameterName,
				quoteSQLiteIdentifier(jobColumn),
			),
			fmt.Sprintf(
				"COALESCE(SUM(CASE WHEN qt.branch = {:%s} AND qt.job = '' THEN qt.hours ELSE 0 END), 0) AS %s",
				parameterName,
				quoteSQLiteIdentifier(noJobColumn),
			),
		)
	}
	columns = append(columns, "total")

	projection := ""
	if len(branchExpressions) > 0 {
		projection = strings.Join(branchExpressions, ",\n  ") + ","
	}
	query := strings.Replace(
		employeeBranchHoursQueryTemplate,
		employeeBranchHoursColumnsToken,
		projection,
		1,
	)

	queryRows, err := app.DB().NewQuery(query).Bind(params).Rows()
	if err != nil {
		return EmployeeBranchHoursReport{}, err
	}
	defer queryRows.Close()

	queryColumns, err := queryRows.Columns()
	if err != nil {
		return EmployeeBranchHoursReport{}, err
	}
	if len(queryColumns) != len(columns) {
		return EmployeeBranchHoursReport{}, fmt.Errorf(
			"report query returned %d columns; expected %d",
			len(queryColumns),
			len(columns),
		)
	}

	report := EmployeeBranchHoursReport{
		Columns: columns,
		Rows:    []map[string]any{},
	}
	for queryRows.Next() {
		values := make([]any, len(queryColumns))
		destinations := make([]any, len(queryColumns))
		for index := range values {
			destinations[index] = &values[index]
		}
		if err := queryRows.Scan(destinations...); err != nil {
			return EmployeeBranchHoursReport{}, err
		}

		row := make(map[string]any, len(queryColumns))
		for index, column := range queryColumns {
			if index < len(employeeBranchHoursBaseColumns) {
				row[column] = employeeBranchHoursText(values[index])
				continue
			}
			number, err := employeeBranchHoursNumber(values[index])
			if err != nil {
				return EmployeeBranchHoursReport{}, fmt.Errorf("invalid numeric value for %s: %w", column, err)
			}
			row[column] = number
		}
		report.Rows = append(report.Rows, row)
	}
	if err := queryRows.Err(); err != nil {
		return EmployeeBranchHoursReport{}, err
	}

	return report, nil
}

func quoteSQLiteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func employeeBranchHoursText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func employeeBranchHoursNumber(value any) (float64, error) {
	switch typed := value.(type) {
	case nil:
		return 0, nil
	case float64:
		return typed, nil
	case int64:
		return float64(typed), nil
	case string:
		return strconv.ParseFloat(typed, 64)
	case []byte:
		return strconv.ParseFloat(string(typed), 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func employeeBranchHoursCSV(report EmployeeBranchHoursReport) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	if err := writer.Write(report.Columns); err != nil {
		return "", err
	}

	record := make([]string, len(report.Columns))
	for _, row := range report.Rows {
		for index, column := range report.Columns {
			switch value := row[column].(type) {
			case float64:
				record[index] = strconv.FormatFloat(value, 'f', -1, 64)
			case nil:
				record[index] = ""
			default:
				record[index] = fmt.Sprint(value)
			}
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return builder.String(), nil
}
