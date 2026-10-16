package migrations_test

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const purchaseOrdersBackfillSystemClosureMigration = "1792171240_purchase_orders_backfill_system_closure.go"

func TestPurchaseOrdersBackfillSystemClosure(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}

	type poRow struct {
		id, status, closed, closer string
		closedBySystem, imported   int
	}
	for _, po := range []poRow{
		{id: "autoclosed00001", status: "Closed"},
		{id: "nocommits000001", status: "Closed"},
		{id: "importedpo00001", status: "Closed", closedBySystem: 1, imported: 1},
		{id: "manualclose0001", status: "Closed", closed: "2025-03-01 10:00:00.000Z", closer: "closeruser00001"},
		{id: "activepo0000001", status: "Active"},
	} {
		if _, err := app.DB().NewQuery(`
			INSERT INTO purchase_orders (id, status, closed, closer, closed_by_system, _imported)
			VALUES ({:id}, {:status}, {:closed}, {:closer}, {:closed_by_system}, {:imported})
		`).Bind(dbx.Params{
			"id":               po.id,
			"status":           po.status,
			"closed":           po.closed,
			"closer":           po.closer,
			"closed_by_system": po.closedBySystem,
			"imported":         po.imported,
		}).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	for _, expense := range []struct{ id, po, committed string }{
		{"autoexpense0001", "autoclosed00001", "2025-02-01 09:00:00.000Z"},
		{"autoexpense0002", "autoclosed00001", "2025-02-08 09:00:00.123Z"},
		{"autoexpense0003", "autoclosed00001", ""},
		{"nocommitexp0001", "nocommits000001", ""},
		{"importedexp0001", "importedpo00001", "2024-01-01 09:00:00.000Z"},
		{"manualexpense01", "manualclose0001", "2025-02-15 09:00:00.000Z"},
		{"activeexpense01", "activepo0000001", "2025-02-20 09:00:00.000Z"},
	} {
		if _, err := app.DB().NewQuery(`
			INSERT INTO expenses (id, purchase_order, committed)
			VALUES ({:id}, {:po}, {:committed})
		`).Bind(dbx.Params{"id": expense.id, "po": expense.po, "committed": expense.committed}).Execute(); err != nil {
			t.Fatal(err)
		}
	}

	var migration *core.Migration
	for _, item := range core.AppMigrations.Items() {
		if item.File == purchaseOrdersBackfillSystemClosureMigration {
			migration = item
		}
	}
	if migration == nil {
		t.Fatal("purchase order system closure backfill migration must be registered")
	}
	if err := migration.Up(app); err != nil {
		t.Fatal(err)
	}

	for _, want := range []poRow{
		// The latest committed expense is the closure time; uncommitted rows are ignored.
		{id: "autoclosed00001", status: "Closed", closed: "2025-02-08 09:00:00.123Z", closedBySystem: 1},
		// No committed expense means no closure time, so the row is left alone.
		{id: "nocommits000001", status: "Closed"},
		{id: "importedpo00001", status: "Closed", closedBySystem: 1, imported: 1},
		{id: "manualclose0001", status: "Closed", closed: "2025-03-01 10:00:00.000Z", closer: "closeruser00001"},
		{id: "activepo0000001", status: "Active"},
	} {
		var got struct {
			Status         string `db:"status"`
			Closed         string `db:"closed"`
			Closer         string `db:"closer"`
			ClosedBySystem int    `db:"closed_by_system"`
		}
		if err := app.DB().NewQuery(`
			SELECT status, closed, closer, closed_by_system FROM purchase_orders WHERE id = {:id}
		`).Bind(dbx.Params{"id": want.id}).One(&got); err != nil {
			t.Fatal(err)
		}
		if got.Status != want.status || got.Closed != want.closed || got.Closer != want.closer || got.ClosedBySystem != want.closedBySystem {
			t.Fatalf("%s: got status=%q closed=%q closer=%q closed_by_system=%d; want %q %q %q %d",
				want.id, got.Status, got.Closed, got.Closer, got.ClosedBySystem,
				want.status, want.closed, want.closer, want.closedBySystem)
		}
	}
}
