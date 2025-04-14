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
		notifications.QueuePoSecondApproverNotifications(app, true)
	})
}
