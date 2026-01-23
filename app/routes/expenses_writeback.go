package routes

import (
	"net/http"
	"strings"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Wrapper response struct for structured expenses writeback
type expensesWritebackResponse struct {
	Expenses       []expenseExportOutput       `json:"expenses"`
	Vendors        []vendorExportOutput        `json:"vendors"`
	PurchaseOrders []purchaseOrderExportOutput `json:"purchaseOrders"`
}

// Vendor export struct for separate vendors array
type vendorExportOutput struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Alias  string `json:"alias,omitempty"`
	Status string `json:"status,omitempty"`
}

// Purchase order export struct for separate purchaseOrders array
type purchaseOrderExportOutput struct {
	Id                        string  `json:"id"`
	PoNumber                  string  `json:"poNumber"`
	VendorId                  string  `json:"vendorId,omitempty"`
	VendorName                string  `json:"vendorName,omitempty"`
	Uid                       string  `json:"uid,omitempty"` // legacy_uid of creator
	Job                       string  `json:"job,omitempty"` // PocketBase job ID
	Description               string  `json:"description,omitempty"` // PO's own description
	Division                  string  `json:"division,omitempty"` // division code
	DivisionName              string  `json:"divisionName,omitempty"`
	Total                     float64 `json:"total"`
	ApprovalTotal             float64 `json:"approvalTotal"`
	Date                      string  `json:"date,omitempty"`
	EndDate                   string  `json:"endDate,omitempty"` // for recurring POs
	Approved                  string  `json:"approved,omitempty"`    // timestamp or empty
	ApproverUid               string  `json:"approverUid,omitempty"` // legacy_uid (first approver)
	SecondApproval            string  `json:"secondApproval,omitempty"`            // timestamp or empty
	SecondApproverUid         string  `json:"secondApproverUid,omitempty"`         // legacy_uid
	PrioritySecondApproverUid string  `json:"prioritySecondApproverUid,omitempty"` // legacy_uid
	Cancelled                 string  `json:"cancelled,omitempty"`                 // timestamp or empty
	CancellerUid              string  `json:"cancellerUid,omitempty"`              // legacy_uid
	Closed                    string  `json:"closed,omitempty"`                    // timestamp or empty
	CloserUid                 string  `json:"closerUid,omitempty"`                 // legacy_uid
	Rejected                  string  `json:"rejected,omitempty"`                  // timestamp or empty
	RejectorUid               string  `json:"rejectorUid,omitempty"`               // legacy_uid
	RejectionReason           string  `json:"rejectionReason,omitempty"`
	PaymentType               string  `json:"paymentType,omitempty"`
	Type                      string  `json:"type,omitempty"`
	Frequency                 string  `json:"frequency,omitempty"`
	Status                    string  `json:"status,omitempty"`
	Category                  string  `json:"category,omitempty"`  // PocketBase category ID
	ParentPo                  string  `json:"parentPo,omitempty"`  // PocketBase parent PO ID
	Branch                    string  `json:"branch,omitempty"`    // PocketBase branch ID
	Attachment                string  `json:"attachment,omitempty"`
	AttachmentHash            string  `json:"attachmentHash,omitempty"`
}

// Internal struct for DB scanning - expenses query
type expenseExportDBRow struct {
	Id                  string  `db:"id"`
	Date                string  `db:"date"`
	Description         string  `db:"description"`
	Total               float64 `db:"total"`
	Distance            float64 `db:"distance"`
	PaymentType         string  `db:"payment_type"`
	AllowanceTypesJSON  string  `db:"allowance_types_json"`
	CcLast4Digits       string  `db:"cc_last_4_digits"`
	Attachment          string  `db:"attachment"`
	AttachmentHash      string  `db:"attachment_hash"`
	Submitted           bool    `db:"submitted"`
	Approved            string  `db:"approved"`
	Committed           string  `db:"committed"`
	CommittedWeekEnding string  `db:"committed_week_ending"`
	PayPeriodEnding     string  `db:"pay_period_ending"`
	Rejected            string  `db:"rejected"`
	RejectionReason     string  `db:"rejection_reason"`
	// Denormalized user fields
	Uid         string `db:"uid"` // legacy_uid of expense owner
	Surname     string `db:"surname"`
	GivenName   string `db:"given_name"`
	DisplayName string `db:"display_name"`
	PayrollId   string `db:"payroll_id"`
	// Denormalized manager/approver fields
	ManagerUid  string `db:"manager_uid"` // legacy_uid of approver
	ManagerName string `db:"manager_name"`
	// Denormalized committer fields
	CommitUid  string `db:"commit_uid"` // legacy_uid of committer
	CommitName string `db:"commit_name"`
	// Denormalized rejector fields
	RejectorId   string `db:"rejector_id"` // legacy_uid of rejector
	RejectorName string `db:"rejector_name"`
	// Denormalized division fields
	Division     string `db:"division"` // division code
	DivisionName string `db:"division_name"`
	// Denormalized job fields
	Job            string `db:"job"` // job number
	JobDescription string `db:"job_description"`
	Client         string `db:"client"` // client name
	// Denormalized vendor/PO fields
	VendorName string `db:"vendor_name"`
	Po         string `db:"po"`       // PO number
	Category   string `db:"category"` // category name
	// ID references for separate arrays
	VendorId        string `db:"vendor_id"`
	PurchaseOrderId string `db:"purchase_order_id"`
}

// Internal struct for DB scanning - vendors query
type vendorExportDBRow struct {
	Id     string `db:"id"`
	Name   string `db:"name"`
	Alias  string `db:"alias"`
	Status string `db:"status"`
}

// Internal struct for DB scanning - purchase orders query
type purchaseOrderExportDBRow struct {
	Id                        string  `db:"id"`
	PoNumber                  string  `db:"po_number"`
	VendorId                  string  `db:"vendor_id"`
	VendorName                string  `db:"vendor_name"`
	Uid                       string  `db:"uid"` // legacy_uid
	Job                       string  `db:"job"` // PocketBase job ID
	Description               string  `db:"description"` // PO's own description
	Division                  string  `db:"division"` // division code
	DivisionName              string  `db:"division_name"`
	Total                     float64 `db:"total"`
	ApprovalTotal             float64 `db:"approval_total"`
	Date                      string  `db:"date"`
	EndDate                   string  `db:"end_date"`
	Approved                  string  `db:"approved"`
	ApproverUid               string  `db:"approver_uid"` // legacy_uid (first approver)
	SecondApproval            string  `db:"second_approval"`
	SecondApproverUid         string  `db:"second_approver_uid"` // legacy_uid
	PrioritySecondApproverUid string  `db:"priority_second_approver_uid"` // legacy_uid
	Cancelled                 string  `db:"cancelled"`
	CancellerUid              string  `db:"canceller_uid"` // legacy_uid
	Closed                    string  `db:"closed"`
	CloserUid                 string  `db:"closer_uid"` // legacy_uid
	Rejected                  string  `db:"rejected"`
	RejectorUid               string  `db:"rejector_uid"` // legacy_uid
	RejectionReason           string  `db:"rejection_reason"`
	PaymentType               string  `db:"payment_type"`
	Type                      string  `db:"type"`
	Frequency                 string  `db:"frequency"`
	Status                    string  `db:"status"`
	Category                  string  `db:"category"`  // PocketBase category ID
	ParentPo                  string  `db:"parent_po"` // PocketBase parent PO ID
	Branch                    string  `db:"branch"`    // PocketBase branch ID
	Attachment                string  `db:"attachment"`
	AttachmentHash            string  `db:"attachment_hash"`
}

// Output struct matching legacy Tybalt Expenses Firestore format
type expenseExportOutput struct {
	// The PocketBase ID equals the immutableID from Tybalt (used for fold matching)
	ImmutableID string `json:"immutableID"`
	// User info (denormalized from profile)
	Uid         string `json:"uid"` // legacy_uid
	Surname     string `json:"surname"`
	GivenName   string `json:"givenName"`
	DisplayName string `json:"displayName"`
	PayrollId   string `json:"payrollId"`
	// Core expense data
	Date           string   `json:"date"`
	Description    string   `json:"description,omitempty"`
	Total          *float64 `json:"total,omitempty"`
	Distance       float64  `json:"distance,omitempty"`
	PaymentType    string   `json:"paymentType"`
	CcLast4Digits  string   `json:"ccLast4digits,omitempty"`
	Attachment     string   `json:"attachment,omitempty"`
	AttachmentHash string   `json:"attachmentHash,omitempty"`
	// Allowance flags (derived from allowance_types array, only present if at least one is true)
	Breakfast *bool `json:"breakfast,omitempty"`
	Lunch     *bool `json:"lunch,omitempty"`
	Dinner    *bool `json:"dinner,omitempty"`
	Lodging   *bool `json:"lodging,omitempty"`
	// Division info (denormalized)
	Division     string `json:"division"` // division code
	DivisionName string `json:"divisionName"`
	// Job info (denormalized)
	Job            string `json:"job,omitempty"` // job number
	JobDescription string `json:"jobDescription,omitempty"`
	Client         string `json:"client,omitempty"`   // client name
	Category       string `json:"category,omitempty"` // category name
	// Vendor/PO info (denormalized)
	VendorName string `json:"vendorName,omitempty"`
	Po         string `json:"po,omitempty"` // PO number
	// Workflow state
	Submitted       bool   `json:"submitted"`
	Approved        bool   `json:"approved"`
	Committed       bool   `json:"committed"`
	Rejected        bool   `json:"rejected,omitempty"`
	RejectionReason string `json:"rejectionReason,omitempty"`
	// Manager/approver info (denormalized)
	ManagerUid  string `json:"managerUid,omitempty"` // legacy_uid
	ManagerName string `json:"managerName,omitempty"`
	// Committer info (denormalized)
	CommitUid           string `json:"commitUid,omitempty"` // legacy_uid
	CommitName          string `json:"commitName,omitempty"`
	CommitTime          string `json:"commitTime,omitempty"` // timestamp
	CommittedWeekEnding string `json:"committedWeekEnding,omitempty"`
	PayPeriodEnding     string `json:"payPeriodEnding,omitempty"`
	// Rejector info (denormalized)
	RejectorId   string `json:"rejectorId,omitempty"` // legacy_uid
	RejectorName string `json:"rejectorName,omitempty"`
	// ID references to separate arrays
	VendorId        string `json:"vendorId,omitempty"`
	PurchaseOrderId string `json:"purchaseOrderId,omitempty"`
}

// containsAllowanceType checks if a JSON array string contains a specific allowance type
func containsAllowanceType(jsonArrayStr, allowanceType string) bool {
	types := parseStringArray(jsonArrayStr)
	for _, t := range types {
		if t == allowanceType {
			return true
		}
	}
	return false
}

func createExpensesExportLegacyHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Try machine auth first (Bearer token matching any unexpired legacy_writeback secret)
		authorized := false
		authHeader := e.Request.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			// TrimSpace handles trailing newlines from secret managers
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if utilities.ValidateMachineToken(app, token, "legacy_writeback") {
				authorized = true
			}
		}

		// Fall back to user auth with report claim
		if !authorized {
			if hasReport, _ := utilities.HasClaim(app, e.Auth, "report"); hasReport {
				authorized = true
			}
		}

		if !authorized {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		updatedAfter := e.Request.PathValue("updatedAfter")
		if updatedAfter == "" {
			return e.Error(http.StatusBadRequest, "updatedAfter is required", nil)
		}

		// Query 1: Expenses with denormalized fields
		expensesQuery := `
			SELECT 
			  e.id,
			  COALESCE(e.date, '') AS date,
			  COALESCE(e.description, '') AS description,
			  COALESCE(e.total, 0) AS total,
			  COALESCE(e.distance, 0) AS distance,
			  COALESCE(e.payment_type, '') AS payment_type,
			  COALESCE(e.allowance_types, '[]') AS allowance_types_json,
			  COALESCE(e.cc_last_4_digits, '') AS cc_last_4_digits,
			  COALESCE(e.attachment, '') AS attachment,
			  COALESCE(e.attachment_hash, '') AS attachment_hash,
			  COALESCE(e.submitted, 0) AS submitted,
			  COALESCE(e.approved, '') AS approved,
			  COALESCE(e.committed, '') AS committed,
			  COALESCE(e.committed_week_ending, '') AS committed_week_ending,
			  COALESCE(e.pay_period_ending, '') AS pay_period_ending,
			  COALESCE(e.rejected, '') AS rejected,
			  COALESCE(e.rejection_reason, '') AS rejection_reason,
			  -- User info (expense owner)
			  COALESCE(ap_uid.legacy_uid, '') AS uid,
			  COALESCE(p_uid.surname, '') AS surname,
			  COALESCE(p_uid.given_name, '') AS given_name,
			  COALESCE(p_uid.given_name || ' ' || p_uid.surname, '') AS display_name,
			  COALESCE(ap_uid.payroll_id, '') AS payroll_id,
			  -- Manager/approver info
			  COALESCE(ap_approver.legacy_uid, '') AS manager_uid,
			  COALESCE(p_approver.given_name || ' ' || p_approver.surname, '') AS manager_name,
			  -- Committer info
			  COALESCE(ap_committer.legacy_uid, '') AS commit_uid,
			  COALESCE(p_committer.given_name || ' ' || p_committer.surname, '') AS commit_name,
			  -- Rejector info
			  COALESCE(ap_rejector.legacy_uid, '') AS rejector_id,
			  COALESCE(p_rejector.given_name || ' ' || p_rejector.surname, '') AS rejector_name,
			  -- Division info
			  COALESCE(d.code, '') AS division,
			  COALESCE(d.name, '') AS division_name,
			  -- Job info
			  COALESCE(j.number, '') AS job,
			  COALESCE(j.description, '') AS job_description,
			  COALESCE(c.name, '') AS client,
			  -- Vendor/PO info
			  COALESCE(v.name, '') AS vendor_name,
			  COALESCE(po.po_number, '') AS po,
			  COALESCE(cat.name, '') AS category,
			  -- ID references
			  COALESCE(e.vendor, '') AS vendor_id,
			  COALESCE(e.purchase_order, '') AS purchase_order_id
			FROM expenses e
			-- User joins
			LEFT JOIN admin_profiles ap_uid ON e.uid = ap_uid.uid
			LEFT JOIN profiles p_uid ON e.uid = p_uid.uid
			-- Approver joins
			LEFT JOIN admin_profiles ap_approver ON e.approver = ap_approver.uid
			LEFT JOIN profiles p_approver ON e.approver = p_approver.uid
			-- Committer joins
			LEFT JOIN admin_profiles ap_committer ON e.committer = ap_committer.uid
			LEFT JOIN profiles p_committer ON e.committer = p_committer.uid
			-- Rejector joins
			LEFT JOIN admin_profiles ap_rejector ON e.rejector = ap_rejector.uid
			LEFT JOIN profiles p_rejector ON e.rejector = p_rejector.uid
			-- Division join
			LEFT JOIN divisions d ON e.division = d.id
			-- Job/client joins
			LEFT JOIN jobs j ON e.job = j.id
			LEFT JOIN clients c ON j.client = c.id
			-- Vendor/PO joins
			LEFT JOIN vendors v ON e.vendor = v.id
			LEFT JOIN purchase_orders po ON e.purchase_order = po.id
			-- Category join
			LEFT JOIN categories cat ON e.category = cat.id
			WHERE e.updated >= {:updatedAfter} AND e._imported = 0 AND e.committed != ''
		`

		var expenseRows []expenseExportDBRow
		if err := app.DB().NewQuery(expensesQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&expenseRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query expenses: "+err.Error(), nil)
		}

		// Query 2: Unique vendors referenced by matched expenses
		vendorsQuery := `
			SELECT DISTINCT 
			  v.id,
			  COALESCE(v.name, '') AS name,
			  COALESCE(v.alias, '') AS alias,
			  COALESCE(v.status, '') AS status
			FROM expenses e
			JOIN vendors v ON e.vendor = v.id
			WHERE e.updated >= {:updatedAfter} AND e._imported = 0 AND e.committed != '' AND e.vendor IS NOT NULL AND e.vendor != ''
		`

		var vendorRows []vendorExportDBRow
		if err := app.DB().NewQuery(vendorsQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&vendorRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query vendors: "+err.Error(), nil)
		}

		// Query 3: Unique purchase orders referenced by matched expenses
		purchaseOrdersQuery := `
			SELECT DISTINCT 
			  po.id,
			  COALESCE(po.po_number, '') AS po_number,
			  COALESCE(po.vendor, '') AS vendor_id,
			  COALESCE(v.name, '') AS vendor_name,
			  COALESCE(ap_uid.legacy_uid, '') AS uid,
			  COALESCE(po.job, '') AS job,
			  COALESCE(po.description, '') AS description,
			  COALESCE(d.code, '') AS division,
			  COALESCE(d.name, '') AS division_name,
			  COALESCE(po.total, 0) AS total,
			  COALESCE(po.approval_total, 0) AS approval_total,
			  COALESCE(po.date, '') AS date,
			  COALESCE(po.end_date, '') AS end_date,
			  COALESCE(po.approved, '') AS approved,
			  COALESCE(ap_approver.legacy_uid, '') AS approver_uid,
			  COALESCE(po.second_approval, '') AS second_approval,
			  COALESCE(ap_second_approver.legacy_uid, '') AS second_approver_uid,
			  COALESCE(ap_priority_second_approver.legacy_uid, '') AS priority_second_approver_uid,
			  COALESCE(po.cancelled, '') AS cancelled,
			  COALESCE(ap_canceller.legacy_uid, '') AS canceller_uid,
			  COALESCE(po.closed, '') AS closed,
			  COALESCE(ap_closer.legacy_uid, '') AS closer_uid,
			  COALESCE(po.rejected, '') AS rejected,
			  COALESCE(ap_rejector.legacy_uid, '') AS rejector_uid,
			  COALESCE(po.rejection_reason, '') AS rejection_reason,
			  COALESCE(po.payment_type, '') AS payment_type,
			  COALESCE(po.type, '') AS type,
			  COALESCE(po.frequency, '') AS frequency,
			  COALESCE(po.status, '') AS status,
			  COALESCE(po.category, '') AS category,
			  COALESCE(po.parent_po, '') AS parent_po,
			  COALESCE(po.branch, '') AS branch,
			  COALESCE(po.attachment, '') AS attachment,
			  COALESCE(po.attachment_hash, '') AS attachment_hash
			FROM expenses e
			JOIN purchase_orders po ON e.purchase_order = po.id
			LEFT JOIN vendors v ON po.vendor = v.id
			LEFT JOIN admin_profiles ap_uid ON po.uid = ap_uid.uid
			LEFT JOIN admin_profiles ap_approver ON po.approver = ap_approver.uid
			LEFT JOIN admin_profiles ap_second_approver ON po.second_approver = ap_second_approver.uid
			LEFT JOIN admin_profiles ap_priority_second_approver ON po.priority_second_approver = ap_priority_second_approver.uid
			LEFT JOIN admin_profiles ap_canceller ON po.canceller = ap_canceller.uid
			LEFT JOIN admin_profiles ap_closer ON po.closer = ap_closer.uid
			LEFT JOIN admin_profiles ap_rejector ON po.rejector = ap_rejector.uid
			LEFT JOIN jobs j ON po.job = j.id
			LEFT JOIN divisions d ON po.division = d.id
			WHERE e.updated >= {:updatedAfter} AND e._imported = 0 AND e.committed != '' AND e.purchase_order IS NOT NULL AND e.purchase_order != ''
		`

		var poRows []purchaseOrderExportDBRow
		if err := app.DB().NewQuery(purchaseOrdersQuery).Bind(dbx.Params{
			"updatedAfter": updatedAfter,
		}).All(&poRows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to query purchase orders: "+err.Error(), nil)
		}

		// Convert expense DB rows to output format
		expenses := make([]expenseExportOutput, len(expenseRows))
		for i, r := range expenseRows {
			expenses[i] = expenseExportOutput{
				// PocketBase ID = immutableID (for fold matching)
				ImmutableID: r.Id,
				// User info
				Uid:         r.Uid,
				Surname:     r.Surname,
				GivenName:   r.GivenName,
				DisplayName: strings.TrimSpace(r.DisplayName),
				PayrollId:   r.PayrollId,
				// Core expense data (Total is set below, conditionally)
				Date:           r.Date,
				Description:    r.Description,
				Distance:       r.Distance,
				PaymentType:    r.PaymentType,
				CcLast4Digits:  r.CcLast4Digits,
				Attachment:     r.Attachment,
				AttachmentHash: r.AttachmentHash,
				// Allowance flags are set below, only if at least one is true
				// Division info
				Division:     r.Division,
				DivisionName: r.DivisionName,
				// Job info
				Job:            r.Job,
				JobDescription: r.JobDescription,
				Client:         r.Client,
				Category:       r.Category,
				// Vendor/PO info
				VendorName: r.VendorName,
				Po:         r.Po,
				// Workflow state (convert timestamps to booleans)
				Submitted:       r.Submitted,
				Approved:        r.Approved != "",
				Committed:       r.Committed != "",
				Rejected:        r.Rejected != "",
				RejectionReason: r.RejectionReason,
				// Manager/approver info
				ManagerUid:  r.ManagerUid,
				ManagerName: strings.TrimSpace(r.ManagerName),
				// Committer info
				CommitUid:           r.CommitUid,
				CommitName:          strings.TrimSpace(r.CommitName),
				CommitTime:          r.Committed, // The committed timestamp
				CommittedWeekEnding: r.CommittedWeekEnding,
				PayPeriodEnding:     r.PayPeriodEnding,
				// Rejector info
				RejectorId:   r.RejectorId,
				RejectorName: strings.TrimSpace(r.RejectorName),
				// ID references
				VendorId:        r.VendorId,
				PurchaseOrderId: r.PurchaseOrderId,
			}

			// Only include allowance flags if at least one is true
			breakfast := containsAllowanceType(r.AllowanceTypesJSON, "Breakfast")
			lunch := containsAllowanceType(r.AllowanceTypesJSON, "Lunch")
			dinner := containsAllowanceType(r.AllowanceTypesJSON, "Dinner")
			lodging := containsAllowanceType(r.AllowanceTypesJSON, "Lodging")
			if breakfast || lunch || dinner || lodging {
				expenses[i].Breakfast = &breakfast
				expenses[i].Lunch = &lunch
				expenses[i].Dinner = &dinner
				expenses[i].Lodging = &lodging
			}

			// Only include total if payment type is not Allowance or Mileage
			if r.PaymentType != "Allowance" && r.PaymentType != "Mileage" && r.PaymentType != "Meals" {
				expenses[i].Total = &r.Total
			}
		}

		// Convert vendor DB rows to output format
		vendors := make([]vendorExportOutput, len(vendorRows))
		for i, r := range vendorRows {
			vendors[i] = vendorExportOutput{
				Id:     r.Id,
				Name:   r.Name,
				Alias:  r.Alias,
				Status: r.Status,
			}
		}

		// Convert purchase order DB rows to output format
		purchaseOrders := make([]purchaseOrderExportOutput, len(poRows))
		for i, r := range poRows {
			purchaseOrders[i] = purchaseOrderExportOutput{
				Id:                        r.Id,
				PoNumber:                  r.PoNumber,
				VendorId:                  r.VendorId,
				VendorName:                r.VendorName,
				Uid:                       r.Uid,
				Job:                       r.Job,
				Description:               r.Description,
				Division:                  r.Division,
				DivisionName:              r.DivisionName,
				Total:                     r.Total,
				ApprovalTotal:             r.ApprovalTotal,
				Date:                      r.Date,
				EndDate:                   r.EndDate,
				Approved:                  r.Approved,
				ApproverUid:               r.ApproverUid,
				SecondApproval:            r.SecondApproval,
				SecondApproverUid:         r.SecondApproverUid,
				PrioritySecondApproverUid: r.PrioritySecondApproverUid,
				Cancelled:                 r.Cancelled,
				CancellerUid:              r.CancellerUid,
				Closed:                    r.Closed,
				CloserUid:                 r.CloserUid,
				Rejected:                  r.Rejected,
				RejectorUid:               r.RejectorUid,
				RejectionReason:           r.RejectionReason,
				PaymentType:               r.PaymentType,
				Type:                      r.Type,
				Frequency:                 r.Frequency,
				Status:                    r.Status,
				Category:                  r.Category,
				ParentPo:                  r.ParentPo,
				Branch:                    r.Branch,
				Attachment:                r.Attachment,
				AttachmentHash:            r.AttachmentHash,
			}
		}

		// Return structured response with all three arrays
		return e.JSON(http.StatusOK, expensesWritebackResponse{
			Expenses:       expenses,
			Vendors:        vendors,
			PurchaseOrders: purchaseOrders,
		})
	}
}
