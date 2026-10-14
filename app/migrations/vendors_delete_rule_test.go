package migrations_test

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

const vendorsDeleteRuleMigration = "1791933889_vendors_delete_rule_vendor_refs.go"

// All migrations must produce the canonical vendors schema, whose deleteRule
// refuses to delete vendors that purchase orders or expenses reference.
func TestVendorsMigrationsMatchCanonicalSchema(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	assertCanonicalCollections(t, app, "vendors")
}

func TestVendorsDeleteRuleMigrationRevert(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.ResetBootstrapState() })
	if err := app.RunAllMigrations(); err != nil {
		t.Fatal(err)
	}
	var migration *core.Migration
	for _, item := range core.AppMigrations.Items() {
		if item.File == vendorsDeleteRuleMigration {
			migration = item
		}
	}
	if migration == nil {
		t.Fatal("vendors delete rule migration must be registered")
	}
	if err := migration.Down(app); err != nil {
		t.Fatal(err)
	}
	rule := migrationCollection(t, app, "vendors").DeleteRule
	if rule == nil || !strings.Contains(*rule, "@collection.purchase_orders.job != id") || !strings.Contains(*rule, "@collection.expenses.job != id") {
		t.Fatalf("down migration must restore the previous deleteRule; got %v", rule)
	}
	if err := migration.Up(app); err != nil {
		t.Fatal(err)
	}
	assertCanonicalCollections(t, app, "vendors")
}
