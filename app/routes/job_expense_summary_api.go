package routes

import (
	"database/sql"
	_ "embed" // Needed for //go:embed
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_expense_summary.sql
var jobExpenseSummaryQuery string

// expenseSummaryRow maps the single-row result of job_expense_summary.sql.
type expenseSummaryRow struct {
	TotalAmount     sql.NullString `db:"total_amount"`
	EarliestExpense sql.NullString `db:"earliest_expense"`
	LatestExpense   sql.NullString `db:"latest_expense"`
	Branches        sql.NullString `db:"branches"`
	Divisions       sql.NullString `db:"divisions"`
	PaymentTypes    sql.NullString `db:"payment_types"`
	Names           sql.NullString `db:"names"`
	Categories      sql.NullString `db:"categories"`
}

// createGetJobExpenseSummaryHandler returns an HTTP handler that executes the
// job_expense_summary.sql query for the requested job id and optional filters.
// The optional filters are provided as query parameters:
//   - division      (division id)
//   - payment_type  (payment_type string)
//   - uid           (user id)
//   - category      (category id)
func createGetJobExpenseSummaryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		division := q.Get("division")
		branch := q.Get("branch")
		paymentType := q.Get("payment_type")
		uid := q.Get("uid")
		category := q.Get("category")

		var row expenseSummaryRow
		if err := app.DB().NewQuery(jobExpenseSummaryQuery).Bind(dbx.Params{
			"id":           id,
			"branch":       branch,
			"division":     division,
			"payment_type": paymentType,
			"uid":          uid,
			"category":     category,
		}).One(&row); err != nil {
			if err == sql.ErrNoRows {
				return e.JSON(http.StatusOK, map[string]any{})
			}
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		// Helper to unwrap sql.NullString to string
		ns := func(n sql.NullString) string {
			if n.Valid {
				return n.String
			}
			return ""
		}

		// Convert total_amount to float64 if possible; otherwise 0
		var total float64
		if row.TotalAmount.Valid {
			if f, err := strconv.ParseFloat(row.TotalAmount.String, 64); err == nil {
				total = f
			}
		}

		resp := map[string]any{
			"total_amount":     total,
			"earliest_expense": ns(row.EarliestExpense),
			"latest_expense":   ns(row.LatestExpense),
			"branches":         ns(row.Branches),
			"divisions":        ns(row.Divisions),
			"payment_types":    ns(row.PaymentTypes),
			"names":            ns(row.Names),
			"categories":       ns(row.Categories),
		}

		return e.JSON(http.StatusOK, resp)
	}
}
