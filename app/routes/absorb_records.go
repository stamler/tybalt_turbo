package routes

import (
	"fmt"
	"net/http"
	"strings"

	"tybalt/utilities"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// AbsorbRequest defines the structure for the absorb request body
type AbsorbRequest struct {
	IdsToAbsorb []string `json:"ids_to_absorb"`
}

func CreateAbsorbRecordsHandler(app core.App, collectionName string) echo.HandlerFunc {
	// This handler absorbs multiple records into one target record for a given collection.
	// It performs the following actions:
	// 1. Validates the request body contains a list of IDs to absorb
	// 2. Calls the AbsorbRecords function to perform the absorption
	return func(c echo.Context) error {
		targetId := c.PathParam("id")
		var request AbsorbRequest
		if err := c.Bind(&request); err != nil {
			return apis.NewBadRequestError("Invalid request body", nil)
		}

		if len(request.IdsToAbsorb) == 0 {
			return apis.NewBadRequestError("No IDs provided to absorb", nil)
		}

		// Check if trying to absorb a record into itself
		for _, id := range request.IdsToAbsorb {
			if id == targetId {
				return apis.NewBadRequestError("Cannot absorb a record into itself", nil)
			}
		}

		// Check if user has the absorb claim
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		hasAbsorbClaim, err := utilities.HasClaim(app.Dao(), authRecord.Id, "absorb")
		if err != nil {
			return apis.NewBadRequestError("Failed to check user claims", err)
		}
		if !hasAbsorbClaim {
			return apis.NewForbiddenError("User does not have permission to absorb records", nil)
		}

		// Check if the collection is supported
		_, err = GetConfigsAndTable(collectionName)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to absorb records", err)
		}

		// First verify that the target record exists
		_, err = app.Dao().FindRecordById(collectionName, targetId)
		if err != nil {
			return apis.NewNotFoundError("Failed to find target record", err)
		}

		// Then verify that all records to absorb exist
		for _, id := range request.IdsToAbsorb {
			_, err := app.Dao().FindRecordById(collectionName, id)
			if err != nil {
				return apis.NewNotFoundError("Failed to find record to absorb", err)
			}
		}

		err = AbsorbRecords(app, collectionName, targetId, request.IdsToAbsorb)
		if err != nil {
			// Customize the error response as needed
			return apis.NewApiError(http.StatusInternalServerError, "Failed to absorb records", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Successfully absorbed %d records into %s", len(request.IdsToAbsorb), targetId),
		})
	}
}

func AbsorbRecords(app core.App, collectionName string, targetID string, idsToAbsorb []string) error {
	// Get reference configs based on collection name. These are the tables and
	// corresponding columns that need to be updated for each record based on
	// the collection name.
	refConfigs, err := GetConfigsAndTable(collectionName)
	if err != nil {
		return fmt.Errorf("error getting reference configs: %w", err)
	}

	// Absorb the records in a transaction
	err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
		// create the temp table to store the ids to be absorbed
		_, err := txDao.DB().NewQuery(`
			CREATE TEMPORARY TABLE ids_to_absorb (old_id TEXT NOT NULL)
		`).Execute()
		if err != nil {
			return fmt.Errorf("creating temp table: %w", err)
		}

		// create the sql-formatted values for the ids to be absorbed
		args := make([]string, len(idsToAbsorb))
		for i, id := range idsToAbsorb {
			args[i] = fmt.Sprintf("('%s')", id)
		}

		// insert the ids to be absorbed into the temp table
		insertQuery := fmt.Sprintf(`
			INSERT INTO ids_to_absorb (old_id) VALUES %s
		`, strings.Join(args, ","))

		_, err = txDao.DB().NewQuery(insertQuery).Execute()
		if err != nil {
			return fmt.Errorf("populating temp table: %w", err)
		}

		// Update all references using EXISTS
		for _, ref := range refConfigs {
			query := fmt.Sprintf(`
					UPDATE %[1]s 
					SET %[2]s = {:target_id} 
					WHERE EXISTS (
							SELECT 1 
							FROM ids_to_absorb 
							WHERE ids_to_absorb.old_id = %[1]s.%[2]s
					)
			`, ref.Table, ref.Column)

			_, err = txDao.DB().NewQuery(query).Bind(dbx.Params{
				"target_id": targetID,
			}).Execute()
			if err != nil {
				return fmt.Errorf("updating references in %s: %w", ref.Table, err)
			}
		}

		// Delete absorbed records
		_, err = txDao.DB().NewQuery(fmt.Sprintf(`
			DELETE FROM %[1]s 
			WHERE EXISTS (
					SELECT 1 
					FROM ids_to_absorb 
					WHERE ids_to_absorb.old_id = %[1]s.id
			)
		`, collectionName)).Execute()
		if err != nil {
			return fmt.Errorf("deleting absorbed records: %w", err)
		}

		// Clean up
		_, err = txDao.DB().NewQuery("DROP TABLE ids_to_absorb").Execute()
		if err != nil {
			return fmt.Errorf("dropping temp table: %w", err)
		}

		return nil // Return nil if transaction is successful

	})

	if err != nil {
		return fmt.Errorf("error absorbing records: %w", err)
	}

	return nil
}

// Get the configs and target table based on collection

type RefConfig struct {
	Table  string
	Column string
}

func GetConfigsAndTable(collectionName string) ([]RefConfig, error) {
	// TODO: Write configs for each collection that we want to be able to absorb
	// against. This is a list of tables and their corresponding columns that need
	// to be updated for each record based on the collection name.
	switch collectionName {
	case "clients":
		return []RefConfig{
			{"client_contacts", "client"},
			{"jobs", "client"},
		}, nil

	case "contacts":
		return []RefConfig{
			{"jobs", "contact"},
		}, nil

	// Add more cases as needed
	default:
		// return an error if the collection is not supported
		return nil, fmt.Errorf("unknown collection: %s", collectionName)
	}
}
