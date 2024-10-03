package hooks

import (
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/models"
)

// We need to instantiate a Collection object to be part of the Record object
// so everything works as expected
var expensesCollection = &models.Collection{
	Name:   "expenses",
	Type:   "base",
	System: false,
}

func buildExpenseRecordFromMap(m map[string]any) *models.Record {
	record := models.NewRecord(expensesCollection)
	record.Load(m)
	return record
}

func TestValidateExpense(t *testing.T) {
	// Test cases
	tests := map[string]struct {
		valid  bool
		field  string
		record *models.Record
	}{
		"total_too_high_without_po":                        {valid: false, field: "total", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT, "vendor_name": "Valid Vendor"})},
		"valid_without_po":                                 {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"valid_with_date":                                  {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"invalid_without_date":                             {valid: false, field: "date", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"invalid_with_invalid_date":                        {valid: false, field: "date", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-02-30", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"valid_with_description":                           {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"invalid_description_too_short":                    {valid: false, field: "description", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "tiny", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"valid_short_description_for_allowance":            {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"valid_short_description_high_total_for_allowance": {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{"Breakfast", "Lunch", "Dinner", "Lodging"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 1000, "vendor_name": "Valid Vendor"})},
		"valid_short_description_low_total_for_allowance":  {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{"Breakfast", "Lunch", "Dinner", "Lodging"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": 0.01, "vendor_name": "Valid Vendor"})},
		"valid_with_job_and_po":                            {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "recordId", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"invalid_with_job_no_po":                           {valid: false, field: "purchase_order", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT - 0.01, "vendor_name": "Valid Vendor"})},
		"invalid_with_job_no_po_no_distance_mileage":       {valid: false, field: "distance", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Mileage", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor_name": "Valid Vendor"})},
		"valid_with_job_and_distance_no_po_mileage":        {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Mileage", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor_name": "Valid Vendor", "distance": 100.00})},
		"valid_with_job_no_po_fuelcard":                    {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "FuelCard", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor_name": "Valid Vendor"})},
		"valid_with_job_no_po_personal_reimbursement":      {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "PersonalReimbursement", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor_name": "Valid Vendor"})},
		"valid_with_job_no_po_allowance":                   {valid: true, record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor_name": "Valid Vendor"})},
		"invalid_no_allowance_types_allowance":             {valid: false, field: "allowance_types", record: buildExpenseRecordFromMap(map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "Allowance", "purchase_order": "", "total": NO_PO_EXPENSE_LIMIT + 100, "vendor_name": "Valid Vendor"})},
	}

	// Run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := validateExpense(tt.record)
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
