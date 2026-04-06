package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type currencyInitStatus struct {
	HomeCurrencyID      string `json:"home_currency_id"`
	HomeCurrencyExists  bool   `json:"home_currency_exists"`
	HomeCurrencyReady   bool   `json:"home_currency_ready"`
	BlankPurchaseOrders int    `json:"blank_purchase_orders"`
	BlankExpenses       int    `json:"blank_expenses"`
}

type currencyListRow struct {
	ID                   string  `db:"id" json:"id"`
	Code                 string  `db:"code" json:"code"`
	Symbol               string  `db:"symbol" json:"symbol"`
	Icon                 string  `db:"icon" json:"icon"`
	Rate                 float64 `db:"rate" json:"rate"`
	RateDate             string  `db:"rate_date" json:"rate_date"`
	UISort               int     `db:"ui_sort" json:"ui_sort"`
	UsedByPurchaseOrders bool    `db:"used_by_purchase_orders" json:"used_by_purchase_orders"`
	UsedByExpenses       bool    `db:"used_by_expenses" json:"used_by_expenses"`
}

type currencyRatesReloadResponse struct {
	OK           bool `json:"ok"`
	Updated      int  `json:"updated"`
	SkippedNewer int  `json:"skipped_newer"`
}

var runCurrencyRateSync = utilities.SyncCurrencyRatesWithResult

func requireAdmin(app core.App, auth *core.Record) error {
	if auth == nil {
		return &errs.HookError{
			Status:  http.StatusUnauthorized,
			Message: "unauthorized",
			Data: map[string]errs.CodeError{
				"global": {Code: "unauthorized", Message: "authentication is required"},
			},
		}
	}

	hasAdmin, err := utilities.HasClaim(app, auth, "admin")
	if err != nil {
		return err
	}
	if !hasAdmin {
		return &errs.HookError{
			Status:  http.StatusForbidden,
			Message: "admin claim required",
			Data: map[string]errs.CodeError{
				"global": {Code: "unauthorized", Message: "admin claim required"},
			},
		}
	}

	return nil
}

func getCurrencyInitStatus(app core.App) (currencyInitStatus, error) {
	status := currencyInitStatus{}

	if home, err := utilities.FindHomeCurrency(app); err == nil && home != nil {
		status.HomeCurrencyExists = true
		status.HomeCurrencyID = home.Id
		status.HomeCurrencyReady = true
	}

	if err := app.DB().NewQuery(`
		SELECT COUNT(*)
		FROM purchase_orders
		WHERE COALESCE(currency, '') = ''
	`).Row(&status.BlankPurchaseOrders); err != nil {
		return status, err
	}

	if err := app.DB().NewQuery(`
		SELECT COUNT(*)
		FROM expenses
		WHERE COALESCE(currency, '') = ''
	`).Row(&status.BlankExpenses); err != nil {
		return status, err
	}

	return status, nil
}

func createCurrencyInitStatusHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		status, err := getCurrencyInitStatus(app)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load currency initialization status", err)
		}

		return e.JSON(http.StatusOK, status)
	}
}

func createCurrencyRatesReloadHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		result, err := runCurrencyRateSync(app)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to reload currency rates", err)
		}

		return e.JSON(http.StatusOK, currencyRatesReloadResponse{
			OK:           true,
			Updated:      result.Updated,
			SkippedNewer: result.SkippedNewer,
		})
	}
}

func createCurrencyInitializeBackfillHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		currencyID := strings.TrimSpace(e.Request.PathValue("id"))
		if currencyID == "" {
			return e.Error(http.StatusBadRequest, "currency id is required", nil)
		}

		record, err := app.FindRecordById("currencies", currencyID)
		if err != nil {
			return e.Error(http.StatusNotFound, "currency not found", err)
		}
		if !strings.EqualFold(strings.TrimSpace(record.GetString("code")), utilities.HomeCurrencyCode) {
			return e.Error(http.StatusBadRequest, "initialization requires the CAD currency row", nil)
		}

		if err := app.RunInTransaction(func(txApp core.App) error {
			params := dbx.Params{"currencyId": currencyID}
			if _, err := txApp.DB().NewQuery(`
				UPDATE purchase_orders
				SET currency = {:currencyId}
				WHERE COALESCE(currency, '') = ''
			`).Bind(params).Execute(); err != nil {
				return err
			}

			if _, err := txApp.DB().NewQuery(`
				UPDATE purchase_orders
				SET approval_total_home = approval_total
				WHERE COALESCE(approval_total_home, 0) = 0
				   OR COALESCE(currency, '') = {:currencyId}
			`).Bind(params).Execute(); err != nil {
				return err
			}

			if _, err := txApp.DB().NewQuery(`
				UPDATE expenses
				SET currency = {:currencyId}
				WHERE COALESCE(currency, '') = ''
			`).Bind(params).Execute(); err != nil {
				return err
			}

			if _, err := txApp.DB().NewQuery(`
				UPDATE expenses
				SET settled_total = total
				WHERE COALESCE(settled_total, 0) = 0
				  AND (
					COALESCE(currency, '') = ''
					OR COALESCE(currency, '') = {:currencyId}
				  )
			`).Bind(params).Execute(); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to initialize CAD backfill", err)
		}

		status, err := getCurrencyInitStatus(app)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load currency initialization status", err)
		}

		return e.JSON(http.StatusOK, status)
	}
}

// This request is intentionally simple because the page uses direct PocketBase
// collection create/update/delete APIs for normal row management.
type updateCurrencyRequest struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
	UISort int    `json:"ui_sort"`
}

func createGetCurrenciesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		var rows []currencyListRow
		query := `
			SELECT
				c.id,
				c.code,
				c.symbol,
				COALESCE(c.icon, '') AS icon,
				COALESCE(CAST(c.rate AS REAL), 0) AS rate,
				COALESCE(c.rate_date, '') AS rate_date,
				COALESCE(CAST(c.ui_sort AS INTEGER), 0) AS ui_sort,
				EXISTS(SELECT 1 FROM purchase_orders po WHERE po.currency = c.id) AS used_by_purchase_orders,
				EXISTS(SELECT 1 FROM expenses e WHERE e.currency = c.id) AS used_by_expenses
			FROM currencies c
			ORDER BY COALESCE(c.ui_sort, 999999), c.code
		`

		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to fetch currencies", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

func createDeleteCurrencyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdmin(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		id := strings.TrimSpace(e.Request.PathValue("id"))
		if id == "" {
			return e.Error(http.StatusBadRequest, "currency id is required", nil)
		}

		type refCount struct {
			PurchaseOrders int `db:"purchase_orders"`
			Expenses       int `db:"expenses"`
		}
		var counts refCount
		if err := app.DB().NewQuery(`
			SELECT
				(SELECT COUNT(*) FROM purchase_orders WHERE currency = {:id}) AS purchase_orders,
				(SELECT COUNT(*) FROM expenses WHERE currency = {:id}) AS expenses
		`).Bind(dbx.Params{"id": id}).One(&counts); err != nil {
			return e.Error(http.StatusInternalServerError, "failed checking currency references", err)
		}
		if counts.PurchaseOrders > 0 || counts.Expenses > 0 {
			return e.Error(http.StatusBadRequest, "cannot delete a currency that is already in use", nil)
		}

		record, err := app.FindRecordById("currencies", id)
		if err != nil {
			return e.Error(http.StatusNotFound, "currency not found", err)
		}

		if err := app.Delete(record); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to delete currency", err)
		}

		return e.JSON(http.StatusOK, map[string]bool{"ok": true})
	}
}

func decodeCurrencyRequest(e *core.RequestEvent) (updateCurrencyRequest, error) {
	var req updateCurrencyRequest
	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return req, err
	}
	req.Code = strings.ToUpper(strings.TrimSpace(req.Code))
	req.Symbol = strings.TrimSpace(req.Symbol)
	return req, nil
}
