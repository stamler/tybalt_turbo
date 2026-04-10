package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const (
	corporateBranchID = "kpj5jijh0if8kx8"
)

func TestCorporateBranchClaimGating_PurchaseOrdersCreate(t *testing.T) {
	noClaimToken, err := testutils.GenerateRecordToken("users", "corp.noclaim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claimToken, err := testutils.GenerateRecordToken("users", "corp.claim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "purchase order create fails when resolved branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_noclaim",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "corporate branch purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"priority_second_approver": "6bq4j0eb26631dy",
				"status": "Unapproved",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": noClaimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"branch_claim_required"`,
				`Corporate requires at least one of these claims: corporate_branch`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "purchase order create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_claim",
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"description": "corporate branch purchase order",
				"payment_type": "Expense",
				"total": 1234.56,
				"vendor": "2zqxtsmymf670ha",
				"approver": "etysnrlup2f6bak",
				"priority_second_approver": "6bq4j0eb26631dy",
				"status": "Unapproved",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": claimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"kpj5jijh0if8kx8"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_ExpensesCreate(t *testing.T) {
	noClaimToken, err := testutils.GenerateRecordToken("users", "corp.noclaim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claimToken, err := testutils.GenerateRecordToken("users", "corp.claim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "expense create fails when resolved branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_noclaim",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"kind": "l3vtlbqg529m52j",
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast"],
				"total": 0,
				"description": "Allowance for Breakfast"
			}`),
			Headers: map[string]string{
				"Authorization": noClaimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"branch_claim_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "expense create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_claim",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"kind": "l3vtlbqg529m52j",
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast"],
				"total": 0,
				"description": "Allowance for Breakfast"
			}`),
			Headers: map[string]string{
				"Authorization": claimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"kpj5jijh0if8kx8"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_TimeEntriesCreate(t *testing.T) {
	noClaimToken, err := testutils.GenerateRecordToken("users", "corp.noclaim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claimToken, err := testutils.GenerateRecordToken("users", "corp.claim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry create fails when default branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_noclaim",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry",
				"hours": 1
			}`),
			Headers: map[string]string{
				"Authorization": noClaimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`branch claim requirement not met`,
				`corporate_branch`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_claim",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry",
				"hours": 1
			}`),
			Headers: map[string]string{
				"Authorization": claimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"kpj5jijh0if8kx8"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_TimeEntriesCreate_WithJobBranch(t *testing.T) {
	noClaimToken, err := testutils.GenerateRecordToken("users", "corp.noclaim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claimToken, err := testutils.GenerateRecordToken("users", "corp.claim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry create fails when job branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_noclaim",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry",
				"hours": 1,
				"job": "jobcorpbranch01",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers: map[string]string{
				"Authorization": noClaimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`branch claim requirement not met`,
				`corporate_branch`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time entry create succeeds when job branch is corporate and caller holds claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "u_corp_claim",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry",
				"hours": 1,
				"job": "jobcorpbranch01",
				"role": "tbgoiwwwfj8cvju"
			}`),
			Headers: map[string]string{
				"Authorization": claimToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"kpj5jijh0if8kx8"`,
				`"job":"jobcorpbranch01"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_TimeEntriesCopyToTomorrow(t *testing.T) {
	noClaimToken, err := testutils.GenerateRecordToken("users", "corp.noclaim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claimToken, err := testutils.GenerateRecordToken("users", "corp.claim@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "copy to tomorrow fails when preserved source branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/time_entries/tecopycorpnc001/copy_to_tomorrow",
			Headers: map[string]string{
				"Authorization": noClaimToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"branch_claim_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "copy to tomorrow preserves explicit non-job branch when caller default branch is corporate",
			Method: http.MethodPost,
			URL:    "/api/time_entries/tecopydefnc0001/copy_to_tomorrow",
			Headers: map[string]string{
				"Authorization": noClaimToken,
			},
			ExpectedStatus: 201,
			ExpectedContent: []string{
				`"message":"Time entry copied to tomorrow"`,
				`"new_record_id":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "copy to tomorrow succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/time_entries/tecopycorpcl001/copy_to_tomorrow",
			Headers: map[string]string{
				"Authorization": claimToken,
			},
			ExpectedStatus: 201,
			ExpectedContent: []string{
				`"message":"Time entry copied to tomorrow"`,
				`"new_record_id":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_LegacyPurchaseOrders(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	corporateManagerToken, err := testutils.GenerateRecordToken("users", "corp.manager@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "legacy purchase order create fails when caller lacks corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5000",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "kpj5jijh0if8kx8",
				"description": "Legacy corporate purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"branch_claim_required"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(tb)
				setLegacyPOCreateUpdate(tb, app, true)
				return app
			},
		},
		{
			Name:   "legacy purchase order create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5001",
				"uid": "u_corp_manager",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "kpj5jijh0if8kx8",
				"description": "Legacy corporate purchase order",
				"payment_type": "OnAccount",
				"total": 321.45,
				"vendor": "yxhycv2ycpvsbt4",
				"type": "One-Time",
				"kind": "l3vtlbqg529m52j"
			}`),
			Headers: map[string]string{
				"Authorization": corporateManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"kpj5jijh0if8kx8"`,
				`"po_number":"2501-5001"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(tb)
				setLegacyPOCreateUpdate(tb, app, true)
				return app
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_AdminProfilesDefaultBranch(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", hrUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "hr cannot assign corporate default branch when subject user lacks claim",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"default_branch":"kpj5jijh0if8kx8"
				}
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"default_branch":{"code":"branch_claim_required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can assign corporate default branch when subject user already holds claim",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/ap_subject_corp/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"default_branch":"kpj5jijh0if8kx8"
				}
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"default_branch":"kpj5jijh0if8kx8"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
