package cron

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	"tybalt/utilities"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

const (
	defaultSnapshotPrefix        = "litestream/0009/"
	defaultStaleThresholdMinutes = 35
)

type litestreamConfig struct {
	snapshotPrefix string
	staleThreshold time.Duration
	alertEmail     string
	restart        bool
}

// getLitestreamConfig reads configurable values from app_config where
// key="litestream". Missing or invalid values fall back to defaults.
//
// Supported JSON properties:
//   - snapshot_prefix (string): S3 prefix to check, default "litestream/0009/"
//   - stale_threshold_minutes (number): age in minutes before alerting, default 35
//   - alert_email (string): email address to notify when replication is stale
//   - restart_on_stale (bool): exit the process after alerting so Fly restarts, default false
func getLitestreamConfig(app core.App) litestreamConfig {
	cfg := litestreamConfig{
		snapshotPrefix: defaultSnapshotPrefix,
		staleThreshold: time.Duration(defaultStaleThresholdMinutes) * time.Minute,
	}

	config, err := utilities.GetConfigValue(app, "litestream")
	if err != nil || config == nil {
		return cfg
	}

	if p, ok := config["snapshot_prefix"].(string); ok && p != "" {
		cfg.snapshotPrefix = p
	}

	if minutes, err := utilities.CoerceFloat64(config["stale_threshold_minutes"]); err == nil && minutes > 0 {
		cfg.staleThreshold = time.Duration(minutes) * time.Minute
	}

	if email, ok := config["alert_email"].(string); ok && email != "" {
		cfg.alertEmail = email
	}

	if restart, ok := config["restart_on_stale"].(bool); ok {
		cfg.restart = restart
	}

	return cfg
}

func checkLitestreamReplication(app core.App) {
	bucket := os.Getenv("LITESTREAM_BUCKET")
	region := os.Getenv("LITESTREAM_REGION")
	accessKey := os.Getenv("LITESTREAM_ACCESS_KEY_ID")
	secretKey := os.Getenv("LITESTREAM_SECRET_ACCESS_KEY")
	endpoint := os.Getenv("LITESTREAM_ENDPOINT")

	if bucket == "" || accessKey == "" || secretKey == "" {
		app.Logger().Warn("litestream replication check skipped: missing S3 credentials")
		return
	}

	if region == "" {
		region = "ca-central-1"
	}

	lsCfg := getLitestreamConfig(app)

	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}

	opts := []func(*s3.Options){}
	if endpoint != "" && strings.HasPrefix(endpoint, "http") {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, opts...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var newestTime time.Time
	var newestKey string

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(lsCfg.snapshotPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			app.Logger().Error("litestream replication check: failed to list S3 objects", "error", err)
			return
		}
		for _, obj := range page.Contents {
			if obj.LastModified != nil && obj.LastModified.After(newestTime) {
				newestTime = *obj.LastModified
				newestKey = *obj.Key
			}
		}
	}

	if newestTime.IsZero() {
		app.Logger().Error("litestream replication check: no snapshots found in S3")
		return
	}

	age := time.Since(newestTime)
	if age <= lsCfg.staleThreshold {
		return
	}

	app.Logger().Error("litestream replication stale",
		"latest_snapshot", newestKey,
		"age_minutes", age.Minutes(),
		"threshold_minutes", lsCfg.staleThreshold.Minutes(),
	)

	if lsCfg.alertEmail != "" {
		sendStaleAlert(app, lsCfg.alertEmail, newestKey, age, lsCfg.staleThreshold)
	}

	if lsCfg.restart {
		app.Logger().Error("litestream replication stale: restarting process")
		os.Exit(1)
	}
}

func sendStaleAlert(app core.App, to string, latestKey string, age, threshold time.Duration) {
	subject := "Litestream replication stale"
	body := fmt.Sprintf(
		"Litestream replication is stale.\n\nLatest snapshot: %s\nAge: %.0f minutes\nThreshold: %.0f minutes\n\nThe application will %s.",
		latestKey,
		age.Minutes(),
		threshold.Minutes(),
		func() string {
			if cfg := getLitestreamConfig(app); cfg.restart {
				return "restart automatically"
			}
			return "continue running"
		}(),
	)

	message := &mailer.Message{
		From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		Text:    body,
	}

	if err := app.NewMailClient().Send(message); err != nil {
		app.Logger().Error("failed to send litestream stale alert email", "error", err)
	}
}
