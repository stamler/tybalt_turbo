package routes

import (
	_ "embed"
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_pos.sql
var jobPOsQuery string

//go:embed job_pos_count.sql
var jobPOsCountQuery string

// JobPOEntry represents a purchase order row
// Fields align with SQL aliases.
type JobPOEntry struct {
	ID           string  `db:"id" json:"id"`
	PONumber     string  `db:"po_number" json:"po_number"`
	Date         string  `db:"date" json:"date"`
	Total        float64 `db:"total" json:"total"`
	Type         string  `db:"type" json:"type"`
	BranchCode   string  `db:"branch_code" json:"branch_code"`
	DivisionCode string  `db:"division_code" json:"division_code"`
	Surname      string  `db:"surname" json:"surname"`
	GivenName    string  `db:"given_name" json:"given_name"`
}

// Paginated response
type PaginatedJobPOsResponse struct {
	Data       []JobPOEntry `json:"data"`
	Page       int          `json:"page"`
	Limit      int          `json:"limit"`
	Total      int          `json:"total"`
	TotalPages int          `json:"total_pages"`
}

// createGetJobPOsHandler returns list handler for active POs with filters:
//   - division (division id)
//   - type     (purchase order type)
//   - uid      (user id)
//   - page, limit same as others
func createGetJobPOsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		division := q.Get("division")
		branch := q.Get("branch")
		poType := q.Get("type")
		uid := q.Get("uid")

		page := 1
		if pStr := q.Get("page"); pStr != "" {
			if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
				page = p
			}
		}
		limit := 50
		if lStr := q.Get("limit"); lStr != "" {
			if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
				if l > 200 {
					l = 200
				}
				limit = l
			}
		}
		offset := (page - 1) * limit

		params := dbx.Params{
			"id":       id,
			"branch":   branch,
			"division": division,
			"type":     poType,
			"uid":      uid,
			"limit":    limit,
			"offset":   offset,
		}

		var totalCount int
		if err := app.DB().NewQuery(jobPOsCountQuery).Bind(params).Row(&totalCount); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute count query: "+err.Error(), err)
		}

		var rows []JobPOEntry
		if err := app.DB().NewQuery(jobPOsQuery).Bind(params).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		totalPages := (totalCount + limit - 1) / limit

		return e.JSON(http.StatusOK, PaginatedJobPOsResponse{
			Data:       rows,
			Page:       page,
			Limit:      limit,
			Total:      totalCount,
			TotalPages: totalPages,
		})
	}
}
