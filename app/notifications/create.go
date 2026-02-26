// create.go contains notification creation and dispatch entry points.
//
// It defines the public dispatch primitive used by callers across deferred and
// immediate flows and keeps feature-flag-aware record creation in one place.
package notifications

import (
	"encoding/json"
	"fmt"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// DispatchNotification creates a notification record and dispatches it according
// to args.Mode.
//
// Modes:
//   - DeliveryDeferred: create pending notification only.
//   - DeliveryImmediate: create notification, then attempt targeted send.
//
// Returns the created notification ID, or an empty string when creation is
// intentionally skipped (for example, disabled feature flag or fail-closed
// config read failure). In immediate mode, send errors are logged but not
// returned to preserve non-blocking business behavior.
func DispatchNotification(app core.App, args DispatchArgs) (notificationID string, err error) {
	if args.Mode != DeliveryDeferred && args.Mode != DeliveryImmediate {
		return "", fmt.Errorf("invalid delivery mode %q", args.Mode)
	}

	notificationID, err = createNotificationWithUser(app, args.TemplateCode, args.RecipientUID, args.Data, args.System, args.ActorUID)
	if err != nil {
		return "", err
	}

	if args.Mode == DeliveryImmediate && notificationID != "" {
		if err := SendNotificationByID(app, notificationID); err != nil {
			app.Logger().Error(
				"failed to send notification immediately after creation",
				"notification_id", notificationID,
				"template_code", args.TemplateCode,
				"error", err,
			)
		}
	}

	return notificationID, nil
}

func createNotificationWithUser(app core.App, templateCode string, recipientUID string, data map[string]any, system bool, actorUID string) (string, error) {
	enabled, err := utilities.IsNotificationFeatureEnabled(app, templateCode)
	if err != nil {
		app.Logger().Error(
			"error reading notifications feature config; skipping notification (fail-closed)",
			"template_code", templateCode,
			"error", err,
		)
		return "", nil
	}
	if !enabled {
		app.Logger().Info(
			"notification creation skipped because feature is disabled",
			"template_code", templateCode,
			"recipient_uid", recipientUID,
		)
		return "", nil
	}

	notificationCollection, err := app.FindCollectionByNameOrId("notifications")
	if err != nil {
		app.Logger().Error(
			"error finding notifications collection",
			"error", err,
		)
		return "", fmt.Errorf("error finding notifications collection: %v", err)
	}

	notificationTemplate, err := app.FindFirstRecordByFilter("notification_templates", "code = {:code}", dbx.Params{
		"code": templateCode,
	})
	if err != nil {
		app.Logger().Error(
			"error finding notification template",
			"template_code", templateCode,
			"error", err,
		)
		return "", fmt.Errorf("error finding notification template %s: %v", templateCode, err)
	}

	notificationRecord := core.NewRecord(notificationCollection)
	notificationRecord.Set("recipient", recipientUID)
	notificationRecord.Set("template", notificationTemplate.Get("id"))
	notificationRecord.Set("subject", notificationTemplate.Get("subject"))
	notificationRecord.Set("text_email", notificationTemplate.Get("text_email"))
	notificationRecord.Set("status", "pending")
	notificationRecord.Set("system_notification", system)
	if actorUID != "" {
		notificationRecord.Set("user", actorUID)
	}

	if len(data) > 0 {
		dataJSON, err := json.Marshal(data)
		if err != nil {
			app.Logger().Error(
				"error marshaling notification data",
				"error", err,
			)
			return "", fmt.Errorf("error marshaling notification data: %v", err)
		}
		notificationRecord.Set("data", string(dataJSON))
	}

	err = app.Save(notificationRecord)
	if err != nil {
		app.Logger().Error(
			"error saving notification",
			"template_code", templateCode,
			"recipient_uid", recipientUID,
			"error", err,
		)
		return "", fmt.Errorf("error saving notification: %v", err)
	}

	return notificationRecord.Id, nil
}
