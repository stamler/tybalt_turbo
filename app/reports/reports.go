package reports

import (
	_ "embed" // Needed for //go:embed
	"net/http"
	"time"
	"tybalt/constants"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed payroll_time.sql
var payrollTimeQuery string

// CreatePayrollTimeReportHandler returns a function that creates a payroll time report for a given week
func CreatePayrollTimeReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		week := e.Request.PathValue("week")

		// if week is not either 1 or 2, return an error
		if week != "1" && week != "2" {
			return e.Error(http.StatusBadRequest, "week must be either 1 or 2", nil)
		}

		payrollEndingDate, err := getPayrollEndingDate(e)
		if err != nil {
			return err
		}

		// if week is 1, subtract 7 days from the payroll ending date
		if week == "1" {
			payrollEndingDate = payrollEndingDate.AddDate(0, 0, -7)
		}

		// Execute the query
		var report []dbx.NullStringMap // TODO: make a type for this
		err = app.DB().NewQuery(payrollTimeQuery).Bind(dbx.Params{
			"weekEnding": payrollEndingDate.Format("2006-01-02"),
		}).All(&report)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// convert the report to a csv string
		headers := []string{"payrollId", "weekEnding", "surname", "givenName", "name", "manager", "meals", "days off rotation", "hours worked", "salaryHoursOver44", "adjustedHoursWorked", "total overtime hours", "overtime hours to pay", "Bereavement", "Stat Holiday", "PPTO", "Sick", "Vacation", "overtime hours to bank", "Overtime Payout Requested", "hasAmendmentsForWeeksEnding", "salary"}
		csvString, err := convertToCSV(report, headers)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to generate CSV report: "+err.Error(), err)
		}

		// Set content type and return the CSV string
		e.Response.Header().Set("Content-Type", "text/csv")
		return e.String(http.StatusOK, csvString)
	}
}

// CreatePayrollExpenseReportHandler returns a function that creates a payroll expense report for a given payroll ending date
func CreatePayrollExpenseReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		payrollEndingDate, err := getPayrollEndingDate(e)
		if err != nil {
			return err
		}

		// TODO: Implement the logic to create the payroll expense report

		return e.JSON(http.StatusOK, map[string]any{"message": "Payroll expense report for " + payrollEndingDate.Format("2006-01-02")})
	}
}

// CreatePayrollReceiptsReportHandler returns a function that creates a payroll receipts zip archive for a given payroll ending date
func CreatePayrollReceiptsReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		payrollEndingDate, err := getPayrollEndingDate(e)
		if err != nil {
			return err
		}

		// TODO: Implement the logic to create the payroll receipts zip archive

		return e.JSON(http.StatusOK, map[string]any{"message": "Payroll receipts report for " + payrollEndingDate.Format("2006-01-02")})
	}
}

// getPayrollEndingDate returns the parsed and validated payroll ending date
// from the request path after ensuring it is a valid date and a Saturday
func getPayrollEndingDate(e *core.RequestEvent) (time.Time, error) {
	payrollEnding := e.Request.PathValue("payrollEnding")
	if payrollEnding == "" {
		return time.Time{}, e.Error(http.StatusBadRequest, "payrollEnding is required", nil)
	}

	// if payrollEnding is not a valid date, return an error
	payrollEndingDate, err := time.Parse("2006-01-02", payrollEnding)
	if err != nil {
		return time.Time{}, e.Error(http.StatusBadRequest, "payrollEnding must be a valid date", nil)
	}

	// if payrollEnding is not a Saturday, return an error
	if payrollEndingDate.Weekday() != time.Saturday {
		return time.Time{}, e.Error(http.StatusBadRequest, "payrollEnding must be a Saturday", nil)
	}

	// Check if payrollEndingDate is a multiple of 2 weeks (14 days) before or
	// after the PAYROLL_EPOCH.
	daysDifference := int(payrollEndingDate.Sub(constants.PAYROLL_EPOCH).Hours() / 24)
	if daysDifference%14 != 0 {
		return time.Time{}, e.Error(http.StatusBadRequest, "payrollEnding must be a multiple of 2 weeks before or after 2025-03-01", nil)
	}

	return payrollEndingDate, nil
}
