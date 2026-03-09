package routes

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type userDefaultsResponse struct {
	DefaultDivision            string `json:"default_division" db:"default_division"`
	DefaultRole                string `json:"default_role" db:"default_role"`
	DefaultBranch              string `json:"default_branch" db:"default_branch"`
	AllowPersonalReimbursement bool   `json:"allow_personal_reimbursement" db:"allow_personal_reimbursement"`
}

func createGetUserDefaultsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		userID := e.Auth.Id

		result := userDefaultsResponse{}
		err := app.DB().NewQuery(`
			SELECT
				COALESCE(p.default_division, '') AS default_division,
				COALESCE(p.default_role, '') AS default_role,
				COALESCE(ap.default_branch, '') AS default_branch,
				COALESCE(ap.allow_personal_reimbursement, 0) AS allow_personal_reimbursement
			FROM users u
			LEFT JOIN profiles p ON p.uid = u.id
			LEFT JOIN admin_profiles ap ON ap.uid = u.id
			WHERE u.id = {:uid}
			LIMIT 1
		`).Bind(dbx.Params{
			"uid": userID,
		}).One(&result)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load user defaults", err)
		}

		return e.JSON(http.StatusOK, result)
	}
}
