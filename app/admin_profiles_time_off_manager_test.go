package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const timeOffManagerUserEmail = "u_with_claim@example.com"

func TestAdminProfilesAugmentedAccess_TimeOffManagerCanListAndView(t *testing.T) {
	timeOffManagerToken, err := testutils.GenerateRecordToken("users", timeOffManagerUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time off manager can list admin profiles",
			Method: http.MethodGet,
			URL:    "/api/collections/admin_profiles_augmented/records?page=1&perPage=200",
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + hrEditableRecordID + `"`,
				hrEditableRecordListExpectedName,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager can view admin profile details",
			Method: http.MethodGet,
			URL:    "/api/collections/admin_profiles_augmented/records/" + hrEditableRecordID,
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
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

func TestAdminProfilesUpdateRule_TimeOffManagerDirectUpdatesAreForbidden(t *testing.T) {
	timeOffManagerToken, err := testutils.GenerateRecordToken("users", timeOffManagerUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time off manager cannot update opening date directly",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"opening_date":"2026-01-04"
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager cannot update opening op directly",
			Method: http.MethodPatch,
			URL:    "/api/collections/admin_profiles/records/" + hrEditableRecordID,
			Body: strings.NewReader(`{
				"payroll_id":"` + hrEditableRecordPayrollID + `",
				"default_charge_out_rate":50,
				"skip_min_time_check":"no",
				"opening_op":12.5
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
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

func TestAdminProfilesLimitedSave_TimeOffManagerCanUpdateOpeningFields(t *testing.T) {
	timeOffManagerToken, err := testutils.GenerateRecordToken("users", timeOffManagerUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time off manager can update opening date",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"opening_date":"2026-01-04"
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"opening_date":"2026-01-04"`,
			},
			NotExpectedContent: []string{
				`"uid":"`,
				`"legacy_uid":"`,
				`"default_branch":`,
				`"payroll_id":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager cannot update opening date to non payroll boundary",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"opening_date":"2026-01-01"
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"data":{"opening_date":{"code":"invalid_opening_date"`,
				`"message":"opening_date must be the Sunday after a pay period ending date"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager can clear opening date when opening balances are zero",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"opening_date":"",
					"opening_op":0,
					"opening_ov":0
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"opening_date":""`,
				`"opening_op":0`,
				`"opening_ov":0`,
			},
			NotExpectedContent: []string{
				`"uid":"`,
				`"legacy_uid":"`,
				`"default_branch":`,
				`"payroll_id":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager cannot set non zero opening op with blank opening date",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"opening_date":"",
					"opening_op":12.5
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"data":{"opening_date":{"code":"invalid_opening_date"`,
				`"message":"opening_date is required when opening balances are non-zero"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager can update opening op without changing opening date",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"opening_op":12.5
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"opening_op":12.5`,
				`"opening_date":"2024-01-07"`,
			},
			NotExpectedContent: []string{
				`"uid":"`,
				`"legacy_uid":"`,
				`"default_branch":`,
				`"payroll_id":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager can update opening ov without changing opening date",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"opening_ov":18.75
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"opening_ov":18.75`,
				`"opening_date":"2024-01-07"`,
			},
			NotExpectedContent: []string{
				`"uid":"`,
				`"legacy_uid":"`,
				`"default_branch":`,
				`"payroll_id":"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager cannot use limited save as a read endpoint",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"missing_admin_profile_changes"`,
				`"message":"at least one admin profile field change is required"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager cannot update restricted fields through limited save endpoint",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + hrEditableRecordID + "/save_limited",
			Body: strings.NewReader(`{
				"admin_profile":{
					"payroll_id":"1001"
				}
			}`),
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"You do not have permission to edit one or more admin profile fields."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
