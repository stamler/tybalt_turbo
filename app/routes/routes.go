package routes

import (
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
type TimeSheetIdRequest struct {
	TimeSheetId string `json:"timeSheetId"`
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
			Path:    "/api/reject-timesheet",
			Handler: createRejectTimesheetHandler(app),
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})
}
