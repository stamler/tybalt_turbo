package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestSetJobNumberEndpoint(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "admin can renumber a regular project",
			Method: http.MethodPost,
			URL:    "/api/jobs/fcprojimpnoprop1/set-number",
			Body:   strings.NewReader(`{"number":"88-9191"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"fcprojimpnoprop1"`,
				`"number":"88-9191"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				job, err := app.FindRecordById("jobs", "fcprojimpnoprop1")
				if err != nil {
					tb.Fatalf("failed to reload job: %v", err)
				}
				if got := job.GetString("number"); got != "88-9191" {
					tb.Fatalf("expected updated number 88-9191, got %q", got)
				}
				if job.GetBool("_imported") {
					tb.Fatal("expected renumbered job to be marked _imported=false")
				}
			},
		},
		{
			Name:   "admin can renumber a proposal",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-number",
			Body:   strings.NewReader(`{"number":"P24-0891"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"test_prop_inprog"`,
				`"number":"P24-0891"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "admin can renumber a child job",
			Method: http.MethodPost,
			URL:    "/api/jobs/testsubjob01id/set-number",
			Body:   strings.NewReader(`{"number":"24-334-09"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"testsubjob01id"`,
				`"number":"24-334-09"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "admin can renumber a cancelled proposal",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_cancelled/set-number",
			Body:   strings.NewReader(`{"number":"P24-0892"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"test_prop_cancelled"`,
				`"number":"P24-0892"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "non-admin is rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/u09fwwcg07y03m7/set-number",
			Body:   strings.NewReader(`{"number":"24-299"}`),
			Headers: map[string]string{
				"Authorization": noClaimsToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
				`"message":"you are not authorized to change job numbers"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "project cannot be renumbered to proposal form",
			Method: http.MethodPost,
			URL:    "/api/jobs/u09fwwcg07y03m7/set-number",
			Body:   strings.NewReader(`{"number":"P24-291"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"number":{"code":"project_number_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal cannot be renumbered to project form",
			Method: http.MethodPost,
			URL:    "/api/jobs/test_prop_inprog/set-number",
			Body:   strings.NewReader(`{"number":"24-0801"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"number":{"code":"proposal_number_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "child job cannot be renumbered to top-level form",
			Method: http.MethodPost,
			URL:    "/api/jobs/testsubjob01id/set-number",
			Body:   strings.NewReader(`{"number":"24-334"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"number":{"code":"child_number_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "top-level job cannot be renumbered to child form",
			Method: http.MethodPost,
			URL:    "/api/jobs/u09fwwcg07y03m7/set-number",
			Body:   strings.NewReader(`{"number":"24-291-01"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"number":{"code":"top_level_number_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid number format is rejected by schema validation",
			Method: http.MethodPost,
			URL:    "/api/jobs/u09fwwcg07y03m7/set-number",
			Body:   strings.NewReader(`{"number":"BAD-NUMBER"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"message":"validation failed"`,
				`"number":{`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "duplicate job number returns conflict on number field",
			Method: http.MethodPost,
			URL:    "/api/jobs/u09fwwcg07y03m7/set-number",
			Body:   strings.NewReader(`{"number":"24-321"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusConflict,
			ExpectedContent: []string{
				`"number":{"code":"validation_not_unique"`,
				`"message":"job number must be unique"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
