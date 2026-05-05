package main

import (
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestClientContactsDeleteRuleRequiresNoReferencingJobs(t *testing.T) {
	jobToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "client_contacts delete rejects contact referenced by jobs",
			Method:         http.MethodDelete,
			URL:            "/api/collections/client_contacts/records/nh5u9z3cyknjclv",
			Headers:        map[string]string{"Authorization": jobToken},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "client_contacts delete allows unreferenced contact",
			Method:         http.MethodDelete,
			URL:            "/api/collections/client_contacts/records/contactdelok001",
			Headers:        map[string]string{"Authorization": jobToken},
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
