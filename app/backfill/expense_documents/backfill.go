// Package expense_documents implements the Phase 2 local-first migration from
// legacy expense attachment fields to reusable expense_documents records.
//
// The migration exists because older expenses store their receipt file directly
// on expenses.attachment/expenses.attachment_hash. Newer code stores the file
// on expense_documents and points expenses.attachment_document at the reusable
// document row. Phase 2 migrates only the production rows that are committed,
// still have a legacy attachment filename, and do not already link to an
// expense_document.
//
// The package is deliberately split into prepare, verify, apply, and report
// operations because the real production procedure is delicate:
//
//   - prepare runs against a local production dump while production may still be
//     online. It does not write to the database. It creates a deterministic TSV
//     manifest, a copy-only S3 shell script, and an errors TSV. A limited
//     prepare can also emit a guarded cleanup S3 shell script for a small smoke
//     test, for example two copy-required rows, so operators can exercise
//     copy/verify/cleanup against production-like data without processing the
//     whole backlog. Full production prepare deliberately does not leave a bulk
//     delete script beside the migration artifacts.
//   - copy_s3.sh is run by an operator outside Go. It copies each legacy object
//     to the expense_documents storage key from the manifest, asking S3 to
//     compute/store SHA-256 checksum metadata for the copied destination. It
//     first preflights every destination key and aborts before copying anything
//     if any planned destination already exists. It never moves or deletes
//     source objects.
//   - verify runs after the S3 copy and compares the copied target object
//     against the manifest attachment_hash. In local mode it streams the object
//     and hashes it locally. In S3 mode it reads S3's built-in ChecksumSHA256
//     metadata with checksum-mode enabled and accepts only full-object checksums.
//     This is important because newly planned expense_documents rows do not
//     exist in the database yet, so verify must not depend on database document
//     hashes except when the manifest is deliberately reusing an already-existing
//     document row.
//   - apply runs only after production has been stopped and a fresh dump has
//     been reconciled locally. It rechecks every verified manifest row, inserts
//     missing expense_documents records using the precomputed manifest IDs, and
//     links expenses to those records in one transaction.
//
// The manifest is the contract between the slow S3 copy and the later database
// write. Each row carries:
//
//   - the expense id and legacy expense attachment filename;
//   - the hash that identifies the file contents;
//   - the expense_document id that apply must use;
//   - the exact document attachment filename and storage key that verify/apply
//     expect;
//   - whether a physical S3 copy is required;
//   - the current migration status for that row.
//
// Generated expense_document IDs are deterministic so prepare can be rerun on a
// refreshed dump without changing copy targets. They are 15 lowercase
// alphanumeric PocketBase IDs derived from:
//
//	SHA-256("expense_document:" + attachment_hash)
//
// encoded in lowercase unpadded base32 and truncated to 15 characters. The
// namespace prefix keeps these IDs distinct from any future IDs that might be
// derived from the raw hash for some other purpose.
//
// The two most important safety rules are:
//
//   - If a hash already exists in expense_documents, the manifest preserves that
//     row's existing id and attachment filename. It must not invent a new
//     document path for a row it is reusing.
//   - If an expense has a non-empty attachment_document that does not match the
//     manifest, verify/apply abort rather than overwriting it. This migration
//     may fill missing links, but it must never replace an unexpected existing
//     link.
package expense_documents

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"

	"tybalt/constants"
)

const (
	StatusReady       = "ready"
	StatusVerified    = "verified"
	StatusVerifyError = "verify_error"
	StatusApplied     = "applied"
)

var hashPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type Paths struct {
	OutDir            string
	ManifestPath      string
	CopyScriptPath    string
	CleanupScriptPath string
	ErrorsPath        string
	VerifyErrorsPath  string
	ReportPath        string
	BucketEnv         string
}

type PrepareOptions struct {
	// Limit caps the number of manifest rows written by prepare. It is intended
	// for operator smoke tests, not for the final production apply. A zero or
	// negative value means no limit.
	Limit int
	// RequireCopy excludes rows whose destination object already exists and
	// matches the manifest hash. Combined with Limit, this ensures a test
	// manifest contains rows that will actually appear in copy_s3.sh and
	// the smoke-only cleanup_s3.sh.
	RequireCopy bool
	// S3ChecksumReportPaths optionally points at one or more locally downloaded
	// S3 Batch Operations Compute checksum completion-report manifest.json files.
	// The bucket named by Paths.BucketEnv must be set. Prepare trusts only rows
	// for that bucket whose SHA256/FULL_OBJECT checksum report ETag still
	// matches current S3 HeadObject metadata; all other rows fall back to local
	// reads.
	S3ChecksumReportPaths []string
}

type ReportOptions struct {
	// S3ChecksumReportPaths has the same meaning as PrepareOptions. It lets the
	// baseline report count missing storage objects by checking current S3 ETag
	// metadata against AWS's checksum report before falling back to local reads.
	S3ChecksumReportPaths []string
}

type ChecksumMode string

const (
	// ChecksumModeLocal streams destination objects through PocketBase storage
	// and computes SHA-256 locally. This is slow but requires no direct S3
	// configuration and remains the package-level default for tests/callers.
	ChecksumModeLocal ChecksumMode = "local"
	// ChecksumModeS3 requires S3's stored full-object SHA-256 checksum metadata.
	// It fails rows whose destination object has no SHA-256 checksum or reports
	// a multipart/composite checksum.
	ChecksumModeS3 ChecksumMode = "s3"
	// ChecksumModeAuto uses S3 checksum metadata when direct S3 configuration is
	// available, and falls back to local hashing when metadata is unavailable.
	ChecksumModeAuto ChecksumMode = "auto"
)

type VerifyOptions struct {
	ChecksumMode ChecksumMode
	// Statuses limits verify to manifest rows whose current status is in this
	// set. It exists for operational retries, for example rerunning only rows
	// marked verify_error with checksum-mode=auto after a strict S3 checksum
	// pass found a small tail of unsupported checksum metadata. An empty set
	// verifies every row.
	Statuses []string
}

type ManifestRow struct {
	ExpenseID          string
	ExpenseAttachment  string
	AttachmentHash     string
	ExpenseDocumentID  string
	DocumentAttachment string
	UploadedBy         string
	OldS3Key           string
	NewS3Key           string
	CopyRequired       bool
	Status             string
}

type ErrorRow struct {
	ExpenseID string
	Stage     string
	Code      string
	Message   string
}

type PrepareResult struct {
	ManifestRows    int
	Errors          int
	CopyCommands    int
	CleanupCommands int
}

type VerifyResult struct {
	Verified int
	Failed   int
}

type ApplyResult struct {
	InsertedDocuments int
	LinkedExpenses    int
	SkippedExpenses   int
}

type ReportResult struct {
	LegacyAttachments              int
	DocumentBackedAttachments      int
	CommittedLegacyOnlyAttachments int
	DocumentBackedBlankLegacy      int
	DocumentBackedMissingTargets   int
	DuplicateDocumentReferences    int
	MissingLegacyFiles             int
	BlankOrInvalidHashes           int
}

type expenseCandidate struct {
	ID             string `db:"id"`
	Attachment     string `db:"attachment"`
	AttachmentHash string `db:"attachment_hash"`
	Creator        string `db:"creator"`
	UID            string `db:"uid"`
}

type documentRow struct {
	ID             string `db:"id"`
	Attachment     string `db:"attachment"`
	AttachmentHash string `db:"attachment_hash"`
	UploadedBy     string `db:"uploaded_by"`
}

type expenseState struct {
	ID                 string `db:"id"`
	Attachment         string `db:"attachment"`
	AttachmentHash     string `db:"attachment_hash"`
	AttachmentDocument string `db:"attachment_document"`
	Committed          string `db:"committed"`
}

type s3ChecksumConfig struct {
	bucket   string
	region   string
	endpoint string
}

type s3ChecksumResult struct {
	HexChecksum string
	Found       bool
	FullObject  bool
}

type s3ObjectID struct {
	Bucket string
	Key    string
}

type s3ReportMetadataReader interface {
	ETag(ctx context.Context, bucket string, key string) (string, error)
}

type s3ChecksumVerifier interface {
	ChecksumSHA256Hex(ctx context.Context, key string) (s3ChecksumResult, error)
}

type defaultS3ChecksumVerifier struct {
	bucket string
	client *s3.Client
}

type defaultS3ReportMetadataReader struct {
	client *s3.Client
}

var newS3ChecksumVerifier = defaultNewS3ChecksumVerifier
var newS3ReportMetadataReader = defaultNewS3ReportMetadataReader

func DefaultPaths(outDir string) Paths {
	if strings.TrimSpace(outDir) == "" {
		outDir = filepath.Join("tmp", "expense_document_backfill")
	}
	return Paths{
		OutDir:            outDir,
		ManifestPath:      filepath.Join(outDir, "manifest.tsv"),
		CopyScriptPath:    filepath.Join(outDir, "copy_s3.sh"),
		CleanupScriptPath: filepath.Join(outDir, "cleanup_s3.sh"),
		ErrorsPath:        filepath.Join(outDir, "errors.tsv"),
		VerifyErrorsPath:  filepath.Join(outDir, "verify_errors.tsv"),
		ReportPath:        filepath.Join(outDir, "report.tsv"),
		BucketEnv:         "TYBALT_S3_BUCKET",
	}
}

func DeterministicDocumentID(attachmentHash string) string {
	// Do not use the raw hash prefix as the PocketBase id. The namespace makes
	// the id specifically belong to this migration's expense_document identity,
	// while base32 gives us lowercase alphanumeric characters after lowercasing.
	sum := sha256.Sum256([]byte("expense_document:" + strings.ToLower(strings.TrimSpace(attachmentHash))))
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	return strings.ToLower(encoded[:15])
}

func Prepare(app core.App, paths Paths) (PrepareResult, error) {
	return PrepareWithOptions(app, paths, PrepareOptions{})
}

func PrepareWithOptions(app core.App, paths Paths, options PrepareOptions) (PrepareResult, error) {
	paths = normalizePaths(paths)
	if err := os.MkdirAll(paths.OutDir, 0o755); err != nil {
		return PrepareResult{}, err
	}

	expensesCollection, err := app.FindCollectionByNameOrId("expenses")
	if err != nil {
		return PrepareResult{}, err
	}
	documentsCollection, err := app.FindCollectionByNameOrId(constants.ExpenseDocumentsCollectionName)
	if err != nil {
		return PrepareResult{}, err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return PrepareResult{}, err
	}
	defer fsys.Close()
	hashSource, err := newStorageHashSource(paths, options.S3ChecksumReportPaths, fsys)
	if err != nil {
		return PrepareResult{}, err
	}

	candidates, err := committedLegacyOnlyExpenses(app)
	if err != nil {
		return PrepareResult{}, err
	}

	rows := make([]ManifestRow, 0, len(candidates))
	errorsOut := []ErrorRow{}
	// The database enforces unique expense attachment hashes, but keeping this
	// in-memory map makes the manifest builder robust if a local dump ever
	// contains duplicate content rows during investigation or after constraints
	// change. The tuple-level manifest identity still remains per expense row.
	byHash := map[string]ManifestRow{}

	for i, expense := range candidates {
		if i > 0 && i%1000 == 0 {
			fmt.Fprintf(os.Stderr, "processed %d/%d candidate expenses; prepared %d manifest rows and %d errors...\n", i, len(candidates), len(rows), len(errorsOut))
		}
		oldKey := storageKey(expensesCollection, expense.ID, expense.Attachment)
		attachmentHash := strings.ToLower(strings.TrimSpace(expense.AttachmentHash))
		if attachmentHash == "" {
			// Older rows may not have attachment_hash populated. In that case the
			// source file is the source of truth, so prepare hashes the legacy
			// object before deciding the document id and target key.
			attachmentHash, err = hashSource.Hash(oldKey)
			if err != nil {
				errorsOut = append(errorsOut, errorRow(expense.ID, "prepare", "missing_legacy_file", err))
				continue
			}
		} else if sourceHash, err := hashSource.Hash(oldKey); err != nil {
			errorsOut = append(errorsOut, errorRow(expense.ID, "prepare", "missing_legacy_file", err))
			continue
		} else if sourceHash != attachmentHash {
			// A manifest row is useful only if the file we are about to copy
			// really matches the hash that will identify/reuse the document.
			errorsOut = append(errorsOut, ErrorRow{
				ExpenseID: expense.ID,
				Stage:     "prepare",
				Code:      "legacy_hash_mismatch",
				Message:   fmt.Sprintf("source %s has hash %s, expected %s", oldKey, sourceHash, attachmentHash),
			})
			continue
		}
		if !hashPattern.MatchString(attachmentHash) {
			errorsOut = append(errorsOut, ErrorRow{
				ExpenseID: expense.ID,
				Stage:     "prepare",
				Code:      "invalid_hash",
				Message:   fmt.Sprintf("attachment_hash must be 64 lowercase hex chars: %q", attachmentHash),
			})
			continue
		}

		docID := ""
		documentAttachment := ""
		uploadedBy := firstNonEmpty(expense.Creator, expense.UID)
		if existing, err := findDocumentByHash(app, attachmentHash); err != nil {
			return PrepareResult{}, err
		} else if existing != nil {
			// Critical: an existing expense_documents row already defines both
			// the document id and the attachment filename/storage key. Reusing
			// only the id while keeping the legacy expense filename would point
			// verify/apply at a path that does not belong to the reused document.
			docID = existing.ID
			documentAttachment = existing.Attachment
			uploadedBy = firstNonEmpty(existing.UploadedBy, uploadedBy)
		} else if previous, ok := byHash[attachmentHash]; ok {
			// If duplicate hashes appear in a dump, all duplicates should point
			// to the same precomputed document row and therefore the same target
			// object. The first manifest row for the hash owns the filename.
			docID = previous.ExpenseDocumentID
			documentAttachment = previous.DocumentAttachment
			uploadedBy = previous.UploadedBy
		} else {
			docID = DeterministicDocumentID(attachmentHash)
			if existing, err := findDocumentByID(app, docID); err != nil {
				return PrepareResult{}, err
			} else if existing != nil && existing.AttachmentHash != attachmentHash {
				errorsOut = append(errorsOut, ErrorRow{
					ExpenseID: expense.ID,
					Stage:     "prepare",
					Code:      "document_id_collision",
					Message:   fmt.Sprintf("generated document id %s already belongs to hash %s", docID, existing.AttachmentHash),
				})
				continue
			}
			documentAttachment = expense.Attachment
		}

		newKey := storageKey(documentsCollection, docID, documentAttachment)
		status := StatusReady
		copyRequired := true
		if targetHash, err := hashSource.Hash(newKey); err == nil {
			// If the target already exists and matches, prepare can mark the row
			// verified and omit it from copy_s3.sh. If it exists with different
			// bytes, we stop instead of overwriting a possibly unrelated object.
			if targetHash != attachmentHash {
				errorsOut = append(errorsOut, ErrorRow{
					ExpenseID: expense.ID,
					Stage:     "prepare",
					Code:      "target_hash_mismatch",
					Message:   fmt.Sprintf("target %s has hash %s, expected %s", newKey, targetHash, attachmentHash),
				})
				continue
			}
			status = StatusVerified
			copyRequired = false
		} else if !isNotFound(err) {
			errorsOut = append(errorsOut, errorRow(expense.ID, "prepare", "target_read_error", err))
			continue
		}

		row := ManifestRow{
			ExpenseID:          expense.ID,
			ExpenseAttachment:  expense.Attachment,
			AttachmentHash:     attachmentHash,
			ExpenseDocumentID:  docID,
			DocumentAttachment: documentAttachment,
			UploadedBy:         uploadedBy,
			OldS3Key:           oldKey,
			NewS3Key:           newKey,
			CopyRequired:       copyRequired,
			Status:             status,
		}
		if options.RequireCopy && !row.CopyRequired {
			continue
		}
		rows = append(rows, row)
		if _, ok := byHash[attachmentHash]; !ok {
			byHash[attachmentHash] = row
		}
		if options.Limit > 0 && len(rows) >= options.Limit {
			// A limited prepare is for pre-apply smoke tests only. We stop after
			// writing a complete, internally consistent prefix of candidate rows;
			// the final migration should rerun prepare without --limit.
			break
		}
	}

	if err := WriteManifest(paths.ManifestPath, rows); err != nil {
		return PrepareResult{}, err
	}
	if err := writeErrors(paths.ErrorsPath, errorsOut); err != nil {
		return PrepareResult{}, err
	}
	copyCommands, err := writeCopyScript(paths.CopyScriptPath, paths.BucketEnv, rows)
	if err != nil {
		return PrepareResult{}, err
	}
	cleanupCommands, err := writeCleanupScript(paths.CleanupScriptPath, paths.BucketEnv, rows, options.Limit > 0)
	if err != nil {
		return PrepareResult{}, err
	}
	return PrepareResult{ManifestRows: len(rows), Errors: len(errorsOut), CopyCommands: copyCommands, CleanupCommands: cleanupCommands}, nil
}

func Verify(app core.App, paths Paths) (VerifyResult, error) {
	return VerifyWithOptions(app, paths, VerifyOptions{ChecksumMode: ChecksumModeLocal})
}

func VerifyWithOptions(app core.App, paths Paths, options VerifyOptions) (VerifyResult, error) {
	paths = normalizePaths(paths)
	rows, err := ReadManifest(paths.ManifestPath)
	if err != nil {
		return VerifyResult{}, err
	}
	options = normalizeVerifyOptions(options)
	verifier, err := checksumVerifierForMode(paths, options.ChecksumMode)
	if err != nil {
		return VerifyResult{}, err
	}
	statusFilter := verifyStatusFilter(options.Statuses)

	fsys, err := app.NewFilesystem()
	if err != nil {
		return VerifyResult{}, err
	}
	defer fsys.Close()

	errorsOut := []ErrorRow{}
	verified := 0
	total := len(rows)
	filteredTotal := verifyFilteredRowCount(rows, statusFilter)
	if total > 0 {
		if len(statusFilter) > 0 {
			fmt.Fprintf(os.Stderr, "verifying %d/%d manifest rows with checksum_mode=%s and status filter %s...\n", filteredTotal, total, options.ChecksumMode, strings.Join(options.Statuses, ","))
		} else {
			fmt.Fprintf(os.Stderr, "verifying %d manifest rows with checksum_mode=%s...\n", total, options.ChecksumMode)
		}
	}
	lastProgressLog := time.Now()
	processed := 0
	for i := range rows {
		if len(statusFilter) > 0 {
			if _, ok := statusFilter[rows[i].Status]; !ok {
				if rows[i].Status == StatusVerified || rows[i].Status == StatusApplied {
					verified++
				}
				continue
			}
		}

		// Verification is intentionally row-local and repeatable. A failed row
		// does not prevent other rows from being checked, but apply later
		// requires every manifest row to be verified/applied.
		rowErrors := verifyManifestRow(app, fsys, verifier, options.ChecksumMode, rows[i])
		if len(rowErrors) == 0 {
			rows[i].Status = StatusVerified
			verified++
		} else {
			rows[i].Status = StatusVerifyError
			errorsOut = append(errorsOut, rowErrors...)
		}

		processed++
		if processed == filteredTotal || processed%250 == 0 || time.Since(lastProgressLog) >= 30*time.Second {
			if len(statusFilter) > 0 {
				fmt.Fprintf(os.Stderr, "verified progress %d/%d filtered rows; total passed %d total failed %d...\n", processed, filteredTotal, verified, total-verified)
			} else {
				fmt.Fprintf(os.Stderr, "verified progress %d/%d rows; passed %d failed %d...\n", processed, total, verified, processed-verified)
			}
			lastProgressLog = time.Now()
		}
	}

	if err := WriteManifest(paths.ManifestPath, rows); err != nil {
		return VerifyResult{}, err
	}
	if err := writeErrors(paths.VerifyErrorsPath, errorsOut); err != nil {
		return VerifyResult{}, err
	}
	return VerifyResult{Verified: verified, Failed: len(rows) - verified}, nil
}

func verifyStatusFilter(statuses []string) map[string]struct{} {
	filter := map[string]struct{}{}
	for _, status := range statuses {
		status = strings.TrimSpace(status)
		if status != "" {
			filter[status] = struct{}{}
		}
	}
	return filter
}

func verifyFilteredRowCount(rows []ManifestRow, statusFilter map[string]struct{}) int {
	if len(statusFilter) == 0 {
		return len(rows)
	}
	total := 0
	for _, row := range rows {
		if _, ok := statusFilter[row.Status]; ok {
			total++
		}
	}
	return total
}

func Apply(app core.App, paths Paths) (ApplyResult, error) {
	return ApplyWithOptions(app, paths, VerifyOptions{ChecksumMode: ChecksumModeLocal})
}

func ApplyWithOptions(app core.App, paths Paths, options VerifyOptions) (ApplyResult, error) {
	paths = normalizePaths(paths)
	rows, err := ReadManifest(paths.ManifestPath)
	if err != nil {
		return ApplyResult{}, err
	}
	options = normalizeVerifyOptions(options)
	verifier, err := checksumVerifierForMode(paths, options.ChecksumMode)
	if err != nil {
		return ApplyResult{}, err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return ApplyResult{}, err
	}
	defer fsys.Close()

	total := len(rows)
	if total > 0 {
		fmt.Fprintf(os.Stderr, "apply preflight checking %d manifest rows with checksum_mode=%s...\n", total, options.ChecksumMode)
	}
	lastProgressLog := time.Now()
	for i, row := range rows {
		// Apply repeats the verify checks instead of trusting yesterday's
		// manifest status. The operator may have refreshed the DB dump, copied
		// more files, or discovered drift after verify was last run.
		if row.Status != StatusVerified && row.Status != StatusApplied {
			return ApplyResult{}, fmt.Errorf("manifest row for expense %s has status %q; run verify before apply", row.ExpenseID, row.Status)
		}
		if errs := verifyManifestRow(app, fsys, verifier, options.ChecksumMode, row); len(errs) > 0 {
			return ApplyResult{}, fmt.Errorf("manifest row for expense %s failed apply preflight: %s", row.ExpenseID, errs[0].Message)
		}

		processed := i + 1
		if processed == total || processed%250 == 0 || time.Since(lastProgressLog) >= 30*time.Second {
			fmt.Fprintf(os.Stderr, "apply preflight progress %d/%d rows...\n", processed, total)
			lastProgressLog = time.Now()
		}
	}
	fmt.Fprintf(os.Stderr, "apply preflight complete; starting database transaction for %d rows...\n", total)

	result := ApplyResult{}
	err = app.RunInTransaction(func(txApp core.App) error {
		lastTransactionLog := time.Now()
		for i, row := range rows {
			// The whole batch is all-or-nothing. Any unexpected expense state,
			// document hash conflict, or insert/update failure aborts the
			// transaction so production cannot be left half-linked.
			expense, err := findExpenseState(txApp, row.ExpenseID)
			if err != nil {
				return err
			}
			if strings.TrimSpace(expense.AttachmentDocument) != "" {
				// Idempotency is allowed only for the exact expected link. A
				// different document means the stopped production dump no longer
				// matches the manifest contract.
				if expense.AttachmentDocument != row.ExpenseDocumentID {
					return fmt.Errorf("expense %s already links to %s, expected %s", row.ExpenseID, expense.AttachmentDocument, row.ExpenseDocumentID)
				}
				result.SkippedExpenses++
				continue
			}

			document, err := findDocumentByHash(txApp, row.AttachmentHash)
			if err != nil {
				return err
			}
			if document != nil {
				// A hash match means the document already exists, either from a
				// previous successful apply attempt or from a row that existed
				// before prepare. In both cases the manifest must agree with the
				// persisted id and attachment path.
				if document.ID != row.ExpenseDocumentID {
					return fmt.Errorf("hash %s already belongs to document %s, expected %s", row.AttachmentHash, document.ID, row.ExpenseDocumentID)
				}
				if document.Attachment != row.DocumentAttachment {
					return fmt.Errorf("document %s attachment is %s, expected %s", row.ExpenseDocumentID, document.Attachment, row.DocumentAttachment)
				}
			} else {
				if byID, err := findDocumentByID(txApp, row.ExpenseDocumentID); err != nil {
					return err
				} else if byID != nil {
					return fmt.Errorf("document id %s already exists with hash %s", row.ExpenseDocumentID, byID.AttachmentHash)
				}
				now := types.NowDateTime()
				if _, err := txApp.DB().Insert(constants.ExpenseDocumentsCollectionName, dbx.Params{
					"id":              row.ExpenseDocumentID,
					"attachment":      row.DocumentAttachment,
					"attachment_hash": row.AttachmentHash,
					"uploaded_by":     row.UploadedBy,
					"created":         now,
					"updated":         now,
				}).Execute(); err != nil {
					return err
				}
				result.InsertedDocuments++
			}

			// Leave expenses.attachment and expenses.attachment_hash in place as
			// legacy audit/source fields. Phase 2 only fills the new relation.
			if _, err := txApp.DB().NewQuery("UPDATE expenses SET attachment_document = {:document}, updated = {:updated} WHERE id = {:id}").
				Bind(dbx.Params{
					"document": row.ExpenseDocumentID,
					"updated":  types.NowDateTime(),
					"id":       row.ExpenseID,
				}).Execute(); err != nil {
				return err
			}
			result.LinkedExpenses++

			processed := i + 1
			if processed == total || processed%500 == 0 || time.Since(lastTransactionLog) >= 10*time.Second {
				fmt.Fprintf(os.Stderr, "apply transaction progress %d/%d rows; inserted_documents=%d linked_expenses=%d skipped_expenses=%d...\n", processed, total, result.InsertedDocuments, result.LinkedExpenses, result.SkippedExpenses)
				lastTransactionLog = time.Now()
			}
		}
		return nil
	})
	if err != nil {
		return ApplyResult{}, err
	}
	fmt.Fprintf(os.Stderr, "apply transaction committed; writing applied manifest statuses...\n")

	for i := range rows {
		rows[i].Status = StatusApplied
	}
	if err := WriteManifest(paths.ManifestPath, rows); err != nil {
		return ApplyResult{}, err
	}
	return result, nil
}

func Report(app core.App, paths Paths) (ReportResult, error) {
	return ReportWithOptions(app, paths, ReportOptions{})
}

func ReportWithOptions(app core.App, paths Paths, options ReportOptions) (ReportResult, error) {
	paths = normalizePaths(paths)
	result := ReportResult{}
	fmt.Fprintln(os.Stderr, "collecting database attachment baseline counts...")
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM expenses WHERE attachment != ''").
		Row(&result.LegacyAttachments); err != nil {
		return ReportResult{}, err
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM expenses WHERE attachment_document != ''").
		Row(&result.DocumentBackedAttachments); err != nil {
		return ReportResult{}, err
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM expenses WHERE attachment != '' AND (attachment_document = '' OR attachment_document IS NULL) AND committed != ''").
		Row(&result.CommittedLegacyOnlyAttachments); err != nil {
		return ReportResult{}, err
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM expenses WHERE attachment_document != '' AND (attachment = '' OR attachment IS NULL)").
		Row(&result.DocumentBackedBlankLegacy); err != nil {
		return ReportResult{}, err
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM (SELECT attachment_document FROM expenses WHERE attachment_document != '' GROUP BY attachment_document HAVING COUNT(*) > 1)").
		Row(&result.DuplicateDocumentReferences); err != nil {
		return ReportResult{}, err
	}
	fmt.Fprintf(os.Stderr, "database counts collected: %d legacy attachments, %d document-backed attachments, %d committed legacy-only attachments\n", result.LegacyAttachments, result.DocumentBackedAttachments, result.CommittedLegacyOnlyAttachments)

	fsys, err := app.NewFilesystem()
	if err != nil {
		return ReportResult{}, err
	}
	defer fsys.Close()
	hashSource, err := newStorageHashSource(paths, options.S3ChecksumReportPaths, fsys)
	if err != nil {
		return ReportResult{}, err
	}

	if result.DocumentBackedMissingTargets, err = countMissingDocumentTargets(app, hashSource); err != nil {
		return ReportResult{}, err
	}
	if result.MissingLegacyFiles, err = countMissingLegacyFiles(app, hashSource); err != nil {
		return ReportResult{}, err
	}
	if result.BlankOrInvalidHashes, err = countBlankOrInvalidLegacyHashes(app); err != nil {
		return ReportResult{}, err
	}

	if err := writeReport(paths.ReportPath, result); err != nil {
		return ReportResult{}, err
	}
	return result, nil
}

func ReadManifest(path string) ([]ManifestRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	rows := make([]ManifestRow, 0, len(records)-1)
	for i, record := range records[1:] {
		if len(record) != 10 {
			return nil, fmt.Errorf("manifest row %d has %d fields, expected 10", i+2, len(record))
		}
		copyRequired, err := strconv.ParseBool(record[8])
		if err != nil {
			return nil, fmt.Errorf("manifest row %d has invalid copy_required: %w", i+2, err)
		}
		rows = append(rows, ManifestRow{
			ExpenseID:          record[0],
			ExpenseAttachment:  record[1],
			AttachmentHash:     record[2],
			ExpenseDocumentID:  record[3],
			DocumentAttachment: record[4],
			UploadedBy:         record[5],
			OldS3Key:           record[6],
			NewS3Key:           record[7],
			CopyRequired:       copyRequired,
			Status:             record[9],
		})
	}
	return rows, nil
}

func WriteManifest(path string, rows []ManifestRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	if err := writer.Write([]string{
		"expense_id",
		"expense_attachment",
		"attachment_hash",
		"expense_document_id",
		"document_attachment",
		"uploaded_by",
		"old_s3_key",
		"new_s3_key",
		"copy_required",
		"status",
	}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{
			row.ExpenseID,
			row.ExpenseAttachment,
			row.AttachmentHash,
			row.ExpenseDocumentID,
			row.DocumentAttachment,
			row.UploadedBy,
			row.OldS3Key,
			row.NewS3Key,
			strconv.FormatBool(row.CopyRequired),
			row.Status,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func committedLegacyOnlyExpenses(app core.App) ([]expenseCandidate, error) {
	rows := []expenseCandidate{}
	err := app.DB().NewQuery(`
		SELECT id, attachment, attachment_hash, creator, uid
		FROM expenses
		WHERE attachment != ''
			AND (attachment_document = '' OR attachment_document IS NULL)
			AND committed != ''
		ORDER BY id
	`).All(&rows)
	return rows, err
}

func findDocumentByHash(app core.App, attachmentHash string) (*documentRow, error) {
	rows := []documentRow{}
	err := app.DB().NewQuery(`
		SELECT id, attachment, attachment_hash, uploaded_by
		FROM expense_documents
		WHERE attachment_hash = {:hash}
		ORDER BY id
	`).Bind(dbx.Params{"hash": attachmentHash}).All(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func findDocumentByID(app core.App, id string) (*documentRow, error) {
	row := documentRow{}
	err := app.DB().NewQuery(`
		SELECT id, attachment, attachment_hash, uploaded_by
		FROM expense_documents
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": id}).One(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func findExpenseState(app core.App, id string) (*expenseState, error) {
	row := expenseState{}
	err := app.DB().NewQuery(`
		SELECT id, attachment, attachment_hash, attachment_document, committed
		FROM expenses
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": id}).One(&row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("expense %s not found", id)
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func verifyManifestRow(app core.App, fsys *filesystem.System, verifier s3ChecksumVerifier, checksumMode ChecksumMode, row ManifestRow) []ErrorRow {
	errs := []ErrorRow{}
	if !hashPattern.MatchString(row.AttachmentHash) {
		errs = append(errs, ErrorRow{ExpenseID: row.ExpenseID, Stage: "verify", Code: "invalid_manifest_hash", Message: "manifest attachment_hash is not 64 lowercase hex chars"})
	}
	// The copied target is checked against the manifest hash, not against an
	// expense_documents row. For newly planned documents there is no row yet;
	// for reused documents, this still proves the actual storage object matches
	// the manifest contract before apply links expenses to it.
	if targetHash, code, err := targetHashForVerify(fsys, verifier, checksumMode, row); err != nil {
		errs = append(errs, errorRow(row.ExpenseID, "verify", code, err))
	} else if targetHash != row.AttachmentHash {
		errs = append(errs, ErrorRow{
			ExpenseID: row.ExpenseID,
			Stage:     "verify",
			Code:      "copied_hash_mismatch",
			Message:   fmt.Sprintf("target %s has hash %s, expected %s", row.NewS3Key, targetHash, row.AttachmentHash),
		})
	}
	// Source objects should still exist because the S3 operation is copy-only.
	// A missing source does not mean the target is bad, but it violates the
	// migration safety rule and should be investigated before apply.
	if err := storageObjectExists(fsys, row.OldS3Key); err != nil {
		errs = append(errs, errorRow(row.ExpenseID, "verify", "missing_legacy_object", err))
	}

	expense, err := findExpenseState(app, row.ExpenseID)
	if err != nil {
		errs = append(errs, errorRow(row.ExpenseID, "verify", "missing_expense", err))
	} else {
		errs = append(errs, verifyExpenseStateAgainstManifest(expense, row)...)
	}

	document, err := findDocumentByHash(app, row.AttachmentHash)
	if err != nil {
		errs = append(errs, errorRow(row.ExpenseID, "verify", "document_lookup_error", err))
	} else if document != nil {
		// This DB check is intentionally conditional. It catches an existing
		// hash that maps to a different document or path, but verify does not
		// require a document row for hashes that apply will insert later.
		if document.ID != row.ExpenseDocumentID {
			errs = append(errs, ErrorRow{
				ExpenseID: row.ExpenseID,
				Stage:     "verify",
				Code:      "hash_mapped_to_different_document",
				Message:   fmt.Sprintf("hash belongs to document %s, expected %s", document.ID, row.ExpenseDocumentID),
			})
		}
		if document.AttachmentHash != row.AttachmentHash || document.Attachment != row.DocumentAttachment {
			errs = append(errs, ErrorRow{
				ExpenseID: row.ExpenseID,
				Stage:     "verify",
				Code:      "existing_document_mismatch",
				Message:   fmt.Sprintf("document %s does not match manifest attachment/hash", row.ExpenseDocumentID),
			})
		}
	}
	return errs
}

func verifyExpenseStateAgainstManifest(expense *expenseState, row ManifestRow) []ErrorRow {
	errs := []ErrorRow{}
	linkedDocument := strings.TrimSpace(expense.AttachmentDocument)
	if linkedDocument != "" {
		if linkedDocument != row.ExpenseDocumentID {
			errs = append(errs, ErrorRow{
				ExpenseID: row.ExpenseID,
				Stage:     "verify",
				Code:      "unexpected_existing_link",
				Message:   fmt.Sprintf("expense links to %s, expected %s", linkedDocument, row.ExpenseDocumentID),
			})
		}
		return errs
	}

	// The manifest was prepared from committed legacy-only rows. Rechecking the
	// current expense fields during verify/apply catches a refreshed dump that no
	// longer describes the same source expense, without any extra storage reads.
	if strings.TrimSpace(expense.Committed) == "" {
		errs = append(errs, ErrorRow{
			ExpenseID: row.ExpenseID,
			Stage:     "verify",
			Code:      "expense_no_longer_committed",
			Message:   "expense is no longer committed",
		})
	}
	if strings.TrimSpace(expense.Attachment) != row.ExpenseAttachment {
		errs = append(errs, ErrorRow{
			ExpenseID: row.ExpenseID,
			Stage:     "verify",
			Code:      "expense_attachment_mismatch",
			Message:   fmt.Sprintf("expense attachment is %q, expected %q", expense.Attachment, row.ExpenseAttachment),
		})
	}
	currentHash := strings.ToLower(strings.TrimSpace(expense.AttachmentHash))
	if currentHash != "" && currentHash != row.AttachmentHash {
		errs = append(errs, ErrorRow{
			ExpenseID: row.ExpenseID,
			Stage:     "verify",
			Code:      "expense_hash_mismatch",
			Message:   fmt.Sprintf("expense attachment_hash is %s, expected %s", currentHash, row.AttachmentHash),
		})
	}
	return errs
}

func targetHashForVerify(fsys *filesystem.System, verifier s3ChecksumVerifier, checksumMode ChecksumMode, row ManifestRow) (string, string, error) {
	switch checksumMode {
	case ChecksumModeLocal:
		hash, err := hashStorageObject(fsys, row.NewS3Key)
		return hash, "missing_copied_object", err
	case ChecksumModeS3:
		if !row.CopyRequired {
			// Strict S3 mode proves that objects created by copy_s3.sh expose the
			// full-object SHA-256 checksum metadata the script requested. Reused
			// existing expense_documents objects may predate that copy script, so
			// their missing S3 checksum metadata is not a migration failure. They
			// still must be byte-verified against the manifest hash, just via a
			// local stream/hash of the already-present object.
			hash, err := hashStorageObject(fsys, row.NewS3Key)
			return hash, "missing_copied_object", err
		}
		hash, code, err := targetHashFromS3Checksum(verifier, row.NewS3Key, false)
		if err != nil {
			return "", code, err
		}
		if err := storageObjectExists(fsys, row.NewS3Key); err != nil {
			return "", "app_storage_unreadable", err
		}
		return hash, "", nil
	case ChecksumModeAuto:
		if verifier == nil {
			hash, err := hashStorageObject(fsys, row.NewS3Key)
			return hash, "missing_copied_object", err
		}
		hash, code, err := targetHashFromS3Checksum(verifier, row.NewS3Key, true)
		if err == nil {
			if err := storageObjectExists(fsys, row.NewS3Key); err != nil {
				return "", "app_storage_unreadable", err
			}
			return hash, "", nil
		}
		if code != "s3_checksum_unavailable" && code != "s3_checksum_not_full_object" {
			hash, localErr := hashStorageObject(fsys, row.NewS3Key)
			if localErr == nil {
				return hash, "", nil
			}
			return "", code, err
		}
		hash, localErr := hashStorageObject(fsys, row.NewS3Key)
		if localErr == nil {
			return hash, "", nil
		}
		return "", code, err
	default:
		return "", "invalid_checksum_mode", fmt.Errorf("unknown checksum mode %q", checksumMode)
	}
}

func targetHashFromS3Checksum(verifier s3ChecksumVerifier, key string, allowUnavailable bool) (string, string, error) {
	if verifier == nil {
		if allowUnavailable {
			return "", "s3_checksum_unavailable", fmt.Errorf("S3 checksum verifier is not configured")
		}
		return "", "s3_checksum_unavailable", fmt.Errorf("S3 checksum verifier is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := verifier.ChecksumSHA256Hex(ctx, key)
	if err != nil {
		return "", "s3_checksum_error", err
	}
	if !result.Found {
		return "", "s3_checksum_unavailable", fmt.Errorf("target %s does not expose ChecksumSHA256 metadata", key)
	}
	if !result.FullObject {
		return "", "s3_checksum_not_full_object", fmt.Errorf("target %s exposes a multipart/composite SHA-256 checksum, not a full-object checksum", key)
	}
	return result.HexChecksum, "", nil
}

func hashStorageObject(fsys *filesystem.System, key string) (string, error) {
	reader, err := fsys.GetReader(key)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func storageObjectExists(fsys *filesystem.System, key string) error {
	reader, err := fsys.GetReader(key)
	if err != nil {
		return err
	}
	return reader.Close()
}

type storageHashSource struct {
	fsys   *filesystem.System
	report *s3ChecksumReportIndex
}

func newStorageHashSource(paths Paths, reportPaths []string, fsys *filesystem.System) (*storageHashSource, error) {
	source := &storageHashSource{fsys: fsys}
	report, err := loadS3ChecksumReports(paths, reportPaths)
	if err != nil {
		return nil, err
	}
	source.report = report
	return source, nil
}

func (source *storageHashSource) Hash(key string) (string, error) {
	if source.report != nil {
		if hash, ok := source.report.Hash(key); ok {
			return hash, nil
		}
	}
	return hashStorageObject(source.fsys, key)
}

func (source *storageHashSource) Exists(key string) error {
	if source.report != nil {
		if _, ok := source.report.Hash(key); ok {
			return nil
		}
	}
	return storageObjectExists(source.fsys, key)
}

type s3ChecksumReportIndex struct {
	entriesByKey map[string][]s3ChecksumReportEntry
	reader       s3ReportMetadataReader
	preferred    string
}

type s3ChecksumReportEntry struct {
	Bucket      string
	Key         string
	ETag        string
	ChecksumHex string
}

type s3ChecksumReportManifest struct {
	Results []s3ChecksumReportResult `json:"Results"`
}

type s3ChecksumReportResult struct {
	Key string `json:"Key"`
}

const s3ChecksumReportColumns = 7

func loadS3ChecksumReports(paths Paths, reportPaths []string) (*s3ChecksumReportIndex, error) {
	reportPaths = cleanReportPaths(reportPaths)
	if len(reportPaths) == 0 {
		return nil, nil
	}
	bucket := strings.TrimSpace(os.Getenv(paths.BucketEnv))
	if bucket == "" {
		return nil, fmt.Errorf("set %s before using --s3-checksum-report", paths.BucketEnv)
	}
	reader, err := newS3ReportMetadataReader(paths)
	if err != nil {
		return nil, err
	}
	index := &s3ChecksumReportIndex{
		entriesByKey: map[string][]s3ChecksumReportEntry{},
		reader:       reader,
		preferred:    bucket,
	}
	for _, reportPath := range reportPaths {
		fmt.Fprintf(os.Stderr, "loading S3 checksum report bundle %s...\n", reportPath)
		if err := index.loadManifest(reportPath); err != nil {
			return nil, err
		}
	}
	index.retainCurrentEntries()
	return index, nil
}

func cleanReportPaths(paths []string) []string {
	cleaned := []string{}
	seen := map[string]struct{}{}
	for _, value := range paths {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}

func (index *s3ChecksumReportIndex) loadManifest(manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read S3 checksum report manifest %s: %w", manifestPath, err)
	}
	manifest := s3ChecksumReportManifest{}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parse S3 checksum report manifest %s: %w", manifestPath, err)
	}
	reportFiles, err := resolveS3ChecksumReportFiles(filepath.Dir(manifestPath), manifest.Results)
	if err != nil {
		return err
	}
	for _, reportFile := range reportFiles {
		if err := index.loadCSV(reportFile); err != nil {
			return err
		}
	}
	return nil
}

func resolveS3ChecksumReportFiles(manifestDir string, results []s3ChecksumReportResult) ([]string, error) {
	reportFiles := []string{}
	seen := map[string]struct{}{}
	for _, result := range results {
		name := path.Base(strings.TrimSpace(result.Key))
		if name == "" || name == "." || name == "/" {
			continue
		}
		match, err := resolveReportFileByBase(manifestDir, name)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		reportFiles = append(reportFiles, match)
	}
	return reportFiles, nil
}

func resolveReportFileByBase(root string, name string) (string, error) {
	matches := []string{}
	err := filepath.WalkDir(root, func(candidate string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == name {
			matches = append(matches, candidate)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("search S3 checksum report bundle: %w", err)
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("S3 checksum report CSV %q was not found under %s", name, root)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("S3 checksum report CSV %q is ambiguous under %s", name, root)
	}
	return matches[0], nil
}

func (index *s3ChecksumReportIndex) loadCSV(reportPath string) error {
	file, err := os.Open(reportPath)
	if err != nil {
		return fmt.Errorf("open S3 checksum report CSV %s: %w", reportPath, err)
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read S3 checksum report CSV %s: %w", reportPath, err)
		}
		if isS3ReportHeader(record) {
			continue
		}
		if len(record) != s3ChecksumReportColumns {
			return fmt.Errorf("S3 checksum report CSV %s has %d columns, expected %d", reportPath, len(record), s3ChecksumReportColumns)
		}
		entry, ok := parseS3ChecksumReportEntry(record)
		if !ok {
			continue
		}
		// Completion reports name the bucket for each row. Only rows from the
		// configured production bucket are allowed to bypass local hashing.
		if entry.Bucket != index.preferred {
			continue
		}
		index.entriesByKey[entry.Key] = append(index.entriesByKey[entry.Key], entry)
	}
}

func isS3ReportHeader(record []string) bool {
	return len(record) > 1 && strings.EqualFold(strings.TrimSpace(record[0]), "Bucket") && strings.EqualFold(strings.TrimSpace(record[1]), "Key")
}

func parseS3ChecksumReportEntry(record []string) (s3ChecksumReportEntry, bool) {
	if !strings.EqualFold(strings.TrimSpace(record[3]), "succeeded") {
		return s3ChecksumReportEntry{}, false
	}
	key, err := url.QueryUnescape(record[1])
	if err != nil {
		key = record[1]
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(record[6]), &raw); err != nil {
		return s3ChecksumReportEntry{}, false
	}
	algorithm := strings.ToUpper(jsonStringValue(raw, "checksumAlgorithm", "checksum_algorithm", "ChecksumAlgorithm"))
	checksumType := strings.ToUpper(jsonStringValue(raw, "checksumType", "checksum_type", "ChecksumType"))
	checksumHex := strings.ToLower(jsonStringValue(raw, "checksum_hex", "checksumHex", "ChecksumHex"))
	if checksumHex == "" {
		if checksumBase64 := jsonStringValue(raw, "checksum_base64", "checksumBase64", "ChecksumBase64"); checksumBase64 != "" {
			if decoded, err := base64.StdEncoding.DecodeString(checksumBase64); err == nil {
				checksumHex = hex.EncodeToString(decoded)
			}
		}
	}
	if algorithm != "SHA256" || checksumType != "FULL_OBJECT" || !hashPattern.MatchString(checksumHex) {
		return s3ChecksumReportEntry{}, false
	}
	etag := normalizeETag(jsonStringValue(raw, "etag", "ETag"))
	if etag == "" {
		return s3ChecksumReportEntry{}, false
	}
	return s3ChecksumReportEntry{
		Bucket:      strings.TrimSpace(record[0]),
		Key:         key,
		ETag:        etag,
		ChecksumHex: checksumHex,
	}, true
}

func jsonStringValue(raw map[string]any, names ...string) string {
	for _, name := range names {
		value, ok := raw[name]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			return strings.TrimSpace(typed)
		default:
			return strings.TrimSpace(fmt.Sprint(typed))
		}
	}
	return ""
}

func (index *s3ChecksumReportIndex) Hash(key string) (string, bool) {
	if index == nil {
		return "", false
	}
	entries := index.entriesByKey[key]
	if len(entries) == 0 {
		return "", false
	}
	for _, entry := range entries {
		if entry.Bucket != index.preferred {
			continue
		}
		return entry.ChecksumHex, true
	}
	return "", false
}

func (index *s3ChecksumReportIndex) retainCurrentEntries() {
	entries := index.allEntries()
	if len(entries) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "validating %d S3 checksum report rows against current ETags...\n", len(entries))

	const workers = 32
	jobs := make(chan s3ChecksumReportEntry)
	results := make(chan s3ChecksumReportValidationResult)
	var wg sync.WaitGroup
	workerCount := workers
	if len(entries) < workerCount {
		workerCount = len(entries)
	}
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for entry := range jobs {
				results <- s3ChecksumReportValidationResult{
					entry:   entry,
					current: index.entryStillCurrent(entry),
				}
			}
		}()
	}
	go func() {
		for _, entry := range entries {
			jobs <- entry
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()

	current := map[string][]s3ChecksumReportEntry{}
	checked := 0
	trusted := 0
	for result := range results {
		checked++
		if result.current {
			current[result.entry.Key] = append(current[result.entry.Key], result.entry)
			trusted++
		}
		if checked%1000 == 0 {
			fmt.Fprintf(os.Stderr, "checked %d/%d S3 report ETags; trusted %d rows...\n", checked, len(entries), trusted)
		}
	}
	index.entriesByKey = current
	fmt.Fprintf(os.Stderr, "trusted %d/%d current S3 checksum report rows\n", trusted, len(entries))
}

type s3ChecksumReportValidationResult struct {
	entry   s3ChecksumReportEntry
	current bool
}

func (index *s3ChecksumReportIndex) allEntries() []s3ChecksumReportEntry {
	entries := []s3ChecksumReportEntry{}
	for _, keyEntries := range index.entriesByKey {
		entries = append(entries, keyEntries...)
	}
	return entries
}

func (index *s3ChecksumReportIndex) entryStillCurrent(entry s3ChecksumReportEntry) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	etag, err := index.reader.ETag(ctx, entry.Bucket, entry.Key)
	if err != nil {
		return false
	}
	return normalizeETag(etag) == entry.ETag
}

func normalizeETag(etag string) string {
	return strings.Trim(strings.TrimSpace(etag), `"`)
}

func storageKey(collection *core.Collection, recordID string, filename string) string {
	return collection.BaseFilesPath() + "/" + recordID + "/" + filename
}

func writeCopyScript(path string, bucketEnv string, rows []ManifestRow) (int, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	if strings.TrimSpace(bucketEnv) == "" {
		bucketEnv = "TYBALT_S3_BUCKET"
	}

	lines := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		fmt.Sprintf(": \"${%s:?set %s}\"", bucketEnv, bucketEnv),
		"",
	}
	copyRows := make([]ManifestRow, 0)
	seenTargets := map[string]struct{}{}
	for _, row := range rows {
		if !row.CopyRequired {
			continue
		}
		if _, ok := seenTargets[row.NewS3Key]; ok {
			continue
		}
		seenTargets[row.NewS3Key] = struct{}{}
		copyRows = append(copyRows, row)
	}
	if len(copyRows) > 0 {
		lines = append(lines,
			"echo \"Preflighting destination keys before copy...\" >&2",
			"",
		)
	}
	for _, row := range copyRows {
		// Preflight all planned destinations before the first copy command. This
		// protects against drift after prepare: if another process created one
		// of these keys, the script aborts instead of overwriting it.
		lines = append(lines,
			fmt.Sprintf("if head_output=$(aws s3api head-object --bucket \"$%s\" --key \"%s\" --output json 2>&1); then", bucketEnv, shellDoubleQuotedPath(row.NewS3Key)),
			fmt.Sprintf("  echo \"Refusing to overwrite existing destination: s3://$%s/%s\" >&2", bucketEnv, shellDoubleQuotedPath(row.NewS3Key)),
			"  exit 3",
			"else",
			"  head_status=$?",
			"  if [[ \"$head_output\" != *\"(404)\"* && \"$head_output\" != *\"Not Found\"* && \"$head_output\" != *\"NotFound\"* && \"$head_output\" != *\"NoSuchKey\"* ]]; then",
			fmt.Sprintf("    echo \"Failed to preflight destination: s3://$%s/%s\" >&2", bucketEnv, shellDoubleQuotedPath(row.NewS3Key)),
			"    echo \"$head_output\" >&2",
			"    exit \"$head_status\"",
			"  fi",
			"fi",
			"",
		)
	}
	if len(copyRows) > 0 {
		lines = append(lines,
			"echo \"Copying legacy expense attachments...\" >&2",
			"",
		)
	}
	for _, row := range copyRows {
		// The generated script performs only aws s3 cp commands. It is safe to
		// run before production is stopped because it never mutates source keys
		// and never writes database state.
		lines = append(lines, fmt.Sprintf("aws s3 cp \"s3://$%s/%s\" \"s3://$%s/%s\" --checksum-algorithm SHA256 --only-show-errors", bucketEnv, shellDoubleQuotedPath(row.OldS3Key), bucketEnv, shellDoubleQuotedPath(row.NewS3Key)))
	}
	_, err = file.WriteString(strings.Join(lines, "\n") + "\n")
	return len(copyRows), err
}

func writeCleanupScript(path string, bucketEnv string, rows []ManifestRow, smokeOnly bool) (int, error) {
	if !smokeOnly {
		// Full production prepare intentionally has no generated bulk delete
		// script. Remove a stale smoke-test script if the operator reused an
		// output directory, but ignore the normal "nothing to remove" case.
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return 0, err
		}
		return 0, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return 0, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	if strings.TrimSpace(bucketEnv) == "" {
		bucketEnv = "TYBALT_S3_BUCKET"
	}

	lines := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		fmt.Sprintf(": \"${%s:?set %s}\"", bucketEnv, bucketEnv),
		"echo \"cleanup_s3.sh is for limited pre-apply smoke tests only.\" >&2",
		": \"${CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS:?set CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS=yes to delete copied test targets}\"",
		"if [[ \"$CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS\" != \"yes\" ]]; then",
		"  echo \"Refusing to delete copied expense document targets without CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS=yes\" >&2",
		"  exit 2",
		"fi",
		"",
	}
	seen := map[string]struct{}{}
	commands := 0
	for _, row := range rows {
		if !row.CopyRequired {
			continue
		}
		if _, ok := seen[row.NewS3Key]; ok {
			continue
		}
		seen[row.NewS3Key] = struct{}{}
		// Cleanup intentionally targets only manifest destination keys that
		// copy_s3.sh was expected to create. It never deletes legacy source keys
		// and it skips existing-document reuse rows where copy_required=false.
		lines = append(lines, fmt.Sprintf("aws s3 rm \"s3://$%s/%s\" --only-show-errors", bucketEnv, shellDoubleQuotedPath(row.NewS3Key)))
		commands++
	}
	_, err = file.WriteString(strings.Join(lines, "\n") + "\n")
	return commands, err
}

func writeErrors(path string, rows []ErrorRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	if err := writer.Write([]string{"expense_id", "stage", "code", "message"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.ExpenseID, row.Stage, row.Code, row.Message}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeReport(path string, result ReportResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	if err := writer.Write([]string{"metric", "value"}); err != nil {
		return err
	}
	rows := [][]string{
		{"legacy_attachments", strconv.Itoa(result.LegacyAttachments)},
		{"document_backed_attachments", strconv.Itoa(result.DocumentBackedAttachments)},
		{"committed_legacy_only_attachments", strconv.Itoa(result.CommittedLegacyOnlyAttachments)},
		{"document_backed_blank_legacy_attachments", strconv.Itoa(result.DocumentBackedBlankLegacy)},
		{"document_backed_missing_targets", strconv.Itoa(result.DocumentBackedMissingTargets)},
		{"duplicate_document_references", strconv.Itoa(result.DuplicateDocumentReferences)},
		{"missing_legacy_files", strconv.Itoa(result.MissingLegacyFiles)},
		{"blank_or_invalid_hashes", strconv.Itoa(result.BlankOrInvalidHashes)},
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func countMissingDocumentTargets(app core.App, hashSource *storageHashSource) (int, error) {
	rows := []struct {
		DocumentID string `db:"document_id"`
		Attachment string `db:"attachment"`
	}{}
	if err := app.DB().NewQuery(`
		SELECT DISTINCT ed.id AS document_id, ed.attachment AS attachment
		FROM expenses e
		INNER JOIN expense_documents ed ON e.attachment_document = ed.id
		WHERE e.attachment_document != ''
	`).All(&rows); err != nil {
		return 0, err
	}
	collection, err := app.FindCollectionByNameOrId(constants.ExpenseDocumentsCollectionName)
	if err != nil {
		return 0, err
	}
	missing := 0
	fmt.Fprintf(os.Stderr, "checking %d document-backed expense document targets...\n", len(rows))
	for i, row := range rows {
		if i > 0 && i%1000 == 0 {
			fmt.Fprintf(os.Stderr, "checked %d/%d document-backed targets; missing %d...\n", i, len(rows), missing)
		}
		if err := hashSource.Exists(storageKey(collection, row.DocumentID, row.Attachment)); err != nil {
			missing++
		}
	}
	fmt.Fprintf(os.Stderr, "checked %d document-backed targets; missing %d\n", len(rows), missing)
	return missing, nil
}

func countMissingLegacyFiles(app core.App, hashSource *storageHashSource) (int, error) {
	rows := []struct {
		ID         string `db:"id"`
		Attachment string `db:"attachment"`
	}{}
	if err := app.DB().NewQuery("SELECT id, attachment FROM expenses WHERE attachment != ''").All(&rows); err != nil {
		return 0, err
	}
	collection, err := app.FindCollectionByNameOrId("expenses")
	if err != nil {
		return 0, err
	}
	missing := 0
	fmt.Fprintf(os.Stderr, "checking %d legacy expense attachment targets...\n", len(rows))
	for i, row := range rows {
		if i > 0 && i%1000 == 0 {
			fmt.Fprintf(os.Stderr, "checked %d/%d legacy targets; missing %d...\n", i, len(rows), missing)
		}
		if err := hashSource.Exists(storageKey(collection, row.ID, row.Attachment)); err != nil {
			missing++
		}
	}
	fmt.Fprintf(os.Stderr, "checked %d legacy targets; missing %d\n", len(rows), missing)
	return missing, nil
}

func countBlankOrInvalidLegacyHashes(app core.App) (int, error) {
	rows := []struct {
		AttachmentHash string `db:"attachment_hash"`
	}{}
	if err := app.DB().NewQuery("SELECT attachment_hash FROM expenses WHERE attachment != ''").All(&rows); err != nil {
		return 0, err
	}
	invalid := 0
	for _, row := range rows {
		hash := strings.ToLower(strings.TrimSpace(row.AttachmentHash))
		if hash == "" || !hashPattern.MatchString(hash) {
			invalid++
		}
	}
	return invalid, nil
}

func normalizeVerifyOptions(options VerifyOptions) VerifyOptions {
	if options.ChecksumMode == "" {
		options.ChecksumMode = ChecksumModeLocal
	}
	return options
}

func ParseChecksumMode(value string) (ChecksumMode, error) {
	switch ChecksumMode(strings.ToLower(strings.TrimSpace(value))) {
	case ChecksumModeAuto:
		return ChecksumModeAuto, nil
	case ChecksumModeS3:
		return ChecksumModeS3, nil
	case ChecksumModeLocal, "":
		return ChecksumModeLocal, nil
	default:
		return "", fmt.Errorf("checksum mode must be one of auto, s3, or local")
	}
}

func checksumVerifierForMode(paths Paths, mode ChecksumMode) (s3ChecksumVerifier, error) {
	if mode == ChecksumModeLocal {
		return nil, nil
	}
	verifier, err := newS3ChecksumVerifier(paths)
	if err != nil {
		if mode == ChecksumModeS3 {
			return nil, err
		}
		return nil, nil
	}
	return verifier, nil
}

func defaultNewS3ChecksumVerifier(paths Paths) (s3ChecksumVerifier, error) {
	cfg := s3ChecksumConfigFromEnv(paths)
	missing := cfg.missingRequiredEnv(paths.BucketEnv)
	if len(missing) > 0 {
		return nil, fmt.Errorf("S3 checksum verification requires %s", strings.Join(missing, ", "))
	}
	// Use the normal AWS SDK default credential chain so this command works
	// with the same auth styles operators use with the AWS CLI: env vars,
	// shared credentials files, profiles, SSO, and related providers.
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.region))
	if err != nil {
		return nil, err
	}
	opts := []func(*s3.Options){}
	if cfg.endpoint != "" && strings.HasPrefix(cfg.endpoint, "http") {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.endpoint)
			o.UsePathStyle = true
		})
	}
	return &defaultS3ChecksumVerifier{
		bucket: cfg.bucket,
		client: s3.NewFromConfig(awsCfg, opts...),
	}, nil
}

func defaultNewS3ReportMetadataReader(paths Paths) (s3ReportMetadataReader, error) {
	cfg := s3ChecksumConfigFromEnv(paths)
	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(cfg.region))
	if err != nil {
		return nil, err
	}
	opts := []func(*s3.Options){}
	if cfg.endpoint != "" && strings.HasPrefix(cfg.endpoint, "http") {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.endpoint)
			o.UsePathStyle = true
		})
	}
	return &defaultS3ReportMetadataReader{client: s3.NewFromConfig(awsCfg, opts...)}, nil
}

func s3ChecksumConfigFromEnv(paths Paths) s3ChecksumConfig {
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "ca-central-1"
	}
	return s3ChecksumConfig{
		bucket:   os.Getenv(paths.BucketEnv),
		region:   region,
		endpoint: firstNonEmpty(os.Getenv("AWS_ENDPOINT_URL_S3"), os.Getenv("AWS_ENDPOINT_URL")),
	}
}

func (cfg s3ChecksumConfig) missingRequiredEnv(bucketEnv string) []string {
	missing := []string{}
	if cfg.bucket == "" {
		missing = append(missing, bucketEnv)
	}
	return missing
}

func (r *defaultS3ReportMetadataReader) ETag(ctx context.Context, bucket string, key string) (string, error) {
	output, err := r.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(output.ETag), nil
}

func (v *defaultS3ChecksumVerifier) ChecksumSHA256Hex(ctx context.Context, key string) (s3ChecksumResult, error) {
	output, err := v.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket:       aws.String(v.bucket),
		Key:          aws.String(key),
		ChecksumMode: s3types.ChecksumModeEnabled,
	})
	if err != nil {
		return s3ChecksumResult{}, err
	}
	if output.ChecksumSHA256 == nil || strings.TrimSpace(aws.ToString(output.ChecksumSHA256)) == "" {
		return s3ChecksumResult{Found: false}, nil
	}
	fullObject := output.ChecksumType == "" || output.ChecksumType == s3types.ChecksumTypeFullObject
	checksumBytes, err := base64.StdEncoding.DecodeString(aws.ToString(output.ChecksumSHA256))
	if err != nil {
		return s3ChecksumResult{}, fmt.Errorf("decode S3 ChecksumSHA256 for %s: %w", key, err)
	}
	if len(checksumBytes) != sha256.Size {
		return s3ChecksumResult{}, fmt.Errorf("S3 ChecksumSHA256 for %s decoded to %d bytes, expected %d", key, len(checksumBytes), sha256.Size)
	}
	return s3ChecksumResult{
		HexChecksum: hex.EncodeToString(checksumBytes),
		Found:       true,
		FullObject:  fullObject,
	}, nil
}

func normalizePaths(paths Paths) Paths {
	if strings.TrimSpace(paths.OutDir) == "" {
		return DefaultPaths("")
	}
	defaults := DefaultPaths(paths.OutDir)
	if paths.ManifestPath == "" {
		paths.ManifestPath = defaults.ManifestPath
	}
	if paths.CopyScriptPath == "" {
		paths.CopyScriptPath = defaults.CopyScriptPath
	}
	if paths.CleanupScriptPath == "" {
		paths.CleanupScriptPath = defaults.CleanupScriptPath
	}
	if paths.ErrorsPath == "" {
		paths.ErrorsPath = defaults.ErrorsPath
	}
	if paths.VerifyErrorsPath == "" {
		paths.VerifyErrorsPath = defaults.VerifyErrorsPath
	}
	if paths.ReportPath == "" {
		paths.ReportPath = defaults.ReportPath
	}
	if paths.BucketEnv == "" {
		paths.BucketEnv = defaults.BucketEnv
	}
	return paths
}

func errorRow(expenseID string, stage string, code string, err error) ErrorRow {
	return ErrorRow{
		ExpenseID: expenseID,
		Stage:     stage,
		Code:      code,
		Message:   err.Error(),
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, filesystem.ErrNotFound) || errors.Is(err, os.ErrNotExist)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func shellDoubleQuotedPath(path string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")
	return replacer.Replace(path)
}
