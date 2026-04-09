package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
)

const (
	testCADCurrencyID                = "cadcurr00000001"
	testUSDCurrencyID                = "usdcurr00000001"
	testForeignNoPOSettleExpenseID   = "fxnoposettle001"
	testForeignNoPOCommitExpenseID   = "fxnopocommit001"
	testForeignNoPOSettleOKExpenseID = "fxnoposettleok01"
	testForeignNoPOCommitOKExpenseID = "fxnopocommitok01"
)

func TestCurrenciesRoutes_ListPermissionsAndBackfill(t *testing.T) {
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	listRes := performTestAPIRequest(t, app, "GET", "/api/currencies", nil, map[string]string{
		"Authorization": regularUserToken,
	})
	mustStatus(t, listRes, 200)

	var rows []map[string]any
	if err := json.Unmarshal(listRes.Body.Bytes(), &rows); err != nil {
		t.Fatalf("failed decoding currency list: %v", err)
	}
	if len(rows) < 2 {
		t.Fatalf("expected at least CAD and USD, got %d rows", len(rows))
	}
	if got := rows[0]["code"]; got != "CAD" {
		t.Fatalf("expected CAD first by ui_sort, got %v", got)
	}
	if got := rows[1]["code"]; got != "USD" {
		t.Fatalf("expected USD second by ui_sort, got %v", got)
	}

	nonAdminInitStatus := performTestAPIRequest(t, app, "GET", "/api/currencies/init_status", nil, map[string]string{
		"Authorization": regularUserToken,
	})
	mustStatus(t, nonAdminInitStatus, 403)

	adminInitStatus := performTestAPIRequest(t, app, "GET", "/api/currencies/init_status", nil, map[string]string{
		"Authorization": adminToken,
	})
	mustStatus(t, adminInitStatus, 200)

	var status struct {
		HomeCurrencyID      string `json:"home_currency_id"`
		HomeCurrencyExists  bool   `json:"home_currency_exists"`
		HomeCurrencyReady   bool   `json:"home_currency_ready"`
		BlankPurchaseOrders int    `json:"blank_purchase_orders"`
		BlankExpenses       int    `json:"blank_expenses"`
	}
	if err := json.Unmarshal(adminInitStatus.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed decoding init status: %v", err)
	}
	if !status.HomeCurrencyExists || !status.HomeCurrencyReady {
		t.Fatalf("expected CAD home currency to exist and be ready, got %+v", status)
	}
	if status.HomeCurrencyID != testCADCurrencyID {
		t.Fatalf("expected home currency id %s, got %s", testCADCurrencyID, status.HomeCurrencyID)
	}

	badBackfill := performTestAPIRequest(t, app, "POST", "/api/currencies/"+testUSDCurrencyID+"/initialize_backfill", strings.NewReader("{}"), map[string]string{
		"Authorization": adminToken,
	})
	mustStatus(t, badBackfill, 400)

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE purchase_orders
		SET approval_total = 100, approval_total_home = 0, currency = ''
		WHERE id = 'standardupd001'
	`).Execute(); err != nil {
		t.Fatalf("failed preparing blank PO fixture: %v", err)
	}
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE purchase_orders
		SET approval_total = 100, approval_total_home = 135, currency = {:currencyId}
		WHERE id = 'gal6e5la2fa4rpn'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing foreign PO fixture: %v", err)
	}
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = '', settled_total = 0
		WHERE id = '2gq9uyxmkcyopa4'
	`).Execute(); err != nil {
		t.Fatalf("failed preparing blank expense fixture: %v", err)
	}
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, settled_total = 91.11
		WHERE id = 'b4o6xph4ngwx4nw'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing foreign expense fixture: %v", err)
	}

	backfillRes := performTestAPIRequest(t, app, "POST", "/api/currencies/"+testCADCurrencyID+"/initialize_backfill", strings.NewReader("{}"), map[string]string{
		"Authorization": adminToken,
	})
	mustStatus(t, backfillRes, 200)

	var postStatus struct {
		BlankPurchaseOrders int `json:"blank_purchase_orders"`
		BlankExpenses       int `json:"blank_expenses"`
	}
	if err := json.Unmarshal(backfillRes.Body.Bytes(), &postStatus); err != nil {
		t.Fatalf("failed decoding backfill status: %v", err)
	}
	if postStatus.BlankPurchaseOrders != 0 || postStatus.BlankExpenses != 0 {
		t.Fatalf("expected all blank currencies backfilled, got %+v", postStatus)
	}

	backfilledPO, err := app.FindRecordById("purchase_orders", "standardupd001")
	if err != nil {
		t.Fatalf("failed loading backfilled PO: %v", err)
	}
	if got := backfilledPO.GetString("currency"); got != testCADCurrencyID {
		t.Fatalf("expected blank PO currency backfilled to CAD, got %q", got)
	}
	if got := backfilledPO.GetFloat("approval_total_home"); got != backfilledPO.GetFloat("approval_total") {
		t.Fatalf("expected blank PO approval_total_home to match approval_total, got %v vs %v", got, backfilledPO.GetFloat("approval_total"))
	}

	foreignPO, err := app.FindRecordById("purchase_orders", "gal6e5la2fa4rpn")
	if err != nil {
		t.Fatalf("failed loading preserved foreign PO: %v", err)
	}
	if got := foreignPO.GetString("currency"); got != testUSDCurrencyID {
		t.Fatalf("expected foreign PO currency preserved, got %q", got)
	}
	if got := foreignPO.GetFloat("approval_total_home"); got != 135 {
		t.Fatalf("expected foreign PO approval_total_home preserved at 135, got %v", got)
	}

	backfilledExpense, err := app.FindRecordById("expenses", "2gq9uyxmkcyopa4")
	if err != nil {
		t.Fatalf("failed loading backfilled expense: %v", err)
	}
	if got := backfilledExpense.GetString("currency"); got != testCADCurrencyID {
		t.Fatalf("expected blank expense currency backfilled to CAD, got %q", got)
	}
	if got := backfilledExpense.GetFloat("settled_total"); got != backfilledExpense.GetFloat("total") {
		t.Fatalf("expected blank expense settled_total to match total, got %v vs %v", got, backfilledExpense.GetFloat("total"))
	}

	foreignExpense, err := app.FindRecordById("expenses", "b4o6xph4ngwx4nw")
	if err != nil {
		t.Fatalf("failed loading preserved foreign expense: %v", err)
	}
	if got := foreignExpense.GetString("currency"); got != testUSDCurrencyID {
		t.Fatalf("expected foreign expense currency preserved, got %q", got)
	}
	if got := foreignExpense.GetFloat("settled_total"); got != 91.11 {
		t.Fatalf("expected foreign expense settled_total preserved at 91.11, got %v", got)
	}

	idempotentRes := performTestAPIRequest(t, app, "POST", "/api/currencies/"+testCADCurrencyID+"/initialize_backfill", strings.NewReader("{}"), map[string]string{
		"Authorization": adminToken,
	})
	mustStatus(t, idempotentRes, 200)

	foreignPO, err = app.FindRecordById("purchase_orders", "gal6e5la2fa4rpn")
	if err != nil {
		t.Fatalf("failed reloading preserved foreign PO: %v", err)
	}
	if got := foreignPO.GetFloat("approval_total_home"); got != 135 {
		t.Fatalf("expected idempotent backfill to preserve foreign approval_total_home, got %v", got)
	}

	deleteInUseRes := performTestAPIRequest(t, app, "DELETE", "/api/currencies/"+testCADCurrencyID, nil, map[string]string{
		"Authorization": adminToken,
	})
	mustStatus(t, deleteInUseRes, 400)
}

func TestExpenseSettlementRoutes_ListSettleAndClear(t *testing.T) {
	payablesAdminToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}
	regularUserToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	usdCurrency, err := app.FindRecordById("currencies", testUSDCurrencyID)
	if err != nil {
		t.Fatalf("failed loading USD currency fixture: %v", err)
	}
	usdRate := usdCurrency.GetFloat("rate")

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, settled_total = 0, settler = '', settled = ''
		WHERE id = 'b4o6xph4ngwx4nw'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing unsettled expense: %v", err)
	}
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, settled_total = 600.12, settler = 'tqqf7q0f3378rvp', settled = '2026-04-03 12:00:00.000Z'
		WHERE id = 'eqhozipupteogp8'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing settled expense: %v", err)
	}
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}
		WHERE id = '77i1224mudailrb'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing foreign expense control row: %v", err)
	}

	unauthorizedList := performTestAPIRequest(t, app, "GET", "/api/expenses/unsettled", nil, map[string]string{
		"Authorization": regularUserToken,
	})
	mustStatus(t, unauthorizedList, 403)

	unsettledRes := performTestAPIRequest(t, app, "GET", "/api/expenses/unsettled", nil, map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, unsettledRes, 200)
	var unsettledRows []struct {
		ID                 string  `json:"id"`
		CreatorName        string  `json:"creator_name"`
		IndicativeCADTotal float64 `json:"indicative_cad_total"`
	}
	if err := json.Unmarshal(unsettledRes.Body.Bytes(), &unsettledRows); err != nil {
		t.Fatalf("failed decoding unsettled settlement queue response: %v", err)
	}
	body := mustReadBody(t, unsettledRes)
	if !strings.Contains(body, `"id":"b4o6xph4ngwx4nw"`) {
		t.Fatalf("expected unsettled list to include unsettled foreign corp card expense, body=%s", body)
	}
	if strings.Contains(body, `"id":"eqhozipupteogp8"`) || strings.Contains(body, `"id":"77i1224mudailrb"`) {
		t.Fatalf("expected unsettled list to exclude settled or Expense payment type rows, body=%s", body)
	}
	foundUnsettledRow := false
	for _, row := range unsettledRows {
		if row.ID != "b4o6xph4ngwx4nw" {
			continue
		}
		foundUnsettledRow = true
		if row.CreatorName == "" {
			t.Fatal("expected unsettled settlement row to include creator_name")
		}
		if row.IndicativeCADTotal <= 0 {
			t.Fatalf("expected unsettled settlement row to include indicative_cad_total, got %v", row.IndicativeCADTotal)
		}
	}
	if !foundUnsettledRow {
		t.Fatal("expected to decode unsettled settlement row for b4o6xph4ngwx4nw")
	}

	settledRes := performTestAPIRequest(t, app, "GET", "/api/expenses/settled", nil, map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, settledRes, 200)
	body = mustReadBody(t, settledRes)
	if !strings.Contains(body, `"id":"eqhozipupteogp8"`) {
		t.Fatalf("expected settled list to include settled foreign on-account expense, body=%s", body)
	}
	if strings.Contains(body, `"id":"b4o6xph4ngwx4nw"`) {
		t.Fatalf("expected settled list to exclude unsettled row, body=%s", body)
	}

	unsettledExpenseRecord, err := app.FindRecordById("expenses", "b4o6xph4ngwx4nw")
	if err != nil {
		t.Fatalf("failed loading unsettled expense fixture: %v", err)
	}
	acceptableSettledTotal := utilities.IndicativeHomeAmount(
		unsettledExpenseRecord.GetFloat("total"),
		utilities.CurrencyInfo{Rate: usdRate},
	)

	unauthorizedSettle := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/settle", strings.NewReader(`{"settled_total":95.55}`), map[string]string{
		"Authorization": regularUserToken,
	})
	mustStatus(t, unauthorizedSettle, 403)

	outOfRangeSettle := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/settle", strings.NewReader(fmt.Sprintf(`{"settled_total":%.2f}`, utilities.RoundCurrencyAmount(acceptableSettledTotal*1.25))), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, outOfRangeSettle, 400)

	settleRes := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/settle", strings.NewReader(fmt.Sprintf(`{"settled_total":%.2f}`, acceptableSettledTotal)), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, settleRes, 200)

	settledExpense, err := app.FindRecordById("expenses", "b4o6xph4ngwx4nw")
	if err != nil {
		t.Fatalf("failed loading newly settled expense: %v", err)
	}
	if got := settledExpense.GetFloat("settled_total"); got != acceptableSettledTotal {
		t.Fatalf("expected settled_total %v, got %v", acceptableSettledTotal, got)
	}
	if got := settledExpense.GetString("settler"); got != "tqqf7q0f3378rvp" {
		t.Fatalf("expected settler to be bookkeeper, got %q", got)
	}
	if settledExpense.GetDateTime("settled").IsZero() {
		t.Fatal("expected settled timestamp to be populated")
	}

	doubleSettleRes := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/settle", strings.NewReader(`{"settled_total":111.11}`), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, doubleSettleRes, 400)

	clearRes := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/clear_settlement", strings.NewReader(`{}`), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, clearRes, 200)

	clearedExpense, err := app.FindRecordById("expenses", "b4o6xph4ngwx4nw")
	if err != nil {
		t.Fatalf("failed loading cleared expense: %v", err)
	}
	if got := clearedExpense.GetFloat("settled_total"); got != 0 {
		t.Fatalf("expected cleared settled_total to be 0, got %v", got)
	}
	if got := clearedExpense.GetString("settler"); got != "" {
		t.Fatalf("expected cleared settler to be blank, got %q", got)
	}
	if !clearedExpense.GetDateTime("settled").IsZero() {
		t.Fatal("expected cleared settled timestamp to be blank")
	}

	doubleClearRes := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/clear_settlement", strings.NewReader(`{}`), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, doubleClearRes, 400)
}

func TestForeignExpenseRoutes_SubmitCommitAndRejectRules(t *testing.T) {
	ownerToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	approverToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	payablesAdminToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	usdCurrency, err := app.FindRecordById("currencies", testUSDCurrencyID)
	if err != nil {
		t.Fatalf("failed loading USD currency fixture: %v", err)
	}
	usdRate := usdCurrency.GetFloat("rate")

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, total = 25, submitted = 0, approved = '', rejected = '', settled_total = 0, settled = '', settler = ''
		WHERE id = '77i1224mudailrb'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing foreign expense row: %v", err)
	}

	submitBlocked := performTestAPIRequest(t, app, "POST", "/api/expenses/77i1224mudailrb/submit", strings.NewReader(`{}`), map[string]string{
		"Authorization": ownerToken,
	})
	mustStatus(t, submitBlocked, 400)
	if !strings.Contains(mustReadBody(t, submitBlocked), "settled total is required before submitting a foreign-currency expense") {
		t.Fatalf("expected submit to require settled_total, body=%s", submitBlocked.Body.String())
	}

	expectedSettledTotal := utilities.IndicativeHomeAmount(25, utilities.CurrencyInfo{Rate: usdRate})

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET submitted = 0, approved = '', settled_total = {:settledTotal}
		WHERE id = '77i1224mudailrb'
	`).Bind(dbx.Params{"settledTotal": utilities.RoundCurrencyAmount(expectedSettledTotal * 1.25)}).Execute(); err != nil {
		t.Fatalf("failed setting foreign expense settled_total: %v", err)
	}

	submitOutOfRange := performTestAPIRequest(t, app, "POST", "/api/expenses/77i1224mudailrb/submit", strings.NewReader(`{}`), map[string]string{
		"Authorization": ownerToken,
	})
	mustStatus(t, submitOutOfRange, 400)
	if !strings.Contains(mustReadBody(t, submitOutOfRange), "settled total must be within 20% of the current CAD equivalent") {
		t.Fatalf("expected submit to enforce settlement tolerance, body=%s", submitOutOfRange.Body.String())
	}

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET settled_total = {:settledTotal}
		WHERE id = '77i1224mudailrb'
	`).Bind(dbx.Params{"settledTotal": expectedSettledTotal}).Execute(); err != nil {
		t.Fatalf("failed restoring foreign expense settled_total: %v", err)
	}

	submitAllowed := performTestAPIRequest(t, app, "POST", "/api/expenses/77i1224mudailrb/submit", strings.NewReader(`{}`), map[string]string{
		"Authorization": ownerToken,
	})
	mustStatus(t, submitAllowed, 200)

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET approved = '2026-04-03 12:00:00.000Z'
		WHERE id = '77i1224mudailrb'
	`).Execute(); err != nil {
		t.Fatalf("failed approving foreign expense fixture: %v", err)
	}

	commitExpenseRes := performTestAPIRequest(t, app, "POST", "/api/expenses/77i1224mudailrb/commit", strings.NewReader(`{}`), map[string]string{
		"Authorization": commitToken,
	})
	mustStatus(t, commitExpenseRes, 200)

	committedForeignExpense, err := app.FindRecordById("expenses", "77i1224mudailrb")
	if err != nil {
		t.Fatalf("failed loading committed foreign expense: %v", err)
	}
	if committedForeignExpense.GetDateTime("committed").IsZero() {
		t.Fatal("expected foreign expense with settled_total to commit successfully")
	}

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, submitted = 1, approved = '2026-04-03 12:00:00.000Z', committed = '', committer = '', settled_total = 500, settled = '', settler = ''
		WHERE id = 'eqhozipupteogp8'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing unsettled foreign on-account expense: %v", err)
	}

	commitBlocked := performTestAPIRequest(t, app, "POST", "/api/expenses/eqhozipupteogp8/commit", strings.NewReader(`{}`), map[string]string{
		"Authorization": commitToken,
	})
	mustStatus(t, commitBlocked, 400)
	if !strings.Contains(mustReadBody(t, commitBlocked), "foreign-currency on-account and corporate card expenses must be settled before commit") {
		t.Fatalf("expected commit to require settlement for foreign on-account expense, body=%s", commitBlocked.Body.String())
	}

	settleForCommit := performTestAPIRequest(t, app, "POST", "/api/expenses/eqhozipupteogp8/settle", strings.NewReader(`{"settled_total":500.25}`), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, settleForCommit, 200)

	commitAllowed := performTestAPIRequest(t, app, "POST", "/api/expenses/eqhozipupteogp8/commit", strings.NewReader(`{}`), map[string]string{
		"Authorization": commitToken,
	})
	mustStatus(t, commitAllowed, 200)

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, committed = '', committer = '', rejected = '', rejection_reason = '', rejector = '', submitted = 1,
		    approved = '2026-04-03 12:00:00.000Z', settled_total = 90, settler = 'tqqf7q0f3378rvp', settled = '2026-04-03 12:30:00.000Z'
		WHERE id = 'b4o6xph4ngwx4nw'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing rejectable foreign corp card expense: %v", err)
	}

	rejectCard := performTestAPIRequest(t, app, "POST", "/api/expenses/b4o6xph4ngwx4nw/reject", strings.NewReader(`{"rejection_reason":"bad fx receipt"}`), map[string]string{
		"Authorization": approverToken,
	})
	mustStatus(t, rejectCard, 200)

	rejectedCard, err := app.FindRecordById("expenses", "b4o6xph4ngwx4nw")
	if err != nil {
		t.Fatalf("failed loading rejected foreign corp card expense: %v", err)
	}
	if rejectedCard.GetFloat("settled_total") != 0 || rejectedCard.GetString("settler") != "" || !rejectedCard.GetDateTime("settled").IsZero() {
		t.Fatalf("expected rejection to clear settlement for foreign corp card, got settled_total=%v settler=%q settled=%v",
			rejectedCard.GetFloat("settled_total"),
			rejectedCard.GetString("settler"),
			rejectedCard.GetDateTime("settled"),
		)
	}

	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE expenses
		SET currency = {:currencyId}, committed = '', committer = '', rejected = '', rejection_reason = '', rejector = '', submitted = 1,
		    approved = '2026-04-03 12:00:00.000Z', settled_total = 123.45, settler = '', settled = ''
		WHERE id = '77i1224mudailrb'
	`).Bind(dbx.Params{"currencyId": testUSDCurrencyID}).Execute(); err != nil {
		t.Fatalf("failed preparing rejectable foreign Expense row: %v", err)
	}

	rejectExpense := performTestAPIRequest(t, app, "POST", "/api/expenses/77i1224mudailrb/reject", strings.NewReader(`{"rejection_reason":"revise settlement"}`), map[string]string{
		"Authorization": approverToken,
	})
	mustStatus(t, rejectExpense, 200)

	rejectedExpense, err := app.FindRecordById("expenses", "77i1224mudailrb")
	if err != nil {
		t.Fatalf("failed loading rejected foreign Expense row: %v", err)
	}
	if got := rejectedExpense.GetFloat("settled_total"); got != 123.45 {
		t.Fatalf("expected rejection to preserve user-entered settled_total for foreign Expense, got %v", got)
	}
}

func TestForeignExpenseSettlementRoute_RequiresPOWhenSettledTotalExceedsNoPOLimit(t *testing.T) {
	payablesAdminToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	settleRes := performTestAPIRequest(t, app, "POST", "/api/expenses/"+testForeignNoPOSettleExpenseID+"/settle", strings.NewReader(`{"settled_total":101.25}`), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, settleRes, 400)

	body := mustReadBody(t, settleRes)
	if !strings.Contains(body, "purchase order is required") {
		t.Fatalf("expected settle route to require a purchase order once settled CAD exceeds the limit, body=%s", body)
	}
}

func TestForeignExpenseCommitRoute_RequiresPOWhenSettledTotalExceedsNoPOLimit(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	commitRes := performTestAPIRequest(t, app, "POST", "/api/expenses/"+testForeignNoPOCommitExpenseID+"/commit", strings.NewReader(`{}`), map[string]string{
		"Authorization": commitToken,
	})
	mustStatus(t, commitRes, 400)

	body := mustReadBody(t, commitRes)
	if !strings.Contains(body, "purchase order is required") {
		t.Fatalf("expected commit route to require a purchase order once settled CAD exceeds the limit, body=%s", body)
	}
}

func TestForeignExpenseSettlementRoute_AllowsSettlementWhenSettledTotalStaysBelowNoPOLimit(t *testing.T) {
	payablesAdminToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	settleRes := performTestAPIRequest(t, app, "POST", "/api/expenses/"+testForeignNoPOSettleOKExpenseID+"/settle", strings.NewReader(`{"settled_total":94.50}`), map[string]string{
		"Authorization": payablesAdminToken,
	})
	mustStatus(t, settleRes, 200)

	record, err := app.FindRecordById("expenses", testForeignNoPOSettleOKExpenseID)
	if err != nil {
		t.Fatalf("failed loading settled fixture: %v", err)
	}
	if got := record.GetFloat("settled_total"); got != 94.5 {
		t.Fatalf("expected settled_total to be updated to 94.5, got %v", got)
	}
	if got := record.GetString("settler"); got != "tqqf7q0f3378rvp" {
		t.Fatalf("expected settler to be bookkeeper, got %q", got)
	}
	if record.GetDateTime("settled").IsZero() {
		t.Fatal("expected settled timestamp to be populated")
	}
}

func TestForeignExpenseCommitRoute_AllowsCommitWhenSettledTotalStaysBelowNoPOLimit(t *testing.T) {
	commitToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	commitRes := performTestAPIRequest(t, app, "POST", "/api/expenses/"+testForeignNoPOCommitOKExpenseID+"/commit", strings.NewReader(`{}`), map[string]string{
		"Authorization": commitToken,
	})
	mustStatus(t, commitRes, 200)

	record, err := app.FindRecordById("expenses", testForeignNoPOCommitOKExpenseID)
	if err != nil {
		t.Fatalf("failed loading committed fixture: %v", err)
	}
	if record.GetDateTime("committed").IsZero() {
		t.Fatal("expected committed timestamp to be populated")
	}
	if got := record.GetString("committer"); got != "wegviunlyr2jjjv" {
		t.Fatalf("expected committer to be fakemanager, got %q", got)
	}
}
