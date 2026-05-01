package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const (
	activeToggleInvalidRecordID = "ap_inactive"
	activeToggleRecordID        = "ap_corp_noclaim"
	activeToggleAdminRecordID   = "ap_admin_only"
)

func TestSetAdminProfileActive_AllowsAdminItAndHR(t *testing.T) {
	scenarios := []struct {
		name         string
		email        string
		recordID     string
		active       bool
		expectedCode int
	}{
		{
			name:         "admin can activate existing invalid inactive user",
			email:        "author@soup.com",
			recordID:     activeToggleInvalidRecordID,
			active:       true,
			expectedCode: http.StatusOK,
		},
		{
			name:         "it can deactivate active user",
			email:        "it.identity@example.com",
			recordID:     activeToggleRecordID,
			active:       false,
			expectedCode: http.StatusOK,
		},
		{
			name:         "hr can activate existing invalid inactive user",
			email:        "hr@example.com",
			recordID:     activeToggleInvalidRecordID,
			active:       true,
			expectedCode: http.StatusOK,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			t.Cleanup(app.Cleanup)
			hooks.AddHooks(app)
			AddRoutes(app)

			if scenario.recordID == activeToggleRecordID {
				// This test needs both directions of the state transition. The fixture
				// row is valid for admin_profiles saves, so we prepare only its active
				// state here instead of adding another near-duplicate fixture row.
				prepareAdminProfileActiveState(t, app, scenario.recordID, !scenario.active)
			}

			token := authTokenForEmail(t, app, scenario.email)
			rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/admin_profiles/"+scenario.recordID+"/active", token, map[string]any{
				"active": scenario.active,
			})
			if rec.Code != scenario.expectedCode {
				t.Fatalf("set active status = %d, want %d; body=%s", rec.Code, scenario.expectedCode, rec.Body.String())
			}

			adminProfile := loadAdminProfile(t, app, scenario.recordID)
			if got := adminProfile.GetBool("active"); got != scenario.active {
				t.Fatalf("active = %v, want %v", got, scenario.active)
			}

			var body adminProfileActiveResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to decode active response: %v", err)
			}
			if body.ID != scenario.recordID || body.Active != scenario.active {
				t.Fatalf("response = %+v, want id=%s active=%v", body, scenario.recordID, scenario.active)
			}
		})
	}
}

func TestSetAdminProfileActive_RejectsUnauthorizedCallers(t *testing.T) {
	scenarios := []struct {
		name  string
		email string
	}{
		{
			name:  "user without claims cannot set active",
			email: "u_no_claims@example.com",
		},
		{
			name:  "time off manager cannot set active",
			email: "u_with_claim@example.com",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			t.Cleanup(app.Cleanup)
			hooks.AddHooks(app)
			AddRoutes(app)

			prepareAdminProfileActiveState(t, app, activeToggleRecordID, false)

			token := authTokenForEmail(t, app, scenario.email)
			rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/admin_profiles/"+activeToggleRecordID+"/active", token, map[string]any{
				"active": true,
			})
			if rec.Code != http.StatusForbidden {
				t.Fatalf("set active status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
			}

			adminProfile := loadAdminProfile(t, app, activeToggleRecordID)
			if adminProfile.GetBool("active") {
				t.Fatal("unauthorized caller activated inactive admin profile")
			}
		})
	}
}

func TestSetAdminProfileActive_RejectsAdminTargets(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	token := authTokenForEmail(t, app, "author@soup.com")
	rec := performClaimsJSONRequest(t, app, http.MethodPost, "/api/admin_profiles/"+activeToggleAdminRecordID+"/active", token, map[string]any{
		"active": false,
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("set admin target active status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}

	adminProfile := loadAdminProfile(t, app, activeToggleAdminRecordID)
	if !adminProfile.GetBool("active") {
		t.Fatal("admin target was deactivated")
	}
}

func TestSetAdminProfileActive_RejectsBadRequests(t *testing.T) {
	scenarios := []struct {
		name           string
		recordID       string
		body           string
		expectedStatus int
	}{
		{
			name:           "missing active",
			recordID:       activeToggleRecordID,
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			recordID:       activeToggleRecordID,
			body:           `{`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unknown profile",
			recordID:       "missingprofile01",
			body:           `{"active":false}`,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			t.Cleanup(app.Cleanup)
			hooks.AddHooks(app)
			AddRoutes(app)

			token := authTokenForEmail(t, app, "author@soup.com")
			rec := performClaimsRawRequest(t, app, http.MethodPost, "/api/admin_profiles/"+scenario.recordID+"/active", token, scenario.body)
			if rec.Code != scenario.expectedStatus {
				t.Fatalf("set active status = %d, want %d; body=%s", rec.Code, scenario.expectedStatus, rec.Body.String())
			}
		})
	}
}

func TestAdminProfileActiveToggleTargets_ExcludesAdminTargets(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	token := authTokenForEmail(t, app, "hr@example.com")
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/admin_profiles/active_toggle_targets", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("active toggle targets status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body adminProfileActiveToggleTargetsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode active toggle targets response: %v", err)
	}
	if !stringSliceContains(body.AdminProfileIDs, activeToggleRecordID) {
		t.Fatalf("expected non-admin profile %s to be toggleable", activeToggleRecordID)
	}
	if stringSliceContains(body.AdminProfileIDs, activeToggleAdminRecordID) {
		t.Fatalf("expected admin profile %s to be excluded from toggleable targets", activeToggleAdminRecordID)
	}
}

func TestAdminProfileActiveToggleTargets_RejectsUnauthorizedCallers(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	token := authTokenForEmail(t, app, "u_no_claims@example.com")
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/admin_profiles/active_toggle_targets", token, nil)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("active toggle targets status = %d, want %d; body=%s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestAdminProfileIdentityListIncludesActiveState(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	token := authTokenForEmail(t, app, "it.identity@example.com")
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/admin_profiles/identity", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("identity list status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var rows []adminProfileIdentityListRow
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("failed to decode identity list: %v", err)
	}

	inactiveRow, ok := findAdminProfileIdentityRow(rows, activeToggleInvalidRecordID)
	if !ok {
		t.Fatalf("expected identity list to contain %s", activeToggleInvalidRecordID)
	}
	if inactiveRow.Active {
		t.Fatalf("expected %s active=false, got true", activeToggleInvalidRecordID)
	}

	activeRow, ok := findAdminProfileIdentityRow(rows, activeToggleRecordID)
	if !ok {
		t.Fatalf("expected identity list to contain %s", activeToggleRecordID)
	}
	if !activeRow.Active {
		t.Fatalf("expected %s active=true, got false", activeToggleRecordID)
	}
}

func prepareAdminProfileActiveState(t *testing.T, app *tests.TestApp, recordID string, active bool) {
	t.Helper()

	record := loadAdminProfile(t, app, recordID)
	record.Set("active", active)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to prepare admin profile %s active=%v: %v", recordID, active, err)
	}
}

func loadAdminProfile(t *testing.T, app *tests.TestApp, recordID string) *core.Record {
	t.Helper()

	record, err := app.FindRecordById("admin_profiles", recordID)
	if err != nil {
		t.Fatalf("failed to load admin profile %s: %v", recordID, err)
	}

	return record
}

func performClaimsRawRequest(t *testing.T, app *tests.TestApp, method string, path string, token string, body string) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create api router: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", "application/json")

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

func findAdminProfileIdentityRow(rows []adminProfileIdentityListRow, id string) (adminProfileIdentityListRow, bool) {
	for _, row := range rows {
		if row.ID == id {
			return row, true
		}
	}
	return adminProfileIdentityListRow{}, false
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
