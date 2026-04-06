package utilities

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var currencyRatesHTTPClient = &http.Client{Timeout: 15 * time.Second}

type bankOfCanadaObservation struct {
	Date string                       `json:"d"`
	Data map[string]bankOfCanadaValue `json:"-"`
}

type bankOfCanadaValue struct {
	Value string `json:"v"`
}

type bankOfCanadaObservationsResponse struct {
	Observations []map[string]json.RawMessage `json:"observations"`
}

type CurrencyRateSyncResult struct {
	Updated      int `json:"updated"`
	SkippedNewer int `json:"skipped_newer"`
}

func SyncCurrencyRates(app core.App) error {
	_, err := SyncCurrencyRatesWithResult(app)
	return err
}

func SyncCurrencyRatesWithResult(app core.App) (CurrencyRateSyncResult, error) {
	type currencyRow struct {
		ID   string `db:"id"`
		Code string `db:"code"`
	}

	result := CurrencyRateSyncResult{}
	var rows []currencyRow
	if err := app.DB().NewQuery(`
		SELECT id, code
		FROM currencies
		WHERE UPPER(COALESCE(code, '')) != {:homeCode}
		ORDER BY COALESCE(ui_sort, 999999), code
	`).Bind(dbx.Params{
		"homeCode": HomeCurrencyCode,
	}).All(&rows); err != nil {
		return result, fmt.Errorf("failed listing currencies for rate sync: %w", err)
	}

	var syncErrors []error
	for _, row := range rows {
		updated, skippedNewer, err := refreshCurrencyRateWithStatus(app, row.ID, row.Code)
		if updated {
			result.Updated++
		}
		if skippedNewer {
			result.SkippedNewer++
		}
		if err != nil {
			syncErrors = append(syncErrors, fmt.Errorf("%s: %w", row.Code, err))
		}
	}

	return result, errors.Join(syncErrors...)
}

func refreshCurrencyRate(app core.App, currencyID string, code string) error {
	_, _, err := refreshCurrencyRateWithStatus(app, currencyID, code)
	return err
}

func refreshCurrencyRateWithStatus(app core.App, currencyID string, code string) (bool, bool, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || code == HomeCurrencyCode {
		return false, false, nil
	}

	rate, rateDate, err := fetchLatestBankOfCanadaRate(code)
	if err != nil {
		return false, false, err
	}
	if rate <= 0 || strings.TrimSpace(rateDate) == "" {
		return false, false, nil
	}

	record, err := app.FindRecordById("currencies", currencyID)
	if err != nil {
		return false, false, err
	}
	if existingRateDate := strings.TrimSpace(record.GetString("rate_date")); existingRateDate != "" && rateDate <= existingRateDate {
		return false, true, nil
	}

	record.Set("rate", rate)
	record.Set("rate_date", rateDate)
	if err := app.Save(record); err != nil {
		return false, false, err
	}
	return true, false, nil
}

func fetchLatestBankOfCanadaRate(code string) (float64, string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, "", fmt.Errorf("currency code is required")
	}

	url := fmt.Sprintf(
		"https://www.bankofcanada.ca/valet/observations/FX%[1]sCAD/json?recent=5",
		code,
	)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}

	resp, err := currencyRatesHTTPClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("unexpected bank of canada status: %s", resp.Status)
	}

	var payload bankOfCanadaObservationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, "", err
	}

	seriesName := fmt.Sprintf("FX%sCAD", code)
	for _, rawObservation := range payload.Observations {
		observation := bankOfCanadaObservation{Data: map[string]bankOfCanadaValue{}}
		for key, raw := range rawObservation {
			if key == "d" {
				if err := json.Unmarshal(raw, &observation.Date); err != nil {
					return 0, "", err
				}
				continue
			}

			var value bankOfCanadaValue
			if err := json.Unmarshal(raw, &value); err != nil {
				return 0, "", err
			}
			observation.Data[key] = value
		}

		seriesValue, ok := observation.Data[seriesName]
		if !ok || strings.TrimSpace(seriesValue.Value) == "" {
			continue
		}

		rate, err := strconv.ParseFloat(seriesValue.Value, 64)
		if err != nil {
			return 0, "", err
		}
		return rate, observation.Date, nil
	}

	return 0, "", nil
}
