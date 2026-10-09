package main

import (
	"net/http"
	"testing"
)

// A request bills the client name its invoicing profile chooses. Changing the
// alias or that choice changes the billed name, so an open review becomes stale.
func TestJobRequestBillsProfileClientName(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	id := createWorkflowRequest(t, app)
	completeWorkflowRequest(t, app, id)
	related := func(path string) (map[string]any, string) {
		t.Helper()
		detail := clientSetupJSON(t, app, http.MethodGet, path, workflowAuthor, nil, http.StatusOK)
		token, _ := detail["review_token"].(string)
		return detail["related"].(map[string]any), token
	}

	before, beforeToken := related("/api/job_requests/" + id)
	if before["bill_to_name"] != "Sepulchi & Co" {
		t.Fatalf("bill_to_name = %#v; want the official name", before["bill_to_name"])
	}

	client := workflowRecord(t, app, "clients", "pqpd90fqd5ohjcs")
	client.Set("alias", "Sepulchi Diner")
	if err := app.Save(client); err != nil {
		t.Fatal(err)
	}
	request := workflowRecord(t, app, "job_requests", id)
	profile := workflowRecord(t, app, "client_invoicing_information", request.GetString("invoicing_information"))
	profile.Set("billing_name", "alias")
	if err := app.Save(profile); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"/api/job_requests/" + id, "/api/job_requests/" + id + "/print"} {
		after, afterToken := related(path)
		if after["bill_to_name"] != "Sepulchi Diner" {
			t.Fatalf("%s bill_to_name = %#v; want the alias", path, after["bill_to_name"])
		}
		// The client, who signs, keeps its official name.
		if name := after["client"].(map[string]any)["name"]; name != "Sepulchi & Co" {
			t.Fatalf("%s client name = %#v", path, name)
		}
		if afterToken == beforeToken {
			t.Fatalf("%s review token did not change with the billed name", path)
		}
	}
}
