package routes

import (
	"net/http"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func requireTimeClaim(app core.App, auth *core.Record) error {
	hasTimeClaim, err := utilities.HasClaim(app, auth, "time")
	if err != nil {
		return err
	}
	if !hasTimeClaim {
		return apis.NewApiError(http.StatusForbidden, "time claim required", nil)
	}
	return nil
}
