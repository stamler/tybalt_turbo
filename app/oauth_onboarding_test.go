package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"tybalt/constants"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	pbauth "github.com/pocketbase/pocketbase/tools/auth"
	"golang.org/x/oauth2"
)

type microsoftOAuthMockProvider struct {
	pbauth.BaseProvider
	authUser *pbauth.AuthUser
}

func (p *microsoftOAuthMockProvider) FetchToken(_ string, _ ...oauth2.AuthCodeOption) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "mock-access-token"}, nil
}

func (p *microsoftOAuthMockProvider) FetchAuthUser(_ *oauth2.Token) (*pbauth.AuthUser, error) {
	return p.authUser, nil
}

// setupMicrosoftOAuthTestApp builds a normal seeded test app and then enables
// only the Microsoft provider on the users collection.
//
// These tests exercise the full PocketBase auth route instead of calling our
// helper functions directly, because the tricky parts of this bug sit at the
// boundary between PocketBase's OAuth flow and our hooks.
func setupMicrosoftOAuthTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app := testutils.SetupTestApp(t)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to load users collection: %v", err)
	}

	users.MFA.Enabled = false
	users.OAuth2.Enabled = true
	users.OAuth2.Providers = []core.OAuth2ProviderConfig{{
		Name:         pbauth.NameMicrosoft,
		ClientId:     "test-client",
		ClientSecret: "test-secret",
	}}
	if err := app.Save(users); err != nil {
		t.Fatalf("failed to update users OAuth config: %v", err)
	}

	return app
}

// installMicrosoftOAuthMock swaps the PocketBase Microsoft provider factory for
// a tiny in-memory test provider whose returned AuthUser we can change between
// requests.
//
// That lets each test describe the Microsoft payload it cares about while still
// going through the real `/auth-with-oauth2` route and the real hook chain.
func installMicrosoftOAuthMock(t *testing.T) func(*pbauth.AuthUser) {
	t.Helper()

	original, hadOriginal := pbauth.Providers[pbauth.NameMicrosoft]
	var currentUser *pbauth.AuthUser

	pbauth.Providers[pbauth.NameMicrosoft] = func() pbauth.Provider {
		return &microsoftOAuthMockProvider{authUser: currentUser}
	}

	t.Cleanup(func() {
		if hadOriginal {
			pbauth.Providers[pbauth.NameMicrosoft] = original
			return
		}
		delete(pbauth.Providers, pbauth.NameMicrosoft)
	})

	return func(user *pbauth.AuthUser) {
		currentUser = user
	}
}

// performMicrosoftAuthRequest sends a real auth-with-oauth2 request through the
// app router. The request body is fixed because the interesting variation is in
// the mocked Microsoft user payload, not in the client request shape.
func performMicrosoftAuthRequest(t *testing.T, app *tests.TestApp) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create api router: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/collections/users/auth-with-oauth2", strings.NewReader(`{
		"provider": "microsoft",
		"code": "test_code",
		"redirectURL": "https://example.com"
	}`))
	req.Header.Set("content-type", "application/json")

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		if err != nil {
			return err
		}
		mux.ServeHTTP(recorder, req)
		return nil
	}); err != nil {
		t.Fatalf("failed to serve auth request: %v", err)
	}

	return recorder
}

func performPasswordAuthRequest(t *testing.T, app *tests.TestApp, identity string, password string) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create api router: %v", err)
	}

	body, err := json.Marshal(map[string]string{
		"identity": identity,
		"password": password,
	})
	if err != nil {
		t.Fatalf("failed to marshal password auth request: %v", err)
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/collections/users/auth-with-password", strings.NewReader(string(body)))
	req.Header.Set("content-type", "application/json")

	serveEvent := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		mux, err := e.Router.BuildMux()
		if err != nil {
			return err
		}
		mux.ServeHTTP(recorder, req)
		return nil
	}); err != nil {
		t.Fatalf("failed to serve password auth request: %v", err)
	}

	return recorder
}

// findAdminProfileByUID and profileExists are thin helpers that keep the tests
// focused on the onboarding scenario rather than repetitive query boilerplate.
func findAdminProfileByUID(t *testing.T, app *tests.TestApp, uid string) *core.Record {
	t.Helper()

	record, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err != nil {
		t.Fatalf("failed to load admin profile for %s: %v", uid, err)
	}

	return record
}

func profileExists(app *tests.TestApp, uid string) (bool, error) {
	_, err := app.FindFirstRecordByFilter("profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

// Happy-path first login:
// - name comes from givenName + surname
// - username comes from the email local-part
// - verified remains true
// - admin_profiles is created
// - profiles is intentionally skipped because manager is required elsewhere
func TestMicrosoftFirstLoginPopulatesIdentityAndOnboards(t *testing.T) {
	app := setupMicrosoftOAuthTestApp(t)
	defer app.Cleanup()

	setMockUser := installMicrosoftOAuthMock(t)
	setMockUser(&pbauth.AuthUser{
		Id:    "provider-alice",
		Name:  "Alice Jones",
		Email: "alice.jones@tbte.ca",
		RawUser: map[string]any{
			"givenName":         "Alice",
			"surname":           "Jones",
			"mail":              "alice.jones@tbte.ca",
			"userPrincipalName": "alice.jones@tbte.ca",
		},
	})

	res := performMicrosoftAuthRequest(t, app)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}

	user, err := app.FindAuthRecordByEmail("users", "alice.jones@tbte.ca")
	if err != nil {
		t.Fatalf("failed to load created user: %v", err)
	}
	if got := user.GetString("name"); got != "Alice Jones" {
		t.Fatalf("users.name = %q, want %q", got, "Alice Jones")
	}
	if got := user.GetString("username"); got != "alice.jones" {
		t.Fatalf("users.username = %q, want %q", got, "alice.jones")
	}
	if !user.Verified() {
		t.Fatal("expected Microsoft email-backed user to be marked verified")
	}

	adminProfile := findAdminProfileByUID(t, app, user.Id)
	if got := adminProfile.GetString("payroll_id"); got == "999999" || !strings.HasPrefix(got, "9") {
		t.Fatalf("admin profile payroll_id = %q, want generated 9xxxxxxxx placeholder", got)
	}

	exists, err := profileExists(app, user.Id)
	if err != nil {
		t.Fatalf("failed checking profile existence: %v", err)
	}
	if exists {
		t.Fatal("expected no profile to be created during Microsoft onboarding because manager is required")
	}
}

// Two users can legitimately share the same email local-part on different
// domains. This verifies we suffix the second username instead of failing, and
// that each user gets a distinct generated payroll placeholder.
func TestMicrosoftFirstLoginUsesUsernameSuffixAndUniquePayrollID(t *testing.T) {
	app := setupMicrosoftOAuthTestApp(t)
	defer app.Cleanup()

	setMockUser := installMicrosoftOAuthMock(t)

	setMockUser(&pbauth.AuthUser{
		Id:    "provider-shared-1",
		Name:  "Shared One",
		Email: "shared@tbte.ca",
		RawUser: map[string]any{
			"givenName":         "Shared",
			"surname":           "One",
			"mail":              "shared@tbte.ca",
			"userPrincipalName": "shared@tbte.ca",
		},
	})
	first := performMicrosoftAuthRequest(t, app)
	if first.Code != http.StatusOK {
		t.Fatalf("first auth expected 200, got %d: %s", first.Code, first.Body.String())
	}

	setMockUser(&pbauth.AuthUser{
		Id:    "provider-shared-2",
		Name:  "Shared Two",
		Email: "shared@example.com",
		RawUser: map[string]any{
			"givenName":         "Shared",
			"surname":           "Two",
			"mail":              "shared@example.com",
			"userPrincipalName": "shared@example.com",
		},
	})
	second := performMicrosoftAuthRequest(t, app)
	if second.Code != http.StatusOK {
		t.Fatalf("second auth expected 200, got %d: %s", second.Code, second.Body.String())
	}

	firstUser, err := app.FindAuthRecordByEmail("users", "shared@tbte.ca")
	if err != nil {
		t.Fatalf("failed to load first user: %v", err)
	}
	secondUser, err := app.FindAuthRecordByEmail("users", "shared@example.com")
	if err != nil {
		t.Fatalf("failed to load second user: %v", err)
	}

	if got := firstUser.GetString("username"); got != "shared" {
		t.Fatalf("first username = %q, want %q", got, "shared")
	}
	if got := secondUser.GetString("username"); got != "shared-2" {
		t.Fatalf("second username = %q, want %q", got, "shared-2")
	}

	firstAdmin := findAdminProfileByUID(t, app, firstUser.Id)
	secondAdmin := findAdminProfileByUID(t, app, secondUser.Id)
	if firstAdmin.GetString("payroll_id") == secondAdmin.GetString("payroll_id") {
		t.Fatalf("expected unique payroll placeholders, got %q for both users", firstAdmin.GetString("payroll_id"))
	}
}

// Microsoft frequently omits `mail` while still providing
// `userPrincipalName`. In that case we still want all three identity fields to
// come out usable: username, email, and verified.
func TestMicrosoftFirstLoginUsesUPNForUsername(t *testing.T) {
	app := setupMicrosoftOAuthTestApp(t)
	defer app.Cleanup()

	setMockUser := installMicrosoftOAuthMock(t)
	setMockUser(&pbauth.AuthUser{
		Id:   "provider-upn",
		Name: "Upn User",
		RawUser: map[string]any{
			"givenName":         "Upn",
			"surname":           "User",
			"userPrincipalName": "upn.user@tbte.ca",
		},
	})

	res := performMicrosoftAuthRequest(t, app)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}

	user, err := app.FindFirstRecordByFilter("users", "username={:username}", dbx.Params{"username": "upn.user"})
	if err != nil {
		t.Fatalf("failed to load user by username: %v", err)
	}
	if got := user.GetString("username"); got != "upn.user" {
		t.Fatalf("users.username = %q, want %q", got, "upn.user")
	}
	if got := user.Email(); got != "upn.user@tbte.ca" {
		t.Fatalf("users.email = %q, want %q", got, "upn.user@tbte.ca")
	}
	if !user.Verified() {
		t.Fatal("expected UPN-backed Microsoft email to be marked verified")
	}
}

// DisplayName-only payloads are common enough that we should still create a
// usable auth record, but not enough to justify inventing a business profile
// that requires data Microsoft did not send.
func TestMicrosoftFirstLoginSkipsProfileWhenNamesAreMissing(t *testing.T) {
	app := setupMicrosoftOAuthTestApp(t)
	defer app.Cleanup()

	setMockUser := installMicrosoftOAuthMock(t)
	setMockUser(&pbauth.AuthUser{
		Id:    "provider-display-only",
		Name:  "Display Only",
		Email: "display.only@tbte.ca",
		RawUser: map[string]any{
			"displayName":       "Display Only",
			"mail":              "display.only@tbte.ca",
			"userPrincipalName": "display.only@tbte.ca",
		},
	})

	res := performMicrosoftAuthRequest(t, app)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}

	user, err := app.FindAuthRecordByEmail("users", "display.only@tbte.ca")
	if err != nil {
		t.Fatalf("failed to load created user: %v", err)
	}
	if got := user.GetString("name"); got != "Display Only" {
		t.Fatalf("users.name = %q, want %q", got, "Display Only")
	}
	if got := user.GetString("username"); got != "display.only" {
		t.Fatalf("users.username = %q, want %q", got, "display.only")
	}
	if !user.Verified() {
		t.Fatal("expected Microsoft display-name-only user to be marked verified")
	}

	findAdminProfileByUID(t, app, user.Id)

	exists, err := profileExists(app, user.Id)
	if err != nil {
		t.Fatalf("failed checking profile existence: %v", err)
	}
	if exists {
		t.Fatal("expected no profile to be created when Microsoft did not provide givenName/surname")
	}
}

// Existing inactive users should stay blocked even when Microsoft auth itself
// succeeds. This protects the business-level deactivation rule from being
// bypassed by OAuth login.
func TestMicrosoftExistingInactiveUserIsBlocked(t *testing.T) {
	app := setupMicrosoftOAuthTestApp(t)
	defer app.Cleanup()

	setMockUser := installMicrosoftOAuthMock(t)
	setMockUser(&pbauth.AuthUser{
		Id:    "provider-inactive",
		Name:  "Inactive User",
		Email: "inactive.user@tbte.ca",
		RawUser: map[string]any{
			"givenName":         "Inactive",
			"surname":           "User",
			"mail":              "inactive.user@tbte.ca",
			"userPrincipalName": "inactive.user@tbte.ca",
		},
	})

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to load users collection: %v", err)
	}
	user := core.NewRecord(users)
	user.SetEmail("inactive.user@tbte.ca")
	user.Set("username", "inactive.user")
	user.Set("name", "Inactive User")
	user.SetRandomPassword()
	user.SetVerified(true)
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to create inactive user: %v", err)
	}

	externalAuth := core.NewExternalAuth(app)
	externalAuth.SetCollectionRef(user.Collection().Id)
	externalAuth.SetRecordRef(user.Id)
	externalAuth.SetProvider(pbauth.NameMicrosoft)
	externalAuth.SetProviderId("provider-inactive")
	if err := app.Save(externalAuth); err != nil {
		t.Fatalf("failed to link inactive user external auth: %v", err)
	}

	adminProfiles, err := app.FindCollectionByNameOrId("admin_profiles")
	if err != nil {
		t.Fatalf("failed to load admin_profiles collection: %v", err)
	}
	adminProfile := core.NewRecord(adminProfiles)
	adminProfile.Set("uid", user.Id)
	adminProfile.Set("active", false)
	adminProfile.Set("work_week_hours", constants.DEFAULT_WORK_WEEK_HOURS)
	adminProfile.Set("default_charge_out_rate", constants.DEFAULT_CHARGE_OUT_RATE)
	adminProfile.Set("skip_min_time_check", "no")
	adminProfile.Set("salary", false)
	adminProfile.Set("untracked_time_off", false)
	adminProfile.Set("time_sheet_expected", false)
	adminProfile.Set("default_branch", constants.DEFAULT_BRANCH_ID)
	adminProfile.Set("payroll_id", "912345678")
	if err := app.Save(adminProfile); err != nil {
		t.Fatalf("failed to create inactive admin profile: %v", err)
	}

	res := performMicrosoftAuthRequest(t, app)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "inactive") {
		t.Fatalf("expected inactive account message, got %s", res.Body.String())
	}
}

// Password auth with no admin_profiles should fail fast instead of silently
// recreating the row. Only OAuth2 logins self-heal this onboarding gap.
func TestPasswordLoginFailsWhenAdminProfileIsMissing(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to load users collection: %v", err)
	}
	users.MFA.Enabled = false
	users.PasswordAuth.Enabled = true
	users.PasswordAuth.IdentityFields = []string{"email"}
	if err := app.Save(users); err != nil {
		t.Fatalf("failed to configure password auth test collection: %v", err)
	}

	user := core.NewRecord(users)
	user.SetEmail("password.only@tbte.ca")
	user.Set("username", "password.only")
	user.Set("name", "Password Only")
	user.SetPassword("test-password-123")
	user.SetVerified(true)
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to create password auth user: %v", err)
	}

	res := performPasswordAuthRequest(t, app, "password.only@tbte.ca", "test-password-123")
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(strings.ToLower(res.Body.String()), "account setup incomplete") {
		t.Fatalf("expected account setup error, got %s", res.Body.String())
	}
}
