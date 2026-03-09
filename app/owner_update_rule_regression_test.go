package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestProfilesUpdateRule_UIDChangedIsRejected(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "profiles update rejects uid changes",
		Method: http.MethodPatch,
		URL:    "/api/collections/profiles/records/np26uewnzy56pq7",
		Body: strings.NewReader(`{
			"uid": "f2j5a8vk006baub"
		}`),
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: http.StatusNotFound,
		ExpectedContent: []string{
			`"message":"The requested resource wasn't found."`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestTimeEntriesUpdateRule_UIDChangedIsRejected(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "time_entries update rejects uid changes",
		Method: http.MethodPatch,
		URL:    "/api/collections/time_entries/records/r464ccf9b3527eb",
		Body: strings.NewReader(`{
			"uid": "u_with_claim",
			"category": ""
		}`),
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: http.StatusNotFound,
		ExpectedContent: []string{
			`"message":"The requested resource wasn't found."`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestTimeSheetsUpdateRule_UnauthorizedDirectUpdateIsRejected(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "time_sheets direct update denied for unauthorized actor",
		Method: http.MethodPatch,
		URL:    "/api/collections/time_sheets/records/aeyl94og4xmnpq4",
		Body: strings.NewReader(`{
			"rejected": false
		}`),
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: http.StatusForbidden,
		ExpectedContent: []string{
			`"message":"Only superusers can perform this action."`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}

func TestTimeSheetsUpdateRule_DirectUpdateDisabledForApprover(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "time_sheets direct update denied even for approver",
		Method: http.MethodPatch,
		URL:    "/api/collections/time_sheets/records/aeyl94og4xmnpq4",
		Body: strings.NewReader(`{
			"rejected": false
		}`),
		Headers:        map[string]string{"Authorization": recordToken},
		ExpectedStatus: http.StatusForbidden,
		ExpectedContent: []string{
			`"message":"Only superusers can perform this action."`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}
