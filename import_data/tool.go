package main

import (
	"flag"
	"fmt"
	"imports/extract"
	"imports/load"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
)

// This file is used to run either an export or an import.

func main() {
	// Parse command line arguments
	exportFlag := flag.Bool("export", false, "Export data to Parquet files")
	importFlag := flag.Bool("import", false, "Import data from Parquet files")
	flag.Parse()

	if *exportFlag {
		fmt.Println("Exporting data to Parquet files...")
		extract.ToParquet()
	}

	if *importFlag {
		fmt.Println("Importing data from Parquet files...")

		// --- Load Clients ---
		// Define the specific SQL for the clients table
		clientInsertSQL := "INSERT INTO clients (id, name) VALUES ({:id}, {:name})"

		// Define the binder function for the Client type
		clientBinder := func(item load.Client) dbx.Params {
			return dbx.Params{
				"id":   item.Id,
				"name": item.Name,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Clients.parquet",
			"../app/test_pb_data/data.db",
			"clients",       // Table name (for logging)
			clientInsertSQL, // The specific INSERT SQL
			clientBinder,    // The specific binder function
		)

		// --- Load Contacts ---
		// Define the specific SQL for the contacts table
		contactInsertSQL := "INSERT INTO client_contacts (id, surname, given_name, client) VALUES ({:id}, {:surname}, {:given_name}, {:client})"

		// Define the binder function for the Contact type
		contactBinder := func(item load.ClientContact) dbx.Params {
			return dbx.Params{
				"id":         item.Id,
				"surname":    item.Surname,
				"given_name": item.GivenName,
				"client":     item.Client,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Contacts.parquet",
			"../app/test_pb_data/data.db",
			"client_contacts", // Table name (for logging)
			contactInsertSQL,  // The specific INSERT SQL
			contactBinder,     // The specific binder function
		)

		// --- Load Users ---
		// Define the specific SQL for the users table
		userInsertSQL := "INSERT INTO users (id, email, username, name, emailVisibility, verified, password, tokenKey) VALUES ({:id}, {:email}, {:username}, {:name}, 0, 1, {:password}, {:tokenKey})"

		// Define the binder function for the User type
		userBinder := func(item load.Profile) dbx.Params {
			return dbx.Params{
				"id":       item.UserId,
				"email":    item.Email,
				"username": strings.Split(item.Email, "@")[0],
				"name":     item.GivenName + " " + item.Surname,
				"password": "",                            // ************ TODO: What should this be?
				"tokenKey": fmt.Sprintf("%x", uuid.New()), // ************ TODO: What should this be?
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Profiles.parquet",
			"../app/test_pb_data/data.db",
			"users",       // Table name (for logging)
			userInsertSQL, // The specific INSERT SQL
			userBinder,    // The specific binder function
		)

		// --- Load Jobs ---
		// Define the specific SQL for the jobs table
		// TODO: categories
		jobInsertSQL := "INSERT INTO jobs (id, number, description, client, contact, manager, alternate_manager, fn_agreement, status, project_award_date, proposal_opening_date, proposal_submission_due_date, proposal, divisions, job_owner) VALUES ({:id}, {:number}, {:description}, {:client}, {:contact}, {:manager}, {:alternate_manager}, {:fn_agreement}, {:status}, {:project_award_date}, {:proposal_opening_date}, {:proposal_submission_due_date}, {:proposal}, {:divisions}, {:job_owner})"

		// Define the binder function for the Job type
		jobBinder := func(item load.Job) dbx.Params {
			return dbx.Params{
				"id":                           item.Id,
				"number":                       item.Number,
				"description":                  item.Description,
				"client":                       item.Client,
				"contact":                      item.Contact,
				"manager":                      item.Manager,
				"alternate_manager":            item.AlternateManagerId,
				"fn_agreement":                 item.FnAgreement,
				"status":                       item.Status,
				"project_award_date":           item.ProjectAwardDate,
				"proposal_opening_date":        item.ProposalOpeningDate,
				"proposal_submission_due_date": item.ProposalSubmissionDueDate,
				"proposal":                     item.ProposalId,
				"divisions":                    item.DivisionsIds,
				"job_owner":                    item.JobOwnerId,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Jobs.parquet",
			"../app/test_pb_data/data.db",
			"jobs",       // Table name (for logging)
			jobInsertSQL, // The specific INSERT SQL
			jobBinder,    // The specific binder function
		)

		// --- Load Categories ---
		// Define the specific SQL for the categories table
		categoryInsertSQL := "INSERT INTO categories (id, name, job) VALUES ({:id}, {:name}, {:job})"

		// Define the binder function for the Category type
		categoryBinder := func(item load.Category) dbx.Params {
			return dbx.Params{
				"id":   item.Id,
				"name": item.Name,
				"job":  item.Job,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Categories.parquet",
			"../app/test_pb_data/data.db",
			"categories",      // Table name (for logging)
			categoryInsertSQL, // The specific INSERT SQL
			categoryBinder,    // The specific binder function
		)

		// --- Load Admin Profiles ---
		// Define the specific SQL for the admin profiles table
		// default_charge_out_rate, opening_ov are type Decimal and so need to be case to float then divided by 100 (Decimal 6,2), opening_op needs to be divided by 10 (Decimal 5,1)
		// if work_week_hours is 0, set it to 40
		// default_charge_out_rate should be set to 50 if it is 0
		adminProfileInsertSQL := "INSERT INTO admin_profiles (uid, work_week_hours, salary, default_charge_out_rate, off_rotation_permitted, skip_min_time_check, opening_date, opening_op, opening_ov, payroll_id, untracked_time_off, time_sheet_expected, allow_personal_reimbursement, mobile_phone, job_title, personal_vehicle_insurance_expiry) VALUES ({:uid}, IIF({:work_week_hours} = 0, 40, {:work_week_hours}), {:salary}, IIF({:default_charge_out_rate} = 0, 50, CAST({:default_charge_out_rate} AS REAL) / 100), {:off_rotation_permitted}, IIF({:skip_min_time_check} IS false, 'no', 'on_next_bundle'), {:opening_date}, CAST({:opening_op} AS REAL) / 10, CAST({:opening_ov} AS REAL) / 100, {:payroll_id}, {:untracked_time_off}, {:time_sheet_expected}, {:allow_personal_reimbursement}, {:mobile_phone}, {:job_title}, {:personal_vehicle_insurance_expiry})"

		// Define the binder function for the Admin type
		adminProfileBinder := func(item load.Profile) dbx.Params {
			return dbx.Params{
				"uid":                               item.UserId,
				"work_week_hours":                   item.WorkWeekHours,
				"salary":                            item.Salary,
				"default_charge_out_rate":           item.DefaultChargeOutRate,
				"off_rotation_permitted":            item.OffRotationPermitted,
				"skip_min_time_check":               item.SkipMinTimeCheckOnNextBundle,
				"opening_date":                      item.OpeningDateTimeOff,
				"opening_op":                        item.OpeningOP,
				"opening_ov":                        item.OpeningOV,
				"payroll_id":                        item.PayrollId,
				"untracked_time_off":                item.UntrackedTimeOff,
				"time_sheet_expected":               item.TimeSheetExpected,
				"allow_personal_reimbursement":      item.AllowPersonalReimbursement,
				"mobile_phone":                      item.MobilePhone,
				"job_title":                         item.JobTitle,
				"personal_vehicle_insurance_expiry": item.PersonalVehicleInsuranceExpiry,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Profiles.parquet",
			"../app/test_pb_data/data.db",
			"admin_profiles",      // Table name (for logging)
			adminProfileInsertSQL, // The specific INSERT SQL
			adminProfileBinder,    // The specific binder function
		)

		// --- Load Profiles ---
		// Define the specific SQL for the profiles table
		profileInsertSQL := "INSERT INTO profiles (surname, given_name, manager, alternate_manager, default_division, uid, notification_type, do_not_accept_submissions) VALUES ({:surname}, {:given_name}, {:manager}, {:alternate_manager}, {:default_division}, {:uid}, 'email_text', {:do_not_accept_submissions})"

		// Define the binder function for the Profile type
		profileBinder := func(item load.Profile) dbx.Params {
			return dbx.Params{
				"surname":                   item.Surname,
				"given_name":                item.GivenName,
				"manager":                   item.ManagerId,
				"alternate_manager":         item.AlternateManager,
				"default_division":          item.DefaultDivision,
				"uid":                       item.UserId,
				"do_not_accept_submissions": item.DoNotAcceptSubmissions,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Profiles.parquet",
			"../app/test_pb_data/data.db",
			"profiles",       // Table name (for logging)
			profileInsertSQL, // The specific INSERT SQL
			profileBinder,    // The specific binder function
		)

		// --- Load _externalAuths ---
		// Define the specific SQL for the _externalAuths table
		externalAuthInsertSQL := "INSERT INTO _externalAuths (collectionRef, provider, providerId, recordRef) VALUES ('_pb_users_auth_', 'microsoft', {:providerId}, {:recordRef})"

		// Define the binder function for the ExternalAuth type
		externalAuthBinder := func(item load.Profile) dbx.Params {
			return dbx.Params{
				"providerId": item.AzureId,
				"recordRef":  item.UserId,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Profiles.parquet",
			"../app/test_pb_data/data.db",
			"_externalAuths",      // Table name (for logging)
			externalAuthInsertSQL, // The specific INSERT SQL
			externalAuthBinder,    // The specific binder function
		)

		// --- Load TimeSheets ---
		// create a time value for the committed and approved fields
		now := time.Now()
		// format the time value as a string like 2024-10-18 12:00:00.000Z
		nowString := now.Format("2006-01-02 15:04:05.000Z")
		// Define the specific SQL for the time_sheets table
		timeSheetInsertSQL := "INSERT INTO time_sheets (id, uid, work_week_hours, salary, week_ending, submitted, approver, approved, committed, committer, payroll_id) VALUES ({:id}, {:uid}, {:work_week_hours}, {:salary}, {:week_ending}, 1, {:approver}, {:approved}, {:committed}, {:committer}, {:payroll_id})"

		// Define the binder function for the TimeSheet type
		timeSheetBinder := func(item load.TimeSheet) dbx.Params {
			return dbx.Params{
				"id":              item.Id,
				"uid":             item.Uid,
				"approver":        item.ApproverUid,
				"work_week_hours": item.WorkWeekHours,
				"payroll_id":      item.PayrollId,
				"salary":          item.Salary,
				"week_ending":     item.WeekEnding,
				"submitted":       nowString,
				"approved":        nowString,
				"committed":       nowString,
				"committer":       "wegviunlyr2jjjv", // a temporary value that works for the test database
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/TimeSheets.parquet",
			"../app/test_pb_data/data.db",
			"time_sheets",      // Table name (for logging)
			timeSheetInsertSQL, // The specific INSERT SQL
			timeSheetBinder,    // The specific binder function
		)

		// --- Load TimeEntries ---
		// Define the specific SQL for the time_entries table
		// hours, job_hours, and meals_hours are type Decimal and so needs to be cast to a float then divided by 10 (Decimal 3,1)
		// we sum hours and job_hours to get the total hours since after June 11, 2021 the job_hours and hours fields were mutually exclusive
		// and job_hours were only allowed to be non-zero if a job was selected. This destroys information about the split of hours prior to that date.
		timeEntryInsertSQL := `
			INSERT INTO time_entries (
				division,
				uid,
				hours,
				description,
				time_type,
				meals_hours,
				job,
				work_record,
				payout_request_amount,
				date,
				week_ending,
				tsid,
				category
			) VALUES (
			 	{:division}, 
				{:uid}, 
				CAST((COALESCE({:job_hours}, 0) + COALESCE({:hours}, 0)) AS REAL) / 10, 
				{:description}, 
				{:time_type}, 
				CAST({:meals_hours} AS REAL) / 10, 
				{:job}, 
				{:work_record}, 
				{:payout_request_amount}, 
				{:date}, 
				{:week_ending}, 
				{:tsid}, 
				{:category}
			)`

		// Define the binder function for the TimeEntry type
		// TODO: FIX SEVERAL ISSUES WITH LOADING OF TIME ENTRIES, INCLUDING BUT NOT LIMITED TO CATEGORY NOT BEING A RELATION
		timeEntryBinder := func(item load.TimeEntry) dbx.Params {
			return dbx.Params{
				"division":              item.Division,
				"uid":                   item.UserId,
				"hours":                 item.Hours,
				"description":           item.Description,
				"time_type":             item.TimeType,
				"meals_hours":           item.MealsHours,
				"job":                   item.Job,
				"work_record":           item.WorkRecord,
				"payout_request_amount": item.PayoutRequestAmount,
				"date":                  item.Date,
				"week_ending":           item.WeekEnding,
				"tsid":                  item.TimeSheet,
				"category":              item.Category,
				"job_hours":             item.JobHours,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/TimeEntries.parquet",
			"../app/test_pb_data/data.db",
			"time_entries",     // Table name (for logging)
			timeEntryInsertSQL, // The specific INSERT SQL
			timeEntryBinder,    // The specific binder function
		)
	}
}
