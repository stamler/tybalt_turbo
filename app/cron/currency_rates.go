package cron

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

var (
	currencyRatesHTTPClient = &http.Client{Timeout: 15 * time.Second}
	currencyRatesNow        = func() time.Time { return time.Now().UTC() }
)

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

func syncCurrencyRates(app core.App) {
	type currencyRow struct {
		ID   string `db:"id"`
		Code string `db:"code"`
	}

	var rows []currencyRow
	if err := app.DB().NewQuery(`
		SELECT id, code
		FROM currencies
		WHERE UPPER(COALESCE(code, '')) != {:homeCode}
		ORDER BY COALESCE(ui_sort, 999999), code
	`).Bind(dbx.Params{
		"homeCode": utilities.HomeCurrencyCode,
	}).All(&rows); err != nil {
		app.Logger().Error("failed listing currencies for rate sync", "error", err)
		return
	}

	for _, row := range rows {
		if err := refreshCurrencyRate(app, row.ID, row.Code); err != nil {
			app.Logger().Error(
				"failed refreshing currency rate",
				"currency_id", row.ID,
				"currency_code", row.Code,
				"error", err,
			)
		}
	}
}

func refreshCurrencyRate(app core.App, currencyID string, code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" || code == utilities.HomeCurrencyCode {
		return nil
	}

	rate, rateDate, err := fetchLatestBankOfCanadaRate(code)
	if err != nil {
		return err
	}
	if rate <= 0 || strings.TrimSpace(rateDate) == "" {
		return nil
	}

	record, err := app.FindRecordById("currencies", currencyID)
	if err != nil {
		return err
	}
	if existingRateDate := strings.TrimSpace(record.GetString("rate_date")); existingRateDate != "" && rateDate <= existingRateDate {
		return nil
	}

	record.Set("rate", rate)
	record.Set("rate_date", rateDate)
	return app.Save(record)
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
