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
	cachedTier1, cachedTier2, cachedTier3 float64
	tiersInitialized                      bool
)

func GetApprovalTiers(app *tests.TestApp) (tier1 float64, tier2 float64, tier3 float64) {
	// Return cached values if already initialized
	if tiersInitialized {
		return cachedTier1, cachedTier2, cachedTier3
	}

	// Get all tiers in a single query
	records, err := app.FindRecordsByFilter(
		"po_approval_tiers",
		"", // No filter to get all tiers
		"", // No sort
		3,  // Max number of tiers
		0,  // No offset
		nil,
	)

	if err != nil || len(records) < 3 {
		panic("Failed to retrieve approval tiers from database: " + err.Error())
	}

	// Map claim names to their respective values
	tierValues := make(map[string]float64)
	for _, record := range records {
		claim := record.GetString("claim")
		if claim == "" {
			continue
		}

		// Get the claim record to access its name
		claimRecord, err := app.FindRecordById("claims", claim)
		if err != nil {
			continue
		}

		claimName := claimRecord.GetString("name")
		maxAmount, _ := record.Get("max_amount").(float64)
		if maxAmount == 0 {
			continue
		}

		tierValues[claimName] = maxAmount
	}

	// Check if all required tiers are present
	tier1, hasTier1 := tierValues["po_approver"]
	tier2, hasTier2 := tierValues["po_approver_tier2"]
	tier3, hasTier3 := tierValues["po_approver_tier3"]

	if !hasTier1 || !hasTier2 || !hasTier3 {
		panic("One or more required approval tiers missing from database")
	}

	// Cache the values
	cachedTier1, cachedTier2, cachedTier3 = tier1, tier2, tier3
	tiersInitialized = true

	return tier1, tier2, tier3
}
