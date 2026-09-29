package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

func TestProjectValueUpdates(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, id, patch string
		want            int
	}{
		{"active unchanged save needs value", "pvactivezero001", `{}`, 400},
		{"closed unchanged save needs value", "pvclosedzero001", `{}`, 400},
		{"active T&M edit needs value", "pvactivezero001", `{"description":"Edited project"}`, 400},
		{"closed T&M edit needs value", "pvclosedzero001", `{"description":"Edited project"}`, 400},
		{"close T&M project needs value", "pvactivezero001", `{"status":"Closed"}`, 400},
		{"reopen T&M project needs value", "pvclosedzero001", `{"status":"Active"}`, 400},
		{"reactivate T&M project needs value", "pvcancelzero001", `{"status":"Active"}`, 400},
		{"cancel without value", "pvactivezero001", `{"status":"Cancelled"}`, 200},
		{"cancelled edit without value", "pvcancelzero001", `{"description":"Edited project"}`, 200},
		{"remove T&M flag still needs value", "pvactivezero001", `{"time_and_materials":false}`, 400},
		{"set positive value with T&M", "pvactivezero001", `{"project_value":1}`, 200},
		{"set positive value without T&M", "pvactivezero001", `{"project_value":1,"time_and_materials":false}`, 200},
		{"clear positive value fails", "pcactivedated01", `{"project_value":0}`, 400},
		{"negative value fails", "pcactivedated01", `{"project_value":-1}`, 400},
	}
	for _, tc := range cases {
		for _, collectionAPI := range []bool{false, true} {
			name, method, url, body := tc.name+" custom API", http.MethodPut, "/api/jobs/"+tc.id, `{"job":`+tc.patch+`,"allocations":[{"division":"fy4i9poneukvq9u","hours":10}]}`
			if collectionAPI {
				name, method, url, body = tc.name+" collection API", http.MethodPatch, "/api/collections/jobs/records/"+tc.id, tc.patch
			}
			content := []string{`"id":"` + tc.id + `"`}
			if tc.want == 400 {
				content = []string{`"code":"value_required_for_status"`, `must have a project value greater than zero`}
			}
			scenario := tests.ApiScenario{
				Name:            name,
				Method:          method,
				URL:             url,
				Body:            strings.NewReader(body),
				Headers:         map[string]string{"Authorization": token},
				ExpectedStatus:  tc.want,
				ExpectedContent: content,
				TestAppFactory:  testutils.SetupTestApp,
			}
			scenario.Test(t)
		}
	}
}

func TestProjectValueCreateWithAndWithoutTimeAndMaterials(t *testing.T) {
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
	for _, status := range []string{"Active", "Closed", "Cancelled"} {
		for _, tm := range []bool{false, true} {
			for _, value := range []int{0, 1} {
				for _, collectionAPI := range []bool{false, true} {
					// Build the request from a CSV fixture without changing the stored fixture.
					job := map[string]any{}
					for _, field := range []string{"description", "client", "contact", "manager", "branch", "location", "project_award_date", "project_completion_date", "rate_sheet"} {
						job[field] = fixture.GetString(field)
					}
					job["status"], job["project_value"], job["time_and_materials"] = status, value, tm
					url, payload := "/api/jobs", any(map[string]any{"job": job, "allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 10}}})
					if collectionAPI {
						url, payload = "/api/collections/jobs/records", job
					}
					body, err := json.Marshal(payload)
					if err != nil {
						t.Fatal(err)
					}
					want, content := 200, []string{`"id":`}
					if status != "Cancelled" && value == 0 {
						want, content = 400, []string{`"code":"value_required_for_status"`, `"project_value":`}
					}
					name := fmt.Sprintf("%s value=%d T&M=%t %s", status, value, tm, url)
					scenario := tests.ApiScenario{
						Name:            name,
						Method:          http.MethodPost,
						URL:             url,
						Body:            strings.NewReader(string(body)),
						Headers:         map[string]string{"Authorization": token},
						ExpectedStatus:  want,
						ExpectedContent: content,
						TestAppFactory:  testutils.SetupTestApp,
					}
					scenario.Test(t)
				}
			}
		}
	}
}

func TestProjectValueQuickClose(t *testing.T) {
	token, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	scenario := tests.ApiScenario{
		Name:            "non-imported Time and Materials project needs a value for quick close",
		Method:          http.MethodPost,
		URL:             "/api/jobs/pvactivezero001/close",
		Headers:         map[string]string{"Authorization": token},
		ExpectedStatus:  http.StatusBadRequest,
		ExpectedContent: []string{`"code":"value_required_for_status"`},
		TestAppFactory:  testutils.SetupTestApp,
	}
	scenario.Test(t)
}
