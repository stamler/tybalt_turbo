package routes

import (
	"encoding/json"
	"net/http"
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

func TestNavBadgesRequiresAuth(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/nav/badges", "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("nav badges status = %d, want %d; body=%s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestNavBadgesReturnsUserScopedCounts(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	token := authTokenForEmail(t, app, "author@soup.com")
	rec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/nav/badges", token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("nav badges status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}

	counts := decodeNavBadgeCounts(t, rec.Body.Bytes())
	userID := "f2j5a8vk006baub"

	assertNavBadgeCount(t, counts, navTimeSheetsPendingHref, expectedPendingTimeSheetCount(t, app, userID))
	assertNavBadgeCount(t, counts, navExpensesPendingHref, expectedPendingExpenseCount(t, app, userID))
	assertNavBadgeCount(t, counts, navPurchaseOrdersPendingHref, expectedPendingPurchaseOrderCount(t, app, userID))
	assertNavBadgeCount(t, counts, navProjectAuthorizationHref, expectedProjectAuthorizationQueueCount(t, app))

	if _, ok := counts[navExpenseCommitQueueHref]; ok {
		t.Fatalf("did not expect %s for user without commit claim; counts=%v", navExpenseCommitQueueHref, counts)
	}
	if _, ok := counts[navExpenseSettlementHref]; ok {
		t.Fatalf("did not expect %s for user without payables_admin claim; counts=%v", navExpenseSettlementHref, counts)
	}
}

func TestNavBadgesReturnsAuthorizedQueueCounts(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	commitToken := authTokenForEmail(t, app, "fakemanager@fakesite.xyz")
	commitRec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/nav/badges", commitToken, nil)
	if commitRec.Code != http.StatusOK {
		t.Fatalf("commit nav badges status = %d, want %d; body=%s", commitRec.Code, http.StatusOK, commitRec.Body.String())
	}
	commitCounts := decodeNavBadgeCounts(t, commitRec.Body.Bytes())
	assertNavBadgeCount(t, commitCounts, navExpenseCommitQueueHref, expectedExpenseCommitQueueCount(t, app))

	payablesAdminToken := authTokenForEmail(t, app, "book@keeper.com")
	payablesRec := performClaimsJSONRequest(t, app, http.MethodGet, "/api/nav/badges", payablesAdminToken, nil)
	if payablesRec.Code != http.StatusOK {
		t.Fatalf("payables nav badges status = %d, want %d; body=%s", payablesRec.Code, http.StatusOK, payablesRec.Body.String())
	}
	payablesCounts := decodeNavBadgeCounts(t, payablesRec.Body.Bytes())
	settlementCount := expectedExpenseSettlementCount(t, app)
	if settlementCount <= 0 {
		t.Fatal("fixture should include at least one unsettled foreign-currency settlement row")
	}
	assertNavBadgeCount(t, payablesCounts, navExpenseSettlementHref, settlementCount)

	if _, ok := payablesCounts[navExpenseCommitQueueHref]; ok {
		t.Fatalf("did not expect %s for payables user without commit claim; counts=%v", navExpenseCommitQueueHref, payablesCounts)
	}
	if _, ok := payablesCounts[navProjectAuthorizationHref]; ok {
		t.Fatalf("did not expect %s for payables user without accounting claim; counts=%v", navProjectAuthorizationHref, payablesCounts)
	}
}

func decodeNavBadgeCounts(t *testing.T, body []byte) map[string]int {
	t.Helper()

	var counts map[string]int
	if err := json.Unmarshal(body, &counts); err != nil {
		t.Fatalf("failed to decode nav badge counts: %v", err)
	}
	return counts
}

func assertNavBadgeCount(t *testing.T, counts map[string]int, href string, want int) {
	t.Helper()

	got, ok := counts[href]
	if !ok {
		t.Fatalf("missing nav badge %s in counts=%v", href, counts)
	}
	if got != want {
		t.Fatalf("nav badge %s = %d, want %d; counts=%v", href, got, want, counts)
	}
}

func expectedCount(t *testing.T, app *tests.TestApp, query string, params dbx.Params) int {
	t.Helper()

	var count int
	if err := app.DB().NewQuery(query).Bind(params).Row(&count); err != nil {
		t.Fatalf("failed counting expected rows: %v", err)
	}
	return count
}

func expectedPendingTimeSheetCount(t *testing.T, app *tests.TestApp, userID string) int {
	t.Helper()

	return expectedCount(t, app, `
		SELECT COUNT(*)
		FROM (
			SELECT te.tsid
			FROM time_entries te
			INNER JOIN time_sheets ts ON te.tsid = ts.id
			WHERE ts.approver = {:uid}
			  AND ts.approved = ''
			GROUP BY te.tsid
		)
	`, dbx.Params{"uid": userID})
}

func expectedPendingExpenseCount(t *testing.T, app *tests.TestApp, userID string) int {
	t.Helper()

	return expectedCount(t, app, buildCountQuery(wherePending), dbx.Params{"auth": userID})
}

func expectedPendingPurchaseOrderCount(t *testing.T, app *tests.TestApp, userID string) int {
	t.Helper()

	query := `
		WITH visibility_base AS (
		` + poVisibilityBaseQuery + `
		)
		SELECT COUNT(*)
		FROM visibility_base
		WHERE is_unapproved_actionable_now = 1
	`
	return expectedCount(t, app, query, purchaseOrderVisibilityParams(app, userID, "all", "", "", 0))
}

func expectedProjectAuthorizationQueueCount(t *testing.T, app *tests.TestApp) int {
	t.Helper()

	return expectedCount(t, app, `
		SELECT COUNT(*)
		FROM jobs j
		WHERE j.status = 'Active'
		  AND j.number NOT LIKE 'P%'
		  AND j.project_authorization_doc != ''
		  AND j.project_authorization_doc_hash != ''
		  AND j.pa_reviewed = ''
		  AND j.pa_reviewer = ''
	`, dbx.Params{})
}

func expectedExpenseCommitQueueCount(t *testing.T, app *tests.TestApp) int {
	t.Helper()

	return expectedCount(t, app, `
		SELECT COUNT(*)
		FROM expenses e
		LEFT JOIN currencies cur ON cur.id = e.currency
		WHERE e.submitted = 1
		  AND e.committed = ''
		  AND e.approved != ''
		  AND NOT (
		    COALESCE(cur.code, 'CAD') != 'CAD'
		    AND e.payment_type IN ('OnAccount', 'CorporateCreditCard')
		    AND COALESCE(e.settled, '') = ''
		  )
	`, dbx.Params{})
}

func expectedExpenseSettlementCount(t *testing.T, app *tests.TestApp) int {
	t.Helper()

	return expectedCount(t, app, `
		SELECT COUNT(*)
		FROM expenses e
		LEFT JOIN currencies cur ON cur.id = e.currency
		WHERE e.payment_type IN ('OnAccount', 'CorporateCreditCard')
		  AND COALESCE(cur.code, 'CAD') != 'CAD'
		  AND COALESCE(e.approved, '') != ''
		  AND COALESCE(e.rejected, '') = ''
		  AND COALESCE(e.committed, '') = ''
		  AND COALESCE(e.settled, '') = ''
	`, dbx.Params{})
}
