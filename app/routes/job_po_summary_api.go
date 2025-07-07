package routes

import (
	"database/sql"
	_ "embed"
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_po_summary.sql
var jobPOSummaryQuery string

// poSummaryRow maps the result from job_po_summary.sql
type poSummaryRow struct {
	TotalAmount sql.NullString `db:"total_amount"`
	EarliestPO  sql.NullString `db:"earliest_po"`
	LatestPO    sql.NullString `db:"latest_po"`
	Divisions   sql.NullString `db:"divisions"`
	Types       sql.NullString `db:"types"`
	Names       sql.NullString `db:"names"`
}

// createGetJobPOSummaryHandler returns handler for summary of active purchase orders for a job
// Optional query params:
//   - division (division id)
//   - type     (purchase order type)
//   - uid      (user id)
func createGetJobPOSummaryHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		division := q.Get("division")
		poType := q.Get("type")
		uid := q.Get("uid")

		var row poSummaryRow
		if err := app.DB().NewQuery(jobPOSummaryQuery).Bind(dbx.Params{
			"id":       id,
			"division": division,
			"type":     poType,
			"uid":      uid,
		}).One(&row); err != nil {
			if err == sql.ErrNoRows {
				return e.JSON(http.StatusOK, map[string]any{})
			}
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		ns := func(n sql.NullString) string {
			if n.Valid {
				return n.String
			}
			return ""
		}

		var total float64
		if row.TotalAmount.Valid {
			if f, err := strconv.ParseFloat(row.TotalAmount.String, 64); err == nil {
				total = f
			}
		}

		resp := map[string]any{
			"total_amount": total,
			"earliest_po":  ns(row.EarliestPO),
			"latest_po":    ns(row.LatestPO),
			"divisions":    ns(row.Divisions),
			"types":        ns(row.Types),
			"names":        ns(row.Names),
		}

		return e.JSON(http.StatusOK, resp)
	}
}
