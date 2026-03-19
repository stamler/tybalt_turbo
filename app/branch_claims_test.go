package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	corporateBranchID     = "corpbranch00001"
	corporateClaimID      = "corpclaim000001"
	defaultBranchID       = "80875lm27v8wgi4"
	timeUserID            = "rzr98oadsp9qc11"
	timeUserEmail         = "time@test.com"
	legacyPOClaimHolderID = "wegviunlyr2jjjv"
	copySourceTimeEntryID = "tecopycorp00001"
)

type corporateBranchTestOptions struct {
	corporateBranchUsers []string
	corporateClaimUsers  []string
	enableLegacyPO       bool
	seedCopySource       bool
}

func setupCorporateBranchTestApp(tb testing.TB, opts corporateBranchTestOptions) *tests.TestApp {
	tb.Helper()

	app := testutils.SetupTestApp(tb)

	if opts.enableLegacyPO {
		setLegacyPOCreateUpdate(tb, app, true)
	}
	for _, uid := range opts.corporateBranchUsers {
		setUserDefaultBranch(tb, app, uid, corporateBranchID)
	}
	for _, uid := range opts.corporateClaimUsers {
		addUserClaim(tb, app, uid, corporateClaimID)
	}
	if opts.seedCopySource {
		seedCopySourceTimeEntry(tb, app, timeUserID)
	}

	return app
}

func setUserDefaultBranch(tb testing.TB, app *tests.TestApp, uid string, branchID string) {
	tb.Helper()

	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid = {:uid}", dbx.Params{
		"uid": uid,
	})
	if err != nil {
		tb.Fatalf("failed to load admin profile for %s: %v", uid, err)
	}

	adminProfile.Set("default_branch", branchID)
	if err := app.Save(adminProfile); err != nil {
		tb.Fatalf("failed to save admin profile for %s: %v", uid, err)
	}
}

func addUserClaim(tb testing.TB, app *tests.TestApp, uid string, claimID string) {
	tb.Helper()

	existing, err := app.FindFirstRecordByFilter(
		"user_claims",
		"uid = {:uid} && cid = {:cid}",
		dbx.Params{"uid": uid, "cid": claimID},
	)
	if err == nil && existing != nil {
		return
	}

	collection, err := app.FindCollectionByNameOrId("user_claims")
	if err != nil {
		tb.Fatalf("failed to load user_claims collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("uid", uid)
	record.Set("cid", claimID)
	if err := app.Save(record); err != nil {
		tb.Fatalf("failed to save user_claim for %s: %v", uid, err)
	}
}

func seedCopySourceTimeEntry(tb testing.TB, app *tests.TestApp, uid string) {
	tb.Helper()

	weekEnding, err := utilities.GenerateWeekEnding("2024-09-02")
	if err != nil {
		tb.Fatalf("failed to generate week ending: %v", err)
	}

	collection, err := app.FindCollectionByNameOrId("time_entries")
	if err != nil {
		tb.Fatalf("failed to load time_entries collection: %v", err)
	}

	record := core.NewRecord(collection)
	record.Set("id", copySourceTimeEntryID)
	record.Set("uid", uid)
	record.Set("branch", defaultBranchID)
	record.Set("date", "2024-09-02")
	record.Set("week_ending", weekEnding)
	record.Set("time_type", "sdyfl3q7j7ap849")
	record.Set("division", "fy4i9poneukvq9u")
	record.Set("description", "copy source entry")
	record.Set("hours", 1.0)
	record.Set("tsid", "")

	if err := app.Save(record); err != nil {
		tb.Fatalf("failed to seed copy source time entry: %v", err)
	}
}

func TestCorporateBranchClaimGating_PurchaseOrdersCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", timeUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "purchase order create fails when resolved branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
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
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"branch_claim_required"`,
				`Corporate requires at least one of these claims: corporate_branch`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
				})
			},
		},
		{
			Name:   "purchase order create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/collections/purchase_orders/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
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
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"corpbranch00001"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
					corporateClaimUsers:  []string{timeUserID},
				})
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_ExpensesCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", timeUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "expense create fails when resolved branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"kind": "l3vtlbqg529m52j",
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast"],
				"total": 0,
				"description": "Allowance for Breakfast"
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
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
				})
			},
		},
		{
			Name:   "expense create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/collections/expenses/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"date": "2025-01-10",
				"division": "vccd5fo56ctbigh",
				"kind": "l3vtlbqg529m52j",
				"payment_type": "Allowance",
				"allowance_types": ["Breakfast"],
				"total": 0,
				"description": "Allowance for Breakfast"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"corpbranch00001"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
					corporateClaimUsers:  []string{timeUserID},
				})
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_TimeEntriesCreate(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", timeUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time entry create fails when default branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry",
				"hours": 1
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`branch claim requirement not met`,
				`corporate_branch`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
				})
			},
		},
		{
			Name:   "time entry create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "test time entry",
				"hours": 1
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"corpbranch00001"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
					corporateClaimUsers:  []string{timeUserID},
				})
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCorporateBranchClaimGating_TimeEntriesCopyToTomorrow(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", timeUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "copy to tomorrow fails when destination branch is corporate and caller lacks claim",
			Method: http.MethodPost,
			URL:    "/api/time_entries/" + copySourceTimeEntryID + "/copy_to_tomorrow",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"branch":{"code":"branch_claim_required"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
					seedCopySource:       true,
				})
			},
		},
		{
			Name:   "copy to tomorrow succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/time_entries/" + copySourceTimeEntryID + "/copy_to_tomorrow",
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 201,
			ExpectedContent: []string{
				`"message":"Time entry copied to tomorrow"`,
				`"new_record_id":"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateBranchUsers: []string{timeUserID},
					corporateClaimUsers:  []string{timeUserID},
					seedCopySource:       true,
				})
			},
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
				"branch": "corpbranch00001",
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
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					enableLegacyPO: true,
				})
			},
		},
		{
			Name:   "legacy purchase order create succeeds when caller holds corporate_branch claim",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/legacy",
			Body: strings.NewReader(`{
				"po_number": "2501-5001",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "corpbranch00001",
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
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"branch":"corpbranch00001"`,
				`"po_number":"2501-5001"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					enableLegacyPO:      true,
					corporateClaimUsers: []string{legacyPOClaimHolderID},
				})
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
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"9999",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"default_branch":"corpbranch00001"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"default_branch":{"code":"branch_claim_required"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{})
			},
		},
		{
			Name:   "hr can assign corporate default branch when subject user already holds claim",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"9999",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"default_branch":"corpbranch00001"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"default_branch":"corpbranch00001"`,
			},
			TestAppFactory: func(tb testing.TB) *tests.TestApp {
				return setupCorporateBranchTestApp(tb, corporateBranchTestOptions{
					corporateClaimUsers: []string{"f2j5a8vk006baub"},
				})
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
