package routes

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
)

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
