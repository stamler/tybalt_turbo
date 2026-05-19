package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"

	"tybalt/internal/testseed"
)

const (
	hashRepairAdminEmail      = "author@soup.com"
	hashRepairNonAdminEmail   = "u_no_claims@example.com"
	hashRepairLegacyExpenseID = "bflegacyalpha01"
	hashRepairMissingFileID   = "bflegmissing001"
	hashRepairDocumentExpense = "bfexistingdoc01"
	hashRepairDocumentID      = "bfexistdoc00001"
	hashRepairAlphaHash       = "f72056b24144bcf8349b9f3bed4e955c8d6ed1a03e1bb964cc311dbaf3b95639"
	hashRepairBlankActualHash = "de97f763576a8cb867473b0798e892ef3ddf60c4df0e6ac3c236aff99717fd87"
	hashRepairFakeHash        = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

func TestExpenseAttachmentHashAuditSupportsLegacyAndDocumentBackedAttachments(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)

	legacy := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, http.StatusOK)
	if legacy.TargetCollection != "expenses" {
		t.Fatalf("legacy target collection = %s, want expenses", legacy.TargetCollection)
	}
	if legacy.TargetID != hashRepairLegacyExpenseID {
		t.Fatalf("legacy target id = %s, want %s", legacy.TargetID, hashRepairLegacyExpenseID)
	}
	if !legacy.Matches || legacy.CalculatedHash != hashRepairAlphaHash || legacy.StoredHash != hashRepairAlphaHash {
		t.Fatalf("legacy audit = %+v, want matching alpha hash", legacy)
	}

	linkExpenseToDocumentForHashRepairTest(t, app, hashRepairDocumentExpense, hashRepairDocumentID)
	documentBacked := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairDocumentExpense, http.StatusOK)
	if documentBacked.TargetCollection != "expense_documents" {
		t.Fatalf("document target collection = %s, want expense_documents", documentBacked.TargetCollection)
	}
	if documentBacked.TargetID != hashRepairDocumentID {
		t.Fatalf("document target id = %s, want %s", documentBacked.TargetID, hashRepairDocumentID)
	}
	if !documentBacked.Matches || !strings.Contains(documentBacked.StoragePath, "pbc_2089657321/"+hashRepairDocumentID+"/existing-doc.pdf") {
		t.Fatalf("document-backed audit = %+v, want matching document storage path", documentBacked)
	}
}

func TestExpenseAttachmentHashRepairRequiresAdmin(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	nonAdminToken := authTokenForEmail(t, app, hashRepairNonAdminEmail)
	audit := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairLegacyExpenseID+"/attachment_hash/audit", nonAdminToken, nil)
	if audit.Code != http.StatusForbidden {
		t.Fatalf("non-admin audit status = %d, want %d; body=%s", audit.Code, http.StatusForbidden, audit.Body.String())
	}

	replace := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairLegacyExpenseID+"/attachment_hash/replace", nonAdminToken, map[string]any{
		"updated": "2026-05-01 00:00:00.000Z",
	})
	if replace.Code != http.StatusForbidden {
		t.Fatalf("non-admin replace status = %d, want %d; body=%s", replace.Code, http.StatusForbidden, replace.Body.String())
	}

	markMissing := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairLegacyExpenseID+"/attachment_missing/mark", nonAdminToken, map[string]any{
		"updated": expenseUpdatedForRepairTest(t, app, hashRepairLegacyExpenseID),
		"reason":  "test reason",
	})
	if markMissing.Code != http.StatusForbidden {
		t.Fatalf("non-admin mark missing status = %d, want %d; body=%s", markMissing.Code, http.StatusForbidden, markMissing.Body.String())
	}
}

func TestExpenseAttachmentHashReplaceUpdatesMismatchedLegacyHash(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	setExpenseAttachmentHashForRepairTest(t, app, hashRepairLegacyExpenseID, hashRepairFakeHash)
	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)

	audit := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, http.StatusOK)
	if audit.Matches {
		t.Fatalf("audit unexpectedly matched after fixture hash mutation: %+v", audit)
	}

	replace := replaceExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, audit.Updated, http.StatusOK)
	if !replace.Replaced || replace.Noop || replace.PreviousHash != hashRepairFakeHash || replace.NewHash != hashRepairAlphaHash {
		t.Fatalf("replace response = %+v, want replaced fake hash with alpha hash", replace)
	}
	if got := expenseAttachmentHashForRepairTest(t, app, hashRepairLegacyExpenseID); got != hashRepairAlphaHash {
		t.Fatalf("persisted hash = %s, want %s", got, hashRepairAlphaHash)
	}
}

func TestExpenseAttachmentHashReplaceUpdatesDocumentBackedHash(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	linkExpenseToDocumentForHashRepairTest(t, app, hashRepairDocumentExpense, hashRepairDocumentID)
	setExpenseDocumentAttachmentHashForRepairTest(t, app, hashRepairDocumentID, hashRepairFakeHash)
	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)

	audit := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairDocumentExpense, http.StatusOK)
	if audit.TargetCollection != "expense_documents" || audit.Matches {
		t.Fatalf("document-backed audit = %+v, want mismatched expense_documents target", audit)
	}

	replace := replaceExpenseAttachmentHashForTest(t, app, adminToken, hashRepairDocumentExpense, audit.Updated, http.StatusOK)
	if !replace.Replaced || replace.TargetCollection != "expense_documents" {
		t.Fatalf("replace response = %+v, want document-backed replacement", replace)
	}
	if got := expenseDocumentAttachmentHashForRepairTest(t, app, hashRepairDocumentID); got != replace.NewHash {
		t.Fatalf("document hash = %s, want %s", got, replace.NewHash)
	}
}

func TestExpenseAttachmentHashReplaceNoopsWhenHashAlreadyMatches(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	audit := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, http.StatusOK)

	replace := replaceExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, audit.Updated, http.StatusOK)
	if !replace.Noop || replace.Replaced {
		t.Fatalf("replace response = %+v, want noop", replace)
	}
	if got := expenseAttachmentHashForRepairTest(t, app, hashRepairLegacyExpenseID); got != hashRepairAlphaHash {
		t.Fatalf("persisted hash = %s, want unchanged %s", got, hashRepairAlphaHash)
	}
}

func TestExpenseAttachmentHashReplaceRejectsStaleUpdated(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	setExpenseAttachmentHashForRepairTest(t, app, hashRepairLegacyExpenseID, hashRepairFakeHash)
	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	audit := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, http.StatusOK)

	if _, err := app.DB().NewQuery("UPDATE expenses SET updated = {:updated} WHERE id = {:id}").
		Bind(dbx.Params{"updated": "2026-05-03 00:00:00.000Z", "id": hashRepairLegacyExpenseID}).Execute(); err != nil {
		t.Fatalf("failed to mutate updated timestamp: %v", err)
	}

	replace := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairLegacyExpenseID+"/attachment_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if replace.Code != http.StatusConflict {
		t.Fatalf("stale replace status = %d, want %d; body=%s", replace.Code, http.StatusConflict, replace.Body.String())
	}
	if got := expenseAttachmentHashForRepairTest(t, app, hashRepairLegacyExpenseID); got != hashRepairFakeHash {
		t.Fatalf("stale replace changed hash to %s, want %s", got, hashRepairFakeHash)
	}
}

func TestExpenseAttachmentHashReplaceRejectsStaleUpdatedBeforeNoop(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	audit := auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairLegacyExpenseID, http.StatusOK)

	if _, err := app.DB().NewQuery("UPDATE expenses SET updated = {:updated} WHERE id = {:id}").
		Bind(dbx.Params{"updated": "2026-05-03 00:00:00.000Z", "id": hashRepairLegacyExpenseID}).Execute(); err != nil {
		t.Fatalf("failed to mutate updated timestamp: %v", err)
	}

	replace := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairLegacyExpenseID+"/attachment_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if replace.Code != http.StatusConflict {
		t.Fatalf("stale noop replace status = %d, want %d; body=%s", replace.Code, http.StatusConflict, replace.Body.String())
	}
}

func TestExpenseAttachmentHashAuditReportsMissingStorageObject(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	auditExpenseAttachmentHashForTest(t, app, adminToken, hashRepairMissingFileID, http.StatusNotFound)
}

func TestExpenseAttachmentHashReplaceReportsUniqueHashConflict(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	audit := auditExpenseAttachmentHashForTest(t, app, adminToken, "bflegacyblank01", http.StatusOK)
	if audit.CalculatedHash != hashRepairBlankActualHash {
		t.Fatalf("blank fixture calculated hash = %s, want %s", audit.CalculatedHash, hashRepairBlankActualHash)
	}

	replace := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/bflegacyblank01/attachment_hash/replace", adminToken, map[string]any{
		"updated": audit.Updated,
	})
	if replace.Code != http.StatusConflict {
		t.Fatalf("conflict replace status = %d, want %d; body=%s", replace.Code, http.StatusConflict, replace.Body.String())
	}
	if got := expenseAttachmentHashForRepairTest(t, app, "bflegacyblank01"); got != "" {
		t.Fatalf("unique conflict changed blank fixture hash to %s, want empty", got)
	}
}

func TestExpenseAttachmentMissingMarkClearsLegacyAttachmentState(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	updated := expenseUpdatedForRepairTest(t, app, hashRepairMissingFileID)
	reason := "Legacy attachment file was already missing during Phase 2 migration; original receipt could not be recovered."

	response := markExpenseAttachmentMissingForTest(t, app, adminToken, hashRepairMissingFileID, updated, reason, http.StatusOK)
	if !response.Marked || response.Noop {
		t.Fatalf("mark missing response = %+v, want marked", response)
	}
	if response.PreviousAttachment == "" || response.AttachmentMissingReason != reason {
		t.Fatalf("mark missing response = %+v, want previous attachment and reason", response)
	}

	state := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairMissingFileID)
	if state.Attachment != "" || state.AttachmentHash != "" || state.AttachmentDocument != "" || state.AttachmentMissingReason != reason {
		t.Fatalf("expense missing state = %+v, want cleared attachment fields and reason", state)
	}
	if state.Updated == updated {
		t.Fatalf("expense updated timestamp did not change after mark missing")
	}
}

func TestExpenseAttachmentMissingMarkClearsDocumentBackedAttachmentState(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	linkExpenseToDocumentForHashRepairTest(t, app, hashRepairDocumentExpense, hashRepairDocumentID)
	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	updated := expenseUpdatedForRepairTest(t, app, hashRepairDocumentExpense)
	reason := "Document-backed attachment was confirmed unrecoverable during admin repair."

	response := markExpenseAttachmentMissingForTest(t, app, adminToken, hashRepairDocumentExpense, updated, reason, http.StatusOK)
	if !response.Marked || response.PreviousDocumentID != hashRepairDocumentID {
		t.Fatalf("mark missing response = %+v, want cleared document link", response)
	}

	state := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairDocumentExpense)
	if state.AttachmentDocument != "" || state.AttachmentMissingReason != reason {
		t.Fatalf("document-backed missing state = %+v, want empty document relation and reason", state)
	}
	if got := expenseDocumentAttachmentHashForRepairTest(t, app, hashRepairDocumentID); got == "" {
		t.Fatalf("document row hash was unexpectedly cleared")
	}
}

func TestExpenseAttachmentMissingMarkRejectsBlankReason(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairMissingFileID+"/attachment_missing/mark", adminToken, map[string]any{
		"updated": expenseUpdatedForRepairTest(t, app, hashRepairMissingFileID),
		"reason":  "   ",
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("blank reason status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestExpenseAttachmentMissingMarkNoopsWhenAlreadyMarkedWithSameReason(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	reason := "Legacy attachment was unrecoverable."
	firstUpdated := expenseUpdatedForRepairTest(t, app, hashRepairMissingFileID)
	markExpenseAttachmentMissingForTest(t, app, adminToken, hashRepairMissingFileID, firstUpdated, reason, http.StatusOK)

	stateAfterFirst := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairMissingFileID)
	response := markExpenseAttachmentMissingForTest(t, app, adminToken, hashRepairMissingFileID, stateAfterFirst.Updated, reason, http.StatusOK)
	if !response.Noop || response.Marked {
		t.Fatalf("second mark missing response = %+v, want noop", response)
	}
	stateAfterSecond := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairMissingFileID)
	if stateAfterSecond.Updated != stateAfterFirst.Updated {
		t.Fatalf("noop mark missing changed updated from %s to %s", stateAfterFirst.Updated, stateAfterSecond.Updated)
	}
}

func TestExpenseAttachmentMissingMarkRejectsReasonChangeOnceMarked(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	originalReason := "Legacy attachment was unrecoverable."
	firstUpdated := expenseUpdatedForRepairTest(t, app, hashRepairMissingFileID)
	markExpenseAttachmentMissingForTest(t, app, adminToken, hashRepairMissingFileID, firstUpdated, originalReason, http.StatusOK)

	stateAfterFirst := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairMissingFileID)
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairMissingFileID+"/attachment_missing/mark", adminToken, map[string]any{
		"updated": stateAfterFirst.Updated,
		"reason":  "A different historical explanation.",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("changed reason status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	stateAfterSecond := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairMissingFileID)
	if stateAfterSecond.AttachmentMissingReason != originalReason || stateAfterSecond.Updated != stateAfterFirst.Updated {
		t.Fatalf("changed reason mutated state from %+v to %+v", stateAfterFirst, stateAfterSecond)
	}
}

func TestExpenseAttachmentMissingMarkRejectsStaleUpdated(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, hashRepairAdminEmail)
	updated := expenseUpdatedForRepairTest(t, app, hashRepairMissingFileID)
	if _, err := app.DB().NewQuery("UPDATE expenses SET updated = {:updated} WHERE id = {:id}").
		Bind(dbx.Params{"updated": "2026-05-03 00:00:00.000Z", "id": hashRepairMissingFileID}).Execute(); err != nil {
		t.Fatalf("failed to mutate updated timestamp: %v", err)
	}

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+hashRepairMissingFileID+"/attachment_missing/mark", adminToken, map[string]any{
		"updated": updated,
		"reason":  "Legacy attachment was unrecoverable.",
	})
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale mark missing status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}

	state := expenseAttachmentMissingStateForRepairTest(t, app, hashRepairMissingFileID)
	if state.Attachment == "" || state.AttachmentMissingReason != "" {
		t.Fatalf("stale mark missing changed state to %+v", state)
	}
}

func auditExpenseAttachmentHashForTest(t *testing.T, app *tests.TestApp, token string, expenseID string, status int) expenseAttachmentHashAuditResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+expenseID+"/attachment_hash/audit", token, nil)
	if rec.Code != status {
		t.Fatalf("audit status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return expenseAttachmentHashAuditResponse{}
	}
	var response expenseAttachmentHashAuditResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode audit response: %v", err)
	}
	return response
}

func replaceExpenseAttachmentHashForTest(t *testing.T, app *tests.TestApp, token string, expenseID string, updated string, status int) expenseAttachmentHashReplaceResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+expenseID+"/attachment_hash/replace", token, map[string]any{
		"updated": updated,
	})
	if rec.Code != status {
		t.Fatalf("replace status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return expenseAttachmentHashReplaceResponse{}
	}
	var response expenseAttachmentHashReplaceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode replace response: %v", err)
	}
	return response
}

func markExpenseAttachmentMissingForTest(t *testing.T, app *tests.TestApp, token string, expenseID string, updated string, reason string, status int) expenseAttachmentMissingMarkResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/expenses/"+expenseID+"/attachment_missing/mark", token, map[string]any{
		"updated": updated,
		"reason":  reason,
	})
	if rec.Code != status {
		t.Fatalf("mark missing status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return expenseAttachmentMissingMarkResponse{}
	}
	var response expenseAttachmentMissingMarkResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode mark missing response: %v", err)
	}
	return response
}

func linkExpenseToDocumentForHashRepairTest(t *testing.T, app *tests.TestApp, expenseID string, documentID string) {
	t.Helper()
	// The canonical CSV fixture set predates the attachment_document column on
	// expenses. This targeted mutation creates the document-backed state needed
	// by the repair endpoint without adding a one-off fixture dump.
	if _, err := app.DB().NewQuery("UPDATE expenses SET attachment_document = {:document} WHERE id = {:id}").
		Bind(dbx.Params{"document": documentID, "id": expenseID}).Execute(); err != nil {
		t.Fatalf("failed to link expense to document fixture: %v", err)
	}
}

func setExpenseAttachmentHashForRepairTest(t *testing.T, app *tests.TestApp, expenseID string, hash string) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE expenses SET attachment_hash = {:hash} WHERE id = {:id}").
		Bind(dbx.Params{"hash": hash, "id": expenseID}).Execute(); err != nil {
		t.Fatalf("failed to mutate expense hash fixture: %v", err)
	}
}

func setExpenseDocumentAttachmentHashForRepairTest(t *testing.T, app *tests.TestApp, documentID string, hash string) {
	t.Helper()
	if _, err := app.DB().NewQuery("UPDATE expense_documents SET attachment_hash = {:hash} WHERE id = {:id}").
		Bind(dbx.Params{"hash": hash, "id": documentID}).Execute(); err != nil {
		t.Fatalf("failed to mutate expense document hash fixture: %v", err)
	}
}

func expenseAttachmentHashForRepairTest(t *testing.T, app *tests.TestApp, expenseID string) string {
	t.Helper()
	var hash string
	if err := app.DB().NewQuery("SELECT attachment_hash FROM expenses WHERE id = {:id}").
		Bind(dbx.Params{"id": expenseID}).Row(&hash); err != nil {
		t.Fatalf("failed to read expense hash fixture: %v", err)
	}
	return hash
}

type expenseAttachmentMissingState struct {
	Attachment              string `db:"attachment"`
	AttachmentHash          string `db:"attachment_hash"`
	AttachmentDocument      string `db:"attachment_document"`
	AttachmentMissingReason string `db:"attachment_missing_reason"`
	Updated                 string `db:"updated"`
}

func expenseAttachmentMissingStateForRepairTest(t *testing.T, app *tests.TestApp, expenseID string) expenseAttachmentMissingState {
	t.Helper()
	var state expenseAttachmentMissingState
	if err := app.DB().NewQuery(`
		SELECT attachment, attachment_hash, attachment_document, attachment_missing_reason, updated
		FROM expenses
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": expenseID}).One(&state); err != nil {
		t.Fatalf("failed to read expense attachment missing state: %v", err)
	}
	return state
}

func expenseUpdatedForRepairTest(t *testing.T, app *tests.TestApp, expenseID string) string {
	t.Helper()
	return expenseAttachmentMissingStateForRepairTest(t, app, expenseID).Updated
}

func expenseDocumentAttachmentHashForRepairTest(t *testing.T, app *tests.TestApp, documentID string) string {
	t.Helper()
	var hash string
	if err := app.DB().NewQuery("SELECT attachment_hash FROM expense_documents WHERE id = {:id}").
		Bind(dbx.Params{"id": documentID}).Row(&hash); err != nil {
		t.Fatalf("failed to read expense document hash fixture: %v", err)
	}
	return hash
}
