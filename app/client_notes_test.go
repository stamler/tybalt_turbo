package main

import (
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestClientNotes_GetClientNotes_Success(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "get client notes returns notes for Mobilia SA client",
			Method: http.MethodGet,
			URL:    "/api/clients/lb0fnenkeyitsny/notes",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"note_lb0"`,
				`"note":"Target client note"`,
				`"job":{"id":"cjf0kt0defhq480","number":"24-321"`,
				`"author":{"id":`,
				`"email":"orphan@poapprover.com"`,
				`"given_name":"Orphaned"`,
				`"surname":"POApprover"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "get client notes returns notes for BN Rail Corporation client",
			Method: http.MethodGet,
			URL:    "/api/clients/eldtxi3i4h00k8r/notes",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"note_eld"`,
				`"note":"Absorbed client note 1"`,
				`"job":{"id":"zke3cs3yipplwtu","number":"24-326"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "get client notes returns empty array when no notes exist",
			Method: http.MethodGet,
			URL:    "/api/clients/someclientid123/notes",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`[]`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestClientNotes_GetJobNotes_Success(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "get job notes returns notes for job 24-321",
			Method: http.MethodGet,
			URL:    "/api/jobs/cjf0kt0defhq480/notes",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"note_lb0"`,
				`"note":"Target client note"`,
				`"job":null`,
				`"email":"orphan@poapprover.com"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "get job notes returns notes for job 24-326",
			Method: http.MethodGet,
			URL:    "/api/jobs/zke3cs3yipplwtu/notes",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"note_eld"`,
				`"note":"Absorbed client note 1"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "get job notes returns empty array when no notes exist",
			Method: http.MethodGet,
			URL:    "/api/jobs/somejobid123/notes",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`[]`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// Note: Missing ID validation happens in the route handlers, but since these are
// custom routes, the router doesn't match the pattern when no ID is provided.
// The core business logic validation is covered by the hook tests below.

func TestClientNotes_CreateNote_MissingClient_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "create client note fails when client is missing",
			Method: http.MethodPost,
			URL:    "/api/collections/client_notes/records",
			Body: strings.NewReader(`{
				"note": "Test note",
				"job_not_applicable": true
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"required"`,
				`"message":"client is required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestClientNotes_CreateNote_MissingJobAndNotApplicable_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "create client note fails when job is missing and not marked not applicable",
			Method: http.MethodPost,
			URL:    "/api/collections/client_notes/records",
			Body: strings.NewReader(`{
				"client": "someclientid123",
				"note": "Test note"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"job_or_flag_required"`,
				`"job is required unless marked not applicable"`,
				`"job must be selected or marked not applicable"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestClientNotes_CreateNote_JobNotFound_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "create client note fails when job does not exist",
			Method: http.MethodPost,
			URL:    "/api/collections/client_notes/records",
			Body: strings.NewReader(`{
				"client": "someclientid123",
				"job": "nonexistentjobid",
				"note": "Test note"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"not_found"`,
				`"message":"job not found"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// Note: Authentication is handled by PocketBase's collection-level auth rules,
// so we don't need to test it here. The hook validation tests above cover
// the business logic validation that happens after auth.
