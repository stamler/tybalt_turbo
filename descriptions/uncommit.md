# Uncommit

## Summary

This staged change adds an uncommit workflow for holders of the `admin` claim for committed `time_sheets` and `expenses` records.

The goal is narrow:

- let holders of the `admin` claim open the relevant details pages for committed records
- let holders of the `admin` claim reverse the committed state from those details pages
- avoid PocketBase rule changes
- avoid broadening access for holders of the `admin` claim to uncommitted workflow data

The implementation relies on custom application routes rather than collection rule changes.

## Scope

This change includes:

- new uncommit endpoints for holders of the `admin` claim for `time_sheets` and `expenses`
- a new custom time-sheet details endpoint
- an update to the existing expense details endpoint so holders of the `admin` claim can open committed expense details
- details-page UI actions for holders of the `admin` claim
- purchase-order reopening logic when an expense is uncommitted
- committed-only discovery for holders of the `admin` claim through the existing tracking surfaces

This change does not include:

- PocketBase rule changes for `time_sheets`, `time_entries`, or `expenses`
- broad access for holders of the `admin` claim to uncommitted expense or time-sheet records
- access for holders of the `admin` claim to the expense commit queue
- access for holders of the `admin` claim to time-sheet missing or not-expected views
- new notifications, audit-log views, or standalone search screens for holders of the `admin` claim

## New And Updated Routes

### New routes

- `GET /api/time_sheets/{id}/details`
- `POST /api/time_sheets/{id}/uncommit`
- `POST /api/expenses/{id}/uncommit`

### Updated route behavior

- `GET /api/expenses/details/{id}` now allows holders of the `admin` claim to load committed expense details
- expense tracking routes now allow holders of the `admin` claim to access committed expense tracking data
- time-sheet tracking routes now allow holders of the `admin` claim to access committed time-sheet tracking data

## Authorization Model

### Uncommit authorization

Both uncommit endpoints require all of the following:

- an authenticated user
- the relevant editing capability to be enabled in app config
- the `admin` claim
- a record that is currently committed

The `commit` claim alone is not enough to uncommit a record.

### Details-page authorization

Holders of the `admin` claim can open details pages only for committed records.

For `expenses`:

- normal visibility rules still apply for owners, approvers, commit holders, and report holders
- holders of the `admin` claim get an additional committed-only details override
- holders of the `admin` claim do not get generic expense list visibility through this change

For `time_sheets`:

- the new details endpoint preserves the existing workflow-based access model
- holders of the `admin` claim are additionally allowed when the time sheet is committed
- holders of the `admin` claim are not allowed to open uncommitted time-sheet details

### Discovery authorization

Holders of the `admin` claim are intentionally given access only to committed-record discovery surfaces.

For `expenses`:

- holders of the `admin` claim can use expense tracking counts
- holders of the `admin` claim can use expense tracking lists
- holders of the `admin` claim cannot use the expense commit queue
- holders of the `admin` claim do not get org-wide access through `/api/expenses/list`
- holders of the `admin` claim do not get approver-style access through pending or approved list endpoints

For `time_sheets`:

- holders of the `admin` claim can use time-sheet tracking counts
- holders of the `admin` claim can use time-sheet tracking lists
- holders of the `admin` claim cannot use missing or not-expected time-sheet views

## Time-Sheet Details Endpoint

`GET /api/time_sheets/{id}/details` is a new custom endpoint that exists so the UI no longer depends on direct PocketBase collection reads for the details page.

The response includes:

- the `timeSheet` record
- expanded `items` from `time_entries`
- `approverInfo`

The details page then combines that with existing UI-side tally and metadata handling.

Authorization for this endpoint is:

- the owner can view
- approvers can view submitted sheets
- reviewers can view submitted sheets
- commit holders can view submitted and approved sheets
- report holders can view committed sheets
- holders of the `admin` claim can view committed sheets

If the record is missing or the user is not authorized, the endpoint returns a not-found style error rather than distinguishing between those cases.

## Uncommit Endpoint Behavior

### Shared behavior

`POST /api/time_sheets/{id}/uncommit` and `POST /api/expenses/{id}/uncommit` share the same core behavior.

The handler:

1. verifies the caller is authenticated
2. verifies editing is enabled for the relevant domain
3. verifies the caller has the `admin` claim
4. loads the record inside a transaction
5. verifies the record is currently committed
6. clears commit-related fields
7. saves the updated record

On success, the handler returns:

```json
{
  "message": "Record uncommitted successfully"
}
```

### Cleared fields

For both collections:

- `committed`
- `committer`

If the collection contains the field, the handler also clears:

- `committed_week_ending`

For `expenses`, the handler additionally clears:

- `pay_period_ending`

This means the record returns to an approved but uncommitted state. The handler does not reset `submitted` or `approved`.

### Error behavior

The handler returns:

- `403 unauthorized` when the caller lacks the `admin` claim
- `404 record_not_found` when the record does not exist
- `400 record_not_committed` when the record is not currently committed

If editing is disabled, the route returns the same editing-disabled error used elsewhere in the application for that domain.

## Expense Purchase Order Reopening

Uncommitting an expense may reopen a linked purchase order.

This logic runs only when:

- the expense has a linked purchase order
- the purchase order is currently `Closed`

If those conditions are not true, the purchase order is left alone.

### One-Time purchase orders

A closed one-time purchase order is reopened when uncommitting the expense leaves it with zero committed expenses.

### Recurring purchase orders

A closed recurring purchase order is reopened when the remaining committed expenses no longer exhaust the recurrence limit.

### Cumulative purchase orders

A closed cumulative purchase order is reopened when the remaining committed expense total drops below the purchase order total.

### Reopen behavior

When a purchase order is reopened, the handler:

- sets `status` to `Active`
- clears `closed`
- clears `closer`
- sets `closed_by_system` to `false`

## Tracking Behavior For Holders Of The `admin` Claim

Discovery for holders of the `admin` claim is intentionally committed-only.

### Expense tracking

Holders of the `admin` claim can use the same tracking routes used for committed expense workflow visibility.

They can:

- fetch expense tracking counts
- fetch expense tracking lists
- navigate from tracking results into committed expense details

They cannot:

- fetch the expense commit queue unless they also hold the `commit` claim
- use tracking access as a back door to uncommitted expense lists

### Time-sheet tracking

Holders of the `admin` claim can use time-sheet tracking counts and lists to discover committed time sheets.

The tracking list query limits results for holders of the `admin` claim to committed time sheets.

The tracking counts response also intentionally avoids exposing uncommitted workflow totals to holders of the `admin` claim:

- `committed_count` is real
- `approved_count` is forced to `0`
- `submitted_count` is forced to `0`
- `rejected_count` is forced to `0`

This lets holders of the `admin` claim discover committed records without broadening visibility into the rest of the time workflow.

## UI Behavior

### Expense details page

On the expense details page:

- holders of the `admin` claim who open a committed expense see an `Uncommit` button
- non-admin users do not get this button through this change
- after uncommit succeeds, the page refreshes in place

The page continues to rely on the existing custom details API, now with the committed-only visibility extension for holders of the `admin` claim.

### Time-sheet details page

The time-sheet details page now loads through the new custom details endpoint rather than direct PocketBase collection reads.

On the page:

- holders of the `admin` claim who open a committed time sheet see an `Uncommit` button
- the page refreshes its data after approve, commit, reject, and uncommit actions
- the details load now comes from the same server-side authorization model used for committed access by holders of the `admin` claim

The page also wraps its `time_entries` realtime subscription in a safe fallback so lack of collection subscription access does not break the details view.

### Navigation

The main layout adds access for holders of the `admin` claim to:

- `/time/tracking`
- `/expenses/tracking`

It does not add holders of the `admin` claim to the broader pending, approved, or commit-queue navigation items.

## Editing Disabled Behavior

The uncommit routes respect the same editing toggles as the existing edit workflows.

If expense editing is disabled:

- `POST /api/expenses/{id}/uncommit` is blocked

If time editing is disabled:

- `POST /api/time_sheets/{id}/uncommit` is blocked

This keeps uncommit aligned with the rest of the application's operational controls.

## Tests Covered By The Staged Change

The staged tests cover:

- holders of the `admin` claim can load committed expense details
- holders of the `admin` claim cannot load uncommitted expense details
- holders of the `admin` claim can uncommit committed expenses
- non-admin commit holders cannot uncommit expenses
- holders of the `admin` claim cannot uncommit uncommitted expenses
- expense uncommit can reopen one-time, recurring, and cumulative purchase orders
- holders of the `admin` claim can access expense tracking counts and tracking lists
- holders of the `admin` claim cannot use the org-wide expense list through this change
- holders of the `admin` claim cannot use pending expense lists through this change
- holders of the `admin` claim cannot use the expense commit queue through this change
- holders of the `admin` claim can load committed time-sheet details
- holders of the `admin` claim cannot load uncommitted time-sheet details
- holders of the `admin` claim can uncommit committed time sheets
- non-admin commit holders cannot uncommit time sheets
- holders of the `admin` claim cannot uncommit uncommitted time sheets
- holders of the `admin` claim can access time-sheet tracking counts and committed tracking lists
- holders of the `admin` claim cannot use missing time-sheet views through this change
- time and expense editing disabled states block uncommit

## Practical Outcome

After this staged change:

- holders of the `admin` claim can discover committed expenses and time sheets through tracking
- holders of the `admin` claim can open the corresponding details pages for committed records
- holders of the `admin` claim can reverse the committed state directly from those pages
- holders of the `admin` claim do not gain broad read access to uncommitted expense or time-sheet workflow data

That makes the feature useful for correcting committed records while keeping the access expansion intentionally narrow.
