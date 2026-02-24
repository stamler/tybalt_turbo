package utilities

import (
	"math"
	"testing"

	"tybalt/constants"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func upsertPurchaseOrdersConfig(t *testing.T, app *tests.TestApp, rawValue string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "purchase_orders")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "purchase_orders")
	}

	record.Set("value", rawValue)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save purchase_orders config: %v", err)
	}
}

func deletePurchaseOrdersConfig(t *testing.T, app *tests.TestApp) {
	t.Helper()

	record, err := app.FindFirstRecordByData("app_config", "key", "purchase_orders")
	if err != nil || record == nil {
		return
	}
	if err := app.Delete(record); err != nil {
		t.Fatalf("failed to delete purchase_orders config: %v", err)
	}
}

func upsertNotificationsConfig(t *testing.T, app *tests.TestApp, rawValue string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "notifications")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "notifications")
	}

	record.Set("value", rawValue)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save notifications config: %v", err)
	}
}

func deleteNotificationsConfig(t *testing.T, app *tests.TestApp) {
	t.Helper()

	record, err := app.FindFirstRecordByData("app_config", "key", "notifications")
	if err != nil || record == nil {
		return
	}
	if err := app.Delete(record); err != nil {
		t.Fatalf("failed to delete notifications config: %v", err)
	}
}

func TestGetPurchaseOrderSecondStageTimeoutHours(t *testing.T) {
	testsTable := []struct {
		name  string
		setup func(t *testing.T, app *tests.TestApp)
		want  float64
	}{
		{
			name: "returns default when purchase_orders config is missing",
			setup: func(t *testing.T, app *tests.TestApp) {
				deletePurchaseOrdersConfig(t, app)
			},
			want: 24.0,
		},
		{
			name: "returns default when timeout key is missing",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertPurchaseOrdersConfig(t, app, `{"some_other_key":1}`)
			},
			want: 24.0,
		},
		{
			name: "returns default when timeout is zero",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertPurchaseOrdersConfig(t, app, `{"second_stage_timeout_hours":0}`)
			},
			want: 24.0,
		},
		{
			name: "returns default when timeout is negative",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertPurchaseOrdersConfig(t, app, `{"second_stage_timeout_hours":-3}`)
			},
			want: 24.0,
		},
		{
			name: "returns configured positive timeout",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertPurchaseOrdersConfig(t, app, `{"second_stage_timeout_hours":36.5}`)
			},
			want: 36.5,
		},
	}

	for _, tc := range testsTable {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp("../test_pb_data")
			if err != nil {
				t.Fatalf("failed to init test app: %v", err)
			}
			defer app.Cleanup()

			tc.setup(t, app)

			got := GetPurchaseOrderSecondStageTimeoutHours(app)
			if got != tc.want {
				t.Fatalf("GetPurchaseOrderSecondStageTimeoutHours() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsNotificationFeatureEnabled(t *testing.T) {
	testsTable := []struct {
		name        string
		setup       func(t *testing.T, app *tests.TestApp)
		template    string
		wantEnabled bool
	}{
		{
			name: "returns default false when notifications config is missing",
			setup: func(t *testing.T, app *tests.TestApp) {
				deleteNotificationsConfig(t, app)
			},
			template:    "po_active",
			wantEnabled: false,
		},
		{
			name: "returns default false when template key is missing",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertNotificationsConfig(t, app, `{"expense_rejected":false}`)
			},
			template:    "po_active",
			wantEnabled: false,
		},
		{
			name: "returns false when template key is explicitly disabled",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertNotificationsConfig(t, app, `{"po_active":false}`)
			},
			template:    "po_active",
			wantEnabled: false,
		},
		{
			name: "returns true when template key is explicitly enabled",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertNotificationsConfig(t, app, `{"po_active":true}`)
			},
			template:    "po_active",
			wantEnabled: true,
		},
	}

	for _, tc := range testsTable {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp("../test_pb_data")
			if err != nil {
				t.Fatalf("failed to init test app: %v", err)
			}
			defer app.Cleanup()

			tc.setup(t, app)

			enabled, err := IsNotificationFeatureEnabled(app, tc.template)
			if err != nil {
				t.Fatalf("IsNotificationFeatureEnabled() returned error: %v", err)
			}
			if enabled != tc.wantEnabled {
				t.Fatalf("IsNotificationFeatureEnabled() = %v, want %v", enabled, tc.wantEnabled)
			}
		})
	}
}

func upsertExpensesConfig(t *testing.T, app *tests.TestApp, rawValue string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "expenses")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "expenses")
	}

	record.Set("value", rawValue)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save expenses config: %v", err)
	}
}

func deleteExpensesConfig(t *testing.T, app *tests.TestApp) {
	t.Helper()

	record, err := app.FindFirstRecordByData("app_config", "key", "expenses")
	if err != nil || record == nil {
		return
	}
	if err := app.Delete(record); err != nil {
		t.Fatalf("failed to delete expenses config: %v", err)
	}
}

func TestGetPOExpenseExcessConfig(t *testing.T) {
	testsTable := []struct {
		name        string
		setup       func(t *testing.T, app *tests.TestApp)
		wantPercent float64
		wantValue   float64
		wantMode    string
	}{
		{
			name: "returns defaults when expenses config is missing",
			setup: func(t *testing.T, app *tests.TestApp) {
				deleteExpensesConfig(t, app)
			},
			wantPercent: constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT,
			wantValue:   constants.MAX_PURCHASE_ORDER_EXCESS_VALUE,
			wantMode:    "lesser_of",
		},
		{
			name: "returns defaults when po_expense_allowed_excess key is missing",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"create_edit_absorb":true}`)
			},
			wantPercent: constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT,
			wantValue:   constants.MAX_PURCHASE_ORDER_EXCESS_VALUE,
			wantMode:    "lesser_of",
		},
		{
			name: "returns configured values when all fields present",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"po_expense_allowed_excess":{"percent":10,"value":200.0,"mode":"greater_of"}}`)
			},
			wantPercent: 0.10,
			wantValue:   200.0,
			wantMode:    "greater_of",
		},
		{
			name: "returns partial override with defaults for missing fields",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"po_expense_allowed_excess":{"percent":3}}`)
			},
			wantPercent: 0.03,
			wantValue:   constants.MAX_PURCHASE_ORDER_EXCESS_VALUE,
			wantMode:    "lesser_of",
		},
		{
			name: "returns default mode for invalid mode string",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"po_expense_allowed_excess":{"percent":5,"value":100.0,"mode":"invalid"}}`)
			},
			wantPercent: 0.05,
			wantValue:   100.0,
			wantMode:    "lesser_of",
		},
		{
			name: "accepts zero percent",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"po_expense_allowed_excess":{"percent":0,"value":50.0,"mode":"lesser_of"}}`)
			},
			wantPercent: 0,
			wantValue:   50.0,
			wantMode:    "lesser_of",
		},
		{
			name: "rejects negative percent and uses default",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"po_expense_allowed_excess":{"percent":-5,"value":100.0,"mode":"lesser_of"}}`)
			},
			wantPercent: constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT,
			wantValue:   100.0,
			wantMode:    "lesser_of",
		},
		{
			name: "rejects percent above 100 and uses default",
			setup: func(t *testing.T, app *tests.TestApp) {
				upsertExpensesConfig(t, app, `{"po_expense_allowed_excess":{"percent":150,"value":100.0,"mode":"lesser_of"}}`)
			},
			wantPercent: constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT,
			wantValue:   100.0,
			wantMode:    "lesser_of",
		},
	}

	for _, tc := range testsTable {
		t.Run(tc.name, func(t *testing.T) {
			app, err := tests.NewTestApp("../test_pb_data")
			if err != nil {
				t.Fatalf("failed to init test app: %v", err)
			}
			defer app.Cleanup()

			tc.setup(t, app)

			got := GetPOExpenseExcessConfig(app)
			if got.Percent != tc.wantPercent {
				t.Errorf("Percent = %v, want %v", got.Percent, tc.wantPercent)
			}
			if got.Value != tc.wantValue {
				t.Errorf("Value = %v, want %v", got.Value, tc.wantValue)
			}
			if got.Mode != tc.wantMode {
				t.Errorf("Mode = %q, want %q", got.Mode, tc.wantMode)
			}
		})
	}
}

func TestCalculatePOExpenseTotalLimit(t *testing.T) {
	tests := []struct {
		name           string
		poTotal        float64
		cfg            POExpenseExcessConfig
		wantTotalLimit float64
		wantExcessText string
	}{
		{
			name:    "lesser_of: percent is lesser (small PO)",
			poTotal: 500.0,
			cfg:     POExpenseExcessConfig{Percent: 0.05, Value: 100.0, Mode: "lesser_of"},
			// 5% of 500 = 25, which is < 100, so percent applies
			wantTotalLimit: 525.0,
			wantExcessText: "5.00%",
		},
		{
			name:    "lesser_of: value is lesser (large PO)",
			poTotal: 5000.0,
			cfg:     POExpenseExcessConfig{Percent: 0.05, Value: 100.0, Mode: "lesser_of"},
			// 5% of 5000 = 250, which is > 100, so value applies
			wantTotalLimit: 5100.0,
			wantExcessText: "$100.00",
		},
		{
			name:    "lesser_of: equal at breakpoint",
			poTotal: 2000.0,
			cfg:     POExpenseExcessConfig{Percent: 0.05, Value: 100.0, Mode: "lesser_of"},
			// 5% of 2000 = 100, equals value, so percent branch applies (value < percent is false)
			wantTotalLimit: 2100.0,
			wantExcessText: "5.00%",
		},
		{
			name:    "greater_of: value is greater (small PO)",
			poTotal: 500.0,
			cfg:     POExpenseExcessConfig{Percent: 0.05, Value: 100.0, Mode: "greater_of"},
			// 5% of 500 = 25, value 100 >= 25, so value applies
			wantTotalLimit: 600.0,
			wantExcessText: "$100.00",
		},
		{
			name:    "greater_of: percent is greater (large PO)",
			poTotal: 5000.0,
			cfg:     POExpenseExcessConfig{Percent: 0.05, Value: 100.0, Mode: "greater_of"},
			// 5% of 5000 = 250, value 100 < 250, so percent applies
			wantTotalLimit: 5250.0,
			wantExcessText: "5.00%",
		},
		{
			name:    "zero PO total",
			poTotal: 0,
			cfg:     POExpenseExcessConfig{Percent: 0.05, Value: 100.0, Mode: "lesser_of"},
			// 5% of 0 = 0, which is < 100, so percent applies; total = 0
			wantTotalLimit: 0,
			wantExcessText: "5.00%",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := CalculatePOExpenseTotalLimit(tc.poTotal, tc.cfg)
			if math.Abs(got.TotalLimit-tc.wantTotalLimit) > 0.001 {
				t.Errorf("TotalLimit = %v, want %v", got.TotalLimit, tc.wantTotalLimit)
			}
			if got.ExcessText != tc.wantExcessText {
				t.Errorf("ExcessText = %q, want %q", got.ExcessText, tc.wantExcessText)
			}
		})
	}
}
