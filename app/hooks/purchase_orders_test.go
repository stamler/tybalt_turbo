package hooks

import (
	"math"
	"testing"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestValidatePurchaseOrder_DescriptionMinimumLength(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	activeAdminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "active = true", dbx.Params{})
	if err != nil {
		t.Fatalf("failed to load active admin profile: %v", err)
	}
	activeVendor, err := app.FindFirstRecordByFilter("vendors", "status = 'Active'", dbx.Params{})
	if err != nil {
		t.Fatalf("failed to load active vendor: %v", err)
	}

	tests := map[string]struct {
		description string
		valid       bool
	}{
		"accepts_four_characters":  {description: "Fuel", valid: true},
		"rejects_three_characters": {description: "Gas", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			record := buildRecordFromMap(poCollection, map[string]any{
				"uid":          activeAdminProfile.GetString("uid"),
				"date":         "2024-09-01",
				"division":     "vccd5fo56ctbigh",
				"description":  tt.description,
				"payment_type": "Expense",
				"total":        1234.56,
				"vendor":       activeVendor.Id,
				"approver":     activeAdminProfile.GetString("uid"),
				"type":         "One-Time",
			})

			err := ValidatePurchaseOrder(app, record, true)
			if tt.valid {
				if err != nil {
					t.Fatalf("expected valid purchase order, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected description validation error, got nil")
			}

			errMap, ok := err.(validation.Errors)
			if !ok {
				t.Fatalf("expected validation.Errors, got %T: %v", err, err)
			}
			if _, ok := errMap["description"]; !ok {
				t.Fatalf("expected description error, got %v", errMap)
			}
		})
	}
}

func TestCleanPurchaseOrder_ComputesApprovalTotalHome(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	activeAdminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "active = true", dbx.Params{})
	if err != nil {
		t.Fatalf("failed to load active admin profile: %v", err)
	}
	cadCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", dbx.Params{"code": "CAD"})
	if err != nil {
		t.Fatalf("failed to load CAD currency: %v", err)
	}
	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", dbx.Params{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}

	tests := map[string]struct {
		currencyID string
		wantTotal  float64
	}{
		"home_currency_uses_nominal_total": {
			currencyID: cadCurrency.Id,
			wantTotal:  100,
		},
		"foreign_currency_uses_cached_rate": {
			currencyID: usdCurrency.Id,
			wantTotal:  100 * usdCurrency.GetFloat("rate"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			record := buildRecordFromMap(poCollection, map[string]any{
				"uid":          activeAdminProfile.GetString("uid"),
				"date":         "2024-09-01",
				"division":     "vccd5fo56ctbigh",
				"description":  "Fuel for active site work",
				"payment_type": "OnAccount",
				"total":        100.0,
				"vendor":       "2zqxtsmymf670ha",
				"approver":     activeAdminProfile.GetString("uid"),
				"type":         "One-Time",
				"currency":     tt.currencyID,
			})

			if err := CleanPurchaseOrder(app, record); err != nil {
				t.Fatalf("expected CleanPurchaseOrder to succeed, got %v", err)
			}

			if math.Abs(record.GetFloat("approval_total_home")-tt.wantTotal) > 0.000001 {
				t.Fatalf("expected approval_total_home %.6f, got %.6f", tt.wantTotal, record.GetFloat("approval_total_home"))
			}
		})
	}
}

func TestValidatePurchaseOrder_ChildCurrencyRules(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", dbx.Params{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}
	cadCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", dbx.Params{"code": "CAD"})
	if err != nil {
		t.Fatalf("failed to load CAD currency: %v", err)
	}

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE purchase_orders
		SET currency = {:currencyId}
		WHERE id = 'ly8xyzpuj79upq1'
	`).Bind(dbx.Params{"currencyId": usdCurrency.Id}).Execute(); err != nil {
		t.Fatalf("failed to prepare parent PO currency: %v", err)
	}

	parentPO, err := app.FindRecordById("purchase_orders", "ly8xyzpuj79upq1")
	if err != nil {
		t.Fatalf("failed to load parent PO: %v", err)
	}
	jobRecord, err := app.FindRecordById("jobs", parentPO.GetString("job"))
	if err != nil {
		t.Fatalf("failed to load parent PO job: %v", err)
	}

	newChild := func(currencyID string) *core.Record {
		return buildRecordFromMap(poCollection, map[string]any{
			"uid":          parentPO.GetString("uid"),
			"date":         "2024-09-15",
			"division":     parentPO.GetString("division"),
			"description":  parentPO.GetString("description"),
			"payment_type": parentPO.GetString("payment_type"),
			"total":        25.0,
			"vendor":       parentPO.GetString("vendor"),
			"approver":     parentPO.GetString("approver"),
			"type":         "One-Time",
			"job":          parentPO.GetString("job"),
			"category":     parentPO.GetString("category"),
			"kind":         parentPO.GetString("kind"),
			"branch":       jobRecord.GetString("branch"),
			"parent_po":    parentPO.Id,
			"currency":     currencyID,
		})
	}

	inheritedCurrencyChild := newChild("")
	if err := ValidatePurchaseOrder(app, inheritedCurrencyChild, true); err != nil {
		t.Fatalf("expected child PO with blank currency to inherit parent currency, got %v", err)
	}
	if got := inheritedCurrencyChild.GetString("currency"); got != usdCurrency.Id {
		t.Fatalf("expected child PO currency to inherit USD (%s), got %q", usdCurrency.Id, got)
	}

	mismatchedCurrencyChild := newChild(cadCurrency.Id)
	err = ValidatePurchaseOrder(app, mismatchedCurrencyChild, true)
	assertHookErrorCode(t, err, 400, "currency", "value_mismatch")
}
