package routes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	paTestProjectID       = "cjf0kt0defhq480"
	paTestOtherProjectID  = "u09fwwcg07y03m7"
	paJobClaimEmail       = "author@soup.com"
	paManagerEmail        = "fakemanager@fakesite.xyz"
	paNoClaimsEmail       = "u_no_claims@example.com"
	paAccountingEmail     = "author@soup.com"
	paAdminEmail          = "author@soup.com"
	paPDFContent          = "%PDF-1.4\n% project authorization test\n"
	paReplacementPDF      = "%PDF-1.4\n% replacement project authorization test\n"
	paNonPDFContent       = "not a pdf"
	paHashRepairFakeHash  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	projectAuthorizationQ = "/api/jobs/" + paTestProjectID
)

func TestProjectAuthorizationDocumentUploadPermissions(t *testing.T) {
	scenarios := []struct {
		name   string
		email  string
		setup  func(t *testing.T, app *tests.TestApp)
		status int
	}{
		{name: "job claim holder", email: paJobClaimEmail, status: http.StatusOK},
		{name: "assigned manager", email: paManagerEmail, status: http.StatusOK},
		{
			name:  "alternate manager",
			email: paNoClaimsEmail,
			setup: func(t *testing.T, app *tests.TestApp) {
				t.Helper()
				// Vary this fixture in-test because alternate-manager identity is
				// the behavior under test and the base job fixture has no alternate.
				setJobFieldForPATest(t, app, paTestProjectID, "alternate_manager", "u_no_claims")
			},
			status: http.StatusOK,
		},
		{
			name:  "branch manager",
			email: paNoClaimsEmail,
			setup: func(t *testing.T, app *tests.TestApp) {
				t.Helper()
				// Branch manager is a new field in this feature, so tests assign it
				// to an existing branch fixture instead of broadening base access.
				setBranchManagerForPATest(t, app, "80875lm27v8wgi4", "u_no_claims")
			},
			status: http.StatusOK,
		},
		{name: "unrelated user", email: paNoClaimsEmail, status: http.StatusForbidden},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := newProjectAuthorizationTestApp(t)
			if scenario.setup != nil {
				scenario.setup(t, app)
			}
			token := authTokenForEmail(t, app, scenario.email)
			rec := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", token, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
			if rec.Code != scenario.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, scenario.status, rec.Body.String())
			}
			if scenario.status == http.StatusOK {
				job, err := app.FindRecordById("jobs", paTestProjectID)
				if err != nil {
					t.Fatalf("failed to load job: %v", err)
				}
				if got := job.GetString("project_authorization_doc_hash"); got != sha256HexForPATest(paPDFContent) {
					t.Fatalf("project_authorization_doc_hash = %s, want calculated hash", got)
				}
				if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
					t.Fatalf("upload should leave approval blank; reviewed=%q reviewer=%q", job.GetString("pa_reviewed"), job.GetString("pa_reviewer"))
				}
			}
		})
	}
}

func TestProjectAuthorizationUploadRejectsNonPDFAndDuplicateHash(t *testing.T) {
	t.Run("non-pdf", func(t *testing.T) {
		app := newProjectAuthorizationTestApp(t)
		token := authTokenForEmail(t, app, paJobClaimEmail)
		rec := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", token, "project_authorization_doc", "not-pa.txt", "text/plain", []byte(paNonPDFContent))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		app := newProjectAuthorizationTestApp(t)
		token := authTokenForEmail(t, app, paJobClaimEmail)
		first := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", token, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
		if first.Code != http.StatusOK {
			t.Fatalf("first status = %d; body=%s", first.Code, first.Body.String())
		}
		duplicate := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, "/api/jobs/"+paTestOtherProjectID+"/project_authorization_doc", token, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
		if duplicate.Code != http.StatusBadRequest || !strings.Contains(duplicate.Body.String(), "duplicate_file") {
			t.Fatalf("duplicate response = %d, body=%s", duplicate.Code, duplicate.Body.String())
		}
	})
}

func TestProjectAuthorizationApprovalAndRevocation(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	jobToken := authTokenForEmail(t, app, paJobClaimEmail)
	accountingToken := authTokenForEmail(t, app, paAccountingEmail)
	noClaimsToken := authTokenForEmail(t, app, paNoClaimsEmail)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	upload := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", jobToken, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
	if upload.Code != http.StatusOK {
		t.Fatalf("upload status = %d; body=%s", upload.Code, upload.Body.String())
	}
	hash := sha256HexForPATest(paPDFContent)

	nonAccounting := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/approve", noClaimsToken, map[string]any{
		"project_authorization_doc_hash": hash,
	})
	if nonAccounting.Code != http.StatusForbidden {
		t.Fatalf("non-accounting status = %d, want forbidden; body=%s", nonAccounting.Code, nonAccounting.Body.String())
	}

	stale := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/approve", accountingToken, map[string]any{
		"project_authorization_doc_hash": strings.Repeat("a", 64),
	})
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "project_authorization_doc_changed") {
		t.Fatalf("stale response = %d, body=%s", stale.Code, stale.Body.String())
	}

	approve := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/approve", accountingToken, map[string]any{
		"project_authorization_doc_hash": hash,
	})
	if approve.Code != http.StatusOK {
		t.Fatalf("approve status = %d; body=%s", approve.Code, approve.Body.String())
	}
	job, err := app.FindRecordById("jobs", paTestProjectID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	if job.GetString("pa_reviewed") == "" || job.GetString("pa_reviewer") != "f2j5a8vk006baub" {
		t.Fatalf("approval fields reviewed=%q reviewer=%q", job.GetString("pa_reviewed"), job.GetString("pa_reviewer"))
	}

	already := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/approve", accountingToken, map[string]any{
		"project_authorization_doc_hash": hash,
	})
	if already.Code != http.StatusConflict || !strings.Contains(already.Body.String(), "project_authorization_already_approved") {
		t.Fatalf("already-approved response = %d, body=%s", already.Code, already.Body.String())
	}

	replaceApproved := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", jobToken, "project_authorization_doc", "replacement-pa.pdf", "application/pdf", []byte(paReplacementPDF))
	if replaceApproved.Code != http.StatusBadRequest || !strings.Contains(replaceApproved.Body.String(), "project_authorization_approved_immutable") {
		t.Fatalf("replace approved response = %d, body=%s", replaceApproved.Code, replaceApproved.Body.String())
	}

	deleteApproved := performClaimsJSONRequest(t, app, http.MethodDelete, projectAuthorizationQ+"/project_authorization_doc", jobToken, nil)
	if deleteApproved.Code != http.StatusBadRequest || !strings.Contains(deleteApproved.Body.String(), "project_authorization_approved_immutable") {
		t.Fatalf("delete approved response = %d, body=%s", deleteApproved.Code, deleteApproved.Body.String())
	}

	nonAdminRevoke := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/revoke", noClaimsToken, nil)
	if nonAdminRevoke.Code != http.StatusForbidden {
		t.Fatalf("non-admin revoke status = %d, want forbidden; body=%s", nonAdminRevoke.Code, nonAdminRevoke.Body.String())
	}

	revoke := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/revoke", adminToken, nil)
	if revoke.Code != http.StatusOK {
		t.Fatalf("revoke status = %d; body=%s", revoke.Code, revoke.Body.String())
	}
	job, _ = app.FindRecordById("jobs", paTestProjectID)
	if job.GetString("project_authorization_doc") == "" || job.GetString("project_authorization_doc_hash") == "" {
		t.Fatalf("revocation should preserve document and hash")
	}
	if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
		t.Fatalf("revocation should clear only approval fields")
	}

	unauthorizedDelete := performClaimsJSONRequest(t, app, http.MethodDelete, projectAuthorizationQ+"/project_authorization_doc", noClaimsToken, nil)
	if unauthorizedDelete.Code != http.StatusForbidden {
		t.Fatalf("unauthorized delete status = %d, want forbidden; body=%s", unauthorizedDelete.Code, unauthorizedDelete.Body.String())
	}

	deleteAfterRevoke := performClaimsJSONRequest(t, app, http.MethodDelete, projectAuthorizationQ+"/project_authorization_doc", jobToken, nil)
	if deleteAfterRevoke.Code != http.StatusOK {
		t.Fatalf("delete after revoke status = %d; body=%s", deleteAfterRevoke.Code, deleteAfterRevoke.Body.String())
	}
	job, _ = app.FindRecordById("jobs", paTestProjectID)
	if job.GetString("project_authorization_doc") != "" || job.GetString("project_authorization_doc_hash") != "" {
		t.Fatalf("delete should clear document and hash; doc=%q hash=%q", job.GetString("project_authorization_doc"), job.GetString("project_authorization_doc_hash"))
	}
	if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
		t.Fatalf("delete should leave approval fields blank")
	}
	queueAfterDelete := performClaimsJSONRequest(t, app, http.MethodGet, "/api/jobs/project_authorization/pending", jobToken, nil)
	if queueAfterDelete.Code != http.StatusOK || strings.Contains(queueAfterDelete.Body.String(), paTestProjectID) {
		t.Fatalf("deleted PA should not remain in approval queue; status=%d body=%s", queueAfterDelete.Code, queueAfterDelete.Body.String())
	}

	replaceAfterRevoke := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", jobToken, "project_authorization_doc", "replacement-pa.pdf", "application/pdf", []byte(paReplacementPDF))
	if replaceAfterRevoke.Code != http.StatusOK {
		t.Fatalf("replace after revoke status = %d; body=%s", replaceAfterRevoke.Code, replaceAfterRevoke.Body.String())
	}
	job, _ = app.FindRecordById("jobs", paTestProjectID)
	if got := job.GetString("project_authorization_doc_hash"); got != sha256HexForPATest(paReplacementPDF) {
		t.Fatalf("replacement hash = %s, want %s", got, sha256HexForPATest(paReplacementPDF))
	}
}

func TestProjectAuthorizationApprovalRejectsBlankStoredHash(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	accountingToken := authTokenForEmail(t, app, paAccountingEmail)
	if _, err := app.DB().NewQuery(`
		UPDATE jobs
		SET project_authorization_doc = 'signed-pa.pdf',
		    project_authorization_doc_hash = ''
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": paTestProjectID}).Execute(); err != nil {
		t.Fatalf("failed to seed blank PA hash fixture: %v", err)
	}

	approve := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization/approve", accountingToken, map[string]any{
		"project_authorization_doc_hash": "",
	})
	if approve.Code != http.StatusConflict || !strings.Contains(approve.Body.String(), "project_authorization_doc_changed") {
		t.Fatalf("blank-hash approval response = %d, body=%s", approve.Code, approve.Body.String())
	}
	job, err := app.FindRecordById("jobs", paTestProjectID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
		t.Fatalf("blank-hash approval must not set approval fields")
	}
}

func TestProjectAuthorizationDocHashAuditRequiresAdminAndDocument(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	noClaimsToken := authTokenForEmail(t, app, paNoClaimsEmail)

	noDocument := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc_hash/audit", adminToken, nil)
	if noDocument.Code != http.StatusNotFound {
		t.Fatalf("no-document audit status = %d, want %d; body=%s", noDocument.Code, http.StatusNotFound, noDocument.Body.String())
	}

	uploadProjectAuthorizationDocForHashRepairTest(t, app)
	audit := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc_hash/audit", noClaimsToken, nil)
	if audit.Code != http.StatusForbidden {
		t.Fatalf("non-admin audit status = %d, want %d; body=%s", audit.Code, http.StatusForbidden, audit.Body.String())
	}

	replace := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc_hash/replace", noClaimsToken, map[string]any{
		"updated": jobUpdatedForPATest(t, app, paTestProjectID),
	})
	if replace.Code != http.StatusForbidden {
		t.Fatalf("non-admin replace status = %d, want %d; body=%s", replace.Code, http.StatusForbidden, replace.Body.String())
	}
}

func TestProjectAuthorizationDocHashAuditReportsUploadedDocument(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	expectedHash := uploadProjectAuthorizationDocForHashRepairTest(t, app)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, http.StatusOK)
	if audit.JobID != paTestProjectID || audit.TargetCollection != "jobs" || audit.TargetID != paTestProjectID {
		t.Fatalf("audit target = %+v, want jobs/%s", audit, paTestProjectID)
	}
	if !audit.Matches || audit.StoredHash != expectedHash || audit.CalculatedHash != expectedHash {
		t.Fatalf("audit hashes = %+v, want matching uploaded PA hash %s", audit, expectedHash)
	}
	if audit.Filename == "" || !strings.Contains(audit.StoragePath, "/"+paTestProjectID+"/") {
		t.Fatalf("audit storage target = filename %q path %q, want job file path", audit.Filename, audit.StoragePath)
	}
}

func TestProjectAuthorizationDocHashReplaceUpdatesMismatchedHash(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	expectedHash := uploadProjectAuthorizationDocForHashRepairTest(t, app)
	// Deliberately corrupt the stored hash after upload; the repair path exists
	// to recover this inconsistent production state, so a static fixture would
	// hide the relationship between the real file bytes and the bad hash.
	setProjectAuthorizationDocHashForPATest(t, app, paTestProjectID, paHashRepairFakeHash)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, http.StatusOK)
	if audit.Matches || audit.StoredHash != paHashRepairFakeHash || audit.CalculatedHash != expectedHash {
		t.Fatalf("audit = %+v, want mismatched PA hash", audit)
	}

	replace := replaceProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, audit.Updated, http.StatusOK)
	if !replace.Replaced || replace.Noop || replace.PreviousHash != paHashRepairFakeHash || replace.NewHash != expectedHash {
		t.Fatalf("replace response = %+v, want PA hash replacement", replace)
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paTestProjectID); got != expectedHash {
		t.Fatalf("job PA hash = %s, want %s", got, expectedHash)
	}
}

func TestProjectAuthorizationDocHashReplaceNoopsWhenHashAlreadyMatches(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	expectedHash := uploadProjectAuthorizationDocForHashRepairTest(t, app)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, http.StatusOK)

	replace := replaceProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, audit.Updated, http.StatusOK)
	if !replace.Noop || replace.Replaced {
		t.Fatalf("replace response = %+v, want noop", replace)
	}
	if replace.StoredHash != expectedHash || replace.NewHash != expectedHash {
		t.Fatalf("replace hashes = %+v, want unchanged %s", replace, expectedHash)
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paTestProjectID); got != expectedHash {
		t.Fatalf("job PA hash = %s, want unchanged %s", got, expectedHash)
	}
}

func TestProjectAuthorizationDocHashReplaceRejectsBlankOrStaleUpdated(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	uploadProjectAuthorizationDocForHashRepairTest(t, app)
	// Deliberately corrupt the stored hash after upload so the replace request
	// has work to do before the stale timestamp guard rejects it.
	setProjectAuthorizationDocHashForPATest(t, app, paTestProjectID, paHashRepairFakeHash)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, http.StatusOK)

	blank := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc_hash/replace", adminToken, map[string]any{
		"updated": "   ",
	})
	if blank.Code != http.StatusBadRequest {
		t.Fatalf("blank updated status = %d, want %d; body=%s", blank.Code, http.StatusBadRequest, blank.Body.String())
	}

	// Simulate a concurrent job write after audit; this is timestamp state, not
	// durable business data, so mutating it in-test keeps the stale guard explicit.
	setJobUpdatedForPATest(t, app, paTestProjectID, "2026-06-03 00:00:00.000Z")
	stale := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale replace status = %d, want %d; body=%s", stale.Code, http.StatusConflict, stale.Body.String())
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paTestProjectID); got != paHashRepairFakeHash {
		t.Fatalf("stale replace changed hash to %s, want %s", got, paHashRepairFakeHash)
	}
}

func TestProjectAuthorizationDocHashAuditReportsMissingStorageObject(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	uploadProjectAuthorizationDocForHashRepairTest(t, app)
	deleteProjectAuthorizationDocFileForPATest(t, app, paTestProjectID)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	auditProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, http.StatusNotFound)
}

func TestProjectAuthorizationDocHashReplaceReportsUniqueHashConflict(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	expectedHash := uploadProjectAuthorizationDocForHashRepairTest(t, app)
	// Deliberately corrupt two uploaded jobs into a hash-collision scenario;
	// constructing it after upload exercises the unique-index failure without
	// making the CSV fixtures permanently inconsistent.
	setProjectAuthorizationDocHashForPATest(t, app, paTestProjectID, paHashRepairFakeHash)
	setProjectAuthorizationDocHashForPATest(t, app, paTestOtherProjectID, expectedHash)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paTestProjectID, http.StatusOK)

	replace := performClaimsJSONRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if replace.Code != http.StatusConflict {
		t.Fatalf("unique conflict status = %d, want %d; body=%s", replace.Code, http.StatusConflict, replace.Body.String())
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paTestProjectID); got != paHashRepairFakeHash {
		t.Fatalf("unique conflict changed hash to %s, want %s", got, paHashRepairFakeHash)
	}
}

func TestProjectAuthorizationQueueAndSchema(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	token := authTokenForEmail(t, app, paJobClaimEmail)
	upload := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", token, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
	if upload.Code != http.StatusOK {
		t.Fatalf("upload status = %d; body=%s", upload.Code, upload.Body.String())
	}

	queue := performClaimsJSONRequest(t, app, http.MethodGet, "/api/jobs/project_authorization/pending", token, nil)
	if queue.Code != http.StatusOK || !strings.Contains(queue.Body.String(), paTestProjectID) || !strings.Contains(queue.Body.String(), sha256HexForPATest(paPDFContent)) {
		t.Fatalf("queue response = %d, body=%s", queue.Code, queue.Body.String())
	}

	jobsCollection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		t.Fatalf("failed to load jobs collection: %v", err)
	}
	raw, err := json.Marshal(jobsCollection.Fields.GetByName("project_authorization_doc"))
	if err != nil {
		t.Fatalf("failed to marshal PA field: %v", err)
	}
	if !strings.Contains(string(raw), `"application/pdf"`) || strings.Contains(string(raw), `"image/png"`) {
		t.Fatalf("unexpected project_authorization_doc field schema: %s", raw)
	}
}

func TestProjectAuthorizationGenericJobUpdateCannotMutateFields(t *testing.T) {
	protectedFields := map[string]any{
		"project_authorization_doc":      "pa.pdf",
		"project_authorization_doc_hash": strings.Repeat("a", 64),
		"pa_reviewer":                    "f2j5a8vk006baub",
		"pa_reviewed":                    "2026-06-02 12:00:00.000Z",
	}

	for field, value := range protectedFields {
		t.Run(field, func(t *testing.T) {
			app := newProjectAuthorizationTestApp(t)
			token := authTokenForEmail(t, app, paJobClaimEmail)
			upsert := performClaimsJSONRequest(t, app, http.MethodPut, projectAuthorizationQ, token, map[string]any{
				"job": map[string]any{
					field: value,
				},
				"allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 1}},
			})
			if upsert.Code != http.StatusBadRequest || !strings.Contains(upsert.Body.String(), "not_editable") || !strings.Contains(upsert.Body.String(), field) {
				t.Fatalf("upsert response = %d, body=%s", upsert.Code, upsert.Body.String())
			}
		})
	}
}

func TestProjectAuthorizationCustomJobCreateCannotMutateFields(t *testing.T) {
	protectedFields := map[string]any{
		"project_authorization_doc":      "pa.pdf",
		"project_authorization_doc_hash": strings.Repeat("a", 64),
		"pa_reviewer":                    "f2j5a8vk006baub",
		"pa_reviewed":                    "2026-06-02 12:00:00.000Z",
	}

	for field, value := range protectedFields {
		t.Run(field, func(t *testing.T) {
			app := newProjectAuthorizationTestApp(t)
			token := authTokenForEmail(t, app, paJobClaimEmail)
			create := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs", token, map[string]any{
				"job": map[string]any{
					field: value,
				},
				"allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 1}},
			})
			if create.Code != http.StatusBadRequest || !strings.Contains(create.Body.String(), "not_editable") || !strings.Contains(create.Body.String(), field) {
				t.Fatalf("create response = %d, body=%s", create.Code, create.Body.String())
			}
		})
	}
}

func auditProjectAuthorizationDocHashForTest(t *testing.T, app *tests.TestApp, token string, jobID string, status int) projectAuthorizationDocHashAuditResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+jobID+"/project_authorization_doc_hash/audit", token, nil)
	if rec.Code != status {
		t.Fatalf("PA hash audit status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return projectAuthorizationDocHashAuditResponse{}
	}
	var response projectAuthorizationDocHashAuditResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode PA hash audit response: %v", err)
	}
	return response
}

func replaceProjectAuthorizationDocHashForTest(t *testing.T, app *tests.TestApp, token string, jobID string, updated string, status int) projectAuthorizationDocHashReplaceResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+jobID+"/project_authorization_doc_hash/replace", token, map[string]any{
		"updated": updated,
	})
	if rec.Code != status {
		t.Fatalf("PA hash replace status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return projectAuthorizationDocHashReplaceResponse{}
	}
	var response projectAuthorizationDocHashReplaceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode PA hash replace response: %v", err)
	}
	return response
}

func uploadProjectAuthorizationDocForHashRepairTest(t *testing.T, app *tests.TestApp) string {
	t.Helper()
	token := authTokenForEmail(t, app, paJobClaimEmail)
	rec := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, projectAuthorizationQ+"/project_authorization_doc", token, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
	if rec.Code != http.StatusOK {
		t.Fatalf("PA upload status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
	return projectAuthorizationDocHashForPATest(t, app, paTestProjectID)
}

func projectAuthorizationDocHashForPATest(t *testing.T, app *tests.TestApp, jobID string) string {
	t.Helper()
	var hash string
	if err := app.DB().NewQuery("SELECT project_authorization_doc_hash FROM jobs WHERE id = {:id}").
		Bind(dbx.Params{"id": jobID}).Row(&hash); err != nil {
		t.Fatalf("failed to read PA document hash: %v", err)
	}
	return hash
}

func setProjectAuthorizationDocHashForPATest(t *testing.T, app *tests.TestApp, jobID string, hash string) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE jobs SET project_authorization_doc_hash = {:hash} WHERE id = {:id}").
		Bind(dbx.Params{"hash": hash, "id": jobID}).Execute(); err != nil {
		t.Fatalf("failed to mutate PA document hash: %v", err)
	}
}

func jobUpdatedForPATest(t *testing.T, app *tests.TestApp, jobID string) string {
	t.Helper()
	var updated string
	if err := app.DB().NewQuery("SELECT updated FROM jobs WHERE id = {:id}").
		Bind(dbx.Params{"id": jobID}).Row(&updated); err != nil {
		t.Fatalf("failed to read job updated timestamp: %v", err)
	}
	return updated
}

func setJobUpdatedForPATest(t *testing.T, app *tests.TestApp, jobID string, updated string) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE jobs SET updated = {:updated} WHERE id = {:id}").
		Bind(dbx.Params{"updated": updated, "id": jobID}).Execute(); err != nil {
		t.Fatalf("failed to mutate job updated timestamp: %v", err)
	}
}

func deleteProjectAuthorizationDocFileForPATest(t *testing.T, app *tests.TestApp, jobID string) {
	t.Helper()
	job, err := app.FindRecordById("jobs", jobID)
	if err != nil {
		t.Fatalf("failed to load job: %v", err)
	}
	filename := strings.TrimSpace(job.GetString("project_authorization_doc"))
	if filename == "" {
		t.Fatal("job has no project authorization document to delete")
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("failed to open filesystem: %v", err)
	}
	defer fsys.Close()
	if err := fsys.Delete(job.BaseFilesPath() + "/" + filename); err != nil {
		t.Fatalf("failed to delete PA document fixture file: %v", err)
	}
}

func performProjectAuthorizationMultipartRequest(t *testing.T, app *tests.TestApp, method string, path string, token string, field string, filename string, contentType string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create api router: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(textprotoMIMEHeader(field, filename, contentType))
	if err != nil {
		t.Fatalf("failed to create multipart part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write multipart content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		if err != nil {
			return err
		}
		mux.ServeHTTP(recorder, req)
		return nil
	}); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return recorder
}

func newProjectAuthorizationTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app := testseed.NewSeededTestApp(t)
	hooks.AddHooks(app)
	AddRoutes(app)
	return app
}

func textprotoMIMEHeader(field string, filename string, contentType string) textproto.MIMEHeader {
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	return header
}

func sha256HexForPATest(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func setJobFieldForPATest(t *testing.T, app *tests.TestApp, jobID string, field string, value string) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE jobs SET " + field + " = {:value} WHERE id = {:id}").Bind(dbx.Params{"value": value, "id": jobID}).Execute(); err != nil {
		t.Fatalf("failed to update job fixture field %s: %v", field, err)
	}
}

func setBranchManagerForPATest(t *testing.T, app *tests.TestApp, branchID string, managerID string) {
	t.Helper()
	branch, err := app.FindRecordById("branches", branchID)
	if err != nil {
		t.Fatalf("failed to load branch fixture: %v", err)
	}
	branch.Set("manager", managerID)
	if err := app.Save(branch); err != nil {
		t.Fatalf("failed to save branch manager fixture: %v", err)
	}
}
