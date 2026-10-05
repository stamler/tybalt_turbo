package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

const workspaceClient = "cwclient0000001"

func newClientWorkspaceTestApp(t *testing.T) *tests.TestApp {
	t.Helper()
	app := testutils.SetupTestApp(t)
	t.Cleanup(app.Cleanup)
	// These CSV rows test historical use in isolation. Loading them here keeps
	// them out of the project queues and totals used by unrelated tests.
	// Insert the stored rows directly, as the main CSV seed loader does.
	for _, table := range []string{"jobs"} {
		f, err := os.Open("test_seed_data/client_workspace/" + table + ".csv")
		if err != nil {
			t.Fatal(err)
		}
		rows, err := csv.NewReader(f).ReadAll()
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		for _, values := range rows[1:] {
			row := dbx.Params{}
			for i, key := range rows[0] {
				row[key] = values[i]
			}
			if _, err := app.DB().Insert(table, row).Execute(); err != nil {
				t.Fatal(err)
			}
		}
	}
	return app
}

func TestClientWorkspaceDetailsAndUsage(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	for _, email := range []string{"author@soup.com", "time@test.com"} {
		t.Run(email, func(t *testing.T) {
			result := clientSetupJSON(t, app, http.MethodGet, "/api/clients/"+workspaceClient, email, nil, http.StatusOK)
			for field, want := range map[string]any{
				"id": workspaceClient, "name": "Workspace Test Client", "address": "100 Test Street",
				"city": "Thunder Bay", "province_state": "ON", "postal_code": "P7A 1A1", "country": "Canada", "phone": "807-555-0100",
				"business_development_lead": "f2j5a8vk006baub", "referencing_jobs_count": float64(2), "outstanding_balance": float64(46),
			} {
				if result[field] != want {
					t.Errorf("%s = %#v; want %#v", field, result[field], want)
				}
			}
			contacts := result["contacts"].([]any)
			if len(contacts) != 3 {
				t.Fatalf("contacts = %d; want only this client's 3 contacts", len(contacts))
			}
			first := contacts[0].(map[string]any)
			for field, want := range map[string]any{
				"id": "cwcontact000001", "client": workspaceClient, "given_name": "Alex", "surname": "Workspace",
				"email": "alex.workspace@example.com", "phone": "807-555-0101", "address": "101 Test Street", "city": "Dryden",
				"province_state": "ON", "postal_code": "P8N 1A1", "country": "Canada",
				"job_count": float64(2), "profile_count": float64(2),
			} {
				if first[field] != want {
					t.Errorf("contact %s = %#v; want %#v", field, first[field], want)
				}
			}
			// Alex is used directly and through invoicing on the first record,
			// and through invoicing alone on the second. Neither is counted twice.
			second := contacts[1].(map[string]any)
			if second["job_count"] != float64(1) || second["profile_count"] != float64(1) {
				t.Fatalf("wrong direct contact counts: %#v", second)
			}
			third := contacts[2].(map[string]any)
			if third["job_count"] != float64(0) || third["profile_count"] != float64(0) {
				t.Fatalf("unused contact has references: %#v", third)
			}
			profiles := result["invoicing_profiles"].([]any)
			if len(profiles) != 3 {
				t.Fatalf("profiles = %d; want only this client's 3 setups", len(profiles))
			}
			for _, value := range profiles {
				profile := value.(map[string]any)
				if profile["id"] == "cwinvoice000001" {
					for field, want := range map[string]any{"name": "Capital projects", "client": workspaceClient, "contact": "cwcontact000001", "fax": "807-555-0199", "invoicing_instructions": "Include the purchase order.", "job_count": float64(2)} {
						if profile[field] != want {
							t.Errorf("profile %s = %#v; want %#v", field, profile[field], want)
						}
					}
				} else if profile["job_count"] != float64(0) {
					t.Errorf("unused setup has references: %#v", profile)
				}
			}
		})
	}
}

func TestClientWorkspaceJobDetailsIncludesInvoicing(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	for _, email := range []string{"author@soup.com", "time@test.com"} {
		t.Run(email, func(t *testing.T) {
			result := clientSetupJSON(t, app, http.MethodGet, "/api/jobs/cwjob0000000001/details", email, nil, http.StatusOK)
			for field, want := range map[string]string{
				"invoicing_information":  "cwinvoice000001",
				"invoicing_profile_name": "Capital projects",
				"invoice_contact_name":   "Alex Workspace",
				"invoice_contact_email":  "alex.workspace@example.com",
				"invoice_instructions":   "Include the purchase order.",
			} {
				if result[field] != want {
					t.Errorf("%s = %#v; want %q", field, result[field], want)
				}
			}
		})
	}
}

func TestClientWorkspaceJobDetailsRequiresActiveAccount(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	path := "/api/jobs/cwjob0000000001/details"
	for _, email := range []string{"inactive@example.com", "francesco@mac.com"} {
		t.Run(email, func(t *testing.T) {
			// Existing fixtures cover an inactive account and an account without
			// an admin profile. Both can still hold a valid authentication token.
			clientSetupJSON(t, app, http.MethodGet, path, email, nil, http.StatusForbidden)
		})
	}
	clientSetupJSON(t, app, http.MethodGet, path, "", nil, http.StatusUnauthorized)
}

func TestClientWorkspaceEmptyMissingAndUnauthenticated(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	empty := clientSetupJSON(t, app, http.MethodGet, "/api/clients/cwclient0000002", "author@soup.com", nil, http.StatusOK)
	for _, key := range []string{"contacts", "invoicing_profiles"} {
		items, ok := empty[key].([]any)
		if !ok || len(items) != 0 {
			t.Errorf("%s must be an empty array, got %#v", key, empty[key])
		}
	}
	clientSetupJSON(t, app, http.MethodGet, "/api/clients/missingclient00", "author@soup.com", nil, http.StatusNotFound)
	clientSetupJSON(t, app, http.MethodGet, "/api/clients/"+workspaceClient, "", nil, http.StatusUnauthorized)
	// Use an existing inactive account fixture with a valid token to verify
	// that the combined response follows the invoicing collection's read rule.
	clientSetupJSON(t, app, http.MethodGet, "/api/clients/"+workspaceClient, "inactive@example.com", nil, http.StatusForbidden)
}

func TestClientWorkspaceInvoicingNamesAndPermissions(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	path := "/api/collections/client_invoicing_information/records/"
	unnamed := clientSetupJSON(t, app, http.MethodPost, strings.TrimSuffix(path, "/"), "author@soup.com", map[string]any{"client": workspaceClient, "contact": "cwcontact000002"}, http.StatusOK)
	if unnamed["name"] != "" {
		t.Fatalf("name should be optional: %#v", unnamed)
	}
	id := unnamed["id"].(string)
	for _, name := range []string{"Field services", ""} {
		updated := clientSetupJSON(t, app, http.MethodPatch, path+id, "author@soup.com", map[string]any{"name": name}, http.StatusOK)
		if updated["name"] != name {
			t.Fatalf("name was not saved: %#v", updated)
		}
	}
	clientSetupJSON(t, app, http.MethodPatch, path+id, "author@soup.com", map[string]any{"name": strings.Repeat("x", 121)}, http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodPost, strings.TrimSuffix(path, "/"), "author@soup.com", map[string]any{"client": workspaceClient, "contact": "184qbpajciwrrp2"}, http.StatusBadRequest)
	for _, collection := range []string{"clients", "client_contacts", "client_invoicing_information"} {
		id := map[string]string{"clients": workspaceClient, "client_contacts": "cwcontact000001", "client_invoicing_information": "cwinvoice000001"}[collection]
		base := "/api/collections/" + collection + "/records/"
		validCreate := map[string]map[string]any{
			"clients":                      {"name": "Unauthorized workspace client", "business_development_lead": "f2j5a8vk006baub"},
			"client_contacts":              {"client": workspaceClient, "given_name": "Unauthorized", "surname": "Contact", "email": "unauthorized@example.com"},
			"client_invoicing_information": {"client": workspaceClient, "contact": "cwcontact000001", "name": "Unauthorized setup"},
		}[collection]
		// PocketBase returns 400 when a create rule is not satisfied, and 404
		// when the same rule prevents selecting a record to update or delete.
		createStatus := http.StatusBadRequest
		clientSetupJSON(t, app, http.MethodPost, strings.TrimSuffix(base, "/"), "time@test.com", validCreate, createStatus)
		clientSetupJSON(t, app, http.MethodPatch, base+id, "time@test.com", map[string]any{}, http.StatusNotFound)
		clientSetupJSON(t, app, http.MethodDelete, base+id, "time@test.com", nil, http.StatusNotFound)
	}
	clientSetupJSON(t, app, http.MethodDelete, path+"cwinvoice000001", "author@soup.com", nil, http.StatusNotFound)
	clientSetupJSON(t, app, http.MethodDelete, "/api/collections/client_contacts/records/cwcontact000001", "author@soup.com", nil, http.StatusNotFound)
	clientSetupJSON(t, app, http.MethodDelete, path+id, "author@soup.com", nil, http.StatusNoContent)
}

func clientSetupJSON(t *testing.T, app *tests.TestApp, method, path, email string, data any, status int) map[string]any {
	t.Helper()
	token := ""
	if email != "" {
		auth, err := app.FindAuthRecordByEmail("users", email)
		if err != nil {
			t.Fatal(err)
		}
		token, err = auth.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
	}
	response := performJSONRequest(t, app, method, path, token, data)
	mustStatus(t, response, status)
	result := map[string]any{}
	if status == http.StatusOK {
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
	}
	return result
}
