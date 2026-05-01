package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestClaimAssignableUsersRequiresAdminAndExcludesExistingHolders(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")
	noClaimsToken := authTokenForEmail(t, app, "u_no_claims@example.com")

	forbidden := performClaimsJSONRequest(t, app, http.MethodGet, "/api/claims/"+corporateClaimID+"/assignable_users", noClaimsToken, nil)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("non-admin status = %d, want %d; body=%s", forbidden.Code, http.StatusForbidden, forbidden.Body.String())
	}

	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/claims/"+corporateClaimID+"/assignable_users", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("assignable users status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body ClaimAssignableUsers
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode assignable users response: %v", err)
	}
	if !claimAssignableUsersContains(body.AssignableUsers, "u_no_claims") {
		t.Fatal("expected user without corporate claim to be assignable")
	}
	if claimAssignableUsersContains(body.AssignableUsers, "u_inactive") {
		t.Fatal("expected inactive user without corporate claim to be excluded")
	}
	if claimAssignableUsersContains(body.AssignableUsers, "u_corp_claim") {
		t.Fatal("expected existing corporate claim holder to be excluded")
	}
}

func TestBulkAssignClaimAddsMissingUsersAndSkipsExistingRows(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/claims/"+corporateClaimID+"/bulk_assign", adminToken, map[string]any{
		"uids": []string{"u_no_claims", "u_corp_claim", "u_no_claims"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("bulk assign status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body bulkAssignClaimResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode bulk assign response: %v", err)
	}
	if body.AssignedCount != 1 || body.SkippedCount != 1 {
		t.Fatalf("bulk assign response = %+v, want assigned=1 skipped=1", body)
	}
	if got := countUserClaimRows(t, app, "u_no_claims", corporateClaimID); got != 1 {
		t.Fatalf("u_no_claims corporate rows = %d, want 1", got)
	}
	if got := countUserClaimRows(t, app, "u_corp_claim", corporateClaimID); got != 1 {
		t.Fatalf("u_corp_claim corporate rows = %d, want existing 1", got)
	}
}

func TestBulkAssignClaimRejectsUnknownUsers(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	adminToken := authTokenForEmail(t, app, "author@soup.com")

	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/claims/"+corporateClaimID+"/bulk_assign", adminToken, map[string]any{
		"uids": []string{"missing_user_id"},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown user status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if got := countUserClaimRows(t, app, "missing_user_id", corporateClaimID); got != 0 {
		t.Fatalf("missing user corporate rows = %d, want 0", got)
	}
}

func claimAssignableUsersContains(users []ClaimAssignableUser, uid string) bool {
	for _, user := range users {
		if user.ID == uid {
			return true
		}
	}
	return false
}

func countUserClaimRows(t *testing.T, app *tests.TestApp, uid string, claimID string) int {
	t.Helper()

	var result struct {
		Count int `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM user_claims
		WHERE uid = {:uid}
		  AND cid = {:claimID}
	`).Bind(dbx.Params{"uid": uid, "claimID": claimID}).One(&result); err != nil {
		t.Fatalf("failed counting user_claim rows: %v", err)
	}

	return result.Count
}

func authTokenForEmail(t *testing.T, app *tests.TestApp, email string) string {
	t.Helper()

	record, err := app.FindAuthRecordByEmail("users", email)
	if err != nil {
		t.Fatalf("failed to load auth user %s: %v", email, err)
	}

	token, err := record.NewAuthToken()
	if err != nil {
		t.Fatalf("failed to mint auth token for %s: %v", email, err)
	}

	return token
}

func performClaimsJSONRequest(t *testing.T, app *tests.TestApp, method string, path string, token string, body any) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create api router: %v", err)
	}

	var bodyReader *bytes.Reader
	if body == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bodyReader)
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		if err != nil {
			return err
		}
		mux.ServeHTTP(recorder, req)
		return nil
	}); err != nil {
		t.Fatalf("failed to serve request: %v", err)
	}

	return recorder
}
