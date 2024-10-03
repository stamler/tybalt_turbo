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
		"total_too_high_without_po": {
			valid: false,
			field: "total",
			record: buildExpenseRecordFromMap(map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "Valid description",
				"job":             "",
				"payment_type":    "OnAccount",
				"purchase_order":  "",
				"total":           100.50,
				"vendor_name":     "Valid Vendor",
			}),
		},
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
