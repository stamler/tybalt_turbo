package main

import (
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// =============================================================================
// Close Project Endpoint Tests
// =============================================================================
//
// These tests intentionally rely on static fixture rows in app/test_pb_data/data.db.
// They do not mutate fixture state for setup during test execution.

func TestCloseJob_ImportedProjectClosesAndCreatesProjectNote(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpnoprop1"
	const projectNoteText = "Project closed with the Close project action"

	scenario := tests.ApiScenario{
		Name:   "imported project closes and writes project note",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"` + projectID + `"`,
			`"status":"Closed"`,
			`"project_note_created":true`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			project, err := app.FindRecordById("jobs", projectID)
			if err != nil {
				tb.Fatalf("failed to reload closed project: %v", err)
			}
			if got := project.GetString("status"); got != "Closed" {
				tb.Fatalf("expected project status Closed, got %q", got)
			}
			if project.GetBool("_imported") {
				tb.Fatal("expected closed project _imported=false")
			}

			var result struct {
				Count int64 `db:"count"`
			}
			if err := app.DB().NewQuery(`
				SELECT COUNT(*) AS count
				FROM client_notes
				WHERE job = {:job} AND note = {:note}
			`).Bind(dbx.Params{
				"job":  projectID,
				"note": projectNoteText,
			}).One(&result); err != nil {
				tb.Fatalf("failed to count project notes after request: %v", err)
			}
			if result.Count != 1 {
				tb.Fatalf("expected exactly one project close note, got %d", result.Count)
			}
		},
	}

	scenario.Test(t)
}

func TestCloseJob_ImportedAutoAwardCreatesProposalNoteAndCloses(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpauto01"
	const proposalID = "fcpropimpinprog"
	const projectNumber = "88-9102"
	const projectNoteText = "Project closed with the Close project action"
	const proposalNoteText = "Proposal auto-awarded when project " + projectNumber + " (" + projectID + ") was closed"

	scenario := tests.ApiScenario{
		Name:   "imported project close auto-awards imported proposal and writes both notes",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"` + projectID + `"`,
			`"status":"Closed"`,
			`"project_note_created":true`,
			`"auto_awarded":true`,
			`"proposal_note_created":true`,
			`"to_status":"Awarded"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			project, err := app.FindRecordById("jobs", projectID)
			if err != nil {
				tb.Fatalf("failed to reload project: %v", err)
			}
			if got := project.GetString("status"); got != "Closed" {
				tb.Fatalf("expected project status Closed, got %q", got)
			}
			if project.GetBool("_imported") {
				tb.Fatal("expected project _imported=false after close")
			}

			proposal, err := app.FindRecordById("jobs", proposalID)
			if err != nil {
				tb.Fatalf("failed to reload proposal: %v", err)
			}
			if got := proposal.GetString("status"); got != "Awarded" {
				tb.Fatalf("expected proposal status Awarded, got %q", got)
			}
			if proposal.GetBool("_imported") {
				tb.Fatal("expected proposal _imported=false after auto-award")
			}

			var projectNoteResult struct {
				Count int64 `db:"count"`
			}
			if err := app.DB().NewQuery(`
				SELECT COUNT(*) AS count
				FROM client_notes
				WHERE job = {:job} AND note = {:note}
			`).Bind(dbx.Params{
				"job":  projectID,
				"note": projectNoteText,
			}).One(&projectNoteResult); err != nil {
				tb.Fatalf("failed to count project notes after request: %v", err)
			}
			if projectNoteResult.Count != 1 {
				tb.Fatalf("expected exactly one project close note, got %d", projectNoteResult.Count)
			}

			var proposalNoteResult struct {
				Count int64 `db:"count"`
			}
			if err := app.DB().NewQuery(`
				SELECT COUNT(*) AS count
				FROM client_notes
				WHERE job = {:job} AND note = {:note}
			`).Bind(dbx.Params{
				"job":  proposalID,
				"note": proposalNoteText,
			}).One(&proposalNoteResult); err != nil {
				tb.Fatalf("failed to count proposal notes after request: %v", err)
			}
			if proposalNoteResult.Count != 1 {
				tb.Fatalf("expected exactly one proposal auto-award note, got %d", proposalNoteResult.Count)
			}
		},
	}

	scenario.Test(t)
}

// A project that was not imported closes without its billing data. The
// fixture has no authorizing document, project value, location or invoicing
// profile, so it would fail full validation.
func TestCloseJob_IncompleteProjectCloses(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "test_job_w_rs"
	const projectNoteText = "Project closed with the Close project action"

	scenario := tests.ApiScenario{
		Name:   "non-imported incomplete project closes and writes project note",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"` + projectID + `"`,
			`"status":"Closed"`,
			`"imported":false`,
			`"project_note_created":true`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			project, err := app.FindRecordById("jobs", projectID)
			if err != nil {
				tb.Fatalf("failed to reload project: %v", err)
			}
			if got := project.GetString("status"); got != "Closed" {
				tb.Fatalf("expected project status Closed, got %q", got)
			}
			if got := project.GetString("authorizing_document"); got != "" {
				tb.Fatalf("close must not fill the authorizing document, got %q", got)
			}

			var result struct {
				Count int64 `db:"count"`
			}
			if err := app.DB().NewQuery(`
				SELECT COUNT(*) AS count
				FROM client_notes
				WHERE job = {:job} AND note = {:note}
			`).Bind(dbx.Params{
				"job":  projectID,
				"note": projectNoteText,
			}).One(&result); err != nil {
				tb.Fatalf("failed to count project notes after request: %v", err)
			}
			if result.Count != 1 {
				tb.Fatalf("expected exactly one project close note, got %d", result.Count)
			}
		},
	}

	scenario.Test(t)
}

// The editor's status-only close uses the same retirement rule as the action.
func TestCloseJob_EditorStatusOnlyCloseMatchesAction(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "test_job_w_rs"

	scenarios := []tests.ApiScenario{
		{
			Name:   "job editor closes incomplete project",
			Method: http.MethodPut,
			URL:    "/api/jobs/" + projectID,
			Body:   strings.NewReader(`{"job":{"status":"Closed"},"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"id":"` + projectID + `"`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:   "collection API closes incomplete project",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/" + projectID,
			Body:   strings.NewReader(`{"status":"Closed"}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{`"status":"Closed"`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			// Retirement relaxes only the status change. Any other edit still
			// needs the full billing data.
			Name:   "collection API close with another edit stays strict",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/" + projectID,
			Body:   strings.NewReader(`{"status":"Closed","description":"Edited"}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"location":{"code":"invalid_or_missing"`},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCloseJob_ProposalTerminalStatusBlocksClose(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpterm01"

	scenario := tests.ApiScenario{
		Name:   "terminal proposal status blocks close",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"code":"proposal_terminal_status_blocks_close"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestCloseJob_Authorization(t *testing.T) {
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	managerToken, err := testutils.GenerateRecordToken("users", allocationManagerEmail)
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpnoprop1"

	scenarios := []tests.ApiScenario{
		{
			Name:           "unauthenticated close rejected",
			Method:         http.MethodPost,
			URL:            "/api/jobs/" + projectID + "/close",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without job claim and not manager rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/" + projectID + "/close",
			Headers: map[string]string{
				"Authorization": noClaimsToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			// Managers without the job claim may change allocations only.
			Name:   "manager without job claim rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/" + projectID + "/close",
			Headers: map[string]string{
				"Authorization": managerToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
				// Assign the fixture user without the job claim as manager.
				if _, err := app.DB().NewQuery("UPDATE jobs SET manager = {:uid} WHERE id = {:id}").Bind(dbx.Params{"uid": allocationManagerID, "id": projectID}).Execute(); err != nil {
					tb.Fatal(err)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCloseJob_RejectsNonProjectAndNonActive(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "proposal target is rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/fcpropimpinprog/close",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"not_a_project"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "non-active project is rejected",
			Method: http.MethodPost,
			URL:    "/api/jobs/zke3cs3yipplwtu/close",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_status_for_close"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCloseJob_FailsWhenProposalWouldNeedAutoAwardButIsNotImported(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpnimp01"
	const proposalID = "fcpropnimpinpr"

	scenario := tests.ApiScenario{
		Name:   "in-progress non-imported proposal cannot be auto-awarded by close",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"code":"proposal_not_awarded"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			project, err := app.FindRecordById("jobs", projectID)
			if err != nil {
				tb.Fatalf("failed to reload project: %v", err)
			}
			if got := project.GetString("status"); got != "Active" {
				tb.Fatalf("expected blocked close to keep project Active, got %q", got)
			}
			proposal, err := app.FindRecordById("jobs", proposalID)
			if err != nil {
				tb.Fatalf("failed to reload proposal: %v", err)
			}
			if got := proposal.GetString("status"); got != "In Progress" {
				tb.Fatalf("expected proposal to remain In Progress, got %q", got)
			}
		},
	}

	scenario.Test(t)
}

func TestCloseJob_ResponseOmitsProposalBlockWhenNoProposal(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpnoprop1"

	scenario := tests.ApiScenario{
		Name:   "close response has no proposal object when project has no proposal",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"` + projectID + `"`,
			`"project_note_created":true`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}
