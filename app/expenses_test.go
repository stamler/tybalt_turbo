// expenses_test.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func setupAdminOnlyExpenseViewerApp(tb testing.TB) *tests.TestApp {
	tb.Helper()

	app := testutils.SetupTestApp(tb)
	claim, err := app.FindFirstRecordByFilter("claims", "name = 'admin'")
	if err != nil {
		tb.Fatalf("failed to load admin claim: %v", err)
	}

	_, err = app.NonconcurrentDB().NewQuery(`
		INSERT OR IGNORE INTO user_claims (
			_imported,
			cid,
			created,
			id,
			uid,
			updated
		) VALUES (
			0,
			{:cid},
			strftime('%Y-%m-%d %H:%M:%fZ', 'now'),
			{:id},
			{:uid},
			strftime('%Y-%m-%d %H:%M:%fZ', 'now')
		)
	`).Bind(dbx.Params{
		"cid": claim.Id,
		"id":  "test_admin_only_expense_claim_u_no_claims",
		"uid": "u_no_claims",
	}).Execute()
	if err != nil {
		tb.Fatalf("failed to grant admin claim in test setup: %v", err)
	}

	return app
}

func TestExpensesCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	mileageValidToken, err := testutils.GenerateRecordToken("users", "u_mileage_valid@example.com")
	if err != nil {
		t.Fatal(err)
	}
	mileageMissingToken, err := testutils.GenerateRecordToken("users", "u_mileage_missing@example.com")
	if err != nil {
		t.Fatal(err)
	}
	mileageExpiredToken, err := testutils.GenerateRecordToken("users", "u_mileage_expired@example.com")
	if err != nil {
		t.Fatal(err)
	}
	mileageSameDayToken, err := testutils.GenerateRecordToken("users", "u_mileage_same_day@example.com")
	if err != nil {
		t.Fatal(err)
	}
	disallowedPersonalReimbursementToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		t.Fatalf("failed to load capital kind: %v", err)
	}
	capitalKindID := capitalKind.Id

	// multipart builder for creates with attachment
	makeMultipart := func(jsonBody string) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		if _, exists := m["kind"]; !exists {
			m["kind"] = capitalKindID
		}
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		// Minimal PNG header so mime detection passes (image/png)
		if _, err := fw.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	scenarios := []tests.ApiScenario{
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "valid expense leaves pay period ending blank until commit and sets approver",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"approved":""`,
					`"approver":"f2j5a8vk006baub"`,
					`"pay_period_ending":""`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "apkev2ow1zjtm7w",
				"description": "inactive division should fail",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "otherwise valid expense with Inactive division fails",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"data":{"division":{"code":"not_active"`,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		{
			Name:   "expense with job fails when division is not allocated to that job",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "90drdtwx5v4ew70",
				"description": "allowance with unallocated division",
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast"],
				"total": 0,
				"job": "test_job_w_rs"
			}`),
			Headers:        map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"division":{"code":"division_not_allowed"`,
				`Division BM is not allocated to this job`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		func() tests.ApiScenario {
			// Using 2025-01-10 so the effective allowance rate row is 2025-01-05
			// Breakfast=20, Lunch=25, Dinner=30, Lodging=50 on that date.
			// With allowance_types ["Breakfast","Dinner"], total should be 20+30=50.
			// Vendor is always cleared for Allowance by the cleanExpense hook and
			// description is set to "Allowance for Breakfast, Dinner".
			body := strings.NewReader(fmt.Sprintf(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"kind": %q,
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast", "Dinner"],
				"total": 0,
				"vendor": "2zqxtsmymf670ha",
				"description": "This will be overwritten"
			}`, utilities.DefaultCapitalExpenditureKindID()))
			return tests.ApiScenario{
				Name:           "valid allowance expense calculates total, clears vendor, and sets description",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           body,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Allowance\"",
					"\"allowance_types\":[\"Breakfast\",\"Dinner\"]",
					"\"total\":50",
					"\"vendor\":\"\"",
					"Allowance for Breakfast, Dinner",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// No-PO expenses without a job should be forced to the capital kind by the hook.
			body := strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast"],
				"total": 0,
				"description": "ignored for allowance"
			}`)
			return tests.ApiScenario{
				Name:           "no-po create without kind is set to capital kind",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           body,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": "application/json"},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					fmt.Sprintf(`"kind":"%s"`, capitalKindID),
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"committed": "2024-11-01 00:00:00",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "writing the committed property is forbidden",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"message":"Failed to create record.","status":400`},
				ExpectedEvents:  map[string]int{"*": 0},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "ctswqva5onxj75q"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "otherwise valid expense with Inactive vendor fails",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"message":"Failed to create record.","status":400`},
				ExpectedEvents:  map[string]int{},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 132.10,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "expense created against an Active, One-Time purchase_orders record succeeds",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  200,
				ExpectedContent: []string{`"purchase_order":"poa1ctvbrnch001"`},
				ExpectedEvents:  map[string]int{"OnRecordCreate": 1},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense branch override",
				"payment_type": "Expense",
				"total": 132.10,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001",
				"branch": "xeq9q81q5307f70"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "expense with purchase order forces branch from linked purchase order",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"purchase_order":"poa1ctvbrnch001"`,
					`"branch":"xeq9q81q5307f70"`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 132.10,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "ponactvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "expense created against a non-Active, One-Time purchase_orders record fails",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"data":{"purchase_order":{"code":"not_active"`},
				ExpectedEvents:  map[string]int{"*": 0, "OnRecordCreateRequest": 1},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "po with missing branch should fail",
				"payment_type": "Expense",
				"total": 132.10,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poactvmissbr001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "expense with purchase order fails when linked purchase order branch is missing",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"purchase_order":{"code":"missing_branch"`,
				},
				ExpectedEvents: map[string]int{"*": 0, "OnRecordCreateRequest": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting category with job succeeds if purchase_order is set",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"category":"t5nmdl188gtlhz0"`,
					`"job":"cjf0kt0defhq480"`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category with job fails if purchase_order is not set",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"data":{"purchase_order":{"code":"validation_required"`},
				ExpectedEvents:  map[string]int{"*": 0, "OnRecordCreateRequest": 1},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting job without category fails if purchase_order is not set",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"data":{"purchase_order":{"code":"validation_required"`},
				ExpectedEvents:  map[string]int{"*": 0, "OnRecordCreateRequest": 1},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "he1f7oej613mxh7",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category with job fails if category does not belong to the job even if purchase_order is set",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"message":"Failed to create record."`},
				ExpectedEvents: map[string]int{
					"OnModelBeforeCreate":         0,
					"OnModelAfterCreate":          0,
					"OnRecordBeforeCreateRequest": 0,
					"OnRecordAfterCreateRequest":  0,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Using 2025-01-10 so the effective expense rate is 2025-01-05
			// Mileage tiers on that date are {"0": 0.70, "5000": 0.64}
			// With no prior mileage in test DB for the period and distance 100,
			// total should be 100 * 0.70 = 70.00 and vendor should be cleared.
			b, ct, err := makeMultipart(`{
				"uid": "u_mileage_valid",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"description": "mileage",
				"payment_type": "Mileage",
				"distance": 100,
				"total": 0,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "valid mileage expense gets total calculated and vendor cleared",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": mileageValidToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Mileage\"",
					"\"distance\":100",
					"\"total\":70",
					"\"vendor\":\"\"",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Period starts 2025-01-05; tiers {0:0.70, 5000:0.64}. With distance 5100
			// and zero prior mileage, expected total is 5000*0.70 + 100*0.64 = 3564.
			b, ct, err := makeMultipart(`{
				"uid": "u_mileage_valid",
				"date": "2025-01-05",
				"division": "vccd5fo56ctbigh",
				"description": "mileage spanning tiers",
				"payment_type": "Mileage",
				"distance": 5100,
				"total": 0,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "valid mileage expense spanning tiers gets total calculated and vendor cleared",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": mileageValidToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Mileage\"",
					"\"distance\":5100",
					"\"total\":3564",
					"\"vendor\":\"\"",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Mileage expense should fail when insurance expiry is not set
			b, ct, err := makeMultipart(`{
				"uid": "u_mileage_missing",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"description": "mileage with no insurance",
				"payment_type": "Mileage",
				"distance": 100,
				"total": 0,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "mileage expense fails when insurance expiry is not set",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": mileageMissingToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"code":"insurance_expiry_missing"`,
					`personal vehicle insurance expiry must be updated with a valid date`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreateRequest": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Mileage expense should fail when insurance has expired
			b, ct, err := makeMultipart(`{
				"uid": "u_mileage_expired",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"description": "mileage with expired insurance",
				"payment_type": "Mileage",
				"distance": 100,
				"total": 0,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "mileage expense fails when insurance has expired",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": mileageExpiredToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"code":"insurance_expired"`,
					`personal vehicle insurance expired on 2024-12-31`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreateRequest": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Mileage expense should succeed when expense date equals insurance expiry date
			b, ct, err := makeMultipart(`{
				"uid": "u_mileage_same_day",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"description": "mileage on expiry day",
				"payment_type": "Mileage",
				"distance": 100,
				"total": 0,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "mileage expense succeeds when expense date equals insurance expiry date",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": mileageSameDayToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Mileage\"",
					"\"distance\":100",
					"\"total\":70",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Non-mileage expense should not be affected by missing insurance expiry
			b, ct, err := makeMultipart(`{
				"uid": "u_mileage_missing",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "regular expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "non-mileage expense not affected by missing insurance expiry",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": mileageMissingToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"payment_type":"Expense"`,
					`"total":99`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Personal reimbursement should fail when the user's admin profile disallows it.
			b, ct, err := makeMultipart(`{
				"uid": "u_no_claims",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"description": "personal reimbursement with disallowed profile",
				"payment_type": "PersonalReimbursement",
				"total": 25,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "personal reimbursement fails when profile flag is false",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": disallowedPersonalReimbursementToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"code":"personal_reimbursement_not_allowed"`,
					`cannot submit personal reimbursement expense: personal reimbursement is not enabled for your profile`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreateRequest": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			// Personal reimbursement should succeed when the user's admin profile allows it.
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"description": "personal reimbursement with allowed profile",
				"payment_type": "PersonalReimbursement",
				"total": 25,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "personal reimbursement succeeds when profile flag is true",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"payment_type":"PersonalReimbursement"`,
					`"total":25`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),

		// TODO: expenses created against an Active purchase_orders record for which the caller is not allowed to create an expense fail
		// TODO: enhance validate_expenses_test.go
		{
			Name:   "unauthenticated request fails",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"attachment": "receipt.png"
				}`),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to create record."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 0,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting rejector, rejected, and rejection_reason fails",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"rejector": "f2j5a8vk006baub",
				"rejected": "2024-09-01 15:04:05",
				"rejection_reason": "This is a rejection"
				}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to create record."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 0,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting approved or approver fails",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"approved": "2024-09-01 15:04:05",
				"approver": "f2j5a8vk006baub"
				}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to create record."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 0,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:   "setting category without job fails",
				Method: http.MethodPost,
				URL:    "/api/collections/expenses/records",
				Body:   b,
				Headers: map[string]string{
					"Authorization": recordToken,
					"Content-Type":  ct,
				},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"message":"Failed to create record."`,
				},
				ExpectedEvents: map[string]int{
					"OnModelBeforeCreate":         0,
					"OnModelAfterCreate":          0,
					"OnRecordBeforeCreateRequest": 0,
					"OnRecordAfterCreateRequest":  0,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting category with job succeeds if purchase_order is set",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"category":"t5nmdl188gtlhz0"`,
					`"job":"cjf0kt0defhq480"`,
				},
				ExpectedEvents: map[string]int{
					"OnRecordCreate": 1,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting category with job fails if purchase_order is not set",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"data":{"purchase_order":{"code":"validation_required"`,
				},
				ExpectedEvents: map[string]int{
					"*":                     0,
					"OnRecordCreateRequest": 1,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting job without category fails if purchase_order is not set",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"data":{"purchase_order":{"code":"validation_required"`,
				},
				ExpectedEvents: map[string]int{
					"*":                     0,
					"OnRecordCreateRequest": 1,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "he1f7oej613mxh7",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category with job fails if category does not belong to the job even if purchase_order is set",
				Method:          http.MethodPost,
				URL:             "/api/collections/expenses/records",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"message":"Failed to create record."`},
				ExpectedEvents: map[string]int{
					"OnModelBeforeCreate":         0,
					"OnModelAfterCreate":          0,
					"OnRecordBeforeCreateRequest": 0,
					"OnRecordAfterCreateRequest":  0,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestRejectExpense_QueuesNotifications verifies that rejecting an expense via the route
// queues one or more expense_rejected notifications.
func TestRejectExpense_QueuesNotifications(t *testing.T) {
	// Choose an expense that is submitted, not committed, and not yet rejected.
	const expenseToReject = "eqhozipupteogp8"

	// Use the same committer user as in other route tests (has commit claim).
	committerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	baselineApp := testutils.SetupTestApp(t)
	t.Cleanup(baselineApp.Cleanup)
	beforeCount := testutils.CountNotificationsByTemplateCode(t, baselineApp, "expense_rejected")

	scenario := tests.ApiScenario{
		Name:   "reject expense queues notifications",
		Method: http.MethodPost,
		URL:    "/api/expenses/" + expenseToReject + "/reject",
		Body:   strings.NewReader(`{"rejection_reason": "Route-level expense rejection test"}`),
		Headers: map[string]string{
			"Authorization": committerToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"message":"record rejected successfully"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// After the request, ensure that at least one new expense_rejected notification was created.
	scenario.AfterTestFunc = func(tb testing.TB, app *tests.TestApp, res *http.Response) {
		resultCount := testutils.CountNotificationsByTemplateCode(tb, app, "expense_rejected")
		if resultCount <= beforeCount {
			tb.Fatalf("expected expense_rejected notifications to be created by reject route, before=%d after=%d", beforeCount, resultCount)
		}
	}

	scenario.Test(t)
}

func TestRejectExpense_CommitHolderCannotRejectUnapprovedExpense(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "commit holder cannot reject a submitted but unapproved expense",
		Method: http.MethodPost,
		URL:    "/api/expenses/exp_approve_closed_po_1/reject",
		Body:   strings.NewReader(`{"rejection_reason": "should fail for unapproved expense"}`),
		Headers: map[string]string{
			"Authorization": commitToken,
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"code":"record_not_approved"`,
			`"message":"only approved records can be rejected by a commit user"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestExpenseCommitQueue_RequiresCommitClaim(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "report holder cannot view expense commit queue",
		Method: http.MethodGet,
		URL:    "/api/expenses/commit_queue",
		Headers: map[string]string{
			"Authorization": reportToken,
		},
		ExpectedStatus: http.StatusForbidden,
		ExpectedContent: []string{
			`"message":"you are not authorized to view the expense commit queue"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

type expenseCommitQueueRow struct {
	ID          string `json:"id"`
	Approved    string `json:"approved"`
	Rejected    string `json:"rejected"`
	Committed   string `json:"committed"`
	Description string `json:"description"`
	Attachment  string `json:"attachment"`
}

func decodeExpenseCommitQueueRows(t testing.TB, res *http.Response) []expenseCommitQueueRow {
	t.Helper()
	defer res.Body.Close()

	var rows []expenseCommitQueueRow
	if err := json.NewDecoder(res.Body).Decode(&rows); err != nil {
		t.Fatalf("failed decoding expense commit queue response: %v", err)
	}

	return rows
}

func expenseCommitQueueContains(rows []expenseCommitQueueRow, id string) bool {
	for _, row := range rows {
		if row.ID == id {
			return true
		}
	}
	return false
}

func expenseCommitQueueRowByID(t testing.TB, rows []expenseCommitQueueRow, id string) expenseCommitQueueRow {
	t.Helper()

	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}

	t.Fatalf("expected expense commit queue to include row %q", id)
	return expenseCommitQueueRow{}
}

func TestExpenseCommitQueue_ReturnsApprovedUncommittedExpenses(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "commit queue only includes approved uncommitted expenses",
		Method: http.MethodGet,
		URL:    "/api/expenses/commit_queue",
		Headers: map[string]string{
			"Authorization": commitToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"eqhozipupteogp8"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			rows := decodeExpenseCommitQueueRows(t, res)

			for _, row := range rows {
				if row.Approved == "" {
					t.Fatalf("queue row %q unexpectedly had empty approved timestamp", row.ID)
				}
				if row.Committed != "" {
					t.Fatalf("queue row %q unexpectedly had committed timestamp %q", row.ID, row.Committed)
				}
			}

			for _, id := range []string{
				"b4o6xph4ngwx4nw",
				"eqhozipupteogp8",
				"hlqb5xdzm2xbii7",
				"um1uoad5a4mhfcu",
			} {
				if !expenseCommitQueueContains(rows, id) {
					t.Fatalf("expected expense commit queue to include approved uncommitted expense %q", id)
				}
			}

			for _, id := range []string{
				"exp_approve_closed_po_1",
				"su3hyft6n9rlt7d",
			} {
				if expenseCommitQueueContains(rows, id) {
					t.Fatalf("expected expense commit queue to exclude %q", id)
				}
			}
		},
	}

	scenario.Test(t)
}

func TestExpenseCommitQueue_ReturnsDescriptionAttachmentAndRejectedRows(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "commit queue includes description attachment and rejected approved rows",
		Method: http.MethodGet,
		URL:    "/api/expenses/commit_queue",
		Headers: map[string]string{
			"Authorization": commitToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"id":"eqhozipupteogp8"`,
		},
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app := testutils.SetupTestApp(tb)

			_, err := app.NonconcurrentDB().NewQuery(`
				UPDATE expenses
				SET rejected = {:rejected},
				    rejection_reason = {:reason},
				    rejector = {:rejector},
				    attachment = {:attachment}
				WHERE id = {:id}
			`).Bind(dbx.Params{
				"id":         "eqhozipupteogp8",
				"rejected":   "2024-11-08 12:34:56.000Z",
				"reason":     "Testing rejected approved expense visibility",
				"rejector":   "wegviunlyr2jjjv",
				"attachment": "queue-receipt.pdf",
			}).Execute()
			if err != nil {
				tb.Fatalf("failed to seed rejected expense commit queue row: %v", err)
			}

			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, res *http.Response) {
			rows := decodeExpenseCommitQueueRows(t, res)
			row := expenseCommitQueueRowByID(t, rows, "eqhozipupteogp8")

			if row.Description != "An approved expense against a Cumulative purchase_orders record. This should commit well because the total is less than than maximum allowed amount by the purchase_orders record." {
				t.Fatalf("description = %q, want seeded expense description", row.Description)
			}
			if row.Attachment != "queue-receipt.pdf" {
				t.Fatalf("attachment = %q, want %q", row.Attachment, "queue-receipt.pdf")
			}
			if row.Rejected != "2024-11-08 12:34:56.000Z" {
				t.Fatalf("rejected = %q, want %q", row.Rejected, "2024-11-08 12:34:56.000Z")
			}
			if row.Approved == "" {
				t.Fatalf("approved unexpectedly empty for row %q", row.ID)
			}
		},
	}

	scenario.Test(t)
}

func TestExpenseCommitSetsCommittedWeekAndPayPeriodEnding(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "committing an expense sets committed week ending and pay period ending",
		Method:         http.MethodPost,
		URL:            "/api/expenses/eqhozipupteogp8/commit",
		Headers:        map[string]string{"Authorization": commitToken},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"message":"Record committed successfully"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordUpdate": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.AfterTestFunc = func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
		record, err := app.FindRecordById("expenses", "eqhozipupteogp8")
		if err != nil {
			tb.Fatalf("failed to load committed expense: %v", err)
		}

		committedDate := record.GetDateTime("committed").Time().Format("2006-01-02")
		expectedWeekEnding, err := utilities.GenerateWeekEnding(committedDate)
		if err != nil {
			tb.Fatalf("failed to generate expected committed week ending: %v", err)
		}

		expectedPayPeriodEnding, err := utilities.GenerateCommittedPayPeriodEnding(record.GetString("date"), expectedWeekEnding)
		if err != nil {
			tb.Fatalf("failed to generate expected pay period ending: %v", err)
		}

		if got := record.GetString("committed_week_ending"); got != expectedWeekEnding {
			tb.Fatalf("committed_week_ending = %q, want %q", got, expectedWeekEnding)
		}

		if got := record.GetString("pay_period_ending"); got != expectedPayPeriodEnding {
			tb.Fatalf("pay_period_ending = %q, want %q", got, expectedPayPeriodEnding)
		}
	}

	scenario.Test(t)
}

func TestSeededExpensesNormalizePayPeriodEndingByCommitStatus(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	type expenseRow struct {
		ID                  string `db:"id"`
		Committed           string `db:"committed"`
		CommittedWeekEnding string `db:"committed_week_ending"`
		PayPeriodEnding     string `db:"pay_period_ending"`
	}

	var rows []expenseRow
	if err := app.DB().NewQuery(`
		SELECT
			id,
			COALESCE(committed, '') AS committed,
			COALESCE(committed_week_ending, '') AS committed_week_ending,
			COALESCE(pay_period_ending, '') AS pay_period_ending
		FROM expenses
		ORDER BY id
	`).All(&rows); err != nil {
		t.Fatalf("failed to query seeded expenses: %v", err)
	}

	for _, row := range rows {
		if row.Committed == "" {
			if row.PayPeriodEnding != "" {
				t.Fatalf("uncommitted expense %s has pay_period_ending %q; expected blank", row.ID, row.PayPeriodEnding)
			}
			continue
		}

		if row.CommittedWeekEnding == "" {
			t.Fatalf("committed expense %s has blank committed_week_ending", row.ID)
		}
		if row.PayPeriodEnding == "" {
			t.Fatalf("committed expense %s has blank pay_period_ending", row.ID)
		}
	}
}

func TestExpensesUpdate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		t.Fatalf("failed to load capital kind: %v", err)
	}
	capitalKindID := capitalKind.Id

	// multipart builder for updates with attachment
	updateMultipart := func(jsonBody string) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		if _, exists := m["kind"]; !exists {
			m["kind"] = capitalKindID
		}
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "update.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
			return nil, "", err
		}
		ct := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, ct, nil
	}

	scenarios := []tests.ApiScenario{
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "valid expense keeps pay period ending blank until commit",
				Method:         http.MethodPatch,
				URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"approved":""`,
					`"approver":"f2j5a8vk006baub"`,
					`"pay_period_ending":""`,
				},
				ExpectedEvents: map[string]int{"OnRecordUpdate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "succeeds when uid is present and matches existing value",
				Method:          http.MethodPatch,
				URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  200,
				ExpectedContent: []string{`"uid":"rzr98oadsp9qc11"`},
				ExpectedEvents:  map[string]int{"OnRecordUpdate": 1},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting category with job succeeds if purchase_order is set",
				Method:         http.MethodPatch,
				URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"category":"t5nmdl188gtlhz0"`,
					`"job":"cjf0kt0defhq480"`,
				},
				ExpectedEvents: map[string]int{"OnRecordUpdate": 1},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category with job fails if purchase_order is not set",
				Method:          http.MethodPatch,
				URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  404,
				ExpectedContent: []string{`"message":"The requested resource wasn't found."`},
				ExpectedEvents:  map[string]int{"OnModelBeforeUpdate": 0, "OnModelAfterUpdate": 0, "OnRecordBeforeUpdateRequest": 0, "OnRecordAfterUpdateRequest": 0},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting job without category fails if purchase_order is not set",
				Method:          http.MethodPatch,
				URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"data":{"purchase_order":{"code":"validation_required"`},
				ExpectedEvents:  map[string]int{"OnRecordUpdateRequest": 1, "*": 0},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "he1f7oej613mxh7",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting category with job fails if category does not belong to the job even if purchase_order is set",
				Method:         http.MethodPatch,
				URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 404,
				ExpectedContent: []string{
					`"message":"The requested resource wasn't found."`,
				},
				ExpectedEvents: map[string]int{
					"OnModelBeforeUpdate":         0,
					"OnModelAfterUpdate":          0,
					"OnRecordBeforeUpdateRequest": 0,
					"OnRecordAfterUpdateRequest":  0,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		{
			Name:   "unauthenticated request fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
				}`),
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 0,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "fails when uid is present but it does not match existing value",
			Method: http.MethodPatch,
			URL:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"uid": "f2j5a8vk006baub",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
				}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         0,
				"OnModelAfterUpdate":          0,
				"OnRecordBeforeUpdateRequest": 0,
				"OnRecordAfterUpdateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting rejector, rejected, and rejection_reason fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"rejector": "f2j5a8vk006baub",
				"rejected": "2024-09-01 15:04:05",
				"rejection_reason": "This is a rejection"
				}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         0,
				"OnModelAfterUpdate":          0,
				"OnRecordBeforeUpdateRequest": 0,
				"OnRecordAfterUpdateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting approved or approver fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"approved": "2024-09-01 15:04:05",
				"approver": "f2j5a8vk006baub"
				}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         0,
				"OnModelAfterUpdate":          0,
				"OnRecordBeforeUpdateRequest": 0,
				"OnRecordAfterUpdateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category without job fails",
				Method:          http.MethodPatch,
				URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  404,
				ExpectedContent: []string{`"message":"The requested resource wasn't found."`},
				ExpectedEvents:  map[string]int{"OnModelBeforeUpdate": 0, "OnModelAfterUpdate": 0, "OnRecordBeforeUpdateRequest": 0, "OnRecordAfterUpdateRequest": 0},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category with job succeeds if purchase_order is set",
				Method:          http.MethodPatch,
				URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  200,
				ExpectedContent: []string{`"category":"t5nmdl188gtlhz0"`, `"job":"cjf0kt0defhq480"`},
				ExpectedEvents:  map[string]int{"OnRecordUpdate": 1},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:            "setting category with job fails if purchase_order is not set",
				Method:          http.MethodPatch,
				URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:            b,
				Headers:         map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus:  400,
				ExpectedContent: []string{`"data":{"purchase_order":{"code":"validation_required"`},
				ExpectedEvents:  map[string]int{"OnRecordUpdateRequest": 1, "*": 0},
				TestAppFactory:  testutils.SetupTestApp,
			}
		}(),
		func() tests.ApiScenario {
			b, ct, err := updateMultipart(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"job": "cjf0kt0defhq480"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "setting job without category fails if purchase_order is not set",
				Method:         http.MethodPatch,
				URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"data":{"purchase_order":{"code":"validation_required"`,
				},
				ExpectedEvents: map[string]int{
					"OnRecordUpdateRequest": 1,
					"*":                     0,
				},
				TestAppFactory: testutils.SetupTestApp,
			}
		}(),
		{
			Name:   "setting category with job fails if category does not belong to the job even if purchase_order is set",
			Method: http.MethodPatch,
			URL:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "he1f7oej613mxh7",
				"job": "cjf0kt0defhq480",
				"purchase_order": "poa1ctvbrnch001"
				}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         0,
				"OnModelAfterUpdate":          0,
				"OnRecordBeforeUpdateRequest": 0,
				"OnRecordAfterUpdateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestExpensesDelete(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	nonCreatorToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "expense cannot be deleted by user whose id does not match uid",
			Method:         http.MethodDelete,
			URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Headers:        map[string]string{"Authorization": nonCreatorToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeDelete":         0,
				"OnModelAfterDelete":          0,
				"OnRecordBeforeDeleteRequest": 0,
				"OnRecordAfterDeleteRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "expense can be deleted by user whose id matches uid",
			Method:         http.MethodDelete,
			URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 204,
			ExpectedEvents: map[string]int{
				"OnRecordDelete": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "expense cannot be deleted by the creator if it is submitted",
			Method:         http.MethodDelete,
			URL:            "/api/collections/expenses/records/b4o6xph4ngwx4nw",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeDelete":         0,
				"OnModelAfterDelete":          0,
				"OnRecordBeforeDeleteRequest": 0,
				"OnRecordAfterDeleteRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "expense cannot be deleted by the creator if it is committed",
			Method:         http.MethodDelete,
			URL:            "/api/collections/expenses/records/xg2yeucklhgbs3n",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeDelete":         0,
				"OnModelAfterDelete":          0,
				"OnRecordBeforeDeleteRequest": 0,
				"OnRecordAfterDeleteRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestExpensesRead(t *testing.T) {
	creatorToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:            "caller can read expenses records containing their uid",
			Method:          http.MethodGet,
			URL:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Headers:         map[string]string{"Authorization": creatorToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"id":"2gq9uyxmkcyopa4"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller cannot read unsubmitted expenses records where they are approver",
			Method:         http.MethodGet,
			URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Headers:        map[string]string{"Authorization": approverToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller can read submitted expenses records where they are approver",
			Method:          http.MethodGet,
			URL:             "/api/collections/expenses/records/xg2yeucklhgbs3n",
			Headers:         map[string]string{"Authorization": approverToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"id":"xg2yeucklhgbs3n"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller with the commit claim can read approved expenses records",
			Method:          http.MethodGet,
			URL:             "/api/collections/expenses/records/b4o6xph4ngwx4nw",
			Headers:         map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"id":"b4o6xph4ngwx4nw"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the commit claim cannot read unapproved expenses records",
			Method:         http.MethodGet,
			URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Headers:        map[string]string{"Authorization": commitToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller with the report claim can read committed expenses records",
			Method:          http.MethodGet,
			URL:             "/api/collections/expenses/records/xg2yeucklhgbs3n",
			Headers:         map[string]string{"Authorization": reportToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"id":"xg2yeucklhgbs3n"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the report claim cannot read uncommitted expenses records",
			Method:         http.MethodGet,
			URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeRead":         0,
				"OnModelAfterRead":          0,
				"OnRecordBeforeReadRequest": 0,
				"OnRecordAfterReadRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestExpensesRoutes(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	ownerToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "submit fails when expense references a closed purchase order",
			Method: http.MethodPost,
			URL:    "/api/expenses/exp_submit_closed_po_1/submit",
			Headers: map[string]string{
				"Authorization": ownerToken,
			},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"error":"purchase order is not active"`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:   "approve fails when expense references a closed purchase order",
			Method: http.MethodPost,
			URL:    "/api/expenses/exp_approve_closed_po_1/approve",
			Headers: map[string]string{
				"Authorization": approverToken,
			},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"error":"purchase order is not active"`},
			TestAppFactory:  testutils.SetupTestApp,
		},
		{
			Name:            "caller with the commit claim can commit approved expenses records",
			Method:          http.MethodPost,
			URL:             "/api/expenses/eqhozipupteogp8/commit",
			Headers:         map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Record committed successfully"`},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "referenced Cumulative purchase_orders record is closed, when total matches or exceeds PO total",
			Method:          http.MethodPost,
			URL:             "/api/expenses/hlqb5xdzm2xbii7/commit",
			Headers:         map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Record committed successfully"`},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 2,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "commit fails against cumulative purchase_orders record if the total exceeds the PO total by more than specified excess",
			Method:          http.MethodPost,
			URL:             "/api/expenses/um1uoad5a4mhfcu/commit",
			Headers:         map[string]string{"Authorization": commitToken},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"error":"the committed expenses total exceeds the total value of the purchase order beyond the allowed surplus"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func setupClosedPurchaseOrderForUncommit(t testing.TB, poID string) *tests.TestApp {
	t.Helper()

	app := testutils.SetupTestApp(t)
	po, err := app.FindRecordById("purchase_orders", poID)
	if err != nil {
		t.Fatalf("failed to load purchase order %s: %v", poID, err)
	}

	po.Set("status", "Closed")
	po.Set("closed", "2026-04-02 12:00:00.000Z")
	po.Set("closer", "tqqf7q0f3378rvp")
	po.Set("closed_by_system", true)

	if err := app.Save(po); err != nil {
		t.Fatalf("failed to close purchase order %s in test setup: %v", poID, err)
	}

	return app
}

func TestExpenseDetailsAndUncommitRoutes(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "admin can read committed expense details through custom route",
			Method:         http.MethodGet,
			URL:            "/api/expenses/details/xg2yeucklhgbs3n",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"xg2yeucklhgbs3n"`,
				`"committed":"2024-09-20 12:00:00.000Z"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin cannot read uncommitted expense details through custom route",
			Method:         http.MethodGet,
			URL:            "/api/expenses/details/eqhozipupteogp8",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"Expense not found or not authorized."`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin can uncommit a committed expense",
			Method:         http.MethodPost,
			URL:            "/api/expenses/xg2yeucklhgbs3n/uncommit",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"message":"Record uncommitted successfully"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				record, err := app.FindRecordById("expenses", "xg2yeucklhgbs3n")
				if err != nil {
					tb.Fatalf("failed to load expense after uncommit: %v", err)
				}
				if got := record.GetString("committed"); got != "" {
					tb.Fatalf("committed = %q, want blank", got)
				}
				if got := record.GetString("committed_week_ending"); got != "" {
					tb.Fatalf("committed_week_ending = %q, want blank", got)
				}
				if got := record.GetString("committer"); got != "" {
					tb.Fatalf("committer = %q, want blank", got)
				}
				if got := record.GetString("pay_period_ending"); got != "" {
					tb.Fatalf("pay_period_ending = %q, want blank", got)
				}
				if got := record.GetString("approved"); got == "" {
					tb.Fatalf("approved unexpectedly blank after uncommit")
				}
			},
		},
		{
			Name:           "commit holder cannot uncommit a committed expense",
			Method:         http.MethodPost,
			URL:            "/api/expenses/xg2yeucklhgbs3n/uncommit",
			Headers:        map[string]string{"Authorization": commitToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin cannot uncommit an uncommitted expense",
			Method:         http.MethodPost,
			URL:            "/api/expenses/eqhozipupteogp8/uncommit",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"record_not_committed"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "uncommitting a committed expense reopens a closed one-time purchase order",
			Method:         http.MethodPost,
			URL:            "/api/expenses/6569323gg8184uh/uncommit",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"message":"Record uncommitted successfully"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				po, err := app.FindRecordById("purchase_orders", "0pia83nnprdlzf8")
				if err != nil {
					tb.Fatalf("failed to load purchase order after uncommit: %v", err)
				}
				if got := po.GetString("status"); got != "Active" {
					tb.Fatalf("status = %q, want Active", got)
				}
			},
		},
		{
			Name:           "uncommitting a committed expense reopens a closed recurring purchase order when it is no longer exhausted",
			Method:         http.MethodPost,
			URL:            "/api/expenses/3yx4y19k40zun2w/uncommit",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"message":"Record uncommitted successfully"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := setupClosedPurchaseOrderForUncommit(tb, "d8463q483f3da28")
				claim, err := app.FindFirstRecordByFilter("claims", "name = 'admin'")
				if err != nil {
					tb.Fatalf("failed to load admin claim: %v", err)
				}
				_, err = app.NonconcurrentDB().NewQuery(`
					INSERT OR IGNORE INTO user_claims (_imported, cid, created, id, uid, updated)
					VALUES (0, {:cid}, strftime('%Y-%m-%d %H:%M:%fZ', 'now'), {:id}, {:uid}, strftime('%Y-%m-%d %H:%M:%fZ', 'now'))
				`).Bind(dbx.Params{
					"cid": claim.Id,
					"id":  "test_admin_only_expense_claim_u_no_claims",
					"uid": "u_no_claims",
				}).Execute()
				if err != nil {
					tb.Fatalf("failed to grant admin claim in test setup: %v", err)
				}
				return app
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				po, err := app.FindRecordById("purchase_orders", "d8463q483f3da28")
				if err != nil {
					tb.Fatalf("failed to load recurring purchase order after uncommit: %v", err)
				}
				if got := po.GetString("status"); got != "Active" {
					tb.Fatalf("status = %q, want Active", got)
				}
			},
		},
		{
			Name:           "uncommitting a committed expense reopens a closed cumulative purchase order when committed total drops below the po total",
			Method:         http.MethodPost,
			URL:            "/api/expenses/su3hyft6n9rlt7d/uncommit",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"message":"Record uncommitted successfully"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := setupClosedPurchaseOrderForUncommit(tb, "ly8xyzpuj79upq1")
				claim, err := app.FindFirstRecordByFilter("claims", "name = 'admin'")
				if err != nil {
					tb.Fatalf("failed to load admin claim: %v", err)
				}
				_, err = app.NonconcurrentDB().NewQuery(`
					INSERT OR IGNORE INTO user_claims (_imported, cid, created, id, uid, updated)
					VALUES (0, {:cid}, strftime('%Y-%m-%d %H:%M:%fZ', 'now'), {:id}, {:uid}, strftime('%Y-%m-%d %H:%M:%fZ', 'now'))
				`).Bind(dbx.Params{
					"cid": claim.Id,
					"id":  "test_admin_only_expense_claim_u_no_claims",
					"uid": "u_no_claims",
				}).Execute()
				if err != nil {
					tb.Fatalf("failed to grant admin claim in test setup: %v", err)
				}
				return app
			},
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				po, err := app.FindRecordById("purchase_orders", "ly8xyzpuj79upq1")
				if err != nil {
					tb.Fatalf("failed to load cumulative purchase order after uncommit: %v", err)
				}
				if got := po.GetString("status"); got != "Active" {
					tb.Fatalf("status = %q, want Active", got)
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestExpenseAdminDiscoveryRoutes(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "admin-only user can view expense tracking counts",
			Method:         http.MethodGet,
			URL:            "/api/expenses/tracking_counts",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"committed_week_ending":"2024-09-21"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin-only user can view committed expense tracking list",
			Method:         http.MethodGet,
			URL:            "/api/expenses/tracking/2024-09-21",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"xg2yeucklhgbs3n"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin-only user cannot view org-wide expenses list",
			Method:         http.MethodGet,
			URL:            "/api/expenses/list",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"data":[]`,
			},
			NotExpectedContent: []string{
				`"id":"2gq9uyxmkcyopa4"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin-only user cannot view pending expenses list",
			Method:         http.MethodGet,
			URL:            "/api/expenses/pending",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"data":[]`,
			},
			NotExpectedContent: []string{
				`"id":"exp_approve_closed_po_1"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
		{
			Name:           "admin-only user cannot view expense commit queue",
			Method:         http.MethodGet,
			URL:            "/api/expenses/commit_queue",
			Headers:        map[string]string{"Authorization": adminToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
			},
			TestAppFactory: setupAdminOnlyExpenseViewerApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestExpenseDetailsRouteIncludesPOOwnerMismatchWarningData(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "expense details includes po owner mismatch data and names",
		Method:         http.MethodGet,
		URL:            "/api/expenses/details/eqhozipupteogp8",
		Headers:        map[string]string{"Authorization": commitToken},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"uid_name":"Horace Silver"`,
			`"po_uid":"rzr98oadsp9qc11"`,
			`"po_uid_name":"Tester Time"`,
			`"po_owner_uid_mismatch":true`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestExpenseDetailsRouteDoesNotFlagPOOwnerMismatchWhenUIDsMatch(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "expense details does not flag po owner mismatch when owner uids match",
		Method:         http.MethodGet,
		URL:            "/api/expenses/details/3yx4y19k40zun2w",
		Headers:        map[string]string{"Authorization": commitToken},
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"uid_name":"Horace Silver"`,
			`"po_uid":"f2j5a8vk006baub"`,
			`"po_uid_name":"Horace Silver"`,
			`"po_owner_uid_mismatch":false`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestPurchaseOrderExpensesRouteRespectsExpenseVisibility(t *testing.T) {
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "report holder sees committed related expenses on purchase order details route",
			Method:         http.MethodGet,
			URL:            "/api/purchase_orders/visible/0pia83nnprdlzf8/expenses",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"6569323gg8184uh"`,
				`"total":1`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "report holder does not see uncommitted related expenses on purchase order details route",
			Method:         http.MethodGet,
			URL:            "/api/purchase_orders/visible/ly8xyzpuj79upq1/expenses",
			Headers:        map[string]string{"Authorization": reportToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"id":"su3hyft6n9rlt7d"`,
				`"total":1`,
			},
			NotExpectedContent: []string{
				`"id":"eqhozipupteogp8"`,
				`"id":"hlqb5xdzm2xbii7"`,
				`"id":"um1uoad5a4mhfcu"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "commit holder cannot use purchase order details route when purchase order itself is not visible",
			Method:         http.MethodGet,
			URL:            "/api/purchase_orders/visible/0pia83nnprdlzf8/expenses",
			Headers:        map[string]string{"Authorization": commitToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"code":"po_not_found_or_not_visible"`,
			},
			NotExpectedContent: []string{
				`"id":"6569323gg8184uh"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestCalculateMileageTotal verifies the standalone mileage calculation helper using
// the rate tiers effective on 2025-01-05. For a 100 km distance on 2025-01-10 and
// with no prior mileage in the annual period, total should be 100 * 0.70 = 70.00.
func TestCalculateMileageTotal(t *testing.T) {
	// Case 1: No prior mileage, single tier
	t.Run("no prior mileage, single tier", func(t *testing.T) {
		app := testutils.SetupTestApp(t)
		defer app.Cleanup()

		expensesCollection := core.NewCollection("expenses", "expenses")
		record := core.NewRecord(expensesCollection)
		record.Load(map[string]any{
			"date":         "2025-01-10", // selects expense_rates effective 2025-01-05
			"payment_type": "Mileage",
			"distance":     100.0,
		})

		rateRecord, err := utilities.GetExpenseRateRecord(app, record)
		if err != nil {
			t.Fatalf("failed to get expense rate record: %v", err)
		}
		total, err := utilities.CalculateMileageTotal(app, record, rateRecord)
		if err != nil {
			t.Fatalf("unexpected error calculating mileage total: %v", err)
		}
		if total != 70.0 {
			t.Fatalf("expected total 70.0, got %v", total)
		}
	})

	// Case 2: No prior mileage, spans two tiers (5100km => 5000*0.70 + 100*0.64 = 3564)
	t.Run("no prior mileage, spans tiers", func(t *testing.T) {
		app := testutils.SetupTestApp(t)
		defer app.Cleanup()

		expensesCollection := core.NewCollection("expenses", "expenses")
		record := core.NewRecord(expensesCollection)
		record.Load(map[string]any{
			"date":         "2025-01-05", // first day of annual period window (no prior mileage)
			"payment_type": "Mileage",
			"distance":     5100.0,
		})

		rateRecord, err := utilities.GetExpenseRateRecord(app, record)
		if err != nil {
			t.Fatalf("failed to get expense rate record: %v", err)
		}
		total, err := utilities.CalculateMileageTotal(app, record, rateRecord)
		if err != nil {
			t.Fatalf("unexpected error calculating mileage total: %v", err)
		}
		if total != 3564.0 {
			t.Fatalf("expected total 3564.0, got %v", total)
		}
	})

	// Case 3: Prior committed pushes boundary using 2023 fixtures and rates 0.61/0.55.
	// Prior committed mileage 4900 (m2023p4900 on 2023-01-08 set committed), new distance 200 on 2023-01-10
	// => 100 @ 0.61 + 100 @ 0.55 = 116.0
	t.Run("prior committed pushes across tier boundary (fixture 2023)", func(t *testing.T) {
		app := testutils.SetupTestApp(t)
		defer app.Cleanup()

		expensesCollection := core.NewCollection("expenses", "expenses")
		record := core.NewRecord(expensesCollection)
		record.Load(map[string]any{
			"uid":          "uid_mileage_2023_test",
			"date":         "2023-01-10",
			"payment_type": "Mileage",
			"distance":     200.0,
		})

		rateRecord, err := utilities.GetExpenseRateRecord(app, record)
		if err != nil {
			t.Fatalf("failed to get expense rate record: %v", err)
		}
		total, err := utilities.CalculateMileageTotal(app, record, rateRecord)
		if err != nil {
			t.Fatalf("unexpected error calculating mileage total: %v", err)
		}
		if total != 116.0 {
			t.Fatalf("expected total 116.0, got %v", total)
		}
	})

	// Case 4: Prior committed-only included (fixtures m2025u1000, m2025c1000)
	// Prior committed total 1000, new distance 3500: 3500 @ 0.70 = 2450.0
	t.Run("prior committed-only included (fixtures 2025)", func(t *testing.T) {
		app := testutils.SetupTestApp(t)
		defer app.Cleanup()

		expensesCollection := core.NewCollection("expenses", "expenses")
		record := core.NewRecord(expensesCollection)
		record.Load(map[string]any{
			"date":         "2025-01-10",
			"payment_type": "Mileage",
			"distance":     3500.0,
		})

		rateRecord, err := utilities.GetExpenseRateRecord(app, record)
		if err != nil {
			t.Fatalf("failed to get expense rate record: %v", err)
		}
		total, err := utilities.CalculateMileageTotal(app, record, rateRecord)
		if err != nil {
			t.Fatalf("unexpected error calculating mileage total: %v", err)
		}
		if total != 2450.0 {
			t.Fatalf("expected total 2450.0, got %v", total)
		}
	})
}

// TestExpensesCreate_DuplicateAttachmentFails verifies that creating an expense
// with an attachment that has the same hash as an existing expense fails.
func TestExpensesCreate_DuplicateAttachmentFails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		t.Fatalf("failed to load capital kind: %v", err)
	}
	capitalKindID := capitalKind.Id

	// Helper to create multipart form data with a specific file content
	makeMultipartWithContent := func(jsonBody string, fileContent []byte) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		if _, exists := m["kind"]; !exists {
			m["kind"] = capitalKindID
		}
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(fileContent); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	duplicateFileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xDE, 0xAD, 0xBE, 0xEF}

	scenario := tests.ApiScenario{
		Name:           "duplicate attachment fails with field-level error",
		Method:         http.MethodPost,
		URL:            "/api/collections/expenses/records",
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"attachment":{"code":"duplicate_file"`,
			`"message":"This file has already been uploaded to another expense"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordCreateRequest": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Create the request body for the duplicate attempt
	b, ct, err := makeMultipartWithContent(`{
		"uid": "rzr98oadsp9qc11",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "second expense with same attachment",
		"payment_type": "Expense",
		"total": 99,
		"vendor": "2zqxtsmymf670ha"
	}`, duplicateFileContent)
	if err != nil {
		t.Fatal(err)
	}
	scenario.Body = b
	scenario.Headers = map[string]string{"Authorization": recordToken, "Content-Type": ct}

	scenario.Test(t)
}

// TestExpensesUpdate_DuplicateAttachmentFails verifies that updating an expense
// with an attachment that has the same hash as another expense fails.
func TestExpensesUpdate_DuplicateAttachmentFails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		t.Fatalf("failed to load capital kind: %v", err)
	}
	capitalKindID := capitalKind.Id

	// Helper to create multipart form data with a specific file content
	makeMultipartWithContent := func(jsonBody string, fileContent []byte) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		if _, exists := m["kind"]; !exists {
			m["kind"] = capitalKindID
		}
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(fileContent); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	duplicateFileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xCA, 0xFE, 0xBA, 0xBE}

	scenario := tests.ApiScenario{
		Name:           "updating expense with duplicate attachment fails",
		Method:         http.MethodPatch,
		URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"attachment":{"code":"duplicate_file"`,
			`"message":"This file has already been uploaded to another expense"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordUpdateRequest": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Create the request body for the update with duplicate attachment
	b, ct, err := makeMultipartWithContent(`{
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "trying to update with duplicate attachment",
		"payment_type": "Expense",
		"total": 99,
		"vendor": "2zqxtsmymf670ha"
	}`, duplicateFileContent)
	if err != nil {
		t.Fatal(err)
	}
	scenario.Body = b
	scenario.Headers = map[string]string{"Authorization": recordToken, "Content-Type": ct}

	scenario.Test(t)
}

// TestExpensesUpdate_SameAttachmentSucceeds verifies that updating an expense
// with the same attachment file (re-uploading its own file) succeeds.
func TestExpensesUpdate_SameAttachmentSucceeds(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		t.Fatalf("failed to load capital kind: %v", err)
	}
	capitalKindID := capitalKind.Id

	// Helper to create multipart form data with a specific file content
	makeMultipartWithContent := func(jsonBody string, fileContent []byte) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		if _, exists := m["kind"]; !exists {
			m["kind"] = capitalKindID
		}
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		if _, err := fw.Write(fileContent); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	fileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x12, 0x34, 0x56, 0x78}
	// The seeded expense fixture already has the same attachment hash as fileContent.
	scenario := tests.ApiScenario{
		Name:           "updating expense with its own attachment succeeds",
		Method:         http.MethodPatch,
		URL:            "/api/collections/expenses/records/exp_same_attach_target_1",
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"description":"updated description"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordUpdate": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Create the request body with the same file content
	b, ct, err := makeMultipartWithContent(`{
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "updated description",
		"payment_type": "Expense",
		"total": 99,
		"vendor": "2zqxtsmymf670ha"
	}`, fileContent)
	if err != nil {
		t.Fatal(err)
	}
	scenario.Body = b
	scenario.Headers = map[string]string{"Authorization": recordToken, "Content-Type": ct}

	scenario.Test(t)
}

// TestExpensesCreate_InactiveApproverFails verifies that creating an expense
// fails when the user's manager (who becomes the approver) is inactive.
func TestExpensesCreate_InactiveApproverFails(t *testing.T) {
	// User has_inactive_mgr@test.com has a profile with manager = u_inactive
	recordToken, err := testutils.GenerateRecordToken("users", "has_inactive_mgr@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	capitalKind, err := app.FindFirstRecordByFilter("expenditure_kinds", "name = 'capital'")
	if err != nil {
		t.Fatalf("failed to load capital kind: %v", err)
	}
	capitalKindID := capitalKind.Id

	// multipart builder for creates with attachment
	makeMultipart := func(jsonBody string) (*bytes.Buffer, string, error) {
		m := map[string]any{}
		if err := json.Unmarshal([]byte(jsonBody), &m); err != nil {
			return nil, "", err
		}
		if _, exists := m["kind"]; !exists {
			m["kind"] = capitalKindID
		}
		buf := &bytes.Buffer{}
		w := multipart.NewWriter(buf)
		for k, v := range m {
			if err := w.WriteField(k, fmt.Sprint(v)); err != nil {
				return nil, "", err
			}
		}
		fw, err := w.CreateFormFile("attachment", "receipt.png")
		if err != nil {
			return nil, "", err
		}
		// Minimal PNG header so mime detection passes (image/png)
		if _, err := fw.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
			return nil, "", err
		}
		contentType := w.FormDataContentType()
		if err := w.Close(); err != nil {
			return nil, "", err
		}
		return buf, contentType, nil
	}

	b, ct, err := makeMultipart(`{
		"uid": "u_has_inactive_mgr",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "test expense with inactive manager",
		"payment_type": "Expense",
		"total": 99,
		"vendor": "2zqxtsmymf670ha"
	}`)
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:           "expense create fails when manager is inactive",
		Method:         http.MethodPost,
		URL:            "/api/collections/expenses/records",
		Body:           b,
		Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
		ExpectedStatus: 400,
		ExpectedContent: []string{
			`"approver":{"code":"approver_not_active"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordCreateRequest": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}
