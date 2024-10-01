package routes

import (
	"mime/multipart"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Define request bodies for the handlers
type WeekEndingRequest struct {
	WeekEnding string `json:"weekEnding"`
}

type RecordIdRequest struct {
	RecordId string `json:"recordId"`
}

type RejectionRequest struct {
	RejectionReason string `json:"rejectionReason"`
}

type PurchaseOrderRequest struct {
	Type        string                `json:"type"`
	Date        string                `json:"date"`
	EndDate     string                `json:"end_date"`
	Frequency   string                `json:"frequency"`
	Division    string                `json:"division"`
	Description string                `json:"description"`
	Total       float64               `json:"total"`
	PaymentType string                `json:"payment_type"`
	VendorName  string                `json:"vendor_name"`
	Attachment  *multipart.FileHeader `json:"attachment"`
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
func AddRoutes(app *pocketbase.PocketBase) {

	// Add the bundle-timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/bundle-timesheet",
			Handler: createBundleTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// add the unbundle-timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/unbundle-timesheet",
			Handler: createUnbundleTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the approve-timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/approve-timesheet",
			Handler: createApproveTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the reject-timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/time_sheets/:id/reject",
			Handler: createRejectTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the approve purchase order route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/purchase_orders/:id/approve",
			Handler: approvePurchaseOrderHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the reject purchase order route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/purchase_orders/:id/reject",
			Handler: rejectPurchaseOrderHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the cancel purchase order route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/purchase_orders/:id/cancel",
			Handler: cancelPurchaseOrderHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the approve expense route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/expenses/:id/approve",
			Handler: createApproveExpenseHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the reject expense route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/expenses/:id/reject",
			Handler: createRejectExpenseHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})
}
