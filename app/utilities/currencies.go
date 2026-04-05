package utilities

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const HomeCurrencyCode = "CAD"

var ErrCurrencyNotFound = errors.New("currency not found")

type CurrencyInfo struct {
	ID       string
	Code     string
	Symbol   string
	Icon     string
	Rate     float64
	RateDate string
	Implicit bool
}

func ImplicitHomeCurrency() CurrencyInfo {
	return CurrencyInfo{
		Code:     HomeCurrencyCode,
		Symbol:   HomeCurrencyCode,
		Rate:     1,
		RateDate: "",
		Implicit: true,
	}
}

func currencyInfoFromRecord(record *core.Record) CurrencyInfo {
	if record == nil {
		return ImplicitHomeCurrency()
	}

	return CurrencyInfo{
		ID:       record.Id,
		Code:     strings.TrimSpace(record.GetString("code")),
		Symbol:   strings.TrimSpace(record.GetString("symbol")),
		Icon:     strings.TrimSpace(record.GetString("icon")),
		Rate:     record.GetFloat("rate"),
		RateDate: strings.TrimSpace(record.GetString("rate_date")),
	}
}

func FindCurrencyByID(app core.App, id string) (*core.Record, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: blank id", ErrCurrencyNotFound)
	}

	record, err := app.FindRecordById("currencies", id)
	if err != nil || record == nil {
		return nil, fmt.Errorf("%w: %s", ErrCurrencyNotFound, id)
	}

	return record, nil
}

func FindCurrencyByCode(app core.App, code string) (*core.Record, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, fmt.Errorf("%w: blank code", ErrCurrencyNotFound)
	}

	record, err := app.FindFirstRecordByFilter("currencies", "code = {:code}", dbx.Params{
		"code": code,
	})
	if err != nil || record == nil {
		return nil, fmt.Errorf("%w: %s", ErrCurrencyNotFound, code)
	}

	return record, nil
}

func FindHomeCurrency(app core.App) (*core.Record, error) {
	return FindCurrencyByCode(app, HomeCurrencyCode)
}

func ResolveCurrencyInfo(app core.App, currencyID string) (CurrencyInfo, error) {
	currencyID = strings.TrimSpace(currencyID)
	if currencyID == "" {
		home, err := FindHomeCurrency(app)
		if err != nil {
			return ImplicitHomeCurrency(), nil
		}
		info := currencyInfoFromRecord(home)
		if info.Code == "" {
			info.Code = HomeCurrencyCode
		}
		if info.Symbol == "" {
			info.Symbol = HomeCurrencyCode
		}
		if info.Rate == 0 {
			info.Rate = 1
		}
		return info, nil
	}

	record, err := FindCurrencyByID(app, currencyID)
	if err != nil {
		return CurrencyInfo{}, err
	}

	info := currencyInfoFromRecord(record)
	if info.Code == "" {
		info.Code = HomeCurrencyCode
	}
	if info.Symbol == "" {
		info.Symbol = info.Code
	}
	if strings.EqualFold(info.Code, HomeCurrencyCode) && info.Rate == 0 {
		info.Rate = 1
	}
	return info, nil
}

func CurrencyRateOrOne(info CurrencyInfo) float64 {
	if info.Rate > 0 {
		return info.Rate
	}
	return 1
}

func CurrencyCodeOrHome(info CurrencyInfo) string {
	code := strings.ToUpper(strings.TrimSpace(info.Code))
	if code == "" {
		return HomeCurrencyCode
	}
	return code
}

func IsHomeCurrencyInfo(info CurrencyInfo) bool {
	return strings.EqualFold(CurrencyCodeOrHome(info), HomeCurrencyCode)
}

func EffectiveCurrencyCode(app core.App, currencyID string) string {
	info, err := ResolveCurrencyInfo(app, currencyID)
	if err != nil {
		return HomeCurrencyCode
	}
	return CurrencyCodeOrHome(info)
}

func EffectiveApprovalTotalHome(record *core.Record) float64 {
	if record == nil {
		return 0
	}

	if value := record.GetFloat("approval_total_home"); value > 0 {
		return value
	}

	return record.GetFloat("approval_total")
}
