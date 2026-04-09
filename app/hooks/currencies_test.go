package hooks

import (
	"net/http"
	"testing"
	"tybalt/internal/testseed"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

func TestProcessCurrency_RejectsNonPositiveForeignRate(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	currenciesCollection, err := app.FindCollectionByNameOrId("currencies")
	if err != nil {
		t.Fatalf("failed to load currencies collection: %v", err)
	}

	record := core.NewRecord(currenciesCollection)
	record.Set("code", " usd ")
	record.Set("symbol", " $ ")
	record.Set("rate", 0)

	err = ProcessCurrency(app, makeRecordRequestEvent(app, record, nil))
	assertHookErrorCode(t, err, http.StatusBadRequest, "rate", "must_be_positive")
}

func TestProcessCurrency_NormalizesCADRateAndFormatting(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	currenciesCollection, err := app.FindCollectionByNameOrId("currencies")
	if err != nil {
		t.Fatalf("failed to load currencies collection: %v", err)
	}

	record := core.NewRecord(currenciesCollection)
	record.Set("code", " cad ")
	record.Set("symbol", " CAD ")
	record.Set("rate", 0)

	err = ProcessCurrency(app, makeRecordRequestEvent(app, record, nil))
	if err != nil {
		t.Fatalf("expected CAD currency processing to succeed, got %v", err)
	}

	if got := record.GetString("code"); got != utilities.HomeCurrencyCode {
		t.Fatalf("expected code normalized to %q, got %q", utilities.HomeCurrencyCode, got)
	}
	if got := record.GetString("symbol"); got != "CAD" {
		t.Fatalf("expected symbol trimmed to CAD, got %q", got)
	}
	if got := record.GetFloat("rate"); got != 1 {
		t.Fatalf("expected CAD rate normalized to 1, got %v", got)
	}
}
