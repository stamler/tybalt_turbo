package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestCreateCurrencyRatesReloadHandler_RequiresAdminAndRunsSync(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	adminAuth, err := app.FindAuthRecordByEmail("users", "author@soup.com")
	if err != nil {
		t.Fatalf("failed loading admin auth record: %v", err)
	}

	originalSync := runCurrencyRateSync
	t.Cleanup(func() {
		runCurrencyRateSync = originalSync
	})

	called := false
	runCurrencyRateSync = func(core.App) (utilities.CurrencyRateSyncResult, error) {
		called = true
		return utilities.CurrencyRateSyncResult{SkippedNewer: 1}, nil
	}

	req := httptest.NewRequest(http.MethodPost, "/api/currencies/reload_rates", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:  app,
		Auth: adminAuth,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	if err := createCurrencyRatesReloadHandler(app)(event); err != nil {
		t.Fatalf("expected handler to succeed, got %v", err)
	}
	if !called {
		t.Fatal("expected currency sync to be triggered")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", rec.Code)
	}

	var body currencyRatesReloadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}
	if !body.OK {
		t.Fatalf("expected ok=true response, got %+v", body)
	}
	if body.SkippedNewer != 1 {
		t.Fatalf("expected skipped_newer=1, got %+v", body)
	}
}

func TestCreateCurrencyRatesReloadHandler_ReturnsSyncFailure(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	adminAuth, err := app.FindAuthRecordByEmail("users", "author@soup.com")
	if err != nil {
		t.Fatalf("failed loading admin auth record: %v", err)
	}

	originalSync := runCurrencyRateSync
	t.Cleanup(func() {
		runCurrencyRateSync = originalSync
	})

	runCurrencyRateSync = func(core.App) (utilities.CurrencyRateSyncResult, error) {
		return utilities.CurrencyRateSyncResult{}, errors.New("boom")
	}

	req := httptest.NewRequest(http.MethodPost, "/api/currencies/reload_rates", nil)
	rec := httptest.NewRecorder()
	event := &core.RequestEvent{
		App:  app,
		Auth: adminAuth,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	}

	err = createCurrencyRatesReloadHandler(app)(event)
	if err == nil {
		t.Fatal("expected handler to return an api error")
	}

	apiErr, ok := err.(*router.ApiError)
	if !ok {
		t.Fatalf("expected *router.ApiError, got %T", err)
	}
	if apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("expected 500 status, got %d", apiErr.Status)
	}
	if apiErr.Message != "Failed to reload currency rates." {
		t.Fatalf("unexpected error message %q", apiErr.Message)
	}
}
