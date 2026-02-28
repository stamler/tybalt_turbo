# Notification System

## Overview

The notification package is split by concern under `app/notifications/`:

- `types.go` — shared internal types (`DispatchArgs`, `ReminderJob`, `DeliveryMode`)
- `create.go` — creation and dispatch entry points
- `send.go` — send engine and status transitions
- `queue_events.go` — immediate event fan-out (reject/share paths)
- `queue_reminders.go` — batched reminder queueing and dedupe engine
- `helpers.go` — shared helper utilities

Behavior is unchanged:

1. **Immediate**: create and attempt send right away.
2. **Deferred/Batched**: create `pending` and send later via cron or explicit `SendNotifications`.

## Status Lifecycle

```text
pending -> inflight -> sent
                   -> error
```

- `pending`: record created and queued.
- `inflight`: template rendered and message accepted for async SMTP send.
- `sent`: SMTP send completed successfully.
- `error`: SMTP send failed; error text persisted.

## Core Internal Flow

### Dispatch Primitive

All create/dispatch callers use one public function:

`DispatchNotification(app, args DispatchArgs) (notificationID string, err error)`

- `DeliveryDeferred`: create only.
- `DeliveryImmediate`: create, then targeted send attempt.
- Send failures are logged but not returned for immediate mode to preserve non-blocking business operations.

### Queue Fan-out Helper

Immediate event queue functions use:

`createAndSendToRecipients(app, templateCode, recipients, data, system, actorUID, logContext)`

- Iterates recipients.
- Uses immediate dispatch mode.
- Increments `createdCount` only when a notification record is actually created.

### Reminder Queue Engine

Batched reminder functions share:

`queueReminderJob(app, job ReminderJob, send bool) error`

with:

- candidate-recipient query
- dedupe check via `notificationExists(...)`
- payload build
- deferred dispatch
- optional send tail via `sendQueuedIfRequested(...)`

This powers:

- `QueueTimesheetSubmissionRemindersForWeek`
- `QueueTimesheetApprovalReminders`
- `QueueExpenseApprovalReminders`

## Public API

- `DispatchNotification(app, args) (string, error)`
- `SendNotificationByID(app, notificationID) error`
- `SendNextPendingNotification(app) (remaining int64, err error)`
- `SendNotifications(app) (int64, error)`
- `BuildActionURL(app, path) string`
- `WriteStatusUpdated(app, e) error`

## Hook/Route Integration

- PO create/update still uses deferred create-then-send-after-save behavior through existing hooks.
- PO approve/reject route paths dispatch with `DeliveryImmediate` mode.
- Timesheet/expense reject and timesheet share event paths now share the same recipient fan-out helper.

## Design Notes

- Feature flags remain fail-closed in creation (`createNotificationWithUser`): on config read errors, creation is skipped (`"", nil`) and business flows continue.
- Async send status updates still use raw SQL through `NonconcurrentDB()` to avoid PocketBase hook side-effects in goroutines.
- Dedupe semantics are preserved:
  - timesheet submission reminders dedupe by recipient + template + `WeekEnding`
  - approval reminders dedupe by recipient + template in the last 24 hours

## PO Second Approval Notifications (`po_second_approval_required`)

Daily notification input for second-stage PO approvals.

### Source of Recipients

Recipients are read from:

- `pending_items_for_qualified_po_second_approvers`

Each row is:

- `id`: user id
- `num_pos_qualified`: count of POs currently visible to that user in post-timeout Stage 2

`QueuePoSecondApproverNotifications` sends `po_second_approval_required` to users where `num_pos_qualified > 0`.

### Qualification Rules (as implemented by the view)

A PO contributes to a user's count only when all are true:

1. PO is first-approved (`approved != ''`)
2. PO is not second-approved (`second_approval = ''`)
3. PO status is still `Unapproved`
4. PO is past Stage-2 timeout window (`approved + T`)
5. PO is dual-required for its kind (`approval_total > second_approval_threshold` and threshold > 0)
6. User is an active `po_approver`
7. User can approve the PO division
8. User is second-stage eligible for the PO kind/amount:
   - resolved limit column is used by kind/job context
   - limit `> second_approval_threshold`
   - limit `>= approval_total`

`T` is loaded from `app_config.purchase_orders.second_stage_timeout_hours` with backend fallback to `24` hours when missing/invalid/non-positive.

### Important Changes from Legacy Model

No longer used:

- `po_approval_thresholds`
- `upper_threshold` / `lower_threshold`
- tier-ceiling grouping
- `user_po_permission_data`

This notification flow now follows stage pools and timeout-based visibility, matching pending queue behavior.
