package routes

import (
	"fmt"

	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// This function will validate the time entries as a group. If the validation
// fails, it will return an error. If the validation passes, it will return nil.
func validateTimeEntries(txDao *daos.Dao, admin_profile *models.Record, entries []*models.Record) error {
	// Expand the time_type relations of the entries so we can access the
	// time_type code stored in the time_types collection.
	if errs := txDao.ExpandRecords(entries, []string{"time_type"}, nil); len(errs) > 0 {
		return fmt.Errorf("error expanding time_type relations: %v", errs)
	}

	// --------------------------------
	// Validate payout request entries
	// --------------------------------
	payoutRequests := []*models.Record{}

	for _, entry := range entries {
		// Access the code from the expanded time_type relation
		timeType := entry.ExpandedOne("time_type")
		if timeType != nil && timeType.GetString("code") == "OTO" {
			payoutRequests = append(payoutRequests, entry)
		}
	}

	if len(payoutRequests) > 1 {
		return fmt.Errorf("only one payout request entry can exist on a timesheet")
	}

	if len(payoutRequests) == 1 {
		payoutRequestAmount := payoutRequests[0].GetFloat("payout_request_amount")
		if payoutRequestAmount > 0 && admin_profile.GetBool("salary") {
			return fmt.Errorf("salaried staff cannot request overtime payouts. Please speak with management")
		}
	}

	return nil
}
