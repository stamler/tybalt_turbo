package routes

import (
	"net/http"

	"github.com/labstack/echo/v5"
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
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/time_sheets/:weekEnding/bundle",
			Handler: createBundleTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// add the unbundle timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/time_sheets/:id/unbundle",
			Handler: createUnbundleTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the approve timesheet route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/time_sheets/:id/approve",
			Handler: createApproveRecordHandler(app, "time_sheets"),
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
			Handler: createRejectRecordHandler(app, "time_sheets"),
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
			Handler: createApprovePurchaseOrderHandler(app),
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
			Handler: createRejectPurchaseOrderHandler(app),
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
			Handler: createCancelPurchaseOrderHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the submit expense route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/expenses/:id/submit",
			Handler: createSubmitRecordHandler(app, "expenses"),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the recall expense route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/expenses/:id/recall",
			Handler: createRecallRecordHandler(app, "expenses"),
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
			Handler: createApproveRecordHandler(app, "expenses"),
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
			Handler: createRejectRecordHandler(app, "expenses"),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the commit expense route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/expenses/:id/commit",
			Handler: createCommitRecordHandler(app, "expenses"),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the close purchase order route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/purchase_orders/:id/close",
			Handler: createClosePurchaseOrderHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})

	// Add the convert purchase order route
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method:  http.MethodPost,
			Path:    "/api/purchase_orders/:id/make_cumulative",
			Handler: createConvertToCumulativePurchaseOrderHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})
}
