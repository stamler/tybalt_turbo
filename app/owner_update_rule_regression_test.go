package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const profilesUpdateRuleHardened = `@request.auth.id != "" &&
uid = @request.auth.id &&
@request.body.uid:changed = false`

const timeEntriesUpdateRuleHardened = `// the creating user can edit if the entry is not yet part of a timesheet
uid = @request.auth.id && tsid = "" &&

// uid must not change after create
@request.body.uid:changed = false &&

// if present, the category belongs to the job, otherwise is blank
(
  // the job is unchanged, compare the new category to job
  ( @request.body.job:isset = false && @request.body.category.job = job ) ||
  // the job has changed, compare the new category to the new job
  ( @request.body.job:isset = true && @request.body.category.job = @request.body.job ) ||
  @request.body.category = ""
)`

func setUpdateRule(t testing.TB, app *tests.TestApp, collectionName string, rule string) {
	t.Helper()
	ruleCopy := rule
	setUpdateRulePtr(t, app, collectionName, &ruleCopy)
}

func clearUpdateRule(t testing.TB, app *tests.TestApp, collectionName string) {
	t.Helper()
	setUpdateRulePtr(t, app, collectionName, nil)
}

func setUpdateRulePtr(t testing.TB, app *tests.TestApp, collectionName string, rule *string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("failed finding collection %s: %v", collectionName, err)
	}

	collection.UpdateRule = rule
	if err := app.SaveNoValidate(collection); err != nil {
		t.Fatalf("failed updating %s updateRule: %v", collectionName, err)
	}
}

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
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			setUpdateRule(tb, app, "profiles", profilesUpdateRuleHardened)
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
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			setUpdateRule(tb, app, "time_entries", timeEntriesUpdateRuleHardened)
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
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			clearUpdateRule(tb, app, "time_sheets")
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
		BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, e *core.ServeEvent) {
			clearUpdateRule(tb, app, "time_sheets")
		},
		TestAppFactory: testutils.SetupTestApp,
	}

	scenario.Test(t)
}
