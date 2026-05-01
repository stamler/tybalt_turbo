# Expense Documents

## Goal

Expense attachments are currently stored directly on `expenses` records and guarded by a unique hash on `expenses.attachment_hash`. That prevents a legitimate finance workflow: a `book_keeper` receives one vendor attachment that supports multiple PO-backed expenses, then needs to enter one expense per PO while reusing the same supporting document.

The target model is:

- `expenses` remain the accounting rows.
- `expense_documents` own uploaded receipt/invoice files.
- Multiple expenses may reference the same `expense_documents` row.
- Only users with the `book_keeper` claim may reuse an existing document.
- Everyone else keeps the same UX and duplicate-file protection they have today.

This will be implemented in three phases.

1. Phase 1 cuts over the app so new expense uploads use `expense_documents`, while legacy expense attachments continue to work through fallback reads.
2. Phase 2 backfills existing legacy expense attachments into `expense_documents`.
3. Phase 3 removes the legacy fallback code and legacy attachment columns after the backfill has been verified.

Phase 1 and Phase 2 do not need to be close together. After Phase 1, the app can run indefinitely with mixed storage: new attachments in `expense_documents`, old attachments still on `expenses`. Phase 3 happens only after Phase 2 has been verified in production.

## Current Storage Model

PocketBase stores files under the collection id and record id:

```text
{collection_id}/{record_id}/{filename}
```

For current expense attachments, the collection id is the `expenses` collection id:

```text
o1vpz1mm7qsfoyy/{expense_id}/{expense_attachment_filename}
```

After this change, new expense documents will be stored under the new `expense_documents` collection id:

```text
{expense_documents_collection_id}/{expense_document_id}/{document_attachment_filename}
```

That storage-prefix change is the main reason this is a two-phase rollout. Phase 1 must read both locations. Phase 2 can later copy old S3 objects into the new prefixes and link the old expenses to the new document rows.

## Phase 1: Cut Over New Writes

Phase 1 is implemented on the `multi_use_attachment` branch and tested locally against a dumped production DB before deployment. The Phase 1 deploy includes schema changes, document-first reads with legacy fallback, and the new write path in the same release.

### 1. Add the `expense_documents` schema

Create a migration using `date +%s` for the filename prefix.

Add a new base collection named `expense_documents` with these fields:

- `attachment`: file, required, `maxSelect: 1`, same max size and MIME types as the current `expenses.attachment` field.
- `attachment_hash`: text, required, 64-character SHA-256 hex string.
- `uploaded_by`: relation to `users`, required.
- standard PocketBase `created` and `updated` autodate fields.

Add this index:

```sql
CREATE UNIQUE INDEX `idx_expense_documents_attachment_hash`
ON `expense_documents` (`attachment_hash`)
WHERE `attachment_hash` != ''
```

Direct API access to `expense_documents` should be intentionally narrow:

- no general list/search UI;
- no user-created direct document records through the generic collection API;
- no direct document browsing.

The backend expense save flow will create and link documents. Users interact through expenses, not through a standalone file cabinet.

### 2. Add `expenses.attachment_document`

In the same migration, add a nullable relation field to `expenses`:

```text
attachment_document -> expense_documents
```

Do not remove or clear these existing fields in Phase 1:

- `expenses.attachment`
- `expenses.attachment_hash`
- the existing unique index on `expenses.attachment_hash`

Those legacy fields remain the fallback source for old rows until Phase 2 cleanup.

### 3. Introduce an effective attachment resolver

Add one backend concept for "the attachment for this expense":

```text
if expenses.attachment_document is set:
  use expense_documents.attachment and expense_documents.attachment_hash
else:
  use expenses.attachment and expenses.attachment_hash
```

The resolver must return enough information for all existing use cases:

- effective filename;
- effective SHA-256 hash;
- effective collection id;
- effective record id;
- effective storage key/source path;
- whether the value came from `expense_documents` or legacy `expenses` fields.

All code that displays, downloads, zips, exports, or writes back expense attachments must use this resolver or SQL equivalent. Do not leave ad hoc reads of `e.attachment` in user-facing or integration paths unless they are deliberately handling the legacy fallback branch.

Update at minimum:

- expense details API and details page;
- expense list and tracking responses when attachment data is shown;
- receipt zip query and zip creation;
- legacy expenses writeback/export;
- any expense report SQL that selects `e.attachment` or `e.attachment_hash`;
- generated TypeScript types after the schema changes.

### 4. Add an authorized expense attachment download route

Because document files are no longer necessarily stored under the expense record, avoid exposing document files through broad collection rules.

Add a route like:

```text
GET /api/expenses/attachment/{id}
```

Behavior:

1. Require auth.
2. Load the expense through the same visibility policy used by expense details.
3. Resolve the effective attachment.
4. Stream or redirect to the underlying PocketBase file only if the caller can view the expense.

The details page and list/report links should use this expense-scoped route rather than constructing document collection file URLs directly.

Server-side report generation can continue reading files internally through PocketBase filesystem APIs, using the effective storage key.

### 5. Add transactional custom expense write endpoints

The editor saves expenses through app-owned write endpoints rather than the generic PocketBase collection write endpoints:

```text
POST  /api/expenses
PATCH /api/expenses/{id}
```

This is intentional because an expense attachment upload now mutates two collections:

```text
expense_documents
expenses
```

The custom endpoints are the canonical Phase 1 write path for expense create/update from the UI. They accept the same logical form payload the editor already produces, including file uploads, duplicate-upload reuse, and `source_expense` reuse.

The endpoint implementation:

1. requires an authenticated `users` record;
2. loads or creates the target `expenses` record;
3. loads request data into the normal PocketBase `RecordUpsert` form so collection field validation and normal model save hooks still run;
4. calls the existing expense processing code for ownership, PO, branch, approver, currency, cumulative total, bookkeeper-on-behalf, attachment-required, and submitted-expense validation;
5. resolves the effective attachment document before saving;
6. saves the `expense_documents` row and the `expenses` row inside one `app.RunInTransaction` call;
7. if the transaction fails after a new document file was uploaded, rolls back the DB changes and best-effort deletes the uploaded document object from PocketBase storage/S3.

The generic PocketBase `expenses` create/update request hooks also wrap their existing `ProcessExpense` + `e.Next()` flow in a transaction and use the same cleanup tracker. This keeps direct collection API writes safe during the transition, but the UI uses only the custom endpoints.

Keep the visible form unchanged for ordinary users:

- same fields;
- same attachment control;
- same validation messages where possible;
- same redirect after save.

All existing backend validation remains centralized in the current expense processing path. The custom endpoint owns the transaction boundary and request parsing, not a second copy of PO, branch, approver, bookkeeper-on-behalf, currency, and attachment-required rules.

After cutover, a direct upload to `expenses.attachment` is not a legacy write. Whether it arrives through the custom endpoint or the generic collection API, the backend converts it to an `expense_documents` link before persistence.

The editor trims its save payload to editable expense fields before calling the custom endpoint. Existing document-backed filenames shown in the attachment control are not resent as `attachment` values. Only a new `File`, an explicit empty attachment removal, or `source_expense` is sent as attachment intent.

### 6. Implement document resolution during create/update

Add a backend helper used by the expense create/update hooks:

```text
resolveExpenseDocumentForSave(auth, targetExpense, uploadedFile, sourceExpenseID)
```

It handles four cases.

#### Case A: New file, hash not seen before

1. Compute SHA-256 from the uploaded file.
2. Create a new `expense_documents` record.
3. Store the uploaded file on `expense_documents.attachment`.
4. Set `expense_documents.attachment_hash`.
5. Set `expense_documents.uploaded_by` to the authenticated user.
6. Set `expenses.attachment_document` to the new document id.
7. Leave `expenses.attachment` and `expenses.attachment_hash` empty for this new/updated expense.

#### Case B: New file, hash already exists, caller is not `book_keeper`

Reject with the existing duplicate attachment semantics:

```text
field: attachment
code: duplicate_file
message: This file has already been uploaded to another expense
```

This preserves current UX for everyone without the `book_keeper` claim.

#### Case C: New file, hash already exists, caller has `book_keeper`

Allow reuse only when the target expense is PO-backed:

```text
expenses.purchase_order != ''
```

Then:

1. Do not create a duplicate `expense_documents` row.
2. Set `expenses.attachment_document` to the existing document id.
3. Leave `expenses.attachment` and `expenses.attachment_hash` empty for this expense.

If the target expense has no PO, reject the reuse. The special workflow exists only for splitting a shared vendor document across multiple PO-backed expenses.

#### Case D: Existing source expense document

For the "Create another expense with this attachment" workflow, accept a `source_expense` id in the create request.

Allow this only when:

- caller has the `book_keeper` claim;
- target expense is PO-backed;
- source expense is visible to the caller;
- source expense has an effective attachment;
- if the source expense is still legacy-only, the source attachment can be migrated to `expense_documents` before linking it.

Then set the target expense's `attachment_document` to the source document id.

Do not implement arbitrary document picking. The only reuse flows are duplicate upload and "create another expense with this attachment" from an existing visible expense.

### 7. Automatically migrate legacy attachments when edited

When an existing expense has:

```text
expenses.attachment != ''
expenses.attachment_document == ''
```

and that expense is saved through the generic expense save path, migrate it as part of the save unless the user replaces the attachment.

Legacy edit behavior:

- If the user uploads a replacement file, use the new uploaded file and the normal document-resolution rules.
- If the user does not upload a replacement and the legacy attachment remains present, copy the existing legacy file into a new or existing `expense_documents` record and set `expenses.attachment_document`.
- If the user removes an attachment and the expense type permits no attachment, clear both the document relation and legacy attachment values as appropriate.

This means normal interaction with old rows gradually moves them to the new model even before the Phase 2 bulk backfill.

### 8. Add bookkeeper-only source reuse UX

On the expense details page, for callers with `book_keeper`, show a new action only when the source expense has an effective attachment. The action button should be next to the attachment field (download) and read:

```text
Create another expense with this attachment
```

The action should lead into the existing PO-backed create flow. The target expense still gets its PO-specific defaults from the selected PO. The source expense contributes only the attachment document.

Recommended route shape:

```text
/expenses/add/{poid}?source_expense={expense_id}
```

The page loader can preserve `source_expense` in page data, and the editor includes it in the create request. The backend remains authoritative and must validate every condition listed in Case D.

### 9. Preserve editor record shape for edit pages

Expense edit pages should continue loading the normal PocketBase `expenses` record rather than the custom details response:

```text
pb.collection("expenses").getOne(expense_id, {
  expand: "purchase_order,attachment_document"
})
```

This is intentional. The editor relies on PocketBase system fields and `expand.purchase_order` to determine ownership, PO payment type, PO currency, and recurring/cumulative PO hints. Loading the custom details response would drop that record shape and make some PO-backed or bookkeeper-created expenses behave like non-editable records.

For document-backed expenses, the edit loader uses `expand.attachment_document.attachment` as the display filename while preserving the canonical `attachment_document` relation on the expense record. For legacy-only expenses during Phase 1 and Phase 2, the loader falls back to `expenses.attachment`.

The editor's attachment link should use the expense-scoped download route:

```text
GET /api/expenses/attachment/{id}
```

Do not construct direct `expense_documents` file URLs in the UI. The route keeps authorization scoped to expense visibility and supports both document-backed and legacy-only attachments.

In Phase 3, after all legacy rows have been backfilled and the legacy columns are removed, the editor can remove the fallback to `expenses.attachment` and rely only on `attachment_document`.

### 10. Known issues and caveats

Phase 1 is intentionally a mixed-storage rollout. The app writes new expense attachments to `expense_documents`, but legacy rows may continue to point at files under the `expenses` collection until Phase 2 backfill. Any code that reads expense attachments must use the effective attachment resolver pattern:

```text
COALESCE(expense_documents.attachment, expenses.attachment)
COALESCE(expense_documents.attachment_hash, expenses.attachment_hash)
```

Known caveats:

- Rolling back to pre-Phase-1 code after new document-backed writes have landed is not data-neutral. Pre-Phase-1 code reads `expenses.attachment` directly, so newly created document-backed expenses with blank legacy attachment fields can lose attachment visibility until the Phase 1 code is restored or the data is repaired.
- Old direct file URLs under `/api/files/expenses/...` are not the long-term attachment contract. UI links should use `GET /api/expenses/attachment/{id}` so authorization stays expense-scoped and both document-backed and legacy-only rows work.
- Legacy writeback keeps the old payload shape. It exports effective `attachment` and `attachmentHash` values, not `attachment_document`, so downstream legacy consumers do not need a schema change. If a downstream process fetches files by filename, confirm it still resolves the correct storage location during mixed storage.
- Receipt ZIP generation must include both legacy-only and document-backed attachments. The ZIP cache manifest must include collection id, storage path, zip filename, and hash so a reused document or duplicate hash cannot produce a stale ZIP.
- The ZIP cache migration deletes existing `zip_cache` rows. The first receipt ZIP requests after deploy should regenerate those archives.
- A concurrent upload of the same new file can still race at the `expense_documents.attachment_hash` unique index. The expected steady-state behavior is still one document row per hash, but the losing request may surface a lower-level save error rather than the normal `duplicate_file` validation message.
- Phase 1 does not delete old legacy expense attachment objects. Storage cleanup should wait until after Phase 2 backfill is verified and a separate retention/rollback decision is made.
- Editing or reusing a legacy-only expense can lazily create or reuse an `expense_documents` row. This is expected, but it means normal user activity will gradually change some old rows before the bulk backfill runs. If the legacy file object is missing or unreadable in storage, that edit or source-reuse request can fail until the legacy attachment is restored or handled manually.
- The generic PocketBase `expenses` create/update hooks remain active and convert direct uploads to `expense_documents`. The custom `/api/expenses` routes are the UI path, but direct collection API writes should not silently create new legacy-only attachments.
- `image/heic` is accepted by backend validation, but browser preview/download behavior may be less polished than PDF, PNG, or JPEG. The attachment route should still stream the original file.

### 11. Phase 1 validation before deploy

Test on `multi_use_attachment` with a dumped production DB.

Minimum manual checks:

1. Open old expenses with legacy attachments. Download links still work.
2. Create a new expense with a new attachment. Verify the S3 key uses the `expense_documents` collection id, not the `expenses` collection id.
3. Create a normal non-bookkeeper duplicate. Verify it is rejected.
4. Create a bookkeeper PO-backed duplicate. Verify it reuses the existing `expense_documents` row.
5. Use "Create another expense with this attachment" as a bookkeeper. Verify the new PO expense shares the same document.
6. Edit an old legacy expense without changing the attachment. Verify it gets an `attachment_document` and the copied document file downloads.
7. Generate receipt zips and legacy writeback/export output for a mix of legacy and document-backed expenses.

After deploying Phase 1 to production, run a short smoke test before treating the
release as stable. The goal is not to re-test every backend branch; it is to
prove that the production app, database schema, file storage, auth rules, and
integration read paths all agree on the mixed-storage model.

Production smoke test:

1. Confirm the migration ran:
   - `expense_documents` exists;
   - `expenses.attachment_document` exists;
   - `zip_cache` has the `manifest` field instead of `hashes` and `filenames`.
2. Open a known old expense that still has a legacy `expenses.attachment` value and no `attachment_document`.
   - The details page should show the attachment.
   - `GET /api/expenses/attachment/{id}` should download or preview the file.
   - The file should still resolve from the old `expenses` storage prefix.
3. Create a normal user expense with a new PDF, PNG, or JPEG attachment.
   - The expense should save through `/api/expenses`.
   - The expense row should have `attachment_document` set.
   - The expense row should have blank legacy `attachment` and `attachment_hash` fields.
   - The linked `expense_documents` row should have the filename, hash, and `uploaded_by`.
   - The stored object should live under the `expense_documents` collection prefix.
   - The details/list attachment link should download through `/api/expenses/attachment/{id}`.
4. Attempt a duplicate upload as a normal non-`book_keeper` user.
   - The request should fail with the existing `attachment.duplicate_file` style error.
   - No new `expense_documents` row should be created.
5. Create a PO-backed expense as a `book_keeper` using a duplicate attachment.
   - The new expense should save.
   - It should reference the existing `expense_documents` row.
   - It should not create a second document row with the same hash.
6. From an expense details page as a `book_keeper`, use "Create another expense with this attachment".
   - The PO search flow should preserve `source_expense`.
   - The created PO-backed expense should share the same `attachment_document`.
   - The target expense should still inherit its normal PO-specific defaults.
7. Edit a legacy-only expense without changing its file.
   - The edit should save.
   - The expense should gain an `attachment_document`.
   - The old legacy filename should remain available for Phase 1 fallback.
   - The copied document-backed attachment should download successfully.
8. Generate a receipt ZIP for a date/week that includes both legacy-only and document-backed expenses.
   - The ZIP should include both files.
   - Re-running the same report should hit the regenerated `zip_cache` record.
   - Changing the attachment set later should miss the cache and regenerate the ZIP.
9. Run the legacy expense export/writeback path for rows created or edited after Phase 1.
   - The payload should still expose effective `attachment` and `attachmentHash` values.
   - Downstream consumers should not need to know about `attachment_document`.
10. Monitor the first production users for errors around:
    - `POST /api/expenses`;
    - `PATCH /api/expenses/{id}`;
    - `GET /api/expenses/attachment/{id}`;
    - weekly receipt ZIP generation;
    - legacy expense export/writeback.

After the smoke test passes, keep Phase 1 running in mixed-storage mode for
several normal business days before starting Phase 2. During that soak period,
prefer roll-forward fixes for any issue that appears. Avoid reverting to
pre-Phase-1 code unless there is a severe incident and a data repair plan has
already accounted for document-backed expenses with blank legacy attachment
fields and reused documents that cannot cleanly map back to the old unique
`expenses.attachment_hash` invariant.

Before starting Phase 2, produce a small production baseline report:

- count of legacy-only expenses with `attachment != ''` and no `attachment_document`;
- count of document-backed expenses;
- count of `expense_documents` rows referenced by more than one expense;
- count of document-backed expenses whose legacy `attachment` field is blank;
- any attachment download, receipt ZIP, or writeback errors observed during the soak period.

### 12. Phase 1 backend tests

Implement backend tests before merging Phase 1.

Expense document creation and linking:

- creating an expense with a new attachment creates exactly one `expense_documents` record;
- the created expense has `attachment_document` set;
- the created expense does not write a new legacy `expenses.attachment` value;
- the stored document hash matches the uploaded file content;
- the document `uploaded_by` field is the authenticated user.

Duplicate hash behavior:

- non-`book_keeper` uploading a file whose hash already exists in `expense_documents` receives the existing field-level duplicate attachment error;
- non-`book_keeper` uploading a file whose hash exists only on a legacy expense is still rejected;
- `book_keeper` uploading a duplicate file for a PO-backed expense links the existing document and creates no duplicate document;
- `book_keeper` uploading a duplicate file for a non-PO expense is rejected.

Source expense reuse:

- `book_keeper` can create a PO-backed expense using `source_expense` when the source has an `attachment_document`;
- `book_keeper` can create a PO-backed expense using `source_expense` when the source is legacy-only, and the source is migrated to `expense_documents` first;
- non-`book_keeper` using `source_expense` is rejected;
- `source_expense` is rejected when the source expense is not visible to the caller;
- `source_expense` is rejected when the source expense has no effective attachment;
- `source_expense` is rejected when the target expense has no PO.

Legacy fallback and auto-migration:

- expense details return a downloadable attachment for legacy-only expenses;
- receipt zip generation includes both legacy-only and document-backed expense attachments;
- legacy writeback/export includes effective attachment filename and hash for both legacy-only and document-backed expenses;
- editing a legacy-only expense without replacing its file creates or reuses an `expense_documents` row and links it;
- editing a legacy-only expense with a replacement file uses the replacement document and does not migrate the old file;
- removing an attachment from an expense type that permits no attachment clears the effective attachment relation/fields.

Regression coverage:

- existing PO-backed expense validation still enforces active PO, matching job, matching currency, and cumulative overflow behavior;
- existing bookkeeper-on-behalf validation still requires `book_keeper`, a PO-backed target, valid payment type, and PO owner matching;
- submitted expenses still cannot be edited through the generic expense save path;
- direct generic collection file upload to `expenses.attachment` is converted to `expense_documents` and does not persist a new legacy expense attachment;
- custom `POST /api/expenses` creates a document-backed expense;
- custom `PATCH /api/expenses/{id}` updates through the same document-first workflow;
- a failed final expense save after document creation rolls back the `expense_documents` row so the unique document hash cannot block a later valid upload.

## Phase 2: Backfill Existing Legacy Attachments

Phase 2 can happen days, weeks, or months after Phase 1. The app will already support mixed legacy/document attachment storage.

The backfill should be idempotent and split into prepare, manual S3 copy, verify, and link steps. This prevents the app from pointing an expense at a document path before the file exists in S3.

### 1. Add a backfill command

Add a command under `app/cmd/backfill_expense_documents`.

It should support at least these modes:

```text
prepare
verify
link
report
```

The command uses the app database and PocketBase collection metadata so it can discover:

- `expenses` collection id;
- `expense_documents` collection id;
- file names;
- record ids;
- hashes.

### 2. Prepare mode

`prepare` scans for expenses that still need migration:

```sql
SELECT *
FROM expenses
WHERE attachment != ''
  AND (attachment_document = '' OR attachment_document IS NULL)
```

For each row:

1. Determine the legacy S3 key:

   ```text
   {expenses_collection_id}/{expense_id}/{expenses.attachment}
   ```

2. Determine the attachment hash:

   - use `expenses.attachment_hash` when present;
   - if blank, read the legacy file and compute SHA-256;
   - if the file cannot be read, record an error and skip that expense.

3. Find an existing `expense_documents` row by `attachment_hash`.

4. If no document exists, create one with:

   - `attachment` set to the same filename as the legacy attachment;
   - `attachment_hash` set to the computed hash;
   - `uploaded_by` set to `expenses.creator` when present, otherwise `expenses.uid`.

5. Do not set `expenses.attachment_document` yet unless the document's S3 object already exists and verifies.

6. Write a manifest row containing:

   ```text
   expense_id
   expense_attachment
   attachment_hash
   expense_document_id
   document_attachment
   old_s3_key
   new_s3_key
   copy_required
   status
   ```

7. Write an executable copy script derived from the manifest.

Recommended output paths:

```text
tmp/expense_document_backfill/manifest.tsv
tmp/expense_document_backfill/copy_s3.sh
tmp/expense_document_backfill/errors.tsv
```

`prepare` must be safe to rerun. If it sees a document it created earlier, it should reuse it and preserve the same `expense_document_id`.

### 3. Manual S3 copy

After `prepare`, manually copy the existing S3 objects from the old expense prefixes to the new expense document prefixes.

This manual copy happens after document records are prepared and before expenses are linked.

For each manifest row with `copy_required = true`, copy:

```text
from: {expenses_collection_id}/{expense_id}/{expenses.attachment}
to:   {expense_documents_collection_id}/{expense_document_id}/{expense_documents.attachment}
```

Example key shape:

```text
from: o1vpz1mm7qsfoyy/b4o6xph4ngwx4nw/receipt.pdf
to:   {expense_documents_collection_id}/{expense_document_id}/receipt.pdf
```

The generated `copy_s3.sh` should contain explicit copy commands so the operator can review exactly what will move before running it. It should not delete the old objects.

Example command shape:

```bash
aws s3 cp "s3://$TYBALT_S3_BUCKET/o1vpz1mm7qsfoyy/{expense_id}/{filename}" \
  "s3://$TYBALT_S3_BUCKET/{expense_documents_collection_id}/{expense_document_id}/{filename}"
```

Do not set `expenses.attachment_document` for rows whose new S3 object has not been copied and verified.

### 4. Verify mode

After the manual S3 copy, run `verify`.

`verify` reads the manifest and checks every target object:

1. New S3 object exists.
2. New S3 object can be read through PocketBase filesystem APIs.
3. SHA-256 of the copied object matches `expense_documents.attachment_hash`.
4. Existing legacy source object still exists.

Rows that fail verification remain unlinked. Write failures to:

```text
tmp/expense_document_backfill/verify_errors.tsv
```

`verify` must be safe to rerun after recopying failed rows.

### 5. Link mode

After `verify` succeeds for a row, `link` updates the expense:

```text
expenses.attachment_document = expense_document_id
```

Leave these fields in place during the first backfill pass:

```text
expenses.attachment
expenses.attachment_hash
```

Keeping the legacy fields until cleanup gives a rollback path and makes it easier to audit the migration.

`link` should be transactional for DB changes and idempotent:

- skip rows already linked to the expected document;
- flag rows linked to a different document;
- never overwrite a non-empty `attachment_document` without an explicit force flag.

### 6. Report mode

`report` summarizes migration progress:

- total expenses with legacy attachment;
- total with `attachment_document`;
- total still legacy-only;
- total document-backed but missing target S3 object;
- duplicate hashes sharing one document;
- rows with missing legacy files;
- rows with blank or invalid hash.

This report is the acceptance checkpoint before cleanup.

### 7. Cleanup after successful backfill

Only after the report shows no legacy-only attachment rows:

1. Remove all remaining code paths that rely on legacy fallback.
2. Remove `expenses.attachment`.
3. Remove `expenses.attachment_hash`.
4. Remove the old unique index on `expenses.attachment_hash`.
5. Update generated PocketBase types.
6. Update imports/exports/writeback queries so `expense_documents` is the only source for expense attachments.
7. Remove any UI handling that displays legacy expense attachment links.

Cleanup should be its own PR/release after the backfill has been verified in production.

### 8. Phase 2 backend tests

Implement backend tests before running Phase 2 in production.

Prepare mode:

- `prepare` creates document records for legacy-only expenses with valid attachments;
- `prepare` reuses an existing document when another expense has the same hash;
- `prepare` uses `expenses.attachment_hash` when present;
- `prepare` computes the SHA-256 hash from the legacy file when `expenses.attachment_hash` is blank;
- `prepare` records an error and skips rows whose legacy file cannot be read;
- `prepare` writes a manifest with correct old and new S3 keys;
- rerunning `prepare` is idempotent and preserves previously created document ids.

Manual copy support:

- generated copy commands use the legacy expense collection prefix as the source;
- generated copy commands use the `expense_documents` collection prefix and document id as the destination;
- rows that already have a verified destination object are marked as not requiring copy;
- generated scripts never delete legacy objects.

Verify mode:

- `verify` succeeds when the copied destination object exists and its SHA-256 matches the document hash;
- `verify` fails when the destination object is missing;
- `verify` fails when the copied destination object hash does not match;
- `verify` fails when the legacy source object is missing;
- rerunning `verify` after correcting a failed copy succeeds without changing linked expenses.

Link mode:

- `link` sets `expenses.attachment_document` only for verified rows;
- `link` leaves `expenses.attachment` and `expenses.attachment_hash` untouched during the backfill pass;
- `link` skips rows already linked to the expected document;
- `link` refuses to overwrite a row linked to a different document unless an explicit force mode is used;
- `link` is transactional for DB updates and leaves no partially linked batch after an injected failure.

Report and cleanup readiness:

- `report` counts legacy-only, document-backed, missing-copy, missing-source, and duplicate-hash rows correctly;
- cleanup tests verify the app no longer needs legacy fallback after all rows are linked;
- post-cleanup export/report tests use only `expense_documents` as the source of expense attachment data.

## Testing Requirements

Backend tests should cover:

- new expense with a new document;
- duplicate upload rejected for non-bookkeeper;
- duplicate upload reused for bookkeeper on PO-backed expense;
- duplicate upload rejected for bookkeeper on non-PO expense;
- source expense reuse accepted for bookkeeper;
- source expense reuse rejected for non-bookkeeper;
- source expense reuse rejected when source is not visible;
- source expense reuse rejected when source has no effective attachment;
- legacy fallback read for expense details and receipt generation;
- legacy edit auto-migrates to `expense_documents`;
- replacement attachment on a legacy expense uses the replacement document rather than migrating the old file;
- backfill prepare is idempotent;
- backfill verify refuses bad or missing copies;
- backfill link skips already-linked rows and refuses mismatched links.

Frontend tests or manual checks should cover:

- ordinary users see the same expense form and attachment control;
- bookkeepers see the details-page action only when an attachment exists;
- source reuse lands on the existing PO-backed expense form;
- validation errors still display next to the attachment field.

Integration/manual testing against the dumped production DB should include:

- mixed legacy and document-backed expenses in details/list/tracking;
- receipt zip generation for a mixed pay period;
- legacy writeback/export for a mixed updated-after range;
- editing old rows with legacy attachments;
- the exact generated S3 copy manifest for Phase 2.

## Operational Notes

- Phase 1 requires a normal deployment but not an outage window.
- Phase 1 should be deployed only after testing locally on `multi_use_attachment` against a dumped production DB.
- Phase 2 does not need to be scheduled immediately.
- The manual S3 copy belongs to Phase 2, after `prepare` and before `verify`/`link`.
- The old S3 objects under the `expenses` collection prefix should not be deleted during Phase 2. Deletion, if desired, should be considered only after a separate retention/rollback decision.
- The `expense_documents.attachment_hash` unique index remains the long-term duplicate prevention mechanism.
- `book_keeper` reuse changes only which expense may point at an existing document. It never creates two document records with the same hash.

## Phase 3: cleanup

After everything is fully migrated and we're only using `expense_documents`, we can remove all the fallback code completely so it's a tight implementation once again.
