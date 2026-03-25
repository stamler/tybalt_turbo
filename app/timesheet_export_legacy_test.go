package main

import (
	"encoding/json"
	"github.com/pocketbase/pocketbase/tests"
	"net/http"
	"testing"
	"tybalt/internal/testutils"
)

func TestTimesheetExportLegacyAuth(t *testing.T) {
	// The test database has a machine_secrets record with:
	// role: legacy_writeback
	// expiry: 2028-05-08 (unexpired)
	// salt: testsalt
	// secret: test-secret-123
	// sha256_hash: SHA256("testsalt" + "test-secret-123")
	validToken := "test-secret-123"

	// User with report claim (fatt@mac.com has report claim in test db)
	reportClaimToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// User without report claim (time@test.com does not have report claim)
	noReportClaimToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing Authorization header returns 401",
			Method:         http.MethodGet,
			URL:            "/api/export_legacy/time_sheets/2024-06-29",
			Headers:        map[string]string{},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid Authorization header format returns 401",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/time_sheets/2024-06-29",
			Headers: map[string]string{
				"Authorization": "Basic sometoken",
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "wrong Bearer token returns 401",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/time_sheets/2024-06-29",
			Headers: map[string]string{
				"Authorization": "Bearer wrong-token",
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "valid Bearer token returns 200",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/time_sheets/2024-06-29",
			Headers: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"timeSheets"`,
				`"timeAmendments"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with report claim returns 200",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/time_sheets/2024-06-29",
			Headers: map[string]string{
				"Authorization": reportClaimToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"timeSheets"`,
				`"timeAmendments"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without report claim returns 401",
			Method: http.MethodGet,
			URL:    "/api/export_legacy/time_sheets/2024-06-29",
			Headers: map[string]string{
				"Authorization": noReportClaimToken,
			},
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestTimesheetExportLegacyIncludesSeparatedTimeSheetsAndAmendments(t *testing.T) {
	validToken := "test-secret-123"

	type exportResponse struct {
		TimeSheets []struct {
			ID         string `json:"id"`
			WeekEnding string `json:"weekEnding"`
		} `json:"timeSheets"`
		TimeAmendments []struct {
			ID                  string `json:"id"`
			WeekEnding          string `json:"weekEnding"`
			CommittedWeekEnding string `json:"committedWeekEnding"`
			Committer           string `json:"committer"`
			CommitterName       string `json:"committerName"`
		} `json:"timeAmendments"`
	}

	scenario := tests.ApiScenario{
		Name:   "export separates timesheets and time amendments",
		Method: http.MethodGet,
		URL:    "/api/export_legacy/time_sheets/2024-09-28",
		Headers: map[string]string{
			"Authorization": "Bearer " + validToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"timeSheets"`,
			`"timeAmendments"`,
			`"id":"j1lr2oddjongtoj"`,
			`"id":"qn4jyrkxp3pfjon"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			defer res.Body.Close()

			var payload exportResponse
			if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
				tb.Fatalf("failed to decode export response: %v", err)
			}

			if len(payload.TimeSheets) != 1 {
				tb.Fatalf("timeSheets length = %d, want 1", len(payload.TimeSheets))
			}
			if payload.TimeSheets[0].ID != "j1lr2oddjongtoj" {
				tb.Fatalf("timeSheets[0].id = %q, want %q", payload.TimeSheets[0].ID, "j1lr2oddjongtoj")
			}

			if len(payload.TimeAmendments) != 1 {
				tb.Fatalf("timeAmendments length = %d, want 1", len(payload.TimeAmendments))
			}
			amendment := payload.TimeAmendments[0]
			if amendment.ID != "qn4jyrkxp3pfjon" {
				tb.Fatalf("timeAmendments[0].id = %q, want %q", amendment.ID, "qn4jyrkxp3pfjon")
			}
			if amendment.WeekEnding != "2024-09-28" {
				tb.Fatalf("timeAmendments[0].weekEnding = %q, want %q", amendment.WeekEnding, "2024-09-28")
			}
			if amendment.CommittedWeekEnding != "2024-09-28" {
				tb.Fatalf("timeAmendments[0].committedWeekEnding = %q, want %q", amendment.CommittedWeekEnding, "2024-09-28")
			}
			if amendment.Committer == "" {
				tb.Fatalf("timeAmendments[0].committer unexpectedly blank")
			}
			if amendment.CommitterName == "" {
				tb.Fatalf("timeAmendments[0].committerName unexpectedly blank")
			}
		},
	}

	scenario.Test(t)
}

func TestTimesheetExportLegacyUsesAdminProfileLegacyUIDs(t *testing.T) {
	validToken := "test-secret-123"

	type exportResponse struct {
		TimeSheets []struct {
			ID         string `json:"id"`
			UID        string `json:"uid"`
			ManagerUID string `json:"managerUid"`
			Entries    []struct {
				ID  string `json:"id"`
				UID string `json:"uid"`
			} `json:"entries"`
		} `json:"timeSheets"`
		TimeAmendments []struct {
			ID        string `json:"id"`
			UID       string `json:"uid"`
			Committer string `json:"committer"`
		} `json:"timeAmendments"`
	}

	scenario := tests.ApiScenario{
		Name:   "export uses legacy uids for time writeback user ids",
		Method: http.MethodGet,
		URL:    "/api/export_legacy/time_sheets/2024-09-28",
		Headers: map[string]string{
			"Authorization": "Bearer " + validToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"timeSheets"`,
			`"timeAmendments"`,
		},
		TestAppFactory: testutils.SetupTestApp,
		AfterTestFunc: func(tb testing.TB, _ *tests.TestApp, res *http.Response) {
			defer res.Body.Close()

			var payload exportResponse
			if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
				tb.Fatalf("failed to decode export response: %v", err)
			}

			if len(payload.TimeSheets) != 1 {
				tb.Fatalf("timeSheets length = %d, want 1", len(payload.TimeSheets))
			}

			sheet := payload.TimeSheets[0]
			if sheet.UID != "legacy_f2j5a8vk006baub" {
				tb.Fatalf("timeSheets[0].uid = %q, want %q", sheet.UID, "legacy_f2j5a8vk006baub")
			}
			if sheet.ManagerUID != "legacy_f2j5a8vk006baub" {
				tb.Fatalf("timeSheets[0].managerUid = %q, want %q", sheet.ManagerUID, "legacy_f2j5a8vk006baub")
			}
			if len(sheet.Entries) == 0 {
				tb.Fatalf("timeSheets[0].entries unexpectedly empty")
			}
			if sheet.Entries[0].UID != "legacy_f2j5a8vk006baub" {
				tb.Fatalf("timeSheets[0].entries[0].uid = %q, want %q", sheet.Entries[0].UID, "legacy_f2j5a8vk006baub")
			}

			if len(payload.TimeAmendments) != 1 {
				tb.Fatalf("timeAmendments length = %d, want 1", len(payload.TimeAmendments))
			}

			amendment := payload.TimeAmendments[0]
			if amendment.UID != "legacy_f2j5a8vk006baub" {
				tb.Fatalf("timeAmendments[0].uid = %q, want %q", amendment.UID, "legacy_f2j5a8vk006baub")
			}
			if amendment.Committer != "legacy_wegviunlyr2jjjv" {
				tb.Fatalf("timeAmendments[0].committer = %q, want %q", amendment.Committer, "legacy_wegviunlyr2jjjv")
			}
		},
	}

	scenario.Test(t)
}
