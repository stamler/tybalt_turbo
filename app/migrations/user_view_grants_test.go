package migrations_test

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	_ "tybalt/migrations"
)

const userViewGrantsMigration = "1791926703_user_view_grants.go"

// The canonical users row also holds auth secrets and fields that differ from
// a fresh migration run, so only its viewRule is compared.
func assertCanonicalUserViewGrants(t *testing.T, app core.App) {
	t.Helper()
	assertCanonicalCollections(t, app, "user_view_grants")
	users := migrationCollection(t, app, "users")
	if want := fixtureRow(t, "data/_collections.csv", users.Id)["viewRule"]; *users.ViewRule != want {
		t.Errorf("users viewRule differs: fixture=%q migration=%q", want, *users.ViewRule)
	}
}

func TestUserViewGrantsMatchCanonicalSchema(t *testing.T) {
	assertCanonicalUserViewGrants(t, newMigratedApp(t))
}

// Undoing the migration restores the @collection rule and drops the view.
func TestUserViewGrantsMigrationRoundTrip(t *testing.T) {
	app := newMigratedApp(t)
	var migration *core.Migration
	for _, item := range core.AppMigrations.Items() {
		if item.File == userViewGrantsMigration {
			migration = item
		}
	}
	if migration == nil {
		t.Fatal("user view grants migration is not registered")
	}

	if err := migration.Down(app); err != nil {
		t.Fatal(err)
	}
	users := migrationCollection(t, app, "users")
	if !strings.Contains(*users.ViewRule, "@collection.purchase_orders.status") {
		t.Fatalf("down migration did not restore the previous rule: %s", *users.ViewRule)
	}
	if _, err := app.FindCollectionByNameOrId("user_view_grants"); err == nil {
		t.Fatal("down migration did not remove user_view_grants")
	}
	if app.HasTable("user_view_grants") {
		t.Fatal("down migration did not drop the user_view_grants view")
	}

	if err := migration.Up(app); err != nil {
		t.Fatal(err)
	}
	assertCanonicalUserViewGrants(t, app)
}
