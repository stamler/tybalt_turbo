// expenses_test.go
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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

func TestExpensesCreate(t *testing.T) {
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
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "valid expense gets a correct pay period ending and approver",
				Method:         http.MethodPost,
				URL:            "/api/collections/expenses/records",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"approved":""`,
					`"approver":"f2j5a8vk006baub"`,
					`"pay_period_ending":"2024-09-14"`,
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
				"pay_period_ending": "2006-01-02",
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
					fmt.Sprintf(`"kind":"%s"`, utilities.DefaultCapitalExpenditureKindID()),
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"uid": "rzr98oadsp9qc11",
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
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Mileage\"",
					"\"distance\":100",
					"\"total\":70",
					"\"vendor\":\"\"",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
				BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
					// Set valid insurance expiry for the user
					adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": "rzr98oadsp9qc11"})
					if err != nil {
						t.Fatalf("failed to find admin_profile: %v", err)
					}
					adminProfile.Set("personal_vehicle_insurance_expiry", "2025-12-31")
					if err := app.Save(adminProfile); err != nil {
						t.Fatalf("failed to save admin_profile: %v", err)
					}
				},
			}
		}(),
		func() tests.ApiScenario {
			// Period starts 2025-01-05; tiers {0:0.70, 5000:0.64}. With distance 5100
			// and zero prior mileage, expected total is 5000*0.70 + 100*0.64 = 3564.
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
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
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Mileage\"",
					"\"distance\":5100",
					"\"total\":3564",
					"\"vendor\":\"\"",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
				BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
					// Set valid insurance expiry for the user
					adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": "rzr98oadsp9qc11"})
					if err != nil {
						t.Fatalf("failed to find admin_profile: %v", err)
					}
					adminProfile.Set("personal_vehicle_insurance_expiry", "2025-12-31")
					if err := app.Save(adminProfile); err != nil {
						t.Fatalf("failed to save admin_profile: %v", err)
					}
				},
			}
		}(),
		func() tests.ApiScenario {
			// Mileage expense should fail when insurance expiry is not set
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
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
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"code":"insurance_expiry_missing"`,
					`personal vehicle insurance expiry must be updated with a valid date`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreateRequest": 1},
				TestAppFactory: testutils.SetupTestApp,
				BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
					// Ensure insurance expiry is empty
					adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": "rzr98oadsp9qc11"})
					if err != nil {
						t.Fatalf("failed to find admin_profile: %v", err)
					}
					adminProfile.Set("personal_vehicle_insurance_expiry", "")
					if err := app.Save(adminProfile); err != nil {
						t.Fatalf("failed to save admin_profile: %v", err)
					}
				},
			}
		}(),
		func() tests.ApiScenario {
			// Mileage expense should fail when insurance has expired
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
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
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"code":"insurance_expired"`,
					`personal vehicle insurance expired on 2024-12-31`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreateRequest": 1},
				TestAppFactory: testutils.SetupTestApp,
				BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
					// Set insurance expiry to a date before the expense date
					adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": "rzr98oadsp9qc11"})
					if err != nil {
						t.Fatalf("failed to find admin_profile: %v", err)
					}
					adminProfile.Set("personal_vehicle_insurance_expiry", "2024-12-31")
					if err := app.Save(adminProfile); err != nil {
						t.Fatalf("failed to save admin_profile: %v", err)
					}
				},
			}
		}(),
		func() tests.ApiScenario {
			// Mileage expense should succeed when expense date equals insurance expiry date
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
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
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					"\"payment_type\":\"Mileage\"",
					"\"distance\":100",
					"\"total\":70",
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
				BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
					// Set insurance expiry to the same date as the expense
					adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": "rzr98oadsp9qc11"})
					if err != nil {
						t.Fatalf("failed to find admin_profile: %v", err)
					}
					adminProfile.Set("personal_vehicle_insurance_expiry", "2025-01-10")
					if err := app.Save(adminProfile); err != nil {
						t.Fatalf("failed to save admin_profile: %v", err)
					}
				},
			}
		}(),
		func() tests.ApiScenario {
			// Non-mileage expense should not be affected by missing insurance expiry
			b, ct, err := makeMultipart(`{
				"uid": "rzr98oadsp9qc11",
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
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"payment_type":"Expense"`,
					`"total":99`,
				},
				ExpectedEvents: map[string]int{"OnRecordCreate": 1},
				TestAppFactory: testutils.SetupTestApp,
				BeforeTestFunc: func(t testing.TB, app *tests.TestApp, e *core.ServeEvent) {
					// Ensure insurance expiry is empty - should not affect non-mileage expenses
					adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": "rzr98oadsp9qc11"})
					if err != nil {
						t.Fatalf("failed to find admin_profile: %v", err)
					}
					adminProfile.Set("personal_vehicle_insurance_expiry", "")
					if err := app.Save(adminProfile); err != nil {
						t.Fatalf("failed to save admin_profile: %v", err)
					}
				},
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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

	var beforeCount int64

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

	// Count notifications before the request is executed.
	scenario.BeforeTestFunc = func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": "expense_rejected",
		}).One(&result)
		if err != nil {
			tb.Fatalf("failed to count notifications for expense_rejected before request: %v", err)
		}
		beforeCount = result.Count
	}

	// After the request, ensure that at least one new expense_rejected notification was created.
	scenario.AfterTestFunc = func(tb testing.TB, app *tests.TestApp, res *http.Response) {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": "expense_rejected",
		}).One(&result)
		if err != nil {
			tb.Fatalf("failed to count notifications for expense_rejected after request: %v", err)
		}
		if result.Count <= beforeCount {
			tb.Fatalf("expected expense_rejected notifications to be created by reject route, before=%d after=%d", beforeCount, result.Count)
		}
	}

	scenario.Test(t)
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
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
			}`)
			if err != nil {
				t.Fatal(err)
			}
			return tests.ApiScenario{
				Name:           "valid expense gets a correct pay period ending",
				Method:         http.MethodPatch,
				URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
				Body:           b,
				Headers:        map[string]string{"Authorization": recordToken, "Content-Type": ct},
				ExpectedStatus: 200,
				ExpectedContent: []string{
					`"approved":""`,
					`"approver":"f2j5a8vk006baub"`,
					`"pay_period_ending":"2024-09-14"`,
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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
				"pay_period_ending": "2006-01-02",
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

	scenarios := []tests.ApiScenario{
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

// computeTestHash computes SHA256 hash of data and returns it as a hex string
func computeTestHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
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

	// Use a specific file content that will produce a consistent hash
	// Compute its SHA256 hash to set on the existing record
	duplicateFileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xDE, 0xAD, 0xBE, 0xEF}
	duplicateHash := computeTestHash(duplicateFileContent)

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
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			// Create an existing expense with the same attachment hash directly in DB
			collection, err := app.FindCollectionByNameOrId("expenses")
			if err != nil {
				tb.Fatalf("failed to find expenses collection: %v", err)
			}
			record := core.NewRecord(collection)
			record.Set("uid", "rzr98oadsp9qc11")
			record.Set("date", "2024-08-01")
			record.Set("division", "vccd5fo56ctbigh")
			record.Set("description", "existing expense with attachment")
			record.Set("pay_period_ending", "2024-08-10")
			record.Set("payment_type", "Expense")
			record.Set("kind", utilities.DefaultCapitalExpenditureKindID())
			record.Set("total", 50)
			record.Set("vendor", "2zqxtsmymf670ha")
			record.Set("approver", "f2j5a8vk006baub")
			record.Set("branch", "80875lm27v8wgi4")
			record.Set("attachment_hash", duplicateHash)
			if err := app.Save(record); err != nil {
				tb.Fatalf("failed to create existing expense: %v", err)
			}
		},
	}

	// Create the request body for the duplicate attempt
	b, ct, err := makeMultipartWithContent(`{
		"uid": "rzr98oadsp9qc11",
		"date": "2024-09-01",
		"division": "vccd5fo56ctbigh",
		"description": "second expense with same attachment",
		"pay_period_ending": "2006-01-02",
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

	// Use a specific file content that will produce a consistent hash
	duplicateFileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0xCA, 0xFE, 0xBA, 0xBE}
	duplicateHash := computeTestHash(duplicateFileContent)

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
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			// Create an existing expense with the same attachment hash directly in DB
			collection, err := app.FindCollectionByNameOrId("expenses")
			if err != nil {
				tb.Fatalf("failed to find expenses collection: %v", err)
			}
			record := core.NewRecord(collection)
			record.Set("uid", "f2j5a8vk006baub") // Different user to avoid conflicts
			record.Set("date", "2024-08-01")
			record.Set("division", "vccd5fo56ctbigh")
			record.Set("description", "expense with attachment for update test")
			record.Set("pay_period_ending", "2024-08-10")
			record.Set("payment_type", "Expense")
			record.Set("kind", utilities.DefaultCapitalExpenditureKindID())
			record.Set("total", 50)
			record.Set("vendor", "2zqxtsmymf670ha")
			record.Set("approver", "etysnrlup2f6bak")
			record.Set("branch", "80875lm27v8wgi4")
			record.Set("attachment_hash", duplicateHash)
			if err := app.Save(record); err != nil {
				tb.Fatalf("failed to create existing expense: %v", err)
			}
		},
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

	// Use a specific file content
	fileContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x12, 0x34, 0x56, 0x78}
	existingHash := computeTestHash(fileContent)

	// The test expense record "2gq9uyxmkcyopa4" is owned by user "rzr98oadsp9qc11"
	// We'll set its attachment_hash to our hash, then try to update it with the same file
	scenario := tests.ApiScenario{
		Name:           "updating expense with its own attachment succeeds",
		Method:         http.MethodPatch,
		URL:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`"description":"updated description"`,
		},
		ExpectedEvents: map[string]int{
			"OnRecordUpdate": 1,
		},
		TestAppFactory: testutils.SetupTestApp,
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			// Set the attachment_hash on the existing expense to match what we'll upload
			expense, err := app.FindRecordById("expenses", "2gq9uyxmkcyopa4")
			if err != nil {
				tb.Fatalf("failed to find expense: %v", err)
			}
			expense.Set("attachment_hash", existingHash)
			if err := app.Save(expense); err != nil {
				tb.Fatalf("failed to update expense: %v", err)
			}
		},
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
