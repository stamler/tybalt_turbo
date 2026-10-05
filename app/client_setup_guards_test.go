package main

import (
	"net/http"
	"testing"

	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

func TestClientSetupSharedReferencesAndModelGuards(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	const actor = "author@soup.com"
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/client_contacts/records/cwcontact000001", actor, map[string]any{"client": "cwclient0000002"}, http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/client_invoicing_information/records/cwinvoice000001", actor, map[string]any{"client": "pqpd90fqd5ohjcs", "contact": "184qbpajciwrrp2"}, http.StatusBadRequest)
	for _, pair := range [][2]string{{"clients", workspaceClient}, {"client_contacts", "cwcontact000001"}, {"client_invoicing_information", "cwinvoice000001"}} {
		record, err := app.FindRecordById(pair[0], pair[1])
		if err != nil {
			t.Fatal(err)
		}
		if err = app.Delete(record); err == nil {
			t.Fatalf("model delete accepted used %s", pair[0])
		}
	}

	record, err := app.FindRecordById("client_invoicing_information", "cwinvoice000003")
	if err != nil {
		t.Fatal(err)
	}
	original := record.GetString("creator")
	record.Set("creator", "f2j5a8vk006baub")
	if err := app.Save(record); err == nil {
		t.Fatal("model save changed the profile creator")
	}
	stored, err := app.FindRecordById("client_invoicing_information", record.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("creator") != original {
		t.Fatalf("profile creator changed: %s -> %s", original, stored.GetString("creator"))
	}
}

// Deleting an unused client still removes its unused contacts, as on main.
func TestClientSetupDeleteClientCascadesUnusedContacts(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	contacts, err := app.FindCollectionByNameOrId("client_contacts")
	if err != nil {
		t.Fatal(err)
	}
	// The empty fixture client has no contacts; give it one unused contact.
	contact := core.NewRecord(contacts)
	contact.Set("client", "cwclient0000002")
	contact.Set("given_name", "Unused")
	contact.Set("surname", "Contact")
	if err := app.Save(contact); err != nil {
		t.Fatal(err)
	}
	clientSetupJSON(t, app, http.MethodDelete, "/api/collections/clients/records/cwclient0000002", "author@soup.com", nil, http.StatusNoContent)
	if _, err := app.FindRecordById("client_contacts", contact.Id); err == nil {
		t.Fatal("unused contact must be deleted with its client")
	}
}

// Collection rules govern ordinary users; superusers keep dashboard access.
func TestClientSetupSuperuserMaintenance(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	token, err := testutils.GenerateRecordToken("_superusers", "test@example.com")
	if err != nil {
		t.Fatal(err)
	}
	response := performJSONRequest(t, app, http.MethodPatch, "/api/collections/client_contacts/records/cwcontact000003", token, map[string]any{"surname": "Renamed"})
	mustStatus(t, response, http.StatusOK)
	response = performJSONRequest(t, app, http.MethodPost, "/api/collections/client_invoicing_information/records", token, map[string]any{
		"client": workspaceClient, "contact": "cwcontact000002", "name": "Dashboard profile",
	})
	mustStatus(t, response, http.StatusOK)
}

func TestClientSetupRelationValidation(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	collection, err := app.FindCollectionByNameOrId("jobs")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []struct {
		name, client, contact, profile string
		valid                          bool
	}{
		{"complete", workspaceClient, "cwcontact000001", "cwinvoice000001", true},
		{"legacy blank", "", "", "", true},
		{"foreign contact", workspaceClient, "184qbpajciwrrp2", "cwinvoice000001", false},
		{"foreign profile", "pqpd90fqd5ohjcs", "184qbpajciwrrp2", "cwinvoice000001", false},
		{"missing profile", workspaceClient, "cwcontact000001", "missingprofile1", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			// This unsaved record is the candidate under test, not database seed data.
			record := core.NewRecord(collection)
			record.Set("client", scenario.client)
			record.Set("contact", scenario.contact)
			record.Set("invoicing_information", scenario.profile)
			if err := utilities.ValidateClientReferences(app, record); (err == nil) != scenario.valid {
				t.Fatalf("valid=%v: %v", scenario.valid, err)
			}
		})
	}
}
