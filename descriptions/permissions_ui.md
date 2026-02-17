### UI permissions: reflect user claims in navigation and controls

Links: [UI should reflect user permissions #58](https://github.com/stamler/tybalt_turbo/issues/58)

#### Goals

- Drive visibility of links, routes, and action controls from the authenticated user's permissions.
- Keep backend rules/hooks authoritative; UI gating is for UX only.
- Provide a single, cacheable capability payload the UI can consume easily.

#### Current state (summary)

- Claims are stored in `claims` and linked via `user_claims`.
- Backend checks claims in hooks/utilities (e.g., `HasClaimByUserID`), and PocketBase rules guard collections (e.g., admin-only create/update on several collections).
- UI now loads `user_claims_summary` (global claims) and `user_po_approver_profile` (PO limits/divisions/claims) into `globalStore`, and displays claims in `profile/[uid]`.

#### Proposed model: user capabilities

Expose an aggregated, read-only “capabilities” view per user that the UI can read at login and refresh periodically.

- Data shape (example):

```json
{
  "id": "<uid>",
  "claims": ["admin", "busdev", "po_approver"],
  "booleans": {
    "can.view.admin": true,
    "can.edit.clients": true,
    "can.create.purchase_orders": true,
    "can.approve.purchase_orders": true,
    "can.view.reports": true
  },
  "limits": {
    "po.approval_max_amount": 2500
  },
  "lists": {
    "po.allowed_divisions": ["division_id_1", "division_id_2"]
  }
}
```

Notes:

- Booleans are coarse-grained feature gates used by nav and buttons.
- Limits and lists express constraints the UI needs (e.g., max approval and permitted divisions).
- `claims` echo the raw claims for debugging and future uses.

#### Backend work

1. Define the mapping from claims → capabilities

   - Centralize in Go (new `app/utilities/capabilities.go`) a registry that maps claim names to booleans/limits/lists.
   - Include derived logic (e.g., PO approver limits from `po_approver_props` and tiers).
   - Keep mapping in code rather than parsing `createRule/listRule/updateRule/viewRule` strings.

2. Expose `user_capabilities` read model

   - Add a PocketBase view collection via migration `user_capabilities` with `id = uid` and JSON fields:
     - `claims` (string[]), `booleans` (JSON), `limits` (JSON), `lists` (JSON).
   - Back it with a SQL `viewQuery` that joins `user_claims` (+ `claims`), and left-joins `po_approver_props` to compute approval `limits/lists`.
   - Alternatively, expose a lightweight HTTP route that returns this shape, but a PB view keeps the UI consistent with current data fetching.

3. Helpers for server logic (optional but useful)

   - Add `GetUserCapabilities(app, uid)` returning the same struct for reuse in hooks/routes when server-side decisions need the aggregate.

4. Keep enforcement on server
   - Do not loosen any existing PocketBase rules or hook checks.
   - Use `HasClaimByUserID`/existing checks as-is; capabilities are an additional, read-only projection.

#### UI work

1. Store and types

   - Add TypeScript types for `user_capabilities` in `ui/src/lib/pocketbase-types.ts`.
   - Create `capabilitiesStore` (or extend `globalStore`) to load `user_capabilities` by `id={:userId}` on navigation refresh (mirroring `user_claims_summary` + `user_po_approver_profile` patterns).
   - Expose helpers:
     - `can(flag: string): boolean` (reads `booleans[flag] === true`).
     - `limit(key: string): number | undefined`.
     - `list(key: string): string[]`.
   - Cache with `maxAge` like existing store to avoid flicker.

2. Navigation and route guards

   - In `+layout.svelte`, gate `navSections` and sidebar items using `capabilities.can("can.view.admin")`, etc.
   - Add simple route guards: if user navigates to an admin route without `can.view.admin`, redirect to home.

3. Component-level gating

   - Wrap action buttons/forms with capability checks, e.g.:
     - Hide “Approve” if `!can("can.approve.purchase_orders")` or show disabled with tooltip.
     - Use `limit("po.approval_max_amount")` and `list("po.allowed_divisions")` to constrain inputs and defaults.

4. Progressive rendering
   - Until capabilities load, render skeletons or hide restricted items by default to avoid flashing controls.

#### Mapping (initial draft)

- `admin` → `can.view.admin`, `can.view.reports`, `can.edit.clients`, `can.edit.users`.
- `busdev` → `can.edit.clients` (lead fields), possibly `can.view.pipeline` if present.
- `po_approver` (+ `po_approver_props`) → `can.create.purchase_orders`, `can.approve.purchase_orders`, limits for `po.approval_max_amount`, lists for `po.allowed_divisions`.
- Add more as features require; prefer additive booleans with consistent naming.

#### Testing

- Backend: unit test `GetUserCapabilities` for representative users/claims. Prefer seeding `test_pb_data/data.db` with fixture data instead of inserts during tests.
- UI: render tests for nav/components to verify gating with mocked stores.

#### Rollout

1. Ship backend `user_capabilities` view and UI store behind feature flag (default on).
2. Convert nav and a few critical screens (admin, purchase orders) first.
3. Incrementally gate remaining components.

#### Risks and mitigations

- Divergence between server rules and UI flags → keep claim→capability mapping next to server logic; review on claim/rule changes.
- Stale capabilities → store already refreshes per-navigation with maxAge; provide manual refresh on login/token refresh.
- Overly granular flags → start coarse; split as needed when a control needs finer distinction.
