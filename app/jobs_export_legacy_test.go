package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestJobsExportLegacyAuth(t *testing.T) {
	// The test database has a machine_secrets record with:
	// role: legacy_writeback
	// expiry: 2028-05-08 (unexpired)
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
			URL:            "/api/export_legacy/jobs/2000-01-01",
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
			URL:    "/api/export_legacy/jobs/2000-01-01",
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
			URL:    "/api/export_legacy/jobs/2000-01-01",
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
			URL:    "/api/export_legacy/jobs/2000-01-01",
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
			Name:   "valid Bearer token with trailing newline returns 200 (simulates secret manager)",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken + "\n",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[", // response is a JSON array
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "valid Bearer token with trailing spaces returns 200",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken + "  ",
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
			URL:    "/api/export_legacy/jobs/2000-01-01",
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
			URL:    "/api/export_legacy/jobs/2000-01-01",
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

func TestJobsExportLegacyCount(t *testing.T) {
	// The test database has 9 jobs with _imported = 0
	// 3 of those have updated >= 2026-01-01:
	//   - test4digit350id (24-0350)
	//   - testsubjob01id (24-334-01)
	//   - testorphan01id (24-350-01)
	validToken := "test-secret-123"

	scenarios := []tests.ApiScenario{
		{
			Name:   "returns all 9 non-imported jobs when updatedAfter is 2000-01-01",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				// Verify several expected jobs are present
				`"number":"P24-487"`,
				`"number":"24-291"`,
				`"number":"24-334"`,
				`"number":"24-326"`,
				`"number":"24-321"`,
				`"number":"P24-999"`,
				`"number":"24-0350"`,
				`"number":"24-334-01"`,
				`"number":"24-350-01"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "returns only jobs updated after 2026-01-01",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2026-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				// These 3 jobs were updated after 2026-01-01
				`"number":"24-0350"`,
				`"number":"24-334-01"`,
				`"number":"24-350-01"`,
			},
			NotExpectedContent: []string{
				// These jobs were NOT updated after 2026-01-01
				`"number":"P24-487"`,
				`"number":"24-291"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "returns 0 jobs when updatedAfter is in the future",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2099-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[]", // empty array
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestJobsExportLegacyClientNotes(t *testing.T) {
	// Test that client notes are included in the export.
	// The test database has client_notes for clients associated with non-imported jobs:
	//   - note_lb0: "Target client note" for client lb0fnenkeyitsny (Mobilia SA), job 24-321
	//   - note_eld: "Absorbed client note 1" for client eldtxi3i4h00k8r (BN Rail Corporation), job 24-326
	//   - note_pqp: "Absorbed client note 2" for client pqpd90fqd5ohjcs (Sepulchi & Co), job 24-291
	//   - note_nobid_test: "We decided not to bid on this project" for client lb0fnenkeyitsny, job P24-0804
	//   - note_cancel_test: "Client cancelled the project" for client lb0fnenkeyitsny, job P24-0805
	validToken := "test-secret-123"

	scenarios := []tests.ApiScenario{
		{
			Name:   "clients include notes array with full fidelity",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				// Verify notes are included in the response
				`"notes":[`,
				// Verify note content
				`"note":"Target client note"`,
				`"note":"Absorbed client note 1"`,
				`"note":"Absorbed client note 2"`,
				// Verify job references are included (job number)
				`"jobNumber":"24-321"`,
				`"jobNumber":"24-326"`,
				`"jobNumber":"24-291"`,
				// Verify note IDs are included
				`"id":"note_lb0"`,
				`"id":"note_eld"`,
				`"id":"note_pqp"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "notes include jobStatusChangedTo for status change notes",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/jobs/2000-01-01",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				// Notes that changed job status should include jobStatusChangedTo
				`"jobStatusChangedTo":"No Bid"`,
				`"jobStatusChangedTo":"Cancelled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
