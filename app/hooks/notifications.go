package hooks

import (
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func ProcessNotification(app core.App, e *core.RecordRequestEvent) error {
	// set the status_updated field to now if the value of status has changed
	if e.Record.Get("status") != e.Record.Original().Get("status") {
		e.Record.Set("status_updated", time.Now())
	}
	return nil
}
