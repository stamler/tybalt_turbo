package expense_documents

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"

	"tybalt/internal/testseed"
)

const (
	alphaExpenseID     = "bflegacyalpha01"
	sharedOneID        = "bflegacyshare01"
	blankHashID        = "bflegacyblank01"
	missingExpenseID   = "bflegmissing001"
	existingExpenseID  = "bfexistingdoc01"
	existingDocumentID = "bfexistdoc00001"

	alphaHash    = "f72056b24144bcf8349b9f3bed4e955c8d6ed1a03e1bb964cc311dbaf3b95639"
	sharedHash   = "4ed5f9dfe13234acd5fa3c1b12994145a7c39c27b7faf3732ed0e1b27686c902"
	blankHash    = "de97f763576a8cb867473b0798e892ef3ddf60c4df0e6ac3c236aff99717fd87"
	existingHash = "2f5a8ae84c688675754280b67c7f218294e1b8a9b55fe59ff055cefc111cac47"
)

func TestPrepareGeneratesDeterministicManifestAndCopyScript(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	if err := os.MkdirAll(paths.OutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CleanupScriptPath, []byte("stale smoke cleanup"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := Prepare(app, paths)
	if err != nil {
		t.Fatal(err)
	}

	rows := fixtureManifestRows(t, paths)
	if len(rows) != 4 {
		t.Fatalf("expected 4 fixture manifest rows, got %d", len(rows))
	}
	if result.CopyCommands < 3 {
		t.Fatalf("expected at least 3 copy commands, got %d", result.CopyCommands)
	}
	if result.CleanupCommands != 0 {
		t.Fatalf("full prepare cleanup commands = %d, want 0", result.CleanupCommands)
	}

	alpha := rows[alphaExpenseID]
	if alpha.AttachmentHash != alphaHash {
		t.Fatalf("alpha hash = %s, want %s", alpha.AttachmentHash, alphaHash)
	}
	assertGeneratedID(t, alpha.ExpenseDocumentID, alphaHash)
	if !alpha.CopyRequired || alpha.Status != StatusReady {
		t.Fatalf("alpha copy/status = %t/%s, want true/%s", alpha.CopyRequired, alpha.Status, StatusReady)
	}

	blank := rows[blankHashID]
	if blank.AttachmentHash != blankHash {
		t.Fatalf("blank-hash row hash = %s, want computed %s", blank.AttachmentHash, blankHash)
	}
	assertGeneratedID(t, blank.ExpenseDocumentID, blankHash)

	sharedOne := rows[sharedOneID]
	assertGeneratedID(t, sharedOne.ExpenseDocumentID, sharedHash)
	if sharedOne.DocumentAttachment != "shared-one.pdf" {
		t.Fatalf("shared document attachment = %s, want shared-one.pdf", sharedOne.DocumentAttachment)
	}

	existing := rows[existingExpenseID]
	if existing.ExpenseDocumentID != existingDocumentID {
		t.Fatalf("existing hash reused doc %s, want %s", existing.ExpenseDocumentID, existingDocumentID)
	}
	if existing.DocumentAttachment != "existing-doc.pdf" {
		t.Fatalf("existing hash document attachment = %s, want existing-doc.pdf", existing.DocumentAttachment)
	}
	if existing.CopyRequired || existing.Status != StatusVerified {
		t.Fatalf("existing hash copy/status = %t/%s, want false/%s", existing.CopyRequired, existing.Status, StatusVerified)
	}

	errorsFile, err := os.ReadFile(paths.ErrorsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(errorsFile), missingExpenseID) || !strings.Contains(string(errorsFile), "missing_legacy_file") {
		t.Fatalf("prepare errors did not include missing fixture: %s", string(errorsFile))
	}

	copyScript, err := os.ReadFile(paths.CopyScriptPath)
	if err != nil {
		t.Fatal(err)
	}
	copyScriptText := string(copyScript)
	if !strings.Contains(copyScriptText, alpha.OldS3Key) || !strings.Contains(copyScriptText, alpha.NewS3Key) {
		t.Fatalf("copy script did not include alpha copy command:\n%s", copyScriptText)
	}
	if !strings.Contains(copyScriptText, "--checksum-algorithm SHA256") {
		t.Fatalf("copy script should ask S3 to store SHA-256 checksum metadata:\n%s", copyScriptText)
	}
	if !strings.Contains(copyScriptText, "aws s3api head-object") {
		t.Fatalf("copy script should preflight destination keys before copying:\n%s", copyScriptText)
	}
	headIndex := strings.Index(copyScriptText, "aws s3api head-object")
	copyIndex := strings.Index(copyScriptText, "aws s3 cp")
	if headIndex < 0 || copyIndex < 0 || headIndex > copyIndex {
		t.Fatalf("copy script should preflight all destinations before copy commands:\n%s", copyScriptText)
	}
	if !strings.Contains(copyScriptText, "Refusing to overwrite existing destination") {
		t.Fatalf("copy script should abort on existing destination keys:\n%s", copyScriptText)
	}
	if strings.Contains(copyScriptText, "existing-doc.pdf") {
		t.Fatalf("copy script should not recopy existing document target:\n%s", copyScriptText)
	}
	if strings.Contains(copyScriptText, " rm ") || strings.Contains(copyScriptText, "delete") {
		t.Fatalf("copy script must copy only:\n%s", copyScriptText)
	}
	if _, err := os.Stat(paths.CleanupScriptPath); !os.IsNotExist(err) {
		t.Fatalf("full prepare should not leave cleanup script, stat err=%v", err)
	}
}

func TestPrepareLimitCanEmitSmallCopyOnlySmokeManifest(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	result, err := PrepareWithOptions(app, paths, PrepareOptions{
		Limit:       2,
		RequireCopy: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ManifestRows != 2 || result.CopyCommands != 2 || result.CleanupCommands != 2 {
		t.Fatalf("manifest/copy/cleanup counts = %d/%d/%d, want 2/2/2", result.ManifestRows, result.CopyCommands, result.CleanupCommands)
	}

	rows, err := ReadManifest(paths.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("limited manifest row count = %d, want 2", len(rows))
	}
	for _, row := range rows {
		if !row.CopyRequired {
			t.Fatalf("limited require-copy manifest included non-copy row: %+v", row)
		}
		if row.ExpenseID == existingExpenseID {
			t.Fatalf("limited require-copy manifest included reused existing document row")
		}
	}

	cleanupScript, err := os.ReadFile(paths.CleanupScriptPath)
	if err != nil {
		t.Fatal(err)
	}
	cleanupText := string(cleanupScript)
	if !strings.Contains(cleanupText, "CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS=yes") {
		t.Fatalf("cleanup script missing explicit confirmation guard:\n%s", cleanupText)
	}
	if !strings.Contains(cleanupText, "limited pre-apply smoke tests only") {
		t.Fatalf("cleanup script should describe smoke-test-only scope:\n%s", cleanupText)
	}
	if !strings.Contains(cleanupText, "aws s3 rm") {
		t.Fatalf("cleanup script did not emit destination deletes:\n%s", cleanupText)
	}
	if strings.Contains(cleanupText, "existing-doc.pdf") {
		t.Fatalf("cleanup script should not delete reused existing document target:\n%s", cleanupText)
	}
	for _, row := range rows {
		if strings.Contains(cleanupText, row.OldS3Key) {
			t.Fatalf("cleanup script must not delete legacy source key %s:\n%s", row.OldS3Key, cleanupText)
		}
		if !strings.Contains(cleanupText, row.NewS3Key) {
			t.Fatalf("cleanup script missing copied target key %s:\n%s", row.NewS3Key, cleanupText)
		}
	}
}

func TestVerifyHashesCopiedTargetsAgainstManifest(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	copyFixtureManifestObjects(t, app, paths)

	result, err := Verify(app, paths)
	if err != nil {
		t.Fatal(err)
	}
	rows := fixtureManifestRows(t, paths)
	for id, row := range rows {
		if row.Status != StatusVerified {
			t.Fatalf("fixture row %s status = %s, want %s", id, row.Status, StatusVerified)
		}
	}
	if result.Failed != 0 {
		t.Fatalf("verify failed %d rows, want 0", result.Failed)
	}

	corruptStorageObject(t, app, rows[alphaExpenseID].NewS3Key, []byte("wrong copied content"))
	result, err = Verify(app, paths)
	if err != nil {
		t.Fatal(err)
	}
	rows = fixtureManifestRows(t, paths)
	if rows[alphaExpenseID].Status != StatusVerifyError {
		t.Fatalf("corrupt target status = %s, want %s", rows[alphaExpenseID].Status, StatusVerifyError)
	}
	if result.Failed == 0 {
		t.Fatalf("verify did not fail after corrupting copied target")
	}
	errorsFile, err := os.ReadFile(paths.VerifyErrorsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(errorsFile), "copied_hash_mismatch") {
		t.Fatalf("verify errors did not include target hash mismatch: %s", string(errorsFile))
	}
}

func TestVerifyCanUseS3FullObjectChecksumMetadata(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	paths.BucketEnv = "TEST_BACKFILL_BUCKET"
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	copyFixtureManifestObjects(t, app, paths)
	rows := fixtureManifestRows(t, paths)
	checksums := map[string]s3ChecksumResult{}
	for _, row := range rows {
		checksums[row.NewS3Key] = s3ChecksumResult{
			HexChecksum: row.AttachmentHash,
			Found:       true,
			FullObject:  true,
		}
	}
	withFakeS3ChecksumVerifier(t, fakeS3ChecksumVerifier{checksums: checksums}, nil)
	t.Setenv("TEST_BACKFILL_BUCKET", "bucket")

	result, err := VerifyWithOptions(app, paths, VerifyOptions{ChecksumMode: ChecksumModeS3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 0 {
		t.Fatalf("S3 checksum verify failed %d rows, want 0", result.Failed)
	}
	rows = fixtureManifestRows(t, paths)
	for id, row := range rows {
		if row.Status != StatusVerified {
			t.Fatalf("fixture row %s status = %s, want %s", id, row.Status, StatusVerified)
		}
	}
}

func TestVerifyS3ModeRequiresAppReadableTarget(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	paths.BucketEnv = "TEST_BACKFILL_BUCKET"
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	rows := fixtureManifestRows(t, paths)
	checksums := map[string]s3ChecksumResult{}
	for _, row := range rows {
		if !row.CopyRequired {
			continue
		}
		checksums[row.NewS3Key] = s3ChecksumResult{
			HexChecksum: row.AttachmentHash,
			Found:       true,
			FullObject:  true,
		}
	}
	withFakeS3ChecksumVerifier(t, fakeS3ChecksumVerifier{checksums: checksums}, nil)
	t.Setenv("TEST_BACKFILL_BUCKET", "bucket")

	result, err := VerifyWithOptions(app, paths, VerifyOptions{ChecksumMode: ChecksumModeS3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed == 0 {
		t.Fatal("S3 checksum verify passed without app-readable copied targets")
	}
	errorsFile, err := os.ReadFile(paths.VerifyErrorsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(errorsFile), "app_storage_unreadable") {
		t.Fatalf("verify errors did not include app storage readability failure: %s", string(errorsFile))
	}
}

func TestVerifyS3ModeLocallyHashesRowsThatDidNotRequireCopy(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	paths.BucketEnv = "TEST_BACKFILL_BUCKET"
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	copyFixtureManifestObjects(t, app, paths)
	rows := fixtureManifestRows(t, paths)
	checksums := map[string]s3ChecksumResult{}
	for _, row := range rows {
		if !row.CopyRequired {
			continue
		}
		checksums[row.NewS3Key] = s3ChecksumResult{
			HexChecksum: row.AttachmentHash,
			Found:       true,
			FullObject:  true,
		}
	}
	withFakeS3ChecksumVerifier(t, fakeS3ChecksumVerifier{checksums: checksums}, nil)
	t.Setenv("TEST_BACKFILL_BUCKET", "bucket")

	result, err := VerifyWithOptions(app, paths, VerifyOptions{ChecksumMode: ChecksumModeS3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed != 0 {
		errorsFile, readErr := os.ReadFile(paths.VerifyErrorsPath)
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Fatalf("S3 checksum verify failed %d rows, want 0; errors=%s", result.Failed, string(errorsFile))
	}
	if rows[existingExpenseID].CopyRequired {
		t.Fatalf("fixture setup expected existing document row to skip copy")
	}
}

func TestVerifyS3ModeRequiresChecksumMetadata(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	paths.BucketEnv = "TEST_BACKFILL_BUCKET"
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	withFakeS3ChecksumVerifier(t, fakeS3ChecksumVerifier{checksums: map[string]s3ChecksumResult{}}, nil)
	t.Setenv("TEST_BACKFILL_BUCKET", "bucket")

	result, err := VerifyWithOptions(app, paths, VerifyOptions{ChecksumMode: ChecksumModeS3})
	if err != nil {
		t.Fatal(err)
	}
	if result.Failed == 0 {
		t.Fatal("S3 checksum verify passed without checksum metadata")
	}
	errorsFile, err := os.ReadFile(paths.VerifyErrorsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(errorsFile), "s3_checksum_unavailable") {
		t.Fatalf("verify errors did not include missing S3 checksum metadata: %s", string(errorsFile))
	}
}

func TestApplyInsertsDocumentsAndLinksExpenses(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	copyFixtureManifestObjects(t, app, paths)
	if _, err := Verify(app, paths); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(app, paths)
	if err != nil {
		t.Fatal(err)
	}
	if result.InsertedDocuments != 3 {
		t.Fatalf("inserted documents = %d, want 3 unique new hashes", result.InsertedDocuments)
	}
	if result.LinkedExpenses != 4 {
		t.Fatalf("linked expenses = %d, want 4", result.LinkedExpenses)
	}

	rows := fixtureManifestRows(t, paths)
	for id, row := range rows {
		documentID := expenseAttachmentDocument(t, app, id)
		if documentID != row.ExpenseDocumentID {
			t.Fatalf("expense %s attachment_document = %s, want %s", id, documentID, row.ExpenseDocumentID)
		}
		if row.Status != StatusApplied {
			t.Fatalf("manifest row %s status = %s, want %s", id, row.Status, StatusApplied)
		}
	}
	assertExpenseLegacyFields(t, app, alphaExpenseID, "alpha.pdf", alphaHash)
	assertDocumentRow(t, app, existingDocumentID, "existing-doc.pdf", existingHash)

	secondResult, err := Apply(app, paths)
	if err != nil {
		t.Fatal(err)
	}
	if secondResult.LinkedExpenses != 0 || secondResult.SkippedExpenses != 4 {
		t.Fatalf("idempotent apply linked/skipped = %d/%d, want 0/4", secondResult.LinkedExpenses, secondResult.SkippedExpenses)
	}
}

func TestApplyAbortsOnUnexpectedExistingLink(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	copyFixtureManifestObjects(t, app, paths)
	if _, err := Verify(app, paths); err != nil {
		t.Fatal(err)
	}

	// The CSV fixtures intentionally leave attachment_document blank for the
	// committed legacy rows; this mutation simulates production state changing
	// after verify but before apply.
	if _, err := app.DB().NewQuery("UPDATE expenses SET attachment_document = {:document} WHERE id = {:id}").
		Bind(dbx.Params{"document": existingDocumentID, "id": alphaExpenseID}).Execute(); err != nil {
		t.Fatal(err)
	}

	_, err := Apply(app, paths)
	if err == nil {
		t.Fatal("expected apply to abort on unexpected existing link")
	}
	if got := expenseAttachmentDocument(t, app, sharedOneID); got != "" {
		t.Fatalf("apply partially linked shared fixture despite abort: %s", got)
	}
}

func TestVerifyFailsWhenExpenseStateNoLongerMatchesManifest(t *testing.T) {
	cases := []struct {
		name   string
		params dbx.Params
		code   string
	}{
		{
			name:   "attachment changed",
			params: dbx.Params{"attachment": "changed.pdf"},
			code:   "expense_attachment_mismatch",
		},
		{
			name:   "hash changed",
			params: dbx.Params{"attachment_hash": strings.Repeat("a", 64)},
			code:   "expense_hash_mismatch",
		},
		{
			name:   "uncommitted",
			params: dbx.Params{"committed": ""},
			code:   "expense_no_longer_committed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			defer app.Cleanup()

			paths := DefaultPaths(t.TempDir())
			if _, err := Prepare(app, paths); err != nil {
				t.Fatal(err)
			}
			copyFixtureManifestObjects(t, app, paths)
			if _, err := app.DB().Update("expenses", tc.params, dbx.HashExp{"id": alphaExpenseID}).Execute(); err != nil {
				t.Fatalf("failed to mutate expense fixture: %v", err)
			}

			result, err := Verify(app, paths)
			if err != nil {
				t.Fatal(err)
			}
			if result.Failed == 0 {
				t.Fatal("verify passed after expense state drifted from manifest")
			}
			errorsFile, err := os.ReadFile(paths.VerifyErrorsPath)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(errorsFile), tc.code) {
				t.Fatalf("verify errors did not include %s: %s", tc.code, string(errorsFile))
			}
		})
	}
}

func TestApplyAbortsWhenExpenseStateNoLongerMatchesManifest(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	paths := DefaultPaths(t.TempDir())
	if _, err := Prepare(app, paths); err != nil {
		t.Fatal(err)
	}
	copyFixtureManifestObjects(t, app, paths)
	if _, err := Verify(app, paths); err != nil {
		t.Fatal(err)
	}
	if _, err := app.DB().Update("expenses", dbx.Params{
		"attachment_hash": strings.Repeat("b", 64),
	}, dbx.HashExp{"id": alphaExpenseID}).Execute(); err != nil {
		t.Fatalf("failed to mutate expense fixture: %v", err)
	}

	if _, err := Apply(app, paths); err == nil || !strings.Contains(err.Error(), "expense attachment_hash") {
		t.Fatalf("apply error = %v, want expense hash mismatch", err)
	}
}

func TestReportWritesBaselineCounts(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	// These mutations make the report-only edge cases explicit without adding
	// one-off fixture rows for values that the migration itself should reject.
	if _, err := app.DB().NewQuery("UPDATE expenses SET attachment = '', attachment_hash = '', attachment_document = {:document} WHERE id = {:id}").
		Bind(dbx.Params{"document": existingDocumentID, "id": existingExpenseID}).Execute(); err != nil {
		t.Fatalf("failed to create document-backed blank legacy fixture state: %v", err)
	}
	if _, err := app.DB().Update("expenses", dbx.Params{
		"attachment_hash": strings.Repeat("g", 64),
	}, dbx.HashExp{"id": alphaExpenseID}).Execute(); err != nil {
		t.Fatalf("failed to create invalid-hash fixture state: %v", err)
	}

	paths := DefaultPaths(t.TempDir())
	result, err := Report(app, paths)
	if err != nil {
		t.Fatal(err)
	}
	if result.CommittedLegacyOnlyAttachments < 4 {
		t.Fatalf("committed legacy-only attachments = %d, want at least fixture rows", result.CommittedLegacyOnlyAttachments)
	}
	if result.DocumentBackedBlankLegacy < 1 {
		t.Fatalf("document-backed blank legacy attachments = %d, want at least fixture row", result.DocumentBackedBlankLegacy)
	}
	if result.BlankOrInvalidHashes < 2 {
		t.Fatalf("blank/invalid hashes = %d, want at least blank and invalid fixtures", result.BlankOrInvalidHashes)
	}
	reportFile, err := os.ReadFile(paths.ReportPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(reportFile), "committed_legacy_only_attachments") {
		t.Fatalf("report file missing metric names:\n%s", string(reportFile))
	}
	if !strings.Contains(string(reportFile), "document_backed_blank_legacy_attachments") {
		t.Fatalf("report file missing document-backed blank legacy metric:\n%s", string(reportFile))
	}
}

func fixtureManifestRows(t *testing.T, paths Paths) map[string]ManifestRow {
	t.Helper()
	allRows, err := ReadManifest(paths.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	rows := map[string]ManifestRow{}
	for _, row := range allRows {
		switch row.ExpenseID {
		case alphaExpenseID, sharedOneID, blankHashID, existingExpenseID:
			rows[row.ExpenseID] = row
		case missingExpenseID:
			t.Fatalf("missing-file fixture should not be in manifest")
		}
	}
	return rows
}

func copyFixtureManifestObjects(t *testing.T, app *tests.TestApp, paths Paths) {
	t.Helper()
	rows := fixtureManifestRows(t, paths)
	for _, row := range rows {
		if row.CopyRequired {
			copyStorageObject(t, app, row.OldS3Key, row.NewS3Key)
		}
	}
}

func copyStorageObject(t *testing.T, app *tests.TestApp, sourceKey string, targetKey string) {
	t.Helper()
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	reader, err := fsys.GetReader(sourceKey)
	if err != nil {
		t.Fatalf("failed to read %s: %v", sourceKey, err)
	}
	defer reader.Close()
	bytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read %s: %v", sourceKey, err)
	}
	if err := fsys.Upload(bytes, targetKey); err != nil {
		t.Fatalf("failed to upload %s: %v", targetKey, err)
	}
}

func corruptStorageObject(t *testing.T, app *tests.TestApp, targetKey string, content []byte) {
	t.Helper()
	fsys, err := app.NewFilesystem()
	if err != nil {
		t.Fatal(err)
	}
	defer fsys.Close()
	if err := fsys.Upload(content, targetKey); err != nil {
		t.Fatal(err)
	}
}

func expenseAttachmentDocument(t *testing.T, app *tests.TestApp, expenseID string) string {
	t.Helper()
	var documentID string
	if err := app.DB().NewQuery("SELECT attachment_document FROM expenses WHERE id = {:id}").
		Bind(dbx.Params{"id": expenseID}).Row(&documentID); err != nil {
		t.Fatal(err)
	}
	return documentID
}

func assertExpenseLegacyFields(t *testing.T, app *tests.TestApp, expenseID string, attachment string, attachmentHash string) {
	t.Helper()
	row := struct {
		Attachment     string `db:"attachment"`
		AttachmentHash string `db:"attachment_hash"`
	}{}
	if err := app.DB().NewQuery("SELECT attachment, attachment_hash FROM expenses WHERE id = {:id}").
		Bind(dbx.Params{"id": expenseID}).One(&row); err != nil {
		t.Fatal(err)
	}
	if row.Attachment != attachment || row.AttachmentHash != attachmentHash {
		t.Fatalf("legacy fields = %s/%s, want %s/%s", row.Attachment, row.AttachmentHash, attachment, attachmentHash)
	}
}

func assertDocumentRow(t *testing.T, app *tests.TestApp, documentID string, attachment string, attachmentHash string) {
	t.Helper()
	row := documentRow{}
	if err := app.DB().NewQuery("SELECT id, attachment, attachment_hash, uploaded_by FROM expense_documents WHERE id = {:id}").
		Bind(dbx.Params{"id": documentID}).One(&row); err != nil {
		t.Fatal(err)
	}
	if row.Attachment != attachment || row.AttachmentHash != attachmentHash {
		t.Fatalf("document row = %s/%s, want %s/%s", row.Attachment, row.AttachmentHash, attachment, attachmentHash)
	}
}

func assertGeneratedID(t *testing.T, id string, hash string) {
	t.Helper()
	if id != DeterministicDocumentID(hash) {
		t.Fatalf("generated id = %s, want %s", id, DeterministicDocumentID(hash))
	}
	if !regexp.MustCompile(`^[a-z0-9]{15}$`).MatchString(id) {
		t.Fatalf("generated id %q is not 15 lowercase alphanumeric chars", id)
	}
}

type fakeS3ChecksumVerifier struct {
	checksums map[string]s3ChecksumResult
	err       error
}

func (v fakeS3ChecksumVerifier) ChecksumSHA256Hex(ctx context.Context, key string) (s3ChecksumResult, error) {
	if v.err != nil {
		return s3ChecksumResult{}, v.err
	}
	result, ok := v.checksums[key]
	if !ok {
		return s3ChecksumResult{Found: false}, nil
	}
	return result, nil
}

func withFakeS3ChecksumVerifier(t *testing.T, verifier s3ChecksumVerifier, err error) {
	t.Helper()
	original := newS3ChecksumVerifier
	newS3ChecksumVerifier = func(paths Paths) (s3ChecksumVerifier, error) {
		if err != nil {
			return nil, err
		}
		if verifier == nil {
			return nil, fmt.Errorf("fake verifier not configured")
		}
		return verifier, nil
	}
	t.Cleanup(func() {
		newS3ChecksumVerifier = original
	})
}
