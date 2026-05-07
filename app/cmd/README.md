# Expense Document Backfill Command

`backfill_expense_documents` is the Phase 2 migration tool for moving committed
legacy expense attachments into reusable `expense_documents` records.

It is designed for a local-first production workflow:

1. Work against a local dump of the production PocketBase database.
2. Generate a deterministic manifest and copy scripts without changing the DB.
3. Copy S3 objects from the legacy `expenses` storage prefix to the new
   `expense_documents` storage prefix.
4. Verify copied destination objects against the manifest.
5. Only after production is stopped and re-dumped, apply the DB writes.

The command lives at:

```sh
go run ./cmd/backfill_expense_documents <prepare|verify|apply|report> --data-dir <path>
```

Run the examples below from `app/`.

## What It Does

`prepare` finds committed legacy-only expenses:

```sql
SELECT *
FROM expenses
WHERE attachment != ''
  AND (attachment_document = '' OR attachment_document IS NULL)
  AND committed != ''
```

It writes operator artifacts under `--out-dir`:

- `manifest.tsv`: the exact expense-to-document plan.
- `copy_s3.sh`: copy-only S3 commands for destination objects.
- `cleanup_s3.sh`: guarded pre-apply cleanup for smoke-test destination objects,
  generated only when `prepare --limit ...` is used.
- `errors.tsv`: rows skipped during prepare.

`prepare` does not write to the database.

`copy_s3.sh` first preflights every planned destination key with:

```sh
aws s3api head-object ...
```

If any destination key already exists, the script aborts before copying the
first object. This protects against overwriting a file that appeared after
`prepare`.

After the preflight passes, `copy_s3.sh` uses:

```sh
aws s3 cp ... --checksum-algorithm SHA256
```

That asks S3 to compute/store SHA-256 checksum metadata on the copied destination
object. `verify --checksum-mode s3` then checks S3 `ChecksumSHA256` metadata
instead of downloading every copied object.

`verify` checks:

- copied destination object exists;
- destination hash matches `manifest.tsv`;
- legacy source object still exists;
- expense is still unlinked or already linked to the expected document;
- existing `expense_documents` rows for reused hashes still match the manifest.

`apply` is the only DB-mutating step. It re-runs verification checks, inserts
missing `expense_documents` rows using manifest IDs, and links expenses in one
transaction. It leaves `expenses.attachment` and `expenses.attachment_hash` in
place.

## Environment

The examples assume the local dump is already present:

```sh
test -f pb_data/data.db
```

If the production dump's PocketBase settings are encrypted, set the same
environment variable the app uses for its 32-character settings encryption key
and pass that variable name to the command:

```sh
export PB_ENCRYPTION_KEY='your-32-character-settings-key'
PB_ENCRYPTION_FLAG='--encryption-env PB_ENCRYPTION_KEY'
```

Leave `PB_ENCRYPTION_FLAG` blank for unencrypted dumps.

Set the production bucket for the generated scripts and S3 checksum verification:

```sh
export TYBALT_S3_BUCKET='your-bucket-name'
```

The AWS CLI and the Go verifier use the normal AWS credential chain. Any of
these are fine as long as `aws sts get-caller-identity` works:

```sh
export AWS_ACCESS_KEY_ID='...'
export AWS_SECRET_ACCESS_KEY='...'
```

or an AWS profile/SSO/shared-credentials setup. Set the region if needed:

```sh
export AWS_REGION='ca-central-1'
```

For S3-compatible storage, also set one of:

```sh
export AWS_ENDPOINT_URL_S3='https://...'
# or
export AWS_ENDPOINT_URL='https://...'
```

Use a recent AWS CLI. The generated copy script requires `aws s3 cp
--checksum-algorithm SHA256` and `aws s3api head-object`.

## Smoke Test

This smoke test uploads only two copied destination objects, verifies them, then
deletes those copied destination objects. It must be run before `apply`.

Use a separate output directory so smoke-test artifacts cannot be confused with
the full migration artifacts:

```sh
SMOKE_OUT='tmp/expense_document_backfill_smoke'
```

Generate a two-row copy-required manifest:

```sh
go run ./cmd/backfill_expense_documents prepare \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$SMOKE_OUT" \
  --limit 2 \
  --require-copy
```

Inspect the artifacts before copying:

```sh
wc -l "$SMOKE_OUT/manifest.tsv" "$SMOKE_OUT/errors.tsv"
sed -n '1,5p' "$SMOKE_OUT/manifest.tsv"
sed -n '1,20p' "$SMOKE_OUT/copy_s3.sh"
sed -n '1,24p' "$SMOKE_OUT/cleanup_s3.sh"
```

Expected:

- `manifest.tsv` has one header plus two data rows.
- both data rows have `copy_required` set to `true`;
- `copy_s3.sh` contains two `aws s3 cp` commands;
- `copy_s3.sh` contains destination `head-object` preflights before those copy
  commands;
- `copy_s3.sh` includes `--checksum-algorithm SHA256`;
- `cleanup_s3.sh` contains only destination-key deletes and has the confirmation
  guard;
- `cleanup_s3.sh` says it is for limited pre-apply smoke tests only.

Run the copy:

```sh
bash "$SMOKE_OUT/copy_s3.sh"
```

Verify using S3 checksum metadata:

```sh
go run ./cmd/backfill_expense_documents verify \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$SMOKE_OUT" \
  --checksum-mode s3
```

Expected:

```text
verified 2 rows, 0 failed
```

If `--checksum-mode s3` fails because checksum metadata is unavailable, use this
only as a diagnostic fallback:

```sh
go run ./cmd/backfill_expense_documents verify \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$SMOKE_OUT" \
  --checksum-mode local
```

Local mode downloads each copied destination through PocketBase storage and
hashes it locally. It is slower, but it proves whether the bytes are correct.

Clean up the two copied destination objects:

```sh
CONFIRM_DELETE_COPIED_EXPENSE_DOCUMENTS=yes \
  bash "$SMOKE_OUT/cleanup_s3.sh"
```

Sanity check that cleanup removed the copied targets:

```sh
go run ./cmd/backfill_expense_documents verify \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$SMOKE_OUT" \
  --checksum-mode s3
```

Expected after cleanup:

```text
verified 0 rows, 2 failed
```

That failure is good in the cleanup check. It proves the test objects were
removed. Do not run `apply` with the smoke-test manifest after cleanup.

## Full Dry Run Before Apply

After the smoke test, generate full artifacts in the normal output directory:

```sh
FULL_OUT='tmp/expense_document_backfill'

go run ./cmd/backfill_expense_documents report \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$FULL_OUT"

go run ./cmd/backfill_expense_documents prepare \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$FULL_OUT"
```

Review:

```sh
sed -n '1,20p' "$FULL_OUT/report.tsv"
sed -n '1,20p' "$FULL_OUT/errors.tsv"
sed -n '1,5p' "$FULL_OUT/manifest.tsv"
test ! -f "$FULL_OUT/cleanup_s3.sh"
```

Expected:

- `report.tsv` includes `document_backed_blank_legacy_attachments` so the
  pre-Phase-2 baseline covers document-backed rows that would not be readable by
  pre-Phase-1 code.
- `cleanup_s3.sh` does not exist for the full migration output.
- full migration cleanup, if ever needed, is handled through Attachment Audit,
  not through a generated bulk delete script.

Then copy and verify:

```sh
bash "$FULL_OUT/copy_s3.sh"

go run ./cmd/backfill_expense_documents verify \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$FULL_OUT" \
  --checksum-mode s3
```

Expected:

```text
verified <manifest row count> rows, 0 failed
```

## Failed Full Run Cleanup

The full migration intentionally does not generate `cleanup_s3.sh`. If the full
S3 copy succeeds but `verify` or `apply` later fails, do not bulk-delete from the
manifest. Leave the copied destination objects in place while diagnosing the
failure.

If you decide the copied-but-unapplied destination files need to be removed, use
the app's Attachment Audit tool after reviewing its orphan CSV:

1. Start the app against the database state you want to audit.
2. Open Settings -> Attachment Audit as an admin user.
3. Refresh the `Expense Documents` target.
4. Download and review the orphaned CSV for `expense_documents_attachment`.
5. Only if the CSV contains the copied destination objects you intend to remove,
   run Delete Orphans for that target.

Attachment Audit is safer than manifest cleanup because it uses the latest
database state. It treats the cached orphan CSV as candidates only, re-checks
each path against the current `expense_documents.attachment` records before
delete, skips files that became referenced, counts files already missing, and
returns per-file failures instead of stopping halfway through a large cleanup.

Important boundary: Attachment Audit deletes storage orphans. It will not delete
an `expense_documents` database row that exists but is not referenced by any
expense, and it will not delete that row's file because the file is still
referenced by the document record itself.

## Apply Warning

`apply` should be run only after:

1. all existing expenses requiring migration are committed;
2. production is stopped;
3. production has been re-dumped into `pb_data`;
4. the manifest has been verified against the stopped-production dump.

Then:

```sh
go run ./cmd/backfill_expense_documents apply \
  --data-dir pb_data \
  $PB_ENCRYPTION_FLAG \
  --out-dir "$FULL_OUT" \
  --checksum-mode s3
```

Do not run `cleanup_s3.sh` after `apply`; the database will reference those
destination files. For the full migration there should be no `cleanup_s3.sh` in
`$FULL_OUT` at all.
