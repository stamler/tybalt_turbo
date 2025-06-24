package routes

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const timesheetReviewersQuery = `
SELECT 
    tsr.id,
    tsr.time_sheet,
    tsr.reviewer,
    tsr.reviewed,
    p.surname,
    p.given_name
FROM time_sheet_reviewers tsr
INNER JOIN time_sheets ts ON ts.id = tsr.time_sheet
LEFT JOIN profiles p ON p.uid = tsr.reviewer
WHERE tsr.time_sheet = {:timesheetId}
  -- since we get the userId from the auth record, we can safely use it to
	-- filter the records
  AND (
    ts.uid = {:userId} OR 
    ts.approver = {:userId} OR 
    tsr.reviewer = {:userId}
  )
ORDER BY tsr.reviewed DESC
`

type TimesheetReviewer struct {
	Id          string `json:"id" db:"id"`
	TimeSheetId string `json:"time_sheet" db:"time_sheet"`
	ReviewerId  string `json:"reviewer" db:"reviewer"`
	Reviewed    string `json:"reviewed" db:"reviewed"`
	Surname     string `json:"surname" db:"surname"`
	GivenName   string `json:"given_name" db:"given_name"`
}

// TimesheetReviewSummary wraps reviewers along with owner name and week ending
type TimesheetReviewSummary struct {
	OwnerName  string              `json:"owner_name"`
	WeekEnding string              `json:"week_ending"`
	Reviewers  []TimesheetReviewer `json:"reviewers"`
}

// createGetReviewersHandler returns a function that gets reviewers for a given time_sheets record
func createGetReviewersHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		userId := authRecord.Id
		timesheetId := e.Request.PathValue("id")

		// Execute the query with authorization built into the SQL
		var reviewers []TimesheetReviewer
		err := app.DB().NewQuery(timesheetReviewersQuery).Bind(dbx.Params{
			"timesheetId": timesheetId,
			"userId":      userId,
		}).All(&reviewers)

		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to fetch reviewers: " + err.Error(),
			})
		}

		// 2. Fetch owner name and week ending
		const metaQuery = `
SELECT
    COALESCE(p.given_name || ' ' || p.surname, u.name) AS owner_name,
    ts.week_ending                                     AS week_ending
FROM time_sheets ts
LEFT JOIN profiles p ON p.uid = ts.uid
LEFT JOIN users    u ON u.id  = ts.uid
WHERE ts.id = {:timesheetId}
LIMIT 1`

		type timesheetMeta struct {
			OwnerName  string `db:"owner_name"`
			WeekEnding string `db:"week_ending"`
		}

		var meta timesheetMeta
		if err := app.DB().NewQuery(metaQuery).
			Bind(dbx.Params{"timesheetId": timesheetId}).
			One(&meta); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to fetch timesheet metadata: " + err.Error(),
			})
		}

		// 3. Assemble summary and return
		summary := TimesheetReviewSummary{
			OwnerName:  meta.OwnerName,
			WeekEnding: meta.WeekEnding,
			Reviewers:  reviewers,
		}

		return e.JSON(http.StatusOK, summary)
	}
}
