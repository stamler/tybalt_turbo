package hooks

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// We need to instantiate a Collection object to be part of the Record object
// so everything works as expected
var expensesCollection = core.NewBaseCollection("expenses")
var poCollection = core.NewBaseCollection("purchase_orders")

func TestValidateExpense(t *testing.T) {
	// Test cases
	tests := map[string]struct {
		valid                 bool
		field                 string
		record                *core.Record
		po                    *core.Record
		hasPayablesAdminClaim bool
	}{
		"total_too_high_without_po":                                    {valid: false, field: "total", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT, "vendor": "2zqxtsmymf670ha"})},
		"high_total_fine_without_po_for_payables_admin_OnAccount":      {valid: true, field: "total", po: nil, hasPayablesAdminClaim: true, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 1000, "vendor": "2zqxtsmymf670ha"})},
		"high_total_fails_without_po_for_payables_admin_not_OnAccount": {valid: false, field: "total", po: nil, hasPayablesAdminClaim: true, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Expense", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 1000, "vendor": "2zqxtsmymf670ha"})},
		"valid_without_po":                                      {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_date":                                       {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_without_date":                                  {valid: false, field: "date", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_with_invalid_date":                             {valid: false, field: "date", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-02-30", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_description":                                {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_description_too_short":                         {valid: false, field: "description", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "tiny", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_short_description_for_allowance":                 {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_short_description_high_total_for_allowance":      {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast", "Lunch", "Dinner", "Lodging"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 1000, "vendor": "2zqxtsmymf670ha"})},
		"valid_short_description_low_total_for_allowance":       {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast", "Lunch", "Dinner", "Lodging"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_job_and_po":                                 {valid: true, po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-22", "total": NO_PO_EXPENSE_LIMIT - 0.01, "type": "Normal"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "recordId", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_with_job_and_po_recurring_date_after_end_date": {valid: false, field: "date", po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-22", "end_date": "2024-02-22", "total": NO_PO_EXPENSE_LIMIT - 0.01, "type": "Recurring"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-02-23", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "recordId", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_with_job_and_po_if_too_early_for_po":           {valid: false, field: "date", po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-23", "total": NO_PO_EXPENSE_LIMIT - 0.01, "type": "Normal"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "recordId", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_with_job_no_po":                                {valid: false, po: nil, hasPayablesAdminClaim: false, field: "purchase_order", record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"invalid_with_job_no_po_no_distance_mileage":            {valid: false, po: nil, hasPayablesAdminClaim: false, field: "distance", record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Mileage", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_job_and_distance_no_po_mileage":             {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Mileage", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha", "distance": 100.00})},
		"valid_with_job_no_po_fuelcard":                         {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "FuelCard", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_job_no_po_personal_reimbursement":           {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "PersonalReimbursement", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_job_no_po_allowance":                        {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
		"invalid_no_allowance_types_allowance":                  {valid: false, field: "allowance_types", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
	}

	// Run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// TODO: Add tests where a PO record is provided
			// TODO: Add tests where an existing expenses total is provided for Cumulative POs
			got := validateExpense(tt.record, tt.po, 0.0, tt.hasPayablesAdminClaim)
			if got != nil {
				if tt.valid {
					t.Errorf("failed validation (%v) but expected valid", got)
				} else {
					// Extract the field from the error
					errMap := got.(validation.Errors)
					if len(errMap) != 1 {
						t.Errorf("expected one error, got %d", len(errMap))
					}
					for field := range errMap {
						if field != tt.field {
							t.Errorf("expected field: %s, got: %s", tt.field, field)
						}
					}
				}
			} else if !tt.valid {
				t.Errorf("passed validation but expected invalid")
			}
		})
	}
}
