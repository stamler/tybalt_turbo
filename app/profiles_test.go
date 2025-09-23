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
