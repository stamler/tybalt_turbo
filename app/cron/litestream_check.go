package cron

import (
	"context"
	"fmt"
	"net/mail"
	"os"
	"sort"
	"strings"
	"sync"
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

type litestreamHealthStatus string

const (
	litestreamHealthStatusHealthy            litestreamHealthStatus = "healthy"
	litestreamHealthStatusMissingCredentials litestreamHealthStatus = "missing_credentials"
	litestreamHealthStatusS3ListFailed       litestreamHealthStatus = "s3_list_failed"
	litestreamHealthStatusNoSnapshots        litestreamHealthStatus = "no_snapshots_found"
	litestreamHealthStatusStaleSnapshot      litestreamHealthStatus = "stale_snapshot"
)

type litestreamConfig struct {
	snapshotPrefix string
	staleThreshold time.Duration
	alertEmail     string
	restart        bool
}

type litestreamS3Config struct {
	bucket    string
	region    string
	accessKey string
	secretKey string
	endpoint  string
}

type litestreamSnapshotInfo struct {
	newestTime time.Time
	newestKey  string
}

type litestreamHealthResult struct {
	status         litestreamHealthStatus
	fingerprint    string
	latestKey      string
	age            time.Duration
	threshold      time.Duration
	missingEnv     []string
	snapshotPrefix string
	err            error
}

type litestreamAlertTracker struct {
	mu                  sync.Mutex
	lastSentFingerprint string
}

var (
	litestreamNow                = time.Now
	litestreamListNewestSnapshot = defaultLitestreamListNewestSnapshot
	litestreamExit               = os.Exit
	litestreamAlerts             litestreamAlertTracker
)

// getLitestreamConfig reads configurable values from app_config where
// key="litestream". Missing or invalid values fall back to defaults.
//
// Supported JSON properties:
//   - snapshot_prefix (string): S3 prefix to check, default "litestream/0009/"
//   - stale_threshold_minutes (number): age in minutes before alerting, default 35
//   - alert_email (string): email address to notify when replication is unhealthy
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
	lsCfg := getLitestreamConfig(app)
	s3Cfg := getLitestreamS3ConfigFromEnv()
	now := litestreamNow()
	result := evaluateLitestreamHealthFromConfig(lsCfg, s3Cfg, now)

	logLitestreamHealth(app, result)

	if result.isHealthy() {
		litestreamAlerts.Reset()
		return
	}

	maybeSendLitestreamAlert(app, lsCfg, result)

	if result.status == litestreamHealthStatusStaleSnapshot && lsCfg.restart {
		app.Logger().Error("litestream replication stale: restarting process")
		litestreamExit(1)
		return
	}
}

func getLitestreamS3ConfigFromEnv() litestreamS3Config {
	region := os.Getenv("LITESTREAM_REGION")
	if region == "" {
		region = "ca-central-1"
	}

	return litestreamS3Config{
		bucket:    os.Getenv("LITESTREAM_BUCKET"),
		region:    region,
		accessKey: os.Getenv("LITESTREAM_ACCESS_KEY_ID"),
		secretKey: os.Getenv("LITESTREAM_SECRET_ACCESS_KEY"),
		endpoint:  os.Getenv("LITESTREAM_ENDPOINT"),
	}
}

func (cfg litestreamS3Config) missingRequiredEnv() []string {
	var missing []string

	if cfg.bucket == "" {
		missing = append(missing, "LITESTREAM_BUCKET")
	}
	if cfg.accessKey == "" {
		missing = append(missing, "LITESTREAM_ACCESS_KEY_ID")
	}
	if cfg.secretKey == "" {
		missing = append(missing, "LITESTREAM_SECRET_ACCESS_KEY")
	}

	sort.Strings(missing)
	return missing
}

func evaluateLitestreamHealthFromConfig(lsCfg litestreamConfig, s3Cfg litestreamS3Config, now time.Time) litestreamHealthResult {
	missing := s3Cfg.missingRequiredEnv()
	if len(missing) > 0 {
		return litestreamHealthResult{
			status:         litestreamHealthStatusMissingCredentials,
			fingerprint:    "missing_credentials:" + strings.Join(missing, ","),
			missingEnv:     missing,
			snapshotPrefix: lsCfg.snapshotPrefix,
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	snapshot, err := litestreamListNewestSnapshot(ctx, lsCfg, s3Cfg)
	if err != nil {
		return litestreamHealthResult{
			status:         litestreamHealthStatusS3ListFailed,
			fingerprint:    "s3_list_failed",
			err:            err,
			snapshotPrefix: lsCfg.snapshotPrefix,
		}
	}

	if snapshot.newestTime.IsZero() {
		return litestreamHealthResult{
			status:         litestreamHealthStatusNoSnapshots,
			fingerprint:    "no_snapshots:" + lsCfg.snapshotPrefix,
			snapshotPrefix: lsCfg.snapshotPrefix,
		}
	}

	age := now.Sub(snapshot.newestTime)
	if age <= lsCfg.staleThreshold {
		return litestreamHealthResult{
			status: litestreamHealthStatusHealthy,
		}
	}

	return litestreamHealthResult{
		status:         litestreamHealthStatusStaleSnapshot,
		fingerprint:    "stale_snapshot:" + lsCfg.snapshotPrefix,
		latestKey:      snapshot.newestKey,
		age:            age,
		threshold:      lsCfg.staleThreshold,
		snapshotPrefix: lsCfg.snapshotPrefix,
	}
}

func defaultLitestreamListNewestSnapshot(ctx context.Context, lsCfg litestreamConfig, s3Cfg litestreamS3Config) (litestreamSnapshotInfo, error) {
	cfg := aws.Config{
		Region:      s3Cfg.region,
		Credentials: credentials.NewStaticCredentialsProvider(s3Cfg.accessKey, s3Cfg.secretKey, ""),
	}

	opts := []func(*s3.Options){}
	if s3Cfg.endpoint != "" && strings.HasPrefix(s3Cfg.endpoint, "http") {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s3Cfg.endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(cfg, opts...)
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3Cfg.bucket),
		Prefix: aws.String(lsCfg.snapshotPrefix),
	})

	var snapshot litestreamSnapshotInfo
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return litestreamSnapshotInfo{}, err
		}
		for _, obj := range page.Contents {
			if obj.LastModified != nil && obj.LastModified.After(snapshot.newestTime) {
				snapshot.newestTime = *obj.LastModified
				snapshot.newestKey = aws.ToString(obj.Key)
			}
		}
	}

	return snapshot, nil
}

func (r litestreamHealthResult) isHealthy() bool {
	return r.status == litestreamHealthStatusHealthy
}

func logLitestreamHealth(app core.App, result litestreamHealthResult) {
	switch result.status {
	case litestreamHealthStatusHealthy:
		return
	case litestreamHealthStatusMissingCredentials:
		app.Logger().Warn(
			"litestream replication check skipped: missing S3 credentials",
			"missing_env", strings.Join(result.missingEnv, ", "),
		)
	case litestreamHealthStatusS3ListFailed:
		app.Logger().Error(
			"litestream replication check: failed to list S3 objects",
			"error", result.err,
			"prefix", result.snapshotPrefix,
		)
	case litestreamHealthStatusNoSnapshots:
		app.Logger().Error(
			"litestream replication check: no snapshots found in S3",
			"prefix", result.snapshotPrefix,
		)
	case litestreamHealthStatusStaleSnapshot:
		app.Logger().Error(
			"litestream replication stale",
			"latest_snapshot", result.latestKey,
			"age_minutes", result.age.Minutes(),
			"threshold_minutes", result.threshold.Minutes(),
		)
	}
}

func maybeSendLitestreamAlert(app core.App, lsCfg litestreamConfig, result litestreamHealthResult) {
	if lsCfg.alertEmail == "" {
		return
	}

	if litestreamAlerts.LastSentFingerprint() == result.fingerprint {
		app.Logger().Info(
			"litestream alert suppressed: already notified for current condition",
			"status", result.status,
		)
		return
	}

	if err := sendLitestreamAlert(app, lsCfg.alertEmail, result, lsCfg.restart); err != nil {
		app.Logger().Error("failed to send litestream alert email", "error", err, "status", result.status)
		return
	}

	litestreamAlerts.MarkSent(result.fingerprint)
}

func sendLitestreamAlert(app core.App, to string, result litestreamHealthResult, restartOnStale bool) error {
	subject, body := litestreamAlertMessage(result, restartOnStale)

	message := &mailer.Message{
		From:    mail.Address{Name: app.Settings().Meta.SenderName, Address: app.Settings().Meta.SenderAddress},
		To:      []mail.Address{{Address: to}},
		Subject: subject,
		Text:    body,
	}

	return app.NewMailClient().Send(message)
}

func litestreamAlertMessage(result litestreamHealthResult, restartOnStale bool) (subject string, body string) {
	switch result.status {
	case litestreamHealthStatusMissingCredentials:
		return "Litestream replication check failed: missing credentials", fmt.Sprintf(
			"Litestream replication health checks cannot run because required environment variables are missing.\n\nMissing variables: %s\nSnapshot prefix: %s\n\nThe application will continue running.",
			strings.Join(result.missingEnv, ", "),
			result.snapshotPrefix,
		)
	case litestreamHealthStatusS3ListFailed:
		return "Litestream replication check failed: S3 listing error", fmt.Sprintf(
			"Litestream replication health checks failed while listing snapshots in object storage.\n\nSnapshot prefix: %s\nError: %v\n\nThe application will continue running.",
			result.snapshotPrefix,
			result.err,
		)
	case litestreamHealthStatusNoSnapshots:
		return "Litestream replication check failed: no snapshots found", fmt.Sprintf(
			"Litestream replication health checks did not find any snapshots in object storage.\n\nSnapshot prefix: %s\n\nThe application will continue running.",
			result.snapshotPrefix,
		)
	case litestreamHealthStatusStaleSnapshot:
		outcome := "continue running"
		if restartOnStale {
			outcome = "restart automatically"
		}

		return "Litestream replication stale", fmt.Sprintf(
			"Litestream replication is stale.\n\nLatest snapshot: %s\nAge: %.0f minutes\nThreshold: %.0f minutes\nSnapshot prefix: %s\n\nThe application will %s.",
			result.latestKey,
			result.age.Minutes(),
			result.threshold.Minutes(),
			result.snapshotPrefix,
			outcome,
		)
	default:
		return "Litestream replication health check", "Litestream replication reported an unhealthy state."
	}
}

func (t *litestreamAlertTracker) LastSentFingerprint() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.lastSentFingerprint
}

func (t *litestreamAlertTracker) MarkSent(fingerprint string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastSentFingerprint = fingerprint
}

func (t *litestreamAlertTracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.lastSentFingerprint = ""
}
