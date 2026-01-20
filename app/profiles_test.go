package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// Profiles: prevent setting default_division to an inactive division
func TestProfilesUpdate_DefaultDivisionInactiveFails(t *testing.T) {
	// Use user time@test.com whose uid is rzr98oadsp9qc11 and profile id np26uewnzy56pq7
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating default_division to inactive fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/profiles/records/np26uewnzy56pq7",
			Body: strings.NewReader(`{
				"default_division": "apkev2ow1zjtm7w"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"default_division":{"code":"not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// Profiles: prevent setting manager to an inactive user
func TestProfilesUpdate_InactiveManagerFails(t *testing.T) {
	// Use user time@test.com whose uid is rzr98oadsp9qc11 and profile id np26uewnzy56pq7
	// Users can only update their own profile
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating manager to inactive user fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/profiles/records/np26uewnzy56pq7",
			Body: strings.NewReader(`{
				"manager": "u_inactive"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"manager":{"code":"manager_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "updating alternate_manager to inactive user fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/profiles/records/np26uewnzy56pq7",
			Body: strings.NewReader(`{
				"alternate_manager": "u_inactive"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"alternate_manager":{"code":"alternate_manager_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// Profiles: setting manager to an active user with tapr claim succeeds
func TestProfilesUpdate_ActiveManagerSucceeds(t *testing.T) {
	// Use user time@test.com whose uid is rzr98oadsp9qc11 and profile id np26uewnzy56pq7
	// Users can only update their own profile
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating manager to active user with tapr claim succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/profiles/records/np26uewnzy56pq7",
			Body: strings.NewReader(`{
				"manager": "f2j5a8vk006baub"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"manager":"f2j5a8vk006baub"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest":      1,
				"OnRecordUpdate":             1,
				"OnRecordUpdateExecute":      1,
				"OnRecordAfterUpdateSuccess": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "updating alternate_manager to active user with tapr claim succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/profiles/records/np26uewnzy56pq7",
			Body: strings.NewReader(`{
				"alternate_manager": "f2j5a8vk006baub"
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"alternate_manager":"f2j5a8vk006baub"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdateRequest":      1,
				"OnRecordUpdate":             1,
				"OnRecordUpdateExecute":      1,
				"OnRecordAfterUpdateSuccess": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
