package main

import (
	"net/http"
	"strings"
	"testing"

	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/tests"
)

// An alias is optional and returned by the client list and details endpoints.
func TestClientAliasSavedAndReturned(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	const actor = "author@soup.com"
	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/clients/records/"+workspaceClient, actor, clientAliasPatch("Workspace Diner"), http.StatusOK)

	details := clientSetupJSON(t, app, http.MethodGet, "/api/clients/"+workspaceClient, actor, nil, http.StatusOK)
	if details["alias"] != "Workspace Diner" {
		t.Fatalf("details alias = %#v", details["alias"])
	}
	for _, profile := range details["invoicing_profiles"].([]any) {
		if value, ok := profile.(map[string]any)["billing_name"]; !ok || value != "" {
			t.Fatalf("profile billing_name = %#v, %v; want blank", value, ok)
		}
	}

	response := performJSONRequest(t, app, http.MethodGet, "/api/clients", clientAliasToken(t, app, actor), nil)
	mustStatus(t, response, http.StatusOK)
	if !strings.Contains(response.Body.String(), `"alias":"Workspace Diner"`) {
		t.Fatal("client list omits the alias")
	}

	clientSetupJSON(t, app, http.MethodPatch, "/api/collections/clients/records/"+workspaceClient, actor, clientAliasPatch(strings.Repeat("x", 121)), http.StatusBadRequest)
}

// A profile bills the alias only when its client has one, and the alias stays
// while a profile bills it.
func TestInvoicingProfileBillingName(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	const actor = "author@soup.com"
	profilePath := "/api/collections/client_invoicing_information/records/cwinvoice000001"
	clientPath := "/api/collections/clients/records/" + workspaceClient
	token := clientAliasToken(t, app, actor)

	response := performJSONRequest(t, app, http.MethodPatch, profilePath, token, map[string]any{"billing_name": "alias"})
	mustStatus(t, response, http.StatusBadRequest)
	if !strings.Contains(response.Body.String(), `"alias_missing"`) {
		t.Fatalf("missing alias error: %s", response.Body.String())
	}
	clientSetupJSON(t, app, http.MethodPatch, profilePath, actor, map[string]any{"billing_name": "nickname"}, http.StatusBadRequest)
	clientSetupJSON(t, app, http.MethodPost, "/api/collections/client_invoicing_information/records", actor, map[string]any{
		"client": workspaceClient, "contact": "cwcontact000002", "billing_name": "alias",
	}, http.StatusBadRequest)

	clientSetupJSON(t, app, http.MethodPatch, clientPath, actor, clientAliasPatch("Workspace Diner"), http.StatusOK)
	profile := clientSetupJSON(t, app, http.MethodPatch, profilePath, actor, map[string]any{"billing_name": "alias"}, http.StatusOK)
	if profile["billing_name"] != "alias" {
		t.Fatalf("billing_name = %#v", profile["billing_name"])
	}

	// Changing the alias keeps the choice; clearing it, even to spaces, is refused.
	clientSetupJSON(t, app, http.MethodPatch, clientPath, actor, clientAliasPatch("Workspace Café"), http.StatusOK)
	for _, alias := range []string{"", "   "} {
		response = performJSONRequest(t, app, http.MethodPatch, clientPath, token, clientAliasPatch(alias))
		mustStatus(t, response, http.StatusBadRequest)
		if !strings.Contains(response.Body.String(), `"alias_in_use"`) {
			t.Fatalf("alias in use error: %s", response.Body.String())
		}
	}

	client, err := app.FindRecordById("clients", workspaceClient)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := app.FindRecordById("client_invoicing_information", "cwinvoice000001")
	if err != nil {
		t.Fatal(err)
	}
	if got := utilities.BillingName(client, stored); got != "Workspace Café" {
		t.Fatalf("billing name = %q", got)
	}

	clientSetupJSON(t, app, http.MethodPatch, profilePath, actor, map[string]any{"billing_name": "name"}, http.StatusOK)
	clientSetupJSON(t, app, http.MethodPatch, clientPath, actor, clientAliasPatch(""), http.StatusOK)
}

func TestBillingNameChoosesClientName(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	client, err := app.FindRecordById("clients", workspaceClient)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := app.FindRecordById("client_invoicing_information", "cwinvoice000001")
	if err != nil {
		t.Fatal(err)
	}
	name := client.GetString("name")
	for _, scenario := range []struct {
		alias, choice, want string
	}{
		{"", "", name},
		{"Diner", "", name},
		{"Diner", "name", name},
		{"Diner", "alias", "Diner"},
		{"  Diner  ", "alias", "Diner"},
		// Validation prevents this state; the official name is the safe fallback.
		{" ", "alias", name},
	} {
		client.Set("alias", scenario.alias)
		profile.Set("billing_name", scenario.choice)
		if got := utilities.BillingName(client, profile); got != scenario.want {
			t.Errorf("alias %q, choice %q: got %q, want %q", scenario.alias, scenario.choice, got, scenario.want)
		}
	}
	if got := utilities.BillingName(client, nil); got != name {
		t.Errorf("no profile: got %q", got)
	}
}

// Absorbing a client moves its profiles to the kept client. A profile billing an
// alias needs the kept client to have one.
func TestClientAbsorbKeepsBilledAlias(t *testing.T) {
	app := newClientWorkspaceTestApp(t)
	const source, target = "pqpd90fqd5ohjcs", "lb0fnenkeyitsny"
	sourceClient, err := app.FindRecordById("clients", source)
	if err != nil {
		t.Fatal(err)
	}
	sourceClient.Set("alias", "Sepulchi Diner")
	if err := app.Save(sourceClient); err != nil {
		t.Fatal(err)
	}
	profile, err := app.FindRecordById("client_invoicing_information", "painvoice000001")
	if err != nil {
		t.Fatal(err)
	}
	profile.Set("billing_name", "alias")
	if err := app.Save(profile); err != nil {
		t.Fatal(err)
	}

	absorbPath := "/api/clients/" + target + "/absorb"
	body := map[string]any{"ids_to_absorb": []string{source}}
	response := performJSONRequest(t, app, http.MethodPost, absorbPath, clientAliasToken(t, app, "book@keeper.com"), body)
	mustStatus(t, response, http.StatusBadRequest)
	if !strings.Contains(response.Body.String(), "Give the kept client an alias first") {
		t.Fatalf("absorb error: %s", response.Body.String())
	}
	if _, err := app.FindRecordById("clients", source); err != nil {
		t.Fatal("refused absorb removed the source client")
	}

	targetClient, err := app.FindRecordById("clients", target)
	if err != nil {
		t.Fatal(err)
	}
	targetClient.Set("alias", "Mobilia")
	if err := app.Save(targetClient); err != nil {
		t.Fatal(err)
	}
	response = performJSONRequest(t, app, http.MethodPost, absorbPath, clientAliasToken(t, app, "book@keeper.com"), body)
	mustStatus(t, response, http.StatusOK)
	profile, err = app.FindRecordById("client_invoicing_information", "painvoice000001")
	if err != nil {
		t.Fatal(err)
	}
	if profile.GetString("client") != target || profile.GetString("billing_name") != "alias" {
		t.Fatalf("profile after absorb: client %s, billing_name %s", profile.GetString("client"), profile.GetString("billing_name"))
	}
}

func clientAliasToken(t *testing.T, app *tests.TestApp, email string) string {
	t.Helper()
	auth, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		t.Fatal(err)
	}
	token, err := auth.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// clientAliasPatch also sets a business development lead with the busdev claim,
// which client saves require and the workspace fixture's lead lacks.
func clientAliasPatch(alias string) map[string]any {
	return map[string]any{"alias": alias, "business_development_lead": "4r70mfovf22m9uh"}
}
