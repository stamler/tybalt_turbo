package main

import (
	"net/http"
	"strings"
	"testing"
	"time"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// Permission validation is tested through both positive and negative
// test cases below. Users without proper claims are blocked at the
// collection-level updateRule, resulting in 404 responses.
//
// The successful update test also verifies that outstanding_balance_date
// is correctly set to today's date when the balance changes.

func TestOutstandingBalance_UpdateWithJobClaim_Succeeds(t *testing.T) {
	// Use a user with 'job' claim: author@soup.com
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Calculate expected date: when outstanding_balance changes, it should be set to today
	// The hook uses time.Now().Format("2006-01-02") which gives YYYY-MM-DD format
	today := time.Now().Format("2006-01-02")

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating outstanding balance succeeds with job claim",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"outstanding_balance": 2500.50
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"outstanding_balance":2500.5`,
				`"outstanding_balance_date":"` + today + `"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestOutstandingBalance_UpdateWithoutJobClaim_Fails(t *testing.T) {
	// Use a user without 'job' or 'payables_admin' claim: noclaims@example.com
	// Users without job claims are blocked at the collection level by updateRule,
	// so they get a 404 instead of reaching the hook-level permission check
	recordToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating outstanding balance fails without job claim",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"outstanding_balance": 2500.50
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// Note: Payables admin functionality would be tested similarly,
// but we're focusing on the core job claim functionality for now.

// Note: Date preservation logic is tested through the hook functions
// in the jobs hooks tests. The API tests focus on the core update functionality.
