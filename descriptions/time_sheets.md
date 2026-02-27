# Time Sheets UI Actions (Current)

This note documents where review actions currently appear in the UI.

## Surfaces

- Main list: `/time/sheets/list` (`ui/src/routes/time/sheets/list/+page.svelte`)
- Pending queue: `/time/sheets/pending` (`ui/src/routes/time/sheets/pending/+page.svelte`)
- Details page: `/time/sheets/{id}/details` (`ui/src/routes/time/sheets/[id]/details/+page.svelte`)

## Action Placement

- Main list (`/time/sheets/list`) currently includes inline actions:
  - `Recall` (unbundle)
  - `Approve` (when not already approved)
  - `Reject`
  - `Share`
- Pending queue (`/time/sheets/pending`) is detail-first:
  - list rows expose only `Details`
  - approve/reject is performed from the details page
- Details page (`/time/sheets/{id}/details`) includes review actions based on state:
  - pending: `Recall`, `Approve`, `Reject`
  - approved/not committed: `Commit` (requires `commit` claim or `showAllUi`), `Reject`
  - rejected: `Recall`

## Important Clarification

Unlike the current purchase-order and expense list policy, timesheets are not fully details-first yet: `/time/sheets/list` still exposes inline `Approve`/`Reject` controls.
