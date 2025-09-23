package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// jobs.divisions: prevent including inactive divisions on update
func TestJobsUpdate_InactiveDivisionFails(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating job divisions to include an inactive division fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"divisions": ["fy4i9poneukvq9u", "apkev2ow1zjtm7w"]
			}`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"data":{"divisions":{"code":"not_active"`,
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
