package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestJobsUnused_PrefixEndpoint(t *testing.T) {
	// Use a user with the 'job' claim: author@soup.com (uid f2j5a8vk006baub)
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "empty prefix (whitespace) uses default and returns 200",
			Method:         http.MethodGet,
			URL:            "/api/jobs/unused?prefix=%20", // space
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[]",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "unknown prefix returns empty list",
			Method:          http.MethodGet,
			URL:             "/api/jobs/unused?prefix=ZZZ-",
			Headers:         map[string]string{"Authorization": recordToken},
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: []string{"[]"},
			TestAppFactory:  testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
