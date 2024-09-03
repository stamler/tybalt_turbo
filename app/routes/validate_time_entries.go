package routes

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
)

// This function will validate the time entries as a group. If the validation
// fails, it will return an error. If the validation passes, it will return nil.
func validateTimeEntries(txDao *daos.Dao, admin_profile *models.Record, timeOffResetDateAsTime time.Time, entries []*models.Record) error {
	// Expand the time_type relations of the entries so we can access the
	// time_type code stored in the time_types collection.
	if errs := txDao.ExpandRecords(entries, []string{"time_type"}, nil); len(errs) > 0 {
		return fmt.Errorf("error expanding time_type relations: %v", errs)
	}

	// get the weekEnding value from the first entry
	weekEnding := entries[0].GetString("week_ending")

	salary := admin_profile.GetBool("salary")
	openingDate := admin_profile.GetString("opening_date")
	offRotationPermitted := admin_profile.GetBool("off_rotation_permitted")
	skipMinTimeCheck := admin_profile.GetString("skip_min_time_check")
	workWeekHours := admin_profile.GetFloat("work_week_hours")
	workRecordsSet := map[string]bool{}
	offRotationDateSet := map[string]bool{}
	offRotationWeekEntryCount := 0
	payoutRequestCount := 0
	bankEntriesCount := 0
	bankedHours := 0.0
	jobHours := 0.0
	nonJobHours := 0.0
	nonWorkHoursTally := map[string]float64{}

	// Loop through each time entry, summarizing information as we go and
	// returning errors if any validations fail.
	for _, entry := range entries {
		// Return an error immediately if the entry has a work_record value that is
		// not an empty string and is already in the workRecordsSet. Otherwise, add
		// the work_record value to the workRecordsSet.
		if workRecord := entry.GetString("work_record"); workRecord != "" {
			if _, keyPresent := workRecordsSet[workRecord]; keyPresent {
				return fmt.Errorf("work record %s appears in multiple entries", workRecord)
			}
			workRecordsSet[workRecord] = true
		}

		// Access the time_type record from the expanded time_type relation
		timeType := entry.ExpandedOne("time_type")
		timeTypeCode := timeType.GetString("code")
		entryHours := entry.GetFloat("hours")

		if timeTypeCode == "OR" {
			// Return an error immediately if the entry is of type OR (off rotation) and
			// the date of the entry is already in the offRotationDateSet. If it is not
			// in the set, add the date to the set.
			if _, keyPresent := offRotationDateSet[entry.GetString("date")]; keyPresent {
				return fmt.Errorf("more than one Off-Rotation entry exists for the date: %s", entry.GetString("date"))
			}
			offRotationDateSet[entry.GetString("date")] = true
		} else if timeTypeCode == "OW" {
			// prevent salaried employees from claiming full week off (OW)
			if salary {
				return fmt.Errorf("salaried staff cannot claim full week off. use OP or OV")
			}
			// Return an error immediately if the total number of entries exceeds 1 since
			// if an OW entry exists, it should be the only entry on the timesheet.
			if len(entries) > 1 {
				return fmt.Errorf("if present, an OW entry must be the only entry on a timesheet")
			}
			// Return an error immediately if the entry is of type OW (off rotation week)
			// and the offRotationWeekEntryCount is greater than 1. If the entry is of
			// type OW, increment the offRotationWeekEntryCount.
			offRotationWeekEntryCount++
			if offRotationWeekEntryCount > 1 {
				return fmt.Errorf("only one off-rotation week entry can exist on a timesheet")
			}
		} else if timeTypeCode == "OTO" {
			// If the entry is of type OTO (Request Overtime Payout), the user must
			// not be a salaried staff member.
			if salary {
				return fmt.Errorf("salaried staff cannot request overtime payouts")
			}
			payoutRequestCount++
			// Return an error immediately if there is more than one payout request
			// entry on the timesheet.
			if payoutRequestCount > 1 {
				return fmt.Errorf("only one payout request entry can exist on a timesheet")
			}
		} else if timeTypeCode == "RB" {
			// If the entry is of type RB (Add Overtime to Bank), the user must
			// not be a salaried staff member.
			if salary {
				return fmt.Errorf("salaried staff cannot bank overtime")
			}
			// Return an error immediately if there is more than one bank entry on
			// the timesheet.
			bankEntriesCount++
			if bankEntriesCount > 1 {
				return fmt.Errorf("only one overtime banking entry can exist on a timesheet")
			}
			bankedHours += entryHours
		} else if timeTypeCode == "R" || timeTypeCode == "RT" {
			// If the entry is of type R (regular) or RT (regular time), add the hours
			// to the jobHours variable if the job field is not empty.
			if jobId := entry.GetString("job"); jobId != "" {
				jobHours += entryHours
			} else {
				nonJobHours += entryHours
			}
		} else {
			if entryHours == 0 {
				return fmt.Errorf("a time entry is missing hours")
			}
			// Initialize the nonWorkHoursTally for the timeTypeCode if it doesn't
			// already exist.
			if _, keyPresent := nonWorkHoursTally[timeTypeCode]; !keyPresent {
				nonWorkHoursTally[timeTypeCode] = 0
			}
			nonWorkHoursTally[timeTypeCode] += entryHours
		}
	}

	// Now we look for validation errors that apply across multiple entries.

	// If banked hours exist, the sum of all hours worked minus the banked hours
	// mustn't be under 44.
	if bankedHours > 0 && jobHours+nonJobHours-bankedHours < 44 {
		return fmt.Errorf("banked hours cannot bring your total worked hours below 44 hours on a timesheet")
	}

	// sum the values of the nonWorkHoursTally into nonWorkHoursTotal
	nonWorkHoursTotal := 0.0
	for _, hours := range nonWorkHoursTally {
		nonWorkHoursTotal += hours
	}

	// discretionaryTimeOff is the sum of the nonWorkHoursTally values for the
	// codes "OP" (PPTO) and "OV" (Vacation). These keys may not be present in
	// the nonWorkHoursTally if there are no entries of those types on the
	// timesheet so we need to check if they exist before adding them.
	discretionaryTimeOff := 0.0
	if _, ok := nonWorkHoursTally["OP"]; ok {
		discretionaryTimeOff += nonWorkHoursTally["OP"]
	}
	if _, ok := nonWorkHoursTally["OV"]; ok {
		discretionaryTimeOff += nonWorkHoursTally["OV"]
	}

	// prevent staff from using vacation or PPTO to raise their timesheet hours
	// beyond workWeekHours.
	if discretionaryTimeOff > 0 && nonJobHours+jobHours+nonWorkHoursTotal > workWeekHours {
		return fmt.Errorf("you cannot claim OV or OP entries that increase hours beyond %v", workWeekHours)
	}

	// prevent salaried employees from claiming off rotation days (OR) unless
	// permitted by admin profile.
	if salary && !offRotationPermitted && len(offRotationDateSet) > 0 {
		return fmt.Errorf("salaried staff need permission to claim OR entries")
	}

	// require salaried employees to have at least workWeekHours hours on a
	// timesheet unless skipMinTimeCheck is set to "yes" or "on_next_bundle"
	offRotationHours := float64(len(offRotationDateSet)) * 8
	if salary && nonJobHours+jobHours+nonWorkHoursTotal+offRotationHours < workWeekHours {
		if skipMinTimeCheck == "no" {
			return fmt.Errorf("you must have a minimum of %v hours on your time sheet", workWeekHours)
		}
	}

	// prevent salaried employees from claiming sick time by reporting an error if
	// the key "OS" (sick time) exists in the nonWorkHoursTally.
	if _, ok := nonWorkHoursTally["OS"]; ok && salary {
		return fmt.Errorf("salaried staff cannot claim OS. Please use OP or OV instead")
	}

	// prevent salaried employees w/ skipMinTimeCheck: "yes" from claiming OB, OH,
	// OP, OV
	if salary && skipMinTimeCheck == "yes" && nonWorkHoursTotal > 0 {
		return fmt.Errorf("staff with untracked time off are only permitted to create R or RT entries")
	}

	// return an error if openingDate is not a valid date in the format
	// "2006-01-02"
	openingDateAsTime, err := time.Parse("2006-01-02", openingDate)
	if err != nil {
		return fmt.Errorf("your admin_profile has an invalid opening_date, contact support")
	}

	// return an error if weekEnding is not a valid date in the format
	// "2006-01-02"
	weekEndingAsTime, err := time.Parse("2006-01-02", weekEnding)
	if err != nil {
		return fmt.Errorf("an entry has an invalid week_ending date, contact support")
	}

	// return an error if openingDate is after the weekEnding. This will prevent
	// submission of an old timesheet if the openingDateTimeOff value has already
	// been updated to the next fiscal year. This error is only triggered if PPTO
	// or Vacation are claimed on this timesheet because the opening balances are
	// otherwise irrelevant to the validation.
	if discretionaryTimeOff > 0 && openingDateAsTime.After(weekEndingAsTime) {
		return fmt.Errorf("your opening balances were set effective %v but you are submitting a timesheet for a prior period, contact support", openingDate)
	}

	// Each timesheet submission is checked against the most timeOffResetDate that
	// is less than the weekEnding of that timesheet. The actual value is passed
	// as an argument to this function. This most recent date must be less than
	// or equal to the opening_date value in the admin_profile. If it isn't, then
	// the opening balances are out of date and the timesheet cannot be submitted
	// until the opening balances are updated by accounting. This is to prevent
	// the user from claiming expired time off from a previous year on a timesheet
	// in the following year. This error is only triggered if PPTO or Vacation are
	// claimed on this timesheet.

	if discretionaryTimeOff > 0 && timeOffResetDateAsTime.After(openingDateAsTime) {
		//      `Your opening balances were set effective ${opdate.toISOString()} but you are submitting a timesheet for the time-off accounting period beginning on ${mostRecentResetDate.toDate().toISOString()}. Please contact accounting to have your opening balances updated for the new period prior to submitting a timesheet for this period.`

		return fmt.Errorf("your opening balances were set effective %v but you are submitting a timesheet for the time-off accounting period beginning on %v. contact accounting to have your opening balances updated for the new period prior to submitting a timesheet", openingDate, timeOffResetDateAsTime.Format("2006-01-02"))
	}

	// TODO continue from Line 346 in tallyAndValidate.ts
	return nil
}
