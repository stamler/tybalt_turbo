package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// TestCreateRateSheet_Success tests successful creation of a rate sheet with entries.
// Uses a user with 'job' claim (author@soup.com) to create a new rate sheet (revision 0).
func TestCreateRateSheet_Success(t *testing.T) {
	// Token for user with 'job' claim
	jobToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get role IDs from test database
	app := testutils.SetupTestApp(t)
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 3, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}
	if len(roles) < 2 {
		t.Fatalf("need at least 2 roles for test, got %d", len(roles))
	}

	requestBody := `{
		"name": "Test New Rate Sheet",
		"effective_date": "2026-03-01",
		"revision": 0,
		"entries": [
			{"role": "` + roles[0].Id + `", "rate": 100, "overtime_rate": 130.5},
			{"role": "` + roles[1].Id + `", "rate": 120, "overtime_rate": 156}
		]
	}`

	scenarios := []tests.ApiScenario{
		{
			Name:   "creates rate sheet with entries atomically",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body:   strings.NewReader(requestBody),
			Headers: map[string]string{
				"Authorization": jobToken,
			},
			ExpectedStatus: http.StatusCreated,
			ExpectedContent: []string{
				`"name":"Test New Rate Sheet"`,
				`"revision":0`,
				`"entries_created":2`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate": 3, // 1 rate_sheet + 2 entries
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestCreateRateSheet_DuplicateName tests that creating a rate sheet with duplicate
// name and revision returns a conflict error.
func TestCreateRateSheet_DuplicateName(t *testing.T) {
	// Token for user with 'job' claim
	jobToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get role IDs from test database
	app := testutils.SetupTestApp(t)
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 2, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	requestBody := `{
		"name": "Duplicate Test Sheet",
		"effective_date": "2026-03-01",
		"revision": 0,
		"entries": [
			{"role": "` + roles[0].Id + `", "rate": 100, "overtime_rate": 130}
		]
	}`

	// Use BeforeTestFunc to create the first rate sheet, then test that a duplicate fails
	// The key behavior is that the transaction is rolled back and no partial data is created
	scenario := tests.ApiScenario{
		Name:   "duplicate name returns error",
		Method: http.MethodPost,
		URL:    "/api/rate_sheets",
		Body:   strings.NewReader(requestBody),
		Headers: map[string]string{
			"Authorization": jobToken,
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			// Create the first rate sheet so the duplicate will fail
			collection, err := app.FindCollectionByNameOrId("rate_sheets")
			if err != nil {
				t.Fatalf("failed to get collection: %v", err)
			}
			record := core.NewRecord(collection)
			record.Set("name", "Duplicate Test Sheet")
			record.Set("effective_date", "2026-03-01")
			record.Set("revision", 0)
			record.Set("active", false)
			if err := app.Save(record); err != nil {
				t.Fatalf("failed to create first rate sheet: %v", err)
			}
		},
		ExpectedStatus:  http.StatusConflict,
		ExpectedContent: []string{`"status":409`, `validation_not_unique`},
		TestAppFactory:  testutils.SetupTestApp,
	}
	scenario.Test(t)
}

// TestCreateRateSheet_PermissionDenied tests that users without claims cannot create rate sheets.
func TestCreateRateSheet_PermissionDenied(t *testing.T) {
	// Token for user without claims
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get role IDs from test database
	app := testutils.SetupTestApp(t)
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 1, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	requestBody := `{
		"name": "Unauthorized Rate Sheet",
		"effective_date": "2026-03-01",
		"revision": 0,
		"entries": [
			{"role": "` + roles[0].Id + `", "rate": 100, "overtime_rate": 130}
		]
	}`

	scenarios := []tests.ApiScenario{
		{
			Name:   "user without job claim cannot create rate sheet",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body:   strings.NewReader(requestBody),
			Headers: map[string]string{
				"Authorization": noClaimsToken,
			},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`You do not have permission to create rate sheets`},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestCreateRateSheet_RevisionSucceedsWithClaim tests that creating a revision (revision > 0)
// succeeds with the rate_sheet_revise claim.
func TestCreateRateSheet_RevisionSucceedsWithClaim(t *testing.T) {
	// Token for user with rate_sheet_revise claim
	reviseToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get role IDs from test database
	app := testutils.SetupTestApp(t)
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 1, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	rev1Body := `{
		"name": "Revise Test Sheet",
		"effective_date": "2026-02-01",
		"revision": 1,
		"entries": [
			{"role": "` + roles[0].Id + `", "rate": 110, "overtime_rate": 143}
		]
	}`

	scenario := tests.ApiScenario{
		Name:   "revision 1 succeeds with rate_sheet_revise claim",
		Method: http.MethodPost,
		URL:    "/api/rate_sheets",
		Body:   strings.NewReader(rev1Body),
		Headers: map[string]string{
			"Authorization": reviseToken,
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			// Create revision 0 first
			collection, err := app.FindCollectionByNameOrId("rate_sheets")
			if err != nil {
				t.Fatalf("failed to get collection: %v", err)
			}
			record := core.NewRecord(collection)
			record.Set("name", "Revise Test Sheet")
			record.Set("effective_date", "2026-01-01")
			record.Set("revision", 0)
			record.Set("active", false)
			if err := app.Save(record); err != nil {
				t.Fatalf("failed to create revision 0: %v", err)
			}
		},
		ExpectedStatus: http.StatusCreated,
		ExpectedContent: []string{
			`"name":"Revise Test Sheet"`,
			`"revision":1`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}
	scenario.Test(t)
}

// TestCreateRateSheet_RevisionFailsWithoutClaim tests that creating a revision (revision > 0)
// fails for users without rate_sheet_revise or admin claim.
func TestCreateRateSheet_RevisionFailsWithoutClaim(t *testing.T) {
	// Token for user without rate_sheet_revise or admin claims
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get role IDs from test database
	app := testutils.SetupTestApp(t)
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 1, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	rev1Body := `{
		"name": "Revise Test Sheet",
		"effective_date": "2026-02-01",
		"revision": 1,
		"entries": [
			{"role": "` + roles[0].Id + `", "rate": 110, "overtime_rate": 143}
		]
	}`

	scenario := tests.ApiScenario{
		Name:   "revision 1 fails without rate_sheet_revise claim",
		Method: http.MethodPost,
		URL:    "/api/rate_sheets",
		Body:   strings.NewReader(rev1Body),
		Headers: map[string]string{
			"Authorization": noClaimsToken,
		},
		BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			// Create revision 0 first
			collection, err := app.FindCollectionByNameOrId("rate_sheets")
			if err != nil {
				t.Fatalf("failed to get collection: %v", err)
			}
			record := core.NewRecord(collection)
			record.Set("name", "Revise Test Sheet")
			record.Set("effective_date", "2026-01-01")
			record.Set("revision", 0)
			record.Set("active", false)
			if err := app.Save(record); err != nil {
				t.Fatalf("failed to create revision 0: %v", err)
			}
		},
		ExpectedStatus:  http.StatusForbidden,
		ExpectedContent: []string{`You do not have permission to revise rate sheets`},
		TestAppFactory:  testutils.SetupTestApp,
	}
	scenario.Test(t)
}

// TestCreateRateSheet_Validation tests various validation error cases.
func TestCreateRateSheet_Validation(t *testing.T) {
	// Token for user with 'job' claim
	jobToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get role IDs from test database
	app := testutils.SetupTestApp(t)
	roles, err := app.FindRecordsByFilter("rate_roles", "1=1", "", 1, 0, nil)
	if err != nil {
		t.Fatalf("failed to get roles: %v", err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "empty name fails",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body: strings.NewReader(`{
				"name": "",
				"effective_date": "2026-03-01",
				"revision": 0,
				"entries": [{"role": "` + roles[0].Id + `", "rate": 100, "overtime_rate": 130}]
			}`),
			Headers: map[string]string{
				"Authorization": jobToken,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"status":400`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:   "invalid date format fails",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body: strings.NewReader(`{
				"name": "Test Sheet",
				"effective_date": "03-01-2026",
				"revision": 0,
				"entries": [{"role": "` + roles[0].Id + `", "rate": 100, "overtime_rate": 130}]
			}`),
			Headers: map[string]string{
				"Authorization": jobToken,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"status":400`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:   "empty entries fails",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body: strings.NewReader(`{
				"name": "Test Sheet",
				"effective_date": "2026-03-01",
				"revision": 0,
				"entries": []
			}`),
			Headers: map[string]string{
				"Authorization": jobToken,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"status":400`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:   "entry rate less than 1 fails",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body: strings.NewReader(`{
				"name": "Test Sheet",
				"effective_date": "2026-03-01",
				"revision": 0,
				"entries": [{"role": "` + roles[0].Id + `", "rate": 0, "overtime_rate": 130}]
			}`),
			Headers: map[string]string{
				"Authorization": jobToken,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"status":400`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:   "entry overtime_rate less than 1 fails",
			Method: http.MethodPost,
			URL:    "/api/rate_sheets",
			Body: strings.NewReader(`{
				"name": "Test Sheet",
				"effective_date": "2026-03-01",
				"revision": 0,
				"entries": [{"role": "` + roles[0].Id + `", "rate": 100, "overtime_rate": 0}]
			}`),
			Headers: map[string]string{
				"Authorization": jobToken,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"status":400`},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestCreateRateSheet_InvalidRoleRejected tests that an entry with a
// non-existent role ID is rejected during validation.
func TestCreateRateSheet_InvalidRoleRejected(t *testing.T) {
	// Token for user with 'job' claim
	jobToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	requestBody := `{
		"name": "Invalid Role Test Sheet",
		"effective_date": "2026-03-01",
		"revision": 0,
		"entries": [
			{"role": "invalid_role_id", "rate": 100, "overtime_rate": 130}
		]
	}`

	scenario := tests.ApiScenario{
		Name:   "invalid role ID is rejected",
		Method: http.MethodPost,
		URL:    "/api/rate_sheets",
		Body:   strings.NewReader(requestBody),
		Headers: map[string]string{
			"Authorization": jobToken,
		},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"status":400`},
		// No records should be created due to rollback
		ExpectedEvents: map[string]int{},
		TestAppFactory: testutils.SetupTestApp,
	}
	scenario.Test(t)
}
