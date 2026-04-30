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
	recordToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenario := tests.ApiScenario{
		Name:   "time_entries update rejects uid changes",
		Method: http.MethodPatch,
		URL:    "/api/collections/time_entries/records/teclaimwrite001",
		Body: strings.NewReader(`{
			"uid": "f2j5a8vk006baub",
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

func TestTimeEntriesWriteRules_TimeClaimRequired(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	collection, err := app.FindCollectionByNameOrId("time_entries")
	if err != nil {
		t.Fatal(err)
	}

	for ruleName, rule := range map[string]*string{
		"createRule": collection.CreateRule,
		"updateRule": collection.UpdateRule,
		"deleteRule": collection.DeleteRule,
	} {
		if rule == nil || !strings.Contains(*rule, "@request.auth.user_claims_via_uid.cid.name ?= 'time'") {
			got := "<nil>"
			if rule != nil {
				got = *rule
			}
			t.Fatalf("time_entries %s = %q, want time claim requirement", ruleName, got)
		}
	}

	noClaimsToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	timeToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "time_entries create rejects users without time claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "u_no_claims",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "attempt without time claim",
				"hours": 1
			}`),
			Headers:        map[string]string{"Authorization": noClaimsToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"message":"Failed to create record."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_entries update rejects users without time claim",
			Method: http.MethodPatch,
			URL:    "/api/collections/time_entries/records/r464ccf9b3527eb",
			Body: strings.NewReader(`{
					"description": "attempted update without time claim",
					"category": ""
				}`),
			Headers:        map[string]string{"Authorization": noClaimsToken},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "time_entries delete rejects users without time claim",
			Method:         http.MethodDelete,
			URL:            "/api/collections/time_entries/records/r464ccf9b3527eb",
			Headers:        map[string]string{"Authorization": noClaimsToken},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"message":"The requested resource wasn't found."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_entries create allows users with time claim",
			Method: http.MethodPost,
			URL:    "/api/collections/time_entries/records",
			Body: strings.NewReader(`{
				"uid": "rzr98oadsp9qc11",
				"time_type": "sdyfl3q7j7ap849",
				"date": "2024-09-02",
				"division": "fy4i9poneukvq9u",
				"description": "allowed with time claim",
				"hours": 1
			}`),
			Headers:        map[string]string{"Authorization": timeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"uid":"rzr98oadsp9qc11"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "time_entries update allows users with time claim",
			Method: http.MethodPatch,
			URL:    "/api/collections/time_entries/records/teclaimwrite001",
			Body: strings.NewReader(`{
					"description": "allowed update with time claim",
					"category": ""
				}`),
			Headers:        map[string]string{"Authorization": timeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"description":"allowed update with time claim"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "time_entries delete allows users with time claim",
			Method:         http.MethodDelete,
			URL:            "/api/collections/time_entries/records/teclaimwrite001",
			Headers:        map[string]string{"Authorization": timeToken},
			ExpectedStatus: http.StatusNoContent,
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
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
