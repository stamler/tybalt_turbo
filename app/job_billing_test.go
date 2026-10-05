package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// These tests use the unchanged legacy CSV jobs. They must not use the billing
// setup helper: a blank profile on a stored job is the rollout starting state.
func TestJobBillingOnProjectEdits(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, patch, code, profile string
		status                     int
	}{
		{"old job needs profile", `{"description":"Billing edit"}`, "required", "", 400},
		{"same client profile accepted", `{"description":"Billing edit","invoicing_information":"painvoice000002"}`, "", "painvoice000002", 200},
		{"other client profile rejected", `{"description":"Billing edit","invoicing_information":"painvoice000001"}`, "client_mismatch", "", 400},
		{"unknown profile rejected", `{"invoicing_information":"doesnotexist001"}`, "invalid_reference", "", 400},
		{"other client contact rejected", `{"invoicing_information":"painvoice000002","contact":"184qbpajciwrrp2"}`, "contact_client_mismatch", "", 400},
		// Status-only changes, such as those from the stale and unused job lists,
		// keep working on projects that have not been given a profile yet.
		{"status only change accepted", `{"status":"Closed"}`, "", "", 200},
	} {
		for _, collection := range []bool{false, true} {
			method, url, body := http.MethodPut, "/api/jobs/pcactivedated01", `{"job":`+tc.patch+`,"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`
			if collection {
				method, url, body = http.MethodPatch, "/api/collections/jobs/records/pcactivedated01", tc.patch
			}
			content := []string{`"id":"pcactivedated01"`}
			if tc.code != "" {
				content = []string{`"code":"` + tc.code + `"`}
			}
			scenario := tests.ApiScenario{Name: tc.name + " " + method, Method: method, URL: url, Body: strings.NewReader(body), Headers: map[string]string{"Authorization": token}, ExpectedStatus: tc.status, ExpectedContent: content, TestAppFactory: testutils.SetupTestApp,
				AfterTestFunc: func(tb testing.TB, app *tests.TestApp, _ *http.Response) {
					r, e := app.FindRecordById("jobs", "pcactivedated01")
					if e != nil {
						tb.Fatal(e)
					}
					if r.GetString("invoicing_information") != tc.profile {
						tb.Fatalf("profile=%q, want %q", r.GetString("invoicing_information"), tc.profile)
					}
				},
			}
			scenario.Test(t)
		}
	}
}

// Closing an older project needs no profile, but setting it Active again runs
// the billing checks: a profile is required, and its contact and profile must
// belong to the job's client.
func TestJobBillingReopening(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, setup, content string
		status               int
	}{
		{"without profile", "", `"invoicing_information":{"code":"required"`, http.StatusBadRequest},
		{"with another client's contact", ", invoicing_information = 'painvoice000002', contact = '184qbpajciwrrp2'", `"contact":{"code":"client_mismatch"`, http.StatusBadRequest},
		{"with valid billing details", ", invoicing_information = 'painvoice000002'", `"status":"Active"`, http.StatusOK},
	} {
		for _, collection := range []bool{false, true} {
			method, url, body := http.MethodPut, "/api/jobs/pcactivedated01", `{"job":{"status":"Active"},"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`
			content := tc.content
			if collection {
				method, url, body = http.MethodPatch, "/api/collections/jobs/records/pcactivedated01", `{"status":"Active"}`
			} else if tc.status == http.StatusOK {
				content = `"id":"pcactivedated01"`
			}
			scenario := tests.ApiScenario{
				Name:            "reopen " + tc.name + " " + method,
				Method:          method,
				URL:             url,
				Body:            strings.NewReader(body),
				Headers:         map[string]string{"Authorization": token},
				ExpectedStatus:  tc.status,
				ExpectedContent: []string{content},
				TestAppFactory:  testutils.SetupTestApp,
				BeforeTestFunc: func(tb testing.TB, app *tests.TestApp, _ *core.ServeEvent) {
					// Start from a closed project in the stated condition; this is
					// state, not a request under test.
					if _, err := app.DB().NewQuery("UPDATE jobs SET status = 'Closed'" + tc.setup + " WHERE id = 'pcactivedated01'").Execute(); err != nil {
						tb.Fatal(err)
					}
				},
			}
			scenario.Test(t)
		}
	}
}

func TestJobBillingOnProjectCreation(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()
	fixture, err := app.FindRecordById("jobs", "pcactivedated01")
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []string{"", "painvoice000002", "painvoice000001"} {
		for _, collection := range []bool{false, true} {
			job := map[string]any{}
			// Build a new-job request from the existing CSV fixture, without inserting seed data.
			for _, field := range []string{"description", "client", "contact", "manager", "branch", "location", "project_award_date", "project_completion_date", "rate_sheet", "status"} {
				job[field] = fixture.GetString(field)
			}
			job["project_value"] = fixture.GetFloat("project_value")
			job["invoicing_information"] = profile
			url, payload := "/api/jobs", any(map[string]any{"job": job, "allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 10}}})
			if collection {
				url, payload = "/api/collections/jobs/records", job
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			status, content := 200, []string{`"id":`}
			if profile == "" {
				status, content = 400, []string{`"invoicing_information":{"code":"required"`}
			} else if profile == "painvoice000001" {
				status, content = 400, []string{`"code":"client_mismatch"`}
			}
			scenario := tests.ApiScenario{Name: profile + " " + url, Method: http.MethodPost, URL: url, Body: strings.NewReader(string(body)), Headers: map[string]string{"Authorization": token}, ExpectedStatus: status, ExpectedContent: content, TestAppFactory: testutils.SetupTestApp}
			scenario.Test(t)
		}
	}
}

func TestJobBillingLegacyExceptions(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, method, url, body string }{
		{"read old job", http.MethodGet, "/api/jobs/pcactivedated01/details", ""},
		{"allocation only", http.MethodPut, "/api/jobs/pcactivedated01", `{"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`},
		{"separate close", http.MethodPost, "/api/jobs/pcactivedated01/close", ""},
		{"proposal edit", http.MethodPut, "/api/jobs/pcproposal00001", `{"job":{"description":"Proposal still optional"},"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`},
	} {
		scenario := tests.ApiScenario{Name: tc.name, Method: tc.method, URL: tc.url, Body: strings.NewReader(tc.body), Headers: map[string]string{"Authorization": token}, ExpectedStatus: 200, ExpectedContent: []string{`"id":`}, TestAppFactory: testutils.SetupTestApp}
		scenario.Test(t)
	}
}
