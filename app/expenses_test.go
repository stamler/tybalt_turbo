// main_test.go
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

	// adminToken, err := generateAdminToken("test@example.com")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	scenarios := []tests.ApiScenario{
		{
			Name:   "valid expense gets a correct pay period ending",
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor"
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
				"vendor_name": "The Vendor",
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
