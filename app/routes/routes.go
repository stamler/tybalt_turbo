package routes

import (
	"net/http"
	"tybalt/constants"
	"tybalt/notifications"
	"tybalt/reports"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// requireExpensesEditing returns a 403 error if the expenses feature flag is
// off and collectionName is a Phase 2 collection ("expenses", "purchase_orders",
// or "vendors"). Returns nil otherwise.
func requireExpensesEditing(app core.App, collectionName string) error {
	switch collectionName {
	case "expenses", "purchase_orders", "vendors":
	default:
		return nil
	}
	enabled, err := utilities.IsExpensesEditingEnabled(app)
	if err != nil {
		return err
	}
	if !enabled {
		return apis.NewForbiddenError(utilities.ErrExpensesEditingDisabled.Message, nil)
	}
	return nil
}

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
		if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
			return err
		}

		// Version endpoint - publicly accessible
		se.Router.GET("/api/version", func(e *core.RequestEvent) error {
			return e.JSON(http.StatusOK, map[string]any{
				"name":           constants.AppName,
				"version":        constants.Version,
				"build":          constants.Build,
				"fullVersion":    constants.FullVersion,
				"gitCommit":      constants.GitCommit,
				"gitCommitShort": constants.GitCommitShort,
				"gitBranch":      constants.GitBranch,
				"buildTime":      constants.BuildTime,
			})
		})

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
		// Allow committers to commit time_sheets
		tsGroup.POST("/{id}/commit", createCommitRecordHandler(app, "time_sheets"))
		tsGroup.GET("/{id}/reviewers", createGetReviewersHandler(app))
		tsGroup.GET("/{id}/approver", createTimesheetApproverHandler(app))
		tsGroup.GET("/tallies", createTimesheetTalliesHandler(app, "uid", 0, 0))
		tsGroup.GET("/tallies/pending", createTimesheetTalliesHandler(app, "approver", 1, 0))
		tsGroup.GET("/tallies/approved", createTimesheetTalliesHandler(app, "approver", 0, 1))
		tsGroup.GET("/tallies/shared", createTimesheetTalliesHandler(app, "reviewer", 0, 0))
		// TimeTracking endpoints for report holders
		tsGroup.GET("/tracking_counts", createTimesheetTrackingCountsHandler(app))
		tsGroup.GET("/tracking/weeks/{weekEnding}", createTimesheetTrackingListHandler(app))
		tsGroup.GET("/tracking/weeks/{weekEnding}/missing", createTimesheetMissingHandler(app))
		tsGroup.GET("/tracking/weeks/{weekEnding}/not_expected", createTimesheetNotExpectedHandler(app))

		// Legacy writeback endpoints (custom auth via machine_secrets)
		se.Router.GET("/api/export_legacy/time_sheets/{weekEnding}", createTimesheetExportLegacyHandler(app))
		se.Router.GET("/api/export_legacy/jobs/{updatedAfter}", createJobsExportLegacyHandler(app))
		se.Router.GET("/api/export_legacy/expenses/{updatedAfter}", createExpensesExportLegacyHandler(app))

		expensesGroup := se.Router.Group("/api/expenses")
		expensesGroup.Bind(apis.RequireAuth("users"))
		expensesGroup.POST("/{id}/submit", createSubmitRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/recall", createRecallRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/approve", createApproveRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/reject", createRejectRecordHandler(app, "expenses"))
		expensesGroup.POST("/{id}/commit", createCommitRecordHandler(app, "expenses"))
		expensesGroup.GET("/list", createGetExpensesListHandler(app))
		expensesGroup.GET("/details/{id}", createGetExpenseDetailsHandler(app))
		expensesGroup.GET("/pending", createGetPendingExpensesHandler(app))
		expensesGroup.GET("/approved", createGetApprovedExpensesHandler(app))
		// Expense tracking endpoints for report holders
		expensesGroup.GET("/tracking_counts", createExpenseTrackingCountsHandler(app))
		expensesGroup.GET("/tracking", createExpenseTrackingAllHandler(app))
		expensesGroup.GET("/tracking/{payPeriodEnding}", createExpenseTrackingListHandler(app))

		timeAmendmentsGroup := se.Router.Group("/api/time_amendments")
		timeAmendmentsGroup.Bind(apis.RequireAuth("users"))
		timeAmendmentsGroup.POST("/{id}/commit", createCommitRecordHandler(app, "time_amendments"))

		// Time Entries routes
		timeEntriesGroup := se.Router.Group("/api/time_entries")
		timeEntriesGroup.Bind(apis.RequireAuth("users"))
		timeEntriesGroup.POST("/{id}/copy_to_tomorrow", createCopyTimeEntryHandler(app))

		usersGroup := se.Router.Group("/api/users")
		usersGroup.Bind(apis.RequireAuth("users"))
		usersGroup.GET("/defaults", createGetUserDefaultsHandler(app))

		poGroup := se.Router.Group("/api/purchase_orders")
		poGroup.Bind(apis.RequireAuth("users"))
		poGroup.GET("/pending", createGetPendingPurchaseOrdersHandler(app))
		poGroup.GET("/pending/{id}", createGetPendingPurchaseOrderHandler(app))
		poGroup.GET("/visible", createGetVisiblePurchaseOrdersHandler(app))
		poGroup.GET("/visible/{id}", createGetVisiblePurchaseOrderHandler(app))
		poGroup.GET("/approvers", createGetApproversHandler(app, false))
		poGroup.GET("/second_approvers", createGetApproversHandler(app, true))

		// PO mutation routes are gated on expenses editing via middleware
		poMutations := se.Router.Group("/api/purchase_orders")
		poMutations.Bind(apis.RequireAuth("users"))
		poMutations.BindFunc(func(e *core.RequestEvent) error {
			if err := requireExpensesEditing(app, "purchase_orders"); err != nil {
				return err
			}
			return e.Next()
		})
		poMutations.POST("/{id}/approve", createApprovePurchaseOrderHandler(app))
		poMutations.POST("/{id}/reject", createRejectPurchaseOrderHandler(app))
		poMutations.POST("/{id}/cancel", createCancelPurchaseOrderHandler(app))
		poMutations.POST("/{id}/close", createClosePurchaseOrderHandler(app))
		poMutations.POST("/{id}/make_cumulative", createConvertToCumulativePurchaseOrderHandler(app))

		clientsGroup := se.Router.Group("/api/clients")
		clientsGroup.Bind(apis.RequireAuth("users"))
		clientsGroup.GET("", createGetClientsHandler(app))
		clientsGroup.GET("/{id}", createGetClientsHandler(app))
		clientsGroup.GET("/{id}/notes", createGetClientNotesHandler(app))
		clientsGroup.GET("/devleads", createGetBusdevLeadsHandler(app))
		clientsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "clients"))
		clientsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "clients"))

		clientContactsGroup := se.Router.Group("/api/client_contacts")
		clientContactsGroup.Bind(apis.RequireAuth("users"))
		clientContactsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "client_contacts"))
		clientContactsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "client_contacts"))

		// Vendor absorb routes
		vendorsGroup := se.Router.Group("/api/vendors")
		vendorsGroup.Bind(apis.RequireAuth("users"))
		vendorsGroup.GET("", createGetVendorsHandler(app))
		vendorsGroup.GET("/{id}", createGetVendorsHandler(app))
		vendorsGroup.POST("/{id}/absorb", CreateAbsorbRecordsHandler(app, "vendors"))
		vendorsGroup.POST("/undo_absorb", CreateUndoAbsorbHandler(app, "vendors"))

		// Jobs endpoints – provide aggregated data in a single query to avoid PocketBase's N+1 expand problem
		jobsGroup := se.Router.Group("/api/jobs")
		jobsGroup.Bind(apis.RequireAuth("users"))
		jobsGroup.POST("", createCreateJobHandler(app))
		jobsGroup.PUT("/{id}", createUpsertJobHandler(app))
		jobsGroup.GET("/{id}/details", createGetJobDetailsHandler(app))
		jobsGroup.GET("/{id}/notes", createGetJobNotesHandler(app))
		jobsGroup.GET("/latest", createGetLatestJobsHandler(app))
		// Query parameters for the following two routes will be used to filter
		// the results. The caller can filter by one or more of division,
		// time_type, user, or category.
		jobsGroup.GET("/{id}/time/summary", createGetJobTimeSummaryHandler(app))
		jobsGroup.GET("/{id}/time/full_report", reports.CreateJobTimeReportHandler(app))
		jobsGroup.GET("/{id}/time/entries", createGetJobTimeEntriesHandler(app))
		jobsGroup.GET("/{id}/staff/summary", createGetJobStaffSummaryHandler(app))
		jobsGroup.GET("/{id}/divisions/summary", createGetJobDivisionsSummaryHandler(app))
		jobsGroup.GET("/{id}/expenses/summary", createGetJobExpenseSummaryHandler(app))
		jobsGroup.GET("/{id}/expenses/list", createGetJobExpensesHandler(app))
		jobsGroup.GET("/{id}/pos/summary", createGetJobPOSummaryHandler(app))
		jobsGroup.GET("/{id}/pos/list", createGetJobPOsHandler(app))
		jobsGroup.GET("/{id}", createGetJobsHandler(app))
		jobsGroup.GET("", createGetJobsHandler(app))
		jobsGroup.GET("/unused", createGetUnusedJobsHandler(app))
		jobsGroup.GET("/stale", createGetStaleJobsHandler(app))
		jobsGroup.POST("/{id}/set-status", createSetJobStatusHandler(app))
		jobsGroup.GET("/{id}/validate-proposal", createValidateProposalHandler(app))

		reportsGroup := se.Router.Group("/api/reports")
		reportsGroup.Bind(apis.RequireAuth("users"))
		reportsGroup.GET("/payroll_time/{date_column_value}/{week}", reports.CreatePayrollTimeReportHandler(app))
		reportsGroup.GET("/weekly_time/{date_column_value}", reports.CreateTimeReportHandler(app, "week_ending", "committed_week_ending"))
		reportsGroup.GET("/weekly_time_by_employee/{date_column_value}", reports.CreateTimeSummaryByEmployeeHandler(app, "week_ending"))
		reportsGroup.GET("/payroll_expense/{date_column_value}", reports.CreateExpenseReportHandler(app, "pay_period_ending"))
		reportsGroup.GET("/weekly_expense/{date_column_value}", reports.CreateExpenseReportHandler(app, "committed_week_ending"))
		reportsGroup.GET("/payroll_receipts/{date_column_value}", reports.CreateReceiptsReportHandler(app, "pay_period_ending"))
		reportsGroup.GET("/weekly_receipts/{date_column_value}", reports.CreateReceiptsReportHandler(app, "committed_week_ending"))

		// Machine secrets management (admin only)
		machineSecretsGroup := se.Router.Group("/api/machine_secrets")
		machineSecretsGroup.Bind(apis.RequireAuth("users"))
		machineSecretsGroup.GET("/list", listMachineSecretsHandler(app))
		machineSecretsGroup.POST("/create", createMachineSecretHandler(app))

		// Rate sheets management (job claim required)
		rateSheetsGroup := se.Router.Group("/api/rate_sheets")
		rateSheetsGroup.Bind(apis.RequireAuth("users"))
		rateSheetsGroup.POST("", createRateSheetHandler(app))
		rateSheetsGroup.POST("/{id}/activate", createActivateRateSheetHandler(app))
		rateSheetsGroup.POST("/{id}/deactivate", createDeactivateRateSheetHandler(app))

		// Rate sheet entries management (admin claim required)
		rateSheetEntriesGroup := se.Router.Group("/api/rate_sheet_entries")
		rateSheetEntriesGroup.Bind(apis.RequireAuth("users"))
		rateSheetEntriesGroup.PUT("/{id}", createUpdateRateSheetEntryHandler(app))

		return se.Next()
	})

}
