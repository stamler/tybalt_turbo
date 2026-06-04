package routes

import (
	"net/http"
	"strings"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	navTimeSheetsPendingHref     = "/time/sheets/pending"
	navExpensesPendingHref       = "/expenses/pending"
	navPurchaseOrdersPendingHref = "/pos/pending"
	navProjectAuthorizationHref  = "/jobs/project_authorization"
	navExpenseCommitQueueHref    = "/reports/expense/queue"
	navExpenseSettlementHref     = "/expenses/settlement"
)

var navPendingPurchaseOrdersCountQuery string

func init() {
	navPendingPurchaseOrdersCountQuery = strings.ReplaceAll(`
		WITH visibility_base AS (
		__PO_VISIBILITY_BASE__
		)
		SELECT COUNT(*)
		FROM visibility_base
		WHERE is_unapproved_actionable_now = 1
	`, poVisibilityBaseToken, poVisibilityBaseQuery)
}

func countNavRows(app core.App, query string, params dbx.Params) (int, error) {
	var count int
	err := app.DB().NewQuery(query).Bind(params).Row(&count)
	return count, err
}

func createGetNavBadgesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		counts := map[string]int{}
		userID := e.Auth.Id

		timeSheetsPendingCount, err := countNavRows(app, `
			SELECT COUNT(*)
			FROM (
				SELECT te.tsid
				FROM time_entries te
				INNER JOIN time_sheets ts ON te.tsid = ts.id
				WHERE ts.approver = {:uid}
				  AND ts.approved = ''
				GROUP BY te.tsid
			)
		`, dbx.Params{"uid": userID})
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count pending time sheets", err)
		}
		counts[navTimeSheetsPendingHref] = timeSheetsPendingCount

		expensesPendingCount, err := countNavRows(
			app,
			buildCountQuery(wherePending),
			dbx.Params{"auth": userID},
		)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count pending expenses", err)
		}
		counts[navExpensesPendingHref] = expensesPendingCount

		purchaseOrdersPendingCount, err := countNavRows(
			app,
			navPendingPurchaseOrdersCountQuery,
			purchaseOrderVisibilityParams(app, userID, "all", "", "", 0),
		)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count pending purchase orders", err)
		}
		counts[navPurchaseOrdersPendingHref] = purchaseOrdersPendingCount

		hasAccounting, err := utilities.HasClaim(app, e.Auth, "accounting")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check accounting claim", err)
		}
		if hasAccounting {
			projectAuthorizationCount, err := countNavRows(app, `
				SELECT COUNT(*)
				FROM jobs j
				WHERE j.status = 'Active'
				  AND j.number NOT LIKE 'P%'
				  AND j.project_authorization_doc != ''
				  AND j.project_authorization_doc_hash != ''
				  AND j.pa_reviewed = ''
				  AND j.pa_reviewer = ''
			`, dbx.Params{})
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to count project authorization queue", err)
			}
			counts[navProjectAuthorizationHref] = projectAuthorizationCount
		}

		hasCommit, err := utilities.HasClaim(app, e.Auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check commit claim", err)
		}
		if hasCommit {
			commitQueueCount, err := countNavRows(app, `
				SELECT COUNT(*)
				FROM expenses e
				LEFT JOIN currencies cur ON cur.id = e.currency
				WHERE e.submitted = 1
				  AND e.committed = ''
				  AND e.approved != ''
				  AND NOT (
				    COALESCE(cur.code, 'CAD') != 'CAD'
				    AND e.payment_type IN ('OnAccount', 'CorporateCreditCard')
				    AND COALESCE(e.settled, '') = ''
				  )
			`, dbx.Params{})
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to count expense commit queue", err)
			}
			counts[navExpenseCommitQueueHref] = commitQueueCount
		}

		hasPayablesAdmin, err := utilities.HasClaim(app, e.Auth, "payables_admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to check payables_admin claim", err)
		}
		if hasPayablesAdmin {
			settlementCount, err := countNavRows(app, `
				SELECT COUNT(*)
				FROM expenses e
				LEFT JOIN currencies cur ON cur.id = e.currency
				WHERE e.payment_type IN ('OnAccount', 'CorporateCreditCard')
				  AND COALESCE(cur.code, 'CAD') != 'CAD'
				  AND COALESCE(e.approved, '') != ''
				  AND COALESCE(e.rejected, '') = ''
				  AND COALESCE(e.committed, '') = ''
				  AND COALESCE(e.settled, '') = ''
			`, dbx.Params{})
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to count expense settlement queue", err)
			}
			counts[navExpenseSettlementHref] = settlementCount
		}

		return e.JSON(http.StatusOK, counts)
	}
}
