### Notification Templates for Timesheets and Expenses

Proposed notification templates for the unimplemented notifications described in [Issue #86](https://github.com/stamler/tybalt_turbo/issues/86).

All link placeholders use the form `__LINK_TO_<THING>__` and should be replaced with real URLs when wiring up the notifications.

---

### `timesheet_submission_reminder`

- **Code**: `timesheet_submission_reminder`
- **Description**: Sent to users who have not submitted their timesheet for the previous week.
- **Subject**: `Reminder: submit your timesheet for {{.WeekEnding}}`
- **Text email**:

```text
Hello {{.RecipientName}},

You have not submitted your timesheet for the week ending {{.WeekEnding}}.

Please review your time entries then submit here:

{APP_URL}/time/entries/list

Thank you.
```

---

### `expense_approval_reminder`

- **Code**: `expense_approval_reminder`
- **Description**: Sent to managers with one or more expenses awaiting their approval.
- **Subject**: `You have expenses awaiting approval`
- **Text email**:

```text
Hello {{.RecipientName}},

One or more expenses are awaiting your approval.

Please review and approve or reject them here:

{APP_URL}/expenses/pending

Thank you.
```

---

### `timesheet_approval_reminder`

- **Code**: `timesheet_approval_reminder`
- **Description**: Sent to managers with one or more timesheets awaiting their approval.
- **Subject**: `You have timesheets awaiting approval`
- **Text email**:

```text
Hello {{.RecipientName}},

One or more timesheets are awaiting your approval.

Please review and approve or reject them here:

{APP_URL}/time/sheets/pending

Thank you.
```

---

### `timesheet_rejected`

- **Code**: `timesheet_rejected`
- **Description**: Sent when a timesheet is rejected to the employee, the rejector, and the employee's manager (if different from the rejector).
- **Subject**: `Timesheet was rejected`
- **Text email**:

```text
Hello {{.RecipientName}},

The timesheet for {{.EmployeeName}} for the week ending {{.WeekEnding}} was rejected by {{.RejectorName}} for the following reason:

{{.RejectionReason}}

You can review the timesheet and make any required changes here:

{APP_URL}/time/sheets/{:RECORD_ID}/details

Thank you.
```

---

### `expense_rejected`

- **Code**: `expense_rejected`
- **Description**: Sent when an expense is rejected to the employee, the rejector, and the employee's manager (if different from the rejector).
- **Subject**: `Expense was rejected`
- **Text email**:

```text
Hello {{.RecipientName}},

The expense submitted by {{.EmployeeName}} on {{.ExpenseDate}} for {{.ExpenseAmount}} was rejected by {{.RejectorName}} for the following reason:

{{.RejectionReason}}

You can review the expense and make any required changes here:

{APP_URL}/expenses/{:RECORD_ID}/details

Thank you.
```

---

### `timesheet_shared`

- **Code**: `timesheet_shared`
- **Description**: Sent to newly added viewers when a timesheet is shared with them for review.
- **Subject**: `A timesheet has been shared with you`
- **Text email**:

```text
Hello {{.RecipientName}},

{{.UserName}} has shared a timesheet for {{.EmployeeName}} for the week ending {{.WeekEnding}} with you for review.

You can view the shared timesheet here:

{APP_URL}/time/sheets/{:RECORD_ID}/details

Thank you.
```
