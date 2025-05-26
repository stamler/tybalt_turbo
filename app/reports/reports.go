package reports

import (
	_ "embed" // Needed for //go:embed
	"net/http"
	"strings"
	"time"
	"tybalt/constants"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var expenseCollectionId = "o1vpz1mm7qsfoyy"

// Attachment represents a file attachment to an record.
// It is used to store the filename, source path, and SHA-256 hash of the attachment.
type Attachment struct {
	Id          string `db:"id"`
	Filename    string `db:"filename"`
	ZipFilename string `db:"zip_filename"`
	SourcePath  string `db:"source_path"`
	Sha256      string `db:"sha256"`
}

//go:embed payroll_time.sql
var payrollTimeQuery string

//go:embed expenses.sql
var expensesQueryTemplate string

//go:embed receipts.sql
var receiptsQueryTemplate string

// CreatePayrollTimeReportHandler returns a function that creates a payroll time report for a given week
func CreatePayrollTimeReportHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		week := e.Request.PathValue("week")

		// if week is not either 1 or 2, return an error
		if week != "1" && week != "2" {
			return e.Error(http.StatusBadRequest, "week must be either 1 or 2", nil)
		}

		dateColumnValue, err := getDateColumnValue(e, true)
		if err != nil {
			return err
		}

		// if week is 1, subtract 7 days from the payroll ending date
		if week == "1" {
			dateColumnValue = dateColumnValue.AddDate(0, 0, -7)
		}

		// Execute the query
		var report []dbx.NullStringMap // TODO: make a type for this
		err = app.DB().NewQuery(payrollTimeQuery).Bind(dbx.Params{
			"weekEnding": dateColumnValue.Format("2006-01-02"),
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

// CreateExpenseReportHandler returns a function that creates an expense report for a given value of date_column (provided in the request path)
func CreateExpenseReportHandler(app core.App, dateColumnName string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		dateColumnValue, err := getDateColumnValue(e, dateColumnName == "pay_period_ending")
		if err != nil {
			return err
		}

		// Replace the placeholder in the query string with the column name. We do
		// this instead of using Bind() because the column name will be incorrectly
		// quoted by Bind() for SQLite (it will be enclosed in single quotes which
		// SQL will interpret as a string literal rather than as an identifier).
		expensesQuery := strings.ReplaceAll(expensesQueryTemplate, "{:date_column}", dateColumnName)

		// Execute the query
		var report []dbx.NullStringMap // TODO: make a type for this
		err = app.DB().NewQuery(expensesQuery).Bind(dbx.Params{
			"date_column_value": dateColumnValue.Format("2006-01-02"),
		}).All(&report)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// convert the report to a csv string
		headers := []string{"payrollId", "Acct/Visa/Exp", "Job #", "Client", "Job Description", "Div", "Date", "Month", "Year", "calculatedSubtotal", "calculatedOntarioHST", "Total", "PO#", "Description", "Company", "Employee", "Approved By"}
		csvString, err := convertToCSV(report, headers)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to generate CSV report: "+err.Error(), err)
		}

		// Set content type and return the CSV string
		e.Response.Header().Set("Content-Type", "text/csv")
		return e.String(http.StatusOK, csvString)
	}
}

// CreateReceiptsReportHandler returns a function that creates a payroll receipts zip archive for a given value of date_column (provided in the request path)
func CreateReceiptsReportHandler(app core.App, dateColumnName string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		dateColumnValue, err := getDateColumnValue(e, dateColumnName == "pay_period_ending")
		if err != nil {
			return err
		}

		// Replace the placeholder in the query string with the column name. We do
		// this instead of using Bind() because the column name will be incorrectly
		// quoted by Bind() for SQLite (it will be enclosed in single quotes which
		// SQL will interpret as a string literal rather than as an identifier).
		receiptsQuery := strings.ReplaceAll(receiptsQueryTemplate, "{:date_column}", dateColumnName)

		// Execute the query
		var report []dbx.NullStringMap // TODO: make a type for this
		err = app.DB().NewQuery(receiptsQuery).Bind(dbx.Params{
			"date_column_value": dateColumnValue.Format("2006-01-02"),
		}).All(&report)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// build a list of receipts
		receipts := []Attachment{}
		for _, rowMap := range report {
			idVal, idOk := rowMap["id"]
			sourcePathVal, sourcePathOk := rowMap["source_path"]
			filenameVal, filenameOk := rowMap["filename"]
			zipFilenameVal, zipFilenameOk := rowMap["zip_filename"]
			sha256Val, sha256Ok := rowMap["sha256"]
			if !idOk || !sourcePathOk || !filenameOk || !zipFilenameOk || !sha256Ok {
				// skip rows that don't have all the required fields
				continue
			}
			receipts = append(receipts, Attachment{
				Id:          idVal.String,
				Filename:    filenameVal.String,
				ZipFilename: zipFilenameVal.String,
				SourcePath:  sourcePathVal.String,
				Sha256:      sha256Val.String,
			})
		}

		// Check the zip cache for a record that matches the dateColumnValue in the
		// specified class and receipts. If there's a cache hit, return the file
		// url. The class for this zip is "receipts_by_" + dateColumnName.
		zipCacheRecord, err := zipCacheLookup(app, dateColumnValue.Format("2006-01-02"), "receipts_by_"+dateColumnName, receipts)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to lookup zip cache: "+err.Error(), err)
		}
		if zipCacheRecord != nil {
			url := zipCacheRecord.BaseFilesPath() + "/" + zipCacheRecord.GetString("zip")
			app.Logger().Debug("zip_cache hit", dateColumnName, dateColumnValue.Format("2006-01-02"))
			return e.JSON(http.StatusOK, map[string]string{"url": url})
		}
		app.Logger().Debug("zip_cache miss", dateColumnName, dateColumnValue.Format("2006-01-02"))

		// If we get here, we have a cache miss. Create the zip and store it in the cache.

		// Define a struct to hold the result from the goroutine
		type zipResult struct {
			zipCacheRecord *core.Record
			err            error
		}

		// Create a channel to receive the result. A buffered channel of size 1 allows
		// the goroutine to send the result and exit without waiting for the receiver.
		resultChan := make(chan zipResult, 1)

		// Launch the goroutine to perform the zipping operation.
		// This allows the zipping (which can be I/O intensive) to happen concurrently.
		go func() {
			// The 'report' variable is captured from the outer function's scope.
			// The 'zipAttachments' function is defined in functions.go within the same package.
			zipCacheRecord, err := zipAttachments(app, receipts, expenseCollectionId, "receipts_by_"+dateColumnName, dateColumnValue.Format("2006-01-02"))
			resultChan <- zipResult{zipCacheRecord: zipCacheRecord, err: err}
		}()

		// Wait for the result from the goroutine.
		res := <-resultChan

		// Handle any error returned from the zipAttachments function.
		if res.err != nil {
			// Optionally, log the error on the server side here using app.Logger()
			// For example: app.Logger().Error("Failed to generate zip archive", "error", res.err)
			return e.Error(http.StatusInternalServerError, "failed to generate zip archive: "+res.err.Error(), res.err)
		}

		url := res.zipCacheRecord.BaseFilesPath() + "/" + res.zipCacheRecord.GetString("zip")
		return e.JSON(http.StatusOK, map[string]string{"url": url})
	}
}

// getDateColumnValue returns the parsed and validated date column value
// from the request path after ensuring it is a valid date and a Saturday
func getDateColumnValue(e *core.RequestEvent, forPayroll bool) (time.Time, error) {
	dateColumnValue := e.Request.PathValue("date_column_value")
	if dateColumnValue == "" {
		return time.Time{}, e.Error(http.StatusBadRequest, "date_column_value is required", nil)
	}

	// if dateColumnValue is not a valid date, return an error
	dateColumnValueDate, err := time.Parse("2006-01-02", dateColumnValue)
	if err != nil {
		return time.Time{}, e.Error(http.StatusBadRequest, "date_column_value must be a valid date", nil)
	}

	// if dateColumnValueDate is not a Saturday, return an error
	if dateColumnValueDate.Weekday() != time.Saturday {
		return time.Time{}, e.Error(http.StatusBadRequest, "date_column_value must be a Saturday", nil)
	}

	// If forPayroll is true, check if dateColumnValueDate is a multiple of 2
	// weeks (14 days) before or after the PAYROLL_EPOCH.
	if forPayroll {
		daysDifference := int(dateColumnValueDate.Sub(constants.PAYROLL_EPOCH).Hours() / 24)
		if daysDifference%14 != 0 {
			return time.Time{}, e.Error(http.StatusBadRequest, "date_column_value must be a multiple of 2 weeks before or after 2025-03-01 when forPayroll is true", nil)
		}
	}

	return dateColumnValueDate, nil
}
