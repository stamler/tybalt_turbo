package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const (
	hrUserEmail                         = "hr@example.com"
	hrEditableRecordID                  = "35i85kqy88hfsfc"
	hrEditableRecordAlternateBranchID   = "1r7r6hyp681vi15"
	hrEditableRecordPayrollID           = "9999"
	hrEditableRecordListExpectedName    = `"given_name":"Horace"`
	hrEditableRecordViewExpectedPayroll = `"payroll_id":"9999"`
	noClaimsUserID                      = "4ssj9f1yg250o9y"
)

func TestAdminProfilesAugmentedAccess_HRCanListAndView(t *testing.T) {
	hrToken, err := testutils.GenerateRecordToken("users", hrUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "hr can list admin profiles",
			Method: http.MethodGet,
			URL:    "/api/collections/admin_profiles_augmented/records?page=1&perPage=200",
			Headers: map[string]string{
				"Authorization": hrToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + hrEditableRecordID + `"`,
				hrEditableRecordListExpectedName,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can view admin profile details",
			Method: http.MethodGet,
			URL:    "/api/collections/admin_profiles_augmented/records/" + hrEditableRecordID,
			Headers: map[string]string{
				"Authorization": hrToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + hrEditableRecordID + `"`,
				hrEditableRecordViewExpectedPayroll,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestAdminProfilesUpdateRule_HRCanUpdateAllowedFields(t *testing.T) {
	hrToken, err := testutils.GenerateRecordToken("users", hrUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "hr can update payroll id",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"1001",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"payroll_id":"1001"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update active flag",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"active":false
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"active":false`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update salary",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"salary":true
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"salary":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update mobile phone",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"mobile_phone":"+1 (555) 555-0100"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"mobile_phone":"+1 (555) 555-0100"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update job title",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"job_title":"HR Manager"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"job_title":"HR Manager"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update default charge out rate",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":99.5,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"default_charge_out_rate":99.5`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update default branch",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"default_branch":"` + hrEditableRecordAlternateBranchID + `"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"default_branch":"` + hrEditableRecordAlternateBranchID + `"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update off rotation permitted",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"off_rotation_permitted":true
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"off_rotation_permitted":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update time sheet expected",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"time_sheet_expected":true
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"time_sheet_expected":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update personal reimbursement flag",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"allow_personal_reimbursement":true
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"allow_personal_reimbursement":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update skip min time check",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"yes"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"skip_min_time_check":"yes"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update personal vehicle insurance expiry",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"personal_vehicle_insurance_expiry":"2026-12-31"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"personal_vehicle_insurance_expiry":"2026-12-31"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr can update all allowed fields together",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"allow_personal_reimbursement": true,
				"skip_min_time_check": "on_next_bundle",
				"payroll_id": "2002",
				"active": false,
				"salary": true,
				"mobile_phone": "+1 (555) 555-0100",
				"job_title": "HR Lead",
				"personal_vehicle_insurance_expiry": "2027-01-15",
				"time_sheet_expected": true,
				"off_rotation_permitted": true,
				"default_charge_out_rate": 123.45,
				"default_branch": "` + hrEditableRecordAlternateBranchID + `"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"allow_personal_reimbursement":true`,
				`"skip_min_time_check":"on_next_bundle"`,
				`"payroll_id":"2002"`,
				`"active":false`,
				`"salary":true`,
				`"mobile_phone":"+1 (555) 555-0100"`,
				`"job_title":"HR Lead"`,
				`"personal_vehicle_insurance_expiry":"2027-01-15"`,
				`"time_sheet_expected":true`,
				`"off_rotation_permitted":true`,
				`"default_charge_out_rate":123.45`,
				`"default_branch":"` + hrEditableRecordAlternateBranchID + `"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestAdminProfilesUpdateRule_HRRejectedForDisallowedFields(t *testing.T) {
	hrToken, err := testutils.GenerateRecordToken("users", hrUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "hr cannot update work week hours",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"work_week_hours":37.5
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr cannot update opening date",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"opening_date":"2026-01-01"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr cannot update uid",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"uid":"hruser000000001"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr cannot update legacy uid",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"legacy_uid":"legacy_hr_override"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "hr cannot update record id",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"id":"abc123def456ghi",
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": hrToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestAdminProfilesFixtures_NoClaimsUserRemainsClaimless(t *testing.T) {
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)

	var result struct {
		Count int64 `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM user_claims
		WHERE uid = {:uid}
	`).Bind(map[string]any{
		"uid": noClaimsUserID,
	}).One(&result); err != nil {
		t.Fatal(err)
	}

	if result.Count != 0 {
		t.Fatalf("expected noclaims fixture to have zero claims, got %d", result.Count)
	}
}

func TestAdminProfilesAccess_AdminStillHasFullAccessAndOthersDoNot(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "admin can update fields outside hr scope",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"job_title":"Admin Updated Title"
			}`),
			Headers: map[string]string{
				"Authorization": adminToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"job_title":"Admin Updated Title"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "admin cannot update immutable uid",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"uid":"hruser000000001",
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": adminToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "admin cannot update immutable legacy uid",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"legacy_uid":"legacy_admin_override",
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": adminToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "admin cannot update immutable record id",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"id":"abc123def456ghi",
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": adminToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without admin or hr cannot view admin profile",
			Method: http.MethodGet,
			URL:    "/api/collections/admin_profiles_augmented/records/" + hrEditableRecordID,
			Headers: map[string]string{
				"Authorization": regularUserToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without admin or hr cannot update admin profile",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"1001",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no"
			}`),
			Headers: map[string]string{
				"Authorization": regularUserToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
