package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// A time entry, expense, or purchase order may have no branch or division, and
// a time entry's employee may have no profile. The job's tabs and summaries
// still list it, with blank codes and names.
func TestJobTabsListRecordsWithoutBranchOrDivision(t *testing.T) {
	app := newWorkflowTestApp(t, false)
	const job = "cjf0kt0defhq480"
	// The fixture's committed time entries on this job already have no branch.
	for _, query := range []string{
		"UPDATE expenses SET branch='', division='' WHERE id='su3hyft6n9rlt7d'",
		"UPDATE purchase_orders SET branch='', division='' WHERE id='25046ft47x49cc2'",
		"UPDATE time_entries SET division='', uid='noprofile000001' WHERE id=(SELECT id FROM time_entries WHERE job='" + job + "' ORDER BY id LIMIT 1)",
	} {
		if _, err := app.DB().NewQuery(query).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	for _, tab := range []string{"time/entries", "expenses/list", "pos/list"} {
		t.Run(tab, func(t *testing.T) {
			result := clientSetupJSON(t, app, http.MethodGet, "/api/jobs/"+job+"/"+tab, workflowAuthor, nil, http.StatusOK)
			if items, _ := result["data"].([]any); len(items) == 0 {
				t.Fatalf("no rows listed: %#v", result)
			}
		})
	}
	auth, err := app.FindAuthRecordByEmail("users", workflowAuthor)
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	for _, summary := range []string{"divisions/summary", "staff/summary"} {
		t.Run(summary, func(t *testing.T) {
			response := performJSONRequest(t, app, http.MethodGet, "/api/jobs/"+job+"/"+summary+"?start_date=2000-01-01&end_date=2100-01-01", token, nil)
			mustStatus(t, response, http.StatusOK)
			var rows []map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
				t.Fatal(err)
			}
			if len(rows) == 0 {
				t.Fatal("no rows listed")
			}
		})
	}
}
