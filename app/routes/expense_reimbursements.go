package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed expense_reimbursements.sql
var expenseReimbursementsQuery string

type expenseReimbursementPeriodRow struct {
	PayPeriodEnding string  `db:"pay_period_ending" json:"pay_period_ending"`
	PayDate         string  `db:"pay_date" json:"pay_date"`
	ExpenseCount    int     `db:"expense_count" json:"expense_count"`
	Total           float64 `db:"total" json:"total"`
}

type expenseReimbursementRow struct {
	ID                   string  `db:"id" json:"id"`
	Date                 string  `db:"date" json:"date"`
	PayPeriodEnding      string  `db:"pay_period_ending" json:"pay_period_ending"`
	PayDate              string  `db:"pay_date" json:"pay_date"`
	PaymentType          string  `db:"payment_type" json:"payment_type"`
	Description          string  `db:"description" json:"description"`
	Distance             float64 `db:"distance" json:"distance"`
	JobNumber            string  `db:"job_number" json:"job_number"`
	JobDescription       string  `db:"job_description" json:"job_description"`
	ClientName           string  `db:"client_name" json:"client_name"`
	CurrencyCode         string  `db:"currency_code" json:"currency_code"`
	ForeignCurrencyTotal float64 `db:"foreign_currency_total" json:"foreign_currency_total"`
	Amount               float64 `db:"amount" json:"amount"`
}

// createExpenseReimbursementPeriodsHandler returns the caller's reimbursed
// expense totals grouped by pay period, newest first.
func createExpenseReimbursementPeriodsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		query := `
			SELECT
				pay_period_ending,
				pay_date,
				COUNT(*) AS expense_count,
				ROUND(SUM(amount), 2) AS total
			FROM (` + expenseReimbursementsQuery + `)
			GROUP BY pay_period_ending
			ORDER BY pay_period_ending DESC
		`

		rows := []expenseReimbursementPeriodRow{}
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"uid":               e.Auth.Id,
			"pay_period_ending": "",
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load reimbursements", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

// createExpenseReimbursementsListHandler returns the caller's reimbursed
// expenses for a single pay period.
func createExpenseReimbursementsListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		payPeriodEnding := e.Request.PathValue("payPeriodEnding")
		if _, err := time.Parse(time.DateOnly, payPeriodEnding); err != nil {
			return e.Error(http.StatusBadRequest, "invalid date format (YYYY-MM-DD)", nil)
		}

		query := `SELECT * FROM (` + expenseReimbursementsQuery + `) ORDER BY date, payment_type, id`

		rows := []expenseReimbursementRow{}
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"uid":               e.Auth.Id,
			"pay_period_ending": payPeriodEnding,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load reimbursements", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
