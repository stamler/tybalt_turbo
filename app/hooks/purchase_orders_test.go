package hooks

import (
	"testing"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
)

func TestValidatePurchaseOrder_DescriptionMinimumLength(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	activeAdminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "active = true", dbx.Params{})
	if err != nil {
		t.Fatalf("failed to load active admin profile: %v", err)
	}
	activeVendor, err := app.FindFirstRecordByFilter("vendors", "status = 'Active'", dbx.Params{})
	if err != nil {
		t.Fatalf("failed to load active vendor: %v", err)
	}

	tests := map[string]struct {
		description string
		valid       bool
	}{
		"accepts_four_characters":  {description: "Fuel", valid: true},
		"rejects_three_characters": {description: "Gas", valid: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			record := buildRecordFromMap(poCollection, map[string]any{
				"uid":          activeAdminProfile.GetString("uid"),
				"date":         "2024-09-01",
				"division":     "vccd5fo56ctbigh",
				"description":  tt.description,
				"payment_type": "Expense",
				"total":        1234.56,
				"vendor":       activeVendor.Id,
				"approver":     activeAdminProfile.GetString("uid"),
				"type":         "One-Time",
			})

			err := ValidatePurchaseOrder(app, record, true)
			if tt.valid {
				if err != nil {
					t.Fatalf("expected valid purchase order, got %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected description validation error, got nil")
			}

			errMap, ok := err.(validation.Errors)
			if !ok {
				t.Fatalf("expected validation.Errors, got %T: %v", err, err)
			}
			if _, ok := errMap["description"]; !ok {
				t.Fatalf("expected description error, got %v", errMap)
			}
		})
	}
}
