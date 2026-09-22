package routes

import (
	"database/sql"
	_ "embed"
	"errors"
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_wip.sql
var jobWIPQuery string

// JobWIP contains separate CAD factors. Inclusion controls never change the
// expense deduction from active POs; this prevents duplicate commitments.
type JobWIP struct {
	ProjectValue     float64 `db:"project_value" json:"project_value"`
	AsOf             string  `db:"as_of" json:"as_of"`
	NoRateSheet      bool    `db:"no_rate_sheet" json:"no_rate_sheet"`
	TimeValue        float64 `db:"time_value" json:"time_value"`
	Hours            float64 `db:"hours" json:"hours"`
	EstimatedHours   float64 `db:"estimated_hours" json:"estimated_hours"`
	UnpricedHours    float64 `db:"unpriced_hours" json:"unpriced_hours"`
	ExpenseValue     float64 `db:"expense_value" json:"expense_value"`
	UnpricedExpenses int     `db:"unpriced_expenses" json:"unpriced_expenses"`
	POValue          float64 `db:"po_value" json:"po_value"`
	UnpricedPOs      int     `db:"unpriced_pos" json:"unpriced_pos"`
	EstimatedPOs     int     `db:"estimated_pos" json:"estimated_pos"`
}

func createGetJobWIPHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var row JobWIP
		// This is a current report, not a historical snapshot. Active POs may
		// cover future work. Time and expenses only include dates through today.
		err := app.DB().NewQuery(jobPricedTimeEntriesQuery + jobWIPQuery).Bind(dbx.Params{
			"job_id":     e.Request.PathValue("id"),
			"start_date": "",
			"end_date":   time.Now().UTC().Format("2006-01-02"),
		}).One(&row)
		if errors.Is(err, sql.ErrNoRows) {
			return e.NotFoundError("Job not found", nil)
		}
		if err != nil {
			return e.InternalServerError("Failed to load WIP", err)
		}
		return e.JSON(http.StatusOK, row)
	}
}
