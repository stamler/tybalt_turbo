# Test DB Fixture Changes (2026-02-17)

This document lists every direct change made to `app/test_pb_data/data.db` while removing setup-time DB mutation from purchase-order tests.

## 1) Normalized legacy PO kind values
- Table: `purchase_orders`
- Change: set `kind='l3vtlbqg529m52j'` (`standard`) for all rows where `kind=''`.
- Reason: remove per-test kind patching and keep fixture rows valid under strict kind validation.

## 2) Enabled computer-kind second-stage test semantics in fixture
- Table: `expenditure_kinds`
- Row: `id='7jgifny7fljd9hi'` (`computer`)
- Change: `second_approval_threshold` from `0` to `500`.
- Reason: allow a stable fixture case for kind-specific limit-column visibility (`computer_max` vs `max_amount`) without test-time updates.

## 3) Added inactive second-approver fixture identity
- Table: `users`
  - Inserted: `id='inactpoappr0001'`, `email='inactive2@poapprover.com'`, `username='inactive2tier2'`, unique `tokenKey='inactive2tier2_token_key_001'`.
- Table: `profiles`
  - Inserted: `id='profinactpo0001'`, `uid='inactpoappr0001'`, `given_name='Inactive'`, `surname='TierTwoB'`.
- Table: `admin_profiles`
  - Inserted: `id='adpinactpo00001'`, `uid='inactpoappr0001'`, `active=0`.
- Table: `user_claims`
  - Inserted: `id='ucinactpo000001'`, `uid='inactpoappr0001'`, `cid='5vh881k048bboim'` (`po_approver`).
- Table: `po_approver_props`
  - Inserted: `id='ppinactpo000001'`, `user_claim='ucinactpo000001'`, limits `{max_amount:2500, project_max:0, sponsorship_max:0, staff_and_social_max:0, media_and_event_max:0, computer_max:0}`, `divisions='[]'`.
- Reason: replace test-time `admin_profiles.active=0` mutation with a fixture-backed inactive approver user.

## 4) Added duplicate user_claim fixture for uniqueness test
- Table: `user_claims`
  - Inserted: `id='dupucclaim00001'`, `uid='inactpoappr0001'`, `cid='5vh881k048bboim'` (`po_approver`).
- Table: `po_approver_props`
  - Inserted: `id='dupprops0000001'`, `user_claim='dupucclaim00001'`, limits `{max_amount:1000, project_max:0, sponsorship_max:0, staff_and_social_max:0, media_and_event_max:0, computer_max:0}`, `divisions='[]'`.
- Reason: remove test-time insert/delete setup in duplicate `po_approver_props.user_claim` coverage.

## 5) Added fixture POs for route edge-cases and duplicate-hash tests
- Table: `purchase_orders`
- Inserted rows:
  - `id='po1stgready0001'`
    - single-stage already-first-approved but `status='Unapproved'`, `approved='2025-01-30 12:00:00.000Z'`, `approver='etysnrlup2f6bak'`, `total=approval_total=329.01`, `kind='l3vtlbqg529m52j'`.
  - `id='pobadassign0001'`
    - first-stage invalid-assigned fixture (`approver=''`), `total=approval_total=2000`, `priority_second_approver='6bq4j0eb26631dy'`, `kind='l3vtlbqg529m52j'`.
  - `id='posecondempty01'`
    - unauthorized-before-`second_pool_empty` fixture, `total=approval_total=1000001`, `approver='etysnrlup2f6bak'`, `priority_second_approver='6bq4j0eb26631dy'`, `kind='l3vtlbqg529m52j'`.
  - `id='poinvalkind0001'`
    - invalid kind fixture, `kind='kind_missing_123'`.
  - `id='pocompkindvis01'`
    - first-approved computer-kind visibility fixture, `approved='2025-01-29 14:22:29.563Z'`, `total=approval_total=1022.69`, `kind='7jgifny7fljd9hi'`.
  - `id='pohashdupcrt001'`
    - `attachment_hash='9ac8553cb9ef35aec3c169790f36d29261e8bb7a34fe1c2307a96f1784211ab5'` (create duplicate-hash test).
  - `id='pohashdupupd001'`
    - `attachment_hash='75edf6c0a35273de96d1e8d1a91558150aed1f3e19f3aa86262c221e4964d5e2'` (update duplicate-hash test), `uid='f2j5a8vk006baub'`.
  - `id='pofirstempty001'`
    - first-stage pool-empty fixture, `division='kxedrbp7vj2mtjd'` (Board), `total=approval_total=2000`, `approver='etysnrlup2f6bak'`, `priority_second_approver='6bq4j0eb26631dy'`, `kind='l3vtlbqg529m52j'`.
  - `id='postg2empty0001'`
    - stage-2 regression fixture (already first-approved), `division='kxedrbp7vj2mtjd'`, `approved='2025-01-29 14:22:29.563Z'`, `total=approval_total=1022.69`, `approver='wegviunlyr2jjjv'`, `priority_second_approver='6bq4j0eb26631dy'`, `kind='l3vtlbqg529m52j'`.

## 6) Updated approver division fixture scope (to support deterministic first-pool-empty case)
- Table: `po_approver_props`
- Rows:
  - `id='1ea212qef65397o'` (fakemanager@fakesite.xyz)
  - `id='5do7ivq31u1e425'` (orphan@poapprover.com)
- Change: `divisions` set to `json_group_array(divisions.id)` for all divisions except `kxedrbp7vj2mtjd`.
- Reason: keep near-global fixture behavior while reserving one division with no low-limit first-stage approvers.
