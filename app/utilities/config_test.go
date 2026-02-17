package utilities

import (
	"testing"

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
