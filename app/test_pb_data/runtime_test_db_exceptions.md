# Runtime Test DB Exceptions

This document records the remaining cases where tests still create, update, or delete data at runtime instead of relying entirely on the canonical fixture DB at `app/test_pb_data/data.db`.

The goal is to keep this list short and explicit. If a new test starts mutating setup data before execution, it should either:

- be moved into the fixture DB, or
- be added here with a clear reason why a single canonical DB cannot represent the case.

## Principles

- Prefer committed fixture rows in `data.db`.
- Do not mutate pre-existing fixture rows at test setup unless the case is inherently contradictory or fault-injection.
- Runtime mutation is still acceptable when the mutation is the behavior under test, rather than missing fixture data.

## 1. Contradictory Singleton Config State

These tests need one config row to be enabled, disabled, or missing across different scenarios. A single canonical DB cannot encode all of those states at once.

- [config_test.go](app/config_test.go)
  - `setupJobsEditingDisabledApp`
  - mutates `app_config.key='jobs'` to `{"create_edit_absorb": false}`
- [config_test.go](app/config_test.go)
  - `setupExpensesEditingDisabledApp`
  - inserts `app_config.key='expenses'` with `{"create_edit_absorb": false}`
- [legacy_purchase_orders_test.go](app/legacy_purchase_orders_test.go)
  - `setupLegacyPOFeatureDisabledApp`
  - mutates `app_config.key='purchase_orders'` to disable `enable_legacy_po_create_update`
- [utilities/config_test.go](app/utilities/config_test.go)
  - upserts and deletes `app_config` rows for `purchase_orders`, `notifications`, and `expenses`
  - these tests are explicitly about fallback/default parsing behavior for present, absent, and malformed config states
- [notifications_test.go](app/notifications_test.go)
  - `upsertNotificationsConfigRawValue(..., {"timesheet_shared":false})`
  - used by disabled-notification behavior tests

## 2. Negative Export Poison-Pill Fixture

This case must stay runtime-only because if it is seeded globally it breaks the happy-path export tests.

- [expenses_export_legacy_test.go](app/expenses_export_legacy_test.go)
  - `setupExpensesExportMissingLegacyUIDApp`
  - inserts transient `po_approver_props.id='pap_missing_legacy_uid'`
  - reason: the row intentionally has no legacy UID mapping, so seeding it in the base DB would make every export fail

## 3. Date-Relative Sequence Setup

This case depends on the current calendar date at test runtime, so a fixed canonical DB cannot permanently represent the needed “empty current period” state.

- [purchase_orders_test.go](app/purchase_orders_test.go)
  - `TestGeneratePONumber`
  - scenario: `first PO of the year`
  - deletes current-period POs before running
  - reason: the expected result depends on `time.Now()`

## 4. Fault Injection / Broken-System Simulation

These tests deliberately corrupt the app environment or inject downstream failures. This is not missing fixture data and should not be seeded.

- [purchase_orders_routes_test.go](app/purchase_orders_routes_test.go)
  - renames `expenses` to `expenses_broken` in one failure-path factory
- [purchase_orders_orphan_notifications_test.go](app/purchase_orders_orphan_notifications_test.go)
  - binds failing create/update hooks with `OnRecordCreateRequest` / `OnRecordUpdateRequest`
- [notifications_test.go](app/notifications_test.go)
  - renames `notifications` or `notification_templates` to `*_broken`
  - corrupts template content for send/render failure cases
- [absorb_test.go](app/absorb_test.go)
  - renames `claims` to `claims_broken` in one unsupported/failure path

## 5. Mutation Is The Behavior Under Test

These tests create or change rows because the test is specifically verifying the side effects of those mutations. They are not fixture gaps.

- [hooks/jobs_dirty_related_test.go](app/hooks/jobs_dirty_related_test.go)
  - creates/deletes categories
  - updates clients and contacts
  - primes `_imported` flags
  - reason: verifies related jobs are marked not imported when upstream records change
- [absorb_test.go](app/absorb_test.go)
  - creates absorb state through `routes.AbsorbRecords(...)`
  - inserts one synthetic `absorb_actions` row in a blocked-undo case
  - reason: absorb state is transient workflow state, not static seed data

## 6. Current Status

Everything else that was previously additive setup has been moved into `data.db` and should stay there unless it proves to be one of the exception types above.
