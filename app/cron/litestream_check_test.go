package cron

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
)

func TestCheckLitestreamReplication_MissingCredentialsAlertsOnceUntilHealthy(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	setLitestreamTestSender(app)
	upsertLitestreamConfigRawValue(t, app, `{"alert_email":"ops@example.com"}`)
	resetLitestreamTestGlobals(t)

	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	litestreamNow = func() time.Time { return now }

	t.Setenv("LITESTREAM_BUCKET", "")
	t.Setenv("LITESTREAM_ACCESS_KEY_ID", "")
	t.Setenv("LITESTREAM_SECRET_ACCESS_KEY", "")

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected 1 alert email for missing credentials, got %d", got)
	}
	if subject := app.TestMailer.LastMessage().Subject; !strings.Contains(subject, "missing credentials") {
		t.Fatalf("expected missing credentials subject, got %q", subject)
	}

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected duplicate missing-credentials alert to be suppressed, got %d emails", got)
	}

	t.Setenv("LITESTREAM_BUCKET", "tybalt-backups")
	t.Setenv("LITESTREAM_ACCESS_KEY_ID", "access")
	t.Setenv("LITESTREAM_SECRET_ACCESS_KEY", "secret")
	litestreamListNewestSnapshot = func(_ context.Context, _ litestreamConfig, _ litestreamS3Config) (litestreamSnapshotInfo, error) {
		return litestreamSnapshotInfo{
			newestTime: now.Add(-5 * time.Minute),
			newestKey:  "litestream/0009/healthy.ltx",
		}, nil
	}

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected healthy reset to be silent, got %d emails", got)
	}

	t.Setenv("LITESTREAM_BUCKET", "")
	t.Setenv("LITESTREAM_ACCESS_KEY_ID", "")
	t.Setenv("LITESTREAM_SECRET_ACCESS_KEY", "")

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 2 {
		t.Fatalf("expected a new alert after returning from healthy state, got %d emails", got)
	}
}

func TestCheckLitestreamReplication_AlertEmailCanBeEnabledDuringActiveIssue(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	setLitestreamTestSender(app)
	resetLitestreamTestGlobals(t)

	t.Setenv("LITESTREAM_BUCKET", "")
	t.Setenv("LITESTREAM_ACCESS_KEY_ID", "")
	t.Setenv("LITESTREAM_SECRET_ACCESS_KEY", "")

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 0 {
		t.Fatalf("expected no alert email when alert_email is unset, got %d", got)
	}

	upsertLitestreamConfigRawValue(t, app, `{"alert_email":"ops@example.com"}`)

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected alert once alert_email is configured, got %d emails", got)
	}
}

func TestCheckLitestreamReplication_S3ListFailureAlertsOnce(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	setLitestreamTestSender(app)
	upsertLitestreamConfigRawValue(t, app, `{"alert_email":"ops@example.com"}`)
	resetLitestreamTestGlobals(t)
	setLitestreamRequiredEnv(t)

	litestreamListNewestSnapshot = func(_ context.Context, _ litestreamConfig, _ litestreamS3Config) (litestreamSnapshotInfo, error) {
		return litestreamSnapshotInfo{}, errors.New("AccessDenied: invalid access key")
	}

	checkLitestreamReplication(app)
	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected one deduped S3 list failure alert, got %d emails", got)
	}

	body := app.TestMailer.LastMessage().Text
	if !strings.Contains(body, "AccessDenied") {
		t.Fatalf("expected S3 error details in email body, got %q", body)
	}
}

func TestCheckLitestreamReplication_NoSnapshotsFoundAlerts(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	setLitestreamTestSender(app)
	upsertLitestreamConfigRawValue(t, app, `{"alert_email":"ops@example.com"}`)
	resetLitestreamTestGlobals(t)
	setLitestreamRequiredEnv(t)

	litestreamListNewestSnapshot = func(_ context.Context, _ litestreamConfig, _ litestreamS3Config) (litestreamSnapshotInfo, error) {
		return litestreamSnapshotInfo{}, nil
	}

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected one no-snapshots alert, got %d emails", got)
	}

	subject := app.TestMailer.LastMessage().Subject
	if !strings.Contains(subject, "no snapshots found") {
		t.Fatalf("expected no-snapshots subject, got %q", subject)
	}
}

func TestCheckLitestreamReplication_StaleSnapshotAlertsAndRestarts(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	setLitestreamTestSender(app)
	upsertLitestreamConfigRawValue(t, app, `{"alert_email":"ops@example.com","restart_on_stale":true}`)
	resetLitestreamTestGlobals(t)
	setLitestreamRequiredEnv(t)

	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	litestreamNow = func() time.Time { return now }
	litestreamListNewestSnapshot = func(_ context.Context, _ litestreamConfig, _ litestreamS3Config) (litestreamSnapshotInfo, error) {
		return litestreamSnapshotInfo{
			newestTime: now.Add(-2 * time.Hour),
			newestKey:  "litestream/0009/stale.ltx",
		}, nil
	}

	exitCode := 0
	litestreamExit = func(code int) {
		exitCode = code
	}

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 1 {
		t.Fatalf("expected one stale-snapshot alert, got %d emails", got)
	}
	if exitCode != 1 {
		t.Fatalf("expected stale snapshot to request exit code 1, got %d", exitCode)
	}

	body := app.TestMailer.LastMessage().Text
	if !strings.Contains(body, "restart automatically") {
		t.Fatalf("expected restart language in stale alert body, got %q", body)
	}
}

func TestCheckLitestreamReplication_NewFailureClassSendsAnotherAlert(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	setLitestreamTestSender(app)
	upsertLitestreamConfigRawValue(t, app, `{"alert_email":"ops@example.com"}`)
	resetLitestreamTestGlobals(t)
	setLitestreamRequiredEnv(t)

	litestreamListNewestSnapshot = func(_ context.Context, _ litestreamConfig, _ litestreamS3Config) (litestreamSnapshotInfo, error) {
		return litestreamSnapshotInfo{}, errors.New("AccessDenied: invalid access key")
	}

	checkLitestreamReplication(app)

	litestreamListNewestSnapshot = func(_ context.Context, _ litestreamConfig, _ litestreamS3Config) (litestreamSnapshotInfo, error) {
		return litestreamSnapshotInfo{}, nil
	}

	checkLitestreamReplication(app)

	if got := len(app.TestMailer.Messages()); got != 2 {
		t.Fatalf("expected a new alert when the unhealthy failure class changes, got %d emails", got)
	}
}

func resetLitestreamTestGlobals(t *testing.T) {
	t.Helper()

	oldNow := litestreamNow
	oldList := litestreamListNewestSnapshot
	oldExit := litestreamExit
	litestreamAlerts.Reset()

	t.Cleanup(func() {
		litestreamNow = oldNow
		litestreamListNewestSnapshot = oldList
		litestreamExit = oldExit
		litestreamAlerts.Reset()
	})
}

func setLitestreamRequiredEnv(t *testing.T) {
	t.Helper()

	t.Setenv("LITESTREAM_BUCKET", "tybalt-backups")
	t.Setenv("LITESTREAM_ACCESS_KEY_ID", "access")
	t.Setenv("LITESTREAM_SECRET_ACCESS_KEY", "secret")
	t.Setenv("LITESTREAM_REGION", "ca-central-1")
}

func setLitestreamTestSender(app core.App) {
	app.Settings().Meta.SenderName = "Tybalt"
	app.Settings().Meta.SenderAddress = "tybalt@example.com"
}

func upsertLitestreamConfigRawValue(t *testing.T, app core.App, rawValue string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("app_config")
	if err != nil {
		t.Fatalf("failed to find app_config collection: %v", err)
	}

	record, err := app.FindFirstRecordByData("app_config", "key", "litestream")
	if err != nil || record == nil {
		record = core.NewRecord(collection)
		record.Set("key", "litestream")
	}

	record.Set("value", rawValue)
	if err := app.Save(record); err != nil {
		t.Fatalf("failed to save litestream config: %v", err)
	}
}
