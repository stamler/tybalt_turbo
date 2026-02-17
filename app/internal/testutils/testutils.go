package testutils

import (
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

	return testApp
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

	thresholds := []struct {
		Threshold float64 `db:"threshold"`
	}{}
	if err := app.DB().NewQuery(`
		SELECT DISTINCT COALESCE(second_approval_threshold, 0) AS threshold
		FROM expenditure_kinds
		WHERE COALESCE(second_approval_threshold, 0) > 0
		ORDER BY threshold ASC
	`).All(&thresholds); err != nil {
		panic("Failed to retrieve approval tiers from database: " + err.Error())
	}

	// Keep existing tier assumptions in legacy tests if fewer than 2 configured thresholds exist.
	if len(thresholds) < 2 {
		cachedTier1, cachedTier2 = 500, 2500
		tiersInitialized = true
		return cachedTier1, cachedTier2
	}

	tier1 := thresholds[0].Threshold
	tier2 := thresholds[1].Threshold

	// Cache the values
	cachedTier1, cachedTier2 = tier1, tier2
	tiersInitialized = true

	return tier1, tier2
}
