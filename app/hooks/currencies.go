package hooks

import (
	"net/http"
	"strings"
	"tybalt/errs"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/core"
)

func ProcessCurrency(_ core.App, e *core.RecordRequestEvent) error {
	record := e.Record

	code := strings.ToUpper(strings.TrimSpace(record.GetString("code")))
	record.Set("code", code)
	record.Set("symbol", strings.TrimSpace(record.GetString("symbol")))

	if strings.EqualFold(code, utilities.HomeCurrencyCode) {
		record.Set("rate", 1)
		return nil
	}

	if code != "" && record.GetFloat("rate") <= 0 {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "hook error when processing currency",
			Data: map[string]errs.CodeError{
				"rate": {
					Code:    "must_be_positive",
					Message: "foreign currencies must have a positive rate",
				},
			},
		}
	}

	return nil
}
