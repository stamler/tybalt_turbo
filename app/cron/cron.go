package cron

import (
	"tybalt/notifications"

	"github.com/pocketbase/pocketbase/core"
)

func AddCronJobs(app core.App) {
	// send po_second_approval_required notifications at 9pm UTC every day. These
	// notifications are sent to the qualified second approvers when a PO is
	// awaiting second approval but the priority_second_approver has not yet
	// approved the PO after 24 hours.
	app.Cron().MustAdd("po_second_approval_notifications", "0 21 * * *", func() {
		// The true flag will send the notifications immediately. In the future we
		// may use false and then schedule a job to send all pending notifications
		// at regular intervals.
		notifications.QueuePoSecondApproverNotifications(app, true)
	})

	// send timesheet_submission_reminder notifications at 8am UTC on Tuesday, Wednesday, and Thursday.
	// These notifications remind users who haven't submitted their timesheet for the previous week.
	app.Cron().MustAdd("timesheet_submission_reminders", "0 8 * * 2-4", func() {
		notifications.QueueTimesheetSubmissionReminders(app, true)
	})

	// send expense_approval_reminder notifications at 9am UTC on Thursday and Friday.
	// These notifications remind managers to approve submitted expenses from their staff.
	app.Cron().MustAdd("expense_approval_reminders", "0 9 * * 4-5", func() {
		notifications.QueueExpenseApprovalReminders(app, true)
	})

	// send timesheet_approval_reminder notifications at 12pm UTC on Tuesday, Wednesday, and Thursday.
	// These notifications remind managers to approve submitted timesheets from their staff.
	app.Cron().MustAdd("timesheet_approval_reminders", "0 12 * * 2-4", func() {
		notifications.QueueTimesheetApprovalReminders(app, true)
	})
}
