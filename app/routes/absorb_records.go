package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/tools/subscriptions"
	"golang.org/x/sync/errgroup"

	"tybalt/absorb"
	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
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
}

func CreateAbsorbRecordsHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	// This handler absorbs multiple records into one target record for a given collection.
	// It performs the following actions:
	// 1. Validates the request body contains a list of IDs to absorb
	// 2. Calls the AbsorbRecords function to perform the absorption
	return func(e *core.RequestEvent) error {
		targetId := e.Request.PathValue("id")
		var request AbsorbRequest
		if err := e.BindBody(&request); err != nil {
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
		authRecord := e.Auth
		hasAbsorbClaim, err := utilities.HasClaim(app, authRecord, "absorb")
		if err != nil {
			return apis.NewBadRequestError("Failed to check user claims", err)
		}
		if !hasAbsorbClaim {
			return apis.NewForbiddenError("User does not have permission to absorb records", nil)
		}

		// Check if job editing is enabled for collections that modify jobs
		if collectionName == "clients" || collectionName == "client_contacts" {
			enabled, err := utilities.IsJobsEditingEnabled(app)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError, "Failed to check config", err)
			}
			if !enabled {
				return apis.NewForbiddenError("Job editing is disabled; cannot absorb records that modify jobs", nil)
			}
		}

		// Check if the collection is supported and get configs
		_, parentConstraint, err := absorb.GetRefConfigs(collectionName)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to absorb records", err)
		}

		// First verify that the target record exists
		targetRecord, err := app.FindRecordById(collectionName, targetId)
		if err != nil {
			return apis.NewNotFoundError("Failed to find target record", err)
		}

		// Then verify that all records to absorb exist and satisfy parent constraint
		for _, id := range request.IdsToAbsorb {
			record, err := app.FindRecordById(collectionName, id)
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

		return e.JSON(http.StatusOK, map[string]string{
			"message": fmt.Sprintf("Successfully absorbed %d records into %s", len(request.IdsToAbsorb), targetId),
		})
	}
}

func AbsorbRecords(app core.App, collectionName string, targetID string, idsToAbsorb []string) error {
	// Get reference configs based on collection name
	refConfigs, _, err := absorb.GetRefConfigs(collectionName)
	if err != nil {
		return fmt.Errorf("error getting reference configs: %w", err)
	}

	// Create reference tracker
	refTracker := newReferenceTracker()

	// Absorb the records in a transaction
	err = app.RunInTransaction(func(txApp core.App) error {
		// First create the temp table to validate IDs
		_, err = txApp.DB().NewQuery(`
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

		_, err = txApp.NonconcurrentDB().NewQuery(insertQuery).Execute()
		if err != nil {
			return fmt.Errorf("populating temp table: %w", err)
		}

		// Check for existing absorb action
		existingAction, err := getAbsorbAction(txApp, collectionName)
		if err != nil {
			return fmt.Errorf("error checking existing absorb action: %w", err)
		}
		if existingAction != nil {
			return fmt.Errorf("there is an existing absorb action for collection %s that must be undone or deleted first", collectionName)
		}

		// Serialize the records to be absorbed
		absorbedRecords, err := serializeRecords(txApp, collectionName, idsToAbsorb)
		if err != nil {
			return fmt.Errorf("error serializing records: %w", err)
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

			rows, err := txApp.DB().NewQuery(trackQuery).Rows()
			if err != nil {
				return fmt.Errorf("tracking references in %s: %w", ref.Table, err)
			}
			defer rows.Close()

			for rows.Next() {
				var id, oldValue string
				if err := rows.Scan(&id, &oldValue); err != nil {
					return fmt.Errorf("scanning reference row: %w", err)
				}
				refTracker.trackUpdate(ref.Table, ref.Column, id, oldValue)
			}

			// Now update the references
			// If the table has an _imported column, also set it to false to ensure
			// the change is written back to the legacy system (since this direct SQL
			// update bypasses PocketBase hooks that normally handle this).
			// We also update the `updated` field since direct SQL bypasses PocketBase's
			// autodate hooks. This is important for writeback queries that filter by
			// updated timestamp.
			importedClause := ""
			hasImported, err := utilities.TableHasImportedColumn(txApp, ref.Table)
			if err != nil {
				return fmt.Errorf("checking _imported column in %s: %w", ref.Table, err)
			}
			if hasImported {
				importedClause = ", _imported = false, updated = strftime('%Y-%m-%d %H:%M:%fZ', 'now')"
			}
			updateQuery := fmt.Sprintf(`
				UPDATE %[1]s 
				SET %[2]s = {:target_id}%[3]s 
				WHERE EXISTS (
					SELECT 1 
					FROM ids_to_absorb 
					WHERE ids_to_absorb.old_id = %[1]s.%[2]s
				)
			`, ref.Table, ref.Column, importedClause)

			_, err = txApp.NonconcurrentDB().NewQuery(updateQuery).Bind(dbx.Params{
				"target_id": targetID,
			}).Execute()
			if err != nil {
				return fmt.Errorf("updating references in %s: %w", ref.Table, err)
			}
		}

		// Update the absorb action with the reference updates
		updatedRefs, err := refTracker.serialize()
		if err != nil {
			return fmt.Errorf("error serializing reference updates: %w", err)
		}

		if err := recordAbsorbAction(txApp, collectionName, targetID, absorbedRecords, updatedRefs); err != nil {
			return fmt.Errorf("error recording absorb action: %w", err)
		}

		// Delete absorbed records
		_, err = txApp.NonconcurrentDB().NewQuery(fmt.Sprintf(`
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
		_, err = txApp.NonconcurrentDB().NewQuery("DROP TABLE ids_to_absorb").Execute()
		if err != nil {
			return fmt.Errorf("dropping temp table: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("error absorbing records: %w", err)
	}

	// Notify clients that an absorb operation has completed for this collection.
	if err := broadcastAbsorbCompletedEvent(app, collectionName); err != nil {
		app.Logger().Error("Failed to broadcast absorb_completed event", "err", err)
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
func CreateUndoAbsorbHandler(app core.App, collectionName string) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		// Step 1: Permission Check
		// Verify that the user has the 'absorb' claim, which is required for both
		// absorbing and undoing absorb operations
		authRecord := e.Auth
		hasAbsorbClaim, err := utilities.HasClaim(app, authRecord, "absorb")
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
		action, err := getAbsorbAction(app, collectionName)
		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to check absorb action", err)
		}
		if action == nil {
			return apis.NewNotFoundError("No absorb action found for collection", nil)
		}

		// Step 2b: Check if a child collection has a pending absorb that depends on this one
		// This enforces LIFO ordering - child absorbs must be undone before parent absorbs
		blocks := absorb.GetUndoBlockers(collectionName)
		for _, childCollection := range blocks {
			childAction, err := getAbsorbAction(app, childCollection)
			if err != nil {
				return apis.NewApiError(http.StatusInternalServerError,
					fmt.Sprintf("Failed to check %s absorb action", childCollection), err)
			}
			if childAction != nil {
				return apis.NewBadRequestError(
					fmt.Sprintf("Cannot undo %s absorb while %s absorb exists; undo %s absorb first",
						collectionName, childCollection, childCollection),
					nil,
				)
			}
		}

		// Step 3: Parse Stored Data
		// Deserialize the JSON data stored in the absorb action record.
		// This includes both the original records that were absorbed and
		// a map of all reference updates that were made.
		var absorbedRecords []map[string]interface{}
		if err := json.Unmarshal(action.AbsorbedRecords, &absorbedRecords); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to parse absorbed records", err)
		}

		// updatedRefs is serialized as updates[table][column][recordId] = oldValue
		var updatedRefs map[string]map[string]map[string]string
		if err := json.Unmarshal(action.UpdatedReferences, &updatedRefs); err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to parse updated references", err)
		}

		// Step 4: Execute Undo Operation
		// Perform all undo operations in a single transaction to ensure data consistency.
		// If any part fails, the entire undo operation is rolled back.
		err = app.RunInTransaction(func(txApp core.App) error {
			// Step 4a: Recreate Absorbed Records
			// Reconstruct each absorbed record with its original data, excluding
			// system fields like created, updated, etc., from the absorbedRecords
			// array.
			collection, err := txApp.FindCollectionByNameOrId(collectionName)
			if err != nil {
				return fmt.Errorf("error finding collection: %w", err)
			}

			for _, recordData := range absorbedRecords {
				record := core.NewRecord(collection)
				for field, value := range recordData {
					record.Set(field, value)
				}
				if err := txApp.Save(record); err != nil {
					return fmt.Errorf("error recreating record: %w", err)
				}
			}

			// Step 4b: Restore References
			// For each table where references were updated, restore the original
			// references that pointed to the absorbed records

			for table, columns := range updatedRefs {
				for column, updates := range columns {
					for recordId, oldValue := range updates {
						query := fmt.Sprintf(`
                            UPDATE %s 
                            SET %s = {:old_value}
                            WHERE id = {:record_id}
                        `, table, column)

						_, err = txApp.NonconcurrentDB().NewQuery(query).Bind(dbx.Params{
							"old_value": oldValue,
							"record_id": recordId,
						}).Execute()
						if err != nil {
							return fmt.Errorf("error restoring reference in %s.%s: %w", table, column, err)
						}
					}
				}
			}

			// Step 4c: Clean Up
			// Delete the absorb action record to allow future absorb operations
			// on this collection
			actionRecord, err := txApp.FindRecordById("absorb_actions", action.Id)
			if err != nil {
				return fmt.Errorf("error finding absorb action: %w", err)
			}
			if err := txApp.Delete(actionRecord); err != nil {
				return fmt.Errorf("error deleting absorb action: %w", err)
			}

			return nil
		})

		if err != nil {
			return apis.NewApiError(http.StatusInternalServerError, "Failed to undo absorb", err)
		}

		// Notify clients that the absorb undo has completed so that they can refresh.
		if err := broadcastAbsorbCompletedEvent(app, collectionName); err != nil {
			app.Logger().Error("Failed to broadcast absorb_completed event (undo)", "err", err)
		}

		return e.JSON(http.StatusOK, map[string]string{
			"message": "Successfully undid absorb operation",
		})
	}
}


// getAbsorbAction returns the existing absorb action for the collection or nil if none exists
func getAbsorbAction(app core.App, collectionName string) (*AbsorbAction, error) {
	record, err := app.FindFirstRecordByData("absorb_actions", "collection_name", collectionName)
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

	return &action, nil
}

// recordAbsorbAction creates a new absorb action record
func recordAbsorbAction(app core.App, collectionName string, targetId string, absorbedRecords []byte, updatedReferences []byte) error {
	collection, err := app.FindCollectionByNameOrId("absorb_actions")
	if err != nil {
		return fmt.Errorf("error finding absorb_actions collection: %w", err)
	}

	record := core.NewRecord(collection)
	record.Set("collection_name", collectionName)
	record.Set("target_id", targetId)
	record.Set("absorbed_records", string(absorbedRecords))
	record.Set("updated_references", string(updatedReferences))

	if err := app.Save(record); err != nil {
		return fmt.Errorf("error saving absorb action: %w", err)
	}

	return nil
}

// serializeRecords fetches and serializes records to be absorbed
func serializeRecords(app core.App, collectionName string, ids []string) ([]byte, error) {
	var records []map[string]interface{}

	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, fmt.Errorf("error finding collection: %w", err)
	}

	for _, id := range ids {
		record, err := app.FindRecordById(collectionName, id)
		if err != nil {
			return nil, fmt.Errorf("error fetching record %s: %w", id, err)
		}

		recordData := make(map[string]interface{})
		// Include original record id
		recordData["id"] = record.Id

		// Then include all schema fields
		for _, field := range collection.Fields {
			recordData[field.GetName()] = record.Get(field.GetName())
		}
		records = append(records, recordData)
	}

	serialized, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("error serializing records: %w", err)
	}

	return serialized, nil
}

// broadcastAbsorbCompletedEvent broadcasts a single custom realtime event indicating
// that an absorb operation for the specified collection has finished. Clients can
// listen to this event ("<collection>/absorb_completed") and refresh their data
// without dealing with individual record mutations.
func broadcastAbsorbCompletedEvent(app core.App, collectionName string) error {
	// Topic pattern: "<collection>/absorb_completed"
	topic := fmt.Sprintf("%s/absorb_completed", collectionName)

	payload := map[string]any{
		"action":     "absorb_completed",
		"collection": collectionName,
	}
	rawData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal absorb_completed payload: %w", err)
	}

	message := subscriptions.Message{
		Name: topic,
		Data: rawData,
	}

	group := new(errgroup.Group)
	chunks := app.SubscriptionsBroker().ChunkedClients(300) // same chunk size as other broadcasts
	for _, chunk := range chunks {
		c := chunk
		group.Go(func() error {
			for _, client := range c {
				if !client.HasSubscription(topic) {
					continue
				}
				client.Send(message)
			}
			return nil
		})
	}
	return group.Wait()
}

// trackReferences tracks reference updates during absorption
type referenceTracker struct {
	// updates[table][column][recordId] = oldValue
	updates map[string]map[string]map[string]string
}

func newReferenceTracker() *referenceTracker {
	return &referenceTracker{
		updates: make(map[string]map[string]map[string]string),
	}
}

func (rt *referenceTracker) trackUpdate(table, column, recordId, oldValue string) {
	if rt.updates[table] == nil {
		rt.updates[table] = make(map[string]map[string]string)
	}
	if rt.updates[table][column] == nil {
		rt.updates[table][column] = make(map[string]string)
	}
	rt.updates[table][column][recordId] = oldValue
}

func (rt *referenceTracker) serialize() ([]byte, error) {
	return json.Marshal(rt.updates)
}
