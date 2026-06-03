package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func TestAttachmentAuditTargetsRequireAdminAndExcludeGeneratedFileCollections(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	noClaimsToken := authTokenForEmail(t, app, "u_no_claims@example.com")

	forbidden := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets", noClaimsToken, nil)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want %d; body=%s", forbidden.Code, http.StatusForbidden, forbidden.Body.String())
	}

	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("targets status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var targets []attachmentAuditTargetResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &targets); err != nil {
		t.Fatalf("failed to decode targets response: %v", err)
	}

	if !attachmentAuditTargetsContain(targets, "purchase_orders_attachment") {
		t.Fatal("expected purchase_orders attachment audit target")
	}
	if !attachmentAuditTargetsContain(targets, "jobs_project_authorization_doc") {
		t.Fatal("expected jobs project authorization document audit target")
	}
	if attachmentAuditTargetsContain(targets, "zip_cache") {
		t.Fatal("expected generated zip_cache files to be excluded")
	}
}

func TestAttachmentAuditRefreshCachesCountsAndReports(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	run := waitForAttachmentAuditRun(t, app, adminToken, "purchase_orders_attachment")
	if run.Status != attachmentAuditStatusComplete {
		t.Fatalf("audit status = %q, want %q; error=%s", run.Status, attachmentAuditStatusComplete, run.Error)
	}
	if run.MatchingRecords < 1 {
		t.Fatalf("matching_records = %d, want at least 1", run.MatchingRecords)
	}
	if run.MissingRecords < 1 {
		t.Fatalf("missing_records = %d, want at least 1", run.MissingRecords)
	}
	if run.OrphanedFiles < 1 {
		t.Fatalf("orphaned_files = %d, want at least 1", run.OrphanedFiles)
	}
	if !run.HasMissingReport || !run.HasOrphanedReport {
		t.Fatalf("expected cached reports to be present: %+v", run)
	}

	missing := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment/missing.csv", adminToken, nil)
	if missing.Code != http.StatusOK {
		t.Fatalf("missing report status = %d, want %d; body=%s", missing.Code, http.StatusOK, missing.Body.String())
	}
	if got := missing.Header().Get("Content-Type"); !strings.Contains(got, "text/csv") {
		t.Fatalf("missing report content type = %q, want text/csv", got)
	}
	if !strings.Contains(missing.Body.String(), "poaudmissing001") {
		t.Fatalf("missing report did not include fixture row: %s", missing.Body.String())
	}

	orphaned := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment/orphaned.csv", adminToken, nil)
	if orphaned.Code != http.StatusOK {
		t.Fatalf("orphaned report status = %d, want %d; body=%s", orphaned.Code, http.StatusOK, orphaned.Body.String())
	}
	if !strings.Contains(orphaned.Body.String(), "poaudorphan0010") {
		t.Fatalf("orphaned report did not include fixture row: %s", orphaned.Body.String())
	}
	if !strings.Contains(orphaned.Body.String(), "record_id_from_path") {
		t.Fatalf("orphaned report header did not describe path-derived record id: %s", orphaned.Body.String())
	}
	if strings.Contains(orphaned.Body.String(), "storage_path,record_id,filename") {
		t.Fatalf("orphaned report still used misleading record_id header: %s", orphaned.Body.String())
	}
}

func TestAttachmentAuditRefreshCompletesForAllTargets(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")

	for _, target := range attachmentAuditTargets {
		t.Run(target.Key, func(t *testing.T) {
			rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/"+target.Key+"/refresh", adminToken, nil)
			if rec.Code != http.StatusAccepted {
				t.Fatalf("refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
			}

			run := waitForAttachmentAuditRun(t, app, adminToken, target.Key)
			if run.Status != attachmentAuditStatusComplete {
				t.Fatalf("audit status = %q, want %q; error=%s", run.Status, attachmentAuditStatusComplete, run.Error)
			}
			if !run.HasMissingReport || !run.HasOrphanedReport {
				t.Fatalf("expected cached reports to be present for %s: %+v", target.Key, run)
			}
		})
	}
}

func TestAttachmentAuditRefreshAndDownloadsRequireAdmin(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	noClaimsToken := authTokenForEmail(t, app, "u_no_claims@example.com")

	refresh := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", noClaimsToken, nil)
	if refresh.Code != http.StatusForbidden {
		t.Fatalf("non-admin refresh status = %d, want %d; body=%s", refresh.Code, http.StatusForbidden, refresh.Body.String())
	}

	deleteOrphans := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/delete_orphans", noClaimsToken, nil)
	if deleteOrphans.Code != http.StatusForbidden {
		t.Fatalf("non-admin delete orphans status = %d, want %d; body=%s", deleteOrphans.Code, http.StatusForbidden, deleteOrphans.Body.String())
	}

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("admin refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	run := waitForAttachmentAuditRun(t, app, adminToken, "purchase_orders_attachment")
	if run.Status != attachmentAuditStatusComplete {
		t.Fatalf("audit status = %q, want %q; error=%s", run.Status, attachmentAuditStatusComplete, run.Error)
	}

	missing := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment/missing.csv", noClaimsToken, nil)
	if missing.Code != http.StatusForbidden {
		t.Fatalf("non-admin missing report status = %d, want %d; body=%s", missing.Code, http.StatusForbidden, missing.Body.String())
	}

	orphaned := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment/orphaned.csv", noClaimsToken, nil)
	if orphaned.Code != http.StatusForbidden {
		t.Fatalf("non-admin orphaned report status = %d, want %d; body=%s", orphaned.Code, http.StatusForbidden, orphaned.Body.String())
	}
}

func TestAttachmentAuditUnknownTargetAndMissingReportResponses(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")

	for _, scenario := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "get", method: http.MethodGet, path: "/api/attachment_audit/targets/not_a_target"},
		{name: "refresh", method: http.MethodPost, path: "/api/attachment_audit/targets/not_a_target/refresh"},
		{name: "delete orphans", method: http.MethodPost, path: "/api/attachment_audit/targets/not_a_target/delete_orphans"},
		{name: "missing download", method: http.MethodGet, path: "/api/attachment_audit/targets/not_a_target/missing.csv"},
		{name: "orphaned download", method: http.MethodGet, path: "/api/attachment_audit/targets/not_a_target/orphaned.csv"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			rec := performClaimsJSONRequest(t, app, scenario.method, scenario.path, adminToken, nil)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusNotFound, rec.Body.String())
			}
		})
	}

	missing := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment/missing.csv", adminToken, nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing report before run status = %d, want %d; body=%s", missing.Code, http.StatusNotFound, missing.Body.String())
	}

	orphaned := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment/orphaned.csv", adminToken, nil)
	if orphaned.Code != http.StatusNotFound {
		t.Fatalf("orphaned report before run status = %d, want %d; body=%s", orphaned.Code, http.StatusNotFound, orphaned.Body.String())
	}

	deleteOrphans := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/delete_orphans", adminToken, nil)
	if deleteOrphans.Code != http.StatusNotFound {
		t.Fatalf("delete orphans before run status = %d, want %d; body=%s", deleteOrphans.Code, http.StatusNotFound, deleteOrphans.Body.String())
	}
}

func TestAttachmentAuditDuplicateRefreshBeforeRunRecordExistsReturnsRunning(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	target, ok := attachmentAuditTargetByKey("purchase_orders_attachment")
	if !ok {
		t.Fatal("expected purchase_orders attachment audit target")
	}

	attachmentAuditActiveRuns.Store(target.Key, true)
	t.Cleanup(func() {
		attachmentAuditActiveRuns.Delete(target.Key)
	})

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("duplicate refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var run attachmentAuditRunResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &run); err != nil {
		t.Fatalf("failed to decode duplicate refresh response: %v", err)
	}
	if run.TargetKey != target.Key || run.Status != attachmentAuditStatusRunning {
		t.Fatalf("duplicate refresh response = %+v, want running response for %s", run, target.Key)
	}
}

func TestAttachmentAuditDeleteOrphansReturnsConflictWhenTargetActive(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	target, ok := attachmentAuditTargetByKey("purchase_orders_attachment")
	if !ok {
		t.Fatal("expected purchase_orders attachment audit target")
	}

	attachmentAuditActiveRuns.Store(target.Key, true)
	t.Cleanup(func() {
		attachmentAuditActiveRuns.Delete(target.Key)
	})

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/delete_orphans", adminToken, nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("delete conflict status = %d, want %d; body=%s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestAttachmentAuditFailedRunKeepsCachedReports(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	target, ok := attachmentAuditTargetByKey("purchase_orders_attachment")
	if !ok {
		t.Fatal("expected purchase_orders attachment audit target")
	}

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	completed := waitForAttachmentAuditRun(t, app, adminToken, target.Key)
	if completed.Status != attachmentAuditStatusComplete {
		t.Fatalf("audit status = %q, want %q; error=%s", completed.Status, attachmentAuditStatusComplete, completed.Error)
	}

	if err := markAttachmentAuditRunFailed(app, target, completed.RequestedBy, errors.New("synthetic storage outage")); err != nil {
		t.Fatalf("failed to mark run failed: %v", err)
	}

	status := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/purchase_orders_attachment", adminToken, nil)
	if status.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body=%s", status.Code, http.StatusOK, status.Body.String())
	}

	var failed attachmentAuditRunResponse
	if err := json.Unmarshal(status.Body.Bytes(), &failed); err != nil {
		t.Fatalf("failed to decode failed run response: %v", err)
	}
	if failed.Status != attachmentAuditStatusFailed || !strings.Contains(failed.Error, "synthetic storage outage") {
		t.Fatalf("failed run response = %+v, want failed status with error", failed)
	}
	if failed.MatchingRecords != completed.MatchingRecords ||
		failed.MissingRecords != completed.MissingRecords ||
		failed.OrphanedFiles != completed.OrphanedFiles ||
		!failed.HasMissingReport ||
		!failed.HasOrphanedReport {
		t.Fatalf("failed run did not keep cached result: completed=%+v failed=%+v", completed, failed)
	}
}

func TestAttachmentAuditDeleteOrphansRemovesCachedOrphansAndRefreshes(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	orphanPath := attachmentAuditStoragePathForTest(t, app, "purchase_orders", "poaudghost00001", "audit-ghost.pdf")
	referencedPath := attachmentAuditStoragePathForTest(t, app, "purchase_orders", "poaudpresent001", "audit-present.pdf")
	if !attachmentAuditFileExists(t, app, orphanPath) {
		t.Fatalf("expected orphan fixture file to exist at %s", orphanPath)
	}
	if !attachmentAuditFileExists(t, app, referencedPath) {
		t.Fatalf("expected referenced fixture file to exist at %s", referencedPath)
	}

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	completed := waitForAttachmentAuditRun(t, app, adminToken, "purchase_orders_attachment")
	if completed.OrphanedFiles < 1 {
		t.Fatalf("orphaned files = %d, want at least 1", completed.OrphanedFiles)
	}

	deleted := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/delete_orphans", adminToken, nil)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete orphans status = %d, want %d; body=%s", deleted.Code, http.StatusOK, deleted.Body.String())
	}

	var response attachmentAuditDeleteOrphansResponse
	if err := json.Unmarshal(deleted.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode delete response: %v", err)
	}
	if response.DeletedFiles < 1 {
		t.Fatalf("deleted files = %d, want at least 1; response=%+v", response.DeletedFiles, response)
	}
	if response.Latest == nil || response.Latest.OrphanedFiles != 0 {
		t.Fatalf("latest run after delete = %+v, want refreshed audit with no orphans", response.Latest)
	}
	if attachmentAuditFileExists(t, app, orphanPath) {
		t.Fatalf("expected orphan fixture file to be deleted at %s", orphanPath)
	}
	if !attachmentAuditFileExists(t, app, referencedPath) {
		t.Fatalf("referenced fixture file was deleted at %s", referencedPath)
	}
}

func TestAttachmentAuditDeleteOrphansCountsAlreadyMissingCachedRows(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	orphanPath := attachmentAuditStoragePathForTest(t, app, "purchase_orders", "poaudghost00001", "audit-ghost.pdf")
	referencedPath := attachmentAuditStoragePathForTest(t, app, "purchase_orders", "poaudpresent001", "audit-present.pdf")

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	completed := waitForAttachmentAuditRun(t, app, adminToken, "purchase_orders_attachment")
	if completed.OrphanedFiles < 1 {
		t.Fatalf("orphaned files = %d, want at least 1", completed.OrphanedFiles)
	}

	// This test intentionally mutates the copied test fixture storage after the
	// audit snapshot so one cached orphan candidate is gone before deletion.
	attachmentAuditDeleteFileForTest(t, app, orphanPath)

	deleted := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/delete_orphans", adminToken, nil)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete orphans status = %d, want %d; body=%s", deleted.Code, http.StatusOK, deleted.Body.String())
	}

	var response attachmentAuditDeleteOrphansResponse
	if err := json.Unmarshal(deleted.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode delete response: %v", err)
	}
	if response.AlreadyMissingFiles < 1 {
		t.Fatalf("already missing files = %d, want at least 1; response=%+v", response.AlreadyMissingFiles, response)
	}
	if !attachmentAuditFileExists(t, app, referencedPath) {
		t.Fatalf("referenced fixture file was deleted at %s", referencedPath)
	}
}

func TestAttachmentAuditDeleteOrphansSkipsCachedRowsThatBecameReferenced(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	orphanPath := attachmentAuditStoragePathForTest(t, app, "purchase_orders", "poaudorphan0010", "audit-orphan.pdf")

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/refresh", adminToken, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("refresh status = %d, want %d; body=%s", rec.Code, http.StatusAccepted, rec.Body.String())
	}
	completed := waitForAttachmentAuditRun(t, app, adminToken, "purchase_orders_attachment")
	if completed.OrphanedFiles < 1 {
		t.Fatalf("orphaned files = %d, want at least 1", completed.OrphanedFiles)
	}

	// This test intentionally mutates the copied test fixture after the audit
	// snapshot so the cached orphan path becomes referenced before deletion.
	if _, err := app.DB().Update("purchase_orders", dbx.Params{
		"attachment": "audit-orphan.pdf",
	}, dbx.HashExp{"id": "poaudorphan0010"}).Execute(); err != nil {
		t.Fatalf("failed to update purchase order fixture attachment: %v", err)
	}

	deleted := performClaimsJSONRequest(t, app, http.MethodPost, "/api/attachment_audit/targets/purchase_orders_attachment/delete_orphans", adminToken, nil)
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete orphans status = %d, want %d; body=%s", deleted.Code, http.StatusOK, deleted.Body.String())
	}

	var response attachmentAuditDeleteOrphansResponse
	if err := json.Unmarshal(deleted.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode delete response: %v", err)
	}
	if response.SkippedReferencedFiles < 1 {
		t.Fatalf("skipped referenced files = %d, want at least 1; response=%+v", response.SkippedReferencedFiles, response)
	}
	if !attachmentAuditFileExists(t, app, orphanPath) {
		t.Fatalf("expected revalidated referenced file to remain at %s", orphanPath)
	}
	if response.Latest == nil || response.Latest.OrphanedFiles != 0 || response.Latest.MatchingRecords < 1 {
		t.Fatalf("latest run after skipped delete = %+v, want refreshed audit with referenced file present", response.Latest)
	}
}

func attachmentAuditTargetsContain(targets []attachmentAuditTargetResponse, key string) bool {
	for _, target := range targets {
		if target.Key == key {
			return true
		}
	}
	return false
}

func waitForAttachmentAuditRun(t *testing.T, app *tests.TestApp, token string, target string) attachmentAuditRunResponse {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/attachment_audit/targets/"+target, token, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status poll = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var run attachmentAuditRunResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &run); err != nil {
			t.Fatalf("failed to decode status response: %v", err)
		}
		if run.Status != attachmentAuditStatusRunning {
			return run
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for attachment audit run")
	return attachmentAuditRunResponse{}
}

func attachmentAuditStoragePathForTest(t *testing.T, app *tests.TestApp, collectionName string, recordID string, filename string) string {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("failed to find collection %s: %v", collectionName, err)
	}
	return collection.BaseFilesPath() + "/" + recordID + "/" + filename
}

func attachmentAuditFileExists(t *testing.T, app *tests.TestApp, storagePath string) bool {
	t.Helper()

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("failed to open filesystem: %v", err)
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(storagePath)
	if err != nil {
		if errors.Is(err, filesystem.ErrNotFound) {
			return false
		}
		t.Fatalf("failed to read %s: %v", storagePath, err)
	}
	reader.Close()
	return true
}

func attachmentAuditDeleteFileForTest(t *testing.T, app *tests.TestApp, storagePath string) {
	t.Helper()

	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatalf("failed to open filesystem: %v", err)
	}
	defer fsys.Close()

	if err := fsys.Delete(storagePath); err != nil {
		t.Fatalf("failed to delete test fixture file %s: %v", storagePath, err)
	}
}
