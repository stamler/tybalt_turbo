package routes

import (
	_ "embed" // for go:embed
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed vendors.sql
var vendorsQuery string

// vendorRow maps DB row
type vendorRow struct {
	ID                  string `db:"id"`
	Name                string `db:"name"`
	Alias               string `db:"alias"`
	ExpensesCount       int    `db:"expenses_count"`
	PurchaseOrdersCount int    `db:"purchase_orders_count"`
}

type Vendor struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Alias               string `json:"alias"`
	ExpensesCount       int    `json:"expenses_count"`
	PurchaseOrdersCount int    `json:"purchase_orders_count"`
}

func createGetVendorsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")

		var rows []vendorRow
		if err := app.DB().NewQuery(vendorsQuery).Bind(dbx.Params{"id": id}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		toVendor := func(r vendorRow) Vendor {
			return Vendor(r)
		}

		if id != "" {
			if len(rows) == 0 {
				return e.Error(http.StatusNotFound, "vendor not found", nil)
			}
			return e.JSON(http.StatusOK, toVendor(rows[0]))
		}

		resp := make([]Vendor, len(rows))
		for i, r := range rows {
			resp[i] = toVendor(r)
		}
		return e.JSON(http.StatusOK, resp)
	}
}
