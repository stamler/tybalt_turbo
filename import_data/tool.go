package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"imports/attachments"
	"imports/extract"
	"imports/load"
	"log"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/marcboeker/go-duckdb" // DuckDB driver (blank import for side-effect registration)
	"github.com/pocketbase/dbx"
	_ "modernc.org/sqlite" // SQLite driver for deletion cleanup
)

var expenseCollectionId = "o1vpz1mm7qsfoyy"
var targetDatabase = "../app/test_pb_data/data.db"

// This file is used to run either an export or an import.

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Parse command line arguments
	exportFlag := flag.Bool("export", false, "Export data to Parquet files")
	importFlag := flag.Bool("import", false, "Import data from Parquet files")
	attachmentsFlag := flag.Bool("attachments", false, "Import attachments from GCS to S3")
	cleanupFlag := flag.Bool("cleanup", false, "Clean up deleted records after import")
	dbFlag := flag.String("db", "../app/test_pb_data/data.db", "Path to the target database")
	flag.Parse()

	// Use the database path from the flag
	targetDatabase = *dbFlag

	if *exportFlag {
		fmt.Println("Exporting data to Parquet files...")
		extract.ToParquet(targetDatabase)
	}

	if *importFlag {
		fmt.Println("Importing data from Parquet files...")

		// --- Load Clients ---
		// Define the specific SQL for the clients table
		clientInsertSQL := "INSERT INTO clients (id, name, _imported) VALUES ({:id}, {:name}, true)"

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
			targetDatabase,
			"clients",       // Table name (for logging)
			clientInsertSQL, // The specific INSERT SQL
			clientBinder,    // The specific binder function
			true,            // Enable upsert for idempotency
		)

		// --- Load Contacts ---
		// Define the specific SQL for the contacts table
		contactInsertSQL := "INSERT INTO client_contacts (id, surname, given_name, client, _imported) VALUES ({:id}, {:surname}, {:given_name}, {:client}, true)"

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
			targetDatabase,
			"client_contacts", // Table name (for logging)
			contactInsertSQL,  // The specific INSERT SQL
			contactBinder,     // The specific binder function
			true,              // Enable upsert for idempotency
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
			targetDatabase,
			"users",       // Table name (for logging)
			userInsertSQL, // The specific INSERT SQL
			userBinder,    // The specific binder function
			true,          // Enable upsert for idempotency
		)

		// --- Load Jobs ---
		// Define the specific SQL for the jobs table
		//
		// IMPORTANT: This INSERT OR REPLACE operates on the `number` field, NOT the `id` field!
		// The jobs table has a unique constraint on `number` (job number like "24-321").
		// When a job with an existing number is imported:
		//   1. SQLite detects unique constraint violation on `number` field
		//   2. OR REPLACE triggers, replacing the ENTIRE existing row
		//   3. The `id` field gets updated to the new pocketbase_id from MySQL export
		//   4. All other fields get updated with fresh MySQL data
		//   5. `_imported` flag gets set to true, marking it as MySQL-sourced data
		//
		// This means:
		//   - Job number remains stable (business key)
		//   - Internal ID changes to maintain consistency with current MySQL export
		//   - Local modifications to imported jobs get overwritten with MySQL data
		//   - Related records (time entries, etc.) work correctly since they also get updated IDs
		//
		jobInsertSQL := "INSERT INTO jobs (id, number, description, client, contact, manager, alternate_manager, fn_agreement, status, project_award_date, proposal_opening_date, proposal_submission_due_date, proposal, divisions, job_owner, _imported) VALUES ({:id}, {:number}, {:description}, {:client}, {:contact}, {:manager}, {:alternate_manager}, {:fn_agreement}, {:status}, {:project_award_date}, {:proposal_opening_date}, {:proposal_submission_due_date}, {:proposal}, {:divisions}, {:job_owner}, true)"

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
			targetDatabase,
			"jobs",       // Table name (for logging)
			jobInsertSQL, // The specific INSERT SQL
			jobBinder,    // The specific binder function
			true,         // Enable upsert for idempotency
		)

		// --- Load Categories ---
		// Define the specific SQL for the categories table
		categoryInsertSQL := "INSERT INTO categories (id, name, job, _imported) VALUES ({:id}, {:name}, {:job}, true)"

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
			targetDatabase,
			"categories",      // Table name (for logging)
			categoryInsertSQL, // The specific INSERT SQL
			categoryBinder,    // The specific binder function
			true,              // Enable upsert for idempotency
		)

		// --- Load Admin Profiles ---
		// Define the specific SQL for the admin profiles table
		// default_charge_out_rate, opening_ov are type Decimal and so need to be case to float then divided by 100 (Decimal 6,2), opening_op needs to be divided by 10 (Decimal 5,1)
		// if work_week_hours is 0, set it to 40
		// default_charge_out_rate should be set to 50 if it is 0
		adminProfileInsertSQL := "INSERT INTO admin_profiles (uid, work_week_hours, salary, default_charge_out_rate, off_rotation_permitted, skip_min_time_check, opening_date, opening_op, opening_ov, payroll_id, untracked_time_off, time_sheet_expected, allow_personal_reimbursement, mobile_phone, job_title, personal_vehicle_insurance_expiry, default_branch, _imported) VALUES ({:uid}, IIF({:work_week_hours} = 0, 40, {:work_week_hours}), {:salary}, IIF({:default_charge_out_rate} = 0, 50, CAST({:default_charge_out_rate} AS REAL) / 100), {:off_rotation_permitted}, IIF({:skip_min_time_check} IS false, 'no', 'on_next_bundle'), {:opening_date}, CAST({:opening_op} AS REAL) / 10, CAST({:opening_ov} AS REAL) / 100, {:payroll_id}, {:untracked_time_off}, {:time_sheet_expected}, {:allow_personal_reimbursement}, {:mobile_phone}, {:job_title}, {:personal_vehicle_insurance_expiry}, {:default_branch}, true)"

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
				"default_branch":                    item.DefaultBranch,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Profiles.parquet",
			targetDatabase,
			"admin_profiles",      // Table name (for logging)
			adminProfileInsertSQL, // The specific INSERT SQL
			adminProfileBinder,    // The specific binder function
			true,                  // Enable upsert for idempotency
		)

		// --- Load Profiles ---
		// Define the specific SQL for the profiles table
		profileInsertSQL := "INSERT INTO profiles (surname, given_name, manager, alternate_manager, default_division, uid, notification_type, do_not_accept_submissions, _imported) VALUES ({:surname}, {:given_name}, {:manager}, {:alternate_manager}, {:default_division}, {:uid}, 'email_text', {:do_not_accept_submissions}, true)"

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
			targetDatabase,
			"profiles",       // Table name (for logging)
			profileInsertSQL, // The specific INSERT SQL
			profileBinder,    // The specific binder function
			true,             // Enable upsert for idempotency
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
			targetDatabase,
			"_externalAuths",      // Table name (for logging)
			externalAuthInsertSQL, // The specific INSERT SQL
			externalAuthBinder,    // The specific binder function
			true,                  // Enable upsert for idempotency
		)

		// --- Load TimeSheets ---
		// Use a fixed timestamp for idempotency instead of dynamic time.Now()
		fixedTimestamp := "2025-05-30 00:00:00.000Z"
		// Define the specific SQL for the time_sheets table
		timeSheetInsertSQL := "INSERT INTO time_sheets (id, uid, work_week_hours, salary, week_ending, submitted, approver, approved, committed, committer, payroll_id, _imported) VALUES ({:id}, {:uid}, {:work_week_hours}, {:salary}, {:week_ending}, 1, {:approver}, {:approved}, {:committed}, {:committer}, {:payroll_id}, true)"

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
				"submitted":       fixedTimestamp,
				"approved":        fixedTimestamp,
				"committed":       fixedTimestamp,
				"committer":       "wegviunlyr2jjjv", // a temporary value that works for the test database
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/TimeSheets.parquet",
			targetDatabase,
			"time_sheets",      // Table name (for logging)
			timeSheetInsertSQL, // The specific INSERT SQL
			timeSheetBinder,    // The specific binder function
			true,               // Enable upsert for idempotency
		)

		// --- Load TimeEntries ---
		// Define the specific SQL for the time_entries table
		// hours, job_hours, and meals_hours are type Decimal and so needs to be cast to a float then divided by 10 (Decimal 3,1)
		// we sum hours and job_hours to get the total hours since after June 11, 2021 the job_hours and hours fields were mutually exclusive
		// and job_hours were only allowed to be non-zero if a job was selected. This destroys information about the split of hours prior to that date.
		timeEntryInsertSQL := `
			INSERT INTO time_entries (
				id,
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
				category,
				_imported
			) VALUES (
				{:id},
			 	{:division}, 
				{:uid},
				CAST((COALESCE({:job_hours}, 0) + COALESCE({:hours}, 0)) AS REAL) / 10, 
				{:description}, 
				{:time_type}, 
				CAST({:meals_hours} AS REAL) / 10, 
				{:job}, 
				{:work_record}, 
				CAST({:payout_request_amount} AS REAL) / 100, 
				{:date}, 
				{:week_ending}, 
				{:tsid}, 
				{:category},
				true
			)`

		// Define the binder function for the TimeEntry type
		timeEntryBinder := func(item load.TimeEntry) dbx.Params {
			return dbx.Params{
				"id":                    item.Id,
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
			targetDatabase,
			"time_entries",     // Table name (for logging)
			timeEntryInsertSQL, // The specific INSERT SQL
			timeEntryBinder,    // The specific binder function
			true,               // Enable upsert for idempotency
		)

		// --- Load TimeAmendments ---
		// Define the specific SQL for the time_amendments table
		timeAmendmentInsertSQL := `
			INSERT INTO time_amendments (
				id,
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
				category,
				creator,
				committed,
				committer,
				committed_week_ending,
				skip_tsid_check,
				_imported
			) VALUES (
				{:id},
				{:division},
				{:uid},
				CAST((COALESCE({:job_hours}, 0) + COALESCE({:hours}, 0)) AS REAL) / 10,
				{:description},
				{:time_type},
				CAST({:meals_hours} AS REAL) / 10, 
				{:job},
				{:work_record},
				CAST({:payout_request_amount} AS REAL) / 100, 
				{:date},
				{:week_ending},
				{:tsid},
				{:category},
				{:creator},
				{:committed},
				{:committer},
				{:committed_week_ending},
				false,
				true
			)`

		// Define the binder function for the TimeAmendment type
		timeAmendmentBinder := func(item load.TimeAmendment) dbx.Params {
			return dbx.Params{
				"id":                    item.Id,
				"division":              item.Division,
				"uid":                   item.User,
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
				"creator":               item.Creator,
				"committed":             item.Committed.Format("2006-01-02 15:04:05.000Z"),
				"committer":             item.Committer,
				"committed_week_ending": item.CommittedWeekEnding,
				"job_hours":             item.JobHours,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/TimeAmendments.parquet",
			targetDatabase,
			"time_amendments",      // Table name (for logging)
			timeAmendmentInsertSQL, // The specific INSERT SQL
			timeAmendmentBinder,    // The specific binder function
			true,                   // Enable upsert for idempotency
		)

		// --- Load Vendors ---
		// Define the specific SQL for the vendors table
		vendorInsertSQL := "INSERT INTO vendors (id, name, status, _imported) VALUES ({:id}, {:name}, {:status}, true)"

		// Define the binder function for the Vendor type
		vendorBinder := func(item load.Vendor) dbx.Params {
			return dbx.Params{
				"id":     item.Id,
				"name":   item.Name,
				"status": "Active",
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Vendors.parquet",
			targetDatabase,
			"vendors",       // Table name (for logging)
			vendorInsertSQL, // The specific INSERT SQL
			vendorBinder,    // The specific binder function
			true,            // Enable upsert for idempotency
		)

		// --- Load Purchase Orders ---
		// Define the specific SQL for the purchase_orders table
		load.FromParquet(
			"./parquet/purchase_orders.parquet",
			targetDatabase,
			"purchase_orders", // Table name (for logging)
			`INSERT INTO purchase_orders (id, po_number, approved, second_approval, closed, type, status, closed_by_system, description, _imported) VALUES ({:id}, {:po_number}, {:approved}, {:second_approval}, {:closed}, 'Normal', 'Closed', 1, 'Imported from Firebase Expenses', true)`,
			func(item load.PurchaseOrder) dbx.Params {
				return dbx.Params{
					"id":              item.Id,
					"po_number":       item.PoNumber,
					"approved":        fixedTimestamp,
					"second_approval": fixedTimestamp,
					"closed":          fixedTimestamp,
				}
			},
			true, // Enable upsert for idempotency
		)

		// --- Load Expenses ---
		// Define the specific SQL for the expenses table
		expenseInsertSQL := `INSERT INTO expenses (
			id,
			uid,
			division,
			job,
			category,
			date,
			pay_period_ending,
			description,
			vendor,
			distance,
			total,
			payment_type,
			attachment,
			cc_last_4_digits,
			allowance_types,
			purchase_order,
			submitted,
			approver,
			approved,
			committer,
			committed,
			committed_week_ending,
			_imported
		) VALUES (
			{:id},
			{:uid},
			{:division},
			{:job},
			{:category},
			{:date},
			{:pay_period_ending},
			{:description},
			{:vendor},
			{:distance},
			CAST({:total} AS REAL) / 100,
			{:payment_type},
			{:attachment},
			{:cc_last_4_digits},
			{:allowance_types},
			{:purchase_order},
			true,
			{:approver},
			{:approved},
			{:committer},
			{:committed},
			{:committed_week_ending},
			true
		)`

		// allowance_types is a json array of strings that are the types of
		// allowances that are allowed for the expense. Choices are "Breakfast",
		// "Lunch", "Dinner", "Lodging". If the expense is not an allowance, the
		// array is empty. Here we create the json array from the boolean fields.
		createAllowanceTypes := func(item load.Expense) string {
			allowanceTypes := []string{}
			if item.Breakfast {
				allowanceTypes = append(allowanceTypes, "Breakfast")
			}
			if item.Lunch {
				allowanceTypes = append(allowanceTypes, "Lunch")
			}
			if item.Dinner {
				allowanceTypes = append(allowanceTypes, "Dinner")
			}
			if item.Lodging {
				allowanceTypes = append(allowanceTypes, "Lodging")
			}
			jsonArray, err := json.Marshal(allowanceTypes)
			if err != nil {
				log.Fatal(err)
			}
			return string(jsonArray)
		}

		// Define the binder function for the Expense type
		expenseBinder := func(item load.Expense) dbx.Params {
			return dbx.Params{
				"id":                item.Id,
				"purchase_order":    item.PurchaseOrderId,
				"uid":               item.Uid,
				"division":          item.Division,
				"job":               item.Job,
				"category":          item.Category,
				"date":              item.Date,
				"pay_period_ending": item.PayPeriodEnding,
				"description":       item.Description,
				"vendor":            item.Vendor,
				"distance":          item.Distance,
				"allowance_types":   createAllowanceTypes(item),
				"total":             item.Total,
				"payment_type":      item.PaymentType,
				"attachment": func() string {
					if item.Attachment == "" {
						return "" // otherwise path.Base will return "."
					}
					return path.Base(item.Attachment)
				}(),
				"cc_last_4_digits":      item.CCLast4Digits,
				"approver":              item.Approver,
				"approved":              item.Committed.Format("2006-01-02 15:04:05.000Z"),
				"committer":             item.Committer,
				"committed":             item.Committed.Format("2006-01-02 15:04:05.000Z"),
				"committed_week_ending": item.CommittedWeekEnding,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Expenses.parquet",
			targetDatabase,
			"expenses",       // Table name (for logging)
			expenseInsertSQL, // The specific INSERT SQL
			expenseBinder,    // The specific binder function
			true,             // Enable upsert for idempotency
		)

		// --- Load User Claims ---
		// Define the specific SQL for the user_claims table
		userClaimInsertSQL := `INSERT INTO user_claims (uid, cid, _imported) VALUES ({:uid}, {:cid}, true)`

		// Define the binder function for the UserClaim type
		userClaimBinder := func(item load.UserClaim) dbx.Params {
			return dbx.Params{
				"uid": item.Uid,
				"cid": item.Cid,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/UserClaims.parquet",
			targetDatabase,
			"user_claims",      // Table name (for logging)
			userClaimInsertSQL, // The specific INSERT SQL
			userClaimBinder,    // The specific binder function
			true,               // Enable upsert for idempotency
		)

		// --- Load MileageResetDates ---
		// Define the specific SQL for the mileage_reset_dates table
		load.FromParquet(
			"./parquet/MileageResetDates.parquet",
			targetDatabase,
			"mileage_reset_dates", // Table name (for logging)
			`INSERT INTO mileage_reset_dates (id, date, _imported) VALUES ({:id}, {:date}, true)`,
			func(item load.MileageResetDate) dbx.Params {
				return dbx.Params{
					"id":   item.Id,
					"date": item.Date,
				}
			},
			true, // Enable upsert for idempotency
		)

		// Automatically run cleanup after import
		if *cleanupFlag {
			fmt.Println("Cleaning up deleted records...")
			cleanupDeletedRecords()
		}
	}

	if *cleanupFlag && !*importFlag {
		fmt.Println("Cleaning up deleted records...")
		cleanupDeletedRecords()
	}

	if *attachmentsFlag {
		attachments.MigrateAttachments("./parquet/Expenses.parquet", "attachment", "destination_attachment", expenseCollectionId)
	}
}

// cleanupDeletedRecords removes imported records that no longer exist in the current MySQL export
func cleanupDeletedRecords() {
	// Open connection to SQLite database
	db, err := sql.Open("sqlite", targetDatabase)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Define collections to clean up
	cleanupTasks := []struct {
		tableName   string
		parquetFile string
		idField     string // field name in parquet that contains the identifier to compare
		dbKeyColumn string // column in the sqlite table to compare against
	}{
		// Tables where the business/record key in parquet maps to the same DB column name
		{"clients", "./parquet/Clients.parquet", "id", "id"},
		{"client_contacts", "./parquet/Contacts.parquet", "id", "id"},
		{"categories", "./parquet/Categories.parquet", "id", "id"},
		{"vendors", "./parquet/Vendors.parquet", "id", "id"},
		{"purchase_orders", "./parquet/purchase_orders.parquet", "id", "id"},

		// Tables where DB primary key is populated from pocketbase_id in parquet
		{"jobs", "./parquet/Jobs.parquet", "pocketbase_id", "id"},
		{"expenses", "./parquet/Expenses.parquet", "pocketbase_id", "id"},
		{"time_sheets", "./parquet/TimeSheets.parquet", "pocketbase_id", "id"},
		{"time_entries", "./parquet/TimeEntries.parquet", "pocketbase_id", "id"},
		{"time_amendments", "./parquet/TimeAmendments.parquet", "pocketbase_id", "id"},
		{"mileage_reset_dates", "./parquet/MileageResetDates.parquet", "pocketbase_id", "id"},

		// Profiles use uid as the stable key in the DB, mapped from pocketbase_uid in parquet
		{"profiles", "./parquet/Profiles.parquet", "pocketbase_uid", "uid"},
		{"admin_profiles", "./parquet/Profiles.parquet", "pocketbase_uid", "uid"},

		// user_claims uses uid+cid composite key, handled separately below
		{"user_claims", "./parquet/UserClaims.parquet", "uid", "uid"},
	}

	for _, task := range cleanupTasks {
		fmt.Printf("Cleaning up %s...\n", task.tableName)

		// Special handling for user_claims which has composite key
		if task.tableName == "user_claims" {
			cleanupUserClaims(db)
			continue
		}

		// Get current IDs from Parquet file
		currentIDs, err := getIDsFromParquet(task.parquetFile, task.idField)
		if err != nil {
			log.Printf("Error getting IDs from %s: %v", task.parquetFile, err)
			continue
		}

		// Delete records that are imported but not in currentIDs
		deletedCount, err := deleteOrphanedRecords(db, task.tableName, task.dbKeyColumn, currentIDs)
		if err != nil {
			log.Printf("Error cleaning up %s: %v", task.tableName, err)
			continue
		}

		if deletedCount > 0 {
			fmt.Printf("  Deleted %d orphaned records from %s\n", deletedCount, task.tableName)
		} else {
			fmt.Printf("  No orphaned records found in %s\n", task.tableName)
		}
	}
}

// getIDsFromParquet extracts all IDs from a Parquet file
func getIDsFromParquet(parquetFile, idField string) (map[string]bool, error) {
	ids := make(map[string]bool)

	// Setup DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %v", err)
	}
	defer db.Close()

	// Query to extract IDs from Parquet file
	query := fmt.Sprintf("SELECT DISTINCT %s FROM read_parquet(?)", idField)
	rows, err := db.Query(query, parquetFile)
	if err != nil {
		return nil, fmt.Errorf("failed to query Parquet file %s: %v", parquetFile, err)
	}
	defer rows.Close()

	// Collect all IDs
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan ID: %v", err)
		}
		if id != "" { // Skip empty IDs
			ids[id] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	return ids, nil
}

// deleteOrphanedRecords deletes records where _imported=true but ID not in currentIDs
func deleteOrphanedRecords(db *sql.DB, tableName string, keyColumn string, currentIDs map[string]bool) (int, error) {
	// Build list of current IDs for SQL IN clause
	if len(currentIDs) == 0 {
		// If no current IDs, delete all imported records
		result, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE _imported = true", tableName))
		if err != nil {
			return 0, err
		}

		rowsAffected, err := result.RowsAffected()
		return int(rowsAffected), err
	}

	// Convert map keys to slice for SQL IN clause
	idList := make([]string, 0, len(currentIDs))
	for id := range currentIDs {
		idList = append(idList, fmt.Sprintf("'%s'", id))
	}

	// Delete imported records whose keyColumn value is not in the current export
	query := fmt.Sprintf("DELETE FROM %s WHERE _imported = true AND %s NOT IN (%s)",
		tableName, keyColumn, strings.Join(idList, ","))

	result, err := db.Exec(query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	return int(rowsAffected), err
}

// cleanupUserClaims handles the special case of user_claims with composite key
func cleanupUserClaims(db *sql.DB) {
	// Get current uid+cid pairs from Parquet file
	currentPairs, err := getUserClaimPairsFromParquet("./parquet/UserClaims.parquet")
	if err != nil {
		log.Printf("Error getting user claim pairs from UserClaims.parquet: %v", err)
		return
	}

	if len(currentPairs) == 0 {
		// If no current pairs, delete all imported user_claims
		result, err := db.Exec("DELETE FROM user_claims WHERE _imported = true")
		if err != nil {
			log.Printf("Error deleting all imported user_claims: %v", err)
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err == nil && rowsAffected > 0 {
			fmt.Printf("  Deleted %d orphaned records from user_claims\n", rowsAffected)
		} else {
			fmt.Printf("  No orphaned records found in user_claims\n")
		}
		return
	}

	// Build WHERE conditions for each current pair
	var conditions []string
	for pair := range currentPairs {
		conditions = append(conditions, fmt.Sprintf("(uid = '%s' AND cid = '%s')", pair.uid, pair.cid))
	}

	// Delete imported records whose uid+cid combinations are not in the current export
	query := fmt.Sprintf("DELETE FROM user_claims WHERE _imported = true AND NOT (%s)",
		strings.Join(conditions, " OR "))

	result, err := db.Exec(query)
	if err != nil {
		log.Printf("Error cleaning up user_claims: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		fmt.Printf("  Deleted %d orphaned records from user_claims\n", rowsAffected)
	} else {
		fmt.Printf("  No orphaned records found in user_claims\n")
	}
}

type userClaimPair struct {
	uid string
	cid string
}

// getUserClaimPairsFromParquet extracts all uid+cid pairs from UserClaims.parquet
func getUserClaimPairsFromParquet(parquetFile string) (map[userClaimPair]bool, error) {
	pairs := make(map[userClaimPair]bool)

	// Setup DuckDB connection
	db, err := sql.Open("duckdb", "")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB: %v", err)
	}
	defer db.Close()

	// Query to extract uid+cid pairs from Parquet file
	query := "SELECT DISTINCT uid, cid FROM read_parquet(?)"
	rows, err := db.Query(query, parquetFile)
	if err != nil {
		return nil, fmt.Errorf("failed to query Parquet file %s: %v", parquetFile, err)
	}
	defer rows.Close()

	// Collect all pairs
	for rows.Next() {
		var uid, cid string
		if err := rows.Scan(&uid, &cid); err != nil {
			return nil, fmt.Errorf("failed to scan uid+cid: %v", err)
		}
		if uid != "" && cid != "" { // Skip empty values
			pairs[userClaimPair{uid: uid, cid: cid}] = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %v", err)
	}

	return pairs, nil
}
