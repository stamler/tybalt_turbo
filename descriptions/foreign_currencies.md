# Implementation Plan: Multi-Currency Support for POs and Expenses (Issue #121)

## Overview

Add the ability to tag Purchase Orders and Expenses with a currency (defaulting to CAD),
display indicative exchange rates throughout the workflow, and track settled CAD amounts
for non-CAD expenses. Exchange rates are sourced daily from the Bank of Canada Valet API.

Non-home-currency OnAccount and CorporateCreditCard expenses require explicit settlement
by a payables admin before they can be committed, managed through a dedicated Currency
Settlement queue.

Approval authority is evaluated in home currency (CAD) via `approval_total_home`, but
PO spending caps on expenses are enforced in the PO's stated currency. This ensures
that forex fluctuations after approval do not invalidate a user's ability to spend
against an approved PO.

---

## 1. New Collection: `currencies`

| Field       | Type   | Notes                                                      |
|-------------|--------|------------------------------------------------------------|
| `code`      | string | ISO 4217 code (e.g. "CAD", "USD"). Unique.                |
| `symbol`    | string | Display symbol (e.g. "$", "US$", "€")                     |
| `icon`      | file   | SVG icon for the currency                                  |
| `rate`      | number | 1 unit of this currency = X CAD. CAD row is always 1.     |
| `rate_date` | string | Date the rate was last fetched (YYYY-MM-DD)                |
| `ui_sort`   | number | Sort order for currency selectors (lower = first).         |

- Seeded with CAD (`rate=1`, `ui_sort=0`) and USD (`ui_sort=1`), and any other currencies desired.
- Open-ended: new currencies can be added at any time.
- The `rate` and `rate_date` fields are updated in place by the daily cron job.
- `ui_sort` controls the display order in DSCurrencyInput selectors.

---

## 2. Daily Exchange Rate Cron Job

- **Source**: Bank of Canada Valet API (free, no API key required).
  - Example: `https://www.bankofcanada.ca/valet/observations/FXUSDCAD/json?recent=1`
  - Series naming: `FX{CODE}CAD` (e.g. `FXUSDCAD`, `FXEURCAD`)
- **Schedule**: Daily, after 16:30 ET on business days.
- **Behaviour**: For each non-CAD row in the `currencies` table, fetch the latest
  rate from the Bank of Canada and update `rate` and `rate_date` on that row.
- **Weekends/holidays**: API returns no new data; existing rate remains unchanged.
- **Implementation**: Add to existing `cron/AddCronJobs`.

---

## 3. Purchase Order Changes

### New fields on `purchase_orders`

| Field                 | Type                   | Notes                                                             |
|-----------------------|------------------------|-------------------------------------------------------------------|
| `currency`            | relation -> currencies | Required. Default CAD.                                            |
| `approval_total_home` | number                 | CAD equivalent of `approval_total`, computed by hook using cached `currencies.rate`. |

### Hook logic

- **Clean/Validate**:
  - `currency` is required and must reference a valid `currencies` record.
  - Compute `approval_total_home = approval_total * currency.rate`.
  - For CAD, `approval_total_home` = `approval_total` (rate is 1).
- **Child POs**: Must inherit parent's `currency` (validated in existing child PO rules).
- **Approval threshold comparison**: Use `approval_total_home` (not `approval_total`)
  against the CAD-denominated limits in `po_approver_props`.
- **Approver pool determination**: Use `approval_total_home` when resolving eligible
  first-stage and second-stage approvers.

### Currency vs. amount enforcement separation

- **Approval authority** is evaluated in CAD (`approval_total_home`) — this is the only
  place currency conversion matters for POs.
- **Expense amount caps** are enforced in the PO's stated currency — `expense.total` is
  compared against `po.total` (+ excess) directly, both in the same currency. This means
  forex fluctuations after PO approval do not require users to obtain a new PO.

### UI changes

- DSCurrencyInput on PO form (default CAD, currency selectable).
- When non-CAD selected, display indicative CAD equivalent next to the total
  (e.g. "≈ CA$13,845 at 1 USD = 1.3845 CAD, Apr 1").
- PO detail/print views show currency code/symbol alongside amounts.
- Approval screens show both the PO currency amount and the CAD equivalent.

---

## 4. Expense Changes

### New fields on `expenses`

| Field           | Type                   | Notes                                               |
|-----------------|------------------------|------------------------------------------------------|
| `currency`      | relation -> currencies | Required. Default CAD.                               |
| `settled_total` | number                 | Amount in CAD. Required for commit.                  |
| `settler`       | relation -> users      | Payables admin who entered the settlement.           |
| `settled`       | datetime               | Timestamp when settlement was set.                   |

### Currency assignment rules

- **Linked to PO**: `currency` is inherited from the PO and immutable.
- **No PO, payment type is `OnAccount`, `Expense`, or `CorporateCreditCard`**:
  User selects currency (default CAD).
- **Payment type is `Allowance`, `FuelCard`, `Mileage`, or `PersonalReimbursement`**:
  Always CAD. No currency selector shown. Hook enforces CAD.

### `settled_total` rules by payment type

| Scenario                                  | Who enters it           | When required     | Mutability                                  |
|-------------------------------------------|-------------------------|-------------------|---------------------------------------------|
| Currency = CAD (any type)                 | Hook (auto-set = total) | Always set auto   | Updates with `total`                        |
| Currency ≠ CAD, `Expense`                 | User                    | Before submit     | Immutable once submitted                    |
| Currency ≠ CAD, `OnAccount`               | Payables admin          | Before commit     | Cleared and re-entered via settlement queue |
| Currency ≠ CAD, `CorporateCreditCard`     | Payables admin          | Before commit     | Cleared and re-entered via settlement queue |

### Settlement fields (`settler`, `settled`) rules

- Only populated for non-home-currency `OnAccount` and `CorporateCreditCard` expenses.
- Set together with `settled_total` by the payables admin via the settlement queue.
- For `Expense` payment type (user-entered `settled_total`), `settler` and `settled`
  remain blank — the user enters `settled_total` directly on the expense form.
- For CAD expenses, all three fields (`settled_total`, `settler`, `settled`) are
  managed by the hook: `settled_total = total`, `settler` and `settled` remain blank.

### Hook logic

- **Clean**: If currency = CAD, set `settled_total = total`.
- **Validate on submit** (`Expense` type, currency ≠ CAD): `settled_total` must be > 0.
- **Validate on submit** (`Expense` type, currency ≠ CAD): `settled_total` is immutable
  (reject changes to `settled_total` if already submitted).
- **Validate on commit** (all types): `settled_total` must be > 0.
- **Commit gate**: Non-home-currency `OnAccount`/`CorporateCreditCard` expenses must
  have non-blank `settled` timestamp to appear in the commit queue.
- **No-PO expense limit**: The $100 limit check compares against `settled_total`
  (CAD equivalent) so the limit is consistently applied in CAD.
- **PO amount caps** (One-Time, Recurring, Cumulative overflow): Compare `expense.total`
  against `po.total` in the PO's stated currency. Both are in the same currency — no
  conversion needed. Forex changes after PO approval do not affect this check.

### Rejection behaviour

- When a non-home-currency `OnAccount` or `CorporateCreditCard` expense is rejected
  (via the reject route or committer-reject path), the settlement is **automatically
  cleared**: `settled_total`, `settler`, and `settled` are all set to their zero values.
- For `Expense` payment type (user-entered `settled_total`), rejection does **not**
  clear `settled_total` — the user manages it themselves upon revision and resubmission.
- For CAD expenses, rejection has no effect on settlement fields — the hook continues
  to set `settled_total = total`.

### UI changes

- DSCurrencyInput on expense form:
  - Linked to PO: currency locked (inherited from PO), amount editable.
  - No PO, eligible type: currency selectable, amount editable.
  - Ineligible type (Allowance, etc.): currency locked to CAD, amount editable.
- `settled_total` input shown for `Expense` payment type when currency ≠ CAD
  (plain number input with CAD symbol — always settling into home currency).
- Indicative CAD equivalent shown alongside foreign currency total.
- Expense lists/detail views show currency alongside amounts.

---

## 5. DSCurrencyInput Component

A reusable Svelte 5 component for currency-aware amount entry.

### Layout

```
┌──────────────────────────────────────┐
│ [🇺🇸 USD ▾]   10,000.00          $ │
└──────────────────────────────────────┘
  ↑ currency       ↑ amount       ↑ symbol
  selector         input          (from currency)
```

- **Left side (inside field)**: Currency selector showing icon + code. Clicking opens
  a dropdown of available currencies, sorted by `ui_sort`.
- **Right side**: Currency symbol displayed as static text inside the input field.
- **Center**: Number input for the amount.

### Props

| Prop                    | Type                | Default     | Notes                                        |
|-------------------------|---------------------|-------------|----------------------------------------------|
| `value`                 | number              | —           | Bindable. The amount entered.                |
| `currency`              | string              | —           | Bindable. Currency record ID.                |
| `currencies`            | CurrencyRecord[]    | —           | Available currencies (pre-fetched, sorted by `ui_sort`). |
| `disableCurrencySelect` | boolean             | `false`     | Locks the currency selector (e.g. expense linked to PO). |
| `disableInput`          | boolean             | `false`     | Locks the amount input (display-only mode).  |
| `placeholder`           | string              | `"0.00"`    | Placeholder text for the amount input.       |

### Derived state

- `symbol`: Resolved from the selected currency record's `symbol` field.
- `icon`: Resolved from the selected currency record's `icon` field (SVG).
- `code`: Resolved from the selected currency record's `code` field.

### Usage examples

```svelte
<!-- PO form: full interactive -->
<DSCurrencyInput bind:value={total} bind:currency={currencyId}
  currencies={allCurrencies} />

<!-- Expense linked to PO: locked currency, editable amount -->
<DSCurrencyInput bind:value={total} bind:currency={currencyId}
  currencies={allCurrencies} disableCurrencySelect={true} />

<!-- Read-only display -->
<DSCurrencyInput value={total} currency={currencyId}
  currencies={allCurrencies} disableCurrencySelect={true}
  disableInput={true} />
```

---

## 6. Currency Settlement Queue

A new page under Expenses, accessible to users with `payables_admin` claim.
Manages settlement of non-home-currency `OnAccount` and `CorporateCreditCard` expenses.

### Entry criteria

An expense appears in the settlement queue when ALL of the following are true:
- Payment type is `OnAccount` or `CorporateCreditCard`
- Currency ≠ CAD (non-home currency)
- Approved (`approved` is non-blank, `rejected` is blank)
- Uncommitted (`committed` is blank)

### Main Tab: Unsettled

Lists expenses matching entry criteria where `settled` is blank.

**Columns displayed:**
- Date
- Creator name
- Vendor
- Description
- PO number (if linked)
- Currency code
- Total (in foreign currency)
- Indicative CAD equivalent (from cached rate on `currencies` row)
- Age (days since approval)

**Per-row action:** CAD amount input + **Save** button.

**Save action** (API route, runs in a database transaction):
1. Re-read the expense record inside the transaction.
2. Verify the expense is still approved, uncommitted, and unsettled.
3. If state has changed (committed, rejected, or already settled), abort with error.
4. Set `settled_total` to the entered amount, `settler` to the authenticated user,
   `settled` to the current timestamp.
5. Commit the transaction.

**Sort order:** Descending by date.

### Second Tab: Settled (Revision)

Lists expenses matching entry criteria where `settled` is non-blank.
This tab is **read-only** — no in-place editing of settled amounts.

**Columns displayed:** Same as unsettled tab, plus:
- Settled total (CAD)
- Settler name
- Settled timestamp

**Per-row action:** **Clear Settlement** button only.

**Clear Settlement** (API route, runs in a database transaction):
1. Re-read the expense record inside the transaction.
2. Verify the expense is still uncommitted.
3. If already committed, abort with error.
4. Clear `settled_total`, `settler`, and `settled`.
5. Commit the transaction.

**After clearing:** Stay on the settled tab and refresh the list. The cleared item
disappears from this tab and reappears on the unsettled tab. This allows the admin
to clear multiple items in rapid succession without navigating away.

**Revision workflow:** To revise a settled amount, the admin clears it (settled tab),
then re-enters the correct amount (unsettled tab). This two-step approach is
preferred because clearing is faster and wins the race against a concurrent commit —
the item is immediately ineligible for commit once cleared.

### Transaction safety

All settlement operations (set, clear) run in database transactions to prevent
races with concurrent commit, reject, or settlement actions:

| Race scenario          | Protection                                                       |
|------------------------|------------------------------------------------------------------|
| Settle vs. commit      | Transaction checks `committed` is blank before settling.         |
| Settle vs. reject      | Transaction checks `rejected` is blank before settling.          |
| Clear vs. commit       | Transaction checks `committed` is blank before clearing.         |
| Concurrent settlement  | Transaction checks `settled` is blank before settling (prevents double-settle). |

---

## 7. API Changes

### New routes

| Route                                      | Method | Auth             | Purpose                              |
|--------------------------------------------|--------|------------------|--------------------------------------|
| `POST /api/expenses/:id/settle`            | POST   | `payables_admin`  | Set settlement amount                |
| `POST /api/expenses/:id/clear_settlement`  | POST   | `payables_admin`  | Clear settlement (back to unsettled) |
| `GET /api/expenses/unsettled`              | GET    | `payables_admin`  | List unsettled queue                 |
| `GET /api/expenses/settled`                | GET    | `payables_admin`  | List settled (uncommitted) queue     |

### Existing route changes

- **Commit route**: Gate non-home-currency `OnAccount`/`CorporateCreditCard` expenses
  on `settled` being non-blank. Exclude unsettled expenses from the commit queue query.
- **Reject route**: Auto-clear settlement (`settled_total`, `settler`, `settled`) for
  non-home-currency `OnAccount`/`CorporateCreditCard` expenses upon rejection.

---

## 8. Existing Logic Impact

| Area                          | Impact                                                                 |
|-------------------------------|------------------------------------------------------------------------|
| Approval policy resolution    | Use `approval_total_home` instead of `approval_total` for threshold checks and approver eligibility. |
| Cumulative PO overflow check  | No change — sums are in PO currency, expense inherits PO currency.     |
| One-Time/Recurring PO caps    | No change — comparison in PO currency.                                 |
| No-PO expense limit ($100)    | Compare `settled_total` (CAD) against the limit.                       |
| Commit queue                  | Exclude unsettled non-home-currency OnAccount/CorporateCreditCard.     |
| Reject route                  | Auto-clear settlement for non-home-currency OnAccount/CorporateCreditCard. |
| PO cancel (block if expenses) | No change.                                                             |
| PO close                      | No change.                                                             |
| Writeback to Firebase         | Include `currency` code if legacy system needs it (TBD).               |
| Reporting / aggregation       | Sum `settled_total` for consistent CAD reporting.                      |

---

## 9. Migration Plan

1. Create `currencies` collection with fields: `code`, `symbol`, `icon`, `rate`,
   `rate_date`, `ui_sort`.
2. Seed CAD (`rate=1`, `ui_sort=0`) and USD (`ui_sort=1`) rows.
3. Add `currency` (default CAD) and `approval_total_home` fields to `purchase_orders`.
4. Add `currency` (default CAD), `settled_total`, `settler`, and `settled` fields
   to `expenses`.
5. Backfill: set `currency` = CAD for all existing POs and expenses.
6. Backfill: set `settled_total` = `total` for all existing expenses.
7. Backfill: set `approval_total_home` = `approval_total` for all existing POs.

---

## 10. Implementation Order

1. **Migration**: Create `currencies` collection, seed data, add new fields to
   POs and expenses, backfill existing records.
2. **Cron job**: Bank of Canada rate fetcher in `cron/AddCronJobs`.
3. **PO hooks**: Currency validation, `approval_total_home` computation, child PO
   inheritance, approval policy updates to use `approval_total_home`.
4. **Expense hooks**: Currency assignment, `settled_total` auto-set for CAD,
   validation for submit/commit gates, PO amount caps in PO currency.
5. **Settlement API routes**: `settle`, `clear_settlement`, `unsettled` list,
   `settled` list — all with transaction safety.
6. **Reject route update**: Auto-clear settlement on rejection for non-home-currency
   OnAccount/CorporateCreditCard.
7. **Commit route update**: Gate on settlement for non-home-currency
   OnAccount/CorporateCreditCard.
8. **DSCurrencyInput component**: Reusable currency-aware amount input with
   icon + code selector, symbol display, and lock props.
9. **UI — PO forms**: Integrate DSCurrencyInput, CAD equivalent display,
   print view updates.
10. **UI — Expense forms**: Integrate DSCurrencyInput (locked currency when PO linked),
    `settled_total` input for `Expense` payment type.
11. **UI — Currency Settlement queue**: Two-tab interface with settle and clear actions.
12. **UI — Approval screens**: Dual-amount display (foreign + CAD equivalent).
13. **Tests**: Unit tests for hooks, approval policy with currency conversion,
    settlement transaction safety, rejection auto-clear, settled_total validation,
    cron job, commit gate, PO amount caps in PO currency.
