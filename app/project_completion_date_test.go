package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestProjectCompletionDateUpdates(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, id, patch, code string
	}{
		{"active edit needs date", "pcactiveblank01", `{"description":"Changed description"}`, "required_for_active_project"},
		{"active status-only save needs date", "pcactiveblank01", `{"status":"Active"}`, "required_for_active_project"},
		{"active edit accepts estimate", "pcactiveblank01", `{"project_completion_date":"2027-12-31"}`, ""},
		{"active edit keeps saved estimate", "pcactivedated01", `{"description":"Changed description"}`, ""},
		{"active date cannot be cleared", "pcactivedated01", `{"project_completion_date":""}`, "required_for_active_project"},
		{"past date accepted", "pcactiveblank01", `{"project_completion_date":"2025-02-01"}`, ""},
		{"completion before award rejected", "pcactiveblank01", `{"project_completion_date":"2025-01-14"}`, "date_order_invalid"},
		{"completion on award date accepted", "pcactiveblank01", `{"project_completion_date":"2025-01-15"}`, ""},
		{"award moved after completion rejected", "pcactivedated01", `{"project_award_date":"2028-01-01"}`, "date_order_invalid"},
		{"award moved to completion accepted", "pcactivedated01", `{"project_award_date":"2027-12-31"}`, ""},
		{"both dates changed together", "pcactivedated01", `{"project_award_date":"2028-01-01","project_completion_date":"2028-01-02"}`, ""},
		{"closed completion before award rejected", "pcclosedblank01", `{"project_completion_date":"2025-01-14"}`, "date_order_invalid"},
		{"cancelled completion before award rejected", "pccancelblank01", `{"project_completion_date":"2025-01-14"}`, "date_order_invalid"},
		{"legacy missing award accepts completion", "cjf0kt0defhq480", `{"project_completion_date":"2025-02-01"}`, ""},
		{"unchanged dates checked on close", "pcorderearly001", `{"status":"Closed"}`, "date_order_invalid"},
		{"unchanged dates checked on save", "pcorderearly001", `{}`, "date_order_invalid"},
		{"leap day accepted", "pcactiveblank01", `{"project_completion_date":"2028-02-29"}`, ""},
		{"invalid leap day rejected", "pcactiveblank01", `{"project_completion_date":"2027-02-29"}`, "invalid_date_format"},
		{"impossible date rejected", "pcactiveblank01", `{"project_completion_date":"2026-02-31"}`, "invalid_date_format"},
		{"wrong format rejected", "pcactiveblank01", `{"project_completion_date":"2026-2-01"}`, "invalid_date_format"},
		{"timestamp rejected", "pcactiveblank01", `{"project_completion_date":"2026-02-01T00:00:00Z"}`, "invalid_date_format"},
		{"close without date", "pcactiveblank01", `{"status":"Closed"}`, ""},
		{"close with other edits without date", "pcactiveblank01", `{"status":"Closed","description":"Changed description"}`, ""},
		{"cancel without date", "pcactiveblank01", `{"status":"Cancelled"}`, ""},
		{"closed edit without date", "pcclosedblank01", `{"description":"Changed description"}`, ""},
		{"cancelled edit without date", "pccancelblank01", `{"description":"Changed description"}`, ""},
		{"reopen closed needs date", "pcclosedblank01", `{"status":"Active"}`, "required_for_active_project"},
		{"reopen cancelled needs date", "pccancelblank01", `{"status":"Active"}`, "required_for_active_project"},
		{"reopen with date", "pcclosedblank01", `{"status":"Active","project_completion_date":"2027-12-31"}`, ""},
		{"reactivate with date", "pccancelblank01", `{"status":"Active","project_completion_date":"2027-12-31"}`, ""},
		{"closed supplied date must be valid", "pcclosedblank01", `{"project_completion_date":"2026-13-01"}`, "invalid_date_format"},
		{"cancelled supplied date must be valid", "pccancelblank01", `{"project_completion_date":"2026-04-31"}`, "invalid_date_format"},
		{"proposal edit without date", "pcproposal00001", `{"description":"Changed description"}`, ""},
	}
	for _, tc := range cases {
		for _, collectionAPI := range []bool{false, true} {
			name, method, url, body := tc.name+" custom API", http.MethodPut, "/api/jobs/"+tc.id, `{"job":`+tc.patch+`,"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`
			if collectionAPI {
				name, method, url, body = tc.name+" collection API", http.MethodPatch, "/api/collections/jobs/records/"+tc.id, tc.patch
			}
			status, content := http.StatusOK, []string{`"id":"` + tc.id + `"`}
			if tc.code != "" {
				status, content = http.StatusBadRequest, []string{`"project_completion_date":`, `"code":"` + tc.code + `"`}
			}
			scenario := tests.ApiScenario{
				Name: name, Method: method, URL: url, Body: strings.NewReader(body), Headers: map[string]string{"Authorization": token}, ExpectedStatus: status, ExpectedContent: content, TestAppFactory: testutils.SetupTestApp,
				AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
					// Read the stored fixture after the request to check persistence or rollback.
					record, err := app.FindRecordById("jobs", tc.id)
					if err != nil {
						tb.Fatal(err)
					}
					expected := ""
					if tc.id == "pcactivedated01" || tc.id == "pcorderearly001" {
						expected = "2027-12-31"
					}
					var patch map[string]any
					if err := json.Unmarshal([]byte(tc.patch), &patch); err != nil {
						tb.Fatal(err)
					}
					if tc.code == "" {
						if date, ok := patch["project_completion_date"].(string); ok {
							expected = date
						}
					}
					if got := record.GetString("project_completion_date"); got != expected {
						tb.Fatalf("saved date = %q, want %q", got, expected)
					}
					if award, ok := patch["project_award_date"].(string); ok {
						if tc.code != "" {
							award = "2025-01-15"
						}
						if got := record.GetString("project_award_date"); got != award {
							tb.Fatalf("saved award date = %q, want %q", got, award)
						}
					}
				},
			}
			scenario.Test(t)
		}
	}
}

func TestProjectCompletionDateCreate(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()
	fixture, err := app.FindRecordById("jobs", "pcactiveblank01")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, status, date, parent string
		want                       int
	}{
		{"active missing", "Active", "", "", 400},
		{"active estimate", "Active", "2027-12-31", "", 200},
		{"active before award", "Active", "2025-01-14", "", 400},
		{"active on award date", "Active", "2025-01-15", "", 200},
		{"active invalid", "Active", "2026-02-31", "", 400},
		{"closed missing", "Closed", "", "", 200},
		{"cancelled missing", "Cancelled", "", "", 200},
		{"child missing", "Active", "", "pcactiveblank01", 400},
		{"child estimate", "Active", "2027-12-31", "pcactiveblank01", 200},
	}
	for _, tc := range cases {
		for _, collectionAPI := range []bool{false, true} {
			// Build a request from a CSV fixture; do not insert setup records.
			job := map[string]any{}
			for _, field := range []string{"description", "client", "contact", "manager", "branch", "location", "project_award_date", "rate_sheet"} {
				job[field] = fixture.GetString(field)
			}
			job["project_value"] = fixture.GetInt("project_value")
			job["status"], job["project_completion_date"], job["time_and_materials"], job["parent"] = tc.status, tc.date, true, tc.parent
			url, name, payload := "/api/jobs", tc.name+" custom API", any(map[string]any{"job": job, "allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 10}}})
			if collectionAPI {
				url, name, payload = "/api/collections/jobs/records", tc.name+" collection API", job
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			content := []string{`"id":`}
			if tc.want == 400 {
				code := "required_for_active_project"
				if tc.date != "" {
					code = "invalid_date_format"
				}
				if tc.date == "2025-01-14" {
					code = "date_order_invalid"
				}
				content = []string{`"project_completion_date":`, `"code":"` + code + `"`}
			}
			scenario := tests.ApiScenario{Name: name, Method: http.MethodPost, URL: url, Body: strings.NewReader(string(body)), Headers: map[string]string{"Authorization": token}, ExpectedStatus: tc.want, ExpectedContent: content, TestAppFactory: testutils.SetupTestApp}
			scenario.Test(t)
		}
	}
}

func TestProjectCompletionDateQuickCloseAndDetails(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pcactiveblank01", "pcimportblank01"} {
		scenario := tests.ApiScenario{Name: "quick close without date " + id, Method: http.MethodPost, URL: "/api/jobs/" + id + "/close", Headers: map[string]string{"Authorization": token}, ExpectedStatus: 200, ExpectedContent: []string{`"status":"Closed"`}, TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
				record, err := app.FindRecordById("jobs", id)
				if err != nil {
					tb.Fatal(err)
				}
				if record.GetString("project_completion_date") != "" {
					tb.Fatal("quick close must not fill the completion date")
				}
			},
		}
		scenario.Test(t)
	}
	scenario := tests.ApiScenario{Name: "details returns completion date", Method: http.MethodGet, URL: "/api/jobs/pcactivedated01/details", Headers: map[string]string{"Authorization": token}, ExpectedStatus: 200, ExpectedContent: []string{`"project_completion_date":"2027-12-31"`}, TestAppFactory: testutils.SetupTestApp}
	scenario.Test(t)
}

func TestProjectCompletionDateInvalidAward(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, collectionAPI := range []bool{false, true} {
		patch := `{"project_award_date":"2025-02-31"}`
		method, url, body := http.MethodPut, "/api/jobs/pcactivedated01", `{"job":`+patch+`,"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`
		if collectionAPI {
			method, url, body = http.MethodPatch, "/api/collections/jobs/records/pcactivedated01", patch
		}
		scenario := tests.ApiScenario{
			Name:            "invalid award date cannot be used for comparison " + method,
			Method:          method,
			URL:             url,
			Body:            strings.NewReader(body),
			Headers:         map[string]string{"Authorization": token},
			ExpectedStatus:  http.StatusBadRequest,
			ExpectedContent: []string{`"project_award_date":{"code":"invalid_date_format"`},
			TestAppFactory:  testutils.SetupTestApp,
		}
		scenario.Test(t)
	}
}
