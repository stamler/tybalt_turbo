# app_config Reference

The `app_config` collection is a key-value store for runtime configuration. Each record has a `key` (the domain), a JSON `value` object, and an optional `description`. Authenticated users can read configs; updates are unrestricted.

Configuration is read via helpers in `app/utilities/config.go`. Missing or invalid values fall back to defaults.

---

## Domain: `jobs`

Controls job-related editing operations.

| Property             | Type | Default | Description                                                                                                |
|----------------------|------|---------|------------------------------------------------------------------------------------------------------------|
| `create_edit_absorb` | bool | `true`  | Enables job creation, updating, and client/contact absorb. When `false`, these operations return HTTP 403. |

**Fail mode:** open (defaults to enabled)

---

## Domain: `expenses`

Controls expense, PO, and vendor editing, plus expense validation thresholds.

| Property                    | Type   | Default   | Description                                                                                                                            |
|-----------------------------|--------|-----------|----------------------------------------------------------------------------------------------------------------------------------------|
| `create_edit_absorb`        | bool   | `true`    | Enables expense, purchase order, and vendor creation/updating/absorb. When `false`, these operations return HTTP 403.                  |
| `no_po_expense_limit`       | number | `100.0`   | Dollar threshold above which a non-exempt expense requires a PO. Set to `0` to require a PO for all non-exempt expenses. Must be >= 0. |
| `po_expense_allowed_excess` | object | see below | Controls how much total expenses on a PO can exceed the PO total.                                                                      |

### `po_expense_allowed_excess` sub-object

| Property  | Type   | Default       | Description                                                                                                                                            |
|-----------|--------|---------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| `percent` | number | `5`           | Percentage overage allowed (0-100). Stored as human-readable percent, normalized internally to a fraction.                                             |
| `value`   | number | `100.0`       | Absolute dollar overage allowed. Must be >= 0.                                                                                                         |
| `mode`    | string | `"lesser_of"` | How to combine percent and value. `"lesser_of"` picks the smaller excess (more restrictive). `"greater_of"` picks the larger excess (more permissive). |

**Mode examples** for a $2,000 PO with percent=5, value=100:

- 5% of $2,000 = $100, absolute = $100 — both equal, limit is $2,100 either way
- For a $5,000 PO: 5% = $250, absolute = $100
  - `"lesser_of"` → uses $100 → limit is $5,100
  - `"greater_of"` → uses $250 → limit is $5,250

**Fail mode:** open (editing defaults to enabled)

---

## Domain: `purchase_orders`

Controls purchase order workflow behavior.

| Property                     | Type   | Default | Description                                                                          |
|------------------------------|--------|---------|--------------------------------------------------------------------------------------|
| `second_stage_timeout_hours` | number | `24.0`  | Hours a PO waits in "pending second approver" status before timing out. Must be > 0. |

---

## Domain: `notifications`

Toggles individual notification templates on/off. Each property key is a template code. All default to `false` (fail-closed — notifications must be explicitly enabled).

| Template Code                          | Description                                 |
|----------------------------------------|---------------------------------------------|
| `po_approval_required`                 | PO requires first-approver action           |
| `po_second_approval_required`          | PO requires second-approver action          |
| `po_priority_second_approval_required` | PO requires priority second-approver action |
| `po_active`                            | PO has been approved and is now active      |
| `po_rejected`                          | PO has been rejected                        |
| `expense_rejected`                     | Expense has been rejected                   |
| `expense_approval_reminder`            | Reminder to approve pending expenses        |
| `timesheet_submission_reminder`        | Reminder to submit timesheet                |
| `timesheet_approval_reminder`          | Reminder to approve pending timesheets      |
| `timesheet_rejected`                   | Timesheet has been rejected                 |
| `timesheet_shared`                     | Timesheet has been shared with a viewer     |

**Fail mode:** closed (defaults to disabled)

---

## Example values

```json
// key: "jobs"
{ "create_edit_absorb": true }

// key: "expenses"
{
  "create_edit_absorb": true,
  "no_po_expense_limit": 100.0,
  "po_expense_allowed_excess": {
    "percent": 5,
    "value": 100.0,
    "mode": "lesser_of"
  }
}

// key: "purchase_orders"
{ "second_stage_timeout_hours": 24 }

// key: "notifications"
{
  "po_approval_required": true,
  "po_active": true,
  "expense_rejected": true,
  "timesheet_submission_reminder": false
}
```
