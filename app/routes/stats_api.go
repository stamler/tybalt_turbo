package routes

import (
	_ "embed"
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed stats.sql
var statsQuery string

type StatsResponse struct {
	QualifyingPOCount    int `db:"qualifying_po_count" json:"qualifying_po_count"`
	ApprovedExpenseCount int `db:"approved_expense_count" json:"approved_expense_count"`
	DistinctUserCount    int `db:"distinct_user_count" json:"distinct_user_count"`
}

func createGetStatsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		hasAdminClaim, err := utilities.HasClaim(app, authRecord, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking admin claim", err)
		}
		if !hasAdminClaim {
			return e.Error(http.StatusForbidden, "admin claim required", nil)
		}

		var rows []StatsResponse
		if err := app.DB().NewQuery(statsQuery).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute stats query: "+err.Error(), err)
		}

		if len(rows) == 0 {
			return e.JSON(http.StatusOK, StatsResponse{})
		}

		return e.JSON(http.StatusOK, rows[0])
	}
}
