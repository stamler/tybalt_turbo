package hooks

import (
	"testing"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// We need to instantiate a Collection object to be part of the Record object
// so everything works as expected
var expensesCollection = core.NewBaseCollection("expenses")
var poCollection = core.NewBaseCollection("purchase_orders")

func TestValidateExpense(t *testing.T) {
	// Initialize a PocketBase TestApp for validations that require DB lookups
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}
	// Test cases
	tests := map[string]struct {
		valid                 bool
		field                 string
		record                *core.Record
		po                    *core.Record
		existingExpensesTotal float64
		hasPayablesAdminClaim bool
		expectedErrorCode     string
	}{
		// Vendor presence validation across payment types
		"invalid_vendor_required_for_expense":                          {valid: false, field: "vendor", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Expense", "purchase_order": "", "total": 10.00, "vendor": "", "attachment": "dummy.pdf"})},
		"invalid_vendor_required_for_onaccount":                        {valid: false, field: "vendor", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "", "attachment": "dummy.pdf"})},
		"invalid_vendor_required_for_fuelcard":                         {valid: false, field: "vendor", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "FuelCard", "purchase_order": "", "total": 10.00, "vendor": "", "attachment": "dummy.pdf"})},
		"invalid_vendor_required_for_corporate_credit_card":            {valid: false, field: "vendor", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "CorporateCreditCard", "purchase_order": "", "total": 10.00, "cc_last_4_digits": "1234", "vendor": "", "attachment": "dummy.pdf"})},
		"valid_vendor_optional_for_allowance_missing":                  {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": 25.00, "vendor": ""})},
		"valid_vendor_optional_for_allowance_present":                  {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": 25.00, "vendor": "2zqxtsmymf670ha"})},
		"valid_vendor_optional_for_personal_reimbursement_missing":     {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "PersonalReimbursement", "purchase_order": "", "total": 25.00, "vendor": ""})},
		"valid_vendor_optional_for_personal_reimbursement_present":     {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "PersonalReimbursement", "purchase_order": "", "total": 25.00, "vendor": "2zqxtsmymf670ha"})},
		"valid_vendor_optional_for_mileage_missing":                    {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Mileage", "purchase_order": "", "total": 25.00, "distance": 10.0, "vendor": ""})},
		"valid_vendor_optional_for_mileage_present":                    {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Mileage", "purchase_order": "", "total": 25.00, "distance": 10.0, "vendor": "2zqxtsmymf670ha"})},
		"total_too_high_without_po":                                    {valid: false, field: "total", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"high_total_fine_without_po_for_payables_admin_OnAccount":      {valid: true, field: "total", po: nil, hasPayablesAdminClaim: true, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 1000, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"high_total_fails_without_po_for_payables_admin_not_OnAccount": {valid: false, field: "total", po: nil, hasPayablesAdminClaim: true, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Expense", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 1000, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_without_po":                                      {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_with_date":                                       {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_without_date":                                  {valid: false, field: "date", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_invalid_date":                             {valid: false, field: "date", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-02-30", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_with_description":                                {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_description_too_short":                         {valid: false, field: "description", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "tiny", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_short_description_for_allowance":                 {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_short_description_high_total_for_allowance":      {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast", "Lunch", "Dinner", "Lodging"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 1000, "vendor": "2zqxtsmymf670ha"})},
		"valid_short_description_low_total_for_allowance":       {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast", "Lunch", "Dinner", "Lodging"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": 0.01, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_job_and_po":                                 {valid: true, po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-22", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "type": "One-Time", "job": "mg0sp9iyjzo4zw9"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "OnAccount", "purchase_order": "recordId", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_job_and_po_recurring_date_after_end_date": {valid: false, field: "date", po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-22", "end_date": "2024-02-22", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "type": "Recurring", "job": "mg0sp9iyjzo4zw9"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-02-23", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "OnAccount", "purchase_order": "recordId", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_job_and_po_if_too_early_for_po":           {valid: false, field: "date", po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-23", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "type": "One-Time", "job": "jobId"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "jobId", "payment_type": "OnAccount", "purchase_order": "recordId", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_job_mismatch_po_job":                      {valid: false, field: "job", expectedErrorCode: "must_match_purchase_order", po: buildRecordFromMap(poCollection, map[string]any{"date": "2024-01-22", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "type": "One-Time", "job": "poJob"}), hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "expenseJob", "payment_type": "OnAccount", "purchase_order": "recordId", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_job_no_po":                                {valid: false, po: nil, hasPayablesAdminClaim: false, field: "purchase_order", record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_job_no_po_no_distance_mileage":            {valid: false, po: nil, hasPayablesAdminClaim: false, field: "distance", record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "Mileage", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
		"valid_with_job_and_distance_no_po_mileage":             {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "Mileage", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha", "distance": 100.00})},
		"valid_with_job_no_po_fuelcard":                         {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "FuelCard", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_with_job_no_po_personal_reimbursement":           {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "PersonalReimbursement", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_with_job_no_po_allowance":                        {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "Allowance", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},
		"invalid_no_allowance_types_allowance":                  {valid: false, field: "allowance_types", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "mg0sp9iyjzo4zw9", "payment_type": "Allowance", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT + 100, "vendor": "2zqxtsmymf670ha"})},

		// Attachment presence validation across payment types
		"invalid_attachment_required_for_expense":                   {valid: false, field: "attachment", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Expense", "purchase_order": "", "total": 10.00, "vendor": "2zqxtsmymf670ha"})},
		"invalid_attachment_required_for_onaccount":                 {valid: false, field: "attachment", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": 10.00, "vendor": "2zqxtsmymf670ha"})},
		"invalid_attachment_required_for_corporate_credit_card_att": {valid: false, field: "attachment", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "CorporateCreditCard", "purchase_order": "", "total": 10.00, "cc_last_4_digits": "1234", "vendor": "2zqxtsmymf670ha"})},
		"invalid_attachment_required_for_fuelcard_att":              {valid: false, field: "attachment", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "FuelCard", "purchase_order": "", "total": 10.00, "vendor": "2zqxtsmymf670ha"})},
		"valid_no_attachment_for_allowance":                         {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{"Breakfast"}, "date": "2024-01-22", "description": "", "job": "", "payment_type": "Allowance", "purchase_order": "", "total": 25.00, "vendor": ""})},
		"valid_no_attachment_for_mileage":                           {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "Mileage", "purchase_order": "", "total": 25.00, "distance": 10.0, "vendor": ""})},
		"valid_no_attachment_for_personal_reimbursement":            {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "PersonalReimbursement", "purchase_order": "", "total": 25.00, "vendor": ""})},

		// Tests that an expense against a Cumulative PO is valid when:
		// 1. The PO has a total of $1000
		// 2. There are existing expenses totaling $500
		// 3. The new expense is $400
		// This should pass because $500 + $400 = $900, which is less than the PO total of $1000
		"valid_cumulative_po_within_limit": {
			valid: true,
			po: buildRecordFromMap(poCollection, map[string]any{
				"date":  "2024-01-22",
				"total": 1000.00,
				"type":  "Cumulative",
				"job":   "mg0sp9iyjzo4zw9",
			}),
			existingExpensesTotal: 500.00,
			record: buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "Valid description",
				"job":             "mg0sp9iyjzo4zw9",
				"payment_type":    "OnAccount",
				"purchase_order":  "recordId",
				"total":           400.00,
				"vendor":          "2zqxtsmymf670ha",
				"attachment":      "dummy.pdf",
			}),
		},

		// Tests that an expense against a Cumulative PO fails with the correct error when:
		// 1. The PO has a total of $1000
		// 2. There are existing expenses totaling $800
		// 3. The new expense is $300
		// This should fail because $800 + $300 = $1100, which exceeds the PO total
		// The test verifies that:
		// - The error is in the "total" field
		// - The error code is specifically "cumulative_po_overflow"
		"invalid_cumulative_po_overflow": {
			valid: false,
			field: "total",
			po: buildRecordFromMap(poCollection, map[string]any{
				"date":  "2024-01-22",
				"total": 1000.00,
				"type":  "Cumulative",
				"job":   "mg0sp9iyjzo4zw9",
			}),
			existingExpensesTotal: 800.00,
			expectedErrorCode:     "cumulative_po_overflow",
			record: buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "Valid description",
				"job":             "mg0sp9iyjzo4zw9",
				"payment_type":    "OnAccount",
				"purchase_order":  "recordId",
				"total":           300.00,
				"vendor":          "2zqxtsmymf670ha",
				"attachment":      "dummy.pdf",
			}),
		},

		// Tests that a single large expense against a Cumulative PO fails appropriately when:
		// 1. The PO has a total of $1000
		// 2. There are NO existing expenses ($0 total)
		// 3. The new expense is $1200
		// This should fail because the single expense of $1200 exceeds the PO total
		// This test is important because it verifies the overflow detection works even
		// when there are no existing expenses. It also tests a different scenario from
		// the previous test where multiple smaller expenses add up to exceed the total.
		// The test verifies that:
		// - The error is in the "total" field
		// - The error code is specifically "cumulative_po_overflow"
		"invalid_cumulative_po_single_expense_overflow": {
			valid: false,
			field: "total",
			po: buildRecordFromMap(poCollection, map[string]any{
				"date":  "2024-01-22",
				"total": 1000.00,
				"type":  "Cumulative",
				"job":   "mg0sp9iyjzo4zw9",
			}),
			existingExpensesTotal: 0.00,
			expectedErrorCode:     "cumulative_po_overflow",
			record: buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "Valid description",
				"job":             "mg0sp9iyjzo4zw9",
				"payment_type":    "OnAccount",
				"purchase_order":  "recordId",
				"total":           1200.00,
				"vendor":          "2zqxtsmymf670ha",
				"attachment":      "dummy.pdf",
			}),
		},
	}

	// Run tests
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// TODO: Add tests where a PO record is provided
			// TODO: Add tests where an existing expenses total is provided for Cumulative POs
			got := validateExpense(app, tt.record, tt.po, tt.existingExpensesTotal, tt.hasPayablesAdminClaim)
			if got != nil {
				if tt.valid {
					t.Errorf("failed validation (%v) but expected valid", got)
				} else {
					// Extract the field from the error
					if hookErr, ok := got.(*errs.HookError); ok {
						if codeErr, ok := hookErr.Data[tt.field]; ok {
							if tt.expectedErrorCode != "" && codeErr.Code != tt.expectedErrorCode {
								t.Errorf("expected error code: %s, got: %s", tt.expectedErrorCode, codeErr.Code)
							}
						} else {
							t.Errorf("expected error in field: %s, but field not found in error", tt.field)
						}
					} else {
						errMap := got.(validation.Errors)
						if _, ok := errMap[tt.field]; !ok {
							t.Errorf("expected field %s to be in errors, got: %v", tt.field, errMap)
						}
					}
				}
			} else if !tt.valid {
				t.Errorf("passed validation but expected invalid")
			}
		})
	}
}

func TestValidateExpense_RejectsClosedJob(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	rec := core.NewRecord(expensesCollection)
	rec.Set("allowance_types", []string{})
	rec.Set("job", "zke3cs3yipplwtu")
	rec.Set("payment_type", "Mileage") // avoid attachment/PO requirement
	rec.Set("kind", utilities.DefaultExpenditureKindID())
	rec.Set("distance", 10.0)
	rec.Set("date", "2024-01-22")
	rec.Set("description", "Valid description")
	rec.Set("total", 25.0)

	if err := validateExpense(app, rec, nil, 0.0, false); err == nil {
		t.Fatalf("expected validation to fail for inactive job, got nil")
	} else {
		if verrs, ok := err.(validation.Errors); ok {
			if _, ok := verrs["job"]; !ok {
				t.Fatalf("expected job error, got: %v", verrs)
			}
		} else {
			t.Fatalf("expected validation.Errors, got %T: %v", err, err)
		}
	}
}
