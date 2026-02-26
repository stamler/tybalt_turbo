// queue_reminders.go contains batched reminder queueing functions.
//
// It provides a generic ReminderJob engine for querying recipients,
// deduplicating pending/inflight reminders, creating notification records,
// and optionally sending queued notifications.
package notifications

import (
	"fmt"
	"strings"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// QueuePoSecondApproverNotifications creates pending notifications for users in
// pending_items_for_qualified_po_second_approvers with num_pos_qualified > 0.
//
// This intentionally queries the PocketBase view via FindRecordsByFilter
// instead of the ReminderJob raw-SQL engine. The source is a PB view and this
// flow keeps the existing record-oriented query path.
//
// When send is true, it immediately drains the pending queue after creation;
// otherwise notifications remain pending for later delivery.
func QueuePoSecondApproverNotifications(app core.App, send bool) error {
	records, err := app.FindRecordsByFilter(
		"pending_items_for_qualified_po_second_approvers",
		"num_pos_qualified > 0",
		"",
		0,
		0,
	)
	if err != nil {
		app.Logger().Error(
			"error querying pending_items_for_qualified_po_second_approvers",
			"error", err,
		)
		return fmt.Errorf("error querying pending_items_for_qualified_po_second_approvers: %v", err)
	}

	createdCount := 0
	for _, record := range records {
		recipientID := record.GetString("id")
		notificationID, err := DispatchNotification(app, DispatchArgs{
			TemplateCode: "po_second_approval_required",
			RecipientUID: recipientID,
			Data: map[string]any{
				"ActionURL": BuildActionURL(app, "/pos/list"),
			},
			System: true,
			Mode:   DeliveryDeferred,
		})
		if err != nil {
			app.Logger().Error(
				"error creating second approval notification",
				"error", err,
				"recipient_id", recipientID,
			)
			return fmt.Errorf("error saving second approval notification: %v", err)
		}
		if notificationID != "" {
			createdCount++
		}
	}

	app.Logger().Info(
		"queued second approval notifications",
		"candidate_count", len(records),
		"created_count", createdCount,
	)

	return sendQueuedIfRequested(app, send, "sent all notifications")
}

// QueueTimesheetSubmissionReminders creates reminders for users missing a
// submitted timesheet for the previous week.
//
// It computes the previous week ending date and delegates to
// QueueTimesheetSubmissionRemindersForWeek.
func QueueTimesheetSubmissionReminders(app core.App, send bool) error {
	today := time.Now()
	todayStr := today.Format(time.DateOnly)
	weekEnding, err := utilities.GenerateWeekEnding(todayStr)
	if err != nil {
		return fmt.Errorf("error calculating week ending: %v", err)
	}

	weekEndingTime, err := time.Parse(time.DateOnly, weekEnding)
	if err != nil {
		return fmt.Errorf("error parsing week ending: %v", err)
	}

	previousWeekEnding := weekEndingTime.AddDate(0, 0, -7).Format(time.DateOnly)

	return QueueTimesheetSubmissionRemindersForWeek(app, previousWeekEnding, send)
}

// QueueTimesheetSubmissionRemindersForWeek queues reminders for users expected
// to submit a timesheet but missing a submission for the specified week ending.
//
// Dedupe is week-based: it skips recipients that already have a pending or
// inflight reminder for the same WeekEnding payload.
func QueueTimesheetSubmissionRemindersForWeek(app core.App, weekEnding string, send bool) error {
	job := ReminderJob{
		Name:         "timesheet submission reminders",
		TemplateCode: "timesheet_submission_reminder",
		Query: `
			SELECT DISTINCT
				u.id AS recipient_uid
			FROM users u
			LEFT JOIN time_sheets ts ON ts.uid = u.id AND ts.week_ending = {:week_ending} AND ts.submitted = 1
			LEFT JOIN admin_profiles ap ON ap.uid = u.id
			WHERE ts.id IS NULL
			  AND COALESCE(ap.time_sheet_expected, 0) = 1
		`,
		QueryParams:  dbx.Params{"week_ending": weekEnding},
		RecipientCol: "recipient_uid",
		Dedupe: DedupeSpec{
			Where: "json_extract(n.data, '$.WeekEnding') = {:week_ending}",
			Params: func(row dbx.NullStringMap) dbx.Params {
				return dbx.Params{"week_ending": weekEnding}
			},
		},
		BuildData: func(row dbx.NullStringMap) map[string]any {
			return map[string]any{
				"WeekEnding": weekEnding,
				"ActionURL":  BuildActionURL(app, "/time/entries/list"),
			}
		},
		LogFields: map[string]any{
			"week_ending": weekEnding,
		},
	}

	if err := queueReminderJob(app, job, send); err != nil {
		return err
	}
	return nil
}

// QueueTimesheetApprovalReminders queues reminders for managers who currently
// have submitted timesheets awaiting approval.
//
// Dedupe uses a rolling 24-hour window on pending/inflight reminders per
// recipient and template.
func QueueTimesheetApprovalReminders(app core.App, send bool) error {
	job := ReminderJob{
		Name:         "timesheet approval reminders",
		TemplateCode: "timesheet_approval_reminder",
		Query: `
			SELECT DISTINCT
				ts.approver AS manager_uid
			FROM time_sheets ts
			WHERE ts.submitted = 1
			  AND ts.approved = ''
			  AND ts.committed = ''
			  AND ts.rejected = ''
			  AND ts.approver != ''
		`,
		RecipientCol: "manager_uid",
		Dedupe: DedupeSpec{
			Where: "datetime(n.created) > datetime('now', '-1 day')",
		},
		BuildData: func(row dbx.NullStringMap) map[string]any {
			return map[string]any{
				"ActionURL": BuildActionURL(app, "/time/sheets/pending"),
			}
		},
	}

	if err := queueReminderJob(app, job, send); err != nil {
		return err
	}
	return nil
}

// QueueExpenseApprovalReminders queues reminders for managers who currently
// have submitted expenses awaiting approval.
//
// Dedupe uses a rolling 24-hour window on pending/inflight reminders per
// recipient and template.
func QueueExpenseApprovalReminders(app core.App, send bool) error {
	job := ReminderJob{
		Name:         "expense approval reminders",
		TemplateCode: "expense_approval_reminder",
		Query: `
			SELECT DISTINCT
				e.approver AS manager_uid
			FROM expenses e
			WHERE e.submitted = 1
			  AND e.approved = ''
			  AND e.committed = ''
			  AND e.rejected = ''
			  AND e.approver != ''
		`,
		RecipientCol: "manager_uid",
		Dedupe: DedupeSpec{
			Where: "datetime(n.created) > datetime('now', '-1 day')",
		},
		BuildData: func(row dbx.NullStringMap) map[string]any {
			return map[string]any{
				"ActionURL": BuildActionURL(app, "/expenses/pending"),
			}
		},
	}

	if err := queueReminderJob(app, job, send); err != nil {
		return err
	}
	return nil
}

func queueReminderJob(app core.App, job ReminderJob, send bool) error {
	notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
		"code": job.TemplateCode,
	})
	if err != nil {
		return fmt.Errorf("error finding notification template: %v", err)
	}

	rows := make([]dbx.NullStringMap, 0)
	query := app.DB().NewQuery(job.Query)
	if len(job.QueryParams) > 0 {
		query.Bind(job.QueryParams)
	}
	err = query.All(&rows)
	if err != nil {
		app.Logger().Error(
			"error querying reminder recipients",
			"job", job.Name,
			"error", err,
		)
		return fmt.Errorf("error querying reminder recipients for %s: %v", job.Name, err)
	}

	createdCount := 0
	for _, row := range rows {
		recipientUID := rowStringValue(row, job.RecipientCol)
		if recipientUID == "" {
			app.Logger().Error(
				"reminder row missing recipient UID",
				"job", job.Name,
				"recipient_col", job.RecipientCol,
			)
			continue
		}

		if strings.TrimSpace(job.Dedupe.Where) != "" {
			dedupeParams := dbx.Params{}
			if job.Dedupe.Params != nil {
				dedupeParams = job.Dedupe.Params(row)
			}

			exists, err := notificationExists(app, recipientUID, notificationTemplate.Id, job.Dedupe.Where, dedupeParams)
			if err != nil {
				app.Logger().Error(
					"error checking for existing notification",
					"recipient", recipientUID,
					"job", job.Name,
					"error", err,
				)
				continue
			}
			if exists {
				continue
			}
		}

		data := map[string]any{}
		if job.BuildData != nil {
			data = job.BuildData(row)
		}

		notificationID, err := DispatchNotification(app, DispatchArgs{
			TemplateCode: job.TemplateCode,
			RecipientUID: recipientUID,
			Data:         data,
			System:       true,
			Mode:         DeliveryDeferred,
		})
		if err != nil {
			app.Logger().Error(
				"error creating reminder notification",
				"recipient", recipientUID,
				"job", job.Name,
				"error", err,
			)
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
	}

	logArgs := []any{"created_count", createdCount}
	for key, value := range job.LogFields {
		logArgs = append(logArgs, key, value)
	}
	app.Logger().Info("queued "+job.Name, logArgs...)

	return sendQueuedIfRequested(app, send, "sent "+job.Name+" notifications")
}

func notificationExists(app core.App, recipientUID, templateID string, where string, params dbx.Params) (bool, error) {
	query := `
		SELECT COUNT(*) AS count
		FROM notifications n
		WHERE n.recipient = {:recipient}
		  AND n.template = {:template}
		  AND n.status IN ('pending', 'inflight')
	`

	whereClause := strings.TrimSpace(where)
	if whereClause != "" {
		query += " AND (" + whereClause + ")"
	}

	allParams := dbx.Params{
		"recipient": recipientUID,
		"template":  templateID,
	}
	for key, value := range params {
		allParams[key] = value
	}

	var result struct {
		Count int `db:"count"`
	}
	if err := app.DB().NewQuery(query).Bind(allParams).One(&result); err != nil {
		return false, err
	}

	return result.Count > 0, nil
}
