# Data Migration

## Export data from MySQL to Parquet

```bash
go run tool.go --export
```

## Import data from Parquet to PocketBase

```bash
go run tool.go --import
```

## Migrate attachments from GCS to S3

```bash
go run tool.go --attachments
```
