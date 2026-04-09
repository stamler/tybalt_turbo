package utilities

import (
	"testing"
	"tybalt/internal/testseed"
)

func TestResolveCurrencyInfo_BlankCurrencyUsesCADHomeRow(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	cad, err := FindHomeCurrency(app)
	if err != nil {
		t.Fatalf("failed to load CAD home currency: %v", err)
	}

	info, err := ResolveCurrencyInfo(app, "")
	if err != nil {
		t.Fatalf("ResolveCurrencyInfo returned error: %v", err)
	}

	if info.ID != cad.Id {
		t.Fatalf("expected blank currency to resolve to CAD row %q, got %q", cad.Id, info.ID)
	}
	if info.Code != HomeCurrencyCode {
		t.Fatalf("expected blank currency to resolve to code %q, got %q", HomeCurrencyCode, info.Code)
	}
	if info.Rate != 1 {
		t.Fatalf("expected CAD home currency rate 1, got %v", info.Rate)
	}
}

func TestResolveCurrencyInfo_BlankCurrencyFallsBackToCADIfCADRowMissing(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()

	cad, err := FindHomeCurrency(app)
	if err != nil {
		t.Fatalf("failed to load CAD home currency: %v", err)
	}
	if err := app.Delete(cad); err != nil {
		t.Fatalf("failed to delete CAD currency fixture: %v", err)
	}

	info, err := ResolveCurrencyInfo(app, "")
	if err != nil {
		t.Fatalf("ResolveCurrencyInfo returned error: %v", err)
	}

	if info.ID != "" {
		t.Fatalf("expected fallback CAD currency to have no backing row id, got %q", info.ID)
	}
	if info.Code != HomeCurrencyCode {
		t.Fatalf("expected fallback code %q, got %q", HomeCurrencyCode, info.Code)
	}
	if info.Symbol != HomeCurrencyCode {
		t.Fatalf("expected fallback symbol %q, got %q", HomeCurrencyCode, info.Symbol)
	}
	if info.Rate != 1 {
		t.Fatalf("expected fallback CAD rate 1, got %v", info.Rate)
	}
}

func TestCurrencyRateOrOne_DoesNotTreatInvalidForeignRateAsCAD(t *testing.T) {
	rate := CurrencyRateOrOne(CurrencyInfo{Code: "JPY", Rate: 0})
	if rate != 0 {
		t.Fatalf("expected invalid foreign rate to remain unusable, got %v", rate)
	}

	homeRate := CurrencyRateOrOne(CurrencyInfo{Code: HomeCurrencyCode, Rate: 0})
	if homeRate != 1 {
		t.Fatalf("expected home currency fallback rate 1, got %v", homeRate)
	}
}

func TestRequirePositiveForeignCurrencyRate(t *testing.T) {
	if err := RequirePositiveForeignCurrencyRate(CurrencyInfo{Code: HomeCurrencyCode, Rate: 0}); err != nil {
		t.Fatalf("expected CAD to allow fallback rate, got %v", err)
	}

	if err := RequirePositiveForeignCurrencyRate(CurrencyInfo{Code: "USD", Rate: 1.35}); err != nil {
		t.Fatalf("expected positive foreign rate to pass, got %v", err)
	}

	if err := RequirePositiveForeignCurrencyRate(CurrencyInfo{Code: "USD", Rate: 0}); err != ErrForeignCurrencyRateMissing {
		t.Fatalf("expected ErrForeignCurrencyRateMissing for zero foreign rate, got %v", err)
	}
}
