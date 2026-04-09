package hooks

import (
	"testing"
	"tybalt/constants"
	"tybalt/errs"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"
)

// We need to instantiate a Collection object to be part of the Record object
// so everything works as expected
var expensesCollection = core.NewBaseCollection("expenses")
var poCollection = core.NewBaseCollection("purchase_orders")

func TestValidateExpense(t *testing.T) {
	// Initialize a PocketBase TestApp for validations that require DB lookups
	app := testseed.NewSeededTestApp(t)
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
		"valid_with_four_character_description":                 {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Fuel", "job": "", "payment_type": "FuelCard", "purchase_order": "", "total": 10.00, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_with_three_character_description":              {valid: false, field: "description", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Gas", "job": "", "payment_type": "FuelCard", "purchase_order": "", "total": 10.00, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"valid_with_description":                                {valid: true, po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Valid description", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
		"invalid_description_too_short":                         {valid: false, field: "description", po: nil, hasPayablesAdminClaim: false, record: buildRecordFromMap(expensesCollection, map[string]any{"allowance_types": []string{}, "date": "2024-01-22", "description": "Gas", "job": "", "payment_type": "OnAccount", "purchase_order": "", "total": constants.NO_PO_EXPENSE_LIMIT - 0.01, "vendor": "2zqxtsmymf670ha", "attachment": "dummy.pdf"})},
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
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	rec := core.NewRecord(expensesCollection)
	rec.Set("allowance_types", []string{})
	rec.Set("job", "zke3cs3yipplwtu")
	rec.Set("payment_type", "Mileage") // avoid attachment/PO requirement
	rec.Set("kind", utilities.DefaultCapitalExpenditureKindID())
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

func TestValidateExpense_ForeignNoPOLimitUsesSettledTotal(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}
	usdRate := usdCurrency.GetFloat("rate")
	sourceTotal := utilities.RoundCurrencyAmount((constants.NO_PO_EXPENSE_LIMIT - 0.01) / usdRate)

	failing := buildRecordFromMap(expensesCollection, map[string]any{
		"allowance_types": []string{},
		"date":            "2024-01-22",
		"description":     "Foreign expense over CAD limit",
		"job":             "",
		"payment_type":    "Expense",
		"purchase_order":  "",
		"total":           sourceTotal,
		"settled_total":   constants.NO_PO_EXPENSE_LIMIT,
		"currency":        usdCurrency.Id,
		"vendor":          "2zqxtsmymf670ha",
		"attachment":      "dummy.pdf",
	})

	err = validateExpense(app, failing, nil, 0, false)
	if err == nil {
		t.Fatal("expected foreign expense at the CAD no-PO limit to require a purchase order")
	}
	errsMap, ok := err.(validation.Errors)
	if !ok {
		t.Fatalf("expected validation.Errors, got %T: %v", err, err)
	}
	if _, ok := errsMap["total"]; !ok {
		t.Fatalf("expected total error for foreign no-PO limit, got %v", errsMap)
	}

	passing := buildRecordFromMap(expensesCollection, map[string]any{
		"allowance_types": []string{},
		"date":            "2024-01-22",
		"description":     "Foreign expense under CAD limit",
		"job":             "",
		"payment_type":    "Expense",
		"purchase_order":  "",
		"total":           sourceTotal,
		"settled_total":   constants.NO_PO_EXPENSE_LIMIT - 0.01,
		"currency":        usdCurrency.Id,
		"vendor":          "2zqxtsmymf670ha",
		"attachment":      "dummy.pdf",
	})

	if err := validateExpense(app, passing, nil, 0, false); err != nil {
		t.Fatalf("expected foreign expense below CAD no-PO limit to validate, got %v", err)
	}
}

func TestValidateExpense_ForeignNoPOLimitLowRateCurrencyWithoutSettledTotal(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	jpyCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "JPY"})
	if err != nil {
		t.Fatalf("failed to load JPY currency: %v", err)
	}
	underLimitSourceTotal := 5000.0
	underLimitSettledTotal := utilities.IndicativeHomeAmount(underLimitSourceTotal, utilities.CurrencyInfo{
		Code: "JPY",
		Rate: jpyCurrency.GetFloat("rate"),
	})
	if underLimitSettledTotal >= constants.NO_PO_EXPENSE_LIMIT {
		t.Fatalf("expected JPY fixture total to stay below no-PO CAD limit, got %.2f", underLimitSettledTotal)
	}

	tests := []struct {
		name            string
		paymentType     string
		ccLast4Digits   string
		wantValid       bool
		wantErrorFields []string
		noErrorFields   []string
	}{
		{
			name:            "foreign_expense_only_requires_settled_total",
			paymentType:     "Expense",
			wantErrorFields: []string{"settled_total"},
			noErrorFields:   []string{"total"},
		},
		{
			name:        "foreign_onaccount_under_limit_draft_validates",
			paymentType: "OnAccount",
			wantValid:   true,
		},
		{
			name:          "foreign_corporate_card_under_limit_draft_validates",
			paymentType:   "CorporateCreditCard",
			ccLast4Digits: "1234",
			wantValid:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "Low-rate foreign draft under CAD cap",
				"job":             "",
				"payment_type":    tt.paymentType,
				"purchase_order":  "",
				"total":           underLimitSourceTotal,
				"settled_total":   0.0,
				"currency":        jpyCurrency.Id,
				"vendor":          "2zqxtsmymf670ha",
				"attachment":      "dummy.pdf",
				"cc_last_4_digits": tt.ccLast4Digits,
			})

			err := validateExpense(app, record, nil, 0, false)
			if tt.wantValid {
				if err != nil {
					t.Fatalf("expected under-limit %s draft to validate, got %v", tt.paymentType, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected validation error for %s draft with missing settled_total", tt.paymentType)
			}

			errMap, ok := err.(validation.Errors)
			if !ok {
				t.Fatalf("expected validation.Errors, got %T: %v", err, err)
			}
			for _, field := range tt.wantErrorFields {
				if _, ok := errMap[field]; !ok {
					t.Fatalf("expected %s validation error, got %v", field, errMap)
				}
			}
			for _, field := range tt.noErrorFields {
				if _, ok := errMap[field]; ok {
					t.Fatalf("did not expect %s validation error, got %v", field, errMap)
				}
			}
		})
	}
}

func TestValidateExpense_ForeignNoPOLimitHighRateCurrencyWithoutSettledTotal(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}

	overLimitSourceTotal := 75.0
	indicativeSettledTotal := utilities.IndicativeHomeAmount(overLimitSourceTotal, utilities.CurrencyInfo{
		Code: "USD",
		Rate: usdCurrency.GetFloat("rate"),
	})
	if indicativeSettledTotal < constants.NO_PO_EXPENSE_LIMIT {
		t.Fatalf("expected USD draft amount to exceed CAD cap, got %.2f", indicativeSettledTotal)
	}

	tests := []struct {
		name            string
		paymentType     string
		ccLast4Digits   string
		wantErrorFields []string
	}{
		{
			name:            "foreign_expense_over_limit_requires_po_and_settled_total",
			paymentType:     "Expense",
			wantErrorFields: []string{"total", "settled_total"},
		},
		{
			name:            "foreign_onaccount_over_limit_requires_po",
			paymentType:     "OnAccount",
			wantErrorFields: []string{"total"},
		},
		{
			name:            "foreign_corporate_card_over_limit_requires_po",
			paymentType:     "CorporateCreditCard",
			ccLast4Digits:   "1234",
			wantErrorFields: []string{"total"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "High-rate foreign draft over CAD cap",
				"job":             "",
				"payment_type":    tt.paymentType,
				"purchase_order":  "",
				"total":           overLimitSourceTotal,
				"settled_total":   0.0,
				"currency":        usdCurrency.Id,
				"vendor":          "2zqxtsmymf670ha",
				"attachment":      "dummy.pdf",
				"cc_last_4_digits": tt.ccLast4Digits,
			})

			err := validateExpense(app, record, nil, 0, false)
			if err == nil {
				t.Fatalf("expected validation error for over-limit %s draft", tt.paymentType)
			}

			errMap, ok := err.(validation.Errors)
			if !ok {
				t.Fatalf("expected validation.Errors, got %T: %v", err, err)
			}
			for _, field := range tt.wantErrorFields {
				if _, ok := errMap[field]; !ok {
					t.Fatalf("expected %s validation error, got %v", field, errMap)
				}
			}
		})
	}
}

func TestValidateExpense_ForeignNoPOLimitLowRateCurrencyWithSettledTotal(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	jpyCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "JPY"})
	if err != nil {
		t.Fatalf("failed to load JPY currency: %v", err)
	}
	jpyInfo := utilities.CurrencyInfo{
		Code: "JPY",
		Rate: jpyCurrency.GetFloat("rate"),
	}

	underLimitSourceTotal := 5000.0
	underLimitSettledTotal := utilities.IndicativeHomeAmount(underLimitSourceTotal, jpyInfo)
	if underLimitSettledTotal >= constants.NO_PO_EXPENSE_LIMIT {
		t.Fatalf("expected JPY under-limit settled total to stay below CAD cap, got %.2f", underLimitSettledTotal)
	}

	overLimitSourceTotal := 12000.0
	overLimitSettledTotal := utilities.IndicativeHomeAmount(overLimitSourceTotal, jpyInfo)
	if overLimitSettledTotal < constants.NO_PO_EXPENSE_LIMIT {
		t.Fatalf("expected JPY over-limit settled total to meet or exceed CAD cap, got %.2f", overLimitSettledTotal)
	}

	tests := []struct {
		name          string
		paymentType   string
		total         float64
		settledTotal  float64
		ccLast4Digits string
		wantError     string
	}{
		{
			name:         "foreign_expense_under_limit_with_settled_total_validates",
			paymentType:  "Expense",
			total:        underLimitSourceTotal,
			settledTotal: underLimitSettledTotal,
		},
		{
			name:         "foreign_expense_over_limit_with_settled_total_requires_po",
			paymentType:  "Expense",
			total:        overLimitSourceTotal,
			settledTotal: overLimitSettledTotal,
			wantError:    "total",
		},
		{
			name:         "foreign_onaccount_under_limit_with_settled_total_validates",
			paymentType:  "OnAccount",
			total:        underLimitSourceTotal,
			settledTotal: underLimitSettledTotal,
		},
		{
			name:         "foreign_onaccount_over_limit_with_settled_total_requires_po",
			paymentType:  "OnAccount",
			total:        overLimitSourceTotal,
			settledTotal: overLimitSettledTotal,
			wantError:    "total",
		},
		{
			name:          "foreign_corporate_card_under_limit_with_settled_total_validates",
			paymentType:   "CorporateCreditCard",
			total:         underLimitSourceTotal,
			settledTotal:  underLimitSettledTotal,
			ccLast4Digits: "1234",
		},
		{
			name:          "foreign_corporate_card_over_limit_with_settled_total_requires_po",
			paymentType:   "CorporateCreditCard",
			total:         overLimitSourceTotal,
			settledTotal:  overLimitSettledTotal,
			ccLast4Digits: "1234",
			wantError:     "total",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := buildRecordFromMap(expensesCollection, map[string]any{
				"allowance_types": []string{},
				"date":            "2024-01-22",
				"description":     "Low-rate foreign expense with explicit settled total",
				"job":             "",
				"payment_type":    tt.paymentType,
				"purchase_order":  "",
				"total":           tt.total,
				"settled_total":   tt.settledTotal,
				"currency":        jpyCurrency.Id,
				"vendor":          "2zqxtsmymf670ha",
				"attachment":      "dummy.pdf",
				"cc_last_4_digits": tt.ccLast4Digits,
			})

			err := validateExpense(app, record, nil, 0, false)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("expected %s to validate, got %v", tt.paymentType, err)
				}
				return
			}

			if err == nil {
				t.Fatalf("expected %s validation error for %s", tt.wantError, tt.paymentType)
			}

			errMap, ok := err.(validation.Errors)
			if !ok {
				t.Fatalf("expected validation.Errors, got %T: %v", err, err)
			}
			if _, ok := errMap[tt.wantError]; !ok {
				t.Fatalf("expected %s validation error, got %v", tt.wantError, errMap)
			}
		})
	}
}

func TestValidateExpense_CurrencyMustMatchPurchaseOrder(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}
	cadCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "CAD"})
	if err != nil {
		t.Fatalf("failed to load CAD currency: %v", err)
	}

	po := buildRecordFromMap(poCollection, map[string]any{
		"date":     "2024-01-22",
		"total":    constants.NO_PO_EXPENSE_LIMIT - 0.01,
		"type":     "One-Time",
		"currency": usdCurrency.Id,
		"job":      "mg0sp9iyjzo4zw9",
	})
	record := buildRecordFromMap(expensesCollection, map[string]any{
		"allowance_types": []string{},
		"date":            "2024-01-22",
		"description":     "Currency mismatch test",
		"job":             "mg0sp9iyjzo4zw9",
		"payment_type":    "OnAccount",
		"purchase_order":  "recordId",
		"total":           25.0,
		"currency":        cadCurrency.Id,
		"vendor":          "2zqxtsmymf670ha",
		"attachment":      "dummy.pdf",
	})

	err = validateExpense(app, record, po, 0, false)
	if err == nil {
		t.Fatal("expected currency mismatch against PO to fail validation")
	}
	hookErr, ok := err.(*errs.HookError)
	if !ok {
		t.Fatalf("expected HookError, got %T: %v", err, err)
	}
	if fieldErr, ok := hookErr.Data["currency"]; !ok || fieldErr.Code != "must_match_purchase_order" {
		t.Fatalf("expected must_match_purchase_order currency error, got %+v", hookErr.Data)
	}
}

func TestValidateExpense_ForeignExpenseSettlementTolerance(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}

	expectedSettledTotal := utilities.IndicativeHomeAmount(100, utilities.CurrencyInfo{
		Rate: usdCurrency.GetFloat("rate"),
	})
	record := buildRecordFromMap(expensesCollection, map[string]any{
		"allowance_types": []string{},
		"date":            "2024-01-22",
		"description":     "Foreign expense tolerance check",
		"job":             "",
		"payment_type":    "Expense",
		"purchase_order":  "",
		"total":           100.0,
		"settled_total":   utilities.RoundCurrencyAmount(expectedSettledTotal * 1.25),
		"currency":        usdCurrency.Id,
		"vendor":          "2zqxtsmymf670ha",
		"attachment":      "dummy.pdf",
	})

	err = validateExpense(app, record, nil, 0, false)
	if err == nil {
		t.Fatal("expected out-of-range foreign settled_total to fail validation")
	}

	errMap, ok := err.(validation.Errors)
	if !ok {
		t.Fatalf("expected validation.Errors, got %T: %v", err, err)
	}
	if _, ok := errMap["settled_total"]; !ok {
		t.Fatalf("expected settled_total validation error, got %v", errMap)
	}
}

func TestCleanExpense_CurrencyAssignmentAndSettlementRules(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	if err := utilities.ValidateExpenditureKindsConfig(app); err != nil {
		t.Fatalf("failed to load expenditure kinds config: %v", err)
	}

	homeCurrency, err := utilities.FindHomeCurrency(app)
	if err != nil {
		t.Fatalf("failed to load home currency: %v", err)
	}
	usdCurrency, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", map[string]any{"code": "USD"})
	if err != nil {
		t.Fatalf("failed to load USD currency: %v", err)
	}
	standardUser, err := app.FindAuthRecordByEmail("users", "time@test.com")
	if err != nil {
		t.Fatalf("failed to load standard user: %v", err)
	}
	mileageUser, err := app.FindAuthRecordByEmail("users", "u_mileage_valid@example.com")
	if err != nil {
		t.Fatalf("failed to load mileage-valid user: %v", err)
	}

	t.Run("po_linked_expense_inherits_po_currency", func(t *testing.T) {
		parentPO, err := app.FindRecordById("purchase_orders", "standardupd001")
		if err != nil {
			t.Fatalf("failed to load purchase order: %v", err)
		}
		if _, err := app.NonconcurrentDB().NewQuery(`
			UPDATE purchase_orders
			SET currency = {:currencyId}
			WHERE id = {:id}
		`).Bind(map[string]any{"currencyId": usdCurrency.Id, "id": parentPO.Id}).Execute(); err != nil {
			t.Fatalf("failed updating purchase order currency: %v", err)
		}

		record := buildRecordFromMap(expensesCollection, map[string]any{
			"uid":            standardUser.Id,
			"date":           "2024-01-22",
			"description":    "PO-linked foreign expense",
			"payment_type":   "OnAccount",
			"purchase_order": parentPO.Id,
			"total":          25.0,
			"vendor":         "2zqxtsmymf670ha",
			"currency":       homeCurrency.Id,
		})

		if err := cleanExpense(app, record); err != nil {
			t.Fatalf("expected cleanExpense to succeed, got %v", err)
		}
		if got := record.GetString("currency"); got != usdCurrency.Id {
			t.Fatalf("expected expense to inherit PO currency %s, got %q", usdCurrency.Id, got)
		}
	})

	t.Run("mileage_forces_home_currency_and_auto_sets_settled_total", func(t *testing.T) {
		record := buildRecordFromMap(expensesCollection, map[string]any{
			"uid":          mileageUser.Id,
			"date":         "2024-01-22",
			"description":  "Mileage to remote site",
			"payment_type": "Mileage",
			"distance":     10.0,
			"currency":     usdCurrency.Id,
		})

		if err := cleanExpense(app, record); err != nil {
			t.Fatalf("expected cleanExpense to succeed for mileage, got %v", err)
		}
		if got := record.GetString("currency"); got != homeCurrency.Id {
			t.Fatalf("expected mileage expense currency to be forced to home currency %s, got %q", homeCurrency.Id, got)
		}
		if record.GetFloat("settled_total") != record.GetFloat("total") {
			t.Fatalf("expected mileage settled_total to match total, got %v vs %v", record.GetFloat("settled_total"), record.GetFloat("total"))
		}
	})

	t.Run("blank_no_po_onaccount_currency_persists_home_currency_when_available", func(t *testing.T) {
		record := buildRecordFromMap(expensesCollection, map[string]any{
			"uid":          standardUser.Id,
			"date":         "2024-01-22",
			"description":  "On-account expense with implicit home currency",
			"payment_type": "OnAccount",
			"total":        25.0,
			"vendor":       "2zqxtsmymf670ha",
			"attachment":   "dummy.pdf",
		})

		if err := cleanExpense(app, record); err != nil {
			t.Fatalf("expected cleanExpense to succeed for blank home-currency expense, got %v", err)
		}
		if got := record.GetString("currency"); got != homeCurrency.Id {
			t.Fatalf("expected blank expense currency to persist home currency %s, got %q", homeCurrency.Id, got)
		}
	})

	t.Run("foreign_onaccount_clears_settlement_fields_for_queue_workflow", func(t *testing.T) {
		record := buildRecordFromMap(expensesCollection, map[string]any{
			"uid":           standardUser.Id,
			"date":          "2024-01-22",
			"description":   "Foreign on-account expense",
			"payment_type":  "OnAccount",
			"total":         25.0,
			"vendor":        "2zqxtsmymf670ha",
			"attachment":    "dummy.pdf",
			"currency":      usdCurrency.Id,
			"settled_total": 88.0,
			"settler":       "tqqf7q0f3378rvp",
			"settled":       "2026-04-03 12:00:00.000Z",
		})

		if err := cleanExpense(app, record); err != nil {
			t.Fatalf("expected cleanExpense to succeed for foreign on-account, got %v", err)
		}
		if record.GetFloat("settled_total") != 0 {
			t.Fatalf("expected foreign on-account settled_total to reset to 0, got %v", record.GetFloat("settled_total"))
		}
		if record.GetString("settler") != "" || !record.GetDateTime("settled").IsZero() {
			t.Fatalf("expected foreign on-account settlement actor fields to clear, got settler=%q settled=%v", record.GetString("settler"), record.GetDateTime("settled"))
		}
	})

	t.Run("foreign_out_of_pocket_keeps_user_settled_total_but_clears_actor_fields", func(t *testing.T) {
		record := buildRecordFromMap(expensesCollection, map[string]any{
			"uid":           standardUser.Id,
			"date":          "2024-01-22",
			"description":   "Foreign reimbursement expense",
			"payment_type":  "Expense",
			"total":         25.0,
			"vendor":        "2zqxtsmymf670ha",
			"attachment":    "dummy.pdf",
			"currency":      usdCurrency.Id,
			"settled_total": 91.25,
			"settler":       "tqqf7q0f3378rvp",
			"settled":       "2026-04-03 12:00:00.000Z",
		})

		if err := cleanExpense(app, record); err != nil {
			t.Fatalf("expected cleanExpense to succeed for foreign Expense, got %v", err)
		}
		if record.GetFloat("settled_total") != 91.25 {
			t.Fatalf("expected foreign Expense settled_total to remain user-provided, got %v", record.GetFloat("settled_total"))
		}
		if record.GetString("settler") != "" || !record.GetDateTime("settled").IsZero() {
			t.Fatalf("expected foreign Expense settlement actor fields to clear, got settler=%q settled=%v", record.GetString("settler"), record.GetDateTime("settled"))
		}
	})

	t.Run("foreign_out_of_pocket_zero_settled_total_remains_zero_during_cleaning", func(t *testing.T) {
		record := buildRecordFromMap(expensesCollection, map[string]any{
			"uid":           standardUser.Id,
			"date":          "2024-01-22",
			"description":   "Foreign reimbursement expense with blank settlement",
			"payment_type":  "Expense",
			"total":         25.0,
			"vendor":        "2zqxtsmymf670ha",
			"attachment":    "dummy.pdf",
			"currency":      usdCurrency.Id,
			"settled_total": 0,
		})

		if err := cleanExpense(app, record); err != nil {
			t.Fatalf("expected cleanExpense to succeed for foreign Expense, got %v", err)
		}
		if record.GetFloat("settled_total") != 0 {
			t.Fatalf("expected foreign Expense settled_total to remain user-provided at 0 during cleaning, got %v", record.GetFloat("settled_total"))
		}
	})
}
