package utilities

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Kind names are the semantic keys; IDs are resolved at runtime from the DB.
const (
	ExpenditureKindNameStandard       = "standard"
	ExpenditureKindNameSponsorship    = "sponsorship"
	ExpenditureKindNameStaffAndSocial = "staff_and_social"
	ExpenditureKindNameMediaAndEvent  = "media_and_event"
	ExpenditureKindNameComputer       = "computer"
)

const (
	POApproverLimitColumnStandardNoJob   = "max_amount"
	POApproverLimitColumnStandardWithJob = "project_max"
	POApproverLimitColumnSponsorship     = "sponsorship_max"
	POApproverLimitColumnStaffAndSocial  = "staff_and_social_max"
	POApproverLimitColumnMediaAndEvent   = "media_and_event_max"
	POApproverLimitColumnComputer        = "computer_max"
)

// nameToPOApproverColumn maps kind name -> po_approver_props limit column (non-standard kinds).
var nameToPOApproverColumn = map[string]string{
	ExpenditureKindNameSponsorship:    POApproverLimitColumnSponsorship,
	ExpenditureKindNameStaffAndSocial: POApproverLimitColumnStaffAndSocial,
	ExpenditureKindNameMediaAndEvent:  POApproverLimitColumnMediaAndEvent,
	ExpenditureKindNameComputer:       POApproverLimitColumnComputer,
}

var expectedExpenditureKindNames = []string{
	ExpenditureKindNameStandard,
	ExpenditureKindNameSponsorship,
	ExpenditureKindNameStaffAndSocial,
	ExpenditureKindNameMediaAndEvent,
	ExpenditureKindNameComputer,
}

var expectedPOApproverPropsColumns = []string{
	POApproverLimitColumnStandardNoJob,
	POApproverLimitColumnStandardWithJob,
	POApproverLimitColumnSponsorship,
	POApproverLimitColumnStaffAndSocial,
	POApproverLimitColumnMediaAndEvent,
	POApproverLimitColumnComputer,
}

// Runtime cache: populated by ValidateExpenditureKindsConfig.
var (
	kindNameToID = map[string]string{}
	kindIDToName = map[string]string{}
)

// DefaultExpenditureKindID returns the canonical "standard" expenditure kind ID.
// Must be called after ValidateExpenditureKindsConfig has run (e.g. after app serve).
func DefaultExpenditureKindID() string {
	return kindNameToID[ExpenditureKindNameStandard]
}

// NormalizeExpenditureKindID returns the provided kind ID or falls back to
// "standard" when no kind is present (legacy records).
func NormalizeExpenditureKindID(kindID string) string {
	if strings.TrimSpace(kindID) == "" {
		return DefaultExpenditureKindID()
	}
	return kindID
}

// ResolvePOApproverLimitColumn returns which po_approver_props limit column to
// use for second-approval qualification.
func ResolvePOApproverLimitColumn(kindID string, hasJob bool) (string, error) {
	if strings.TrimSpace(kindID) == "" {
		return POApproverLimitColumnStandardNoJob, nil
	}
	name := kindIDToName[kindID]
	if name == "" {
		name = ExpenditureKindNameStandard
	}
	if name == ExpenditureKindNameStandard {
		if hasJob {
			return POApproverLimitColumnStandardWithJob, nil
		}
		return POApproverLimitColumnStandardNoJob, nil
	}
	column, ok := nameToPOApproverColumn[name]
	if !ok {
		return "", fmt.Errorf("unsupported expenditure kind id %q (name %q)", kindID, name)
	}
	return column, nil
}

// ValidateExpenditureKindsConfig loads expenditure_kinds into the name↔id cache and
// verifies that required kind names and po_approver_props columns exist.
func ValidateExpenditureKindsConfig(app core.App) error {
	type kindRow struct {
		ID   string `db:"id"`
		Name string `db:"name"`
	}
	rows := []kindRow{}
	err := app.DB().NewQuery(`SELECT id, name FROM expenditure_kinds`).All(&rows)
	if err != nil {
		return fmt.Errorf("validate expenditure kinds query failed: %w", err)
	}

	kindNameToID = make(map[string]string, len(rows))
	kindIDToName = make(map[string]string, len(rows))
	for _, row := range rows {
		kindNameToID[row.Name] = row.ID
		kindIDToName[row.ID] = row.Name
	}

	for _, expectedName := range expectedExpenditureKindNames {
		if _, ok := kindNameToID[expectedName]; !ok {
			return fmt.Errorf("missing expenditure_kind with name %q", expectedName)
		}
	}

	type columnRow struct {
		Name string `db:"name"`
	}
	columnRows := []columnRow{}
	if err := app.DB().NewQuery(`PRAGMA table_info(po_approver_props)`).All(&columnRows); err != nil {
		return fmt.Errorf("validate po_approver_props columns failed: %w", err)
	}

	columns := map[string]bool{}
	for _, col := range columnRows {
		columns[col.Name] = true
	}
	for _, columnName := range expectedPOApproverPropsColumns {
		if !columns[columnName] {
			return fmt.Errorf("missing po_approver_props column %q", columnName)
		}
	}

	return nil
}
