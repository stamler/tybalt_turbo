package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestProjectAuthorizationDownstreamUpdateRoutes(t *testing.T) {
	expenseOwnerToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	legacyPOToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "expense update route returns PA gate code",
			Method: http.MethodPatch,
			URL:    "/api/expenses/77i1224mudailrb",
			Body: strings.NewReader(`{
				"date": "2024-09-01",
				"division": "vccd5fo56ctbigh",
				"allowance_types": ["Breakfast"],
				"description": "PA gate allowance update",
				"job": "cjf0kt0defhq480",
				"kind": "prj0kind0000001",
				"payment_type": "Allowance",
				"purchase_order": "",
				"total": 700,
				"vendor": "2zqxtsmymf670ha"
			}`),
			Headers: map[string]string{
				"Authorization": expenseOwnerToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				hooks.ProjectAuthorizationNotApprovedCode,
				hooks.ProjectAuthorizationNotApprovedMessage,
			},
			TestAppFactory: setupProjectAuthorizationEnforcedApp,
		},
		{
			Name:   "legacy purchase order update route returns PA gate code",
			Method: http.MethodPatch,
			URL:    "/api/purchase_orders/legacy/legacyupd00001",
			Body: strings.NewReader(`{
				"date": "2025-01-15",
				"division": "vccd5fo56ctbigh",
				"branch": "80875lm27v8wgi4",
				"description": "PA gate purchase order update",
				"job": "cjf0kt0defhq480",
				"kind": "prj0kind0000001",
				"payment_type": "OnAccount",
				"po_number": "2502-5011",
				"total": 100,
				"type": "One-Time",
				"uid": "f2j5a8vk006baub",
				"approver": "wegviunlyr2jjjv",
				"vendor": "yxhycv2ycpvsbt4"
			}`),
			Headers: map[string]string{
				"Authorization": legacyPOToken,
				"Content-Type":  "application/json",
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				hooks.ProjectAuthorizationNotApprovedCode,
				hooks.ProjectAuthorizationNotApprovedMessage,
			},
			TestAppFactory: setupProjectAuthorizationEnforcedLegacyPOApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func setupProjectAuthorizationEnforcedApp(tb testing.TB) *tests.TestApp {
	tb.Helper()
	app := testutils.SetupTestApp(tb)
	setProjectAuthorizationBundleGateConfig(tb, app, true)
	return app
}

func setupProjectAuthorizationEnforcedLegacyPOApp(tb testing.TB) *tests.TestApp {
	tb.Helper()
	app := setupProjectAuthorizationEnforcedApp(tb)
	setLegacyPOCreateUpdate(tb, app, true)
	return app
}
