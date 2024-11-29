package main

import (
	"fmt"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/routes"
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
			var initialCount int64                     // Total records in the collection
			initialRefCounts := make(map[string]int64) // References in related tables

			// For successful test cases, we need to verify the initial state
			// Error cases skip this as they'll fail before any state changes
			if !tt.wantErr {
				// Step 1: Get the initial count of records in the collection
				// This will be used later to verify that the count decreased
				// by exactly the number of absorbed records
				var result CountResult
				err := app.Dao().DB().NewQuery(fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tt.collectionName)).One(&result)
				if err != nil {
					t.Fatalf("failed to get initial count: %v", err)
				}
				initialCount = result.Count

				// Step 2: Verify the target record exists
				// This is crucial as it's the record that will absorb the others
				targetRecord, err := app.Dao().FindRecordById(tt.collectionName, tt.targetID)
				if err != nil {
					t.Fatalf("failed to find target record: %v", err)
				}
				if targetRecord == nil {
					t.Fatal("target record is nil")
				}

				// Step 3: Verify all records to be absorbed exist
				// We need to ensure they exist before trying to absorb them
				for _, id := range tt.idsToAbsorb {
					record, err := app.Dao().FindRecordById(tt.collectionName, id)
					if err != nil {
						t.Fatalf("failed to find record %s: %v", id, err)
					}
					if record == nil {
						t.Fatalf("record %s is nil", id)
					}
				}

				// Step 4: Capture the initial state of all reference tables
				// These tables contain foreign keys pointing to our records
				refConfigs, err := routes.GetConfigsAndTable(tt.collectionName)
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
					err = app.Dao().DB().NewQuery(query).One(&result)
					if err != nil {
						t.Fatalf("failed to get reference count for %s: %v", ref.Table, err)
					}
					initialRefCounts[ref.Table] = result.Count
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
				err = app.Dao().DB().NewQuery(fmt.Sprintf("SELECT COUNT(*) as count FROM %s", tt.collectionName)).One(&result)
				if err != nil {
					t.Fatalf("failed to get final count: %v", err)
				}
				expectedFinalCount := initialCount - int64(len(tt.idsToAbsorb))
				if result.Count != expectedFinalCount {
					t.Errorf("expected count %d, got %d", expectedFinalCount, result.Count)
				}

				// Step 6b: Verify all absorbed records were deleted
				for _, id := range tt.idsToAbsorb {
					record, err := app.Dao().FindRecordById(tt.collectionName, id)
					if err == nil {
						t.Errorf("expected error finding absorbed record %s", id)
					}
					if record != nil {
						t.Errorf("absorbed record %s still exists", id)
					}
				}

				// Step 6c: Verify the target record still exists
				targetRecord, err := app.Dao().FindRecordById(tt.collectionName, tt.targetID)
				if err != nil {
					t.Fatalf("failed to find target record: %v", err)
				}
				if targetRecord == nil {
					t.Fatal("target record is nil")
				}

				// Step 6d: Verify all references were properly updated
				refConfigs, err := routes.GetConfigsAndTable(tt.collectionName)
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
					err = app.Dao().DB().NewQuery(query).One(&result)
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
					err = app.Dao().DB().NewQuery(query).One(&targetResult)
					if err != nil {
						t.Fatalf("failed to check target references: %v", err)
					}
					if targetResult.Count != initialRefCounts[ref.Table] {
						t.Errorf("expected %d references in %s, got %d", initialRefCounts[ref.Table], ref.Table, targetResult.Count)
					}
				}
			}
		})
	}
}
