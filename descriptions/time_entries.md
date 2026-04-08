# Time Entries

This note documents create/update behavior for `time_entries`, with special
attention to how the `branch` field is resolved.

## Branch Resolution

`time_entries.branch` now follows the same precedence as `purchase_orders`:

1. If `job` is set, `branch` is forced to the selected job's branch.
2. If `job` is blank and `branch` is blank, `branch` defaults from the user's
   `admin_profiles.default_branch`.
3. If `job` is blank and `branch` is already set, that explicit branch is
   preserved.

This means time entries no longer always derive branch from
`admin_profiles.default_branch`.

## Related Behavior

- When a job is present, the job must exist and be valid for time tracking.
- When a job is present, `role` is required.
- When a job is present, the selected division must be allocated to that job.
- If a referenced job exists but has no `branch`, the create/update is rejected.
- Branch claim enforcement applies to the resolved branch value.

## UI Exposure of Rule (3)

Rule (3) — preserving an explicit non-default branch on a jobless time entry —
is currently **only reachable via the API**. The SvelteKit time-entry editor
does **not** expose a branch picker by default.

The picker exists in `ui/src/lib/components/TimeEntriesEditor.svelte` but is
gated by an in-code feature flag:

```ts
const EXPLICIT_BRANCH_PICKER_ENABLED = false;
```

While the flag is `false`:

- the editor never renders a branch field
- the editor never sends a `branch` value in create/update payloads
- the backend therefore always falls through to rule (2) (default branch)
  for entries created via the UI

When the flag is flipped to `true`, the picker mirrors the
`PurchaseOrdersEditor` behavior: visible only when no job is selected,
filtered by `branches.allowed_claims`, defaulting to the caller's default
branch, and "pinned" once the user manually changes it.

The flag is intentionally code-only (no env var, no `appConfig` toggle, no
user claim) so that enabling it requires a code review and a merged PR. This
is to give stakeholders time to discuss whether jobless cross-branch time
entries are a workflow we want to surface to end users at all.

## Copy To Tomorrow

`POST /api/time_entries/{id}/copy_to_tomorrow` copies the source entry's schema
fields into a new record and then runs normal time-entry processing. As a
result:

- copied entries with a job resolve branch from the job
- copied entries without a job preserve an explicit branch from the source entry
- copied entries without a job and without a branch fall back to the user's
  default branch
