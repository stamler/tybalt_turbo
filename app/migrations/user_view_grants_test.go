package migrations_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	_ "tybalt/migrations"
)

const userViewGrantsMigration = "1791926703_user_view_grants.go"

// The canonical users and expenses rows also hold auth secrets and fields that
// differ from a fresh migration run, so only the users viewRule and the
// expenses indexes are compared. TestProjectMigrationsMatchCanonicalSchema
// calls this after all migrations.
func assertCanonicalUserViewGrants(t *testing.T, app core.App) {
	t.Helper()
	assertCanonicalCollections(t, app, "user_view_grants")
	users := migrationCollection(t, app, "users")
	if want := fixtureRow(t, "data/_collections.csv", users.Id)["viewRule"]; *users.ViewRule != want {
		t.Errorf("users viewRule differs: fixture=%q migration=%q", want, *users.ViewRule)
	}
	expenses := migrationCollection(t, app, "expenses")
	var want []string
	if err := json.Unmarshal([]byte(fixtureRow(t, "data/_collections.csv", expenses.Id)["indexes"]), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual([]string(expenses.Indexes), want) {
		t.Errorf("expenses indexes differ: fixture=%v migration=%v", want, expenses.Indexes)
	}
}

// Undoing the migration restores exactly the users rule and expenses indexes
// it replaced and drops the view; redoing it restores the migrated state.
func TestUserViewGrantsMigrationRoundTrip(t *testing.T) {
	app, migration := newAppBeforeMigration(t, userViewGrantsMigration)
	type state struct {
		viewRule string
		indexes  []string
	}
	current := func() state {
		users := migrationCollection(t, app, "users")
		expenses := migrationCollection(t, app, "expenses")
		return state{*users.ViewRule, []string(expenses.Indexes)}
	}
	before := current()

	applyMigration(t, app, migration)
	after := current()
	if reflect.DeepEqual(after, before) {
		t.Fatal("the migration must change the users rule and expenses indexes")
	}
	if !app.HasTable("user_view_grants") {
		t.Fatal("the migration must create the user_view_grants view")
	}

	if err := migration.Down(app); err != nil {
		t.Fatal(err)
	}
	if got := current(); !reflect.DeepEqual(got, before) {
		t.Fatalf("down migration did not restore the previous state:\nwant %+v\ngot  %+v", before, got)
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
	if got := current(); !reflect.DeepEqual(got, after) {
		t.Fatalf("redoing the migration did not restore the migrated state:\nwant %+v\ngot  %+v", after, got)
	}
}
