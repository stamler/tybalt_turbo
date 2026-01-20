package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestClientsCreate_BusDevLeadMissingClaim_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "client create fails when business_development_lead lacks busdev claim",
			Method: http.MethodPost,
			URL:    "/api/collections/clients/records",
			Body: strings.NewReader(`{
                "name": "Acme Widgets",
                "business_development_lead": "4ssj9f1yg250o9y"
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"business_development_lead":{"code":"missing_claim"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestClientsCreate_BusDevLeadWithClaim_Succeeds(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "client create succeeds when business_development_lead has busdev claim",
			Method: http.MethodPost,
			URL:    "/api/collections/clients/records",
			Body: strings.NewReader(`{
                "name": "Globex Corporation",
                "business_development_lead": "4r70mfovf22m9uh"
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"name":"Globex Corporation"`,
				`"business_development_lead":"4r70mfovf22m9uh"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestClientsCreate_BusDevLeadInactive_Fails(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "client create fails when business_development_lead is inactive",
			Method: http.MethodPost,
			URL:    "/api/collections/clients/records",
			Body: strings.NewReader(`{
                "name": "Inactive Lead Corp",
                "business_development_lead": "u_inactive"
            }`),
			Headers:        map[string]string{"Authorization": recordToken},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"business_development_lead":{"code":"business_development_lead_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreateRequest": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
