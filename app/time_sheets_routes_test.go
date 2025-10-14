package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestTimeSheetsRoutes(t *testing.T) {
	committerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	// A submitted, not committed, not approved timesheet to exercise reject path
	// Using one of the seeded ids seen in test db query: aeyl94og4xmnpq4
	tsToReject := "aeyl94og4xmnpq4"

	scenarios := []tests.ApiScenario{
		{
			Name:   "committer can view tracking counts",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/tracking_counts",
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"week_ending":`,
				`"submitted_count":`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "committer can view tracking list for a week",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/tracking/weeks/2024-06-29",
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"id":`,
				`"surname":`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "committer can reject a submitted timesheet",
			Method: http.MethodPost,
			URL:    "/api/time_sheets/" + tsToReject + "/reject",
			Body:   strings.NewReader(`{"rejection_reason": "Insufficient detail"}`),
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"message":"record rejected successfully"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "committer cannot commit an already committed timesheet",
			Method: http.MethodPost,
			URL:    "/api/time_sheets/j1lr2oddjongtoj/commit", // this one is committed in fixtures
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"record_already_committed"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
