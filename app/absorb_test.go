package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/routes"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

// CountResult is a helper struct used throughout the test to store and compare
// record counts from database queries. It's used both for the main collection
// counts and for counting references in related tables.
type CountResult struct {
	Count int64 `db:"count"`
}

// TestAbsorbRecords verifies the record absorption/merging functionality.
// The test ensures that when multiple records are absorbed into a target record:
// 1. The target record remains intact
// 2. The absorbed records are properly deleted
// 3. All references to absorbed records are updated to point to the target
// 4. No data is lost in the process
func TestAbsorbRecords(t *testing.T) {
	// Set up a clean test environment and ensure cleanup after test completion
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// Define test scenarios to cover both successful and error cases.
	// Each test case specifies:
	// - A target record that will absorb other records
	// - A list of records to be absorbed
	// - Whether we expect an error
	// - For error cases, what the error should contain
	tests := []struct {
		name           string
		collectionName string
		targetID       string
		idsToAbsorb    []string
		wantErr        bool
		errorContains  string
	}{
		{
			name:           "successfully_absorb_client_records",
			collectionName: "clients",
			targetID:       "lb0fnenkeyitsny", // This is our target client that will remain
			idsToAbsorb: []string{ // These clients will be absorbed and deleted
				"eldtxi3i4h00k8r",
				"pqpd90fqd5ohjcs",
			},
			wantErr: false,
		},
		{
			name:           "fail_absorb_with_unknown_collection",
			collectionName: "unknown_collection", // This collection doesn't exist
			targetID:       "some_id",
			idsToAbsorb:    []string{"id1", "id2"},
			wantErr:        true,
			errorContains:  "unknown collection",
		},
		{
			name:           "fail_absorb_with_sql_injection_attempt",
			collectionName: "clients",
			targetID:       "lb0fnenkeyitsny",
			idsToAbsorb: []string{
				"'; DROP TABLE clients; --", // SQL injection attempt
			},
			wantErr:       true,
			errorContains: "populating temp table", // The error occurs when trying to insert malformed data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These variables track the state before absorption
			var initialCount int64 // Total records in the collection
			// Track reference counts per table+column
			initialRefCounts := make(map[string]int64) // key: "table.column" -> count
			// Track client note IDs per absorbed client so we can verify they moved
			absorbedClientNoteIDs := make(map[string][]string)

			// For successful test cases, we need to verify the initial state
			// Error cases skip this as they'll fail before any state changes
			if !tt.wantErr {
				// Step 1: Get the initial count of records in the collection
				// This will be used later to verify that the count decreased
				// by exactly the number of absorbed records
				var result CountResult
				err := app.DB().NewQuery(fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tt.collectionName)).One(&result)
				if err != nil {
					t.Fatalf("failed to get initial count: %v", err)
				}
				initialCount = result.Count

				// Step 2: Verify the target record exists
				// This is crucial as it's the record that will absorb the others
				targetRecord, err := app.FindRecordById(tt.collectionName, tt.targetID)
				if err != nil {
					t.Fatalf("failed to find target record: %v", err)
				}
				if targetRecord == nil {
					t.Fatal("target record is nil")
				}

				// Step 3: Verify all records to be absorbed exist
				// We need to ensure they exist before trying to absorb them
				for _, id := range tt.idsToAbsorb {
					record, err := app.FindRecordById(tt.collectionName, id)
					if err != nil {
						t.Fatalf("failed to find record %s: %v", id, err)
					}
					if record == nil {
						t.Fatalf("record %s is nil", id)
					}
				}

				// Step 4: Capture the initial state of all reference tables
				// These tables contain foreign keys pointing to our records
				refConfigs, _, err := routes.GetConfigsAndTable(tt.collectionName)
				if err != nil {
					t.Fatalf("failed to get ref configs: %v", err)
				}

				// For each reference table, count how many references exist
				// to either the target or the records being absorbed
				for _, ref := range refConfigs {
					var result CountResult
					query := fmt.Sprintf(
						"SELECT COUNT(*) as count FROM %s WHERE %s IN (%s)",
						ref.Table,
						ref.Column,
						"'"+strings.Join(append(tt.idsToAbsorb, tt.targetID), "','")+"'",
					)
					err = app.DB().NewQuery(query).One(&result)
					if err != nil {
						t.Fatalf("failed to get reference count for %s: %v", ref.Table, err)
					}
					initialRefCounts[ref.Table+"."+ref.Column] = result.Count
				}

				// Track client note IDs for clients to be absorbed so we can assert they switch to the target client
				for _, id := range tt.idsToAbsorb {
					var noteRows []struct {
						ID string `db:"id"`
					}
					query := `SELECT id FROM client_notes WHERE client = {:client}`
					if err := app.DB().NewQuery(query).Bind(dbx.Params{"client": id}).All(&noteRows); err != nil {
						t.Fatalf("failed to get client notes for %s: %v", id, err)
					}
					for _, row := range noteRows {
						absorbedClientNoteIDs[id] = append(absorbedClientNoteIDs[id], row.ID)
					}
				}
			}

			// Step 5: Execute the actual absorption operation
			err := routes.AbsorbRecords(app, tt.collectionName, tt.targetID, tt.idsToAbsorb)

			// Step 6: Verify the results
			if tt.wantErr {
				// For error cases, verify we got the expected error
				if err == nil {
					t.Error("expected error but got nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}
			} else {
				// For successful cases, perform comprehensive verification
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Step 6a: Verify the total record count decreased correctly
				var result CountResult
				err = app.DB().NewQuery(fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tt.collectionName)).One(&result)
				if err != nil {
					t.Fatalf("failed to get final count: %v", err)
				}
				expectedFinalCount := initialCount - int64(len(tt.idsToAbsorb))
				if result.Count != expectedFinalCount {
					t.Errorf("expected count %d, got %d", expectedFinalCount, result.Count)
				}

				// Step 6b: Verify all absorbed records were deleted
				for _, id := range tt.idsToAbsorb {
					record, err := app.FindRecordById(tt.collectionName, id)
					if err == nil {
						t.Errorf("expected error finding absorbed record %s", id)
					}
					if record != nil {
						t.Errorf("absorbed record %s still exists", id)
					}
				}

				// Step 6c: Verify the target record still exists
				targetRecord, err := app.FindRecordById(tt.collectionName, tt.targetID)
				if err != nil {
					t.Fatalf("failed to find target record: %v", err)
				}
				if targetRecord == nil {
					t.Fatal("target record is nil")
				}

				// Step 6d: Verify all references were properly updated
				refConfigs, _, err := routes.GetConfigsAndTable(tt.collectionName)
				if err != nil {
					t.Fatalf("failed to get ref configs: %v", err)
				}

				for _, ref := range refConfigs {
					// First verify no references to absorbed records exist
					var result CountResult
					query := fmt.Sprintf(
						"SELECT COUNT(*) as count FROM %s WHERE %s IN (%s)",
						ref.Table,
						ref.Column,
						"'"+strings.Join(tt.idsToAbsorb, "','")+"'",
					)
					err = app.DB().NewQuery(query).One(&result)
					if err != nil {
						t.Fatalf("failed to check absorbed references: %v", err)
					}
					if result.Count != 0 {
						t.Errorf("found %d references to absorbed records in %s", result.Count, ref.Table)
					}

					// Then verify all references now point to the target record
					// The total count should match our initial reference count
					var targetResult CountResult
					query = fmt.Sprintf(
						"SELECT COUNT(*) as count FROM %s WHERE %s = '%s'",
						ref.Table,
						ref.Column,
						tt.targetID,
					)
					err = app.DB().NewQuery(query).One(&targetResult)
					if err != nil {
						t.Fatalf("failed to check target references: %v", err)
					}
					key := ref.Table + "." + ref.Column
					if targetResult.Count != initialRefCounts[key] {
						t.Errorf("expected %d references in %s, got %d", initialRefCounts[key], key, targetResult.Count)
					}
				}

				// Step 6e: Verify each client note from absorbed clients now points to the target client
				for sourceClient, noteIDs := range absorbedClientNoteIDs {
					for _, noteID := range noteIDs {
						var note struct {
							Client string `db:"client"`
						}
						query := `SELECT client FROM client_notes WHERE id = {:id}`
						if err := app.DB().NewQuery(query).Bind(dbx.Params{"id": noteID}).One(&note); err != nil {
							t.Fatalf("failed to load client note %s: %v", noteID, err)
						}
						if note.Client != tt.targetID {
							t.Errorf("client note %s from %s still references %s instead of %s", noteID, sourceClient, note.Client, tt.targetID)
						}
					}
				}
			}
		})
	}
}

// TestAbsorbSetsImportedFalseOnJobs verifies that when absorbing clients,
// any jobs that reference the absorbed clients have their _imported flag
// set to false and their updated timestamp is refreshed. This is necessary
// because absorb uses direct SQL updates which bypass PocketBase hooks, so
// we need to explicitly flag these jobs for writeback to the legacy system
// and update the timestamp so they're picked up by writeback queries that
// filter by updated timestamp.
func TestAbsorbSetsImportedFalseOnJobs(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// The test data has:
	// - Job u09fwwcg07y03m7 with client pqpd90fqd5ohjcs
	// - Job zke3cs3yipplwtu with client eldtxi3i4h00k8r
	// We will absorb these clients into lb0fnenkeyitsny

	targetID := "lb0fnenkeyitsny"
	idsToAbsorb := []string{"eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"}
	jobsToCheck := []string{"u09fwwcg07y03m7", "zke3cs3yipplwtu"}

	// First, set _imported = true and a known old timestamp on the jobs that will be affected
	for _, jobID := range jobsToCheck {
		_, err := app.NonconcurrentDB().NewQuery("UPDATE jobs SET _imported = true, updated = '2020-01-01 00:00:00.000Z' WHERE id = {:id}").Bind(dbx.Params{"id": jobID}).Execute()
		if err != nil {
			t.Fatalf("failed to set _imported = true on job %s: %v", jobID, err)
		}
	}

	// Verify _imported is true and capture the old updated timestamp before absorb
	oldUpdated := make(map[string]string)
	for _, jobID := range jobsToCheck {
		var result struct {
			Imported bool   `db:"_imported"`
			Updated  string `db:"updated"`
		}
		err := app.DB().NewQuery("SELECT _imported, updated FROM jobs WHERE id = {:id}").Bind(dbx.Params{"id": jobID}).One(&result)
		if err != nil {
			t.Fatalf("failed to check _imported on job %s: %v", jobID, err)
		}
		if !result.Imported {
			t.Fatalf("_imported should be true before absorb for job %s", jobID)
		}
		oldUpdated[jobID] = result.Updated
	}

	// Perform the absorb
	err := routes.AbsorbRecords(app, "clients", targetID, idsToAbsorb)
	if err != nil {
		t.Fatalf("failed to absorb: %v", err)
	}

	// Verify _imported is now false and updated timestamp has changed on the affected jobs
	for _, jobID := range jobsToCheck {
		var result struct {
			Imported bool   `db:"_imported"`
			Updated  string `db:"updated"`
		}
		err := app.DB().NewQuery("SELECT _imported, updated FROM jobs WHERE id = {:id}").Bind(dbx.Params{"id": jobID}).One(&result)
		if err != nil {
			t.Fatalf("failed to check _imported on job %s after absorb: %v", jobID, err)
		}
		if result.Imported {
			t.Errorf("_imported should be false after absorb for job %s, but it is still true", jobID)
		}
		if result.Updated == oldUpdated[jobID] {
			t.Errorf("updated timestamp should have changed after absorb for job %s, but it is still %s", jobID, result.Updated)
		}
	}
}

// TestAbsorbRoutes tests the HTTP API endpoints for record absorption
func TestAbsorbRoutes(t *testing.T) {
	userToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	bookKeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate an invalid token to test auth record retrieval failure
	invalidToken := "invalid_token_format"

	// Create a custom test app factory for testing unsupported collection
	unsupportedCollectionTestApp := func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp("./test_pb_data")
		if err != nil {
			t.Fatal(err)
		}

		// Add a route with an unsupported collection
		app.OnServe().BindFunc(func(e *core.ServeEvent) error {
			e.Router.POST("/api/test_unsupported/{id}/absorb", routes.CreateAbsorbRecordsHandler(app, "unsupported_collection")).Bind(apis.RequireAuth("users"))
			return e.Next()
		})

		return app
	}

	// Create a custom test app factory for testing claim check failure
	claimCheckFailureTestApp := func(t testing.TB) *tests.TestApp {
		app, err := tests.NewTestApp("./test_pb_data")
		if err != nil {
			t.Fatal(err)
		}

		// Add routes with the broken claims table
		app.OnServe().BindFunc(func(e *core.ServeEvent) error {
			// Break the claims table
			_, err := app.NonconcurrentDB().NewQuery("ALTER TABLE claims RENAME TO claims_broken").Execute()
			if err != nil {
				t.Fatal(err)
			}

			e.Router.POST("/api/clients/{id}/absorb", routes.CreateAbsorbRecordsHandler(app, "clients")).Bind(apis.RequireAuth("users"))
			return e.Next()
		})

		return app
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "unauthorized request",
			Method:         http.MethodPost,
			URL:            "/api/clients/lb0fnenkeyitsny/absorb",
			Body:           strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			ExpectedStatus: 401,
			ExpectedContent: []string{
				`"message":"The request requires valid record authorization token."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid request body",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`invalid json`),
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Invalid request body."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "empty ids list",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": []}`),
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"No IDs provided to absorb."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user (no absorb claim)",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"message":"User does not have permission to absorb records."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "authorized user (has absorb claim)",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Successfully absorbed 2 records into lb0fnenkeyitsny"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordValidate":           1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelValidate":            1,
				"OnModelAfterCreateSuccess":  1,
				"*":                          0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid auth token format",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			Headers: map[string]string{
				"Authorization": invalidToken,
			},
			ExpectedStatus: 401,
			ExpectedContent: []string{
				`"message":"The request requires valid record authorization token."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "absorb non-existent records",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["definitely_nonexistent_1", "definitely_nonexistent_2"]}`),
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"Failed to find record to absorb.","status":404`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "absorb into non-existent target",
			Method: http.MethodPost,
			URL:    "/api/clients/definitely_nonexistent_target/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"Failed to find target record.","status":404`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unsupported collection",
			Method: http.MethodPost,
			URL:    "/api/test_unsupported/test_id/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["test1", "test2"]}`),
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"message":"Failed to absorb records.","status":500`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: unsupportedCollectionTestApp,
		},
		{
			Name:   "absorb record into itself",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["lb0fnenkeyitsny"]}`),
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Cannot absorb a record into itself."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "claim check failure",
			Method: http.MethodPost,
			URL:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to check user claims."`,
			},
			TestAppFactory: claimCheckFailureTestApp,
		},
		{
			Name:   "undo absorb unauthorized user",
			Method: http.MethodPost,
			URL:    "/api/clients/undo_absorb",
			Headers: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"message":"User does not have permission to undo absorb."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "undo absorb no action exists",
			Method: http.MethodPost,
			URL:    "/api/clients/undo_absorb",
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"message":"No absorb action found for collection."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "undo absorb successful",
			Method: http.MethodPost,
			URL:    "/api/clients/undo_absorb",
			Headers: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Successfully undid absorb operation"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordDelete":             1,
				"OnRecordDeleteExecute":      1,
				"OnRecordValidate":           1,
				"OnRecordCreate":             1,
				"OnRecordCreateExecute":      1,
				"OnRecordAfterCreateSuccess": 1,
				"OnModelDelete":              1,
				"OnModelDeleteExecute":       1,
				"OnRecordAfterDeleteSuccess": 1,
				"OnModelValidate":            1,
				"OnModelCreate":              1,
				"OnModelCreateExecute":       1,
				"OnModelAfterCreateSuccess":  1,
				"OnModelAfterDeleteSuccess":  1,
				"*":                          0,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(t)
				// Create an absorb action to undo
				err := routes.AbsorbRecords(app, "clients", "lb0fnenkeyitsny", []string{"eldtxi3i4h00k8r"})
				if err != nil {
					t.Fatal(err)
				}
				return app
			},
		},
		/*
			{
				Name:   "undo absorb claim check failure",
				Method: http.MethodPost,
				URL:    "/api/clients/undo_absorb",
				Headers: map[string]string{
					"Authorization": bookKeeperToken,
				},
				ExpectedStatus: 400,
				ExpectedContent: []string{
					`"message":"Failed to check user claims."`,
				},
				TestAppFactory: claimCheckFailureTestApp,
			},
		*/
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestUndoAbsorb verifies the undo functionality of record absorption.
// The test ensures that when an absorb operation is undone:
// 1. The absorbed records are restored with their original data
// 2. All references are restored to their original values
// 3. The absorb action record is deleted
// 4. The system returns to its original state
func TestUndoAbsorb(t *testing.T) {
	app := testutils.SetupTestApp(t)
	defer app.Cleanup()

	// First perform an absorption to create the state we want to undo
	collectionName := "clients"
	targetID := "lb0fnenkeyitsny"
	idsToAbsorb := []string{"eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"}

	// Store initial state
	initialState := make(map[string]map[string]interface{})
	// Track reference counts per table+column
	initialRefCounts := make(map[string]int64)

	// Get reference configs for the collection
	refConfigs, _, err := routes.GetConfigsAndTable(collectionName)
	if err != nil {
		t.Fatalf("failed to get ref configs: %v", err)
	}

	// Store initial record data
	for _, id := range append(idsToAbsorb, targetID) {
		record, err := app.FindRecordById(collectionName, id)
		if err != nil {
			t.Fatalf("failed to find record %s: %v", id, err)
		}
		recordData := make(map[string]interface{})
		for _, field := range record.Collection().Fields.FieldNames() {
			recordData[field] = record.Get(field)
		}
		initialState[id] = recordData
	}

	// Store initial reference counts
	for _, ref := range refConfigs {
		var result CountResult
		query := fmt.Sprintf(
			"SELECT COUNT(*) as count FROM %s WHERE %s IN (%s)",
			ref.Table,
			ref.Column,
			"'"+strings.Join(append(idsToAbsorb, targetID), "','")+"'",
		)
		err = app.DB().NewQuery(query).One(&result)
		if err != nil {
			t.Fatalf("failed to get reference count for %s: %v", ref.Table, err)
		}
		initialRefCounts[ref.Table+"."+ref.Column] = result.Count
	}

	// Perform the absorption
	err = routes.AbsorbRecords(app, collectionName, targetID, idsToAbsorb)
	if err != nil {
		t.Fatalf("failed to absorb records: %v", err)
	}

	// Verify absorption was successful
	for _, id := range idsToAbsorb {
		record, err := app.FindRecordById(collectionName, id)
		if err == nil || record != nil {
			t.Errorf("absorbed record %s still exists", id)
		}
	}

	// Now perform the undo operation
	err = app.RunInTransaction(func(txApp core.App) error {
		// Get the absorb action record
		record, err := txApp.FindFirstRecordByData("absorb_actions", "collection_name", collectionName)
		if err != nil {
			return err
		}
		if record == nil {
			return fmt.Errorf("no absorb action found")
		}

		// Parse the absorb action data
		var absorbedRecords []map[string]interface{}
		if err := json.Unmarshal([]byte(record.GetString("absorbed_records")), &absorbedRecords); err != nil {
			return err
		}

		// updatedRefs is nested as table -> column -> recordId -> oldValue
		var updatedRefs map[string]map[string]map[string]string
		if err := json.Unmarshal([]byte(record.GetString("updated_references")), &updatedRefs); err != nil {
			return err
		}

		collection, err := txApp.FindCollectionByNameOrId(collectionName)
		if err != nil {
			return err
		}

		// Recreate absorbed records
		for _, recordData := range absorbedRecords {
			record := core.NewRecord(collection)
			for field, value := range recordData {
				record.Set(field, value)
			}
			if err := txApp.Save(record); err != nil {
				return err
			}
		}

		// Restore references
		for table, columns := range updatedRefs {
			for column, updates := range columns {
				for recordID, oldValue := range updates {
					updateQuery := fmt.Sprintf("UPDATE %s SET %s = {:old_value} WHERE id = {:record_id}", table, column)
					_, err = txApp.NonconcurrentDB().NewQuery(updateQuery).Bind(dbx.Params{
						"old_value": oldValue,
						"record_id": recordID,
					}).Execute()
					if err != nil {
						return err
					}
				}
			}
		}

		// Delete absorb action
		if err := txApp.Delete(record); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to undo absorb: %v", err)
	}

	// Verify the undo operation restored everything correctly
	// 1. Check all records exist with original data
	for id, originalData := range initialState {
		record, err := app.FindRecordById(collectionName, id)
		if err != nil {
			t.Errorf("failed to find restored record %s: %v", id, err)
			continue
		}
		if record == nil {
			t.Errorf("restored record %s is nil", id)
			continue
		}
		for field, originalValue := range originalData {
			// skip the created and updated fields because these system fields are not
			// backed up in an absorb action
			if field == "created" || field == "updated" {
				continue
			}
			currentValue := record.Get(field)
			if !reflect.DeepEqual(currentValue, originalValue) {
				t.Errorf("field %s of record %s has value %v, want %v", field, id, currentValue, originalValue)
			}
		}
	}

	// 2. Check reference counts are restored
	for _, ref := range refConfigs {
		var result CountResult
		query := fmt.Sprintf(
			"SELECT COUNT(*) as count FROM %s WHERE %s IN (%s)",
			ref.Table,
			ref.Column,
			"'"+strings.Join(append(idsToAbsorb, targetID), "','")+"'",
		)
		err = app.DB().NewQuery(query).One(&result)
		if err != nil {
			t.Errorf("failed to get reference count for %s: %v", ref.Table, err)
			continue
		}
		key := ref.Table + "." + ref.Column
		if result.Count != initialRefCounts[key] {
			t.Errorf("reference count for %s is %d, want %d", key, result.Count, initialRefCounts[key])
		}
	}

	// 3. Verify absorb action record is deleted
	record, err := app.FindFirstRecordByData("absorb_actions", "collection_name", collectionName)
	if err != nil && err.Error() != "sql: no rows in result set" {
		t.Errorf("error checking absorb action: %v", err)
	}
	if record != nil {
		t.Error("absorb action still exists after undo")
	}
}
