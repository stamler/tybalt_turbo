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
		{"time", "time@test.com", http.StatusOK},
		{"job without time", "invoice-job@example.com", http.StatusOK},
		{"job and time", workflowAuthor, http.StatusOK},
		{"no claim", "u_no_claims@example.com", http.StatusBadRequest},
		{"admin without job or time", "admin.only@example.com", http.StatusBadRequest},
		{"inactive time", "invoice-inactive@example.com", http.StatusBadRequest},
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
	result := clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "time@test.com", map[string]any{
		"client": workspaceClient, "contact": "cwcontact000002", "name": "Capital projects",
		"creator": "f2j5a8vk006baub", "created": forgedDate,
	}, http.StatusOK)
	if result["creator"] != "rzr98oadsp9qc11" {
		t.Fatalf("caller could spoof creator: %#v", result)
	}
	created, err := time.Parse("2006-01-02 15:04:05.000Z", result["created"].(string))
	if err != nil || created.Before(before) || created.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("creation time must come from the server: %#v (%v)", result["created"], err)
	}
	second := clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "time@test.com", map[string]any{
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
			if profile["creator"] != "rzr98oadsp9qc11" || profile["creator_name"] != "Tester Time" || profile["created"] != result["created"] {
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
		{"creator": "f2j5a8vk006baub"}, {"creator": ""},
	} {
		clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/"+id, "invoice-job@example.com", change, http.StatusBadRequest)
	}
	// PocketBase ignores supplied auto-date fields before the update hook.
	// A successful request must still leave the original creation time intact.
	timestampAttempt := clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/"+id, "invoice-job@example.com", map[string]any{"created": forgedDate}, http.StatusOK)
	if timestampAttempt["created"] != result["created"] {
		t.Fatal("caller changed the creation timestamp")
	}
	updated := clientSetupJSON(t, app, http.MethodPatch, invoicingRecords+"/"+id, "invoice-job@example.com", map[string]any{"invoicing_instructions": "Send a PDF."}, http.StatusOK)
	if updated["creator"] != result["creator"] || updated["created"] != result["created"] {
		t.Fatalf("job edit changed creation audit: %#v", updated)
	}
	clientSetupJSON(t, app, http.MethodDelete, invoicingRecords+"/"+id, "invoice-job@example.com", nil, http.StatusNoContent)
}

func TestInvoicingProfileTimeCreationValidatesClientAndContact(t *testing.T) {
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
			clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "time@test.com", map[string]any{
				"client": scenario.client, "contact": scenario.contact,
			}, http.StatusBadRequest)
		})
	}
}

func TestInvoicingProfileCreatorCannotEditOwnProfileOrClientContacts(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()
	created := clientSetupJSON(t, app, http.MethodPost, invoicingRecords, "time@test.com", map[string]any{
		"client": workspaceClient, "contact": "cwcontact000002", "name": "Creator cannot edit",
	}, http.StatusOK)
	path := invoicingRecords + "/" + created["id"].(string)
	clientSetupJSON(t, app, http.MethodPatch, path, "time@test.com", map[string]any{"name": "Unauthorized change"}, http.StatusNotFound)
	clientSetupJSON(t, app, http.MethodDelete, path, "time@test.com", nil, http.StatusNotFound)
	stored := workflowRecord(t, app, "client_invoicing_information", created["id"].(string))
	if stored.GetString("name") != "Creator cannot edit" || stored.GetString("creator") != "rzr98oadsp9qc11" {
		t.Fatal("creator changed the saved profile")
	}
	for _, scenario := range []struct {
		collection, id string
		create, update map[string]any
	}{
		{"clients", workspaceClient, map[string]any{"name": "Unauthorized client", "business_development_lead": "f2j5a8vk006baub"}, map[string]any{"name": "Unauthorized rename"}},
		{"client_contacts", "cwcontact000002", map[string]any{"client": workspaceClient, "given_name": "Unauthorized", "surname": "Contact", "email": "unauthorized@example.com"}, map[string]any{"email": "unauthorized@example.com"}},
	} {
		base := "/api/collections/" + scenario.collection + "/records"
		clientSetupJSON(t, app, http.MethodPost, base, "time@test.com", scenario.create, http.StatusBadRequest)
		clientSetupJSON(t, app, http.MethodPatch, base+"/"+scenario.id, "time@test.com", scenario.update, http.StatusNotFound)
		clientSetupJSON(t, app, http.MethodDelete, base+"/"+scenario.id, "time@test.com", nil, http.StatusNotFound)
	}
}
