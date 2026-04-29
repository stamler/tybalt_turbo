package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

const (
	identityAdminProfileID      = "apidsubject0001"
	identityOtherAdminProfileID = "apidother000001"
	identityMicrosoftAuthID     = "easubject000001"
	identityGoogleAuthID        = "easubject000002"
	identityOtherAuthID         = "eaother00000001"
	itIdentityUserEmail         = "it.identity@example.com"
)

func TestAdminProfileIdentityAccess_AdminAndITCanRead(t *testing.T) {
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	itToken, err := testutils.GenerateRecordToken("users", itIdentityUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "admin can list identity records",
			Method: http.MethodGet,
			URL:    "/api/admin_profiles/identity",
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityAdminProfileID + `"`,
				`"legacy_uid":"legacy_identity_subject"`,
				`"provider_count":2`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "it can list identity records",
			Method: http.MethodGet,
			URL:    "/api/admin_profiles/identity",
			Headers: map[string]string{
				"Authorization": itToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityAdminProfileID + `"`,
				`"legacy_uid":"legacy_identity_subject"`,
				`"provider_count":2`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "it can view identity details",
			Method: http.MethodGet,
			URL:    "/api/admin_profiles/" + identityAdminProfileID + "/identity",
			Headers: map[string]string{
				"Authorization": itToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityAdminProfileID + `"`,
				`"legacy_uid":"legacy_identity_subject"`,
				`"provider":"microsoft"`,
				`"provider_id":"provider-subject-ms"`,
				`"provider":"google"`,
				`"provider_id":"provider-subject-google"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestAdminProfileIdentityAccess_OtherClaimsCannotRead(t *testing.T) {
	hrToken, err := testutils.GenerateRecordToken("users", hrUserEmail)
	if err != nil {
		t.Fatal(err)
	}
	timeOffManagerToken, err := testutils.GenerateRecordToken("users", "u_with_claim@example.com")
	if err != nil {
		t.Fatal(err)
	}
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "hr cannot list identity records",
			Method: http.MethodGet,
			URL:    "/api/admin_profiles/identity",
			Headers: map[string]string{
				"Authorization": hrToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"You do not have permission to manage admin profile identity fields."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time off manager cannot view identity details",
			Method: http.MethodGet,
			URL:    "/api/admin_profiles/" + identityAdminProfileID + "/identity",
			Headers: map[string]string{
				"Authorization": timeOffManagerToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"You do not have permission to manage admin profile identity fields."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without claims cannot view identity details",
			Method: http.MethodGet,
			URL:    "/api/admin_profiles/" + identityAdminProfileID + "/identity",
			Headers: map[string]string{
				"Authorization": noClaimsToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"You do not have permission to manage admin profile identity fields."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestAdminProfileIdentitySaveLegacyUID(t *testing.T) {
	itToken, err := testutils.GenerateRecordToken("users", itIdentityUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "it can update legacy uid",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + identityAdminProfileID + "/identity",
			Body: strings.NewReader(`{
				"legacy_uid": "  legacy_identity_updated  "
			}`),
			Headers: map[string]string{
				"Authorization": itToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityAdminProfileID + `"`,
				`"legacy_uid":"legacy_identity_updated"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "it can clear legacy uid",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + identityAdminProfileID + "/identity",
			Body: strings.NewReader(`{
				"legacy_uid": ""
			}`),
			Headers: map[string]string{
				"Authorization": itToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityAdminProfileID + `"`,
				`"legacy_uid":""`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "duplicate legacy uid is rejected",
			Method: http.MethodPost,
			URL:    "/api/admin_profiles/" + identityAdminProfileID + "/identity",
			Body: strings.NewReader(`{
				"legacy_uid": "legacy_identity_other"
			}`),
			Headers: map[string]string{
				"Authorization": itToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"duplicate_legacy_uid"`,
				`"message":"legacy_uid is already assigned to another admin profile"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestAdminProfileIdentityClearAuthorizedProvider(t *testing.T) {
	itToken, err := testutils.GenerateRecordToken("users", itIdentityUserEmail)
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "it can clear one authorized provider",
			Method: http.MethodPost,
			URL: "/api/admin_profiles/" + identityAdminProfileID +
				"/authorized_providers/" + identityMicrosoftAuthID + "/clear",
			Headers: map[string]string{
				"Authorization": itToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityAdminProfileID + `"`,
				`"id":"` + identityGoogleAuthID + `"`,
				`"provider_id":"provider-subject-google"`,
			},
			NotExpectedContent: []string{
				`"id":"` + identityMicrosoftAuthID + `"`,
				`"provider_id":"provider-subject-ms"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "it cannot clear another user's provider through this profile",
			Method: http.MethodPost,
			URL: "/api/admin_profiles/" + identityAdminProfileID +
				"/authorized_providers/" + identityOtherAuthID + "/clear",
			Headers: map[string]string{
				"Authorization": itToken,
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"code":"not_found"`,
				`"message":"authorized provider not found for this admin profile"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "clearing subject provider does not remove another user's provider",
			Method: http.MethodPost,
			URL: "/api/admin_profiles/" + identityOtherAdminProfileID +
				"/authorized_providers/" + identityOtherAuthID + "/clear",
			Headers: map[string]string{
				"Authorization": itToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":"` + identityOtherAdminProfileID + `"`,
				`"authorized_providers":[]`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
