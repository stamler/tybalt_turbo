package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"tybalt/internal/testutils"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

const missingExceptionWeekEnding = "2024-06-29"

type timeSheetMissingExceptionRouteResponse struct {
	UID        string `json:"uid"`
	WeekEnding string `json:"week_ending"`
	Ignored    bool   `json:"ignored"`
}

type trackingUserRow struct {
	ID string `json:"id"`
}

func TestTimeSheetMissingExceptionsRouteFlow(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	committerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	missingBefore := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing")
	if len(missingBefore) == 0 {
		t.Fatal("expected at least one missing user in seed data")
	}
	targetUID := missingBefore[0].ID

	notExpectedBefore := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/not_expected")
	var stableNotExpectedUID string
	if len(notExpectedBefore) > 0 {
		stableNotExpectedUID = notExpectedBefore[0].ID
	}

	createRec := performJSONRequest(t, app, http.MethodPost, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/"+targetUID, committerToken, nil)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create status = %d, want %d; body=%s", createRec.Code, http.StatusOK, createRec.Body.String())
	}

	createResp := decodeMissingExceptionResponse(t, createRec)
	if createResp.UID != targetUID || createResp.WeekEnding != missingExceptionWeekEnding || !createResp.Ignored {
		t.Fatalf("unexpected create response: %+v", createResp)
	}
	if countMissingExceptions(t, app, targetUID, missingExceptionWeekEnding) != 1 {
		t.Fatalf("expected exactly one missing exception after create")
	}

	createAgainRec := performJSONRequest(t, app, http.MethodPost, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/"+targetUID, committerToken, nil)
	if createAgainRec.Code != http.StatusOK {
		t.Fatalf("duplicate create status = %d, want %d; body=%s", createAgainRec.Code, http.StatusOK, createAgainRec.Body.String())
	}
	if countMissingExceptions(t, app, targetUID, missingExceptionWeekEnding) != 1 {
		t.Fatalf("expected duplicate create to keep one missing exception row")
	}

	missingAfterCreate := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing")
	ignoredAfterCreate := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/ignored")
	notExpectedAfterCreate := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/not_expected")

	if trackingRowsContainUID(missingAfterCreate, targetUID) {
		t.Fatalf("missing list still contains ignored uid %s", targetUID)
	}
	if !trackingRowsContainUID(ignoredAfterCreate, targetUID) {
		t.Fatalf("ignored list does not contain uid %s after create", targetUID)
	}
	if stableNotExpectedUID != "" && !trackingRowsContainUID(notExpectedAfterCreate, stableNotExpectedUID) {
		t.Fatalf("not_expected list lost stable uid %s after create", stableNotExpectedUID)
	}

	deleteRec := performJSONRequest(t, app, http.MethodDelete, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/"+targetUID, committerToken, nil)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body=%s", deleteRec.Code, http.StatusOK, deleteRec.Body.String())
	}

	deleteResp := decodeMissingExceptionResponse(t, deleteRec)
	if deleteResp.UID != targetUID || deleteResp.WeekEnding != missingExceptionWeekEnding || deleteResp.Ignored {
		t.Fatalf("unexpected delete response: %+v", deleteResp)
	}
	if countMissingExceptions(t, app, targetUID, missingExceptionWeekEnding) != 0 {
		t.Fatalf("expected zero missing exception rows after delete")
	}

	deleteAgainRec := performJSONRequest(t, app, http.MethodDelete, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/"+targetUID, committerToken, nil)
	if deleteAgainRec.Code != http.StatusOK {
		t.Fatalf("duplicate delete status = %d, want %d; body=%s", deleteAgainRec.Code, http.StatusOK, deleteAgainRec.Body.String())
	}

	missingAfterDelete := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing")
	ignoredAfterDelete := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/ignored")
	notExpectedAfterDelete := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/not_expected")

	if !trackingRowsContainUID(missingAfterDelete, targetUID) {
		t.Fatalf("missing list does not contain uid %s after delete", targetUID)
	}
	if trackingRowsContainUID(ignoredAfterDelete, targetUID) {
		t.Fatalf("ignored list still contains uid %s after delete", targetUID)
	}
	if stableNotExpectedUID != "" && !trackingRowsContainUID(notExpectedAfterDelete, stableNotExpectedUID) {
		t.Fatalf("not_expected list lost stable uid %s after delete", stableNotExpectedUID)
	}
}

func TestTimeSheetMissingExceptionsRouteAuthAndValidation(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	committerToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}
	reportToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}
	noClaimsToken, err := testutils.GenerateRecordToken("users", "noclaims@example.com")
	if err != nil {
		t.Fatal(err)
	}

	missingRows := getTrackingRows(t, app, committerToken, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing")
	if len(missingRows) == 0 {
		t.Fatal("expected at least one missing user in seed data")
	}
	targetUID := missingRows[0].ID

	reportCreate := performJSONRequest(t, app, http.MethodPost, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/"+targetUID, reportToken, nil)
	if reportCreate.Code != http.StatusForbidden {
		t.Fatalf("report create status = %d, want %d; body=%s", reportCreate.Code, http.StatusForbidden, reportCreate.Body.String())
	}

	noClaimsCreate := performJSONRequest(t, app, http.MethodPost, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/"+targetUID, noClaimsToken, nil)
	if noClaimsCreate.Code != http.StatusForbidden {
		t.Fatalf("no-claims create status = %d, want %d; body=%s", noClaimsCreate.Code, http.StatusForbidden, noClaimsCreate.Body.String())
	}

	invalidWeekCreate := performJSONRequest(t, app, http.MethodPost, "/api/time_sheets/tracking/weeks/not-a-date/missing_exceptions/"+targetUID, committerToken, nil)
	if invalidWeekCreate.Code != http.StatusBadRequest {
		t.Fatalf("invalid week create status = %d, want %d; body=%s", invalidWeekCreate.Code, http.StatusBadRequest, invalidWeekCreate.Body.String())
	}

	missingUserCreate := performJSONRequest(t, app, http.MethodPost, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/missing_exceptions/uid_missing_for_exception", committerToken, nil)
	if missingUserCreate.Code != http.StatusNotFound {
		t.Fatalf("missing user create status = %d, want %d; body=%s", missingUserCreate.Code, http.StatusNotFound, missingUserCreate.Body.String())
	}

	reportIgnored := performJSONRequest(t, app, http.MethodGet, "/api/time_sheets/tracking/weeks/"+missingExceptionWeekEnding+"/ignored", reportToken, nil)
	if reportIgnored.Code != http.StatusOK {
		t.Fatalf("report ignored status = %d, want %d; body=%s", reportIgnored.Code, http.StatusOK, reportIgnored.Body.String())
	}
}

func decodeMissingExceptionResponse(t *testing.T, rec *httptest.ResponseRecorder) timeSheetMissingExceptionRouteResponse {
	t.Helper()

	var response timeSheetMissingExceptionRouteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response body %q: %v", rec.Body.String(), err)
	}
	return response
}

func getTrackingRows(t *testing.T, app *tests.TestApp, token string, path string) []trackingUserRow {
	t.Helper()

	rec := performJSONRequest(t, app, http.MethodGet, path, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tracking request %s status = %d, want %d; body=%s", path, rec.Code, http.StatusOK, rec.Body.String())
	}

	var rows []trackingUserRow
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatalf("failed to decode tracking rows %q: %v", rec.Body.String(), err)
	}
	return rows
}

func trackingRowsContainUID(rows []trackingUserRow, uid string) bool {
	for _, row := range rows {
		if row.ID == uid {
			return true
		}
	}
	return false
}

func countMissingExceptions(t *testing.T, app *tests.TestApp, uid string, weekEnding string) int64 {
	t.Helper()

	var result struct {
		Count int64 `db:"count"`
	}
	if err := app.DB().NewQuery(`
		SELECT COUNT(*) AS count
		FROM time_sheet_missing_exceptions
		WHERE uid = {:uid} AND week_ending = {:week_ending}
	`).Bind(dbx.Params{
		"uid":         uid,
		"week_ending": weekEnding,
	}).One(&result); err != nil {
		t.Fatalf("failed to count missing exceptions: %v", err)
	}
	return result.Count
}

func performJSONRequest(t *testing.T, app *tests.TestApp, method string, path string, token string, body any) *httptest.ResponseRecorder {
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
