package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	reportViewOpenRule   = "@request.auth.id != \"\""
	reportViewReportRule = "@request.auth.user_claims_via_uid.cid.name ?= 'report'"
)

func init() {
	m.Register(func(app core.App) error {
		if err := updateRule(app, "time_report_week_endings", "listRule", reportViewReportRule); err != nil {
			return err
		}
		if err := updateRule(app, "time_report_week_endings", "viewRule", reportViewReportRule); err != nil {
			return err
		}
		if err := updateRule(app, "payroll_report_week_endings", "listRule", reportViewReportRule); err != nil {
			return err
		}
		return updateRule(app, "payroll_report_week_endings", "viewRule", reportViewReportRule)
	}, func(app core.App) error {
		if err := updateRule(app, "time_report_week_endings", "listRule", reportViewOpenRule); err != nil {
			return err
		}
		if err := updateRule(app, "time_report_week_endings", "viewRule", reportViewOpenRule); err != nil {
			return err
		}
		if err := updateRule(app, "payroll_report_week_endings", "listRule", reportViewOpenRule); err != nil {
			return err
		}
		return updateRule(app, "payroll_report_week_endings", "viewRule", reportViewOpenRule)
	})
}
