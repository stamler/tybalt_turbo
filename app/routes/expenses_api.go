package routes

import (
	_ "embed"
	"net/http"
	"strconv"

	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed expenses_select_base.sql
var expensesSelectBaseQuery string

//go:embed expense_details.sql
var expenseDetailsQuery string

// Predefined WHERE clauses to minimize inline query building
const (
	whereListMine     = "e.uid = {:auth}"
	whereListMineByPO = "e.uid = {:auth} AND e.purchase_order = {:purchase_order}"
	wherePending      = "e.approver = {:auth} AND e.submitted = 1 AND (e.approved = '' OR e.approved IS NULL)"
	whereApproved     = "e.approver = {:auth} AND (e.approved != '' AND e.approved IS NOT NULL)"
)

func buildCountQuery(whereClause string) string {
	return "SELECT COUNT(*) AS total FROM expenses e WHERE " + whereClause
}

func buildListQuery(whereClause string) string {
	return expensesSelectBaseQuery + "\nWHERE " + whereClause + "\nORDER BY e.date DESC\nLIMIT {:limit} OFFSET {:offset}"
}

// ExpensesAugmentedRow models the augmented expense row returned by SQL.
type ExpensesAugmentedRow struct {
	ID                  string  `db:"id" json:"id"`
	UID                 string  `db:"uid" json:"uid"`
	Date                string  `db:"date" json:"date"`
	Division            string  `db:"division" json:"division"`
	Description         string  `db:"description" json:"description"`
	Total               float64 `db:"total" json:"total"`
	PaymentType         string  `db:"payment_type" json:"payment_type"`
	Attachment          string  `db:"attachment" json:"attachment"`
	AttachmentHash      string  `db:"attachment_hash" json:"attachment_hash"`
	Rejector            string  `db:"rejector" json:"rejector"`
	Rejected            string  `db:"rejected" json:"rejected"`
	RejectionReason     string  `db:"rejection_reason" json:"rejection_reason"`
	Approver            string  `db:"approver" json:"approver"`
	Approved            string  `db:"approved" json:"approved"`
	Job                 string  `db:"job" json:"job"`
	Category            string  `db:"category" json:"category"`
	Kind                string  `db:"kind" json:"kind"`
	PayPeriodEnding     string  `db:"pay_period_ending" json:"pay_period_ending"`
	AllowanceTypes      string  `db:"allowance_types" json:"allowance_types"`
	Submitted           bool    `db:"submitted" json:"submitted"`
	Committer           string  `db:"committer" json:"committer"`
	Committed           string  `db:"committed" json:"committed"`
	CommittedWeekEnding string  `db:"committed_week_ending" json:"committed_week_ending"`
	Distance            float64 `db:"distance" json:"distance"`
	CCLast4Digits       string  `db:"cc_last_4_digits" json:"cc_last_4_digits"`
	PurchaseOrder       string  `db:"purchase_order" json:"purchase_order"`
	Vendor              string  `db:"vendor" json:"vendor"`
	PurchaseOrderNumber string  `db:"purchase_order_number" json:"purchase_order_number"`
	ClientName          string  `db:"client_name" json:"client_name"`
	CategoryName        string  `db:"category_name" json:"category_name"`
	KindName            string  `db:"kind_name" json:"kind_name"`
	JobNumber           string  `db:"job_number" json:"job_number"`
	JobDescription      string  `db:"job_description" json:"job_description"`
	DivisionName        string  `db:"division_name" json:"division_name"`
	DivisionCode        string  `db:"division_code" json:"division_code"`
	VendorName          string  `db:"vendor_name" json:"vendor_name"`
	VendorAlias         string  `db:"vendor_alias" json:"vendor_alias"`
	UIDName             string  `db:"uid_name" json:"uid_name"`
	ApproverName        string  `db:"approver_name" json:"approver_name"`
	RejectorName        string  `db:"rejector_name" json:"rejector_name"`
	BranchName          string  `db:"branch_name" json:"branch_name"`
}

// ExpenseDetailsRow extends ExpensesAugmentedRow with PO comparison fields
// used only by the expense details endpoint.
type ExpenseDetailsRow struct {
	ExpensesAugmentedRow
	POVendor       string `db:"po_vendor" json:"po_vendor"`
	POVendorName   string `db:"po_vendor_name" json:"po_vendor_name"`
	POVendorAlias  string `db:"po_vendor_alias" json:"po_vendor_alias"`
	POJob          string `db:"po_job" json:"po_job"`
	POJobNumber    string `db:"po_job_number" json:"po_job_number"`
	POJobDesc      string `db:"po_job_description" json:"po_job_description"`
	PODivision     string `db:"po_division" json:"po_division"`
	PODivisionCode string `db:"po_division_code" json:"po_division_code"`
	PODivisionName string `db:"po_division_name" json:"po_division_name"`
	POCategory     string `db:"po_category" json:"po_category"`
	POCategoryName string `db:"po_category_name" json:"po_category_name"`
	PODescription  string `db:"po_description" json:"po_description"`
	POPaymentType  string `db:"po_payment_type" json:"po_payment_type"`
	POBranch       string `db:"po_branch" json:"po_branch"`
	POBranchName   string `db:"po_branch_name" json:"po_branch_name"`
	POKind         string `db:"po_kind" json:"po_kind"`
	POKindName     string  `db:"po_kind_name" json:"po_kind_name"`
	POTotal        float64 `db:"po_total" json:"po_total"`
}

type PaginatedExpensesResponse struct {
	Data       []ExpensesAugmentedRow `json:"data"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	Total      int                    `json:"total"`
	TotalPages int                    `json:"total_pages"`
}

// Default page size used by expenses list endpoints unless overridden via query param
const defaultPageLimit = 20

// createGetExpensesListHandler returns the caller's own expenses, paginated,
// with optional purchase_order filter.
func createGetExpensesListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}
		authID := auth.Id

		q := e.Request.URL.Query()
		page := 1
		if s := q.Get("page"); s != "" {
			if p, err := strconv.Atoi(s); err == nil && p > 0 {
				page = p
			}
		}
		limit := defaultPageLimit
		if s := q.Get("limit"); s != "" {
			if l, err := strconv.Atoi(s); err == nil && l > 0 {
				if l > 200 {
					l = 200
				}
				limit = l
			}
		}
		offset := (page - 1) * limit

		purchaseOrder := q.Get("purchase_order")

		params := dbx.Params{
			"auth":           authID,
			"purchase_order": purchaseOrder,
			"limit":          limit,
			"offset":         offset,
		}

		// Choose WHERE based on whether a purchase_order filter is supplied
		whereClause := whereListMine
		if purchaseOrder != "" {
			whereClause = whereListMineByPO
		}

		// total count (no joins needed)
		countQuery := buildCountQuery(whereClause)
		var total int
		if err := app.DB().NewQuery(countQuery).Bind(params).Row(&total); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count expenses: "+err.Error(), err)
		}

		// rows
		listQuery := buildListQuery(whereClause)
		var rows []ExpensesAugmentedRow
		if err := app.DB().NewQuery(listQuery).Bind(params).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load expenses: "+err.Error(), err)
		}

		resp := PaginatedExpensesResponse{
			Data:       rows,
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		}
		return e.JSON(http.StatusOK, resp)
	}
}

// createGetExpenseDetailsHandler returns a single augmented expense row if the
// caller is authorized to view it according to the allowed predicate.
func createGetExpenseDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}
		authID := auth.Id

		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		hasCommit, err := utilities.HasClaim(app, e.Auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking commit claim", err)
		}
		hasReport, err := utilities.HasClaim(app, e.Auth, "report")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking report claim", err)
		}

		// Build predicate parameters
		params := dbx.Params{
			"id":         id,
			"auth":       authID,
			"has_commit": boolToInt(hasCommit),
			"has_report": boolToInt(hasReport),
		}

		// Append allowed predicate to the details query
		query := expenseDetailsQuery + `
AND (
  e.uid = {:auth}
  OR (e.approver = {:auth} AND e.submitted = 1)
  OR (({:has_commit} = 1) AND e.approved != '')
  OR (({:has_report} = 1) AND e.committed != '')
)`

		var row ExpenseDetailsRow
		if err := app.DB().NewQuery(query).Bind(params).One(&row); err != nil {
			return e.Error(http.StatusNotFound, "expense not found or not authorized", err)
		}

		return e.JSON(http.StatusOK, row)
	}
}

// boolToInt converts a bool to 0/1 for SQL parameterization.
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// createGetPendingExpensesHandler returns submitted-but-not-approved expenses for which
// the caller is the approver.
func createGetPendingExpensesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		q := e.Request.URL.Query()
		page := 1
		if s := q.Get("page"); s != "" {
			if p, err := strconv.Atoi(s); err == nil && p > 0 {
				page = p
			}
		}
		limit := defaultPageLimit
		if s := q.Get("limit"); s != "" {
			if l, err := strconv.Atoi(s); err == nil && l > 0 {
				if l > 200 {
					l = 200
				}
				limit = l
			}
		}
		offset := (page - 1) * limit

		params := dbx.Params{
			"auth":   auth.Id,
			"limit":  limit,
			"offset": offset,
		}

		countQuery := buildCountQuery(wherePending)
		var total int
		if err := app.DB().NewQuery(countQuery).Bind(params).Row(&total); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count pending expenses: "+err.Error(), err)
		}

		listQuery := buildListQuery(wherePending)
		var rows []ExpensesAugmentedRow
		if err := app.DB().NewQuery(listQuery).Bind(params).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load pending expenses: "+err.Error(), err)
		}

		resp := PaginatedExpensesResponse{
			Data:       rows,
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		}
		return e.JSON(http.StatusOK, resp)
	}
}

// createGetApprovedExpensesHandler returns the expenses approved by the caller.
func createGetApprovedExpensesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		q := e.Request.URL.Query()
		page := 1
		if s := q.Get("page"); s != "" {
			if p, err := strconv.Atoi(s); err == nil && p > 0 {
				page = p
			}
		}
		limit := defaultPageLimit
		if s := q.Get("limit"); s != "" {
			if l, err := strconv.Atoi(s); err == nil && l > 0 {
				if l > 200 {
					l = 200
				}
				limit = l
			}
		}
		offset := (page - 1) * limit

		params := dbx.Params{
			"auth":   auth.Id,
			"limit":  limit,
			"offset": offset,
		}

		countQuery := buildCountQuery(whereApproved)
		var total int
		if err := app.DB().NewQuery(countQuery).Bind(params).Row(&total); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count approved expenses: "+err.Error(), err)
		}

		listQuery := buildListQuery(whereApproved)
		var rows []ExpensesAugmentedRow
		if err := app.DB().NewQuery(listQuery).Bind(params).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load approved expenses: "+err.Error(), err)
		}

		resp := PaginatedExpensesResponse{
			Data:       rows,
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: (total + limit - 1) / limit,
		}
		return e.JSON(http.StatusOK, resp)
	}
}
