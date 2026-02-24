package migrations

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type notificationTemplateUpdate struct {
	Code    string
	Text    string
	Default string
}

func migrationBuildActionURL(app core.App, path string) string {
	base := strings.TrimRight(strings.TrimSpace(app.Settings().Meta.AppURL), "/")
	if base == "" {
		return ""
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return base
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	return base + cleanPath
}

func applyNotificationTemplateUpdate(app core.App, update notificationTemplateUpdate) error {
	if _, err := app.DB().NewQuery(`
		UPDATE notification_templates
		SET text_email = {:text}
		WHERE code = {:code}
	`).Bind(map[string]any{
		"code": update.Code,
		"text": update.Text,
	}).Execute(); err != nil {
		return fmt.Errorf("error updating notification template %s: %w", update.Code, err)
	}

	if update.Default == "" {
		return nil
	}

	if _, err := app.DB().NewQuery(`
		UPDATE notifications
		SET data = json_set(
			CASE WHEN json_valid(data) THEN data ELSE '{}' END,
			'$.ActionURL',
			{:actionURL}
		)
		WHERE template IN (
			SELECT id
			FROM notification_templates
			WHERE code = {:code}
		)
		  AND json_extract(
			CASE WHEN json_valid(data) THEN data ELSE '{}' END,
			'$.ActionURL'
		  ) IS NULL
	`).Bind(map[string]any{
		"code":      update.Code,
		"actionURL": update.Default,
	}).Execute(); err != nil {
		return fmt.Errorf("error backfilling ActionURL for %s notifications: %w", update.Code, err)
	}

	return nil
}

func applyNotificationTemplateRevert(app core.App, update notificationTemplateUpdate) error {
	if _, err := app.DB().NewQuery(`
		UPDATE notification_templates
		SET text_email = {:text}
		WHERE code = {:code}
	`).Bind(map[string]any{
		"code": update.Code,
		"text": update.Text,
	}).Execute(); err != nil {
		return fmt.Errorf("error reverting notification template %s: %w", update.Code, err)
	}

	if _, err := app.DB().NewQuery(`
		UPDATE notifications
		SET data = json_remove(
			CASE WHEN json_valid(data) THEN data ELSE '{}' END,
			'$.ActionURL'
		)
		WHERE template IN (
			SELECT id
			FROM notification_templates
			WHERE code = {:code}
		)
		  AND json_extract(
			CASE WHEN json_valid(data) THEN data ELSE '{}' END,
			'$.ActionURL'
		  ) IS NOT NULL
	`).Bind(map[string]any{
		"code": update.Code,
	}).Execute(); err != nil {
		return fmt.Errorf("error removing ActionURL for %s notifications: %w", update.Code, err)
	}

	return nil
}

func init() {
	m.Register(func(app core.App) error {
		updates := []notificationTemplateUpdate{
			{
				Code: "expense_approval_reminder",
				Text: `Hello {{.RecipientName}},

One or more expenses are awaiting your approval.

Please review and approve or reject them here:

{{.ActionURL}}`,
				Default: migrationBuildActionURL(app, "/expenses/pending"),
			},
			{
				Code: "expense_rejected",
				Text: `Hello {{.RecipientName}},

The expense submitted by {{.EmployeeName}} on {{.ExpenseDate}} for {{.ExpenseAmount}} was rejected by {{.RejectorName}} for the following reason:

{{.RejectionReason}}

You can review the expense and make any required changes here:

{{.ActionURL}}`,
				Default: migrationBuildActionURL(app, "/expenses/list"),
			},
			{
				Code: "po_active",
				Text: `Hello {{.RecipientName}}, your purchase order {{.PONumber}} is now active. You may submit expenses against it here:

{{.ActionURL}}

Thank you.`,
				Default: migrationBuildActionURL(app, "/pos/list"),
			},
			{
				Code: "po_approval_required",
				Text: `Hello {{.RecipientName}}, {{.UserName}} has created a purchase order and specified you as the approver. You may review the purchase order then approve or reject it here:

{{.ActionURL}}

Thank you.`,
				Default: migrationBuildActionURL(app, "/pos/list"),
			},
			{
				Code: "po_priority_second_approval_required",
				Text: `Hello {{.RecipientName}}, {{.POCreatorName}} has had a purchase order approved by {{.UserName}} but it requires second approval. They have requested that you be given priority to approve the purchase order. After 24 hours, the purchase order will be available for approval by all qualified approvers.

You may review the purchase order here:

{{.ActionURL}}

Thank you.`,
				Default: migrationBuildActionURL(app, "/pos/list"),
			},
			{
				Code: "po_rejected",
				Text: `Hello {{.RecipientName}}, your purchase order was rejected by {{.UserName}}.

{{.ActionURL}}

Thank you.`,
				Default: migrationBuildActionURL(app, "/pos/list"),
			},
			{
				Code: "po_second_approval_required",
				Text: `Hello {{.RecipientName}}, there are one or more purchase orders awaiting second approval. Please review them then accept or reject them here:

{{.ActionURL}}

Thank you.`,
				Default: migrationBuildActionURL(app, "/pos/list"),
			},
			{
				Code: "timesheet_approval_reminder",
				Text: `Hello {{.RecipientName}},

One or more timesheets are awaiting your approval.

Please review and approve or reject them here:

{{.ActionURL}}`,
				Default: migrationBuildActionURL(app, "/time/sheets/pending"),
			},
			{
				Code: "timesheet_rejected",
				Text: `Hello {{.RecipientName}},

The timesheet for {{.EmployeeName}} for the week ending {{.WeekEnding}} was rejected by {{.RejectorName}} for the following reason:

{{.RejectionReason}}

You can review the timesheet and make any required changes here:

{{.ActionURL}}`,
				Default: migrationBuildActionURL(app, "/time/sheets/list"),
			},
			{
				Code: "timesheet_shared",
				Text: `Hello {{.RecipientName}},

{{.UserName}} has shared a timesheet for {{.EmployeeName}} for the week ending {{.WeekEnding}} with you for review.

You can view the shared timesheet here:

{{.ActionURL}}`,
				Default: migrationBuildActionURL(app, "/time/sheets/list"),
			},
			{
				Code: "timesheet_submission_reminder",
				Text: `Hello {{.RecipientName}},

You have not submitted your timesheet for the week ending {{.WeekEnding}}.

Please review your time entries then submit here:

{{.ActionURL}}`,
				Default: migrationBuildActionURL(app, "/time/entries/list"),
			},
		}

		for _, update := range updates {
			if err := applyNotificationTemplateUpdate(app, update); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		updates := []notificationTemplateUpdate{
			{
				Code: "expense_approval_reminder",
				Text: `Hello {{.RecipientName}},

One or more expenses are awaiting your approval.

Please review and approve or reject them here:

{APP_URL}/expenses/pending`,
			},
			{
				Code: "expense_rejected",
				Text: `Hello {{.RecipientName}},

The expense submitted by {{.EmployeeName}} on {{.ExpenseDate}} for {{.ExpenseAmount}} was rejected by {{.RejectorName}} for the following reason:

{{.RejectionReason}}

You can review the expense and make any required changes here:

{APP_URL}/expenses/{:RECORD_ID}/details`,
			},
			{
				Code: "po_active",
				Text: `Hello {{.RecipientName}}, your purchase order {{.PONumber}} is now active. You may submit expenses against the purchase order here:

https://tybalt.tbte.ca/pos/{{.POId}}/details

Thank you.`,
			},
			{
				Code: "po_approval_required",
				Text: `Hello {{.RecipientName}}, {{.UserName}} has created a purchase order and specified you as the approver. You may review the purchase order then approve or reject it here:

https://tybalt.tbte.ca/pos/{{.POId}}/edit

Thank you.`,
			},
			{
				Code: "po_priority_second_approval_required",
				Text: `Hello {{.RecipientName}}, {{.POCreatorName}} has had a purchase order approved by {{.UserName}} but it requires second approval. They have requested that you be given priority to approve the purchase order. After 24 hours, the purchase order will be available for approval by all qualified approvers.

You may review the purchase order here:

https://tybalt.tbte.ca/pos/{{.POId}}/edit

Thank you.`,
			},
			{
				Code: "po_rejected",
				Text: `Hello {{.RecipientName}}, your purchase order was rejected by {{.UserName}}.

http://tybalt.tbte.ca/pos/{{.POId}}/details

Thank you.`,
			},
			{
				Code: "po_second_approval_required",
				Text: `Hello {{.RecipientName}}, there are one or more purchase orders awaiting second approval. Please review them then accept or reject them here:

https://tybalt.tbte.ca/pos/list

Thank you.`,
			},
			{
				Code: "timesheet_approval_reminder",
				Text: `Hello {{.RecipientName}},

One or more timesheets are awaiting your approval.

Please review and approve or reject them here:

{APP_URL}/time/sheets/pending`,
			},
			{
				Code: "timesheet_rejected",
				Text: `Hello {{.RecipientName}},

The timesheet for {{.EmployeeName}} for the week ending {{.WeekEnding}} was rejected by {{.RejectorName}} for the following reason:

{{.RejectionReason}}

You can review the timesheet and make any required changes here:

{APP_URL}/time/sheets/{:RECORD_ID}/details`,
			},
			{
				Code: "timesheet_shared",
				Text: `Hello {{.RecipientName}},

{{.UserName}} has shared a timesheet for {{.EmployeeName}} for the week ending {{.WeekEnding}} with you for review.

You can view the shared timesheet here:

{APP_URL}/time/sheets/{:RECORD_ID}/details`,
			},
			{
				Code: "timesheet_submission_reminder",
				Text: `Hello {{.RecipientName}},

You have not submitted your timesheet for the week ending {{.WeekEnding}}.

Please review your time entries then submit here:

{APP_URL}/time/entries/list`,
			},
		}

		for _, update := range updates {
			if err := applyNotificationTemplateRevert(app, update); err != nil {
				return err
			}
		}

		return nil
	})
}
