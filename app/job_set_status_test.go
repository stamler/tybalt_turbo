package main

import (
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// =============================================================================
// Job Set-Status Endpoint Tests
// =============================================================================
//
// These tests cover POST /api/jobs/{id}/set-status which atomically creates a
// client_note and updates the proposal status in a single transaction.
//
// Test data:
//   - test_prop_inprog (P24-0801) is a proposal with status "In Progress",
//     client "ee3xvodl583b61o"
//   - test_prop_cancelled (P24-0802) is a proposal with status "Cancelled"
//   - test_job_w_rs (98-8001) is a project (not a proposal)
//   - author@soup.com has the "job" claim
//   - noclaims@example.com has no job claim and is not a manager

func TestSetJobStatus_Success(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "cancel proposal with comment succeeds",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "Client withdrew the request"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Cancelled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "no bid proposal with comment succeeds",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "No Bid",
				"comment": "Not competitive enough"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"No Bid"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSetJobStatus_Validation(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "empty comment rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": ""
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"comment_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "whitespace-only comment rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "   "
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"comment_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid status rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "Active",
				"comment": "Some comment"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "already cancelled proposal rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_cancelled/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "Trying again"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"already_cancelled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "project rejected (not a proposal)",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_job_w_rs/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "Some comment"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"not_a_proposal"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "nonexistent job returns 404",
			Method: http.MethodPost,
			URL:    "/api/jobs/nonexistent_id/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "Some comment"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"code":"job_not_found"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestSetJobStatus_Authorization(t *testing.T) {
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "unauthenticated request rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "Some comment"
			}`),
			ExpectedStatus: 401,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without job claim rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-status",
			Body: strings.NewReader(`{
				"status": "Cancelled",
				"comment": "Some comment"
			}`),
			Headers: map[string]string{
				"Authorization": noClaimsToken,
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
