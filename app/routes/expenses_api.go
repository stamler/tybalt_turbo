package routes

import (
	_ "embed"
	"fmt"
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

// Predefined WHERE clauses to minimize inline query building.
//
// Important policy note:
//   - These clauses model EXPENSE visibility, not PURCHASE ORDER visibility.
//   - That distinction matters because some callers can legitimately view a PO
//     details page without being allowed to view every linked expense.
//   - In particular, PO creator / PO approver / PO second approver visibility is
//     broader in some cases than expense visibility, while expense owners can in
//     some cases still view their own expense even if they no longer qualify to
//     view the linked PO (for example after the PO reaches a terminal state and
//     they are not a PO participant).
//
// The shared expense visibility predicate below currently allows:
// - expense owner: always
// - expense approver: while the expense is submitted
// - commit claim holder: once the expense is approved
// - report claim holder: once the expense is committed
//
// It intentionally does NOT automatically grant visibility based solely on PO
// participation. That means a caller may be able to open a PO details page and
// still see a filtered subset of linked expenses.
const (
	whereListMine     = "e.uid = {:auth}"
	whereListMineByPO = "e.uid = {:auth} AND e.purchase_order = {:purchase_order}"
	wherePending      = "e.approver = {:auth} AND e.submitted = 1 AND (e.approved = '' OR e.approved IS NULL)"
	whereApproved     = "e.approver = {:auth} AND (e.approved != '' AND e.approved IS NOT NULL)"
	// expenseVisibilityPredicate is the canonical EXPENSE-level visibility rule
	// used by expense details and by the PO-related-expenses endpoint.
	//
	// Keep this separate from PO visibility. A visible PO does not imply that all
	// linked expenses are visible, and widening this predicate changes more than
	// just the PO details page.
	//
	// Consequences of the current rule:
	// - report and commit holders can inspect relevant linked expenses from a PO
	//   details page, fixing the old "Expenses(0)" bug caused by reusing the
	//   "my expenses" endpoint.
	// - PO owners, PO approvers, priority second approvers, and second approvers
	//   are NOT granted access here unless they also satisfy one of the
	//   expense-specific rules below.
	// - this means PO aggregate fields such as committed_expenses_count may
	//   legitimately reflect more linked expenses than this predicate returns for
	//   a given caller.
	expenseVisibilityPredicate = `
  e.uid = {:auth}
  OR (e.approver = {:auth} AND e.submitted = 1)
  OR (({:has_commit} = 1) AND e.approved != '')
  OR (({:has_report} = 1) AND e.committed != '')
`
)

func buildCountQuery(whereClause string) string {
	return "SELECT COUNT(*) AS total FROM expenses e WHERE " + whereClause
}

func buildListQuery(whereClause string) string {
	return expensesSelectBaseQuery + "\nWHERE " + whereClause + "\nORDER BY e.date DESC\nLIMIT {:limit} OFFSET {:offset}"
}

func buildOrderedExpensesQuery(whereClause string, orderBy string) string {
	return expensesSelectBaseQuery + "\nWHERE " + whereClause + "\nORDER BY " + orderBy
}

func buildPaginatedOrderedExpensesQuery(whereClause string, orderBy string) string {
	return buildOrderedExpensesQuery(whereClause, orderBy) + "\nLIMIT {:limit} OFFSET {:offset}"
}

// expenseVisibilityParams resolves the claim-dependent inputs needed by the
// shared expense visibility predicate.
//
// This helper intentionally knows nothing about PO participation. If future
// policy needs "PO viewer can also see linked expenses", do not silently change
// this helper without also re-evaluating expense details visibility, because
// this helper feeds both the expense details endpoint and the PO-related-
// expenses endpoint.
func expenseVisibilityParams(app core.App, auth *core.Record) (dbx.Params, error) {
	hasCommit, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return nil, err
	}
	hasReport, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return nil, err
	}

	return dbx.Params{
		"auth":       auth.Id,
		"has_commit": boolToInt(hasCommit),
		"has_report": boolToInt(hasReport),
	}, nil
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
	Currency            string  `db:"currency" json:"currency"`
	CurrencyCode        string  `db:"currency_code" json:"currency_code"`
	CurrencySymbol      string  `db:"currency_symbol" json:"currency_symbol"`
	CurrencyIcon        string  `db:"currency_icon" json:"currency_icon"`
	CurrencyRate        float64 `db:"currency_rate" json:"currency_rate"`
	CurrencyRateDate    string  `db:"currency_rate_date" json:"currency_rate_date"`
	SettledTotal        float64 `db:"settled_total" json:"settled_total"`
	Settler             string  `db:"settler" json:"settler"`
	Settled             string  `db:"settled" json:"settled"`
	SettlerName         string  `db:"settler_name" json:"settler_name"`
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
	POVendor           string  `db:"po_vendor" json:"po_vendor"`
	POVendorName       string  `db:"po_vendor_name" json:"po_vendor_name"`
	POVendorAlias      string  `db:"po_vendor_alias" json:"po_vendor_alias"`
	POJob              string  `db:"po_job" json:"po_job"`
	POJobNumber        string  `db:"po_job_number" json:"po_job_number"`
	POJobDesc          string  `db:"po_job_description" json:"po_job_description"`
	PODivision         string  `db:"po_division" json:"po_division"`
	PODivisionCode     string  `db:"po_division_code" json:"po_division_code"`
	PODivisionName     string  `db:"po_division_name" json:"po_division_name"`
	POCategory         string  `db:"po_category" json:"po_category"`
	POCategoryName     string  `db:"po_category_name" json:"po_category_name"`
	PODescription      string  `db:"po_description" json:"po_description"`
	POPaymentType      string  `db:"po_payment_type" json:"po_payment_type"`
	POBranch           string  `db:"po_branch" json:"po_branch"`
	POBranchName       string  `db:"po_branch_name" json:"po_branch_name"`
	POKind             string  `db:"po_kind" json:"po_kind"`
	POKindName         string  `db:"po_kind_name" json:"po_kind_name"`
	POTotal            float64 `db:"po_total" json:"po_total"`
	POCurrency         string  `db:"po_currency" json:"po_currency"`
	POCurrencyCode     string  `db:"po_currency_code" json:"po_currency_code"`
	POCurrencySymbol   string  `db:"po_currency_symbol" json:"po_currency_symbol"`
	POCurrencyIcon     string  `db:"po_currency_icon" json:"po_currency_icon"`
	POCurrencyRate     float64 `db:"po_currency_rate" json:"po_currency_rate"`
	POCurrencyRateDate string  `db:"po_currency_rate_date" json:"po_currency_rate_date"`
	POUID              string  `db:"po_uid" json:"po_uid"`
	POUIDName          string  `db:"po_uid_name" json:"po_uid_name"`
	POOwnerUIDMismatch bool    `json:"po_owner_uid_mismatch"`
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

// expenses/list shows all non-committed rows on the first page, then paginates
// committed overflow in fixed-size batches behind "Load More".
const expensesListCommittedOverflowLimit = 50

// createGetExpensesListHandler returns the caller's own expenses, paginated,
// with optional purchase_order filter.
//
// Special pagination contract for /api/expenses/list:
//   - caller-supplied ?limit= is intentionally ignored for this route
//   - the response Limit field means "committed overflow batch size"
//   - page 1 may therefore return more than Limit rows because it includes all
//     non-committed expenses plus up to Limit committed expenses
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
		purchaseOrder := q.Get("purchase_order")

		params := dbx.Params{
			"auth":           authID,
			"purchase_order": purchaseOrder,
		}

		// Choose WHERE based on whether a purchase_order filter is supplied
		whereClause := whereListMine
		if purchaseOrder != "" {
			whereClause = whereListMineByPO
		}
		nonCommittedWhereClause := whereClause + " AND COALESCE(e.committed, '') = ''"
		committedWhereClause := whereClause + " AND COALESCE(e.committed, '') != ''"

		// total count (no joins needed)
		countQuery := buildCountQuery(whereClause)
		var total int
		if err := app.DB().NewQuery(countQuery).Bind(params).Row(&total); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count expenses: "+err.Error(), err)
		}

		var committedCount int
		if err := app.DB().NewQuery(buildCountQuery(committedWhereClause)).Bind(params).Row(&committedCount); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to count committed expenses: "+err.Error(), err)
		}

		var rows []ExpensesAugmentedRow
		if page == 1 {
			nonCommittedQuery := buildOrderedExpensesQuery(nonCommittedWhereClause, "e.date DESC, e.created DESC")
			if err := app.DB().NewQuery(nonCommittedQuery).Bind(params).All(&rows); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to load non-committed expenses: "+err.Error(), err)
			}

			params["limit"] = expensesListCommittedOverflowLimit
			params["offset"] = 0

			var committedRows []ExpensesAugmentedRow
			committedQuery := buildPaginatedOrderedExpensesQuery(committedWhereClause, "e.committed DESC, e.date DESC, e.created DESC")
			if err := app.DB().NewQuery(committedQuery).Bind(params).All(&committedRows); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to load committed expenses: "+err.Error(), err)
			}

			rows = append(rows, committedRows...)
		} else {
			params["limit"] = expensesListCommittedOverflowLimit
			params["offset"] = expensesListCommittedOverflowLimit + (page-2)*expensesListCommittedOverflowLimit

			committedQuery := buildPaginatedOrderedExpensesQuery(committedWhereClause, "e.committed DESC, e.date DESC, e.created DESC")
			if err := app.DB().NewQuery(committedQuery).Bind(params).All(&rows); err != nil {
				return e.Error(http.StatusInternalServerError, "failed to load committed expenses: "+err.Error(), err)
			}
		}

		totalPages := 1
		if committedCount > expensesListCommittedOverflowLimit {
			overflowCommittedCount := committedCount - expensesListCommittedOverflowLimit
			totalPages += (overflowCommittedCount + expensesListCommittedOverflowLimit - 1) / expensesListCommittedOverflowLimit
		}

		resp := PaginatedExpensesResponse{
			Data: rows,
			Page: page,
			// Limit is the committed overflow batch size for this mixed first-page
			// contract, not a guarantee about the number of rows returned on page 1.
			Limit:      expensesListCommittedOverflowLimit,
			Total:      total,
			TotalPages: totalPages,
		}
		return e.JSON(http.StatusOK, resp)
	}
}

// createGetPurchaseOrderExpensesHandler returns related expenses for a single
// purchase order.
//
// This endpoint exists because the old PO details page reused
// /api/expenses/list?purchase_order=..., which is a "my expenses" endpoint and
// therefore incorrectly hid other users' expenses from report/commit holders.
//
// Current policy for this route:
//   - backend first proves the PO itself is visible through the PO visibility
//     layer; the frontend also reaches this route only from that page
//   - returned expenses are then filtered again using EXPENSE visibility
//   - therefore this route can legitimately return fewer rows than PO-level
//     aggregates such as committed_expenses_count suggest
//
// This is intentional as of now. We are fixing a concrete bug for report and
// commit holders without broadening access for all PO participants.
//
// Known discrepancy to preserve/document:
//   - PO creator / PO approver / PO second approver may be able to view a PO but
//     not every linked expense unless they also satisfy expense visibility
//   - expense owner may be able to view their expense even when they cannot view
//     the linked PO under terminal-state PO visibility rules
//
// If policy later changes, this route is the place to decide whether PO
// participant visibility should be unioned with expense visibility.
func createGetPurchaseOrderExpensesHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		purchaseOrderID := e.Request.PathValue("id")
		if purchaseOrderID == "" {
			return e.Error(http.StatusBadRequest, "purchase order id is required", nil)
		}

		// Enforce the same PO visibility contract as /api/purchase_orders/visible/:id
		// so this endpoint cannot be used as a backdoor PO-id keyed expense
		// enumerator for callers who satisfy expense visibility but not PO
		// visibility.
		visiblePO, err := findVisiblePurchaseOrderByID(app, auth.Id, purchaseOrderID)
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_fetching_visible_po",
				"message": fmt.Sprintf("error fetching visible purchase order: %v", err),
			})
		}
		if visiblePO == nil {
			return e.JSON(http.StatusNotFound, map[string]string{
				"code":    "po_not_found_or_not_visible",
				"message": "purchase order not found or not visible",
			})
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

		params, err := expenseVisibilityParams(app, auth)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking expense visibility claims", err)
		}
		params["purchase_order"] = purchaseOrderID
		params["limit"] = limit
		params["offset"] = offset

		// Note that the route does not simply mean "all expenses under a visible
		// PO". The PO id narrows the set, but expenseVisibilityPredicate still
		// controls which of those rows the caller may inspect.
		whereClause := "e.purchase_order = {:purchase_order} AND (\n" + expenseVisibilityPredicate + "\n)"

		countQuery := buildCountQuery(whereClause)
		var total int
		if err := app.DB().NewQuery(countQuery).Bind(params).Row(&total); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_counting_purchase_order_expenses",
				"message": fmt.Sprintf("failed to count purchase order expenses: %v", err),
			})
		}

		listQuery := expensesSelectBaseQuery + "\nWHERE " + whereClause + "\nORDER BY e.committed DESC, e.date DESC, e.created DESC\nLIMIT {:limit} OFFSET {:offset}"
		var rows []ExpensesAugmentedRow
		if err := app.DB().NewQuery(listQuery).Bind(params).All(&rows); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{
				"code":    "error_loading_purchase_order_expenses",
				"message": fmt.Sprintf("failed to load purchase order expenses: %v", err),
			})
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
// caller is authorized to view it according to the allowed expense predicate.
//
// This endpoint is intentionally expense-centric. It does not inherit PO
// visibility. Keep that in mind when comparing behavior against the PO details
// page or PO search results: the surrounding PO may be visible to a somewhat
// different set of users than the expense itself.
func createGetExpenseDetailsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		if auth == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		id := e.Request.PathValue("id")
		if id == "" {
			return e.Error(http.StatusBadRequest, "id is required", nil)
		}

		params, err := expenseVisibilityParams(app, auth)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking expense visibility claims", err)
		}
		hasAdmin, err := utilities.HasClaim(app, auth, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking expense visibility claims", err)
		}
		params["id"] = id
		params["has_admin"] = boolToInt(hasAdmin)

		query := expenseDetailsQuery + "\nAND (\n" + expenseVisibilityPredicate + "\nOR ({:has_admin} = 1 AND e.committed != '')\n)"

		var row ExpenseDetailsRow
		if err := app.DB().NewQuery(query).Bind(params).One(&row); err != nil {
			return e.Error(http.StatusNotFound, "expense not found or not authorized", err)
		}
		row.POOwnerUIDMismatch = expensePurchaseOrderOwnerUIDMismatch(row.UID, row.POUID)

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
