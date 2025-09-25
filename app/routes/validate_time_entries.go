package routes

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// This function will validate the time entries as a group. If the validation
// fails, it will return an error. If the validation passes, it will return nil.
func validateTimeEntries(txApp core.App, admin_profile *core.Record, payrollYearEndDateAsTime time.Time, entries []*core.Record) error {
	// Expand the time_type relations of the entries so we can access the
	// time_type code stored in the time_types collection.
	if errs := txApp.ExpandRecords(entries, []string{"time_type"}, nil); len(errs) > 0 {
		return &CodeError{
			Code:    "error_expanding_time_type_relations",
			Message: fmt.Sprintf("error expanding time_type relations: %v", errs),
		}
	}

	// get the weekEnding value from the first entry
	weekEnding := entries[0].GetString("week_ending")

	salary := admin_profile.GetBool("salary")
	openingDate := admin_profile.GetString("opening_date")
	openingOP := admin_profile.GetFloat("opening_op")
	openingOV := admin_profile.GetFloat("opening_ov")
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
				return &CodeError{
					Code:    "multiple_work_records",
					Message: fmt.Sprintf("work record %s appears in multiple entries", workRecord),
				}
			}
			workRecordsSet[workRecord] = true
		}

		// Access the time_type record from the expanded time_type relation
		timeType := entry.ExpandedOne("time_type")
		timeTypeCode := timeType.GetString("code")
		entryHours := entry.GetFloat("hours")

		switch timeTypeCode {
		case "OR":
			// Return an error immediately if the entry is of type OR (off rotation) and
			// the date of the entry is already in the offRotationDateSet. If it is not
			// in the set, add the date to the set.
			if _, keyPresent := offRotationDateSet[entry.GetString("date")]; keyPresent {
				return &CodeError{
					Code:    "multiple_off_rotation_entries",
					Message: fmt.Sprintf("more than one OR entry exists for the date: %s", entry.GetString("date")),
				}
			}
			offRotationDateSet[entry.GetString("date")] = true
		case "OW":
			// prevent salaried employees from claiming full week off (OW)
			if salary {
				return &CodeError{
					Code:    "salary_with_time_type_OW",
					Message: "salaried staff cannot claim full week off, use OP or OV",
				}
			}
			// Return an error immediately if the total number of entries exceeds 1 since
			// if an OW entry exists, it should be the only entry on the timesheet.
			if len(entries) > 1 {
				return &CodeError{
					Code:    "multiple_OW_entries",
					Message: "if present, an OW entry must be the only entry on a timesheet",
				}
			}
			// Return an error immediately if the entry is of type OW (off rotation week)
			// and the offRotationWeekEntryCount is greater than 1. If the entry is of
			// type OW, increment the offRotationWeekEntryCount.
			offRotationWeekEntryCount++
			if offRotationWeekEntryCount > 1 {
				return &CodeError{
					Code:    "multiple_off_rotation_week_entries",
					Message: "only one off-rotation week entry can exist on a timesheet",
				}
			}
		case "OTO":
			// If the entry is of type OTO (Request Overtime Payout), the user must
			// not be a salaried staff member.
			if salary {
				return &CodeError{
					Code:    "salary_with_time_type_OTO",
					Message: "salaried staff cannot request overtime payouts",
				}
			}
			payoutRequestCount++
			// Return an error immediately if there is more than one payout request
			// entry on the timesheet.
			if payoutRequestCount > 1 {
				return &CodeError{
					Code:    "multiple_payout_request_entries",
					Message: "only one payout request entry can exist on a timesheet",
				}
			}
		case "RB":
			// If the entry is of type RB (Add Overtime to Bank), the user must
			// not be a salaried staff member.
			if salary {
				return &CodeError{
					Code:    "salary_with_time_type_RB",
					Message: "salaried staff cannot bank overtime",
				}
			}
			// Return an error immediately if there is more than one bank entry on
			// the timesheet.
			bankEntriesCount++
			if bankEntriesCount > 1 {
				return &CodeError{
					Code:    "multiple_overtime_banking_entries",
					Message: "only one overtime banking entry can exist on a timesheet",
				}
			}
			bankedHours += entryHours
		case "R", "RT":
			// If the entry is of type R (regular) or RT (regular time), add the hours
			// to the jobHours variable if the job field is not empty.
			if jobId := entry.GetString("job"); jobId != "" {
				jobHours += entryHours
			} else {
				nonJobHours += entryHours
			}
		default:
			if entryHours == 0 {
				return &CodeError{
					Code:    "time_entry_missing_hours",
					Message: "a time entry is missing hours",
				}
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
		return &CodeError{
			Code:    "too_many_banked_hours",
			Message: "banked hours cannot bring your total worked hours below 44 hours on a timesheet",
		}
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
		return &CodeError{
			Code:    "too_much_discretionary_time_off",
			Message: fmt.Sprintf("you cannot claim OV or OP entries that increase hours beyond %v", workWeekHours),
		}
	}

	// prevent salaried employees from claiming off rotation days (OR) unless
	// permitted by admin profile.
	if salary && !offRotationPermitted && len(offRotationDateSet) > 0 {
		return &CodeError{
			Code:    "salary_with_time_type_OR_without_permission",
			Message: "salaried staff need permission to claim OR entries",
		}
	}

	// require salaried employees to have at least workWeekHours hours on a
	// timesheet unless skipMinTimeCheck is set to "yes" or "on_next_bundle"
	offRotationHours := float64(len(offRotationDateSet)) * 8
	if salary && nonJobHours+jobHours+nonWorkHoursTotal+offRotationHours < workWeekHours {
		if skipMinTimeCheck == "no" {
			return &CodeError{
				Code:    "too_few_hours_on_timesheet",
				Message: fmt.Sprintf("you must have a minimum of %v hours on your time sheet", workWeekHours),
			}
		}
	}

	// prevent salaried employees from claiming sick time by reporting an error if
	// the key "OS" (sick time) exists in the nonWorkHoursTally.
	if _, ok := nonWorkHoursTally["OS"]; ok && salary {
		return &CodeError{
			Code:    "salary_with_time_type_OS",
			Message: "salaried staff cannot claim OS. Please use OP or OV instead",
		}
	}

	// prevent salaried employees w/ skipMinTimeCheck: "yes" from claiming OB, OH,
	// OP, OV
	if salary && skipMinTimeCheck == "yes" && nonWorkHoursTotal > 0 {
		return &CodeError{
			Code:    "untracked_time_off_restricted",
			Message: "staff with untracked time off are only permitted to create R or RT entries",
		}
	}

	// return an error if openingDate is not a valid date in the format
	// "2006-01-02"
	openingDateAsTime, err := time.Parse("2006-01-02", openingDate)
	if err != nil {
		return &CodeError{
			Code:    "invalid_opening_date",
			Message: "your admin_profile has an invalid opening_date, contact support",
		}
	}

	// return an error if openingDate is not a Sunday (is this necessary?, why
	// can't it be any other day of the week?). In original Tybalt,
	// openingDateTimeOff was a Saturday at 11:59:59 PM UNLESS it hadn't yet been
	// set upon profile creation. In that case, it was the moment the profile was
	// created. Logically it makes sense to require that the opening_date is a
	// Sunday because it's the opening balance in a given payroll period and
	// Sunday is the first day of a payroll period. However since the opening
	// values for OV and OP are set to zero by default, it won't matter what the
	// opening_date is. So we will comment out this check.
	// if openingDateAsTime.Weekday() != time.Sunday {
	// 	return &CodeError{
	// 		Code:    "opening_date_not_sunday",
	// 		Message: "opening_date on your admin_profile must be a Sunday, contact support",
	// 	}
	// }

	// return an error if weekEnding is not a valid date in the format
	// "2006-01-02"
	weekEndingAsTime, err := time.Parse("2006-01-02", weekEnding)
	if err != nil {
		return &CodeError{
			Code:    "invalid_week_ending_date",
			Message: "an entry has an invalid week_ending date, contact support",
		}
	}

	// return an error if openingDate is after the weekEnding. This will prevent
	// submission of an old timesheet if the openingDateTimeOff value has already
	// been updated to the next fiscal year. This error is only triggered if PPTO
	// or Vacation are claimed on this timesheet because the opening balances are
	// otherwise irrelevant to the validation.
	if discretionaryTimeOff > 0 && openingDateAsTime.After(weekEndingAsTime) {
		return &CodeError{
			Code:    "timesheet_prior_to_opening_date",
			Message: fmt.Sprintf("your opening balances were set effective %v but you are submitting a timesheet for a prior period, contact support", openingDate),
		}
	}

	// Each timesheet submission is checked against the most recent payrollYearEndDate
	// that is less than the weekEnding of that timesheet. The actual value is
	// passed as an argument to this function. This most recent date must be less
	// than the opening_date value in the admin_profile. If it isn't,
	// then the opening balances are out of date and the timesheet cannot be
	// submitted until the opening balances are updated by accounting. This is to
	// prevent the user from claiming expired time off from a previous year on a
	// timesheet in the following year. This error is only triggered if PPTO or
	// Vacation are claimed on this timesheet.

	if discretionaryTimeOff > 0 && payrollYearEndDateAsTime.After(openingDateAsTime) {
		return &CodeError{
			Code:    "opening_balances_out_of_date",
			Message: fmt.Sprintf("your opening balances were set effective %v but you are submitting a timesheet for the time-off accounting period beginning on %v. contact accounting to have your opening balances updated for the new period prior to submitting a timesheet", openingDate, payrollYearEndDateAsTime.Format("2006-01-02")),
		}
	}

	// get the total PPTO and Vacation hours used in the period since the
	// openingDate then check if the sum of the time entries for PPTO and Vacation
	// is greater than corresponding opening values. If it is, return an
	// error.

	// usedVacation is sum of all hours for the time_entries records where
	// time_type.code is "OV" and week_ending is greater than or equal to
	// openingDate and less than or equal to weekEnding for the uid of the
	// admin_profile
	type SumResult struct {
		TotalHours float64 `db:"total_hours"`
	}
	results := []SumResult{}
	queryError := txApp.DB().NewQuery("SELECT COALESCE(SUM(hours), 0) AS total_hours FROM time_entries LEFT JOIN time_types ON time_entries.time_type = time_types.id WHERE uid = {:uid} AND week_ending >= {:openingDate} AND week_ending <= {:weekEnding} AND time_types.code = {:timeTypeCode}").Bind(dbx.Params{
		"uid":          admin_profile.Get("uid"),
		"openingDate":  openingDate,
		"weekEnding":   weekEnding,
		"timeTypeCode": "OV",
	}).All(&results)
	if queryError != nil {
		return &CodeError{
			Code:    "error_querying_for_used_vacation",
			Message: fmt.Sprintf("error querying for used vacation: %v", queryError),
		}
	}
	usedOV := 0.0
	if len(results) == 1 {
		usedOV = results[0].TotalHours
	}

	// usedPpto is sum of all hours for the time_entries records where
	// time_type.code is "OP" and week_ending is greater than or equal to
	// openingDate and less than or equal to weekEnding
	results = []SumResult{}
	queryError = txApp.DB().NewQuery("SELECT COALESCE(SUM(hours), 0) AS total_hours FROM time_entries LEFT JOIN time_types ON time_entries.time_type = time_types.id WHERE uid = {:uid} AND week_ending >= {:openingDate} AND week_ending <= {:weekEnding} AND time_types.code = {:timeTypeCode}").Bind(dbx.Params{
		"uid":          admin_profile.Get("uid"),
		"openingDate":  openingDate,
		"weekEnding":   weekEnding,
		"timeTypeCode": "OP",
	}).All(&results)
	if queryError != nil {
		return &CodeError{
			Code:    "error_querying_for_used_ppto",
			Message: fmt.Sprintf("error querying for used ppto: %v", queryError),
		}
	}
	usedOP := 0.0
	if len(results) == 1 {
		usedOP = results[0].TotalHours
	}

	// Only enforce the openingOV balance check if at least one OV entry exists on this
	// timesheet. This prevents an error when the user's available vacation balance
	// is already negative from previous activity but no additional vacation is
	// being claimed in the current bundle (see issue #25).
	if _, ovClaimed := nonWorkHoursTally["OV"]; ovClaimed && usedOV > openingOV {
		return &CodeError{
			Code:    "ov_claim_exceeds_balance",
			Message: "your vacation claim exceeds your available vacation balance",
		}
	}

	// Similarly, only enforce the PPTO balance check if at least one OP entry is
	// present on this timesheet.
	if _, opClaimed := nonWorkHoursTally["OP"]; opClaimed && usedOP > openingOP {
		return &CodeError{
			Code:    "ppto_claim_exceeds_balance",
			Message: "your PPTO claim exceeds your available PPTO balance",
		}
	}

	// return an error if OP was claimed on this timesheet and the remaining
	// available OV is greater than 0.
	if _, pptoClaimed := nonWorkHoursTally["OP"]; pptoClaimed && openingOV-usedOV > 0 {
		return &CodeError{
			Code:    "ppto_used_before_ov",
			Message: fmt.Sprintf("exhaust your vacation balance (%v hours) prior to claiming PPTO", openingOV-usedOV),
		}
	}

	// default_charge_out_rate is mandatory on the admin_profile in pocketbase
	// rules so there is no need to check for it.

	// payroll_id is mandatory on the admin_profile in pocketbase rules so there
	// is no need to check for it.

	return nil
}
