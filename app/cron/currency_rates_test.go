package cron

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func useCurrencyRateTestServer(t *testing.T, handler http.HandlerFunc) {
	t.Helper()

	server := httptest.NewServer(handler)
	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}

	oldClient := currencyRatesHTTPClient
	testClient := server.Client()
	baseTransport := testClient.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	testClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = baseURL.Scheme
		cloned.URL.Host = baseURL.Host
		return baseTransport.RoundTrip(cloned)
	})
	currencyRatesHTTPClient = testClient

	t.Cleanup(func() {
		currencyRatesHTTPClient = oldClient
		server.Close()
	})
}

func loadCurrencyByCode(t *testing.T, app core.App, code string) *core.Record {
	t.Helper()

	record, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", dbx.Params{"code": code})
	if err != nil {
		t.Fatalf("failed to load %s currency: %v", code, err)
	}
	return record
}

func TestFetchLatestBankOfCanadaRate_ReturnsNewestNonEmptyObservation(t *testing.T) {
	useCurrencyRateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/valet/observations/FXUSDCAD/json") {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{
			"observations": [
				{"d":"2026-04-03","FXUSDCAD":{"v":""}},
				{"d":"2026-04-02","FXUSDCAD":{"v":"1.4012"}}
			]
		}`)
	})

	rate, rateDate, err := fetchLatestBankOfCanadaRate("USD")
	if err != nil {
		t.Fatalf("expected fetchLatestBankOfCanadaRate to succeed, got %v", err)
	}
	if rate != 1.4012 || rateDate != "2026-04-02" {
		t.Fatalf("expected 1.4012 on 2026-04-02, got %v on %s", rate, rateDate)
	}
}

func TestRefreshCurrencyRate_SkipsStaleObservation(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	usd := loadCurrencyByCode(t, app, "USD")
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE currencies
		SET rate = 1.5000, rate_date = '2026-04-03'
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": usd.Id}).Execute(); err != nil {
		t.Fatalf("failed preparing USD currency fixture: %v", err)
	}

	useCurrencyRateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"observations": [
				{"d":"2026-04-02","FXUSDCAD":{"v":"1.4012"}}
			]
		}`)
	})

	if err := refreshCurrencyRate(app, usd.Id, "USD"); err != nil {
		t.Fatalf("expected refreshCurrencyRate to succeed, got %v", err)
	}

	reloaded, err := app.FindRecordById("currencies", usd.Id)
	if err != nil {
		t.Fatalf("failed to reload USD currency: %v", err)
	}
	if reloaded.GetFloat("rate") != 1.5 || !strings.HasPrefix(reloaded.GetString("rate_date"), "2026-04-03") {
		t.Fatalf("expected stale response to leave existing rate untouched, got rate=%v rate_date=%q", reloaded.GetFloat("rate"), reloaded.GetString("rate_date"))
	}
}

func TestRefreshCurrencyRate_EmptyResponseLeavesExistingRateUntouched(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	usd := loadCurrencyByCode(t, app, "USD")
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE currencies
		SET rate = 1.3333, rate_date = '2026-04-01'
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": usd.Id}).Execute(); err != nil {
		t.Fatalf("failed preparing USD currency fixture: %v", err)
	}

	useCurrencyRateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"observations":[]}`)
	})

	if err := refreshCurrencyRate(app, usd.Id, "USD"); err != nil {
		t.Fatalf("expected refreshCurrencyRate to succeed, got %v", err)
	}

	reloaded, err := app.FindRecordById("currencies", usd.Id)
	if err != nil {
		t.Fatalf("failed to reload USD currency: %v", err)
	}
	if reloaded.GetFloat("rate") != 1.3333 || !strings.HasPrefix(reloaded.GetString("rate_date"), "2026-04-01") {
		t.Fatalf("expected empty API response to leave rate untouched, got rate=%v rate_date=%q", reloaded.GetFloat("rate"), reloaded.GetString("rate_date"))
	}
}

func TestSyncCurrencyRates_UpdatesNonHomeCurrenciesOnly(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	usd := loadCurrencyByCode(t, app, "USD")
	cad := loadCurrencyByCode(t, app, "CAD")
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE currencies
		SET rate = 1.1111, rate_date = '2026-04-01'
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": usd.Id}).Execute(); err != nil {
		t.Fatalf("failed preparing USD currency fixture: %v", err)
	}
	if _, err := app.NonconcurrentDB().NewQuery(`
		UPDATE currencies
		SET rate = 1.0000, rate_date = '2026-01-01'
		WHERE id = {:id}
	`).Bind(dbx.Params{"id": cad.Id}).Execute(); err != nil {
		t.Fatalf("failed preparing CAD currency fixture: %v", err)
	}

	requestedSeries := []string{}
	useCurrencyRateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requestedSeries = append(requestedSeries, r.URL.Path)
		switch {
		case strings.Contains(r.URL.Path, "FXUSDCAD"):
			fmt.Fprint(w, `{
				"observations": [
					{"d":"2026-04-03","FXUSDCAD":{"v":"1.4567"}}
				]
			}`)
		default:
			t.Fatalf("unexpected currency rate request: %s", r.URL.Path)
		}
	})

	syncCurrencyRates(app)

	reloadedUSD, err := app.FindRecordById("currencies", usd.Id)
	if err != nil {
		t.Fatalf("failed to reload USD currency: %v", err)
	}
	if reloadedUSD.GetFloat("rate") != 1.4567 || !strings.HasPrefix(reloadedUSD.GetString("rate_date"), "2026-04-03") {
		t.Fatalf("expected USD rate to update, got rate=%v rate_date=%q", reloadedUSD.GetFloat("rate"), reloadedUSD.GetString("rate_date"))
	}

	reloadedCAD, err := app.FindRecordById("currencies", cad.Id)
	if err != nil {
		t.Fatalf("failed to reload CAD currency: %v", err)
	}
	if reloadedCAD.GetFloat("rate") != 1 || !strings.HasPrefix(reloadedCAD.GetString("rate_date"), "2026-01-01") {
		t.Fatalf("expected CAD rate to remain unchanged, got rate=%v rate_date=%q", reloadedCAD.GetFloat("rate"), reloadedCAD.GetString("rate_date"))
	}
	if len(requestedSeries) != 1 || !strings.Contains(requestedSeries[0], "FXUSDCAD") {
		t.Fatalf("expected only USD lookup during sync, got %v", requestedSeries)
	}
}
