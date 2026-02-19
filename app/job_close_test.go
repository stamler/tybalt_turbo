package main

import (
	"net/http"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// =============================================================================
// Fast Close Endpoint Tests
// =============================================================================
//
// These tests intentionally rely on static fixture rows in app/test_pb_data/data.db.
// They do not mutate fixture state for setup during test execution.

func TestCloseJob_ImportedBypassClosesAndCreatesProjectNote(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpnoprop1"
	const projectNoteText = "Project closed via imported fast close flow"

	var beforeProjectNoteCount int64

	scenario := tests.ApiScenario{
		Name:   "imported project closes via bypass and writes project note",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"` + projectID + `"`,
			`"status":"Closed"`,
			`"mode":"bypass"`,
			`"project_note_created":true`,
		},
		TestAppFactory: testutils.SetupTestApp,
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
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
				tb.Fatalf("failed to count project notes before request: %v", err)
			}
			beforeProjectNoteCount = result.Count
		},
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
			if result.Count <= beforeProjectNoteCount {
				tb.Fatalf("expected project close note to be created, before=%d after=%d", beforeProjectNoteCount, result.Count)
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
	const projectNoteText = "Project closed via imported fast close flow"
	const proposalNoteText = "Proposal auto-awarded during imported fast close of project " + projectNumber + " (" + projectID + ")"

	var beforeProjectNoteCount int64
	var beforeProposalNoteCount int64

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
			`"mode":"bypass"`,
			`"project_note_created":true`,
			`"auto_awarded":true`,
			`"proposal_note_created":true`,
			`"to_status":"Awarded"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
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
				tb.Fatalf("failed to count project notes before request: %v", err)
			}
			beforeProjectNoteCount = projectNoteResult.Count

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
				tb.Fatalf("failed to count proposal notes before request: %v", err)
			}
			beforeProposalNoteCount = proposalNoteResult.Count
		},
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
			if projectNoteResult.Count <= beforeProjectNoteCount {
				tb.Fatalf("expected project close note to be created, before=%d after=%d", beforeProjectNoteCount, projectNoteResult.Count)
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
			if proposalNoteResult.Count <= beforeProposalNoteCount {
				tb.Fatalf("expected proposal auto-award note to be created, before=%d after=%d", beforeProposalNoteCount, proposalNoteResult.Count)
			}
		},
	}

	scenario.Test(t)
}

func TestCloseJob_NonImportedUsesStrictValidation(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "test_job_w_rs"

	scenario := tests.ApiScenario{
		Name:   "non-imported close fails strict validation for incomplete project",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"data":{"location":{"code":"invalid_or_missing"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
			project, err := app.FindRecordById("jobs", projectID)
			if err != nil {
				tb.Fatalf("failed to reload project: %v", err)
			}
			if got := project.GetString("status"); got != "Active" {
				tb.Fatalf("expected strict-validation failure to keep status Active, got %q", got)
			}
		},
	}

	scenario.Test(t)
}

func TestCloseJob_ProposalTerminalStatusBlocksClose(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "fcprojimpterm01"

	scenario := tests.ApiScenario{
		Name:   "terminal proposal status blocks imported close",
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
		Name:   "in-progress non-imported proposal cannot be auto-awarded by fast close",
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

func TestCloseJob_ValidationErrorShapeForNonImportedMatchesHookPattern(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	const projectID = "test_job_w_rs"

	scenario := tests.ApiScenario{
		Name:   "strict close returns hook-style field errors",
		Method: http.MethodPost,
		URL:    "/api/jobs/" + projectID + "/close",
		Headers: map[string]string{
			"Authorization": recordToken,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"message":"invalid or missing location"`,
			`"data":{"location":{"code":"invalid_or_missing"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}
