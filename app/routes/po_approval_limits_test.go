package routes

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPOApprovalLimitsAccess(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	for _, tc := range []struct {
		name, email string
		status      int
	}{
		{"anonymous", "", http.StatusUnauthorized},
		{"no claims", "u_no_claims@example.com", http.StatusForbidden},
		{"approver only", "tier2@poapprover.com", http.StatusOK},
		{"report and approver", "fatt@mac.com", http.StatusOK},
		{"admin only", "admin.only@example.com", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := ""
			if tc.email != "" {
				token = authTokenForEmail(t, app, tc.email)
			}
			rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/purchase_orders/approval_limits", token, nil)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tc.status, rec.Body.String())
			}
		})
	}

}

func TestPOApprovalLimitsClaimRevocation(t *testing.T) {
	for _, tc := range []struct {
		name, email string
		claimIDs    []string
	}{
		{"report removed first", "fatt@mac.com", []string{"p5hg1ck0cbjrp0z", "6dqxhrtmxin2jz5"}},
		{"approver removed first", "fatt@mac.com", []string{"6dqxhrtmxin2jz5", "p5hg1ck0cbjrp0z"}},
		{"approver only", "tier2@poapprover.com", []string{"9azfu0gh25n6mjm"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := testseed.NewSeededTestApp(t)
			t.Cleanup(app.Cleanup)
			hooks.AddHooks(app)
			AddRoutes(app)
			token := authTokenForEmail(t, app, tc.email)
			readPOApprovalLimits(t, app, token)
			// Remove claims only in this isolated fixture DB to test an existing
			// token after each revocation. Either remaining claim must allow access.
			for i, id := range tc.claimIDs {
				if _, err := app.DB().NewQuery(`DELETE FROM user_claims WHERE id = {:id}`).Bind(dbx.Params{"id": id}).Execute(); err != nil {
					t.Fatal(err)
				}
				want := http.StatusOK
				if i == len(tc.claimIDs)-1 {
					want = http.StatusForbidden
				}
				rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/purchase_orders/approval_limits", token, nil)
				if rec.Code != want {
					t.Fatalf("after revoking %s: status = %d, want %d; body=%s", id, rec.Code, want, rec.Body.String())
				}
			}
		})
	}
}

func TestPOApprovalLimitsDataAndReadOnlyAccess(t *testing.T) {
	for _, email := range []string{"fatt@mac.com", "tier2@poapprover.com"} {
		t.Run(email, func(t *testing.T) {
			testPOApprovalLimitsDataAndReadOnlyAccess(t, email)
		})
	}
}

func testPOApprovalLimitsDataAndReadOnlyAccess(t *testing.T, email string) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)
	token := authTokenForEmail(t, app, email)

	// Give one existing fixture distinct amounts so swapped category columns
	// cannot pass the test. This change exists only in this isolated test DB.
	props, err := app.FindRecordById("po_approver_props", "papfxtr2a000001")
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range map[string]float64{
		"max_amount": 101, "project_max": 202, "sponsorship_max": 303,
		"staff_and_social_max": 404, "media_and_event_max": 505, "computer_max": 606,
	} {
		props.Set(field, value)
	}
	if err := app.Save(props); err != nil {
		t.Fatal(err)
	}
	// Reload after saving so the baseline uses the stored field types.
	props, err = app.FindRecordById("po_approver_props", props.Id)
	if err != nil {
		t.Fatal(err)
	}
	before := props.PublicExport()

	body := readPOApprovalLimits(t, app, token)
	rows := map[string]poApprovalLimitsRow{}
	for _, row := range body.Items {
		rows[row.ID] = row
		if row.Divisions == nil {
			t.Fatalf("nil divisions for %s", row.ID)
		}
	}
	if _, exists := rows["ucinactpo000001"]; exists {
		t.Fatal("inactive approver was included")
	}
	if _, exists := rows["auncwoq1bbgvr9s"]; exists {
		t.Fatal("non-PO claim was included")
	}
	full := rows["9azfu0gh25n6mjm"]
	if !full.Configured || full.GivenName != "Tier" || full.Surname != "Two" || len(full.Divisions) != 0 {
		t.Fatalf("unexpected unrestricted approver: %+v", full)
	}
	if full.MaxAmount != 101 || full.ProjectMax != 202 || full.SponsorshipMax != 303 || full.StaffAndSocialMax != 404 || full.MediaAndEventMax != 505 || full.ComputerMax != 606 {
		t.Fatalf("wrong category amounts: %+v", full)
	}
	restricted := rows["ziusrr1491mn7ug"]
	if !reflect.DeepEqual(restricted.Divisions, []string{"hcd86z57zjty6jo", "fy4i9poneukvq9u"}) || restricted.ComputerMax != 0 {
		t.Fatalf("wrong divisions or zero limit: %+v", restricted)
	}
	wantFields := map[string]string{
		"Capital": "max_amount", "Project": "project_max", "Sponsorship": "sponsorship_max",
		"Staff and Social": "staff_and_social_max", "Media/Advertising": "media_and_event_max", "Computers/Software": "computer_max",
	}
	if len(body.Categories) != len(wantFields) {
		t.Fatalf("categories = %+v", body.Categories)
	}
	for _, c := range body.Categories {
		if wantFields[c.Label] != c.LimitField {
			t.Fatalf("wrong category mapping: %+v", c)
		}
		if c.Label == "Capital" && c.SecondApprovalThreshold != 500 {
			t.Fatalf("wrong second approval threshold: %+v", c)
		}
	}
	if len(body.Divisions) == 0 || body.Divisions[0].Name == "" {
		t.Fatal("division labels are missing")
	}

	// Reading the report must not grant access to the existing write API.
	write := performClaimsJSONRequest(t, app, http.MethodPatch, "/api/collections/po_approver_props/records/"+props.Id, token, map[string]any{"max_amount": 999999})
	if write.Code < 400 {
		t.Fatalf("report reader could edit limits: %d %s", write.Code, write.Body.String())
	}
	for _, method := range []string{http.MethodPost, http.MethodPatch, http.MethodDelete} {
		rec := performClaimsJSONRequest(t, app, method, "/api/purchase_orders/approval_limits", token, map[string]any{"max_amount": 999999})
		if rec.Code < 400 {
			t.Fatalf("report endpoint accepted %s: %d", method, rec.Code)
		}
	}
	after, err := app.FindRecordById("po_approver_props", props.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after.PublicExport()) {
		t.Fatal("read or denied write changed the approval settings")
	}

	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/purchase_orders/approval_limits", token, nil)
	for _, privateField := range []string{"payroll_id", "salary", "opening_op", "email", "user_claim", "division_json"} {
		if strings.Contains(rec.Body.String(), `"`+privateField+`"`) {
			t.Fatalf("response contains unrelated field %s", privateField)
		}
	}
}

func TestPOApprovalLimitsMissingSettingsAndEmptyList(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)
	token := authTokenForEmail(t, app, "fatt@mac.com")

	// Remove an existing fixture's settings to exercise an incomplete claim.
	// Do not add test records or change the shared CSV fixtures.
	if _, err := app.DB().NewQuery(`DELETE FROM po_approver_props WHERE id = 'papfxtr2a000001'`).Execute(); err != nil {
		t.Fatal(err)
	}
	body := readPOApprovalLimits(t, app, token)
	found := false
	for _, row := range body.Items {
		if row.ID == "9azfu0gh25n6mjm" {
			found = true
			if row.Configured || row.MaxAmount != 0 || len(row.Divisions) != 0 {
				t.Fatalf("missing settings misrepresented: %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("claim with missing settings was hidden")
	}
	// Exercise an empty organization without adding a separate seed package.
	if _, err := app.DB().NewQuery(`UPDATE admin_profiles SET active = 0`).Execute(); err != nil {
		t.Fatal(err)
	}
	body = readPOApprovalLimits(t, app, token)
	if body.Items == nil || len(body.Items) != 0 {
		t.Fatalf("expected an empty array, got %+v", body.Items)
	}
}

func TestPOApprovalLimitsInvalidDivisionsFailClosed(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	AddRoutes(app)
	token := authTokenForEmail(t, app, "fatt@mac.com")

	// Simulate damaged stored data in this isolated fixture DB. Do not present
	// an unreadable division restriction as unrestricted approval authority.
	if _, err := app.DB().NewQuery(`UPDATE po_approver_props SET divisions = 'invalid' WHERE id = 'papfxtr2a000001'`).Execute(); err != nil {
		t.Fatal(err)
	}
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/purchase_orders/approval_limits", token, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

func readPOApprovalLimits(t *testing.T, app *tests.TestApp, token string) poApprovalLimitsResponse {
	t.Helper()
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/purchase_orders/approval_limits", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	var body poApprovalLimitsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body
}
