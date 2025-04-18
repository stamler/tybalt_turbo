package reports

import (
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

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

		// TODO: Implement the logic to create the payroll time report

		return e.JSON(http.StatusOK, map[string]any{"message": "Payroll time report for " + payrollEndingDate.Format("2006-01-02") + " week " + week})
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

	return payrollEndingDate, nil
}
