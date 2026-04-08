package reports

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const payablesSpreadsheetNowStoreKey = "tybalt.reports.payablesSpreadsheetNow"

func currentPayablesSpreadsheetTime(app core.App) time.Time {
	if override, ok := app.Store().Get(payablesSpreadsheetNowStoreKey).(func() time.Time); ok && override != nil {
		return override().UTC()
	}
	return time.Now().UTC()
}

// SetPayablesSpreadsheetNowForTest overrides the clock used by the payables
// spreadsheet handlers for a specific app instance.
func SetPayablesSpreadsheetNowForTest(app core.App, now func() time.Time) {
	if now == nil {
		app.Store().Remove(payablesSpreadsheetNowStoreKey)
		return
	}
	app.Store().Set(payablesSpreadsheetNowStoreKey, now)
}

//go:embed payables_spreadsheet.sql
var payablesSpreadsheetQuery string

var payablesSpreadsheetHeaders = []string{
	"Acct/Visa/Exp", "Job #", "Div", "Branch", "type", "Date", "Mon", "Year",
	"Subtotal", "HST", "Total", "Unit", "NC", "Meals",
	"PO#", "Category", "Description", "Supplier", "Employee",
	"Approved By", "Entered By", "Vendor Inv #", "Inv Date",
	"Notes", "Pd By", "TBTE #", "Status",
}

type payablesRow struct {
	PaymentType  string `db:"payment_type"`
	JobNumber    string `db:"job_number"`
	DivisionCode string `db:"division_code"`
	BranchCode   string `db:"branch_code"`
	POType       string `db:"po_type"`
	RecordDate   string `db:"record_date"`
	ApprovalDate string `db:"approval_date"`
	Total        string `db:"total"`
	PONumber     string `db:"po_number"`
	Description  string `db:"description"`
	VendorName   string `db:"vendor_name"`
	Employee     string `db:"employee"`
	ApprovedBy   string `db:"approved_by"`
	Status       string `db:"status"`
}

func (r payablesRow) toRecord() []string {
	day, mon, year := "", "", ""
	if r.RecordDate != "" {
		if t, err := time.Parse("2006-01-02", r.RecordDate); err == nil {
			day = fmt.Sprintf("%d", t.Day())
			mon = t.Format("Jan")
			year = fmt.Sprintf("%d", t.Year())
		}
	}
	return []string{
		r.PaymentType, r.JobNumber, r.DivisionCode, r.BranchCode, r.POType,
		day, mon, year,
		"", "", r.Total, "", "", "",
		r.PONumber, "", r.Description, r.VendorName, r.Employee,
		r.ApprovedBy, "TURBO", "", "", "", "", "",
		r.Status,
	}
}

func queryPayablesRows(app core.App, extraWhere string, params dbx.Params) ([]payablesRow, error) {
	query := payablesSpreadsheetQuery
	if extraWhere != "" {
		query += "\n  AND " + extraWhere
	}
	query += "\nORDER BY approval_date ASC"

	var rows []payablesRow
	err := app.DB().NewQuery(query).Bind(params).All(&rows)
	return rows, err
}

func rowsToCSV(rows []payablesRow) (string, error) {
	var b strings.Builder
	w := csv.NewWriter(&b)
	if err := w.Write(payablesSpreadsheetHeaders); err != nil {
		return "", err
	}
	for _, row := range rows {
		if err := w.Write(row.toRecord()); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return b.String(), nil
}

func rowsToTSV(rows []payablesRow) string {
	var b strings.Builder
	for _, row := range rows {
		b.WriteString(strings.Join(row.toRecord(), "\t"))
		b.WriteString("\n")
	}
	return b.String()
}

func requireReportClaim(app core.App, e *core.RequestEvent) error {
	isReportHolder, err := utilities.HasClaim(app, e.Auth, "report")
	if err != nil {
		return e.Error(http.StatusInternalServerError, "error checking claims", err)
	}
	if !isReportHolder {
		return e.Error(http.StatusForbidden, "you are not authorized to view this report", nil)
	}
	return nil
}

// CreatePayablesSpreadsheetDatesHandler returns distinct approval dates with PO data,
// going back 4 weeks and excluding the current UTC day so only complete days appear.
//
// Keep the PO eligibility rule here in sync with payables_spreadsheet.sql.
const payablesEligiblePONumberWhere = `(INSTR(po.po_number, '-') = 0 OR CAST(SUBSTR(po.po_number, INSTR(po.po_number, '-') + 1) AS INTEGER) < 5000)`

func CreatePayablesSpreadsheetDatesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireReportClaim(app, e); err != nil {
			return err
		}

		nowUTC := currentPayablesSpreadsheetTime(app)
		fourWeeksAgo := nowUTC.AddDate(0, 0, -28).Format("2006-01-02")
		latestAvailableDate := nowUTC.AddDate(0, 0, -1).Format("2006-01-02")

		approvalDateExpr := "CASE WHEN po.second_approval != '' AND po.second_approval > po.approved THEN po.second_approval ELSE po.approved END"

		datesQuery := fmt.Sprintf(
			"SELECT DISTINCT SUBSTR(%s, 1, 10) AS date_val FROM purchase_orders po WHERE po.status != 'Unapproved' AND po.po_number != '' AND %s AND (po.approved != '' OR po.second_approval != '') AND SUBSTR(%s, 1, 10) >= {:fourWeeksAgo} AND SUBSTR(%s, 1, 10) <= {:latestAvailableDate} ORDER BY date_val ASC",
			approvalDateExpr, payablesEligiblePONumberWhere, approvalDateExpr, approvalDateExpr,
		)

		type dateRow struct {
			DateVal string `db:"date_val"`
		}
		var dateRows []dateRow
		err := app.DB().NewQuery(datesQuery).Bind(dbx.Params{
			"fourWeeksAgo":        fourWeeksAgo,
			"latestAvailableDate": latestAvailableDate,
		}).All(&dateRows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query dates: "+err.Error(), err)
		}

		dates := make([]string, 0, len(dateRows))
		for _, r := range dateRows {
			if r.DateVal != "" {
				dates = append(dates, r.DateVal)
			}
		}

		return e.JSON(http.StatusOK, dates)
	}
}

// CreatePayablesSpreadsheetHandler returns PO data for a given approval date as CSV or TSV.
func CreatePayablesSpreadsheetHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireReportClaim(app, e); err != nil {
			return err
		}

		dateStr := e.Request.PathValue("date")
		if dateStr == "" {
			return e.Error(http.StatusBadRequest, "date is required", nil)
		}
		dateValue, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return e.Error(http.StatusBadRequest, "date must be in YYYY-MM-DD format", nil)
		}
		if !dateValue.Before(currentPayablesSpreadsheetTime(app).Truncate(24 * time.Hour)) {
			return e.Error(http.StatusBadRequest, "date must be at least one UTC day old", nil)
		}

		rows, err := queryPayablesRows(app,
			"SUBSTR(CASE WHEN po.second_approval != '' AND po.second_approval > po.approved THEN po.second_approval ELSE po.approved END, 1, 10) = {:approvalDate}",
			dbx.Params{"approvalDate": dateStr},
		)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query payables: "+err.Error(), err)
		}

		format := e.Request.URL.Query().Get("format")
		if format == "tsv" {
			e.Response.Header().Set("Content-Type", "text/tab-separated-values")
			return e.String(http.StatusOK, rowsToTSV(rows))
		}

		csvString, err := rowsToCSV(rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to generate CSV: "+err.Error(), err)
		}
		e.Response.Header().Set("Content-Type", "text/csv")
		return e.String(http.StatusOK, csvString)
	}
}

var yymmPattern = regexp.MustCompile(`^\d{4}$`)

// CreatePayablesSpreadsheetMonthlyHandler returns PO data for a given month (YYMM format) as CSV or TSV.
func CreatePayablesSpreadsheetMonthlyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireReportClaim(app, e); err != nil {
			return err
		}

		yymm := e.Request.PathValue("yymm")
		if !yymmPattern.MatchString(yymm) {
			return e.Error(http.StatusBadRequest, "yymm must be a 4-digit string (e.g. 2603)", nil)
		}

		prefix := yymm + "-"
		rows, err := queryPayablesRows(app,
			"po.po_number LIKE {:prefix}",
			dbx.Params{"prefix": prefix + "%"},
		)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query payables: "+err.Error(), err)
		}

		format := e.Request.URL.Query().Get("format")
		if format == "tsv" {
			e.Response.Header().Set("Content-Type", "text/tab-separated-values")
			return e.String(http.StatusOK, rowsToTSV(rows))
		}

		csvString, err := rowsToCSV(rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to generate CSV: "+err.Error(), err)
		}
		e.Response.Header().Set("Content-Type", "text/csv")
		return e.String(http.StatusOK, csvString)
	}
}
