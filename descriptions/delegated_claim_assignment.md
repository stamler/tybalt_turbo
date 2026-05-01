# Delegated claim assignment

## STATUS: UNIMPLEMENTED, NOT REVIEWED

### Goal

Allow selected non-admin users to assign specific claims to other users without
granting them full `admin` power.

This is meant to answer the question: "How can someone assign some claims, but
obviously not `admin`?" The core idea is to keep `admin` as the unrestricted
authority while introducing a narrow, auditable delegation model for ordinary
claim assignment.

### Current state

- Claims are stored in `claims`.
- User claim grants are stored in `user_claims`.
- Claim editing is currently restricted to users who hold the `admin` claim
  through custom routes such as `/api/admin_profiles/save_with_claims` and
  `/api/claims/{id}/bulk_assign`.
- The UI can show claim holders and can now bulk assign users who do not already
  hold a selected claim.

This is safe, but coarse. Anyone who can assign claims can assign every claim,
including high-risk claims.

### Proposed model

Add a new explicit delegation table that describes which claim grants a user is
allowed to assign.

Recommended collection:

```text
claim_assignment_permissions
```

Fields:

- `assigner_claim`: relation to `claims`
- `assignable_claim`: relation to `claims`
- `created`
- `updated`

Example rows:

```text
assigner_claim = hr
assignable_claim = time

assigner_claim = hr
assignable_claim = time_off_manager

assigner_claim = it
assignable_claim = time
```

This makes delegation role-based. A user can assign claim `X` if they hold a
claim that is permitted to assign `X`.

### Why claim-to-claim instead of user-to-claim

Prefer claim-to-claim delegation first.

It has a smaller administrative surface:

- Administrators grant a role-like claim once.
- The allowed assignment scope follows that role.
- Removing the assigner claim automatically removes assignment authority.
- The rules are easy to inspect as a small matrix.

A user-to-claim model could be added later if the business needs one-off
exceptions, but it should not be the first version.

### Hard exclusions

Some claims should never be assignable through delegation.

At minimum:

- `admin`

Likely also consider excluding claims that can bypass major financial or
identity controls unless management explicitly approves them:

- `payables_admin`
- `accounting`
- `it`
- `po_approver` if its paired `po_approver_props` limits/divisions are not
  handled in the same workflow

These exclusions should be enforced on the server, not only hidden in the UI.

### Backend authorization

Add a helper such as:

```go
CanAssignClaim(app core.App, assignerUID string, targetClaimID string) (bool, error)
```

Behavior:

1. If the assigner holds `admin`, return true.
2. Load the target claim.
3. Reject hard-excluded target claim names such as `admin`.
4. Check whether the assigner holds any claim with a matching
   `claim_assignment_permissions.assigner_claim`.
5. Return true only if one of those permissions points at `targetClaimID`.

All claim-writing routes should use this helper:

- Single-user claim save in admin profile editing.
- Bulk claim assignment from `claims/{id}/bulk_assign`.
- Any future direct claim assignment route.

The helper must check the target claim being added. It should not grant blanket
permission to edit a user's full claim set.

### Important distinction: add-only vs full sync

Bulk assignment is naturally add-only: it adds one selected claim to selected
users. Delegated users can safely use this if they are allowed to assign that
one claim.

Admin profile editing is more dangerous because it currently syncs the whole
claim set for a user. A delegated assigner should not be able to remove claims
they do not control or add claims they do not control.

Recommended rule:

- `admin` can continue full claim sync.
- Delegated assigners get add/remove authority only for claims they are allowed
  to assign.
- If a request changes any claim outside the assigner's allowed set, reject the
  whole request.

This prevents a delegated user from accidentally or intentionally stripping
unrelated permissions.

### UI behavior

The claims list/details pages should stay available only to users who can manage
at least one claim.

For a delegated user:

- Show only claims they are allowed to assign.
- On `claims/{id}/details`, show `Bulk Assign` only when they can assign that
  claim.
- On the bulk assignment page, keep the same list/search/checkbox workflow.
- If they navigate directly to an unauthorized claim assignment page, show a
  forbidden state from the backend response.

For a user who holds the `admin` claim:

- Keep the current behavior: all claims visible, all assignable except any
  hard-excluded claim if we choose to block even admin from UI convenience flows.
  Backend admin authority should remain the source of truth.

### Auditability

Claim assignment changes should be auditable.

Minimum audit fields for future implementation:

- actor uid
- target uid
- claim id
- action: `add` or `remove`
- source: `admin_profile_save`, `claim_bulk_assign`, etc.
- timestamp

This can be a separate `claim_assignment_audit` collection or structured app
logs, but a collection is easier to inspect from the application.

### Tests

Backend tests should cover:

- Admin can assign any non-excluded claim.
- Delegated claim holder can assign an allowed claim.
- Delegated claim holder cannot assign `admin`.
- Delegated claim holder cannot assign a claim absent from
  `claim_assignment_permissions`.
- Bulk assignment skips users who already hold the claim.
- Full claim sync rejects changes outside the delegated assigner's allowed set.
- Removing a delegated assigner claim removes assignment authority.

Use fixture CSV rows for the new collections and append-only fixture data.

### Rollout

1. Add the `claim_assignment_permissions` collection with schema migration.
2. Seed initial permissions conservatively.
3. Add `CanAssignClaim`.
4. Update claim-writing routes to use either `admin` or `CanAssignClaim`.
5. Narrow the claims UI based on assignable claims.
6. Add audit records after the authorization model is stable.

### Open decision

Management must decide which claims are safe to delegate.

The technical model can support any claim-to-claim mapping, but the initial
matrix is a policy decision. Start small: delegate `time` only, then expand once
the workflow and audit trail are trusted.
