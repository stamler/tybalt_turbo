# PO / Expenses System Summary

## Interactions between `purchase_orders` and `expenses` records

### 1. Anyone can create a `purchase_orders` record

### 2. `purchase_orders` records must be approved by one or two managers depending on their `approval_total`

1. Approval is single-tier when `approval_total` is less than or equal to the first threshold returned by `po_approval_thresholds`; otherwise, a second approval is required. (app/routes/purchase_orders.go: `createApprovePurchaseOrderHandler` — sets `recordRequiresSecondApproval` based on `approval_total > thresholds[0]` after calling `utilities.GetPOApprovalThresholds`; app/utilities/po_approvers.go: `GetPOApprovalThresholds`.)

2. The `approval_total` is set to total for `Normal`/`Cumulative` POs and computed for `Recurring` POs using the full period value (app/hooks/purchase_orders.go: `cleanPurchaseOrder`, which calls `utilities.CalculateRecurringPurchaseOrderTotalValue`; app/utilities/po_approvers.go: `GetPOApprovalThresholds`).

3. During approval, first approval sets `approved` and `approver`; if second approval is required, `second_approval` and `second_approver` must also be set.

4. Status becomes `Active` and a `po_number` is generated only after either no second approval is needed or second approval is complete (app/routes/purchase_orders.go: `createApprovePurchaseOrderHandler` and `GeneratePONumber`).

5. Eligibility for approvers and second approvers is determined by division and amount via app/utilities/po_approvers.go:`GetPOApprovers`; enforcement happens in app/routes/purchase_orders.go:`isApprover`.

6. The optional `priority_second_approver` must be an eligible second approver and is auto-cleared when `approval_total` is at or below the first threshold (app/hooks/purchase_orders.go: `validatePurchaseOrder` and `cleanPurchaseOrder`).

### 3. Expenses records must reference a `purchase_orders` record under the following conditions

1. Basic conditions

   1. the expense has a non-empty `job` and `payment_type` is not one of `Mileage`, `FuelCard`, `PersonalReimbursement`, or `Allowance` (app/hooks/validate_expenses.go: `validateExpense`);
   2. for non-exempt types without a `purchase_order`, totals greater than or equal to `constants.NO_PO_EXPENSE_LIMIT` (currently $100) are rejected, effectively requiring a PO unless the caller has the `payables_admin` claim and `payment_type` is `OnAccount` (app/constants/constants.go and app/hooks/validate_expenses.go: `validateExpense` with `byPassTotalLimit` set via app/hooks/expenses.go: `ProcessExpense`).

2. Additionally, any provided `purchase_order` must have status `Active` (app/hooks/expenses.go: `ProcessExpense`).

3. When an `expenses` record references a `purchase_orders` record, the following validations also apply (app/hooks/validate_expenses.go: `validateExpense`):
   1. The `expenses` record's `date` must be on or after the PO's date (uses `utilities.DateStringLimit` with the PO's date).
   2. If the PO type is `Recurring`, the expense `date` must be on or before the PO's `end_date` (uses utilities.DateStringLimit with max=true).
   3. For `Normal` or `Recurring` POs, the expense `total` must not exceed the allowed overage limit (the lesser of `constants.MAX_PURCHASE_ORDER_EXCESS_PERCENT` or `constants.MAX_PURCHASE_ORDER_EXCESS_VALUE` over the PO total); enforced via validation.Max(totalLimit) (app/hooks/validate_expenses.go: `validateExpense`; app/constants/constants.go).
   4. For `Cumulative` POs, the sum of existing expenses plus the new expense must not exceed the PO total; overflow returns a `cumulative_po_overflow` error with details (app/hooks/validate_expenses.go: `validateExpense`; existing total computed in app/hooks/expenses.go: ProcessExpense via utilities.CumulativeTotalExpensesForPurchaseOrder).

### 4. As of now, anybody can submit an expense against any purchase_orders record if the record has a `status` field value of `Active`

1. The only enforced constraint today is that the referenced `purchase_orders` record must have status `Active` (app/hooks/expenses.go: `ProcessExpense`).

2. There are no authorization checks restricting who may create an expense against an Active purchase order; this is explicitly left as a TODO in code (app/hooks/expenses.go: ProcessExpense — "WHO CAN CREATE AN EXPENSE AGAINST AN ACTIVE PURCHASE ORDER?").
