// helpers.go contains shared utility helpers for notification flows.
//
// This includes URL/status helpers, profile name lookup, immediate recipient
// fan-out, send-tail handling, and row value extraction helpers.
package notifications

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func appURL(app core.App) string {
	return strings.TrimRight(strings.TrimSpace(app.Settings().Meta.AppURL), "/")
}

// BuildActionURL converts an app-relative path into an absolute URL rooted at
// app.Settings().Meta.AppURL.
//
// If the app URL is empty, it returns an empty string. If path is empty, it
// returns the base app URL. Non-leading-slash paths are normalized to include a
// leading slash.
func BuildActionURL(app core.App, path string) string {
	base := appURL(app)
	if base == "" {
		return ""
	}
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return base
	}
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}
	return base + cleanPath
}

func unresolvedLegacyPlaceholder(text string) string {
	if strings.Contains(text, "{APP_URL}") {
		return "{APP_URL}"
	}
	if strings.Contains(text, "{:RECORD_ID}") {
		return "{:RECORD_ID}"
	}
	return ""
}

// WriteStatusUpdated is a record hook helper that updates the
// notifications.status_updated field whenever status changes or a record is
// newly created.
//
// This should be registered on notification record save events so status
// transitions performed via app.Save() keep status_updated accurate.
func WriteStatusUpdated(app core.App, e *core.RecordEvent) error {
	if e.Record.Get("status") != e.Record.Original().Get("status") || e.Record.IsNew() {
		e.Record.Set("status_updated", time.Now())
	}
	return nil
}

func sendQueuedIfRequested(app core.App, send bool, context string) error {
	if !send {
		// Intentional no-op for queue-only callers.
		return nil
	}
	sentCount, err := SendNotifications(app)
	if err != nil {
		return fmt.Errorf("error sending notifications: %v", err)
	}
	app.Logger().Info(
		context,
		"sent_count", sentCount,
	)
	return nil
}

func getProfileDisplayName(app core.App, uid string) (name string, profile *core.Record, err error) {
	profile, err = app.FindFirstRecordByFilter("profiles", "uid = {:uid}", dbx.Params{
		"uid": uid,
	})
	if err != nil {
		return "", nil, err
	}
	name = strings.TrimSpace(profile.GetString("given_name") + " " + profile.GetString("surname"))
	return name, profile, nil
}

func createAndSendToRecipients(
	app core.App,
	templateCode string,
	recipients []string,
	data map[string]any,
	system bool,
	actorUID string,
	logContext map[string]any,
) (createdCount int) {
	for _, recipientUID := range recipients {
		notificationID, err := DispatchNotification(app, DispatchArgs{
			TemplateCode: templateCode,
			RecipientUID: recipientUID,
			Data:         data,
			System:       system,
			ActorUID:     actorUID,
			Mode:         DeliveryImmediate,
		})
		if err != nil {
			logArgs := []any{
				"template_code", templateCode,
				"recipient_uid", recipientUID,
				"error", err,
			}
			for key, value := range logContext {
				logArgs = append(logArgs, key, value)
			}
			app.Logger().Error("error creating immediate notification", logArgs...)
			continue
		}
		if notificationID == "" {
			continue
		}
		createdCount++
	}

	return createdCount
}

func rowStringValue(row dbx.NullStringMap, key string) string {
	value, ok := row[key]
	if !ok || !value.Valid {
		return ""
	}
	return value.String
}
