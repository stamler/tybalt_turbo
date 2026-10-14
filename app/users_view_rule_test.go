package main

import (
	"net/http"
	"slices"
	"strings"
	"testing"
	"tybalt/internal/testseed"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/search"
)

// previousUsersViewRule is the users viewRule before 1791926703_user_view_grants.go.
// Its @collection references joined every expense to every purchase order, but
// it defines the access the current rule must keep.
const previousUsersViewRule = `@request.auth.id != "" &&
(
  @request.auth.id = id ||
  user_claims_via_uid.cid.name ?= 'po_approver' ||
  user_claims_via_uid.cid.name ?= 'tapr' ||
  @request.auth.user_claims_via_uid.cid.name ?= 'tapr' ||
  (
    @collection.expenses.approver ?= @request.auth.id &&
    @collection.expenses.uid ?= id &&
    @collection.expenses.committed ?!= ''
  ) ||
  (
    (
      @collection.purchase_orders.uid ?= id ||
      @collection.purchase_orders.approver ?= id ||
      @collection.purchase_orders.second_approver ?= id
    ) &&
    @collection.purchase_orders.status ?= "Active"
  )
)`

// usersViewRuleQuery builds the query PocketBase runs to check rule for the
// caller, as CanAccessRecord does, without its single-record filter, so it
// selects every user the caller can see.
func usersViewRuleQuery(t testing.TB, app core.App, caller *core.Record, rule string) *dbx.SelectQuery {
	t.Helper()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	query := app.RecordQuery(users).Select("users.id")
	resolver := core.NewRecordFieldResolver(app, users, &core.RequestInfo{Auth: caller}, true)
	expr, err := search.FilterData(rule).BuildExpr(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if err := resolver.UpdateQuery(query); err != nil {
		t.Fatal(err)
	}
	return query.AndWhere(expr)
}

func visibleUserIds(t testing.TB, app core.App, caller *core.Record, rule string) []string {
	t.Helper()
	var ids []string
	if err := usersViewRuleQuery(t, app, caller, rule).Column(&ids); err != nil {
		t.Fatal(err)
	}
	slices.Sort(ids)
	return ids
}

// usersVisibleByPreviousRule restates the previous rule with EXISTS
// subqueries. Running the previous rule itself for every seeded caller takes
// most of a minute because of its cross joins.
const usersVisibleByPreviousRule = `
SELECT u.id FROM users u
WHERE u.id = {:caller}
  OR EXISTS (
    SELECT 1 FROM user_claims uc JOIN claims c ON c.id = uc.cid
    WHERE uc.uid = u.id AND c.name IN ('po_approver', 'tapr')
  )
  OR EXISTS (
    SELECT 1 FROM user_claims uc JOIN claims c ON c.id = uc.cid
    WHERE uc.uid = {:caller} AND c.name = 'tapr'
  )
  OR EXISTS (
    SELECT 1 FROM expenses e
    WHERE e.uid = u.id AND e.approver = {:caller} AND e.committed != ''
  )
  OR EXISTS (
    SELECT 1 FROM purchase_orders po
    WHERE po.status = 'Active'
      AND (po.uid = u.id OR po.approver = u.id OR po.second_approver = u.id)
  )
ORDER BY u.id`

// Every seeded caller must see exactly the users the previous rule allowed.
func TestUsersViewRuleMatchesPreviousRuleForSeedData(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	callers, err := app.FindAllRecords("users")
	if err != nil {
		t.Fatal(err)
	}
	for _, caller := range callers {
		var want []string
		if err := app.DB().NewQuery(usersVisibleByPreviousRule).Bind(dbx.Params{"caller": caller.Id}).Column(&want); err != nil {
			t.Fatal(err)
		}
		slices.Sort(want)
		got := visibleUserIds(t, app, caller, *users.ViewRule)
		if !slices.Equal(got, want) {
			t.Errorf("%s: visible users differ\nprevious rule: %v\ncurrent rule:  %v", caller.Id, want, got)
		}
	}
}

// Each access path, and the near misses that must stay denied, using seeded
// fixtures. The uvg* users, expense uvgexpsubmit001, and purchase orders
// uvgactivepo0001 and uvgclosedpo0001 exist for these cases.
func TestUsersViewRuleAccessPaths(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		caller string // blank for a guest
		target string
		want   bool
	}{
		{"caller views self", "u_no_claims", "u_no_claims", true},
		{"user with no claims or grants is denied", "u_no_claims", "u_no_ppto_claim", false},
		{"user with only an unrelated claim is denied", "u_no_claims", "invoicejob00001", false},
		{"record has po_approver claim", "u_no_claims", "66ct66w380ob6w8", true},
		{"record has tapr claim", "u_no_claims", "u_self_apv_yes", true},
		{"caller has tapr claim", "u_self_apv_yes", "invoicejob00001", true},
		{"caller approved a committed expense of the record", "4r70mfovf22m9uh", "4ssj9f1yg250o9y", true},
		{"caller approved an uncommitted expense of the record", "uvgexpapprove01", "uvgexpowner0001", false},
		{"caller approved a committed expense of another user", "4r70mfovf22m9uh", "u_no_claims", false},
		// 4ssj9f1yg250o9y is also uid on Unapproved, Closed and Cancelled POs.
		{"another user approved the record's expense; record only on non-Active POs", "u_no_claims", "4ssj9f1yg250o9y", false},
		{"record is uid of an active purchase order", "u_no_claims", "rzr98oadsp9qc11", true},
		{"record is approver of an active purchase order", "u_no_claims", "uvgpoapprover01", true},
		{"record is second_approver of an active purchase order", "u_no_claims", "uvgposecond0001", true},
		{"record is second_approver of a closed purchase order", "u_no_claims", "uvgclosedsec001", false},
		{"guest is denied", "", "rzr98oadsp9qc11", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var caller *core.Record
			if tc.caller != "" {
				found, err := app.FindRecordById("users", tc.caller)
				if err != nil {
					t.Fatal(err)
				}
				caller = found
			}
			target, err := app.FindRecordById("users", tc.target)
			if err != nil {
				t.Fatal(err)
			}
			info := &core.RequestInfo{Auth: caller}
			for _, rule := range []struct {
				name string
				rule string
			}{{"current rule", *users.ViewRule}, {"previous rule", previousUsersViewRule}} {
				got, err := app.CanAccessRecord(target, info, &rule.rule)
				if err != nil {
					t.Fatal(err)
				}
				if got != tc.want {
					t.Errorf("%s: caller can view target = %v, want %v", rule.name, got, tc.want)
				}
			}
		})
	}
}

// The rule must join expense and purchase order data on the users row, not
// pair every expense with every purchase order.
func TestUsersViewRuleHasNoUnconditionedJoins(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	caller, err := app.FindAuthRecordByEmail("users", "u_no_claims@example.com")
	if err != nil {
		t.Fatal(err)
	}
	sql := usersViewRuleQuery(t, app, caller, *users.ViewRule).Build().SQL()

	if strings.Contains(sql, "__collection_") {
		t.Fatalf("rule must not use @collection joins:\n%s", sql)
	}
	grantsJoin := strings.Index(sql, "LEFT JOIN `user_view_grants`")
	if grantsJoin < 0 {
		t.Fatalf("rule must join user_view_grants:\n%s", sql)
	}
	// Joins run in rule order. The caller's claims multiply every later join,
	// so the grants join must come first. On production data this kept a
	// 50-user expand for a caller with 12 claims near 0.2s instead of 0.9s.
	if callerClaimsJoin := strings.Index(sql, "LEFT JOIN `user_claims` `__auth_users"); callerClaimsJoin < grantsJoin {
		t.Fatalf("the caller claims join must follow the grants join:\n%s", sql)
	}
	for _, join := range strings.Split(sql, "LEFT JOIN ")[1:] {
		table := strings.Fields(join)[0]
		if !strings.Contains(join, " ON ") {
			t.Errorf("join on %s has no condition:\n%s", table, sql)
		}
		if table == "`expenses`" || table == "`purchase_orders`" {
			t.Errorf("rule must not join %s directly:\n%s", table, sql)
		}
	}
}

func TestUsersViewRuleApi(t *testing.T) {
	token := func(email string) string {
		t.Helper()
		value, err := testutils.GenerateRecordToken("users", email)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	noClaimsToken := token("u_no_claims@example.com")
	approverToken := token("orphan@poapprover.com") // approved noclaims@example.com's committed expense
	taprToken := token("author@soup.com")

	view := func(name, id, authToken string, status int, content ...string) tests.ApiScenario {
		scenario := tests.ApiScenario{
			Name:            name,
			Method:          http.MethodGet,
			URL:             "/api/collections/users/records/" + id,
			ExpectedStatus:  status,
			ExpectedContent: content,
			TestAppFactory:  testutils.SetupTestApp,
		}
		if authToken != "" {
			scenario.Headers = map[string]string{"Authorization": authToken}
		}
		if status == http.StatusOK {
			scenario.ExpectedEvents = map[string]int{"OnRecordViewRequest": 1, "OnRecordEnrich": 1}
		} else {
			scenario.ExpectedContent = []string{`"data":{}`}
		}
		return scenario
	}

	scenarios := []tests.ApiScenario{
		view("caller views self", "u_no_claims", noClaimsToken, http.StatusOK, `"id":"u_no_claims"`),
		view("caller views po_approver holder", "66ct66w380ob6w8", noClaimsToken, http.StatusOK, `"id":"66ct66w380ob6w8"`),
		view("caller views tapr holder", "u_self_apv_yes", noClaimsToken, http.StatusOK, `"id":"u_self_apv_yes"`),
		view("tapr caller views unrelated user", "invoicejob00001", taprToken, http.StatusOK, `"id":"invoicejob00001"`),
		view("approver views user whose committed expense they approved", "4ssj9f1yg250o9y", approverToken, http.StatusOK, `"id":"4ssj9f1yg250o9y"`),
		view("caller views user on an active purchase order", "rzr98oadsp9qc11", noClaimsToken, http.StatusOK, `"id":"rzr98oadsp9qc11"`),
		view("unrelated caller cannot view user whose expense another approved", "4ssj9f1yg250o9y", noClaimsToken, http.StatusNotFound),
		view("unrelated caller cannot view unrelated user", "invoicejob00001", noClaimsToken, http.StatusNotFound),
		view("guest cannot view user on an active purchase order", "rzr98oadsp9qc11", "", http.StatusNotFound),
		{
			Name:           "expanding a purchase order uid applies the users rule",
			Method:         http.MethodGet,
			URL:            "/api/collections/purchase_orders/records/2plsetqdxht7esg?expand=uid",
			Headers:        map[string]string{"Authorization": noClaimsToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"expand":{"uid":{`,
				`"id":"rzr98oadsp9qc11"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			// uvgexpapprove01 may read the submitted expense but not its owner.
			Name:               "expanding an expense uid omits a user the caller cannot view",
			Method:             http.MethodGet,
			URL:                "/api/collections/expenses/records/uvgexpsubmit001?expand=uid",
			Headers:            map[string]string{"Authorization": token("users.view.expense.approver@example.com")},
			ExpectedStatus:     http.StatusOK,
			ExpectedContent:    []string{`"id":"uvgexpsubmit001"`, `"uid":"uvgexpowner0001"`, `"expand":{}`},
			NotExpectedContent: []string{`"expand":{"uid"`},
			TestAppFactory:     testutils.SetupTestApp,
		},
		{
			Name:           "the grants view is not listable through the API",
			Method:         http.MethodGet,
			URL:            "/api/collections/user_view_grants/records",
			Headers:        map[string]string{"Authorization": taprToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"data":{}`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			// An active_po row id is the user id, so it is easy to guess.
			Name:           "a grants view record is not readable through the API",
			Method:         http.MethodGet,
			URL:            "/api/collections/user_view_grants/records/rzr98oadsp9qc11",
			Headers:        map[string]string{"Authorization": taprToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"data":{}`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
