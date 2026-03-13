package cron

import (
	"context"
	"os"
	"strings"
	"time"

	"tybalt/utilities"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pocketbase/pocketbase/core"
)

const (
	defaultSnapshotPrefix        = "litestream/0009/"
	defaultStaleThresholdMinutes = 35
)

// getLitestreamConfig reads configurable values from app_config where
// key="litestream". Missing or invalid values fall back to defaults.
//
// Supported JSON properties:
//   - snapshot_prefix (string): S3 prefix to check, default "litestream/0009/"
//   - stale_threshold_minutes (number): age in minutes before alerting, default 35
func getLitestreamConfig(app core.App) (prefix string, threshold time.Duration) {
	prefix = defaultSnapshotPrefix
	threshold = time.Duration(defaultStaleThresholdMinutes) * time.Minute

	config, err := utilities.GetConfigValue(app, "litestream")
	if err != nil || config == nil {
		return
	}

	if p, ok := config["snapshot_prefix"].(string); ok && p != "" {
		prefix = p
	}

	if minutes, err := utilities.CoerceFloat64(config["stale_threshold_minutes"]); err == nil && minutes > 0 {
		threshold = time.Duration(minutes) * time.Minute
	}

	return
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

	snapshotPrefix, staleThreshold := getLitestreamConfig(app)

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
		Prefix: aws.String(snapshotPrefix),
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
	if age > staleThreshold {
		app.Logger().Error("litestream replication stale",
			"latest_snapshot", newestKey,
			"age_minutes", age.Minutes(),
			"threshold_minutes", staleThreshold.Minutes(),
		)
	}
}
