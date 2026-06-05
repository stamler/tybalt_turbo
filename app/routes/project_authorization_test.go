package routes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
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
	paPayablesAdminEmail  = "book@keeper.com"
	paPDFContent          = "%PDF-1.4\n% project authorization test\n"
	paReplacementPDF      = "%PDF-1.4\n% replacement project authorization test\n"
	paNonPDFContent       = "not a pdf"
	paPendingProjectID    = "pafixpending01"
	paAltManagerProjectID = "paaltmgrjob0001"
	paBranchProjectID     = "pabrmgrjob0001"
	paPendingHash         = "006bd775c07b0d78770fb855e5b2e814e98243432cc99f51ea7b0dd4f5914f2d"
	paBlankHashProjectID  = "pafixblankhash"
	paStaleHashProjectID  = "pafixstale001"
	paStaleHashStoredHash = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	paStaleHashActualHash = "4fc693a8dd23eb3c5f4ed362bfce20769906daaea8f476ba2cc0e0da9d82608e"
	projectAuthorizationQ = "/api/jobs/" + paTestProjectID
)

func TestProjectAuthorizationDocumentUploadPermissions(t *testing.T) {
	scenarios := []struct {
		name   string
		email  string
		jobID  string
		status int
	}{
		{name: "job claim holder", email: paJobClaimEmail, status: http.StatusOK},
		{name: "assigned manager", email: paManagerEmail, status: http.StatusOK},
		{name: "alternate manager", email: paNoClaimsEmail, jobID: paAltManagerProjectID, status: http.StatusOK},
		{name: "branch manager", email: paNoClaimsEmail, jobID: paBranchProjectID, status: http.StatusOK},
		{name: "unrelated user", email: paNoClaimsEmail, status: http.StatusForbidden},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := newProjectAuthorizationTestApp(t)
			jobID := scenario.jobID
			if jobID == "" {
				jobID = paTestProjectID
			}
			token := authTokenForEmail(t, app, scenario.email)
			rec := performProjectAuthorizationMultipartRequest(t, app, http.MethodPost, "/api/jobs/"+jobID+"/project_authorization_doc", token, "project_authorization_doc", "signed-pa.pdf", "application/pdf", []byte(paPDFContent))
			if rec.Code != scenario.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, scenario.status, rec.Body.String())
			}
			if scenario.status == http.StatusOK {
				job, err := app.FindRecordById("jobs", jobID)
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
	blankHashQ := "/api/jobs/" + paBlankHashProjectID

	approve := performClaimsJSONRequest(t, app, http.MethodPost, blankHashQ+"/project_authorization/approve", accountingToken, map[string]any{
		"project_authorization_doc_hash": "",
	})
	if approve.Code != http.StatusConflict || !strings.Contains(approve.Body.String(), "project_authorization_doc_changed") {
		t.Fatalf("blank-hash approval response = %d, body=%s", approve.Code, approve.Body.String())
	}
	job, err := app.FindRecordById("jobs", paBlankHashProjectID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if job.GetString("pa_reviewed") != "" || job.GetString("pa_reviewer") != "" {
		t.Fatalf("blank-hash approval must not set approval fields")
	}
}

func TestProjectAuthorizationUploadRejectsDocumentApprovedAfterInitialRead(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	jobToken := authTokenForEmail(t, app, paJobClaimEmail)
	pendingQ := "/api/jobs/" + paPendingProjectID

	upload := performBlockingProjectAuthorizationMultipartRequest(t, app, http.MethodPost, pendingQ+"/project_authorization_doc", jobToken, "project_authorization_doc", "replacement-pa.pdf", "application/pdf", []byte(paReplacementPDF), func() {
		setProjectAuthorizationApprovedForRouteTest(t, app, paPendingProjectID)
	})
	if upload.Code != http.StatusBadRequest || !strings.Contains(upload.Body.String(), "project_authorization_approved_immutable") {
		t.Fatalf("concurrent approval upload response = %d, body=%s", upload.Code, upload.Body.String())
	}

	job, err := app.FindRecordById("jobs", paPendingProjectID)
	if err != nil {
		t.Fatalf("failed to reload job: %v", err)
	}
	if job.GetString("pa_reviewed") == "" || job.GetString("pa_reviewer") == "" {
		t.Fatalf("upload cleared concurrent approval fields: reviewed=%q reviewer=%q", job.GetString("pa_reviewed"), job.GetString("pa_reviewer"))
	}
	if got := job.GetString("project_authorization_doc_hash"); got != paPendingHash {
		t.Fatalf("upload changed document hash despite concurrent approval: got %s, want %s", got, paPendingHash)
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

	audit := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+paPendingProjectID+"/project_authorization_doc_hash/audit", noClaimsToken, nil)
	if audit.Code != http.StatusForbidden {
		t.Fatalf("non-admin audit status = %d, want %d; body=%s", audit.Code, http.StatusForbidden, audit.Body.String())
	}

	replace := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+paPendingProjectID+"/project_authorization_doc_hash/replace", noClaimsToken, map[string]any{
		"updated": jobUpdatedForPATest(t, app, paPendingProjectID),
	})
	if replace.Code != http.StatusForbidden {
		t.Fatalf("non-admin replace status = %d, want %d; body=%s", replace.Code, http.StatusForbidden, replace.Body.String())
	}
}

func TestProjectAuthorizationDocHashAuditReportsUploadedDocument(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paPendingProjectID, http.StatusOK)
	if audit.JobID != paPendingProjectID || audit.TargetCollection != "jobs" || audit.TargetID != paPendingProjectID {
		t.Fatalf("audit target = %+v, want jobs/%s", audit, paPendingProjectID)
	}
	if !audit.Matches || audit.StoredHash != paPendingHash || audit.CalculatedHash != paPendingHash {
		t.Fatalf("audit hashes = %+v, want matching uploaded PA hash %s", audit, paPendingHash)
	}
	if audit.Filename == "" || !strings.Contains(audit.StoragePath, "/"+paPendingProjectID+"/") {
		t.Fatalf("audit storage target = filename %q path %q, want job file path", audit.Filename, audit.StoragePath)
	}
}

func TestProjectAuthorizationDocHashReplaceUpdatesMismatchedHash(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paStaleHashProjectID, http.StatusOK)
	if audit.Matches || audit.StoredHash != paStaleHashStoredHash || audit.CalculatedHash != paStaleHashActualHash {
		t.Fatalf("audit = %+v, want mismatched PA hash", audit)
	}

	replace := replaceProjectAuthorizationDocHashForTest(t, app, adminToken, paStaleHashProjectID, audit.Updated, http.StatusOK)
	if !replace.Replaced || replace.Noop || replace.PreviousHash != paStaleHashStoredHash || replace.NewHash != paStaleHashActualHash {
		t.Fatalf("replace response = %+v, want PA hash replacement", replace)
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paStaleHashProjectID); got != paStaleHashActualHash {
		t.Fatalf("job PA hash = %s, want %s", got, paStaleHashActualHash)
	}
}

func TestProjectAuthorizationDocHashReplaceNoopsWhenHashAlreadyMatches(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paPendingProjectID, http.StatusOK)

	replace := replaceProjectAuthorizationDocHashForTest(t, app, adminToken, paPendingProjectID, audit.Updated, http.StatusOK)
	if !replace.Noop || replace.Replaced {
		t.Fatalf("replace response = %+v, want noop", replace)
	}
	if replace.StoredHash != paPendingHash || replace.NewHash != paPendingHash {
		t.Fatalf("replace hashes = %+v, want unchanged %s", replace, paPendingHash)
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paPendingProjectID); got != paPendingHash {
		t.Fatalf("job PA hash = %s, want unchanged %s", got, paPendingHash)
	}
}

func TestProjectAuthorizationDocHashReplaceRejectsBlankOrStaleUpdated(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paStaleHashProjectID, http.StatusOK)

	blank := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+paStaleHashProjectID+"/project_authorization_doc_hash/replace", adminToken, map[string]any{
		"updated": "   ",
	})
	if blank.Code != http.StatusBadRequest {
		t.Fatalf("blank updated status = %d, want %d; body=%s", blank.Code, http.StatusBadRequest, blank.Body.String())
	}

	// Simulate a concurrent job write after audit; this is timestamp state, not
	// durable business data, so mutating it in-test keeps the stale guard explicit.
	setJobUpdatedForPATest(t, app, paStaleHashProjectID, "2026-06-03 01:00:00.000Z")
	stale := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+paStaleHashProjectID+"/project_authorization_doc_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale replace status = %d, want %d; body=%s", stale.Code, http.StatusConflict, stale.Body.String())
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paStaleHashProjectID); got != paStaleHashStoredHash {
		t.Fatalf("stale replace changed hash to %s, want %s", got, paStaleHashStoredHash)
	}
}

func TestProjectAuthorizationDocHashAuditReportsMissingStorageObject(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	deleteProjectAuthorizationDocFileForPATest(t, app, paPendingProjectID)
	adminToken := authTokenForEmail(t, app, paAdminEmail)

	auditProjectAuthorizationDocHashForTest(t, app, adminToken, paPendingProjectID, http.StatusNotFound)
}

func TestProjectAuthorizationDocHashReplaceReportsUniqueHashConflict(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	// Deliberately create a hash-collision scenario; constructing it at runtime
	// exercises the unique-index failure without making fixtures inconsistent.
	setProjectAuthorizationDocHashForPATest(t, app, paTestOtherProjectID, paStaleHashActualHash)
	adminToken := authTokenForEmail(t, app, paAdminEmail)
	audit := auditProjectAuthorizationDocHashForTest(t, app, adminToken, paStaleHashProjectID, http.StatusOK)

	replace := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+paStaleHashProjectID+"/project_authorization_doc_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if replace.Code != http.StatusConflict {
		t.Fatalf("unique conflict status = %d, want %d; body=%s", replace.Code, http.StatusConflict, replace.Body.String())
	}
	if got := projectAuthorizationDocHashForPATest(t, app, paStaleHashProjectID); got != paStaleHashStoredHash {
		t.Fatalf("unique conflict changed hash to %s, want %s", got, paStaleHashStoredHash)
	}
}

func TestProjectAuthorizationQueueAndSchema(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	token := authTokenForEmail(t, app, paAccountingEmail)
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

func TestProjectAuthorizationQueueRequiresAccountingClaim(t *testing.T) {
	scenarios := []struct {
		name   string
		email  string
		status int
	}{
		{name: "accounting", email: paAccountingEmail, status: http.StatusOK},
		{name: "payables admin only", email: paPayablesAdminEmail, status: http.StatusForbidden},
		{name: "no claims", email: paNoClaimsEmail, status: http.StatusForbidden},
		{name: "unauthenticated", status: http.StatusUnauthorized},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := newProjectAuthorizationTestApp(t)
			token := ""
			if scenario.email != "" {
				token = authTokenForEmail(t, app, scenario.email)
			}
			queue := performClaimsJSONRequest(t, app, http.MethodGet, "/api/jobs/project_authorization/pending", token, nil)
			if queue.Code != scenario.status {
				t.Fatalf("queue status = %d, want %d; body=%s", queue.Code, scenario.status, queue.Body.String())
			}
		})
	}
}

func TestProjectAuthorizationMissingQueueSegmentsAndPagination(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)
	token := authTokenForEmail(t, app, paAccountingEmail)

	inUse := getMissingProjectAuthorizationsForTest(t, app, token, "in_use", 1, 1, http.StatusOK)
	if inUse.Total <= 0 || inUse.TotalPages <= 0 {
		t.Fatalf("in-use response totals = total %d pages %d, want positive", inUse.Total, inUse.TotalPages)
	}
	if len(inUse.Items) != 1 || inUse.Items[0].ID != "pamissuse000001" {
		t.Fatalf("first in-use item = %+v, want pamissuse000001", inUse.Items)
	}
	if inUse.Items[0].Priority != projectAuthorizationMissingPriorityInUse || inUse.Items[0].TimeEntryCount == 0 {
		t.Fatalf("in-use fixture priority/counts = %+v", inUse.Items[0])
	}
	if inUse.PendingReviewCount != expectedProjectAuthorizationPendingReviewCount(t, app) {
		t.Fatalf("pending review count = %d, want %d", inUse.PendingReviewCount, expectedProjectAuthorizationPendingReviewCount(t, app))
	}

	defaultQueue := getMissingProjectAuthorizationsForTest(t, app, token, "", 1, 1, http.StatusOK)
	if defaultQueue.Priority != projectAuthorizationMissingPriorityInUse {
		t.Fatalf("default priority = %q, want in_use", defaultQueue.Priority)
	}
	if len(defaultQueue.Items) != 1 || defaultQueue.Items[0].ID != "pamissuse000001" {
		t.Fatalf("first default item = %+v, want in-use fixture", defaultQueue.Items)
	}

	recent := getMissingProjectAuthorizationsForTest(t, app, token, "recent", 1, 1, http.StatusOK)
	if len(recent.Items) != 1 || recent.Items[0].ID != "pamissrecent001" {
		t.Fatalf("first recent item = %+v, want pamissrecent001", recent.Items)
	}

	dormant := getMissingProjectAuthorizationsForTest(t, app, token, "dormant", 1, 1, http.StatusOK)
	if len(dormant.Items) != 1 || dormant.Items[0].ID != "pamissdormant01" {
		t.Fatalf("first dormant item = %+v, want pamissdormant01", dormant.Items)
	}

	all := getMissingProjectAuthorizationsForTest(t, app, token, "all", 1, 200, http.StatusOK)
	ids := missingProjectAuthorizationIDs(all.Items)
	for _, want := range []string{"pafixmissing01", "pafixblankhash", "pamissuse000001", "pamissrecent001", "pamissdormant01"} {
		if !ids[want] {
			t.Fatalf("all missing queue did not include %s; ids=%v", want, ids)
		}
	}
	for _, excluded := range []string{"pafixpending01", "pafixapprove01", "pamissclosed001", "pamissprop00001"} {
		if ids[excluded] {
			t.Fatalf("all missing queue included excluded job %s; ids=%v", excluded, ids)
		}
	}
	if all.Counts[projectAuthorizationMissingPriorityAll] != all.Total {
		t.Fatalf("all total = %d, all count = %d", all.Total, all.Counts[projectAuthorizationMissingPriorityAll])
	}

	overlargePage := getMissingProjectAuthorizationsForTest(t, app, token, "all", 999, 1, http.StatusOK)
	if overlargePage.Page != overlargePage.TotalPages {
		t.Fatalf("overlarge page = %d, want clamped total pages %d", overlargePage.Page, overlargePage.TotalPages)
	}
	if overlargePage.Total > 0 && len(overlargePage.Items) == 0 {
		t.Fatalf("overlarge page returned no items despite total %d: %+v", overlargePage.Total, overlargePage)
	}
}

func TestProjectAuthorizationMissingQueueScopesByUploadPermission(t *testing.T) {
	app := newProjectAuthorizationTestApp(t)

	jobToken := authTokenForEmail(t, app, paJobClaimEmail)
	jobResponse := getMissingProjectAuthorizationsForTest(t, app, jobToken, "all", 1, 200, http.StatusOK)
	jobIDs := missingProjectAuthorizationIDs(jobResponse.Items)
	if !jobIDs["pamissuse000001"] || !jobIDs["paaltmgrjob0001"] {
		t.Fatalf("job claim holder should see broad missing PA list; ids=%v", jobIDs)
	}

	scopedToken := authTokenForEmail(t, app, paNoClaimsEmail)
	scopedResponse := getMissingProjectAuthorizationsForTest(t, app, scopedToken, "all", 1, 200, http.StatusOK)
	scopedIDs := missingProjectAuthorizationIDs(scopedResponse.Items)
	if !scopedIDs["paaltmgrjob0001"] || !scopedIDs["pabrmgrjob0001"] {
		t.Fatalf("scoped user should see alternate-manager and branch-manager jobs; ids=%v", scopedIDs)
	}
	scopedDefaultResponse := getMissingProjectAuthorizationsForTest(t, app, scopedToken, "", 1, 50, http.StatusOK)
	if scopedDefaultResponse.Priority != projectAuthorizationMissingPriorityInUse {
		t.Fatalf("scoped default priority = %q, want in_use", scopedDefaultResponse.Priority)
	}
	if scopedDefaultResponse.PendingReviewCount != 0 {
		t.Fatalf("scoped pending review count = %d, want 0", scopedDefaultResponse.PendingReviewCount)
	}
	if scopedIDs["pamissuse000001"] || scopedIDs["pafixmissing01"] {
		t.Fatalf("scoped user saw unrelated missing PA jobs; ids=%v", scopedIDs)
	}
	for _, item := range scopedResponse.Items {
		if !item.CanUpload {
			t.Fatalf("scoped visible item should be uploadable by caller: %+v", item)
		}
	}

	payablesToken := authTokenForEmail(t, app, paPayablesAdminEmail)
	payablesResponse := getMissingProjectAuthorizationsForTest(t, app, payablesToken, "all", 1, 200, http.StatusOK)
	if payablesResponse.Total != 0 || len(payablesResponse.Items) != 0 {
		t.Fatalf("unscoped payables user should see no missing PA jobs: %+v", payablesResponse)
	}
	payablesOverlargePage := getMissingProjectAuthorizationsForTest(t, app, payablesToken, "all", 999, 50, http.StatusOK)
	if payablesOverlargePage.Page != 1 || payablesOverlargePage.TotalPages != 0 {
		t.Fatalf("empty queue page = %d total pages = %d, want page 1 total pages 0", payablesOverlargePage.Page, payablesOverlargePage.TotalPages)
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
	return decodeJSONResponseForTest[projectAuthorizationDocHashAuditResponse](t, rec, status, "PA hash audit")
}

func replaceProjectAuthorizationDocHashForTest(t *testing.T, app *tests.TestApp, token string, jobID string, updated string, status int) projectAuthorizationDocHashReplaceResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/jobs/"+jobID+"/project_authorization_doc_hash/replace", token, map[string]any{
		"updated": updated,
	})
	return decodeJSONResponseForTest[projectAuthorizationDocHashReplaceResponse](t, rec, status, "PA hash replace")
}

func decodeJSONResponseForTest[T any](t *testing.T, rec *httptest.ResponseRecorder, status int, label string) T {
	t.Helper()
	var response T
	if rec.Code != status {
		t.Fatalf("%s status = %d, want %d; body=%s", label, rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return response
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode %s response: %v", label, err)
	}
	return response
}

func getMissingProjectAuthorizationsForTest(t *testing.T, app *tests.TestApp, token string, priority string, page int, limit int, status int) projectAuthorizationMissingResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/jobs/project_authorization/missing?priority="+priority+"&page="+strconv.Itoa(page)+"&limit="+strconv.Itoa(limit), token, nil)
	return decodeJSONResponseForTest[projectAuthorizationMissingResponse](t, rec, status, "missing PA queue")
}

func missingProjectAuthorizationIDs(items []projectAuthorizationMissingRow) map[string]bool {
	ids := map[string]bool{}
	for _, item := range items {
		ids[item.ID] = true
	}
	return ids
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

type blockingMultipartReader struct {
	reader  *bytes.Reader
	started chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func (r *blockingMultipartReader) Read(p []byte) (int, error) {
	r.once.Do(func() {
		close(r.started)
		<-r.release
	})
	return r.reader.Read(p)
}

func performBlockingProjectAuthorizationMultipartRequest(t *testing.T, app *tests.TestApp, method string, path string, token string, field string, filename string, contentType string, content []byte, onBlocked func()) *httptest.ResponseRecorder {
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

	release := make(chan struct{})
	reader := &blockingMultipartReader{
		reader:  bytes.NewReader(body.Bytes()),
		started: make(chan struct{}),
		release: release,
	}
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	done := make(chan error, 1)
	go func() {
		serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
		done <- app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
			mux, err := e.Router.BuildMux()
			if err != nil {
				return err
			}
			mux.ServeHTTP(recorder, req)
			return nil
		})
	}()

	select {
	case <-reader.started:
		if onBlocked != nil {
			onBlocked()
		}
		close(release)
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("multipart upload did not start reading request body")
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("multipart upload did not finish after release")
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

func setProjectAuthorizationApprovedForRouteTest(t *testing.T, app *tests.TestApp, jobID string) {
	t.Helper()
	// This test mutates PA state mid-request to model a concurrent approval;
	// a static fixture cannot represent that interleaving.
	job, err := app.FindRecordById("jobs", jobID)
	if err != nil {
		t.Fatalf("failed to load PA fixture: %v", err)
	}
	job.Set("pa_reviewed", "2026-06-03 12:00:00.000Z")
	job.Set("pa_reviewer", "f2j5a8vk006baub")
	if err := app.SaveWithContext(hooks.WithProjectAuthorizationMutation(context.Background(), hooks.ProjectAuthorizationMutationApprove), job); err != nil {
		t.Fatalf("failed to approve PA fixture: %v", err)
	}
}
