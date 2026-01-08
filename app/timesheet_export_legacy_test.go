package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestTimesheetExportLegacyAuth(t *testing.T) {
	// The test database has a machine_secrets record with:
	// id: legacy_time_writeback
	// salt: testsalt
	// secret: test-secret-123
	// sha256_hash: SHA256("testsalt" + "test-secret-123")
	validToken := "test-secret-123"

	// User with report claim (fatt@mac.com has report claim in test db)
	reportClaimToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// User without report claim (time@test.com does not have report claim)
	noReportClaimToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing Authorization header returns 401",
			Method:         http.MethodGet,
			URL:            "/api/time_sheets/2024-06-29/export_legacy",
			Headers:        map[string]string{},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid Authorization header format returns 401",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/2024-06-29/export_legacy",
			Headers: map[string]string{
				"Authorization": "Basic sometoken",
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "wrong Bearer token returns 401",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/2024-06-29/export_legacy",
			Headers: map[string]string{
				"Authorization": "Bearer wrong-token",
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "valid Bearer token returns 200",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/2024-06-29/export_legacy",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[", // response is a JSON array
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with report claim returns 200",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/2024-06-29/export_legacy",
			Headers: map[string]string{
				"Authorization": reportClaimToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[", // response is a JSON array
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without report claim returns 401",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/2024-06-29/export_legacy",
			Headers: map[string]string{
				"Authorization": noReportClaimToken,
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
