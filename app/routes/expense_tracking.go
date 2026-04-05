package routes

import (
	"net/http"
	"time"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type expenseTrackingCountRow struct {
	CommittedWeekEnding              string `db:"committed_week_ending" json:"committed_week_ending"`
	CommittedCount                   int    `db:"committed_count" json:"committed_count"`
	PlaceholderPayrollIDExpenseCount int    `db:"placeholder_payroll_id_expense_count" json:"placeholder_payroll_id_expense_count"`
}

func requireExpenseTrackingViewer(app core.App, auth *core.Record) error {
	isReportHolder, err := utilities.HasClaim(app, auth, "report")
	if err != nil {
		return err
	}
	if isReportHolder {
		return nil
	}

	isCommitter, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return err
	}
	if isCommitter {
		return nil
	}

	hasAdmin, err := utilities.HasClaim(app, auth, "admin")
	if err != nil {
		return err
	}
	if hasAdmin {
		return nil
	}

	return &errs.HookError{
		Status:  http.StatusForbidden,
		Message: "you are not authorized to view expense tracking",
		Data: map[string]errs.CodeError{
			"global": {
				Code:    "unauthorized",
				Message: "you are not authorized to view expense tracking",
			},
		},
	}
}

func requireExpenseCommitQueueViewer(app core.App, auth *core.Record) error {
	isCommitter, err := utilities.HasClaim(app, auth, "commit")
	if err != nil {
		return err
	}
	if isCommitter {
		return nil
	}

	return &errs.HookError{
		Status:  http.StatusForbidden,
		Message: "you are not authorized to view the expense commit queue",
		Data: map[string]errs.CodeError{
			"global": {
				Code:    "unauthorized",
				Message: "you are not authorized to view the expense commit queue",
			},
		},
	}
}

// createExpenseTrackingCountsHandler returns committed expense counts grouped by committed_week_ending for report/commit/admin viewers.
func createExpenseTrackingCountsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireExpenseTrackingViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		query := `
			SELECT
				e.committed_week_ending,
				COUNT(*) AS committed_count,
				SUM(CASE WHEN ` + utilities.GeneratedPayrollPlaceholderSQLCondition(`ap.payroll_id`) + ` THEN 1 ELSE 0 END) AS placeholder_payroll_id_expense_count
			FROM expenses e
			LEFT JOIN admin_profiles ap ON ap.uid = e.uid
			WHERE committed != '' AND committed_week_ending != ''
			GROUP BY e.committed_week_ending
			ORDER BY e.committed_week_ending DESC
		`

		var rows []expenseTrackingCountRow
		if err := app.DB().NewQuery(query).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

type expenseTrackingListRow struct {
	ID               string  `db:"id" json:"id"`
	Uid              string  `db:"uid" json:"uid"`
	GivenName        string  `db:"given_name" json:"given_name"`
	Surname          string  `db:"surname" json:"surname"`
	Submitted        bool    `db:"submitted" json:"submitted"`
	Approved         string  `db:"approved" json:"approved"`
	Rejected         string  `db:"rejected" json:"rejected"`
	Committed        string  `db:"committed" json:"committed"`
	Approver         string  `db:"approver" json:"approver"`
	Committer        string  `db:"committer" json:"committer"`
	ApproverName     string  `db:"approver_name" json:"approver_name"`
	CommitterName    string  `db:"committer_name" json:"committer_name"`
	RejectorName     string  `db:"rejector_name" json:"rejector_name"`
	RejectionReason  string  `db:"rejection_reason" json:"rejection_reason"`
	Date             string  `db:"date" json:"date"`
	Description      string  `db:"description" json:"description"`
	PaymentType      string  `db:"payment_type" json:"payment_type"`
	Distance         float64 `db:"distance" json:"distance"`
	CCLast4Digits    string  `db:"cc_last_4_digits" json:"cc_last_4_digits"`
	Attachment       string  `db:"attachment" json:"attachment"`
	AllowanceStr     string  `db:"allowance_str" json:"allowance_str"`
	JobNumber        string  `db:"job_number" json:"job_number"`
	JobDescription   string  `db:"job_description" json:"job_description"`
	ClientName       string  `db:"client_name" json:"client_name"`
	Division         string  `db:"division" json:"division"`
	DivisionCode     string  `db:"division_code" json:"division_code"`
	DivisionName     string  `db:"division_name" json:"division_name"`
	Category         string  `db:"category" json:"category"`
	CategoryName     string  `db:"category_name" json:"category_name"`
	Vendor           string  `db:"vendor" json:"vendor"`
	VendorName       string  `db:"vendor_name" json:"vendor_name"`
	VendorAlias      string  `db:"vendor_alias" json:"vendor_alias"`
	Total            float64 `db:"total" json:"total"`
	SettledTotal     float64 `db:"settled_total" json:"settled_total"`
	Currency         string  `db:"currency" json:"currency"`
	CurrencyCode     string  `db:"currency_code" json:"currency_code"`
	CurrencySymbol   string  `db:"currency_symbol" json:"currency_symbol"`
	CurrencyIcon     string  `db:"currency_icon" json:"currency_icon"`
	CurrencyRate     float64 `db:"currency_rate" json:"currency_rate"`
	CurrencyRateDate string  `db:"currency_rate_date" json:"currency_rate_date"`
	PONumber         string  `db:"po_number" json:"po_number"`
}

// createExpenseTrackingListHandler returns org-wide committed expenses for a given committed week ending for report/commit/admin viewers.
func createExpenseTrackingListHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireExpenseTrackingViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
		}

		committedWeekEnding := e.Request.PathValue("committedWeekEnding")
		if committedWeekEnding == "" {
			return e.Error(http.StatusBadRequest, "committedWeekEnding is required", nil)
		}
		if _, err := time.Parse("2006-01-02", committedWeekEnding); err != nil {
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
                CAST(e.total AS REAL) AS total,
                CAST(COALESCE(e.settled_total, 0) AS REAL) AS settled_total,
                COALESCE(e.currency, '') AS currency,
                COALESCE(cur.code, 'CAD') AS currency_code,
                COALESCE(cur.symbol, 'CAD') AS currency_symbol,
                COALESCE(cur.icon, '') AS currency_icon,
                COALESCE(CAST(cur.rate AS REAL), 1) AS currency_rate,
                COALESCE(cur.rate_date, '') AS currency_rate_date,
                COALESCE(po.po_number, '') AS po_number
            FROM expenses e
            LEFT JOIN profiles p ON p.uid = e.uid
            LEFT JOIN profiles ap ON ap.uid = e.approver
            LEFT JOIN profiles cp ON cp.uid = e.committer
            LEFT JOIN profiles rp ON rp.uid = e.rejector
            LEFT JOIN jobs j ON j.id = e.job
            LEFT JOIN clients cl ON cl.id = j.client
            LEFT JOIN divisions d ON d.id = e.division
                LEFT JOIN vendors v ON v.id = e.vendor
                LEFT JOIN currencies cur ON cur.id = e.currency
                LEFT JOIN categories cat ON cat.id = e.category
                LEFT JOIN purchase_orders po ON po.id = e.purchase_order
            WHERE e.committed_week_ending = {:committed_week_ending}
	              AND e.committed != ''
            ORDER BY p.surname, p.given_name, e.date
        `

		var rows []expenseTrackingListRow
		if err := app.DB().NewQuery(query).Bind(dbx.Params{
			"committed_week_ending": committedWeekEnding,
		}).All(&rows); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to execute query", err)
		}

		return e.JSON(http.StatusOK, rows)
	}
}

// createExpenseCommitQueueHandler returns all submitted + uncommitted expenses
// (org-wide) for the expense commit queue.
func createExpenseCommitQueueHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireExpenseCommitQueueViewer(app, e.Auth); err != nil {
			return writeHookError(e, err)
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
                e.description,
                COALESCE(e.attachment, '') AS attachment,
                TRIM(
                    (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN 'Breakfast ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Lunch"%' THEN 'Lunch ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Dinner"%' THEN 'Dinner ' ELSE '' END) ||
                    (CASE WHEN e.allowance_types LIKE '%"Lodging"%' THEN 'Lodging ' ELSE '' END)
                ) AS allowance_str,
                COALESCE(j.number, '') AS job_number,
                COALESCE(j.description, '') AS job_description,
                COALESCE(c.name, '') AS client_name,
                CAST(e.total AS REAL) AS total,
                CAST(COALESCE(e.settled_total, 0) AS REAL) AS settled_total,
                COALESCE(e.currency, '') AS currency,
                COALESCE(cur.code, 'CAD') AS currency_code,
                COALESCE(cur.symbol, 'CAD') AS currency_symbol,
                COALESCE(cur.icon, '') AS currency_icon,
                COALESCE(CAST(cur.rate AS REAL), 1) AS currency_rate,
                COALESCE(cur.rate_date, '') AS currency_rate_date,
                COALESCE(po.po_number, '') AS po_number
            FROM expenses e
            LEFT JOIN profiles p ON p.uid = e.uid
            LEFT JOIN profiles ap ON ap.uid = e.approver
            LEFT JOIN profiles cp ON cp.uid = e.committer
            LEFT JOIN profiles rp ON rp.uid = e.rejector
            LEFT JOIN jobs j ON j.id = e.job
            LEFT JOIN clients c ON c.id = j.client
            LEFT JOIN currencies cur ON cur.id = e.currency
            LEFT JOIN purchase_orders po ON po.id = e.purchase_order
            WHERE e.submitted = 1
              AND e.committed = ''
              AND e.approved != ''
              AND NOT (
                COALESCE(cur.code, 'CAD') != 'CAD'
                AND e.payment_type IN ('OnAccount', 'CorporateCreditCard')
                AND COALESCE(e.settled, '') = ''
              )
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
