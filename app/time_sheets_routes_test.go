package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
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
			Name:   "committer can view missing list for a week",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/tracking/weeks/2024-06-29/missing",
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[",
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "committer can view not expected list for a week",
			Method: http.MethodGet,
			URL:    "/api/time_sheets/tracking/weeks/2024-06-29/not_expected",
			Headers: map[string]string{
				"Authorization": committerToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				"[",
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

// TestRejectTimesheet_QueuesNotifications verifies that rejecting a timesheet via the route
// queues one or more timesheet_rejected notifications.
func TestRejectTimesheet_QueuesNotifications(t *testing.T) {
	// Use the same committed user and timesheet id as the route scenarios above.
	const tsToReject = "aeyl94og4xmnpq4"

	committerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	var beforeCount int64

	scenario := tests.ApiScenario{
		Name:   "reject timesheet queues notifications",
		Method: http.MethodPost,
		URL:    "/api/time_sheets/" + tsToReject + "/reject",
		Body:   strings.NewReader(`{"rejection_reason": "Route-level rejection test"}`),
		Headers: map[string]string{
			"Authorization": committerToken,
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"message":"record rejected successfully"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Count notifications before the request is executed.
	scenario.BeforeTestFunc = func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": "timesheet_rejected",
		}).One(&result)
		if err != nil {
			tb.Fatalf("failed to count notifications for timesheet_rejected before request: %v", err)
		}
		beforeCount = result.Count
	}

	// After the request, ensure that at least one new timesheet_rejected notification was created.
	scenario.AfterTestFunc = func(tb testing.TB, app *tests.TestApp, res *http.Response) {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": "timesheet_rejected",
		}).One(&result)
		if err != nil {
			tb.Fatalf("failed to count notifications for timesheet_rejected after request: %v", err)
		}
		if result.Count <= beforeCount {
			tb.Fatalf("expected timesheet_rejected notifications to be created by reject route, before=%d after=%d", beforeCount, result.Count)
		}
	}

	scenario.Test(t)
}

// TestAddTimesheetReviewer_QueuesSharedNotifications verifies that adding a reviewer
// via the API queues one or more timesheet_shared notifications.
func TestAddTimesheetReviewer_QueuesSharedNotifications(t *testing.T) {
	// Use the same timesheet id as other tests.
	const timesheetID = "aeyl94og4xmnpq4"

	// Approver is the sharer for timesheet sharing.
	approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// Choose a viewer different from the approver (time@test.com).
	const viewerUID = "rzr98oadsp9qc11"

	var beforeCount int64

	body := fmt.Sprintf(`{"time_sheet": "%s", "reviewer": "%s"}`, timesheetID, viewerUID)

	scenario := tests.ApiScenario{
		Name:   "adding timesheet reviewer queues timesheet_shared notifications",
		Method: http.MethodPost,
		URL:    "/api/collections/time_sheet_reviewers/records",
		Body:   strings.NewReader(body),
		Headers: map[string]string{
			"Authorization": approverToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`"time_sheet":"` + timesheetID + `"`,
			`"reviewer":"` + viewerUID + `"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	// Count notifications before the request is executed.
	scenario.BeforeTestFunc = func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": "timesheet_shared",
		}).One(&result)
		if err != nil {
			tb.Fatalf("failed to count notifications for timesheet_shared before request: %v", err)
		}
		beforeCount = result.Count
	}

	// After the request, ensure that at least one new timesheet_shared notification was created.
	scenario.AfterTestFunc = func(tb testing.TB, app *tests.TestApp, res *http.Response) {
		var result struct {
			Count int64 `db:"count"`
		}
		err := app.DB().NewQuery(`
			SELECT COUNT(*) AS count
			FROM notifications n
			JOIN notification_templates t ON n.template = t.id
			WHERE t.code = {:code}
		`).Bind(dbx.Params{
			"code": "timesheet_shared",
		}).One(&result)
		if err != nil {
			tb.Fatalf("failed to count notifications for timesheet_shared after request: %v", err)
		}
		if result.Count <= beforeCount {
			tb.Fatalf("expected timesheet_shared notifications to be created by sharing route, before=%d after=%d", beforeCount, result.Count)
		}
	}

	scenario.Test(t)
}

// TestAddTimesheetReviewer_InactiveReviewerFails verifies that adding an inactive
// user as a reviewer is rejected.
func TestAddTimesheetReviewer_InactiveReviewerFails(t *testing.T) {
	// Use the same timesheet id as other tests.
	const timesheetID = "aeyl94og4xmnpq4"

	// Approver is the sharer for timesheet sharing.
	approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// u_inactive is an inactive user (admin_profiles.active=false)
	const inactiveUserID = "u_inactive"

	body := fmt.Sprintf(`{"time_sheet": "%s", "reviewer": "%s"}`, timesheetID, inactiveUserID)

	scenario := tests.ApiScenario{
		Name:   "adding inactive user as timesheet reviewer fails",
		Method: http.MethodPost,
		URL:    "/api/collections/time_sheet_reviewers/records",
		Body:   strings.NewReader(body),
		Headers: map[string]string{
			"Authorization": approverToken,
			"Content-Type":  "application/json",
		},
		ExpectedStatus: http.StatusBadRequest,
		ExpectedContent: []string{
			`"reviewer":{"code":"reviewer_not_active"`,
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}
