package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
)

// Add custom routes to the app
type WeekEndingRequest struct {
	WeekEnding string `json:"weekEnding"`
}

func AddRoutes(app *pocketbase.PocketBase) {

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		e.Router.AddRoute(echo.Route{
			Method: http.MethodPost,
			Path:   "/api/bundle-timesheet",
			Handler: func(c echo.Context) error {
				var req WeekEndingRequest
				if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
				}

				// Validate the date format (YYYY-MM-DD)
				if _, err := time.Parse("2006-01-02", req.WeekEnding); err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid date format. Use YYYY-MM-DD"})
				}

				/*
					Should time_sheets be tied to profiles or to users?

					This function will throw an error if a time_sheets record already
					exists for the user and week ending date. If a timesheet does not
					exist, it will validate information across all of the user's time
					entries records for the week ending date. If the information is valid,
					it will then create a new time_sheets record for the user with the
					given week ending date then write the timesheet id to every time
					entries record that has the same user id and week ending date.
				*/

				// Start a transaction
				err := app.Dao().RunInTransaction(func(txDao *daos.Dao) error {

					/*
						The time_sheets table has the following columns:
						- id (string, primary key)
						- uid (string, foreign key to users table) should this reference profile instead?
						- manager_id (string, foreign key to profiles table)
						- work_week_hours (number representing a full week's worth of hours for the user at the time)
						- salary (boolean representing whether the user was on salary at the time)
						- week_ending (string representing the week ending date in format YYYY-MM-DD)
					*/

					// TODO: Implement your SQLite transaction logic here
					// For example:
					// _, err := txDao.DB().NewQuery("YOUR RAW SQL QUERY HERE").Execute()
					// if err != nil {
					//     return err
					// }

					return nil // Return nil if transaction is successful
				})

				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Transaction failed"})
				}

				return c.JSON(http.StatusOK, map[string]string{"message": "Transaction successful"})
			},
			Middlewares: []echo.MiddlewareFunc{
				apis.RequireRecordAuth("users"),
			},
		})

		return nil
	})
}
