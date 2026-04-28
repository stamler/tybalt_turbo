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
	"time"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

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
		TestAppFactory: setupTestAppWithSynchronousImmediateNotifications,
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

func TestBookKeeperPurchaseOrderExpenseFlow(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	ownerToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	hasBookKeeperClaim, err := utilities.HasClaimByUserID(app, "tqqf7q0f3378rvp", "book_keeper")
	if err != nil {
		t.Fatalf("failed checking book_keeper claim: %v", err)
	}
	if !hasBookKeeperClaim {
		t.Fatal("book@keeper.com fixture must have book_keeper claim")
	}
	body, contentType := mustMultipartExpense(t, map[string]string{
		"uid":            "rzr98oadsp9qc11",
		"date":           "2024-09-01",
		"division":       "vccd5fo56ctbigh",
		"description":    "bookkeeper entered invoice",
		"payment_type":   "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job":            "cjf0kt0defhq480",
		"category":       "t5nmdl188gtlhz0",
		"kind":           "prj0kind0000001",
		"total":          "25",
		"vendor":         "z66xe6vqhwtokt4",
	}, "bookkeeper-flow.png")
	createRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, createRes, http.StatusOK)

	var created struct {
		ID       string `json:"id"`
		UID      string `json:"uid"`
		Creator  string `json:"creator"`
		Approver string `json:"approver"`
	}
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v; body=%s", err, createRes.Body.String())
	}
	if created.UID != "rzr98oadsp9qc11" || created.Creator != "tqqf7q0f3378rvp" || created.Approver != "tqqf7q0f3378rvp" {
		t.Fatalf("unexpected actor fields after create: %+v", created)
	}

	updateRes := performTestAPIRequest(t, app, http.MethodPatch, "/api/collections/expenses/records/"+created.ID, strings.NewReader(`{
		"uid": "rzr98oadsp9qc11",
		"creator": "tqqf7q0f3378rvp",
		"date": "2024-09-01",
		"division":       "vccd5fo56ctbigh",
		"description": "bookkeeper updated invoice",
		"payment_type": "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job":            "cjf0kt0defhq480",
		"category":       "t5nmdl188gtlhz0",
		"kind": "prj0kind0000001",
		"total": 25,
		"vendor": "z66xe6vqhwtokt4"
	}`), map[string]string{"Authorization": bookkeeperToken})
	mustStatus(t, updateRes, http.StatusOK)

	badPaymentUpdateRes := performTestAPIRequest(t, app, http.MethodPatch, "/api/collections/expenses/records/"+created.ID, strings.NewReader(`{
		"uid": "rzr98oadsp9qc11",
		"creator": "tqqf7q0f3378rvp",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "bookkeeper bad payment update",
		"payment_type": "CorporateCreditCard",
		"cc_last_4_digits": "1234",
		"purchase_order": "poa1ctvbrnch001",
		"job": "cjf0kt0defhq480",
		"category": "t5nmdl188gtlhz0",
		"kind": "prj0kind0000001",
		"total": 25,
		"vendor": "z66xe6vqhwtokt4"
	}`), map[string]string{"Authorization": bookkeeperToken})
	mustStatus(t, badPaymentUpdateRes, http.StatusBadRequest)

	ownerSubmitRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/submit", nil, map[string]string{
		"Authorization": ownerToken,
	})
	mustStatus(t, ownerSubmitRes, http.StatusForbidden)

	submitRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/submit", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, submitRes, http.StatusOK)

	pendingRes := performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/pending", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, pendingRes, http.StatusOK)
	if !strings.Contains(pendingRes.Body.String(), created.ID) {
		t.Fatalf("expected pending queue to include bookkeeper expense %s; body=%s", created.ID, pendingRes.Body.String())
	}

	ownerApproveRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/approve", nil, map[string]string{
		"Authorization": ownerToken,
	})
	mustStatus(t, ownerApproveRes, http.StatusForbidden)

	approveRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/approve", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, approveRes, http.StatusOK)

	body, contentType = mustMultipartExpense(t, map[string]string{
		"uid":            "rzr98oadsp9qc11",
		"date":           "2024-09-01",
		"division":       "vccd5fo56ctbigh",
		"description":    "non bookkeeper invoice",
		"payment_type":   "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job":            "cjf0kt0defhq480",
		"category":       "t5nmdl188gtlhz0",
		"kind":           "prj0kind0000001",
		"total":          "25",
		"vendor":         "z66xe6vqhwtokt4",
	}, "non-bookkeeper-flow.png")
	noClaimCreateRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": noClaimsToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, noClaimCreateRes, http.StatusBadRequest)

	body, contentType = mustMultipartExpense(t, map[string]string{
		"uid":              "rzr98oadsp9qc11",
		"date":             "2024-09-01",
		"division":         "vccd5fo56ctbigh",
		"description":      "payment mismatch invoice",
		"payment_type":     "CorporateCreditCard",
		"cc_last_4_digits": "1234",
		"purchase_order":   "poa1ctvbrnch001",
		"job":              "cjf0kt0defhq480",
		"category":         "t5nmdl188gtlhz0",
		"kind":             "prj0kind0000001",
		"total":            "25",
		"vendor":           "z66xe6vqhwtokt4",
	}, "payment-mismatch-flow.png")
	mismatchRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, mismatchRes, http.StatusBadRequest)
}

func TestBookKeeperSameUserPurchaseOrderUsesRegularApproval(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	activatePurchaseOrderFixtures(t, app, "bkpoownacct01")

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	managerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	fields := bookKeeperExpenseFields(map[string]string{
		"uid":            "tqqf7q0f3378rvp",
		"date":           "2026-04-28",
		"description":    "same-user bookkeeper invoice",
		"purchase_order": "bkpoownacct01",
	})
	body, contentType := mustMultipartExpense(t, fields, "bookkeeper-own-po.png")
	createRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, createRes, http.StatusOK)

	var created bookKeeperExpenseCreateResult
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v; body=%s", err, createRes.Body.String())
	}
	if created.UID != "tqqf7q0f3378rvp" || created.Creator != "tqqf7q0f3378rvp" || created.Approver != "wegviunlyr2jjjv" {
		t.Fatalf("same-user bookkeeper PO should use regular manager approval, got %+v", created)
	}

	submitRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/submit", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, submitRes, http.StatusOK)

	bookkeeperApproveRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/approve", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, bookkeeperApproveRes, http.StatusForbidden)

	managerApproveRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/approve", nil, map[string]string{
		"Authorization": managerToken,
	})
	mustStatus(t, managerApproveRes, http.StatusOK)
}

func TestBookKeeperPurchaseOrderExpenseCorporateCreditCardFlow(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	activatePurchaseOrderFixtures(t, app, "bkpootherccc1")

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	fields := bookKeeperExpenseFields(map[string]string{
		"description":      "bookkeeper corporate card invoice",
		"date":             "2026-04-28",
		"payment_type":     "CorporateCreditCard",
		"purchase_order":   "bkpootherccc1",
		"cc_last_4_digits": "1234",
	})
	body, contentType := mustMultipartExpense(t, fields, "bookkeeper-corporate-card.png")
	createRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, createRes, http.StatusOK)

	var created bookKeeperExpenseCreateResult
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v; body=%s", err, createRes.Body.String())
	}
	if created.UID != "rzr98oadsp9qc11" || created.Creator != "tqqf7q0f3378rvp" || created.Approver != "tqqf7q0f3378rvp" {
		t.Fatalf("unexpected actor fields after corporate card create: %+v", created)
	}
}

func TestBookKeeperPurchaseOrderExpenseRejectsIneligibleCreates(t *testing.T) {
	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		overrides   map[string]string
		activePOIDs []string
		want        string
	}{
		{
			name:      "without purchase order",
			overrides: map[string]string{"purchase_order": ""},
		},
		{
			name:      "uid does not match purchase order owner",
			overrides: map[string]string{"uid": "f2j5a8vk006baub"},
			want:      "must_match_purchase_order_owner",
		},
		{
			name:      "closed purchase order",
			overrides: map[string]string{"purchase_order": "exp_closed_po_1"},
			want:      "not_active",
		},
		{
			name:      "cancelled purchase order",
			overrides: map[string]string{"uid": "4ssj9f1yg250o9y", "purchase_order": "1cqrvp4mna33k2b"},
			want:      "not_active",
		},
		{
			name:      "unapproved purchase order",
			overrides: map[string]string{"purchase_order": "ponactvbrnch001"},
			want:      "not_active",
		},
		{
			name:      "unsupported payment type",
			overrides: map[string]string{"payment_type": "Mileage"},
		},
		{
			name:        "caller cannot use purchase order branch",
			overrides:   map[string]string{"date": "2026-04-28", "purchase_order": "bkpocorpbr01"},
			activePOIDs: []string{"bkpocorpbr01"},
			want:        "branch_claim_required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			t.Cleanup(app.Cleanup)
			activatePurchaseOrderFixtures(t, app, tt.activePOIDs...)

			body, contentType := mustMultipartExpense(t, bookKeeperExpenseFields(tt.overrides), strings.ReplaceAll(tt.name, " ", "-")+".png")
			res := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
				"Authorization": bookkeeperToken,
				"Content-Type":  contentType,
			})
			mustStatus(t, res, http.StatusBadRequest)
			if tt.want != "" && !strings.Contains(res.Body.String(), tt.want) {
				t.Fatalf("expected response to contain %q; body=%s", tt.want, res.Body.String())
			}
		})
	}
}

func TestBookKeeperExpenseCannotBeApprovedByUnrelatedBookKeeper(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	unrelatedBookkeeperToken, err := testutils.GenerateRecordToken("users", "corp.claim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	created := createBookKeeperPOExpense(t, app, bookkeeperToken, "bookkeeper unrelated approval invoice", "bookkeeper-unrelated-approval.png")
	submitRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/submit", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, submitRes, http.StatusOK)

	approveRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/approve", nil, map[string]string{
		"Authorization": unrelatedBookkeeperToken,
	})
	mustStatus(t, approveRes, http.StatusForbidden)
}

func TestBookKeeperExpenseEditRejectsOverwrittenOwnerPayload(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	body, contentType := mustMultipartExpense(t, map[string]string{
		"uid":            "rzr98oadsp9qc11",
		"date":           "2024-09-01",
		"division":       "vccd5fo56ctbigh",
		"description":    "bookkeeper entered invoice",
		"payment_type":   "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job":            "cjf0kt0defhq480",
		"category":       "t5nmdl188gtlhz0",
		"kind":           "prj0kind0000001",
		"total":          "25",
		"vendor":         "z66xe6vqhwtokt4",
	}, "bookkeeper-edit-payload.png")
	createRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, createRes, http.StatusOK)

	var created struct {
		ID      string `json:"id"`
		UID     string `json:"uid"`
		Creator string `json:"creator"`
	}
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v; body=%s", err, createRes.Body.String())
	}
	if created.UID == created.Creator {
		t.Fatalf("fixture setup must create an on-behalf expense, got owner fields %+v", created)
	}

	updateRes := performTestAPIRequest(t, app, http.MethodPatch, "/api/collections/expenses/records/"+created.ID, strings.NewReader(`{
		"uid": "tqqf7q0f3378rvp",
		"creator": "tqqf7q0f3378rvp",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "bookkeeper updated invoice",
		"payment_type": "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job": "cjf0kt0defhq480",
		"category": "t5nmdl188gtlhz0",
		"kind": "prj0kind0000001",
		"total": 25,
		"vendor": "z66xe6vqhwtokt4"
	}`), map[string]string{"Authorization": bookkeeperToken})
	if updateRes.Code == http.StatusOK {
		t.Fatalf("expected overwritten uid payload to be rejected, got status %d; body=%s", updateRes.Code, updateRes.Body.String())
	}

	reloaded, err := app.FindRecordById("expenses", created.ID)
	if err != nil {
		t.Fatalf("failed to reload expense: %v", err)
	}
	if got := reloaded.GetString("uid"); got != created.UID {
		t.Fatalf("expense uid changed after rejected update: got %q, want %q", got, created.UID)
	}
}

func TestBookKeeperExpenseDraftCannotBeDeletedByPurchaseOrderOwner(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	ownerToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	body, contentType := mustMultipartExpense(t, map[string]string{
		"uid":            "rzr98oadsp9qc11",
		"date":           "2024-09-01",
		"division":       "vccd5fo56ctbigh",
		"description":    "bookkeeper entered invoice for delete regression",
		"payment_type":   "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job":            "cjf0kt0defhq480",
		"category":       "t5nmdl188gtlhz0",
		"kind":           "prj0kind0000001",
		"total":          "25",
		"vendor":         "z66xe6vqhwtokt4",
	}, "bookkeeper-delete-regression.png")
	createRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, createRes, http.StatusOK)

	var created struct {
		ID      string `json:"id"`
		UID     string `json:"uid"`
		Creator string `json:"creator"`
	}
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v; body=%s", err, createRes.Body.String())
	}
	if created.UID != "rzr98oadsp9qc11" || created.Creator != "tqqf7q0f3378rvp" {
		t.Fatalf("fixture setup must create an on-behalf expense, got owner fields %+v", created)
	}

	ownerDeleteRes := performTestAPIRequest(t, app, http.MethodDelete, "/api/collections/expenses/records/"+created.ID, nil, map[string]string{
		"Authorization": ownerToken,
	})
	if ownerDeleteRes.Code == http.StatusNoContent {
		t.Fatalf("purchase order owner deleted a bookkeeper-created draft; body=%s", ownerDeleteRes.Body.String())
	}

	if _, err := app.FindRecordById("expenses", created.ID); err != nil {
		t.Fatalf("bookkeeper-created draft was removed by rejected owner delete: %v", err)
	}

	bookkeeperDeleteRes := performTestAPIRequest(t, app, http.MethodDelete, "/api/collections/expenses/records/"+created.ID, nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, bookkeeperDeleteRes, http.StatusNoContent)
}

func TestBookKeeperExpenseVisibilityIncludesCreatorAndOwner(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	ownerToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	created := createBookKeeperPOExpense(t, app, bookkeeperToken, "bookkeeper visibility invoice", "bookkeeper-visibility.png")

	for _, tt := range []struct {
		name  string
		token string
	}{
		{name: "creator", token: bookkeeperToken},
		{name: "purchase order owner", token: ownerToken},
	} {
		t.Run(tt.name+" sees draft in custom list", func(t *testing.T) {
			listRes := performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/list", nil, map[string]string{
				"Authorization": tt.token,
			})
			mustStatus(t, listRes, http.StatusOK)
			body := listRes.Body.String()
			for _, want := range []string{
				`"id":"` + created.ID + `"`,
				`"uid":"rzr98oadsp9qc11"`,
				`"creator":"tqqf7q0f3378rvp"`,
				`"creator_name":"Ultra Chifres"`,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("expected list response to contain %s; body=%s", want, body)
				}
			}
		})

		t.Run(tt.name+" sees draft details", func(t *testing.T) {
			detailsRes := performTestAPIRequest(t, app, http.MethodGet, "/api/expenses/details/"+created.ID, nil, map[string]string{
				"Authorization": tt.token,
			})
			mustStatus(t, detailsRes, http.StatusOK)
			body := detailsRes.Body.String()
			for _, want := range []string{
				`"id":"` + created.ID + `"`,
				`"uid":"rzr98oadsp9qc11"`,
				`"creator":"tqqf7q0f3378rvp"`,
				`"uid_name":"Tester Time"`,
				`"creator_name":"Ultra Chifres"`,
				`"po_owner_uid_mismatch":false`,
			} {
				if !strings.Contains(body, want) {
					t.Fatalf("expected details response to contain %s; body=%s", want, body)
				}
			}
		})
	}
}

func TestBookKeeperExpenseRecallUsesCreator(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	ownerToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	created := createBookKeeperPOExpense(t, app, bookkeeperToken, "bookkeeper recall invoice", "bookkeeper-recall.png")

	submitRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/submit", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, submitRes, http.StatusOK)

	ownerRecallRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/recall", nil, map[string]string{
		"Authorization": ownerToken,
	})
	mustStatus(t, ownerRecallRes, http.StatusForbidden)

	reloaded, err := app.FindRecordById("expenses", created.ID)
	if err != nil {
		t.Fatalf("failed to reload expense: %v", err)
	}
	if !reloaded.GetBool("submitted") {
		t.Fatal("expense was recalled by purchase order owner")
	}

	bookkeeperRecallRes := performTestAPIRequest(t, app, http.MethodPost, "/api/expenses/"+created.ID+"/recall", nil, map[string]string{
		"Authorization": bookkeeperToken,
	})
	mustStatus(t, bookkeeperRecallRes, http.StatusOK)

	reloaded, err = app.FindRecordById("expenses", created.ID)
	if err != nil {
		t.Fatalf("failed to reload expense after bookkeeper recall: %v", err)
	}
	if reloaded.GetBool("submitted") {
		t.Fatal("expense remained submitted after creator recall")
	}
}

func TestBookKeeperExpenseEditRejectsCreatorChange(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	bookkeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	created := createBookKeeperPOExpense(t, app, bookkeeperToken, "bookkeeper creator immutable invoice", "bookkeeper-creator-immutable.png")
	updateRes := performTestAPIRequest(t, app, http.MethodPatch, "/api/collections/expenses/records/"+created.ID, strings.NewReader(`{
		"uid": "rzr98oadsp9qc11",
		"creator": "rzr98oadsp9qc11",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "bookkeeper changed creator invoice",
		"payment_type": "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job": "cjf0kt0defhq480",
		"category": "t5nmdl188gtlhz0",
		"kind": "prj0kind0000001",
		"total": 25,
		"vendor": "z66xe6vqhwtokt4"
	}`), map[string]string{"Authorization": bookkeeperToken})
	if updateRes.Code == http.StatusOK {
		t.Fatalf("expected creator change payload to be rejected, got status %d; body=%s", updateRes.Code, updateRes.Body.String())
	}

	reloaded, err := app.FindRecordById("expenses", created.ID)
	if err != nil {
		t.Fatalf("failed to reload expense: %v", err)
	}
	if got := reloaded.GetString("creator"); got != created.Creator {
		t.Fatalf("expense creator changed after rejected update: got %q, want %q", got, created.Creator)
	}
}

type bookKeeperExpenseCreateResult struct {
	ID       string `json:"id"`
	UID      string `json:"uid"`
	Creator  string `json:"creator"`
	Approver string `json:"approver"`
}

func bookKeeperExpenseFields(overrides map[string]string) map[string]string {
	fields := map[string]string{
		"uid":            "rzr98oadsp9qc11",
		"date":           "2024-09-01",
		"division":       "vccd5fo56ctbigh",
		"description":    "bookkeeper entered invoice",
		"payment_type":   "OnAccount",
		"purchase_order": "poa1ctvbrnch001",
		"job":            "cjf0kt0defhq480",
		"category":       "t5nmdl188gtlhz0",
		"kind":           "prj0kind0000001",
		"total":          "25",
		"vendor":         "z66xe6vqhwtokt4",
	}
	for key, value := range overrides {
		fields[key] = value
	}
	return fields
}

func activatePurchaseOrderFixtures(t testing.TB, app *tests.TestApp, ids ...string) {
	t.Helper()

	// These fixtures are stored as Unapproved so global active-PO visibility counts
	// remain stable. Tests that need a narrowly eligible PO activate only their
	// private app copy.
	for _, id := range ids {
		record, err := app.FindRecordById("purchase_orders", id)
		if err != nil {
			t.Fatalf("failed to load purchase order fixture %s: %v", id, err)
		}
		record.Set("status", "Active")
		if err := app.Save(record); err != nil {
			t.Fatalf("failed to activate purchase order fixture %s: %v", id, err)
		}
	}
}

func createBookKeeperPOExpense(t testing.TB, app *tests.TestApp, bookkeeperToken string, description string, filename string) bookKeeperExpenseCreateResult {
	t.Helper()

	body, contentType := mustMultipartExpense(t, bookKeeperExpenseFields(map[string]string{"description": description}), filename)
	createRes := performTestAPIRequest(t, app, http.MethodPost, "/api/collections/expenses/records", body, map[string]string{
		"Authorization": bookkeeperToken,
		"Content-Type":  contentType,
	})
	mustStatus(t, createRes, http.StatusOK)

	var created bookKeeperExpenseCreateResult
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v; body=%s", err, createRes.Body.String())
	}
	if created.UID != "rzr98oadsp9qc11" || created.Creator != "tqqf7q0f3378rvp" || created.Approver != "tqqf7q0f3378rvp" {
		t.Fatalf("unexpected actor fields after create: %+v", created)
	}
	return created
}

func mustMultipartExpense(t testing.TB, fields map[string]string, filename string) (*bytes.Buffer, string) {
	t.Helper()

	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			t.Fatalf("failed to write multipart field %s: %v", k, err)
		}
	}
	fw, err := w.CreateFormFile("attachment", filename)
	if err != nil {
		t.Fatalf("failed to create attachment field: %v", err)
	}
	if _, err := fw.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); err != nil {
		t.Fatalf("failed to write attachment: %v", err)
	}
	contentType := w.FormDataContentType()
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}
	return buf, contentType
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
	adminToken, err := testutils.GenerateRecordToken("users", "admin.only@example.com")
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
				return setupClosedPurchaseOrderForUncommit(tb, "d8463q483f3da28")
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
				return setupClosedPurchaseOrderForUncommit(tb, "ly8xyzpuj79upq1")
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
	adminToken, err := testutils.GenerateRecordToken("users", "admin.only@example.com")
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
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
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

const (
	expensesListMixedPaginationOwnerEmail      = "admin.only@example.com"
	expensesListMixedPaginationOwnerID         = "u_admin_only"
	expensesListMixedPaginationApproverID      = "f2j5a8vk006baub"
	expensesListMixedPaginationDivisionID      = "vccd5fo56ctbigh"
	expensesListMixedPaginationKindID          = "prj0kind0000001"
	expensesListMixedPaginationPurchaseOrderID = "ly8xyzpuj79upq1"
	expensesListMixedPaginationCommittedCount  = 55
	expensesListMixedPaginationOpenCount       = 4
)

type paginatedExpensesListRow struct {
	ID            string `json:"id"`
	Committed     string `json:"committed"`
	PurchaseOrder string `json:"purchase_order"`
}

type paginatedExpensesListResponse struct {
	Data       []paginatedExpensesListRow `json:"data"`
	Page       int                        `json:"page"`
	Limit      int                        `json:"limit"`
	Total      int                        `json:"total"`
	TotalPages int                        `json:"total_pages"`
}

func decodePaginatedExpensesListResponse(t testing.TB, res *http.Response) paginatedExpensesListResponse {
	t.Helper()
	defer res.Body.Close()

	var decoded paginatedExpensesListResponse
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		t.Fatalf("failed decoding paginated expenses list response: %v", err)
	}

	return decoded
}

func mixedPaginationOpenExpenseID(index int) string {
	return fmt.Sprintf("mxlopen%03d", index)
}

func mixedPaginationCommittedExpenseID(index int) string {
	return fmt.Sprintf("mxlcomm%03d", index)
}

func mixedPaginationTimestamp(ts time.Time) string {
	return ts.UTC().Format("2006-01-02 15:04:05.000Z")
}

func expectedMixedPaginationPage1IDs() []string {
	ids := make([]string, 0, expensesListMixedPaginationOpenCount+50)
	for i := expensesListMixedPaginationOpenCount - 1; i >= 0; i-- {
		ids = append(ids, mixedPaginationOpenExpenseID(i))
	}
	for i := expensesListMixedPaginationCommittedCount - 1; i >= 5; i-- {
		ids = append(ids, mixedPaginationCommittedExpenseID(i))
	}
	return ids
}

func expectedMixedPaginationPage2IDs() []string {
	ids := make([]string, 0, 5)
	for i := 4; i >= 0; i-- {
		ids = append(ids, mixedPaginationCommittedExpenseID(i))
	}
	return ids
}

func expectedMixedPaginationPurchaseOrderPage1IDs() []string {
	ids := []string{
		mixedPaginationOpenExpenseID(1),
		mixedPaginationOpenExpenseID(0),
	}
	for i := expensesListMixedPaginationCommittedCount - 1; i >= 5; i-- {
		ids = append(ids, mixedPaginationCommittedExpenseID(i))
	}
	return ids
}

func seedExpensesListMixedPaginationFixture(tb testing.TB, app *tests.TestApp) {
	tb.Helper()

	insertExpenseQuery := `
		INSERT INTO expenses (
			id,
			uid,
			date,
			division,
			description,
			total,
			payment_type,
			attachment,
			attachment_hash,
			rejector,
			rejected,
			rejection_reason,
			approver,
			approved,
			job,
			category,
			kind,
			pay_period_ending,
			allowance_types,
			submitted,
			committer,
			committed,
			committed_week_ending,
			distance,
			cc_last_4_digits,
			currency,
			settled_total,
			settler,
			settled,
			purchase_order,
			vendor,
			branch,
			created,
			updated
		) VALUES (
			{:id},
			{:uid},
			{:date},
			{:division},
			{:description},
			{:total},
			{:payment_type},
			{:attachment},
			{:attachment_hash},
			{:rejector},
			{:rejected},
			{:rejection_reason},
			{:approver},
			{:approved},
			{:job},
			{:category},
			{:kind},
			{:pay_period_ending},
			{:allowance_types},
			{:submitted},
			{:committer},
			{:committed},
			{:committed_week_ending},
			{:distance},
			{:cc_last_4_digits},
			{:currency},
			{:settled_total},
			{:settler},
			{:settled},
			{:purchase_order},
			{:vendor},
			{:branch},
			{:created},
			{:updated}
		)
	`

	insertExpense := func(params dbx.Params) {
		if _, err := app.NonconcurrentDB().NewQuery(insertExpenseQuery).Bind(params).Execute(); err != nil {
			tb.Fatalf("failed seeding mixed pagination expense %q: %v", params["id"], err)
		}
	}

	baseCreated := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)

	for i := 0; i < expensesListMixedPaginationOpenCount; i++ {
		purchaseOrder := ""
		if i < 2 {
			purchaseOrder = expensesListMixedPaginationPurchaseOrderID
		}

		createdAt := baseCreated.Add(time.Duration(i) * time.Minute)
		expenseDate := time.Date(2026, 5, 1+i, 0, 0, 0, 0, time.UTC)
		insertExpense(dbx.Params{
			"id":                    mixedPaginationOpenExpenseID(i),
			"uid":                   expensesListMixedPaginationOwnerID,
			"date":                  expenseDate.Format(time.DateOnly),
			"division":              expensesListMixedPaginationDivisionID,
			"description":           fmt.Sprintf("Mixed pagination open expense %02d", i),
			"total":                 100 + i,
			"payment_type":          "OnAccount",
			"attachment":            "",
			"attachment_hash":       "",
			"rejector":              "",
			"rejected":              "",
			"rejection_reason":      "",
			"approver":              expensesListMixedPaginationApproverID,
			"approved":              "",
			"job":                   "",
			"category":              "",
			"kind":                  expensesListMixedPaginationKindID,
			"pay_period_ending":     "",
			"allowance_types":       "[]",
			"submitted":             0,
			"committer":             "",
			"committed":             "",
			"committed_week_ending": "",
			"distance":              0,
			"cc_last_4_digits":      "",
			"currency":              "",
			"settled_total":         0,
			"settler":               "",
			"settled":               "",
			"purchase_order":        purchaseOrder,
			"vendor":                "",
			"branch":                "",
			"created":               mixedPaginationTimestamp(createdAt),
			"updated":               mixedPaginationTimestamp(createdAt),
		})
	}

	committedBase := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)
	for i := 0; i < expensesListMixedPaginationCommittedCount; i++ {
		purchaseOrder := expensesListMixedPaginationPurchaseOrderID
		if i < 2 {
			purchaseOrder = ""
		}

		approvedAt := committedBase.Add(time.Duration(i) * time.Hour)
		committedAt := approvedAt.Add(15 * time.Minute)
		committedDate := committedAt.Format(time.DateOnly)
		committedWeekEnding, err := utilities.GenerateWeekEnding(committedDate)
		if err != nil {
			tb.Fatalf("failed generating committed week ending for mixed pagination fixture: %v", err)
		}

		expenseDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		createdAt := baseCreated.Add(24*time.Hour + time.Duration(i)*time.Minute)
		insertExpense(dbx.Params{
			"id":                    mixedPaginationCommittedExpenseID(i),
			"uid":                   expensesListMixedPaginationOwnerID,
			"date":                  expenseDate.Format(time.DateOnly),
			"division":              expensesListMixedPaginationDivisionID,
			"description":           fmt.Sprintf("Mixed pagination committed expense %02d", i),
			"total":                 200 + i,
			"payment_type":          "OnAccount",
			"attachment":            "",
			"attachment_hash":       "",
			"rejector":              "",
			"rejected":              "",
			"rejection_reason":      "",
			"approver":              expensesListMixedPaginationApproverID,
			"approved":              mixedPaginationTimestamp(approvedAt),
			"job":                   "",
			"category":              "",
			"kind":                  expensesListMixedPaginationKindID,
			"pay_period_ending":     committedWeekEnding,
			"allowance_types":       "[]",
			"submitted":             1,
			"committer":             expensesListMixedPaginationApproverID,
			"committed":             mixedPaginationTimestamp(committedAt),
			"committed_week_ending": committedWeekEnding,
			"distance":              0,
			"cc_last_4_digits":      "",
			"currency":              "",
			"settled_total":         0,
			"settler":               "",
			"settled":               "",
			"purchase_order":        purchaseOrder,
			"vendor":                "",
			"branch":                "",
			"created":               mixedPaginationTimestamp(createdAt),
			"updated":               mixedPaginationTimestamp(createdAt),
		})
	}
}

func TestExpensesListMixedPagination(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", expensesListMixedPaginationOwnerEmail)
	if err != nil {
		t.Fatal(err)
	}

	makeSeededApp := func(tb testing.TB) *tests.TestApp {
		app := testutils.SetupTestApp(tb)
		seedExpensesListMixedPaginationFixture(tb, app)
		return app
	}

	assertRowIDs := func(tb testing.TB, rows []paginatedExpensesListRow, wantIDs []string) {
		tb.Helper()

		if len(rows) != len(wantIDs) {
			tb.Fatalf("row count = %d, want %d", len(rows), len(wantIDs))
		}

		for i, wantID := range wantIDs {
			if got := rows[i].ID; got != wantID {
				tb.Fatalf("row %d id = %q, want %q", i, got, wantID)
			}
		}
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "expenses list page 1 shows all open expenses then 50 newest committed expenses",
			Method:         http.MethodGet,
			URL:            "/api/expenses/list",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"page":1`,
			},
			TestAppFactory: makeSeededApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				decoded := decodePaginatedExpensesListResponse(tb, res)
				wantIDs := expectedMixedPaginationPage1IDs()

				if decoded.Page != 1 {
					tb.Fatalf("page = %d, want 1", decoded.Page)
				}
				if decoded.Limit != 50 {
					tb.Fatalf("limit = %d, want 50", decoded.Limit)
				}
				if decoded.Total != expensesListMixedPaginationOpenCount+expensesListMixedPaginationCommittedCount {
					tb.Fatalf("total = %d, want %d", decoded.Total, expensesListMixedPaginationOpenCount+expensesListMixedPaginationCommittedCount)
				}
				if decoded.TotalPages != 2 {
					tb.Fatalf("total_pages = %d, want 2", decoded.TotalPages)
				}

				assertRowIDs(tb, decoded.Data, wantIDs)

				for i := 0; i < expensesListMixedPaginationOpenCount; i++ {
					if decoded.Data[i].Committed != "" {
						tb.Fatalf("row %d committed = %q, want blank for non-committed expense", i, decoded.Data[i].Committed)
					}
				}
				for i := expensesListMixedPaginationOpenCount; i < len(decoded.Data); i++ {
					if decoded.Data[i].Committed == "" {
						tb.Fatalf("row %d committed unexpectedly blank for committed expense", i)
					}
				}
			},
		},
		{
			Name:           "expenses list page 2 shows remaining committed overflow only",
			Method:         http.MethodGet,
			URL:            "/api/expenses/list?page=2",
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"page":2`,
			},
			TestAppFactory: makeSeededApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				decoded := decodePaginatedExpensesListResponse(tb, res)
				wantIDs := expectedMixedPaginationPage2IDs()

				if decoded.Page != 2 {
					tb.Fatalf("page = %d, want 2", decoded.Page)
				}
				if decoded.Limit != 50 {
					tb.Fatalf("limit = %d, want 50", decoded.Limit)
				}
				if decoded.Total != expensesListMixedPaginationOpenCount+expensesListMixedPaginationCommittedCount {
					tb.Fatalf("total = %d, want %d", decoded.Total, expensesListMixedPaginationOpenCount+expensesListMixedPaginationCommittedCount)
				}
				if decoded.TotalPages != 2 {
					tb.Fatalf("total_pages = %d, want 2", decoded.TotalPages)
				}

				assertRowIDs(tb, decoded.Data, wantIDs)

				for i, row := range decoded.Data {
					if row.Committed == "" {
						tb.Fatalf("page 2 row %d committed unexpectedly blank", i)
					}
				}
			},
		},
		{
			Name:           "expenses list purchase order filter applies before mixed pagination buckets",
			Method:         http.MethodGet,
			URL:            "/api/expenses/list?purchase_order=" + expensesListMixedPaginationPurchaseOrderID,
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"page":1`,
			},
			TestAppFactory: makeSeededApp,
			AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
				decoded := decodePaginatedExpensesListResponse(tb, res)
				wantIDs := expectedMixedPaginationPurchaseOrderPage1IDs()

				if decoded.Page != 1 {
					tb.Fatalf("page = %d, want 1", decoded.Page)
				}
				if decoded.Limit != 50 {
					tb.Fatalf("limit = %d, want 50", decoded.Limit)
				}
				if decoded.Total != 55 {
					tb.Fatalf("total = %d, want 55", decoded.Total)
				}
				if decoded.TotalPages != 2 {
					tb.Fatalf("total_pages = %d, want 2", decoded.TotalPages)
				}

				assertRowIDs(tb, decoded.Data, wantIDs)

				for i, row := range decoded.Data {
					if row.PurchaseOrder != expensesListMixedPaginationPurchaseOrderID {
						tb.Fatalf("row %d purchase_order = %q, want %q", i, row.PurchaseOrder, expensesListMixedPaginationPurchaseOrderID)
					}
				}
			},
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
