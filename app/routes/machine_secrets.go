package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
	"tybalt/utilities"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/pocketbase/pocketbase/core"
)

// Alphanumeric alphabet for salt (matches schema pattern ^[a-zA-Z0-9]{16,128}$)
const alphanumericAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Valid roles for machine_secrets
var validRoles = map[string]bool{
	"legacy_writeback": true,
}

type CreateMachineSecretRequest struct {
	Days int    `json:"days"`
	Role string `json:"role"`
}

type CreateMachineSecretResponse struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
	Expiry string `json:"expiry"`
	Role   string `json:"role"`
}

type MachineSecretListItem struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	Expiry string `json:"expiry"`
}

func createMachineSecretHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		// Check for admin claim
		hasAdminClaim, err := utilities.HasClaim(app, authRecord, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking admin claim", err)
		}
		if !hasAdminClaim {
			return e.Error(http.StatusForbidden, "admin claim required", nil)
		}

		// Parse request body
		var req CreateMachineSecretRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return e.Error(http.StatusBadRequest, "invalid JSON body", err)
		}

		// Validate days
		if req.Days <= 0 {
			return e.Error(http.StatusBadRequest, "days must be a positive integer", nil)
		}

		// Validate role
		if !validRoles[req.Role] {
			return e.Error(http.StatusBadRequest, "invalid role", nil)
		}

		// Generate secret (32 chars using default nanoid alphabet)
		secret, err := gonanoid.New(32)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to generate secret", err)
		}

		// Generate salt (21 chars using alphanumeric-only alphabet)
		salt, err := gonanoid.Generate(alphanumericAlphabet, 21)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to generate salt", err)
		}

		// Hash salt + secret using SHA256
		h := sha256.New()
		h.Write([]byte(salt + secret))
		hashBytes := h.Sum(nil)
		hashHex := hex.EncodeToString(hashBytes)

		// Calculate expiry date
		expiry := time.Now().UTC().AddDate(0, 0, req.Days)

		// Create machine_secrets record
		collection, err := app.FindCollectionByNameOrId("machine_secrets")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to find machine_secrets collection", err)
		}

		record := core.NewRecord(collection)
		record.Set("sha256_hash", hashHex)
		record.Set("salt", salt)
		record.Set("role", req.Role)
		record.Set("expiry", expiry.Format("2006-01-02 15:04:05"))

		if err := app.Save(record); err != nil {
			return e.Error(http.StatusInternalServerError, "failed to save machine secret", err)
		}

		// Return response with unhashed secret
		return e.JSON(http.StatusCreated, CreateMachineSecretResponse{
			ID:     record.Id,
			Secret: secret,
			Expiry: expiry.Format(time.RFC3339),
			Role:   req.Role,
		})
	}
}

func listMachineSecretsHandler(app core.App) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		authRecord := e.Auth
		if authRecord == nil {
			return e.Error(http.StatusUnauthorized, "unauthorized", nil)
		}

		// Check for admin claim
		hasAdminClaim, err := utilities.HasClaim(app, authRecord, "admin")
		if err != nil {
			return e.Error(http.StatusInternalServerError, "error checking admin claim", err)
		}
		if !hasAdminClaim {
			return e.Error(http.StatusForbidden, "admin claim required", nil)
		}

		// Fetch all machine_secrets records
		records, err := app.FindRecordsByFilter(
			"machine_secrets",
			"",         // no filter - get all
			"-created", // sort by created descending (newest first)
			0,          // no limit
			0,          // no offset
		)
		if err != nil {
			return e.Error(http.StatusInternalServerError, "failed to fetch machine secrets", err)
		}

		// Map to response items (only expose id, role, expiry - never hash or salt)
		items := make([]MachineSecretListItem, len(records))
		for i, record := range records {
			items[i] = MachineSecretListItem{
				ID:     record.Id,
				Role:   record.GetString("role"),
				Expiry: record.GetString("expiry"),
			}
		}

		return e.JSON(http.StatusOK, items)
	}
}
