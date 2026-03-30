package main

import (
	"bytes"
	"encoding/json"
	"github.com/pocketbase/pocketbase/tests"
	"io"
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
			UID                 string `json:"uid"`
			GivenName           string `json:"givenName"`
			Surname             string `json:"surname"`
			DisplayName         string `json:"displayName"`
			PayrollID           string `json:"payrollId"`
			Salary              bool   `json:"salary"`
			Created             string `json:"created"`
			Creator             string `json:"creator"`
			CreatorName         string `json:"creatorName"`
			WeekEnding          string `json:"weekEnding"`
			Committed           bool   `json:"committed"`
			CommitTime          string `json:"commitTime"`
			CommittedWeekEnding string `json:"committedWeekEnding"`
			CommitUID           string `json:"commitUid"`
			CommitName          string `json:"commitName"`
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

			body, err := io.ReadAll(res.Body)
			if err != nil {
				tb.Fatalf("failed to read export response body: %v", err)
			}

			var payload exportResponse
			if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
				tb.Fatalf("failed to decode export response: %v", err)
			}

			var rawPayload struct {
				TimeAmendments []map[string]any `json:"timeAmendments"`
			}
			if err := json.NewDecoder(bytes.NewReader(body)).Decode(&rawPayload); err != nil {
				tb.Fatalf("failed to decode raw export response: %v", err)
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
			if amendment.UID != "legacy_f2j5a8vk006baub" {
				tb.Fatalf("timeAmendments[0].uid = %q, want %q", amendment.UID, "legacy_f2j5a8vk006baub")
			}
			if amendment.GivenName != "Horace" {
				tb.Fatalf("timeAmendments[0].givenName = %q, want %q", amendment.GivenName, "Horace")
			}
			if amendment.Surname != "Silver" {
				tb.Fatalf("timeAmendments[0].surname = %q, want %q", amendment.Surname, "Silver")
			}
			if amendment.DisplayName != "Horace Silver" {
				tb.Fatalf("timeAmendments[0].displayName = %q, want %q", amendment.DisplayName, "Horace Silver")
			}
			if amendment.PayrollID != "9999" {
				tb.Fatalf("timeAmendments[0].payrollId = %q, want %q", amendment.PayrollID, "9999")
			}
			if !amendment.Salary {
				tb.Fatalf("timeAmendments[0].salary = false, want true")
			}
			if amendment.Created == "" {
				tb.Fatalf("timeAmendments[0].created unexpectedly blank")
			}
			if amendment.Creator != "legacy_f2j5a8vk006baub" {
				tb.Fatalf("timeAmendments[0].creator = %q, want %q", amendment.Creator, "legacy_f2j5a8vk006baub")
			}
			if amendment.CreatorName != "Horace Silver" {
				tb.Fatalf("timeAmendments[0].creatorName = %q, want %q", amendment.CreatorName, "Horace Silver")
			}
			if !amendment.Committed {
				tb.Fatalf("timeAmendments[0].committed = false, want true")
			}
			if amendment.CommitTime == "" {
				tb.Fatalf("timeAmendments[0].commitTime unexpectedly blank")
			}
			if amendment.CommittedWeekEnding != "2024-09-28" {
				tb.Fatalf("timeAmendments[0].committedWeekEnding = %q, want %q", amendment.CommittedWeekEnding, "2024-09-28")
			}
			if amendment.CommitUID != "legacy_wegviunlyr2jjjv" {
				tb.Fatalf("timeAmendments[0].commitUid = %q, want %q", amendment.CommitUID, "legacy_wegviunlyr2jjjv")
			}
			if amendment.CommitName != "Fakesy Manjor" {
				tb.Fatalf("timeAmendments[0].commitName = %q, want %q", amendment.CommitName, "Fakesy Manjor")
			}

			rawAmendment := rawPayload.TimeAmendments[0]
			if _, ok := rawAmendment["committer"]; ok {
				tb.Fatalf("timeAmendments[0].committer unexpectedly present")
			}
			if _, ok := rawAmendment["committerName"]; ok {
				tb.Fatalf("timeAmendments[0].committerName unexpectedly present")
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
			CommitUID string `json:"commitUid"`
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
			if amendment.CommitUID != "legacy_wegviunlyr2jjjv" {
				tb.Fatalf("timeAmendments[0].commitUid = %q, want %q", amendment.CommitUID, "legacy_wegviunlyr2jjjv")
			}
		},
	}

	scenario.Test(t)
}
