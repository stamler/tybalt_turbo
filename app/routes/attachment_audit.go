package routes

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	attachmentAuditRunsCollection = "attachment_audit_runs"
	attachmentAuditStatusRunning  = "running"
	attachmentAuditStatusComplete = "completed"
	attachmentAuditStatusFailed   = "failed"
)

type attachmentAuditTarget struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Collection string `json:"collection"`
	Field      string `json:"field"`
}

type attachmentAuditRunResponse struct {
	TargetKey         string `json:"target_key"`
	Label             string `json:"label"`
	Collection        string `json:"collection"`
	Field             string `json:"field"`
	Status            string `json:"status"`
	RequestedBy       string `json:"requested_by"`
	StartedAt         string `json:"started_at"`
	FinishedAt        string `json:"finished_at"`
	TotalRecords      int    `json:"total_records"`
	ReferencedRecords int    `json:"referenced_records"`
	MatchingRecords   int    `json:"matching_records"`
	MissingRecords    int    `json:"missing_records"`
	OrphanedFiles     int    `json:"orphaned_files"`
	Error             string `json:"error"`
	HasMissingReport  bool   `json:"has_missing_report"`
	HasOrphanedReport bool   `json:"has_orphaned_report"`
}

type attachmentAuditTargetResponse struct {
	attachmentAuditTarget
	Latest *attachmentAuditRunResponse `json:"latest"`
}

type attachmentAuditResult struct {
	Target            attachmentAuditTarget
	TotalRecords      int
	ReferencedRecords int
	MatchingRecords   int
	MissingRecords    int
	OrphanedFiles     int
	MissingRows       []attachmentAuditMissingRow
	OrphanedRows      []attachmentAuditOrphanedRow
}

type attachmentAuditReference struct {
	RecordID    string
	Filename    string
	StoragePath string
}

type attachmentAuditMissingRow struct {
	Collection  string
	Field       string
	RecordID    string
	Filename    string
	StoragePath string
}

type attachmentAuditOrphanedRow struct {
	Collection  string
	Field       string
	StoragePath string
	RecordID    string
	Filename    string
}

var (
	attachmentAuditTargets = []attachmentAuditTarget{
		{Key: "expenses_attachment", Label: "Expenses (legacy attachments)", Collection: "expenses", Field: "attachment"},
		{Key: "expense_documents_attachment", Label: "Expense Documents", Collection: "expense_documents", Field: "attachment"},
		{Key: "purchase_orders_attachment", Label: "Purchase Orders", Collection: "purchase_orders", Field: "attachment"},
	}
	attachmentAuditActiveRuns sync.Map
)

func requireAdminClaimForAttachmentAudit(app core.App, e *core.RequestEvent, action string) error {
	hasAdminClaim, err := utilities.HasClaim(app, e.Auth, "admin")
	if err != nil {
		return e.Error(http.StatusInternalServerError, "failed to check admin claim", err)
	}
	if !hasAdminClaim {
		return e.Error(http.StatusForbidden, "you do not have permission to "+action, nil)
	}
	return nil
}

func createListAttachmentAuditTargetsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForAttachmentAudit(app, e, "view attachment audits"); err != nil {
			return err
		}

		responses := make([]attachmentAuditTargetResponse, 0, len(attachmentAuditTargets))
		for _, target := range attachmentAuditTargets {
			response := attachmentAuditTargetResponse{attachmentAuditTarget: target}
			record, err := findAttachmentAuditRun(app, target.Key)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return e.Error(http.StatusInternalServerError, "failed to load attachment audit run", err)
			}
			if record != nil {
				response.Latest = attachmentAuditRunResponseFromRecord(record)
			}
			responses = append(responses, response)
		}

		return e.JSON(http.StatusOK, responses)
	}
}

func createGetAttachmentAuditRunHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForAttachmentAudit(app, e, "view attachment audit status"); err != nil {
			return err
		}

		target, ok := attachmentAuditTargetByKey(e.Request.PathValue("target"))
		if !ok {
			return e.Error(http.StatusNotFound, "attachment audit target not found", nil)
		}

		record, err := findAttachmentAuditRun(app, target.Key)
		if errors.Is(err, sql.ErrNoRows) {
			return e.JSON(http.StatusOK, map[string]any{"target_key": target.Key, "status": ""})
		}
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load attachment audit run", err)
		}

		return e.JSON(http.StatusOK, attachmentAuditRunResponseFromRecord(record))
	}
}

func createRefreshAttachmentAuditHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForAttachmentAudit(app, e, "refresh attachment audits"); err != nil {
			return err
		}

		target, ok := attachmentAuditTargetByKey(e.Request.PathValue("target"))
		if !ok {
			return e.Error(http.StatusNotFound, "attachment audit target not found", nil)
		}

		if _, loaded := attachmentAuditActiveRuns.LoadOrStore(target.Key, true); loaded {
			record, err := findAttachmentAuditRun(app, target.Key)
			if errors.Is(err, sql.ErrNoRows) {
				return e.JSON(http.StatusAccepted, attachmentAuditRunningResponseFromTarget(target, e.Auth.Id))
			}
			if err != nil {
				return e.Error(http.StatusInternalServerError, "failed to load running attachment audit", err)
			}
			return e.JSON(http.StatusAccepted, attachmentAuditRunResponseFromRecord(record))
		}

		requestedBy := e.Auth.Id
		record, err := markAttachmentAuditRunStarted(app, target, requestedBy)
		if err != nil {
			attachmentAuditActiveRuns.Delete(target.Key)
			return e.Error(http.StatusInternalServerError, "failed to start attachment audit", err)
		}

		go func() {
			defer attachmentAuditActiveRuns.Delete(target.Key)
			if err := completeAttachmentAuditRun(app, target, requestedBy); err != nil {
				if failErr := markAttachmentAuditRunFailed(app, target, requestedBy, err); failErr != nil {
					app.Logger().Error("failed to mark attachment audit run failed", "target", target.Key, "error", failErr)
				}
			}
		}()

		return e.JSON(http.StatusAccepted, attachmentAuditRunResponseFromRecord(record))
	}
}

func createDownloadAttachmentAuditReportHandler(app core.App, reportField string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForAttachmentAudit(app, e, "download attachment audit reports"); err != nil {
			return err
		}

		target, ok := attachmentAuditTargetByKey(e.Request.PathValue("target"))
		if !ok {
			return e.Error(http.StatusNotFound, "attachment audit target not found", nil)
		}

		record, err := findAttachmentAuditRun(app, target.Key)
		if errors.Is(err, sql.ErrNoRows) {
			return e.Error(http.StatusNotFound, "attachment audit report not found", nil)
		}
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to load attachment audit run", err)
		}

		filename := record.GetString(reportField)
		if filename == "" {
			return e.Error(http.StatusNotFound, "attachment audit report not found", nil)
		}

		fsys, err := app.NewFilesystem()
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to open filesystem", err)
		}
		defer fsys.Close()

		reader, err := fsys.GetReader(record.BaseFilesPath() + "/" + filename)
		if err != nil {
			return e.Error(http.StatusNotFound, "attachment audit report not found", err)
		}
		defer reader.Close()

		reportName := strings.TrimSuffix(reportField, "_report")
		downloadName := fmt.Sprintf("%s_%s.csv", target.Key, reportName)
		e.Response.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, downloadName))
		return e.Stream(http.StatusOK, "text/csv; charset=utf-8", reader)
	}
}

func attachmentAuditTargetByKey(key string) (attachmentAuditTarget, bool) {
	for _, target := range attachmentAuditTargets {
		if target.Key == key {
			return target, true
		}
	}
	return attachmentAuditTarget{}, false
}

func findAttachmentAuditRun(app core.App, targetKey string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(
		attachmentAuditRunsCollection,
		"target_key = {:targetKey}",
		dbx.Params{"targetKey": targetKey},
	)
}

func getOrCreateAttachmentAuditRun(app core.App, target attachmentAuditTarget) (*core.Record, error) {
	record, err := findAttachmentAuditRun(app, target.Key)
	if err == nil {
		return record, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	collection, err := app.FindCollectionByNameOrId(attachmentAuditRunsCollection)
	if err != nil {
		return nil, err
	}
	record = core.NewRecord(collection)
	record.Set("target_key", target.Key)
	record.Set("label", target.Label)
	record.Set("collection_name", target.Collection)
	record.Set("field_name", target.Field)
	record.Set("status", attachmentAuditStatusRunning)
	return record, nil
}

func markAttachmentAuditRunStarted(app core.App, target attachmentAuditTarget, requestedBy string) (*core.Record, error) {
	record, err := getOrCreateAttachmentAuditRun(app, target)
	if err != nil {
		return nil, err
	}

	record.Set("label", target.Label)
	record.Set("collection_name", target.Collection)
	record.Set("field_name", target.Field)
	record.Set("status", attachmentAuditStatusRunning)
	record.Set("requested_by", requestedBy)
	record.Set("started_at", types.NowDateTime())
	record.Set("finished_at", "")
	record.Set("error", "")
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func markAttachmentAuditRunFailed(app core.App, target attachmentAuditTarget, requestedBy string, runErr error) error {
	record, err := getOrCreateAttachmentAuditRun(app, target)
	if err != nil {
		return err
	}
	record.Set("label", target.Label)
	record.Set("collection_name", target.Collection)
	record.Set("field_name", target.Field)
	record.Set("status", attachmentAuditStatusFailed)
	record.Set("requested_by", requestedBy)
	record.Set("finished_at", types.NowDateTime())
	record.Set("error", runErr.Error())
	return app.Save(record)
}

func completeAttachmentAuditRun(app core.App, target attachmentAuditTarget, requestedBy string) error {
	result, err := runAttachmentAudit(app, target)
	if err != nil {
		return err
	}

	missingCSV, err := buildAttachmentAuditMissingCSV(result.MissingRows)
	if err != nil {
		return err
	}
	orphanedCSV, err := buildAttachmentAuditOrphanedCSV(result.OrphanedRows)
	if err != nil {
		return err
	}
	missingFile, err := filesystem.NewFileFromBytes(missingCSV, target.Key+"_missing.csv")
	if err != nil {
		return err
	}
	orphanedFile, err := filesystem.NewFileFromBytes(orphanedCSV, target.Key+"_orphaned.csv")
	if err != nil {
		return err
	}

	record, err := getOrCreateAttachmentAuditRun(app, target)
	if err != nil {
		return err
	}
	record.Set("label", target.Label)
	record.Set("collection_name", target.Collection)
	record.Set("field_name", target.Field)
	record.Set("status", attachmentAuditStatusComplete)
	record.Set("requested_by", requestedBy)
	record.Set("finished_at", types.NowDateTime())
	record.Set("total_records", result.TotalRecords)
	record.Set("referenced_records", result.ReferencedRecords)
	record.Set("matching_records", result.MatchingRecords)
	record.Set("missing_records", result.MissingRecords)
	record.Set("orphaned_files", result.OrphanedFiles)
	record.Set("error", "")
	record.Set("missing_report", missingFile)
	record.Set("orphaned_report", orphanedFile)
	return app.Save(record)
}

func runAttachmentAudit(app core.App, target attachmentAuditTarget) (attachmentAuditResult, error) {
	collection, err := app.FindCollectionByNameOrId(target.Collection)
	if err != nil {
		return attachmentAuditResult{}, err
	}
	if collection.Fields.GetByName(target.Field) == nil {
		return attachmentAuditResult{}, fmt.Errorf("field %s.%s not found", target.Collection, target.Field)
	}

	records, err := app.FindRecordsByFilter(target.Collection, "1 = 1", "id", 0, 0)
	if err != nil {
		return attachmentAuditResult{}, err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return attachmentAuditResult{}, err
	}
	defer fsys.Close()

	result := attachmentAuditResult{
		Target:       target,
		TotalRecords: len(records),
		MissingRows:  []attachmentAuditMissingRow{},
		OrphanedRows: []attachmentAuditOrphanedRow{},
	}
	expectedPaths := map[string]attachmentAuditReference{}
	storedPaths := map[string]struct{}{}

	objects, err := fsys.List(collection.BaseFilesPath() + "/")
	if err != nil {
		return attachmentAuditResult{}, err
	}
	for _, object := range objects {
		if strings.Contains(object.Key, "/thumbs_") {
			continue
		}
		storedPaths[object.Key] = struct{}{}
	}

	for _, record := range records {
		filename := strings.TrimSpace(record.GetString(target.Field))
		if filename == "" {
			continue
		}

		storagePath := collection.BaseFilesPath() + "/" + record.Id + "/" + filename
		ref := attachmentAuditReference{
			RecordID:    record.Id,
			Filename:    filename,
			StoragePath: storagePath,
		}
		expectedPaths[storagePath] = ref
		result.ReferencedRecords++

		if _, ok := storedPaths[storagePath]; ok {
			result.MatchingRecords++
		} else {
			result.MissingRecords++
			result.MissingRows = append(result.MissingRows, attachmentAuditMissingRow{
				Collection:  target.Collection,
				Field:       target.Field,
				RecordID:    record.Id,
				Filename:    filename,
				StoragePath: storagePath,
			})
		}
	}

	for storagePath := range storedPaths {
		if _, ok := expectedPaths[storagePath]; ok {
			continue
		}

		recordID, filename := splitAttachmentAuditStoragePath(collection.BaseFilesPath(), storagePath)
		result.OrphanedRows = append(result.OrphanedRows, attachmentAuditOrphanedRow{
			Collection:  target.Collection,
			Field:       target.Field,
			StoragePath: storagePath,
			RecordID:    recordID,
			Filename:    filename,
		})
		result.OrphanedFiles++
	}

	sort.Slice(result.MissingRows, func(i, j int) bool {
		return result.MissingRows[i].StoragePath < result.MissingRows[j].StoragePath
	})
	sort.Slice(result.OrphanedRows, func(i, j int) bool {
		return result.OrphanedRows[i].StoragePath < result.OrphanedRows[j].StoragePath
	})

	return result, nil
}

func splitAttachmentAuditStoragePath(collectionPath string, storagePath string) (string, string) {
	trimmed := strings.TrimPrefix(storagePath, collectionPath+"/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) != 2 {
		return "", path.Base(storagePath)
	}
	return parts[0], path.Base(parts[1])
}

func buildAttachmentAuditMissingCSV(rows []attachmentAuditMissingRow) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	if err := writer.Write([]string{"collection", "field", "record_id", "filename", "storage_path"}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Collection, row.Field, row.RecordID, row.Filename, row.StoragePath}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildAttachmentAuditOrphanedCSV(rows []attachmentAuditOrphanedRow) ([]byte, error) {
	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	if err := writer.Write([]string{"collection", "field", "storage_path", "record_id", "filename"}); err != nil {
		return nil, err
	}
	for _, row := range rows {
		if err := writer.Write([]string{row.Collection, row.Field, row.StoragePath, row.RecordID, row.Filename}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func attachmentAuditRunResponseFromRecord(record *core.Record) *attachmentAuditRunResponse {
	return &attachmentAuditRunResponse{
		TargetKey:         record.GetString("target_key"),
		Label:             record.GetString("label"),
		Collection:        record.GetString("collection_name"),
		Field:             record.GetString("field_name"),
		Status:            record.GetString("status"),
		RequestedBy:       record.GetString("requested_by"),
		StartedAt:         attachmentAuditDateString(record, "started_at"),
		FinishedAt:        attachmentAuditDateString(record, "finished_at"),
		TotalRecords:      int(record.GetFloat("total_records")),
		ReferencedRecords: int(record.GetFloat("referenced_records")),
		MatchingRecords:   int(record.GetFloat("matching_records")),
		MissingRecords:    int(record.GetFloat("missing_records")),
		OrphanedFiles:     int(record.GetFloat("orphaned_files")),
		Error:             record.GetString("error"),
		HasMissingReport:  record.GetString("missing_report") != "",
		HasOrphanedReport: record.GetString("orphaned_report") != "",
	}
}

func attachmentAuditRunningResponseFromTarget(target attachmentAuditTarget, requestedBy string) *attachmentAuditRunResponse {
	return &attachmentAuditRunResponse{
		TargetKey:   target.Key,
		Label:       target.Label,
		Collection:  target.Collection,
		Field:       target.Field,
		Status:      attachmentAuditStatusRunning,
		RequestedBy: requestedBy,
	}
}

func attachmentAuditDateString(record *core.Record, field string) string {
	value := record.GetDateTime(field)
	if value.IsZero() {
		return ""
	}
	return value.Time().UTC().Format(time.RFC3339)
}
