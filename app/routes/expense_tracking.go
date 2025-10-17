package routes

import (
	"net/http"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type expenseTrackingCountRow struct {
	PayPeriodEnding string `db:"pay_period_ending" json:"pay_period_ending"`
	SubmittedCount  int    `db:"submitted_count" json:"submitted_count"`
	ApprovedCount   int    `db:"approved_count" json:"approved_count"`
	CommittedCount  int    `db:"committed_count" json:"committed_count"`
	RejectedCount   int    `db:"rejected_count" json:"rejected_count"`
}

// createExpenseTrackingCountsHandler returns pay-period grouped submitted/approved/committed counts for committers
func createExpenseTrackingCountsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		hasCommit, err := utilities.HasClaim(app, auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !hasCommit {
			return e.Error(http.StatusForbidden, "you are not authorized to view expense tracking", nil)
		}

		query := `
			SELECT
				pay_period_ending,
				-- committed are exclusive
				SUM(CASE WHEN committed != '' THEN 1 ELSE 0 END) AS committed_count,
				-- approved but not committed
				SUM(CASE WHEN approved != '' AND committed = '' THEN 1 ELSE 0 END) AS approved_count,
				-- submitted but neither approved nor committed
				SUM(CASE WHEN submitted = 1 AND approved = '' AND committed = '' THEN 1 ELSE 0 END) AS submitted_count,
				-- rejected is non-exclusive (can overlap others)
				SUM(CASE WHEN rejected != '' THEN 1 ELSE 0 END) AS rejected_count
			FROM expenses
			GROUP BY pay_period_ending
			ORDER BY pay_period_ending DESC
		`

		var rows []expenseTrackingCountRow
		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

type expenseTrackingListRow struct {
	ID              string  `db:"id" json:"id"`
	Uid             string  `db:"uid" json:"uid"`
	GivenName       string  `db:"given_name" json:"given_name"`
	Surname         string  `db:"surname" json:"surname"`
	Submitted       bool    `db:"submitted" json:"submitted"`
	Approved        string  `db:"approved" json:"approved"`
	Rejected        string  `db:"rejected" json:"rejected"`
	Committed       string  `db:"committed" json:"committed"`
	Approver        string  `db:"approver" json:"approver"`
	Committer       string  `db:"committer" json:"committer"`
	ApproverName    string  `db:"approver_name" json:"approver_name"`
	CommitterName   string  `db:"committer_name" json:"committer_name"`
	RejectorName    string  `db:"rejector_name" json:"rejector_name"`
	RejectionReason string  `db:"rejection_reason" json:"rejection_reason"`
	Phase           string  `db:"phase" json:"phase"`
	Date            string  `db:"date" json:"date"`
	Description     string  `db:"description" json:"description"`
	PaymentType     string  `db:"payment_type" json:"payment_type"`
	Distance        float64 `db:"distance" json:"distance"`
	CCLast4Digits   string  `db:"cc_last_4_digits" json:"cc_last_4_digits"`
	Attachment      string  `db:"attachment" json:"attachment"`
	AllowanceStr    string  `db:"allowance_str" json:"allowance_str"`
	JobNumber       string  `db:"job_number" json:"job_number"`
	JobDescription  string  `db:"job_description" json:"job_description"`
	ClientName      string  `db:"client_name" json:"client_name"`
	Division        string  `db:"division" json:"division"`
	DivisionCode    string  `db:"division_code" json:"division_code"`
	DivisionName    string  `db:"division_name" json:"division_name"`
	Category        string  `db:"category" json:"category"`
	CategoryName    string  `db:"category_name" json:"category_name"`
	Vendor          string  `db:"vendor" json:"vendor"`
	VendorName      string  `db:"vendor_name" json:"vendor_name"`
	VendorAlias     string  `db:"vendor_alias" json:"vendor_alias"`
	Total           float64 `db:"total" json:"total"`
}

// createExpenseTrackingListHandler returns org-wide expenses for a given pay period ending
func createExpenseTrackingListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		hasCommit, err := utilities.HasClaim(app, auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !hasCommit {
			return e.Error(http.StatusForbidden, "you are not authorized to view expense tracking", nil)
		}

		payPeriodEnding := e.Request.PathValue("payPeriodEnding")
		if payPeriodEnding == "" {
			return e.Error(http.StatusBadRequest, "payPeriodEnding is required", nil)
		}
		if _, err := time.Parse("2006-01-02", payPeriodEnding); err != nil {
			return e.Error(http.StatusBadRequest, "invalid date format (YYYY-MM-DD)", nil)
		}

		query := `
            SELECT
                e.id,
                e.uid,
                COALESCE(p.given_name, '') AS given_name,
                COALESCE(p.surname, '') AS surname,
                e.submitted,
                e.approved,
                e.rejected,
                e.committed,
                COALESCE(e.approver, '') AS approver,
                COALESCE(e.committer, '') AS committer,
                COALESCE(ap.given_name || ' ' || ap.surname, '') AS approver_name,
                COALESCE(cp.given_name || ' ' || cp.surname, '') AS committer_name,
                COALESCE(rp.given_name || ' ' || rp.surname, '') AS rejector_name,
                e.rejection_reason,
                CASE
                    WHEN e.committed != '' THEN 'Committed'
                    WHEN e.approved != '' AND e.committed = '' THEN 'Approved'
                    WHEN e.submitted = 1 AND e.approved = '' AND e.committed = '' THEN 'Submitted'
                    ELSE 'Unsubmitted'
                END AS phase,
                e.date,
                e.description,
                e.payment_type,
                CAST(e.distance AS REAL) AS distance,
                COALESCE(e.cc_last_4_digits, '') AS cc_last_4_digits,
                COALESCE(e.attachment, '') AS attachment,
                TRIM(
                    (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN 'Breakfast ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Lunch"%' THEN 'Lunch ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Dinner"%' THEN 'Dinner ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Lodging"%' THEN 'Lodging ' ELSE '' END)
                ) AS allowance_str,
                COALESCE(j.number, '') AS job_number,
                COALESCE(j.description, '') AS job_description,
                COALESCE(cl.name, '') AS client_name,
                e.division,
                COALESCE(d.code, '') AS division_code,
                COALESCE(d.name, '') AS division_name,
                e.category,
                COALESCE(cat.name, '') AS category_name,
                e.vendor,
                COALESCE(v.name, '') AS vendor_name,
                COALESCE(v.alias, '') AS vendor_alias,
                CAST(e.total AS REAL) AS total
            FROM expenses e
            LEFT JOIN profiles p ON p.uid = e.uid
            LEFT JOIN profiles ap ON ap.uid = e.approver
            LEFT JOIN profiles cp ON cp.uid = e.committer
            LEFT JOIN profiles rp ON rp.uid = e.rejector
            LEFT JOIN jobs j ON j.id = e.job
            LEFT JOIN clients cl ON cl.id = j.client
            LEFT JOIN divisions d ON d.id = e.division
            LEFT JOIN vendors v ON v.id = e.vendor
            LEFT JOIN categories cat ON cat.id = e.category
            WHERE e.pay_period_ending = {:pay_period_ending}
              AND e.submitted = 1
            ORDER BY
                CASE
                    WHEN e.committed != '' THEN 3
                    WHEN e.approved != '' AND e.committed = '' THEN 1
                    ELSE 2
                END,
                p.surname, p.given_name
        `

		var rows []expenseTrackingListRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"pay_period_ending": payPeriodEnding,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

// createExpenseTrackingAllHandler returns all submitted + uncommitted expenses (org-wide)
func createExpenseTrackingAllHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		auth := e.Auth
		hasCommit, err := utilities.HasClaim(app, auth, "commit")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking claims", err)
		}
		if !hasCommit {
			return e.Error(http.StatusForbidden, "you are not authorized to view expense tracking", nil)
		}

		query := `
            SELECT
                e.id,
                e.uid,
                COALESCE(p.given_name, '') AS given_name,
                COALESCE(p.surname, '') AS surname,
                e.submitted,
                e.approved,
                e.rejected,
                e.committed,
                COALESCE(e.approver, '') AS approver,
                COALESCE(e.committer, '') AS committer,
                COALESCE(ap.given_name || ' ' || ap.surname, '') AS approver_name,
                COALESCE(cp.given_name || ' ' || cp.surname, '') AS committer_name,
                COALESCE(rp.given_name || ' ' || rp.surname, '') AS rejector_name,
                CASE
                    WHEN e.committed != '' THEN 'Committed'
                    WHEN e.approved != '' AND e.committed = '' THEN 'Approved'
                    WHEN e.submitted = 1 AND e.approved = '' AND e.committed = '' THEN 'Submitted'
                    ELSE 'Unsubmitted'
                END AS phase,
                e.date,
                TRIM(
                    (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN 'Breakfast ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Lunch"%' THEN 'Lunch ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Dinner"%' THEN 'Dinner ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Lodging"%' THEN 'Lodging ' ELSE '' END)
                ) AS allowance_str,
                COALESCE(j.number, '') AS job_number,
                COALESCE(j.description, '') AS job_description,
                COALESCE(c.name, '') AS client_name,
                CAST(e.total AS REAL) AS total
            FROM expenses e
            LEFT JOIN profiles p ON p.uid = e.uid
            LEFT JOIN profiles ap ON ap.uid = e.approver
            LEFT JOIN profiles cp ON cp.uid = e.committer
            LEFT JOIN profiles rp ON rp.uid = e.rejector
            LEFT JOIN jobs j ON j.id = e.job
            LEFT JOIN clients c ON c.id = j.client
            WHERE e.submitted = 1
              AND e.committed = ''
            ORDER BY
                CASE
                    WHEN e.committed != '' THEN 3
                    WHEN e.approved != '' AND e.committed = '' THEN 1
                    ELSE 2
                END,
                p.surname, p.given_name
        `

		var rows []expenseTrackingListRow
		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}
