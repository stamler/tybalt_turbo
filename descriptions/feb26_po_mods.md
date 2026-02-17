# February 26 PO Approval Model Update (`feb26_po_mods.md`)

## Purpose

This document is the authoritative implementation spec for replacing the current global PO threshold model with a per-kind, two-stage routing model aligned to role authority intent in `authority_matrix.md`.

This spec is intended to be sufficient as the sole implementation document for backend, UI, migrations, tests, and rollout.

---

## Why This Change

## Problem in current model

The current `po_approval_thresholds` tiering model does two different jobs:

1. Determines whether a PO requires one or two approvals.
2. Denoises second-approver visibility via tier ceilings.

This causes policy coupling and misalignment with the authority matrix:

- Higher-authority approvers (Executives/Branch Managers) can be filtered out of second-approver candidate visibility by tier-ceiling logic even when they are valid approvers for that kind.
- Some code paths still rely on legacy `max_amount` assumptions despite newer per-kind limits (`project_max`, `sponsorship_max`, `computer_max`, etc.).

## Design intent

1. Keep denoising behavior (not identical to current threshold-tier denoising).
2. Make one-vs-two approval policy explicit per kind.
3. Align workflow with authority matrix semantics ("vet first, finalize second").
4. Remove global tier infrastructure that no longer matches the approval model.

---

## Alignment With Authority Matrix

Reference: `/Users/dean/code/tybalt_turbo/authority_matrix.md`

Authority matrix intent:

- Different activities/kinds have different authority bands.
- Some kinds are effectively high-authority-only.

This spec aligns by:

1. Keeping kind-specific approval limits in `po_approver_props`.
2. Adding per-kind second-approval trigger (`second_approval_threshold`).
3. Routing first-stage review to "lower-band" approvers and second-stage final approval to higher-band approvers.
4. Preserving escalation after a configurable timeout to prevent work from stalling.

---

## Final Decisions (Normative)

## D1. Approval trigger model

Global `po_approval_thresholds` is removed.

Approval stage requirement is determined per kind:

- `dual_required = kind.second_approval_threshold > 0 AND approval_total > kind.second_approval_threshold`
- Otherwise single approval.

## D2. Threshold location

`second_approval_threshold` is stored on `expenditure_kinds` (not `app_config`).

Rationale:

- This is kind metadata/policy, not a global feature toggle.
- `app_config` is loosely typed/fail-open and not appropriate for approval governance.

## D3. Pool model for dual-required POs

For a given PO, resolve `kind_limit_column` using existing kind/job logic.

"Otherwise eligible" means user is:

- Active.
- Holder of `po_approver` claim.
- Division-authorized for the PO.
- Has a valid value for the PO's resolved kind limit column.

Then compute:

- First-stage pool (vet): otherwise eligible and `user_kind_limit <= kind.second_approval_threshold`
- Second-stage pool (final): otherwise eligible, `user_kind_limit > kind.second_approval_threshold`, and `user_kind_limit >= approval_total`

For single-stage POs (not dual-required):

- First-stage pool is otherwise eligible and `user_kind_limit >= approval_total`.

## D4. Final approver hard safety check

For final approval authority, backend must require:

- user in second-stage pool, and
- `user_kind_limit >= approval_total`

This prevents under-authorized final approvals.

## D5. Empty pool policy

Both are hard errors (configuration/data-entry issue requiring admin correction):

- Empty first-stage pool when dual is required.
- Empty second-stage pool when dual is required.
- Missing `priority_second_approver` when dual is required.

Expected enforcement points:

1. Hook validation (create/update): for dual-required PO, block save unless:
   - first-stage pool is non-empty,
   - second-stage pool is non-empty,
   - selected `approver` is valid for first-stage pool,
   - selected `priority_second_approver` is present and valid for second-stage pool.
2. Approve endpoint transition guard: block stage transitions when required pool(s) are empty or assigned actors are invalid for required stage.

## D6. Visibility and denoising policy

For dual-required POs:

1. `Unapproved` and not yet first-approved: visible only to assigned first approver.
2. After first approval and before second:
   - For first `T` hours: visible only to `priority_second_approver`.
   - After `T` hours: visible to all second-stage eligible approvers.

For single-approval POs:

- Normal first-stage approval flow only.

## D7. Time anchor for escalation window

Use `approved` timestamp (first-approval timestamp) as the escalation anchor.

Do not use `updated` for escalation timing.

`T` is configurable via `app_config`:

- Domain key: `purchase_orders`
- Property: `second_stage_timeout_hours`
- Type: positive number
- Bounds/default: if missing, non-numeric, or non-positive, backend must use `24`

For dual-required POs, exclusive window ownership is guaranteed because `priority_second_approver` is mandatory by D5.

## D8. API behavior: bypass and same-user approvals

Backend API allows a second-stage approver to bypass first-stage ownership and complete approval, even if they are not the assigned first approver.

The same user can perform both approvals via API when they satisfy required checks.

Bypass behavior for dual-required POs where both `approved` and `second_approval` are empty:

1. In one API action, set:
   - `approver = caller`
   - `approved = now`
   - `second_approver = caller`
   - `second_approval = now`
2. Set `status = Active`.
3. Generate and assign `po_number` in the same action.

This avoids forcing the same actor to perform two separate perceived approvals.

This is intentional and is for operational flexibility; UI denoising remains the primary anti-noise mechanism.

## D9. Edit/recompute behavior

When PO fields affecting eligibility change (`kind`, `division`, `job`, `total`/`approval_total`, recurring terms):

- Recompute pools.
- Keep current `approver` and/or `priority_second_approver` if still valid.
- Raise field-level validation errors if either becomes invalid.

No silent reassignment by default.

## D10. Standard kind future split

Current decision: no split between standard-with-job and standard-without-job thresholds.

Future-compatible note:

- If needed later, split `standard` into two kinds and configure thresholds independently.

---

## Policy Flow Table

| Condition | First-stage required | Second-stage required | Queue owner(s) |
| --- | --- | --- | --- |
| `kind.second_approval_threshold = 0` | Yes | No | Assigned first approver only |
| `threshold > 0` and `approval_total <= threshold` | Yes | No | Assigned first approver only |
| `threshold > 0` and `approval_total > threshold` | Yes | Yes | Stage 1: assigned first approver; Stage 2: priority second (`T` hours), then all second-stage eligible |

---

## Stage Authorization Rules

## Stage 1 (vet)

Required:

1. PO is not first-approved (`approved` empty).
2. Caller is assigned first approver.
3. Assigned first approver is valid in first-stage pool at approval time.
4. If dual-required:
   - second-stage pool is non-empty, and
   - `priority_second_approver` is present and valid in second-stage pool.

Effect:

- Set `approved` timestamp.
- Set `approver` to caller (audit truth of who actually approved stage 1).

## Combined dual approval (bypass fast path)

Required:

1. PO is dual-required.
2. `approved` and `second_approval` are both empty.
3. Caller is valid second-stage approver and satisfies final safety check (`limit >= approval_total`).

Effect:

1. Set `approver = caller`.
2. Set `approved = now`.
3. Set `second_approver = caller`.
4. Set `second_approval = now`.
5. Set `status = Active`.
6. Generate and assign `po_number`.

## Stage 2 (final)

Required:

1. PO has `approved` set and `second_approval` empty.
2. Caller is in second-stage pool (or is valid `priority_second_approver` during exclusive window).
3. Caller kind limit `>= approval_total` (hard safety gate).

Effect:

- Set `second_approval`, `second_approver`.
- Set PO `status = Active` when final conditions are met.

---

## Data Model Changes

## Remove

1. `po_approval_thresholds` collection/table.
2. Any `lower_threshold` / `upper_threshold` computed fields and dependencies.

## Add

On `expenditure_kinds`:

1. `second_approval_threshold` (number, non-negative).
2. Migration default: `0` (safe default to avoid accidental dual-approval rollout).

Operationally, thresholds can then be configured per kind by admins.

On `app_config`:

1. Ensure domain key `purchase_orders` exists.
2. Add property `second_stage_timeout_hours` (positive number; default `24`).

---

## View/Store Naming and Scope Cleanup

Current `user_po_permission_data` name becomes misleading after threshold removal.

## Required cleanup

1. Introduce a general user-claims view for global UI/nav gating: `user_claims_summary`.
2. Introduce PO-specific profile view for PO screens: `user_po_approver_profile`.
3. Transition `global.ts` claims loading to the general view.
4. Remove threshold-derived fields from PO-specific view payloads.
5. Important dependency note: `user_po_permission_data` currently depends on `po_approval_thresholds`; both must be replaced in the same cutover to avoid broken reads.

Cutover strategy (no compatibility shim):

1. None. This change set intentionally targets correct end-state in first pass.
2. Replace old views/usages directly (`user_po_permission_data` -> `user_claims_summary` + `user_po_approver_profile`).

---

## API Contract Changes

## Existing endpoints expected to remain

1. `GET /api/purchase_orders/approvers`
2. `GET /api/purchase_orders/second_approvers`
3. `GET /api/purchase_orders/pending`
4. `POST /api/purchase_orders/{id}/approve`

## Required behavior updates

1. Create/update validation must enforce:
   - if dual-required: valid `approver` from first-stage pool and valid, non-empty `priority_second_approver` from second-stage pool,
     - exception (self-bypass setup): allow `approver = uid` when `uid` is second-stage-qualified and `priority_second_approver = uid`,
   - if single-required: valid `approver` from first-stage pool.
2. Approver endpoints must reflect new pool logic using `second_approval_threshold` from kind.
   - `GET /api/purchase_orders/approvers` must return the full first-stage pool (including requester if qualified), not an implicit self-qualified empty list.
   - For single-stage requests, first-stage pool must already satisfy `limit >= approval_total`.
   - `GET /api/purchase_orders/second_approvers` must return `second_pool_empty` when second approval is required, requester is not self-qualifying, and no second-stage candidate can final-approve the amount.
3. Pending endpoint/query must use:
   - assigned-first-only for stage 1,
   - priority-second-only for first `T` hours after `approved`,
   - all eligible second-stage users after `T`.
4. Approve endpoint must enforce stage paths:
   - stage-1 route path,
   - stage-2 route path,
   - combined dual bypass fast path.
5. Approve endpoint must enforce D3/D4/D8 and return deterministic error codes for pool/config failures.
6. Timeout config handling is fail-safe: missing/invalid/non-positive `second_stage_timeout_hours` uses `24` without throwing config errors.

---

## Error Model (Required)

Use explicit machine-readable errors:

1. `priority_second_approver_required` - dual required but `priority_second_approver` missing.
2. `first_pool_empty` - dual required but no first-stage candidates configured.
3. `second_pool_empty` - dual required but no second-stage candidates configured.
   - Also returned by second-approvers endpoint when no candidate can final-approve the evaluated amount.
4. `invalid_approver_for_stage` - selected first approver invalid after recompute/stage check.
5. `invalid_priority_second_approver_for_stage` - selected priority second invalid after recompute/stage check.
6. `insufficient_final_limit` - caller attempted final approval without `limit >= approval_total`.

These must surface as field-level errors in create/edit where applicable.
Timeout config read failures are non-fatal and must resolve to default `24`.

---

## UI Requirements

## PurchaseOrdersEditor

1. Always request and render first/second-stage candidate sets based on current PO draft.
2. For dual-required PO:
   - approver selector from first-stage pool.
   - priority second selector from second-stage pool.
   - `priority_second_approver` is required.
   - own-PO bypass UX exception: if second-approver endpoint returns `requester_qualifies`, hide both selectors and persist `approver = requester`, `priority_second_approver = requester`.
   - this is a UX/create-validation exception only; stage pool definitions and approve-path authorization remain unchanged.
3. On field changes that alter eligibility:
   - recompute pools.
   - keep selected values if still valid.
   - show field errors if invalid for either assignee field.
4. Second-approver UI visibility:
   - show second-approver selector only when second approval is required and candidates exist.
   - show second-approver error state only when second approval is required and candidates do not exist.
   - do not show a "second approver not required" informational hint for single-stage POs.

## Pending list behavior

1. First-stage pending list shows only records assigned to caller as first approver.
2. Second-stage pending:
   - first `T` hours after `approved`: only priority second assignee.
   - after `T`: all second-stage eligible claim holders.
   - `T` is read from config; missing/invalid/non-positive resolves to `24`.

---

## Notification Requirements

1. On create: notify assigned first approver.
2. On first approval of dual-required PO:
   - notify `priority_second_approver`.
3. On timeout expiry `T` (not yet second-approved):
   - notify all second-stage eligible users.

All notification selection logic must align with the same eligibility/pool source of truth used for API authorization.

---

## Implementation Plan (File-Oriented)

## Backend

1. Migrations:
   - add `expenditure_kinds.second_approval_threshold`.
   - remove `po_approval_thresholds`.
   - update dependent views/rules.
2. Utilities:
   - replace threshold fetch helpers with kind-threshold resolver.
   - update approver pool query builder.
3. Routes:
   - `purchase_orders.go`: approval authorization/transition logic updates.
   - `purchase_orders.go`: implement combined bypass fast path to set both approval stages in one action (`approver`/`approved` + `second_approver`/`second_approval`), set `status=Active`, and assign `po_number`.
   - `pending_pos.sql`: visibility routing rewrite.
4. Hooks:
   - `purchase_orders.go`: validation/recompute for approver and priority second fields.
   - `purchase_orders.go`: enforce dual-required `priority_second_approver` requirement.
   - `purchase_orders.go`: enforce first/second pool membership validation on save.
5. Notifications:
   - update candidate and escalation selection to new second-stage logic.

## Frontend

1. `poApprovers` request/response handling for new pool semantics.
2. `PurchaseOrdersEditor` selection and error rendering for recompute invalidation.
3. `global.ts` claims data source decoupling from PO-specific view naming.

---

## Historical Audit Notes

The pre-cutover, line-by-line gap checklist has been removed from this document
to keep this spec decision-complete and implementation-current.

If needed, that audit snapshot remains available in git history for this file.

---

## Testing Requirements

## Unit/Integration tests must cover

1. Dual trigger:
   - threshold `0` => single.
   - threshold `>0` and amount below/equal => single.
   - threshold `>0` and amount above => dual.
2. Pool membership:
   - dual-required: first pool (`limit <= threshold`) and second pool (`limit > threshold` and `limit >= approval_total`) correctness.
   - single-stage: first pool requires `limit >= approval_total`.
3. Empty pool hard failures.
4. Dual-required save fails when `priority_second_approver` is missing.
5. Dual-required save fails when `approver` is not in first-stage pool.
6. Dual-required save fails when `priority_second_approver` is not in second-stage pool.
   - exception case: dual-required save allows `approver = uid` when `uid` is second-stage-qualified and `priority_second_approver = uid`.
7. Final approval safety check (`limit >= approval_total`).
8. Visibility timing based on `approved` timestamp.
9. Second-stage bypass via API.
10. Bypass fast path with no prior approvals sets both approver fields, both timestamps, activates status, and assigns PO number in one call.
11. Same-user both approvals allowed via API.
12. Timeout config missing/invalid/non-positive falls back to `24`.
13. Two-view data path:
    - global claims from `user_claims_summary`,
    - PO-specific approver data from `user_po_approver_profile`.
14. Edit recompute retain/invalid cases.
15. Notification target correctness at create, first approval, and escalation.
16. Regression: no remaining dependencies/references to `po_approval_thresholds`.
17. UI bypass mode: dual-required + own PO + `requester_qualifies` hides both selectors and saves both approver fields as requester.

---

## Management Caveats / Risks

1. **Intentional bypass capability**:
   - High-authority users can fully approve via API, bypassing first-stage assignment.
   - This is deliberate; audit logs must clearly show actor and sequence.
2. **Operational dependency on data quality**:
   - Empty pool is now a hard block by policy.
   - Admin ownership of `po_approver_props` data completeness becomes critical.
3. **Initial noise profile**:
   - Second-stage escalation after timeout `T` may increase noise, intentionally to avoid dropped requests.
4. **Behavior change for users**:
   - Pending queue semantics become stricter for first-stage ownership and clearer for second-stage exclusivity.
5. **Migration complexity**:
   - Removing threshold-derived view fields and legacy dependencies requires coordinated backend/UI rollout.
6. **Denoising profile is intentionally different**:
   - Denoising remains, but it is no longer tier-ceiling-based; it is assignment + timeout based.
7. **Config governance**:
   - Timeout comes from `app_config`; invalid or missing config must not fail-open unpredictably. Backend must enforce numeric bounds and a deterministic default.

---

## Rollout Strategy

1. Apply schema + code changes as one coordinated cutover (no compatibility shim).
2. Update APIs and UI to new pool logic.
3. Backfill per-kind thresholds in `expenditure_kinds`.
4. Add/validate `app_config.purchase_orders.second_stage_timeout_hours`.
5. Validate with fixture/scenario tests.
6. Remove `po_approval_thresholds` and legacy threshold references.
7. Replace `user_po_permission_data` with `user_claims_summary` + `user_po_approver_profile` and remove old view in same cutover.

---

## Public Interfaces and Types Changed

1. `expenditure_kinds.second_approval_threshold` (new required governance field).
2. `app_config.purchase_orders.second_stage_timeout_hours` (new runtime policy field with default/fallback semantics).
3. `user_claims_summary` (new canonical global claims view).
4. `user_po_approver_profile` (new canonical PO approver-profile view).
5. Removal of threshold-derived view fields and `po_approval_thresholds` dependency chain.

---

## Locked Assumptions and Defaults

1. No compatibility shim; direct single end-state cutover.
2. Timeout default is `24` when config is missing/invalid/non-positive.
3. Dual-required POs must always have a valid `priority_second_approver`.
4. Bypass is intentional and performs full two-stage approval in one action.
5. Denoising remains, but intentionally differs from legacy tier-ceiling behavior (assignment + timeout ownership).

---

## Document Sufficiency Criteria

1. No optional or ambiguous wording remains in policy-critical sections.
2. Every stage transition defines explicit preconditions, side effects, and error outcomes.
3. Every new/removed interface is named and mapped to implementation and testing.
4. Gap checklist rows align to final policy decisions without contradiction.
5. Implementers can execute from this document without further product decisions.

---

## Out of Scope

1. Splitting standard into two kinds now.
2. Introducing organizational hierarchy tables beyond current claim/limit data.
3. Changing non-PO approval systems.

---

## Summary

This spec replaces global threshold tiers with kind-level dual-approval thresholds and explicit two-stage routing:

1. Policy trigger is per kind.
2. Routing denoises by ownership and timed escalation.
3. Authorization remains strict at final approval (`limit >= approval_total`).
4. Behavior aligns with authority matrix intent and removes global-threshold coupling that currently creates visibility inconsistencies.
