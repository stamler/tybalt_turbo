# Turbo Po Approver Props Roundtrip Runbook

## Scope

This runbook documents the `po_approver_props` synchronization loop required for kind-aware PO approvals.

## Data Flow

1. Turbo endpoint `/api/export_legacy/expenses/{updatedAfter}` emits `poApproverProps`.
2. Legacy `turboSync` writes payload rows into Firestore `TurboPoApproverProps`.
3. Legacy `syncToSQL` exports Firestore rows into MySQL `TurboPoApproverProps`.
4. `import_data --export` writes `parquet/PoApproverProps.parquet`.
5. `import_data --import --users` upserts PocketBase `po_approver_props`.

## Authority Rules

- Turbo rows are authoritative per `uid`.
- Missing Turbo row for a `uid` falls back to synthesized values.
- Missing/invalid required fields in Turbo rows are fast-fail.

## Required Contract Fields

- `id`
- `uid`
- `max_amount`
- `project_max`
- `sponsorship_max`
- `staff_and_social_max`
- `media_and_event_max`
- `computer_max`
- `divisions` (JSON array string)
- `created`
- `updated`

## Verification Checklist

1. Turbo response contains `poApproverProps` key.
2. Firestore `TurboPoApproverProps` has expected row count.
3. MySQL `TurboPoApproverProps` is upserted and cleanup query runs.
4. `parquet/PoApproverProps.parquet` exists after `--export`.
5. `--import --users` recreates/links `user_claims` and writes `po_approver_props`.
6. Turbo-sourced users retain non-zero kind-specific limits after rerun.
