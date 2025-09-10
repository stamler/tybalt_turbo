package routes

import (
	_ "embed" // Needed for //go:embed
	"net/http"
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed job_expenses.sql
var jobExpensesQuery string

//go:embed job_expenses_count.sql
var jobExpensesCountQuery string

// JobExpenseEntry models a single expense row returned by job_expenses.sql.
type JobExpenseEntry struct {
	Description         string  `db:"description" json:"description"`
	Total               float64 `db:"total" json:"total"`
	ID                  string  `db:"id" json:"id"`
	Date                string  `db:"date" json:"date"`
	CommittedWeekEnding string  `db:"committed_week_ending" json:"committed_week_ending"`
	BranchCode          string  `db:"branch_code" json:"branch_code"`
	DivisionCode        string  `db:"division_code" json:"division_code"`
	PaymentType         string  `db:"payment_type" json:"payment_type"`
	Surname             string  `db:"surname" json:"surname"`
	GivenName           string  `db:"given_name" json:"given_name"`
	CategoryName        string  `db:"category_name" json:"category_name"`
}

// PaginatedJobExpensesResponse represents the paginated response structure
// for expenses.
type PaginatedJobExpensesResponse struct {
	Data       []JobExpenseEntry `json:"data"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	Total      int               `json:"total"`
	TotalPages int               `json:"total_pages"`
}

// createGetJobExpensesHandler returns an HTTP handler that fetches the expenses
// for a job with optional filters and pagination. Supported query parameters:
//   - division      (division id)
//   - payment_type  (payment_type string)
//   - uid           (user id)
//   - category      (category id)
//   - page          (page number, default: 1)
//   - limit         (page size, default: 50, max: 200)
func createGetJobExpensesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		q := e.Request.URL.Query()
		division := q.Get("division")
		branch := q.Get("branch")
		paymentType := q.Get("payment_type")
		uid := q.Get("uid")
		category := q.Get("category")

		// Parse pagination parameters
		page := 1
		if pageStr := q.Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		limit := 50 // default page size
		if limitStr := q.Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				if l > 200 {
					l = 200 // max page size
				}
				limit = l
			}
		}

		offset := (page - 1) * limit

		params := dbx.Params{
			"id":           id,
			"branch":       branch,
			"division":     division,
			"payment_type": paymentType,
			"uid":          uid,
			"category":     category,
			"limit":        limit,
			"offset":       offset,
		}

		// Get total count for pagination metadata
		var totalCount int
		err := app.DB().NewQuery(jobExpensesCountQuery).Bind(params).Row(&totalCount)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute count query: "+err.Error(), err)
		}

		// Get paginated results
		var rows []JobExpenseEntry
		err = app.DB().NewQuery(jobExpensesQuery).Bind(params).All(&rows)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query: "+err.Error(), err)
		}

		totalPages := (totalCount + limit - 1) / limit // ceiling division

		response := PaginatedJobExpensesResponse{
			Data:       rows,
			Page:       page,
			Limit:      limit,
			Total:      totalCount,
			TotalPages: totalPages,
		}

		return e.JSON(http.StatusOK, response)
	}
}
