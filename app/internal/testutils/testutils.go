package testutils

import (
	"encoding/json"
	"testing"
	"tybalt/hooks"
	"tybalt/routes"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const testDataDir = "./test_pb_data"

// setup the test ApiScenario app instance
func SetupTestApp(t testing.TB) *tests.TestApp {
	testApp, err := tests.NewTestApp(testDataDir)
	if err != nil {
		t.Fatal(err)
	}
	// no need to cleanup since scenario.Test() will do that for us
	// defer testApp.Cleanup()

	// Add the hooks to the test app
	hooks.AddHooks(testApp)

	// Add the routes to the test app
	routes.AddRoutes(testApp)

	enableAllNotificationTemplates(t, testApp)

	return testApp
}

func enableAllNotificationTemplates(t testing.TB, app *tests.TestApp) {
	t.Helper()

	type templateRow struct {
		Code string `db:"code"`
	}
	rows := []templateRow{}
	if err := app.DB().NewQuery(`
		SELECT code
		FROM notification_templates
		WHERE code != ''
	`).All(&rows); err != nil {
		t.Fatalf("failed loading notification templates for test config: %v", err)
	}

	enabledTemplates := map[string]bool{}
	for _, row := range rows {
		enabledTemplates[row.Code] = true
	}

	rawValue, err := json.Marshal(enabledTemplates)
	if err != nil {
		t.Fatalf("failed marshaling notifications test config: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed finding app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "notifications")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "notifications")
	}
	record.Set("value", string(rawValue))
	if err := app.Save(record); err != nil {
		t.Fatalf("failed saving notifications test config: %v", err)
	}
}

func GenerateAdminToken(email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	admin, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
	if err != nil {
		return "", err
	}

	return admin.NewAuthToken() // Is this an admin token? I think so because it's a super user
}

func GenerateRecordToken(collectionNameOrId string, email string) (string, error) {
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	record, err := app.FindAuthRecordByEmail(collectionNameOrId, email)
	if err != nil {
		return "", err
	}

	return record.NewAuthToken()
}

// GetApprovalTiers retrieves approval tier values from the database
// Uses a single query and caches results for performance
var (
	cachedTier1, cachedTier2 float64
	tiersInitialized         bool
)

func GetApprovalTiers(app *tests.TestApp) (float64, float64) {
	// Return cached values if already initialized
	if tiersInitialized {
		return cachedTier1, cachedTier2
	}

	tier1Result := struct {
		Threshold float64 `db:"threshold"`
	}{}
	if err := app.DB().NewQuery(`
		SELECT COALESCE(second_approval_threshold, 0) AS threshold
		FROM expenditure_kinds
		WHERE name = 'standard'
		LIMIT 1
	`).One(&tier1Result); err != nil {
		panic("Failed to retrieve standard approval threshold: " + err.Error())
	}

	tier1 := tier1Result.Threshold
	if tier1 <= 0 {
		tier1 = 500
	}

	tier2Result := struct {
		Threshold float64 `db:"threshold"`
	}{}
	if err := app.DB().NewQuery(`
		SELECT MIN(max_amount) AS threshold
		FROM po_approver_props
		WHERE max_amount > {:tier1}
	`).Bind(map[string]any{
		"tier1": tier1,
	}).One(&tier2Result); err != nil {
		panic("Failed to retrieve second approval tier: " + err.Error())
	}

	tier2 := tier2Result.Threshold
	if tier2 <= tier1 {
		tier2 = 2500
	}

	// Cache the values
	cachedTier1, cachedTier2 = tier1, tier2
	tiersInitialized = true

	return tier1, tier2
}
