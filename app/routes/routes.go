package routes

import (
	"net/http"
	"tybalt/notifications"
	"tybalt/reports"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Define request bodies for the handlers
type RejectionRequest struct {
	RejectionReason string `json:"rejection_reason"`
}

// CodeError is a custom error type that includes a code
type CodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *CodeError) Error() string {
	return e.Message
}

// Add custom routes to the app
func AddRoutes(app core.App) {

	// Add the bundle timesheet route
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {

		// TODO: This is a temporary route to send a single notification for testing
		// purposes remove this before going to production
		notificationsGroup := se.Router.Group("/api/notifications")
		notificationsGroup.POST("/send_one", func(e *core.RequestEvent) error {
			remaining, err := notifications.SendNextPendingNotification(app)
			if err != nil {
				return e.Error(http.StatusInternalServerError, err.Error(), nil)
			}
			return e.JSON(200, map[string]any{
				"remaining": remaining,
			})
		})
		// TODO: this is a temporary route to send all notifications for testing
		// purposes remove this before going to production
		notificationsGroup.POST("/send_all", func(e *core.RequestEvent) error {
			sentCount, err := notifications.SendNotifications(app)
			if err != nil {
				return e.Error(http.StatusInternalServerError, err.Error(), nil)
			}
			return e.JSON(http.StatusOK, map[string]any{"notificationsSent": sentCount})
		})
		// TODO: This is a temporary route to send the po_second_approval_required
		// notifications for testing purposes. Remove this before going to
		// production.
		notificationsGroup.POST("/send_po_second_approval_notifications", func(e *core.RequestEvent) error {
			err := notifications.QueuePoSecondApproverNotifications(app, true)
			return e.JSON(http.StatusOK, map[string]any{"error": err})
		})

		tsGroup := se.Router.Group("/api/time_sheets")
		tsGroup.Bind(apis.RequireAuth("users"))
		tsGroup.POST("/{weekEnding}/bundle", createBundleTimesheetHandler(app))
		tsGroup.POST("/{id}/unbundle", createUnbundleTimesheetHandler(app))
		tsGroup.POST("/{id}/approve", createApproveRecordHandler(app, "time_sheets"))
		tsGroup.POST("/{id}/reject", createRejectRecordHandler(app, "time_sheets"))
		tsGroup.GET("/tallies", createTimesheetTalliesHandler(app))

		expensesGroup := se.Router.Group("/api/expenses")
		expensesGroup.Bind(apis.RequireAuth("users"))
		expensesGroup.POST("/{id}/submit", createSubmitRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/recall", createRecallRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/approve", createApproveRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/reject", createRejectRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/commit", createCommitRecordHandler(app, "expenses"))

		timeAmendmentsGroup := se.Router.Group("/api/time_amendments")
		timeAmendmentsGroup.Bind(apis.RequireAuth("users"))
		timeAmendmentsGroup.POST("/{id}/commit", createCommitRecordHandler(app, "time_amendments"))

		poGroup := se.Router.Group("/api/purchase_orders")
		poGroup.Bind(apis.RequireAuth("users"))
		poGroup.POST("/{id}/approve", createApprovePurchaseOrderHandler(app))
		poGroup.POST("/{id}/reject", createRejectPurchaseOrderHandler(app))
		poGroup.POST("/{id}/cancel", createCancelPurchaseOrderHandler(app))
		poGroup.POST("/{id}/close", createClosePurchaseOrderHandler(app))
		poGroup.POST("/{id}/make_cumulative", createConvertToCumulativePurchaseOrderHandler(app))
		poGroup.GET("/approvers/{division}/{amount}", createGetApproversHandler(app, false))
		poGroup.GET("/second_approvers/{division}/{amount}", createGetApproversHandler(app, true))

		clientsGroup := se.Router.Group("/api/clients")
		clientsGroup.Bind(apis.RequireAuth("users"))
		clientsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "clients"))
		clientsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "clients"))

		clientContactsGroup := se.Router.Group("/api/client_contacts")
		clientContactsGroup.Bind(apis.RequireAuth("users"))
		clientContactsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "client_contacts"))
		clientContactsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "client_contacts"))

		reportsGroup := se.Router.Group("/api/reports")
		reportsGroup.Bind(apis.RequireAuth("users"))
		reportsGroup.GET("/payroll_time/{payrollEnding}/{week}", reports.CreatePayrollTimeReportHandler(app))
		reportsGroup.GET("/payroll_expense/{payrollEnding}", reports.CreatePayrollExpenseReportHandler(app))
		reportsGroup.GET("/payroll_receipts/{payrollEnding}", reports.CreatePayrollReceiptsReportHandler(app))
		return se.Next()
	})

}
