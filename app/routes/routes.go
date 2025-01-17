package routes

import (
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

		tsGroup := se.Router.Group("/api/time_sheets")
		tsGroup.Bind(apis.RequireAuth("users"))
		tsGroup.POST("/{weekEnding}/bundle", createBundleTimesheetHandler(app))
		tsGroup.POST("/{id}/unbundle", createUnbundleTimesheetHandler(app))
		tsGroup.POST("/{id}/approve", createApproveRecordHandler(app, "time_sheets"))
		tsGroup.POST("/{id}/reject", createRejectRecordHandler(app, "time_sheets"))

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

		clientsGroup := se.Router.Group("/api/clients")
		clientsGroup.Bind(apis.RequireAuth("users"))
		clientsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "clients"))
		clientsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "clients"))

		clientContactsGroup := se.Router.Group("/api/client_contacts")
		clientContactsGroup.Bind(apis.RequireAuth("users"))
		clientContactsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "client_contacts"))
		clientContactsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "client_contacts"))

		return se.Next()
	})

}
