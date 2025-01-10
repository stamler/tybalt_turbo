package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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

// AbsorbAction represents a record in the absorb_actions collection
type AbsorbAction struct {
	Id                string          `json:"id"`
	CollectionName    string          `json:"collection_name"`
	TargetId          string          `json:"target_id"`
	AbsorbedRecords   json.RawMessage `json:"absorbed_records"`
	UpdatedReferences json.RawMessage `json:"updated_references"`
	Created           time.Time       `json:"created"`
	Updated           time.Time       `json:"updated"`
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

		// Check if the collection is supported and get configs
		_, parentConstraint, err := GetConfigsAndTable(collectionName)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to absorb records", err)
		}

		// First verify that the target record exists
		targetRecord, err := app.Dao().FindRecordById(collectionName, targetId)
		if err != nil {
			return apis.NewNotFoundError("Failed to find target record", err)
		}

		// Then verify that all records to absorb exist and satisfy parent constraint
		for _, id := range request.IdsToAbsorb {
			record, err := app.Dao().FindRecordById(collectionName, id)
			if err != nil {
				return apis.NewNotFoundError("Failed to find record to absorb", err)
			}

			// Check parent constraint if it exists
			if parentConstraint != nil {
				targetParent := targetRecord.GetString(parentConstraint.Field)
				recordParent := record.GetString(parentConstraint.Field)
				if targetParent != recordParent {
					return apis.NewBadRequestError(
						fmt.Sprintf("Cannot absorb records with different %s values", parentConstraint.Field),
						nil,
					)
				}
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
	// Get reference configs based on collection name
	refConfigs, _, err := GetConfigsAndTable(collectionName)
	if err != nil {
		return fmt.Errorf("error getting reference configs: %w", err)
	}

	// Create reference tracker
	refTracker := newReferenceTracker()

	// Absorb the records in a transaction
	err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
		// First create the temp table to validate IDs
		_, err = txDao.DB().NewQuery(`
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

		// Check for existing absorb action
		existingAction, err := getAbsorbAction(txDao, collectionName)
		if err != nil {
			return fmt.Errorf("error checking existing absorb action: %w", err)
		}
		if existingAction != nil {
			return fmt.Errorf("there is an existing absorb action for collection %s that must be undone or deleted first", collectionName)
		}

		// Serialize the records to be absorbed
		absorbedRecords, err := serializeRecords(txDao, collectionName, idsToAbsorb)
		if err != nil {
			return fmt.Errorf("error serializing records: %w", err)
		}

		// Record the absorb action
		updatedRefs, err := refTracker.serialize()
		if err != nil {
			return fmt.Errorf("error serializing reference updates: %w", err)
		}

		if err := recordAbsorbAction(txDao, collectionName, targetID, absorbedRecords, updatedRefs); err != nil {
			return fmt.Errorf("error recording absorb action: %w", err)
		}

		// Update all references using EXISTS
		for _, ref := range refConfigs {
			// First get the current values for tracking
			trackQuery := fmt.Sprintf(`
				SELECT id, %[2]s 
				FROM %[1]s 
				WHERE EXISTS (
					SELECT 1 
					FROM ids_to_absorb 
					WHERE ids_to_absorb.old_id = %[1]s.%[2]s
				)
			`, ref.Table, ref.Column)

			rows, err := txDao.DB().NewQuery(trackQuery).Rows()
			if err != nil {
				return fmt.Errorf("tracking references in %s: %w", ref.Table, err)
			}
			defer rows.Close()

			for rows.Next() {
				var id, oldValue string
				if err := rows.Scan(&id, &oldValue); err != nil {
					return fmt.Errorf("scanning reference row: %w", err)
				}
				refTracker.trackUpdate(ref.Table, id, oldValue)
			}

			// Now update the references
			updateQuery := fmt.Sprintf(`
				UPDATE %[1]s 
				SET %[2]s = {:target_id} 
				WHERE EXISTS (
					SELECT 1 
					FROM ids_to_absorb 
					WHERE ids_to_absorb.old_id = %[1]s.%[2]s
				)
			`, ref.Table, ref.Column)

			_, err = txDao.DB().NewQuery(updateQuery).Bind(dbx.Params{
				"target_id": targetID,
			}).Execute()
			if err != nil {
				return fmt.Errorf("updating references in %s: %w", ref.Table, err)
			}
		}

		// Update the absorb action with the final reference updates
		updatedRefs, err = refTracker.serialize()
		if err != nil {
			return fmt.Errorf("error serializing final reference updates: %w", err)
		}

		action, err := getAbsorbAction(txDao, collectionName)
		if err != nil {
			return fmt.Errorf("error fetching absorb action for update: %w", err)
		}

		actionRecord, err := txDao.FindRecordById("absorb_actions", action.Id)
		if err != nil {
			return fmt.Errorf("error fetching absorb action record: %w", err)
		}

		actionRecord.Set("updated_references", string(updatedRefs))
		if err := txDao.SaveRecord(actionRecord); err != nil {
			return fmt.Errorf("error updating absorb action: %w", err)
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

		return nil
	})

	if err != nil {
		return fmt.Errorf("error absorbing records: %w", err)
	}

	return nil
}

// CreateUndoAbsorbHandler creates a handler for undoing an absorb operation.
// This handler performs the following steps:
//  1. Verifies user permissions via the 'absorb' claim
//  2. Retrieves the absorb action record for the collection
//  3. Deserializes the absorbed records and reference updates
//  4. In a transaction:
//     a. Recreates all previously absorbed records with their original data
//     b. Restores all references in related tables back to their original values
//     c. Deletes the absorb action record to allow future absorb operations
func CreateUndoAbsorbHandler(app core.App, collectionName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Step 1: Permission Check
		// Verify that the user has the 'absorb' claim, which is required for both
		// absorbing and undoing absorb operations
		authRecord, _ := c.Get(apis.ContextAuthRecordKey).(*models.Record)
		hasAbsorbClaim, err := utilities.HasClaim(app.Dao(), authRecord.Id, "absorb")
		if err != nil {
			return apis.NewBadRequestError("Failed to check user claims", err)
		}
		if !hasAbsorbClaim {
			return apis.NewForbiddenError("User does not have permission to undo absorb", nil)
		}

		// Step 2: Retrieve Absorb Action
		// Get the existing absorb action for this collection. There can only be
		// one absorb action per collection at a time because the collection_name
		// has a unique constraint.
		action, err := getAbsorbAction(app.Dao(), collectionName)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to check absorb action", err)
		}
		if action == nil {
			return apis.NewNotFoundError("No absorb action found for collection", nil)
		}

		// Step 3: Parse Stored Data
		// Deserialize the JSON data stored in the absorb action record.
		// This includes both the original records that were absorbed and
		// a map of all reference updates that were made.
		var absorbedRecords []map[string]interface{}
		if err := json.Unmarshal(action.AbsorbedRecords, &absorbedRecords); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to parse absorbed records", err)
		}

		var updatedRefs map[string]map[string]string
		if err := json.Unmarshal(action.UpdatedReferences, &updatedRefs); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to parse updated references", err)
		}

		// Step 4: Execute Undo Operation
		// Perform all undo operations in a single transaction to ensure data consistency.
		// If any part fails, the entire undo operation is rolled back.
		err = app.Dao().RunInTransaction(func(txDao *daos.Dao) error {
			// Step 4a: Recreate Absorbed Records
			// Reconstruct each absorbed record with its original data, including
			// system fields like id, created, updated, etc., from the absorbedRecords
			// array.
			collection, err := txDao.FindCollectionByNameOrId(collectionName)
			if err != nil {
				return fmt.Errorf("error finding collection: %w", err)
			}

			for _, recordData := range absorbedRecords {
				record := models.NewRecord(collection)
				for field, value := range recordData {
					record.Set(field, value)
				}
				if err := txDao.SaveRecord(record); err != nil {
					return fmt.Errorf("error recreating record: %w", err)
				}
			}

			// Step 4b: Restore References
			// For each table where references were updated, restore the original
			// references that pointed to the absorbed records
			refConfigs, _, err := GetConfigsAndTable(collectionName)
			if err != nil {
				return fmt.Errorf("error getting reference configs: %w", err)
			}

			// Create a map for quick lookup of column names by table
			columnsByTable := make(map[string]string)
			for _, ref := range refConfigs {
				columnsByTable[ref.Table] = ref.Column
			}

			for table, updates := range updatedRefs {
				column, ok := columnsByTable[table]
				if !ok {
					return fmt.Errorf("no column configuration found for table %s", table)
				}

				for recordId, oldValue := range updates {
					query := fmt.Sprintf(`
						UPDATE %s 
						SET %s = {:old_value}
						WHERE id = {:record_id}
					`, table, column)

					_, err = txDao.DB().NewQuery(query).Bind(dbx.Params{
						"old_value": oldValue,
						"record_id": recordId,
					}).Execute()
					if err != nil {
						return fmt.Errorf("error restoring reference in %s: %w", table, err)
					}
				}
			}

			// Step 4c: Clean Up
			// Delete the absorb action record to allow future absorb operations
			// on this collection
			actionRecord, err := txDao.FindRecordById("absorb_actions", action.Id)
			if err != nil {
				return fmt.Errorf("error finding absorb action: %w", err)
			}
			if err := txDao.DeleteRecord(actionRecord); err != nil {
				return fmt.Errorf("error deleting absorb action: %w", err)
			}

			return nil
		})

		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to undo absorb", err)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Successfully undid absorb operation",
		})
	}
}

// Get the configs and target table based on collection

type RefConfig struct {
	Table  string
	Column string
}

// Add parent constraint configuration
type ParentConstraint struct {
	Collection string // The collection that must match (e.g., "clients")
	Field      string // The field that must match (e.g., "client")
}

func GetConfigsAndTable(collectionName string) ([]RefConfig, *ParentConstraint, error) {
	// TODO: Write configs for each collection that we want to be able to absorb
	// against. This is a list of tables and their corresponding columns that need
	// to be updated for each record based on the collection name.
	switch collectionName {
	case "clients":
		return []RefConfig{
			{"client_contacts", "client"},
			{"jobs", "client"},
		}, nil, nil

	case "client_contacts":
		return []RefConfig{
				{"jobs", "contact"},
			}, &ParentConstraint{
				Collection: "clients",
				Field:      "client",
			}, nil

	// Add more cases as needed
	default:
		// return an error if the collection is not supported
		return nil, nil, fmt.Errorf("unknown collection: %s", collectionName)
	}
}

// getAbsorbAction returns the existing absorb action for the collection or nil if none exists
func getAbsorbAction(dao *daos.Dao, collectionName string) (*AbsorbAction, error) {
	record, err := dao.FindFirstRecordByData("absorb_actions", "collection_name", collectionName)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("error checking existing absorb action: %w", err)
	}

	var action AbsorbAction
	action.Id = record.Id
	action.CollectionName = record.GetString("collection_name")
	action.TargetId = record.GetString("target_id")
	action.AbsorbedRecords = []byte(record.GetString("absorbed_records"))
	action.UpdatedReferences = []byte(record.GetString("updated_references"))
	action.Created = record.GetDateTime("created").Time()
	action.Updated = record.GetDateTime("updated").Time()

	return &action, nil
}

// recordAbsorbAction creates a new absorb action record
func recordAbsorbAction(dao *daos.Dao, collectionName string, targetId string, absorbedRecords []byte, updatedReferences []byte) error {
	collection, err := dao.FindCollectionByNameOrId("absorb_actions")
	if err != nil {
		return fmt.Errorf("error finding absorb_actions collection: %w", err)
	}

	record := models.NewRecord(collection)
	record.Set("collection_name", collectionName)
	record.Set("target_id", targetId)
	record.Set("absorbed_records", string(absorbedRecords))
	record.Set("updated_references", string(updatedReferences))

	if err := dao.SaveRecord(record); err != nil {
		return fmt.Errorf("error saving absorb action: %w", err)
	}

	return nil
}

// serializeRecords fetches and serializes records to be absorbed
func serializeRecords(dao *daos.Dao, collectionName string, ids []string) ([]byte, error) {
	var records []map[string]interface{}

	collection, err := dao.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("error finding collection: %w", err)
	}

	for _, id := range ids {
		record, err := dao.FindRecordById(collectionName, id)
		if err != nil {
			return nil, fmt.Errorf("error fetching record %s: %w", id, err)
		}

		recordData := make(map[string]interface{})
		// Include base model fields
		recordData["id"] = record.Id
		recordData["created"] = record.Created
		recordData["updated"] = record.Updated
		recordData["collectionId"] = record.Collection().Id
		recordData["collectionName"] = record.Collection().Name

		// Then include all schema fields
		for _, field := range collection.Schema.Fields() {
			recordData[field.Name] = record.Get(field.Name)
		}
		records = append(records, recordData)
	}

	serialized, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("error serializing records: %w", err)
	}

	return serialized, nil
}

// trackReferences tracks reference updates during absorption
type referenceTracker struct {
	updates map[string]map[string]string // collection -> recordId -> oldValue
}

func newReferenceTracker() *referenceTracker {
	return &referenceTracker{
		updates: make(map[string]map[string]string),
	}
}

func (rt *referenceTracker) trackUpdate(collection, recordId, oldValue string) {
	if rt.updates[collection] == nil {
		rt.updates[collection] = make(map[string]string)
	}
	rt.updates[collection][recordId] = oldValue
}

func (rt *referenceTracker) serialize() ([]byte, error) {
	return json.Marshal(rt.updates)
}
