// expenses_test.go
package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestExpensesCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid expense gets a correct pay period ending and approver",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"f2j5a8vk006baub"`,
				`"pay_period_ending":"2024-09-14"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         1,
				"OnModelAfterCreate":          1,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid expense with Inactive vendor fails",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "ctswqva5onxj75q"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`{"code":400,"message":"Failed to create record."`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "expense created against an Active, Normal purchase_orders record succeeds",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
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
				"purchase_order": "2plsetqdxht7esg"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"purchase_order":"2plsetqdxht7esg"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         1,
				"OnModelAfterCreate":          1,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "expense created against a non-Active, Normal purchase_orders record fails",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
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
				"purchase_order": "gal6e5la2fa4rpn"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"purchase_order":{"code":"not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":        1,
				"OnModelAfterCreate":         1,
				"OnRecordAfterCreateRequest": 1,
				"OnBeforeApiError":           0,
				"OnAfterApiError":            0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},

		// TODO: valid mileage expense gets a correct total calculated and vendor cleared
		// TODO: valid mileage expense that spans multiple mileage tiers gets a correct total calculated and vendor cleared
		// TODO: unit test for CalculateMileageTotal
		// TODO: valid allowance expense gets a correct total calculated and vendor cleared and description set

		// TODO: expenses created against an Active purchase_orders record for which the caller is not allowed to create an expense fail
		// TODO: enhance validate_expenses_test.go
		{
			Name:   "unauthenticated request fails",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
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
			Name:   "uid must be present in the request",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Url:    "/api/collections/expenses/records",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Url:    "/api/collections/expenses/records",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Name:   "setting category without job fails",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Name:   "setting category with job succeeds if purchase_order is set",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
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
				"purchase_order": "2plsetqdxht7esg"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"category":"t5nmdl188gtlhz0"`,
				`"job":"cjf0kt0defhq480"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         1,
				"OnModelAfterCreate":          1,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting category with job fails if purchase_order is not set",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
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
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"purchase_order":{"code":"validation_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting job without category fails if purchase_order is not set",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"job": "cjf0kt0defhq480"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"purchase_order":{"code":"validation_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeCreate":         0,
				"OnModelAfterCreate":          0,
				"OnRecordBeforeCreateRequest": 1,
				"OnRecordAfterCreateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting category with job fails if category does not belong to the job even if purchase_order is set",
			Method: http.MethodPost,
			Url:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
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
				"purchase_order": "2plsetqdxht7esg"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestExpensesUpdate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid expense gets a correct pay period ending",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"approved":""`,
				`"approver":"f2j5a8vk006baub"`,
				`"pay_period_ending":"2024-09-14"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         1,
				"OnModelAfterUpdate":          1,
				"OnRecordBeforeUpdateRequest": 1,
				"OnRecordAfterUpdateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "otherwise valid expense update with Inactive vendor fails",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "ctswqva5onxj75q"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			ExpectedEvents: map[string]int{},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthenticated request fails",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
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
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Name:   "succeeds when uid is present and matches existing value",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"uid":"rzr98oadsp9qc11"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         1,
				"OnModelAfterUpdate":          1,
				"OnRecordBeforeUpdateRequest": 1,
				"OnRecordAfterUpdateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting rejector, rejected, and rejection_reason fails",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
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
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Name:   "setting category without job fails",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Name:   "setting category with job succeeds if purchase_order is set",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480",
				"purchase_order": "2plsetqdxht7esg"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"category":"t5nmdl188gtlhz0"`,
				`"job":"cjf0kt0defhq480"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         1,
				"OnModelAfterUpdate":          1,
				"OnRecordBeforeUpdateRequest": 1,
				"OnRecordAfterUpdateRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting category with job fails if purchase_order is not set",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"category": "t5nmdl188gtlhz0",
				"job": "cjf0kt0defhq480"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"purchase_order":{"code":"validation_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         0,
				"OnModelAfterUpdate":          0,
				"OnRecordBeforeUpdateRequest": 1,
				"OnRecordAfterUpdateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting job without category fails if purchase_order is not set",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "test expense",
				"pay_period_ending": "2006-01-02",
				"payment_type": "Expense",
				"total": 99,
				"vendor": "2zqxtsmymf670ha",
				"job": "cjf0kt0defhq480"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"purchase_order":{"code":"validation_required"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate":         0,
				"OnModelAfterUpdate":          0,
				"OnRecordBeforeUpdateRequest": 1,
				"OnRecordAfterUpdateRequest":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "setting category with job fails if category does not belong to the job even if purchase_order is set",
			Method: http.MethodPatch,
			Url:    "/api/collections/expenses/records/2gq9uyxmkcyopa4",
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
				"purchase_order": "2plsetqdxht7esg"
				}`),
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Url:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			RequestHeaders: map[string]string{"Authorization": nonCreatorToken},
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
			Url:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			RequestHeaders: map[string]string{"Authorization": recordToken},
			ExpectedStatus: 204,
			ExpectedEvents: map[string]int{
				"OnModelBeforeDelete":         1,
				"OnModelAfterDelete":          1,
				"OnRecordBeforeDeleteRequest": 1,
				"OnRecordAfterDeleteRequest":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "expense cannot be deleted by the creator if it is committed",
			Method:         http.MethodDelete,
			Url:            "/api/collections/expenses/records/xg2yeucklhgbs3n",
			RequestHeaders: map[string]string{"Authorization": recordToken},
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
			Url:             "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			RequestHeaders:  map[string]string{"Authorization": creatorToken},
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
			Url:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			RequestHeaders: map[string]string{"Authorization": approverToken},
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
			Url:             "/api/collections/expenses/records/xg2yeucklhgbs3n",
			RequestHeaders:  map[string]string{"Authorization": approverToken},
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
			Url:             "/api/collections/expenses/records/b4o6xph4ngwx4nw",
			RequestHeaders:  map[string]string{"Authorization": commitToken},
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
			Url:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			RequestHeaders: map[string]string{"Authorization": commitToken},
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
			Url:             "/api/collections/expenses/records/xg2yeucklhgbs3n",
			RequestHeaders:  map[string]string{"Authorization": reportToken},
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
			Url:            "/api/collections/expenses/records/2gq9uyxmkcyopa4",
			RequestHeaders: map[string]string{"Authorization": reportToken},
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
			Url:             "/api/expenses/eqhozipupteogp8/commit",
			RequestHeaders:  map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Record committed successfully"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 1,
				"OnModelAfterUpdate":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "referenced Cumulative purchase_orders record is closed, when total matches or exceeds PO total",
			Method:          http.MethodPost,
			Url:             "/api/expenses/hlqb5xdzm2xbii7/commit",
			RequestHeaders:  map[string]string{"Authorization": commitToken},
			ExpectedStatus:  200,
			ExpectedContent: []string{`"message":"Record committed successfully"`},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 2,
				"OnModelAfterUpdate":  2,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "commit fails against cumulative purchase_orders record if the total exceeds the PO total by more than specified excess",
			Method:          http.MethodPost,
			Url:             "/api/expenses/um1uoad5a4mhfcu/commit",
			RequestHeaders:  map[string]string{"Authorization": commitToken},
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
