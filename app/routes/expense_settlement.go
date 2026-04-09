package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type expenseSettlementRow struct {
	ID                 string  `db:"id" json:"id"`
	Date               string  `db:"date" json:"date"`
	UID                string  `db:"uid" json:"uid"`
	CreatorName        string  `db:"creator_name" json:"creator_name"`
	VendorName         string  `db:"vendor_name" json:"vendor_name"`
	Description        string  `db:"description" json:"description"`
	PONumber           string  `db:"po_number" json:"po_number"`
	Currency           string  `db:"currency" json:"currency"`
	CurrencyCode       string  `db:"currency_code" json:"currency_code"`
	CurrencySymbol     string  `db:"currency_symbol" json:"currency_symbol"`
	CurrencyIcon       string  `db:"currency_icon" json:"currency_icon"`
	Total              float64 `db:"total" json:"total"`
	IndicativeCADTotal float64 `db:"indicative_cad_total" json:"indicative_cad_total"`
	CurrencyRate       float64 `db:"currency_rate" json:"currency_rate"`
	CurrencyRateDate   string  `db:"currency_rate_date" json:"currency_rate_date"`
	Approved           string  `db:"approved" json:"approved"`
	AgeDays            int     `db:"age_days" json:"age_days"`
	SettledTotal       float64 `db:"settled_total" json:"settled_total"`
	Settler            string  `db:"settler" json:"settler"`
	SettlerName        string  `db:"settler_name" json:"settler_name"`
	Settled            string  `db:"settled" json:"settled"`
	PaymentType        string  `db:"payment_type" json:"payment_type"`
	CCLast4Digits      string  `db:"cc_last_4_digits" json:"cc_last_4_digits"`
}

type settleExpenseRequest struct {
	SettledTotal float64 `json:"settled_total"`
}

func requirePayablesAdmin(app core.App, auth *core.Record) error {
	if auth == nil {
		return &errs.HookError{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized",
			Data: map[string]errs.CodeError{
				"global": {Code: "unauthorized", Message: "authentication is required"},
			},
		}
	}

	hasClaim, err := utilities.HasClaim(app, auth, "payables_admin")
	if err != nil {
		return err
	}
	if !hasClaim {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "payables_admin claim required",
			Data: map[string]errs.CodeError{
				"global": {Code: "unauthorized", Message: "payables_admin claim required"},
			},
		}
	}
	return nil
}

func createExpenseSettlementListHandler(app core.App, settled bool) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requirePayablesAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		query := `
			SELECT
				e.id,
				e.date,
				e.uid,
				COALESCE(p.given_name || ' ' || p.surname, '') AS creator_name,
				COALESCE(v.name, '') AS vendor_name,
				COALESCE(e.description, '') AS description,
				COALESCE(po.po_number, '') AS po_number,
				COALESCE(e.currency, '') AS currency,
				COALESCE(cur.code, 'CAD') AS currency_code,
				COALESCE(cur.symbol, 'CAD') AS currency_symbol,
				COALESCE(cur.icon, '') AS currency_icon,
				CAST(e.total AS REAL) AS total,
				CAST(
					e.total * CASE
						WHEN COALESCE(cur.code, 'CAD') = 'CAD' THEN 1
						ELSE COALESCE(CAST(cur.rate AS REAL), 0)
					END
				AS REAL) AS indicative_cad_total,
				CASE
					WHEN COALESCE(cur.code, 'CAD') = 'CAD' THEN 1
					ELSE COALESCE(CAST(cur.rate AS REAL), 0)
				END AS currency_rate,
				COALESCE(cur.rate_date, '') AS currency_rate_date,
				COALESCE(e.approved, '') AS approved,
				CAST(MAX(0, julianday('now') - julianday(substr(e.approved, 1, 10))) AS INTEGER) AS age_days,
				CAST(COALESCE(e.settled_total, 0) AS REAL) AS settled_total,
				COALESCE(e.settler, '') AS settler,
				COALESCE(sp.given_name || ' ' || sp.surname, '') AS settler_name,
				COALESCE(e.settled, '') AS settled,
				COALESCE(e.payment_type, '') AS payment_type,
				COALESCE(e.cc_last_4_digits, '') AS cc_last_4_digits
			FROM expenses e
			LEFT JOIN profiles p ON p.uid = e.uid
			LEFT JOIN profiles sp ON sp.uid = e.settler
			LEFT JOIN vendors v ON v.id = e.vendor
			LEFT JOIN purchase_orders po ON po.id = e.purchase_order
			LEFT JOIN currencies cur ON cur.id = e.currency
			WHERE e.payment_type IN ('OnAccount', 'CorporateCreditCard')
			  AND COALESCE(cur.code, 'CAD') != 'CAD'
			  AND COALESCE(e.approved, '') != ''
			  AND COALESCE(e.rejected, '') = ''
			  AND COALESCE(e.committed, '') = ''
			  AND (
			    ({:settled} = 1 AND COALESCE(e.settled, '') != '')
			    OR ({:settled} = 0 AND COALESCE(e.settled, '') = '')
			  )
			ORDER BY e.date DESC, e.created DESC
		`

		var rows []expenseSettlementRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"settled": boolToInt(settled),
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load settlement queue", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func validateExpenseSettlementCandidate(app core.App, record *core.Record, requireSettled bool) error {
	currencyInfo, err := utilities.ResolveCurrencyInfo(app, record.GetString("currency"))
	if err != nil {
		return err
	}
	if utilities.IsHomeCurrencyInfo(currencyInfo) {
		return fmt.Errorf("home-currency expenses are not part of the settlement queue")
	}
	if record.GetString("payment_type") != "OnAccount" && record.GetString("payment_type") != "CorporateCreditCard" {
		return fmt.Errorf("expense type is not eligible for settlement")
	}
	if record.GetDateTime("approved").IsZero() || !record.GetDateTime("rejected").IsZero() || !record.GetDateTime("committed").IsZero() {
		return fmt.Errorf("expense is not currently eligible for settlement")
	}
	if requireSettled && record.GetDateTime("settled").IsZero() {
		return fmt.Errorf("expense is not settled")
	}
	if !requireSettled && !record.GetDateTime("settled").IsZero() {
		return fmt.Errorf("expense is already settled")
	}
	return nil
}

func createSettleExpenseHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requirePayablesAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		var req settleExpenseRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid JSON body", err)
		}
		if req.SettledTotal <= 0 {
			return e.Error(http.StatusBadRequest, "settled total must be greater than 0", nil)
		}

		expenseID := e.Request.PathValue("id")
		if expenseID == "" {
			return e.Error(http.StatusBadRequest, "expense id is required", nil)
		}

		if err := app.RunInTransaction(func(txApp core.App) error {
			record, err := txApp.FindRecordById("expenses", expenseID)
			if err != nil {
				return err
			}
			if err := validateExpenseSettlementCandidate(txApp, record, false); err != nil {
				return err
			}
			currencyInfo, err := utilities.ResolveCurrencyInfo(txApp, record.GetString("currency"))
			if err != nil {
				return err
			}
			if rateErr := validatePositiveForeignCurrencyRate(currencyInfo); rateErr != nil {
				return rateErr
			}
			if !utilities.IsSettledTotalWithinTolerance(record.GetFloat("total"), req.SettledTotal, currencyInfo) {
				return &CodeError{
					Code:    "settled_total_out_of_range",
					Message: utilities.SettledTotalToleranceMessage(record.GetFloat("total"), currencyInfo),
				}
			}
			if limitErr := validateExpenseNoPurchaseOrderLimit(
				txApp,
				record,
				currencyInfo,
				req.SettledTotal,
			); limitErr != nil {
				return limitErr
			}

			record.Set("settled_total", req.SettledTotal)
			record.Set("settler", e.Auth.Id)
			record.Set("settled", time.Now())
			return txApp.Save(record)
		}); err != nil {
			if codeErr, ok := err.(*CodeError); ok {
				return e.JSON(http.StatusBadRequest, map[string]any{
					"code":    codeErr.Code,
					"message": codeErr.Message,
				})
			}
			return e.Error(http.StatusBadRequest, "failed to settle expense", err)
		}

		return e.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func createClearExpenseSettlementHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requirePayablesAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		expenseID := e.Request.PathValue("id")
		if expenseID == "" {
			return e.Error(http.StatusBadRequest, "expense id is required", nil)
		}

		if err := app.RunInTransaction(func(txApp core.App) error {
			record, err := txApp.FindRecordById("expenses", expenseID)
			if err != nil {
				return err
			}
			if !record.GetDateTime("committed").IsZero() {
				return fmt.Errorf("committed expenses cannot be modified")
			}
			if err := validateExpenseSettlementCandidate(txApp, record, true); err != nil {
				return err
			}

			record.Set("settled_total", 0)
			record.Set("settler", "")
			record.Set("settled", "")
			return txApp.Save(record)
		}); err != nil {
			return e.Error(http.StatusBadRequest, "failed to clear settlement", err)
		}

		return e.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}
