package routes

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
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

type attachmentAuditDeleteOrphansResponse struct {
	TargetKey              string                         `json:"target_key"`
	DeletedFiles           int                            `json:"deleted_files"`
	SkippedReferencedFiles int                            `json:"skipped_referenced_files"`
	AlreadyMissingFiles    int                            `json:"already_missing_files"`
	SkippedInvalidFiles    int                            `json:"skipped_invalid_files"`
	FailedFiles            int                            `json:"failed_files"`
	Failures               []attachmentAuditDeleteFailure `json:"failures"`
	RefreshError           string                         `json:"refresh_error"`
	Latest                 *attachmentAuditRunResponse    `json:"latest"`
}

type attachmentAuditDeleteFailure struct {
	StoragePath string `json:"storage_path"`
	Error       string `json:"error"`
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

func createDeleteAttachmentAuditOrphansHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := requireAdminClaimForAttachmentAudit(app, e, "delete orphaned attachment files"); err != nil {
			return err
		}

		target, ok := attachmentAuditTargetByKey(e.Request.PathValue("target"))
		if !ok {
			return e.Error(http.StatusNotFound, "attachment audit target not found", nil)
		}

		if _, loaded := attachmentAuditActiveRuns.LoadOrStore(target.Key, true); loaded {
			return e.Error(http.StatusConflict, "attachment audit target is already running", nil)
		}
		defer attachmentAuditActiveRuns.Delete(target.Key)

		response, err := deleteCachedAttachmentAuditOrphans(app, target, e.Auth.Id)
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, filesystem.ErrNotFound) {
			return e.Error(http.StatusNotFound, "attachment audit orphaned report not found", err)
		}
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to delete orphaned attachment files", err)
		}

		return e.JSON(http.StatusOK, response)
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

// deleteCachedAttachmentAuditOrphans deletes orphaned attachment files using the
// latest cached orphan report as the candidate list, but it does not trust that
// report as authoritative at delete time.
//
// The audit report is only a snapshot. Between the audit run and this cleanup
// action, a user or import may have attached one of those files to a record, a
// file may have already been removed from storage, or storage may contain a row
// whose path is malformed or belongs to a different target. Because the database
// record state and the configured file storage cannot be updated in one shared
// transaction, this function treats every CSV row as a delete candidate that
// must be revalidated immediately before its storage object is removed.
//
// The process is:
//  1. Load the single cached attachment_audit_runs record for the target.
//  2. Read that run's cached orphaned_report CSV from PocketBase storage.
//  3. For each CSV row, validate that the collection, field, and storage path
//     still match the requested audit target and that the path still lives under
//     the target collection's file prefix.
//  4. Parse the path-derived record id and filename from the storage path.
//  5. Look up the current record with that path-derived id. If the record now
//     exists and its audited attachment field references the same filename, skip
//     the delete because the file is no longer orphaned.
//  6. Delete the storage object only after that current-state check passes.
//     Missing files are counted separately, and non-not-found storage errors are
//     reported per file without aborting the rest of the cleanup.
//  7. Run a fresh attachment audit after the delete pass so the cached counts
//     and downloadable CSV reports reflect the post-cleanup storage state.
//
// This still cannot close the tiny race where a file becomes referenced after
// the revalidation query but before the storage delete completes. The important
// safety property is that we never delete solely because an older cached report
// said a file was orphaned; we always re-check current database state first.
func deleteCachedAttachmentAuditOrphans(app core.App, target attachmentAuditTarget, requestedBy string) (attachmentAuditDeleteOrphansResponse, error) {
	record, err := findAttachmentAuditRun(app, target.Key)
	if err != nil {
		return attachmentAuditDeleteOrphansResponse{}, err
	}

	rows, err := readCachedAttachmentAuditOrphanedRows(app, record)
	if err != nil {
		return attachmentAuditDeleteOrphansResponse{}, err
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return attachmentAuditDeleteOrphansResponse{}, err
	}
	defer fsys.Close()

	response := attachmentAuditDeleteOrphansResponse{
		TargetKey: target.Key,
		Failures:  []attachmentAuditDeleteFailure{},
	}
	for _, row := range rows {
		status, err := deleteCachedAttachmentAuditOrphan(app, fsys, target, row)
		if err != nil {
			response.FailedFiles++
			response.Failures = append(response.Failures, attachmentAuditDeleteFailure{
				StoragePath: row.StoragePath,
				Error:       err.Error(),
			})
			continue
		}

		switch status {
		case "deleted":
			response.DeletedFiles++
		case "referenced":
			response.SkippedReferencedFiles++
		case "missing":
			response.AlreadyMissingFiles++
		case "invalid":
			response.SkippedInvalidFiles++
		}
	}

	if err := completeAttachmentAuditRun(app, target, requestedBy); err != nil {
		response.RefreshError = err.Error()
		return response, nil
	}

	updated, err := findAttachmentAuditRun(app, target.Key)
	if err != nil {
		response.RefreshError = err.Error()
		return response, nil
	}
	response.Latest = attachmentAuditRunResponseFromRecord(updated)
	return response, nil
}

func readCachedAttachmentAuditOrphanedRows(app core.App, record *core.Record) ([]attachmentAuditOrphanedRow, error) {
	filename := record.GetString("orphaned_report")
	if filename == "" {
		return nil, filesystem.ErrNotFound
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(record.BaseFilesPath() + "/" + filename)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	csvReader := csv.NewReader(reader)
	if _, err := csvReader.Read(); err != nil {
		if errors.Is(err, io.EOF) {
			return []attachmentAuditOrphanedRow{}, nil
		}
		return nil, err
	}

	rows := []attachmentAuditOrphanedRow{}
	for {
		fields, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(fields) < 5 {
			rows = append(rows, attachmentAuditOrphanedRow{})
			continue
		}
		rows = append(rows, attachmentAuditOrphanedRow{
			Collection:  fields[0],
			Field:       fields[1],
			StoragePath: fields[2],
			RecordID:    fields[3],
			Filename:    fields[4],
		})
	}
	return rows, nil
}

func deleteCachedAttachmentAuditOrphan(app core.App, fsys *filesystem.System, target attachmentAuditTarget, row attachmentAuditOrphanedRow) (string, error) {
	if row.Collection != target.Collection || row.Field != target.Field || row.StoragePath == "" {
		return "invalid", nil
	}

	collection, err := app.FindCollectionByNameOrId(target.Collection)
	if err != nil {
		return "", err
	}
	prefix := collection.BaseFilesPath() + "/"
	if !strings.HasPrefix(row.StoragePath, prefix) {
		return "invalid", nil
	}

	recordID, filename := splitAttachmentAuditStoragePath(collection.BaseFilesPath(), row.StoragePath)
	if recordID == "" || filename == "" {
		return "invalid", nil
	}

	record, err := app.FindFirstRecordByFilter(
		target.Collection,
		"id = {:recordID}",
		dbx.Params{"recordID": recordID},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if record != nil && strings.TrimSpace(record.GetString(target.Field)) == filename {
		return "referenced", nil
	}

	if err := fsys.Delete(row.StoragePath); err != nil {
		if errors.Is(err, filesystem.ErrNotFound) {
			return "missing", nil
		}
		return "", err
	}
	return "deleted", nil
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
	if err := writer.Write([]string{"collection", "field", "storage_path", "record_id_from_path", "filename"}); err != nil {
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
