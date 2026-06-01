package main

import (
	"database/sql"
	"errors"
	"net/http"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

func TestCopyTimesheetEntriesNextWeek(t *testing.T) {
	ownerToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	otherTimeUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	noClaimsToken, err := testutils.GenerateRecordToken("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "owner copies bundled time sheet entries into loose next-week entries",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/cpynextsheet001/copy_to_next_week",
			Headers:        map[string]string{"Authorization": ownerToken},
			ExpectedStatus: http.StatusCreated,
			ExpectedContent: []string{
				`"message":"Time sheet entries copied to next week"`,
				`"copied_count":2`,
				`"week_ending":"2031-01-11"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				if _, err := app.FindFirstRecordByFilter("time_sheets", "uid={:uid} && week_ending={:weekEnding}", dbx.Params{
					"uid":        "f2j5a8vk006baub",
					"weekEnding": "2031-01-11",
				}); err == nil {
					tb.Fatalf("copy should not create a time_sheets record for the target week")
				} else if !errors.Is(err, sql.ErrNoRows) {
					tb.Fatalf("failed to check target time sheet: %v", err)
				}

				entries, err := app.FindRecordsByFilter("time_entries", "uid={:uid} && week_ending={:weekEnding}", "date", 0, 0, dbx.Params{
					"uid":        "f2j5a8vk006baub",
					"weekEnding": "2031-01-11",
				})
				if err != nil {
					tb.Fatalf("failed to load copied entries: %v", err)
				}
				if len(entries) != 2 {
					tb.Fatalf("copied entry count = %d, want 2", len(entries))
				}

				wantDates := []string{"2031-01-06", "2031-01-07"}
				wantDescriptions := []string{"Copy next source regular one", "Copy next source regular two"}
				for i, entry := range entries {
					if got := entry.GetString("tsid"); got != "" {
						tb.Fatalf("entry %d tsid = %q, want empty", i, got)
					}
					if got := entry.GetString("date"); got != wantDates[i] {
						tb.Fatalf("entry %d date = %q, want %q", i, got, wantDates[i])
					}
					if got := entry.GetString("description"); got != wantDescriptions[i] {
						tb.Fatalf("entry %d description = %q, want %q", i, got, wantDescriptions[i])
					}
				}
			},
		},
		{
			Name:           "copy next week requires time claim",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/cpynextsheet001/copy_to_next_week",
			Headers:        map[string]string{"Authorization": noClaimsToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"message":"Time claim required."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "copy next week requires source owner",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/cpynextsheet001/copy_to_next_week",
			Headers:        map[string]string{"Authorization": otherTimeUserToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized"`,
				`"error":"you are not the owner of this time sheet"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "copy next week rejects when target time sheet already exists",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/av32qwch9xrcb5n/copy_to_next_week",
			Headers:        map[string]string{"Authorization": ownerToken},
			ExpectedStatus: http.StatusConflict,
			ExpectedContent: []string{
				`"code":"target_time_sheet_exists"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "copy next week rejects when loose target entries already exist",
			Method:         http.MethodPost,
			URL:            "/api/time_sheets/cpynextdupesht1/copy_to_next_week",
			Headers:        map[string]string{"Authorization": ownerToken},
			ExpectedStatus: http.StatusConflict,
			ExpectedContent: []string{
				`"code":"target_time_entries_exist"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
