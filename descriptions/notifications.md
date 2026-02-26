# Notification System

## Overview

The notification system lives in `app/notifications/notifications.go`. It creates notification records in the `notifications` table and delivers them as emails via the app's configured mail client. There are two delivery models:

1. **Immediate** — the notification is created and sent right away (used by event-driven paths like PO approval, rejection, timesheet sharing)
2. **Batched** — notifications are created in `pending` status and a cron job sends them later (used by periodic reminders)

## Status Lifecycle

Every notification record transitions through these statuses:

```text
pending → inflight → sent
                   → error
```

- **pending** — record created, awaiting delivery
- **inflight** — template rendered, email handed to SMTP goroutine
- **sent** — SMTP delivery succeeded
- **error** — SMTP delivery failed (error message stored in `error` column)

The `status_updated` column is set on every transition: by the `WriteStatusUpdated` hook for `pending` (on record creation), by `sendNotificationByID` raw SQL for `pending→inflight`, and by `updateNotificationStatus` raw SQL for `inflight→sent` and `inflight→error`.

## Architecture Diagram

```mermaid
flowchart TB
    subgraph "Trigger Sources"
        PO_CREATE["PO Create/Update<br/><i>hooks/hooks.go:142-163</i>"]
        PO_APPROVE["PO Approve<br/><i>routes/purchase_orders.go:464,478</i>"]
        PO_REJECT["PO Reject<br/><i>routes/purchase_orders.go:793</i>"]
        TS_REJECT["Timesheet Reject<br/><i>notifications.go:870</i>"]
        EXP_REJECT["Expense Reject<br/><i>notifications.go:961</i>"]
        TS_SHARE["Timesheet Share<br/><i>notifications.go:1056</i>"]
        CRON["Cron Jobs<br/><i>cron/cron.go</i>"]
    end

    subgraph "Delivery Strategy Selection"
        DEFERRED["Deferred Send Path<br/><i>create now, send after save</i>"]
        IMMEDIATE["Immediate Send Path<br/><i>create + send inline</i>"]
        BATCHED["Batched Send Path<br/><i>create now, cron sends later</i>"]
    end

    PO_CREATE --> DEFERRED
    PO_APPROVE --> IMMEDIATE
    PO_REJECT --> IMMEDIATE
    TS_REJECT --> IMMEDIATE
    EXP_REJECT --> IMMEDIATE
    TS_SHARE --> IMMEDIATE
    CRON --> BATCHED

    subgraph "Deferred Send Path (PO Create/Update)"
        direction TB
        PPO["ProcessPurchaseOrder()<br/><i>hooks/purchase_orders.go:615</i><br/>validates + cleans record"]
        PPO --> CREATE_ONLY["createPOApprovalRequiredNotification()<br/><i>hooks/purchase_orders.go:598</i><br/>calls CreateNotificationWithUser()"]
        CREATE_ONLY --> CNWU_PUB
        CREATE_ONLY --> |"returns notificationID"| HOOK_NEXT
        HOOK_NEXT{"e.Next()<br/>PO save pipeline"}
        HOOK_NEXT --> |"success"| SEND_AFTER["sendNotificationAfterSave()<br/><i>hooks/hooks.go:31</i>"]
        HOOK_NEXT --> |"failure"| DELETE_ORPHAN["deleteOrphanedNotification()<br/><i>hooks/hooks.go:48</i><br/>deletes pending record"]
        SEND_AFTER --> SEND_BY_ID_PUB
    end

    subgraph "Immediate Send Path (Approve/Reject/Share)"
        direction TB
        CASN["CreateAndSendNotificationWithUser()<br/><i>notifications.go:340</i><br/><b>public</b>"]
        CASN --> CNWU2["createNotificationWithUser()<br/><i>notifications.go:368</i>"]
        CNWU2 --> |"notificationID"| SEND_INLINE["sendNotificationByID()<br/><i>notifications.go:112</i>"]
    end

    subgraph "Immediate Send Path (Queue* functions)"
        direction TB
        QUEUE_FN["QueueTimesheetRejectedNotifications()<br/>QueueExpenseRejectedNotifications()<br/>QueueTimesheetSharedNotifications()<br/><i>notifications.go:870,961,1056</i>"]
        QUEUE_FN --> CNWU3["createNotificationWithUser()<br/><i>notifications.go:368</i><br/><b>private — called directly</b>"]
        CNWU3 --> |"notificationID != ''"| SEND_INLINE2["sendNotificationByID()<br/><i>notifications.go:112</i>"]
        CNWU3 --> |"notificationID == ''"| SKIP_COUNT["skip createdCount++<br/><i>feature disabled</i>"]
    end

    subgraph "Batched Send Path (Cron Reminders)"
        direction TB
        QUEUE_BATCH["QueuePoSecondApproverNotifications()<br/>QueueTimesheetSubmissionReminders()<br/>QueueTimesheetApprovalReminders()<br/>QueueExpenseApprovalReminders()<br/><i>notifications.go:457,517,654,762</i>"]
        QUEUE_BATCH --> CNWU4["createNotificationWithUser()<br/><i>notifications.go:368</i>"]
        CNWU4 --> PENDING_DB[("notifications table<br/>status = 'pending'")]
        PENDING_DB --> SEND_ALL["SendNotifications()<br/><i>notifications.go:293</i><br/>loops SendNextPendingNotification()"]
        SEND_ALL --> SNPN["SendNextPendingNotification()<br/><i>notifications.go:256</i><br/>finds next pending ID"]
        SNPN --> SEND_BY_ID2["sendNotificationByID()<br/><i>notifications.go:112</i>"]
    end

    subgraph "Record Creation Layer"
        CNWU_PUB["CreateNotificationWithUser()<br/><i>notifications.go:323</i><br/><b>public — returns (string, error)</b>"]
        CNWU_PUB --> CNWU_PRIV
        CNWU_PRIV["createNotificationWithUser()<br/><i>notifications.go:368</i><br/><b>private — shared implementation</b>"]
        CNWU_PRIV --> FEAT_CHECK{"IsNotificationFeatureEnabled()<br/><i>utilities/config.go</i>"}
        FEAT_CHECK --> |"disabled / error"| SKIP_RETURN["return '', nil<br/><i>fail-closed</i>"]
        FEAT_CHECK --> |"enabled"| FIND_TPL["Find notification_templates<br/>record by code"]
        FIND_TPL --> BUILD_REC["Build notification record<br/>status='pending', marshal data JSON"]
        BUILD_REC --> SAVE_REC["app.Save(record)<br/><i>triggers WriteStatusUpdated hook</i>"]
        SAVE_REC --> |"record.Id"| RETURN_ID["return notificationID"]
    end

    subgraph "Send Engine"
        SEND_BY_ID_PUB["SendNotificationByID()<br/><i>notifications.go:248</i><br/><b>public</b>"]
        SEND_BY_ID_PUB --> SEND_PRIV
        SEND_PRIV["sendNotificationByID()<br/><i>notifications.go:112</i><br/><b>private — shared workhorse</b>"]

        SEND_PRIV --> TX_START["RunInTransaction()"]
        TX_START --> FETCH["SELECT notification + JOINs<br/><i>profiles, users, templates</i><br/>WHERE id=? AND status='pending'"]
        FETCH --> |"no rows"| TX_NOOP["return nil<br/><i>already sent or missing</i>"]
        FETCH --> |"found"| UNMARSHAL["JSON.Unmarshal(data)"]
        UNMARSHAL --> RENDER["template.Execute()<br/><i>Go text/template</i>"]
        RENDER --> PLACEHOLDER_CHECK{"unresolvedLegacyPlaceholder()?"}
        PLACEHOLDER_CHECK --> |"found"| TX_ERR["return error"]
        PLACEHOLDER_CHECK --> |"clean"| BUILD_MSG["Build mailer.Message<br/><i>From, To, Subject, Text</i>"]
        BUILD_MSG --> INFLIGHT["UPDATE status='inflight'<br/><i>raw SQL in transaction</i>"]
        INFLIGHT --> TX_COMMIT["commit transaction"]

        TX_COMMIT --> GOROUTINE["go func() — async email send"]
        GOROUTINE --> SMTP["app.NewMailClient().Send()"]
        SMTP --> UPDATE_STATUS["updateNotificationStatus()<br/><i>notifications.go:82</i>"]
        UPDATE_STATUS --> |"success"| SENT["status='sent'<br/>+ status_updated"]
        UPDATE_STATUS --> |"failure"| ERR_STATUS["status='error'<br/>+ error message<br/>+ status_updated"]
    end

    style DEFERRED fill:#e1f5fe
    style IMMEDIATE fill:#e8f5e9
    style BATCHED fill:#fff3e0
    style SKIP_RETURN fill:#ffebee
    style DELETE_ORPHAN fill:#ffebee
    style TX_NOOP fill:#f5f5f5
    style TX_ERR fill:#ffebee
```

## Function Reference

### Public API (`notifications` package)

| Function                            | Signature                                                                   | Description                                                                       |
|-------------------------------------|-----------------------------------------------------------------------------|-----------------------------------------------------------------------------------|
| `CreateNotification`                | `(app, templateCode, recipientUID, data, system) error`                     | Create-only. Discards notification ID. For callers that don't need deferred send. |
| `CreateNotificationWithUser`        | `(app, templateCode, recipientUID, data, system, actorUID) (string, error)` | Create-only. Returns notification ID for deferred send or cleanup.                |
| `CreateAndSendNotification`         | `(app, templateCode, recipientUID, data, system) error`                     | Create + immediate send. Delegates to `CreateAndSendNotificationWithUser`.        |
| `CreateAndSendNotificationWithUser` | `(app, templateCode, recipientUID, data, system, actorUID) error`           | Create + immediate send. Send errors are logged, not propagated.                  |
| `SendNotificationByID`              | `(app, notificationID) error`                                               | Send a specific pending notification. No-op if already sent or missing.           |
| `SendNextPendingNotification`       | `(app) (remaining int64, err error)`                                        | Pick and send the next pending notification. Returns remaining count.             |
| `SendNotifications`                 | `(app) (int64, error)`                                                      | Loop `SendNextPendingNotification` until no pending remain.                       |
| `BuildActionURL`                    | `(app, path) string`                                                        | Build absolute URL from app-relative path.                                        |
| `WriteStatusUpdated`                | `(app, e) error`                                                            | Hook: sets `status_updated` on status change or new record.                       |

### Private internals (`notifications` package)

| Function                      | Description                                                                                                                                               |
|-------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------|
| `createNotificationWithUser`  | Shared implementation for all Create variants. Checks feature flag, finds template, builds record, calls `app.Save()`. Returns `(notificationID, error)`. |
| `sendNotificationByID`        | Shared send workhorse. Transaction: fetch + render + inflight. Then async goroutine for SMTP.                                                             |
| `updateNotificationStatus`    | Raw SQL update of `status`, `error`, and `status_updated` after SMTP completes. Uses `NonconcurrentDB()` to avoid hook side-effects.                      |
| `unresolvedLegacyPlaceholder` | Safety check for `{APP_URL}` / `{:RECORD_ID}` in rendered text.                                                                                           |

### Queue functions (`notifications` package)

| Function                                   | Delivery  | Description                                                           |
|--------------------------------------------|-----------|-----------------------------------------------------------------------|
| `QueuePoSecondApproverNotifications`       | Batched   | Create notifications for second-stage approvers, optionally send all. |
| `QueueTimesheetSubmissionReminders`        | Batched   | Remind users missing timesheets. Dedup by week.                       |
| `QueueTimesheetSubmissionRemindersForWeek` | Batched   | Implementation for a specific week ending.                            |
| `QueueTimesheetApprovalReminders`          | Batched   | Remind managers of pending timesheets. Dedup by 24h window.           |
| `QueueExpenseApprovalReminders`            | Batched   | Remind managers of pending expenses. Dedup by 24h window.             |
| `QueueTimesheetRejectedNotifications`      | Immediate | Notify employee + rejector + manager on timesheet rejection.          |
| `QueueExpenseRejectedNotifications`        | Immediate | Notify employee + rejector + manager on expense rejection.            |
| `QueueTimesheetSharedNotifications`        | Immediate | Notify newly added timesheet viewers.                                 |

### Hook helpers (`hooks` package)

| Function                               | Description                                                                                                           |
|----------------------------------------|-----------------------------------------------------------------------------------------------------------------------|
| `sendNotificationAfterSave`            | Trigger `SendNotificationByID` after PO save succeeds. No-op if ID is empty. Logs send errors without propagating.    |
| `deleteOrphanedNotification`           | Delete a notification record whose PO save failed. Logs a warning on DB lookup failure (orphan may be sent by cron).  |
| `createPOApprovalRequiredNotification` | Create-only wrapper for `po_approval_required`. Returns notification ID for deferred send.                            |

## Key Design Decisions

### Deferred send for PO create/update hooks

`ProcessPurchaseOrder` runs in a pre-save request hook (before `e.Next()`). The notification record is created during validation so it can reference the PO ID, but email delivery is deferred until after `e.Next()` succeeds. If the save pipeline fails, `deleteOrphanedNotification` removes the pending record so the cron doesn't deliver it later.

### Immediate send for route handlers

PO approval, rejection, and the Queue* event-driven functions call `CreateAndSendNotificationWithUser` or `createNotificationWithUser` + `sendNotificationByID` directly. These run after the business record is already persisted, so there's no orphan risk.

### Raw SQL for post-send status updates

`updateNotificationStatus` uses `NonconcurrentDB()` with raw SQL instead of `app.Save()` to avoid triggering PocketBase model/record hooks from an async goroutine. It explicitly sets `status_updated` in the SQL since the `WriteStatusUpdated` hook won't fire.

### Feature flags (fail-closed)

`createNotificationWithUser` checks `IsNotificationFeatureEnabled()` before creating a record. If the config read fails, it returns `("", nil)` — skipping creation silently. This keeps business workflows non-blocking while ensuring notifications are never sent unless explicitly enabled.

### createdCount accuracy in Queue* functions

The immediate-send Queue* functions (`QueueTimesheetRejected`, `QueueExpenseRejected`, `QueueTimesheetShared`) call `createNotificationWithUser` directly and only increment `createdCount` when `notificationID != ""`. This prevents inflated counts when the feature is disabled.
