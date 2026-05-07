// Command backfill_expense_documents is the operator-facing wrapper for the
// Phase 2 expense document migration.
//
// This command is intentionally small: all migration rules live in
// tybalt/backfill/expense_documents, while main.go is responsible only for
// opening an existing PocketBase data directory and dispatching one of the four
// migration steps:
//
//   - prepare: read committed legacy-only expenses from a local production dump,
//     generate manifest.tsv, copy_s3.sh, and errors.tsv, and perform no
//     database writes. Operators may pass --limit and --require-copy to create a
//     tiny smoke-test manifest before the full run; limited prepares also
//     generate the only supported cleanup_s3.sh helper.
//   - verify: after copy_s3.sh has copied S3 objects to the expense_documents
//     storage prefix, compare those copied targets against manifest.tsv and mark
//     each row verified or verify_error. By default this command uses S3's
//     stored full-object SHA-256 checksum metadata when direct S3 credentials
//     are available, falling back to local hashing only when needed.
//   - apply: after production has been stopped and the database has been
//     re-dumped/reconciled locally, insert the missing expense_documents rows
//     and link expenses to them in one database transaction.
//   - report: write a small baseline TSV summarizing current legacy/document
//     attachment state and storage gaps.
//
// The expected production workflow is deliberately copy-first and DB-last:
//
//  1. Dump production locally while production is still running.
//  2. Run prepare against that local dump.
//  3. Run the generated copy_s3.sh to copy, never move/delete, legacy files into
//     their expense_documents storage keys.
//  4. For a limited smoke test only, cleanup_s3.sh can delete the copied
//     destination keys so the test leaves no pre-apply objects behind. Full
//     prepare removes any stale cleanup_s3.sh instead of generating one.
//  5. Run verify until the manifest is clean.
//  6. Stop production.
//  7. Re-dump production locally and rerun verify/apply against the stopped
//     state.
//
// The CLI does not import routes, hooks, or migrations. It expects --data-dir to
// point at a PocketBase data directory that already has the deployed schema.
// That keeps the command suitable for a production dump without accidentally
// starting the application server or running unrelated app behavior.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"

	expensedocuments "tybalt/backfill/expense_documents"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	dataDir := flags.String("data-dir", "", "PocketBase data directory")
	encryptionEnv := flags.String("encryption-env", "", "environment variable name containing the PocketBase settings encryption key")
	outDir := flags.String("out-dir", "", "backfill output directory")
	bucketEnv := flags.String("bucket-env", "TYBALT_S3_BUCKET", "environment variable used by copy_s3.sh for the S3 bucket")
	limit := flags.Int("limit", 0, "prepare only: maximum number of manifest rows to emit; 0 means no limit")
	requireCopy := flags.Bool("require-copy", false, "prepare only: include only rows that require a destination S3 copy")
	checksumModeFlag := flags.String("checksum-mode", string(expensedocuments.ChecksumModeAuto), "verify/apply only: checksum source, one of auto, s3, or local")
	flags.Usage = func() {
		usageForCommand(command, flags)
	}
	if err := flags.Parse(os.Args[2:]); err != nil {
		log.Fatal(err)
	}
	if *dataDir == "" {
		log.Fatal("--data-dir is required")
	}

	// Use a bare PocketBase app because this command needs direct DB/filesystem
	// access only. Registering the HTTP app would add irrelevant hooks/routes to
	// a migration that must stay predictable and operator-driven.
	app := core.NewBaseApp(core.BaseAppConfig{
		DataDir:       *dataDir,
		EncryptionEnv: *encryptionEnv,
	})
	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	defer app.ResetBootstrapState()

	paths := expensedocuments.DefaultPaths(*outDir)
	paths.BucketEnv = *bucketEnv
	checksumMode, err := expensedocuments.ParseChecksumMode(*checksumModeFlag)
	if err != nil {
		log.Fatal(err)
	}

	switch command {
	case "prepare":
		// Prepare is safe to run before production is stopped: it reads the local
		// dump and writes only local operator artifacts under --out-dir.
		result, err := expensedocuments.PrepareWithOptions(app, paths, expensedocuments.PrepareOptions{
			Limit:       *limit,
			RequireCopy: *requireCopy,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("prepared %d manifest rows, %d copy commands, %d cleanup commands, %d errors\n", result.ManifestRows, result.CopyCommands, result.CleanupCommands, result.Errors)
	case "verify":
		// Verify is also read-only with respect to the database. It is the guard
		// that proves copied S3 targets match manifest hashes before apply is
		// allowed to create/link database rows.
		result, err := expensedocuments.VerifyWithOptions(app, paths, expensedocuments.VerifyOptions{
			ChecksumMode: checksumMode,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("verified %d rows, %d failed\n", result.Verified, result.Failed)
	case "apply":
		// Apply is the only mutating step. Operators should run it only after
		// production is stopped and verify has passed against the final local
		// dump, so no new committed legacy-only expense can appear mid-apply.
		result, err := expensedocuments.ApplyWithOptions(app, paths, expensedocuments.VerifyOptions{
			ChecksumMode: checksumMode,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("inserted %d documents, linked %d expenses, skipped %d already-linked expenses\n", result.InsertedDocuments, result.LinkedExpenses, result.SkippedExpenses)
	case "report":
		// Report is a baseline/checkpoint artifact. It is useful both before the
		// migration and after apply to show how many legacy-only committed rows
		// remain and whether storage references are missing.
		result, err := expensedocuments.Report(app, paths)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("legacy_attachments\t%d\n", result.LegacyAttachments)
		fmt.Printf("document_backed_attachments\t%d\n", result.DocumentBackedAttachments)
		fmt.Printf("committed_legacy_only_attachments\t%d\n", result.CommittedLegacyOnlyAttachments)
		fmt.Printf("document_backed_blank_legacy_attachments\t%d\n", result.DocumentBackedBlankLegacy)
		fmt.Printf("document_backed_missing_targets\t%d\n", result.DocumentBackedMissingTargets)
		fmt.Printf("duplicate_document_references\t%d\n", result.DuplicateDocumentReferences)
		fmt.Printf("missing_legacy_files\t%d\n", result.MissingLegacyFiles)
		fmt.Printf("blank_or_invalid_hashes\t%d\n", result.BlankOrInvalidHashes)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: backfill_expense_documents <prepare|verify|apply|report> --data-dir <path> [options]

Commands:
  prepare  Build manifest.tsv, copy_s3.sh, and errors.tsv without DB writes. With --limit, also build smoke-only cleanup_s3.sh.
  verify   Compare copied destination objects to manifest hashes and mark rows verified or verify_error.
  apply    Insert missing expense_documents rows and link expenses in one transaction.
  report   Write report.tsv and print baseline attachment counts.

Common options:
  --data-dir <path>       PocketBase data directory to open.
  --encryption-env <name> Environment variable containing the PocketBase settings encryption key.
  --out-dir <path>        Artifact directory. Defaults to tmp/expense_document_backfill.
  --bucket-env <name>     Environment variable used by generated S3 scripts. Defaults to TYBALT_S3_BUCKET.
  --checksum-mode <mode>  For verify/apply: auto, s3, or local. Defaults to auto.

Prepare smoke-test options:
  --limit <n>             Emit at most n manifest rows. Use only for pre-apply smoke tests.
  --require-copy          With prepare, include only rows that actually need copy_s3.sh work.

Generated cleanup_s3.sh exists only for limited pre-apply smoke tests. Full prepare
removes stale cleanup_s3.sh files instead of generating a production delete script.
The smoke cleanup deletes only destination keys from manifest rows where
copy_required=true, skips reused existing documents, and requires
CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS=yes before it will run.

Checksum modes:
  auto   Use S3 ChecksumSHA256 metadata when TYBALT_S3_BUCKET and normal AWS SDK
         credentials are available; fall back to local hashing when checksum
         metadata is missing or unavailable. AWS_REGION defaults to ca-central-1;
         AWS_ENDPOINT_URL_S3 or AWS_ENDPOINT_URL may be used for non-AWS
         S3-compatible storage.
  s3     Require S3 ChecksumSHA256 metadata and require it to be a full-object
         checksum. This is the fastest strict sanity check after copy_s3.sh.
  local  Download each destination through PocketBase storage and compute SHA-256
         locally.`)
}

func usageForCommand(command string, flags *flag.FlagSet) {
	usage()
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Options accepted by %s:\n", command)
	flags.PrintDefaults()
}
