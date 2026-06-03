package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"

	"tybalt/utilities"
)

type storedFileHashTarget struct {
	TargetCollection string
	TargetID         string
	Filename         string
	StoragePath      string
	StoredHash       string
	Updated          string
	HashField        string
}

type storedFileHashAudit struct {
	Target         storedFileHashTarget
	CalculatedHash string
	Matches        bool
}

type storedFileHashReplace struct {
	Audit        storedFileHashAudit
	PreviousHash string
	NewHash      string
	Replaced     bool
	Noop         bool
}

type storedFileHashMessages struct {
	EmptyStoragePath string
	OpenFilesystem   string
	FileNotFound     string
	HashFailed       string
	TargetChanged    string
	UpdatedChanged   string
	UniqueConflict   string
}

type storedFileHashHTTPError struct {
	status  int
	message string
	err     error
}

func (e *storedFileHashHTTPError) Error() string {
	if e.err == nil {
		return e.message
	}
	return fmt.Sprintf("%s: %v", e.message, e.err)
}

func requireAdminForStoredFileHashRepair(app core.App, e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.Error(http.StatusUnauthorized, "unauthorized", nil)
	}
	hasAdmin, err := utilities.HasClaim(app, e.Auth, "admin")
	if err != nil {
		return e.Error(http.StatusInternalServerError, "error checking admin claim", err)
	}
	if !hasAdmin {
		return e.Error(http.StatusForbidden, "admin claim required", nil)
	}
	return nil
}

func auditStoredFileHash(app core.App, resolve func(core.App) (storedFileHashTarget, error), messages storedFileHashMessages) (storedFileHashAudit, error) {
	target, err := resolve(app)
	if err != nil {
		return storedFileHashAudit{}, err
	}
	calculatedHash, err := calculateStoredFileSHA256(app, target.StoragePath, messages)
	if err != nil {
		return storedFileHashAudit{}, err
	}
	return storedFileHashAudit{
		Target:         target,
		CalculatedHash: calculatedHash,
		Matches:        strings.EqualFold(strings.TrimSpace(target.StoredHash), calculatedHash),
	}, nil
}

func replaceStoredFileHash(app core.App, expectedUpdated string, resolve func(core.App) (storedFileHashTarget, error), messages storedFileHashMessages) (storedFileHashReplace, error) {
	audit, err := auditStoredFileHash(app, resolve, messages)
	if err != nil {
		return storedFileHashReplace{}, err
	}

	response := storedFileHashReplace{
		Audit:        audit,
		PreviousHash: audit.Target.StoredHash,
		NewHash:      audit.CalculatedHash,
	}

	var updated string
	err = app.RunInTransaction(func(txApp core.App) error {
		freshTarget, err := resolve(txApp)
		if err != nil {
			return err
		}
		if freshTarget.TargetCollection != audit.Target.TargetCollection ||
			freshTarget.TargetID != audit.Target.TargetID ||
			freshTarget.StoragePath != audit.Target.StoragePath ||
			freshTarget.HashField != audit.Target.HashField {
			return &storedFileHashHTTPError{status: http.StatusConflict, message: messages.TargetChanged}
		}
		if freshTarget.Updated != expectedUpdated {
			return &storedFileHashHTTPError{status: http.StatusConflict, message: messages.UpdatedChanged}
		}
		if strings.EqualFold(strings.TrimSpace(freshTarget.StoredHash), audit.CalculatedHash) {
			updated = freshTarget.Updated
			response.Audit.Target.StoredHash = freshTarget.StoredHash
			response.PreviousHash = freshTarget.StoredHash
			response.Noop = true
			return nil
		}

		newUpdated := types.NowDateTime()
		result, err := txApp.DB().NewQuery(fmt.Sprintf(`
			UPDATE %s
			SET %s = {:hash}, updated = {:updated}
			WHERE id = {:id} AND updated = {:expected_updated}
		`, freshTarget.TargetCollection, freshTarget.HashField)).Bind(dbx.Params{
			"hash":             audit.CalculatedHash,
			"updated":          newUpdated,
			"id":               freshTarget.TargetID,
			"expected_updated": expectedUpdated,
		}).Execute()
		if err != nil {
			if isUniqueConstraintError(err) {
				return &storedFileHashHTTPError{status: http.StatusConflict, message: messages.UniqueConflict, err: err}
			}
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return &storedFileHashHTTPError{status: http.StatusConflict, message: messages.UpdatedChanged}
		}

		updated = newUpdated.String()
		response.Replaced = true
		return nil
	})
	if err != nil {
		return storedFileHashReplace{}, err
	}

	response.Audit.Target.Updated = updated
	response.Audit.Target.StoredHash = audit.CalculatedHash
	response.Audit.Matches = true
	return response, nil
}

func storedFileHashUpdatedString(app core.App, collectionName string, recordID string) (string, error) {
	var updated string
	if err := app.DB().NewQuery("SELECT updated FROM " + collectionName + " WHERE id = {:id}").
		Bind(dbx.Params{"id": recordID}).Row(&updated); err != nil {
		return "", err
	}
	return updated, nil
}

func calculateStoredFileSHA256(app core.App, storagePath string, messages storedFileHashMessages) (string, error) {
	if strings.TrimSpace(storagePath) == "" {
		return "", &storedFileHashHTTPError{status: http.StatusNotFound, message: messages.EmptyStoragePath}
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", &storedFileHashHTTPError{status: http.StatusInternalServerError, message: messages.OpenFilesystem, err: err}
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(storagePath)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, filesystem.ErrNotFound) {
			status = http.StatusNotFound
		}
		return "", &storedFileHashHTTPError{status: status, message: messages.FileNotFound, err: err}
	}
	defer reader.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", &storedFileHashHTTPError{status: http.StatusInternalServerError, message: messages.HashFailed, err: err}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
