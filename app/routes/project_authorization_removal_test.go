package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"
)

func TestRetiredProjectAuthorizationRoutesAreUnavailable(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)
	token := authTokenForEmail(t, app, "author@soup.com")
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/jobs/project_authorization/missing"},
		{http.MethodGet, "/api/jobs/project_authorization/pending"},
		{http.MethodGet, "/api/jobs/project_authorization/rejected"},
		{http.MethodPost, "/api/jobs/cjf0kt0defhq480/project_authorization_doc"},
		{http.MethodDelete, "/api/jobs/cjf0kt0defhq480/project_authorization_doc"},
		{http.MethodPost, "/api/jobs/cjf0kt0defhq480/project_authorization_doc_hash/audit"},
		{http.MethodPost, "/api/jobs/cjf0kt0defhq480/project_authorization_doc_hash/replace"},
		{http.MethodPost, "/api/jobs/cjf0kt0defhq480/project_authorization/approve"},
		{http.MethodPost, "/api/jobs/cjf0kt0defhq480/project_authorization/reject"},
		{http.MethodPost, "/api/jobs/cjf0kt0defhq480/project_authorization/revoke"},
	} {
		t.Run(route.method+route.path, func(t *testing.T) {
			response := performClaimsJSONRequest(t, app, route.method, route.path, token, nil)
			if response.Code != http.StatusNotFound {
				t.Fatalf("retired route status = %d, want 404; body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestJobDetailsExcludeRetiredProjectAuthorizationFields(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)
	token := authTokenForEmail(t, app, "author@soup.com")
	response := performClaimsJSONRequest(t, app, http.MethodGet, "/api/jobs/cjf0kt0defhq480/details", token, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("job details status = %d; body=%s", response.Code, response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	for field := range fields {
		if strings.HasPrefix(field, "pa_") || strings.HasPrefix(field, "project_authorization_doc") || field == "branch_manager_id" {
			t.Fatalf("retired response field remains: %s", field)
		}
	}
}
