package main

import (
	"net/http"
	"testing"
	"time"
	"tybalt/internal/testutils"
)

const invoicingRecords = "/api/collections/client_invoicing_information/records"

func TestInvoicingProfileCreatePermissions(t *testing.T) {
	for _, scenario := range []struct {
		name, email string
		status      int
	}{
		{"time", "time@test.com", http.StatusBadRequest},
		{"job and time", "author@soup.com", http.StatusOK},
		{"no claim", "u_no_claims@example.com", http.StatusBadRequest},
		{"admin without job or time", "admin.only@example.com", http.StatusBadRequest},
		{"inactive time", "inactive@example.com", http.StatusBadRequest},
		{"unauthenticated", "", http.StatusBadRequest},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			defer app.Cleanup()
			// Clients, contacts, and users come from CSV fixtures. The POST is
			// the action under test, not fixture setup.
			result := clientSetupJSON(t, app, http.MethodPost, invoicingRecords, scenario.email, map[string]any{
				"client": workspaceClient, "contact": "cwcontact000002", "name": "Field services", "invoicing_instructions": "Include the purchase order.",
			}, scenario.status)
			if scenario.status == http.StatusOK {
				actor, err := app.FindAuthRecordByEmail("users", scenario.email)
				if err != nil {
					t.Fatal(err)
				}
				if result["creator"] != actor.Id {
					t.Fatalf("creator = %#v; want %s", result["creator"], actor.Id)
				}
			} else {
				clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/cwinvoice000003", scenario.email, map[string]any{"name": "Unauthorized change"}, http.StatusNotFound)
				clientSetupJSON(t, app, http.MethodDelete, invoicingRecords+"/cwinvoice000003", scenario.email, nil, http.StatusNotFound)
			}
		})
	}
}

func TestInvoicingProfileCreationAuditAndMultipleProfiles(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()
	before := time.Now().UTC().Add(-time.Second)
	const forgedDate = "2001-01-01 00:00:00.000Z"
	result := clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "author@soup.com", map[string]any{
		"client": workspaceClient, "contact": "cwcontact000002", "name": "Capital projects",
		"creator": "rzr98oadsp9qc11", "created": forgedDate,
	}, http.StatusOK)
	if result["creator"] != "f2j5a8vk006baub" {
		t.Fatalf("caller could spoof creator: %#v", result)
	}
	created, err := time.Parse("2006-01-02 15:04:05.000Z", result["created"].(string))
	if err != nil || created.Before(before) || created.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("creation time must come from the server: %#v (%v)", result["created"], err)
	}
	second := clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "author@soup.com", map[string]any{
		"client": workspaceClient, "contact": "cwcontact000002", "name": "Maintenance",
	}, http.StatusOK)
	if result["id"] == second["id"] {
		t.Fatal("different profiles for the same contact must stay separate")
	}
	client := clientSetupJSON(t, app, http.MethodGet, "/api/clients/"+workspaceClient, "time@test.com", nil, http.StatusOK)
	found := false
	for _, value := range client["invoicing_profiles"].([]any) {
		profile := value.(map[string]any)
		if profile["id"] == result["id"] {
			found = true
			if profile["creator"] != "f2j5a8vk006baub" || profile["creator_name"] != "Horace Silver" || profile["created"] != result["created"] {
				t.Fatalf("client detail lost creator audit: %#v", profile)
			}
		}
		if profile["id"] == "cwinvoice000001" && (profile["creator"] != "" || profile["creator_name"] != "" || profile["created"] != "2026-10-01 00:00:00.000Z") {
			t.Fatalf("legacy creator must remain unknown: %#v", profile)
		}
	}
	if !found {
		t.Fatal("created profile missing from client detail")
	}

	id := result["id"].(string)
	for _, change := range []map[string]any{
		{"creator": "rzr98oadsp9qc11"}, {"creator": ""},
	} {
		clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/"+id, "author@soup.com", change, http.StatusBadRequest)
	}
	// PocketBase ignores supplied auto-date fields before the update hook.
	// A successful request must still leave the original creation time intact.
	timestampAttempt := clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/"+id, "author@soup.com", map[string]any{"created": forgedDate}, http.StatusOK)
	if timestampAttempt["created"] != result["created"] {
		t.Fatal("caller changed the creation timestamp")
	}
	updated := clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/"+id, "author@soup.com", map[string]any{"invoicing_instructions": "Send a PDF."}, http.StatusOK)
	if updated["creator"] != result["creator"] || updated["created"] != result["created"] {
		t.Fatalf("job edit changed creation audit: %#v", updated)
	}
	clientSetupJSON(t, app, http.MethodDelete, invoicingRecords+"/"+id, "author@soup.com", nil, http.StatusNoContent)
}

func TestInvoicingProfileCreationValidatesClientAndContact(t *testing.T) {
	for _, scenario := range []struct{ name, client, contact string }{
		{"missing contact", workspaceClient, ""},
		{"unknown contact", workspaceClient, "missingcontact1"},
		{"different client", workspaceClient, "184qbpajciwrrp2"},
		{"missing client", "", "cwcontact000002"},
		{"unknown client", "missingclient01", "cwcontact000002"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			app := testutils.SetupTestApp(t)
			defer app.Cleanup()
			clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "author@soup.com", map[string]any{
				"client": scenario.client, "contact": scenario.contact,
			}, http.StatusBadRequest)
		})
	}
}
