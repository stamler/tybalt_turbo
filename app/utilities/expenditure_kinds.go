package utilities

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// Kind names are the semantic keys; IDs are resolved at runtime from the DB.
const (
	ExpenditureKindNameCapital       = "capital"
	ExpenditureKindNameProject       = "project"
	ExpenditureKindNameSponsorship   = "sponsorship"
	ExpenditureKindNameStaffAndSocial = "staff_and_social"
	ExpenditureKindNameMediaAndEvent  = "media_and_event"
	ExpenditureKindNameComputer       = "computer"

	// Legacy name kept only for import/migration back-compat.
	ExpenditureKindNameStandard = "standard"
)

const (
	POApproverLimitColumnCapital       = "max_amount"
	POApproverLimitColumnProject       = "project_max"
	POApproverLimitColumnSponsorship   = "sponsorship_max"
	POApproverLimitColumnStaffAndSocial = "staff_and_social_max"
	POApproverLimitColumnMediaAndEvent  = "media_and_event_max"
	POApproverLimitColumnComputer       = "computer_max"
)

// nameToPOApproverColumn maps kind name -> po_approver_props limit column.
var nameToPOApproverColumn = map[string]string{
	ExpenditureKindNameCapital:       POApproverLimitColumnCapital,
	ExpenditureKindNameProject:       POApproverLimitColumnProject,
	ExpenditureKindNameSponsorship:   POApproverLimitColumnSponsorship,
	ExpenditureKindNameStaffAndSocial: POApproverLimitColumnStaffAndSocial,
	ExpenditureKindNameMediaAndEvent:  POApproverLimitColumnMediaAndEvent,
	ExpenditureKindNameComputer:       POApproverLimitColumnComputer,
}

var expectedExpenditureKindNames = []string{
	ExpenditureKindNameCapital,
	ExpenditureKindNameProject,
	ExpenditureKindNameSponsorship,
	ExpenditureKindNameStaffAndSocial,
	ExpenditureKindNameMediaAndEvent,
	ExpenditureKindNameComputer,
}

var expectedPOApproverPropsColumns = []string{
	POApproverLimitColumnCapital,
	POApproverLimitColumnProject,
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

// DefaultCapitalExpenditureKindID returns the "capital" expenditure kind ID.
func DefaultCapitalExpenditureKindID() string {
	return kindNameToID[ExpenditureKindNameCapital]
}

// DefaultProjectExpenditureKindID returns the "project" expenditure kind ID.
func DefaultProjectExpenditureKindID() string {
	return kindNameToID[ExpenditureKindNameProject]
}

// DefaultExpenditureKindIDForJob returns the appropriate default kind ID
// based on whether a job is present: project when true, capital when false.
func DefaultExpenditureKindIDForJob(hasJob bool) string {
	if hasJob {
		return DefaultProjectExpenditureKindID()
	}
	return DefaultCapitalExpenditureKindID()
}

// NormalizeExpenditureKindID returns the provided kind ID or falls back to
// a job-aware default when no kind is present (legacy records).
// If the resolved kind is capital but hasJob is true, it returns project
// to prevent invalid capital+job combos.
func NormalizeExpenditureKindID(kindID string, hasJob bool) string {
	if strings.TrimSpace(kindID) == "" {
		return DefaultExpenditureKindIDForJob(hasJob)
	}
	// Prevent capital+job combos by normalizing to project.
	name := kindIDToName[kindID]
	if name == ExpenditureKindNameCapital && hasJob {
		return DefaultProjectExpenditureKindID()
	}
	return kindID
}

// ResolvePOApproverLimitColumn returns which po_approver_props limit column to
// use for second-approval qualification.
func ResolvePOApproverLimitColumn(kindID string, hasJob bool) (string, error) {
	if strings.TrimSpace(kindID) == "" {
		// Legacy records with no kind: default by job presence.
		if hasJob {
			return POApproverLimitColumnProject, nil
		}
		return POApproverLimitColumnCapital, nil
	}
	name := kindIDToName[kindID]
	if name == "" {
		// Unknown kind ID: fall back by job presence.
		if hasJob {
			return POApproverLimitColumnProject, nil
		}
		return POApproverLimitColumnCapital, nil
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
