package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"imports/attachments"
	"imports/extract"
	"imports/load"
	"io"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"imports/backfill"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/marcboeker/go-duckdb" // DuckDB driver (blank import for side-effect registration)
	"github.com/pocketbase/dbx"
	_ "modernc.org/sqlite" // SQLite driver for deletion cleanup
)

var expenseCollectionId = "o1vpz1mm7qsfoyy"
var targetDatabase = "../app/test_pb_data/data.db"

// expenseImportedFalseCount controls how many expenses are imported with _imported=false
// (triggering writeback to Firebase) during --import --expenses. Set to 0 for production
// (all _imported=true), or a positive number for testing writeback with a subset.
const expenseImportedFalseCount = 0

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
	initFlag := flag.Bool("init", false, "Initialize app database by copying the test database (overwrites existing)")
	dbFlag := flag.String("db", "../app/test_pb_data/data.db", "Path to the target database")

	// Phase flags for selective import (opt-in, running --import with no phase flags is a no-op)
	jobsFlag := flag.Bool("jobs", false, "Import jobs, clients, contacts, categories, job_time_allocations")
	expensesFlag := flag.Bool("expenses", false, "Import vendors, purchase_orders, expenses")
	timeFlag := flag.Bool("time", false, "Import time_sheets, time_entries, time_amendments")
	usersFlag := flag.Bool("users", false, "Import users, profiles, admin_profiles, user_claims, mileage_reset_dates")

	flag.Parse()

	// Use the database path from the flag
	targetDatabase = *dbFlag

	if *initFlag {
		fmt.Println("This will overwrite any existing data in app/pb_data/data.db.")
		fmt.Print("Proceed? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Aborted.")
			return
		}

		src := "../app/test_pb_data/data.db"
		dst := "../app/pb_data/data.db"

		if err := os.MkdirAll(path.Dir(dst), 0755); err != nil {
			log.Fatalf("Failed to ensure destination directory: %v", err)
		}

		if err := copyFile(src, dst); err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}

		if err := cleanupFreshDatabase(dst); err != nil {
			log.Fatalf("Failed to clean initialized database: %v", err)
		}

		return
	}

	if *exportFlag {
		fmt.Println("Exporting data to Parquet files...")
		extract.ToParquet(targetDatabase)
	}

	if *importFlag {
		// Check if at least one phase flag is set
		if !*jobsFlag && !*expensesFlag && !*timeFlag && !*usersFlag {
			fmt.Println("No phases specified. Use --jobs, --expenses, --time, --users to select what to import.")
			fmt.Println("Running --import with no phase flags is a no-op to prevent accidental data overwrites.")
			return
		}

		fmt.Println("Importing data from Parquet files...")

		// Use a fixed timestamp for idempotency instead of dynamic time.Now()
		// (used by time_sheets and purchase_orders)
		fixedTimestamp := "2025-05-30 00:00:00.000Z"

		// =========================================================================
		// PHASE: JOBS (--jobs)
		// Tables: clients, client_contacts, client_notes, jobs, categories, job_time_allocations
		// =========================================================================
		if *jobsFlag {
			fmt.Println("Importing jobs phase: clients, contacts, client_notes, jobs, categories, job_time_allocations...")

			// Delete all existing records before import (full replace)
			// Order doesn't matter since foreign keys are disabled during deletion
			err := deleteAllFromTables(targetDatabase, []string{
				"job_time_allocations",
				"jobs",
				"client_contacts",
				"client_notes",
				"categories",
				"clients",
			})
			if err != nil {
				log.Fatalf("Failed to clear jobs phase tables: %v", err)
			}

			// --- Load Clients ---
			// business_development_lead is pre-resolved during export (augment_clients_export.go)
			// by joining Clients with Profiles to convert legacy Firebase UID to PocketBase UID.
			clientInsertSQL := `INSERT INTO clients (id, name, business_development_lead, _imported) 
			VALUES ({:id}, {:name}, IIF({:business_development_lead} = '', NULL, {:business_development_lead}), true)`

			clientBinder := func(item load.Client) dbx.Params {
				return dbx.Params{
					"id":                        item.Id,
					"name":                      item.Name,
					"business_development_lead": item.BusinessDevelopmentLead,
				}
			}

			load.FromParquet(
				"./parquet/Clients.parquet",
				targetDatabase,
				"clients",
				clientInsertSQL,
				clientBinder,
				true,
			)

			// --- Load Contacts ---
			// Define the specific SQL for the contacts table.
			// Now includes email field for contacts that were written back from Turbo.
			contactInsertSQL := "INSERT INTO client_contacts (id, surname, given_name, email, client, _imported) VALUES ({:id}, {:surname}, {:given_name}, {:email}, {:client}, true)"

			// Define the binder function for the Contact type
			contactBinder := func(item load.ClientContact) dbx.Params {
				return dbx.Params{
					"id":         item.Id,
					"surname":    item.Surname,
					"given_name": item.GivenName,
					"email":      item.Email,
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
			jobInsertSQL := "INSERT INTO jobs (id, number, description, client, contact, manager, alternate_manager, fn_agreement, status, project_award_date, proposal_opening_date, proposal_submission_due_date, proposal, job_owner, branch, outstanding_balance, outstanding_balance_date, parent, _imported) VALUES ({:id}, {:number}, {:description}, {:client}, {:contact}, {:manager}, {:alternate_manager}, {:fn_agreement}, {:status}, {:project_award_date}, {:proposal_opening_date}, {:proposal_submission_due_date}, {:proposal}, {:job_owner}, (SELECT id FROM branches WHERE code = {:branch}), {:outstanding_balance}, {:outstanding_balance_date}, {:parent}, true)"

			// Define the binder function for the Job type
			jobBinder := func(item load.Job) dbx.Params {
				// Default outstanding_balance_date to today if not set in parquet
				outstandingBalanceDate := item.OutstandingBalanceDate
				if outstandingBalanceDate == "" {
					outstandingBalanceDate = time.Now().Format("2006-01-02")
				}

				// For proposals (number starts with "P"), map "Active" or blank status to "In Progress"
				// This is because "Active" is not a valid status for proposals in the new workflow.
				status := item.Status
				isProposal := strings.HasPrefix(item.Number, "P")
				if isProposal && (status == "Active" || status == "") {
					status = "In Progress"
				}

				return dbx.Params{
					"id":                           item.Id,
					"number":                       item.Number,
					"description":                  item.Description,
					"client":                       item.Client,
					"contact":                      item.Contact,
					"manager":                      item.Manager,
					"alternate_manager":            item.AlternateManagerId,
					"fn_agreement":                 item.FnAgreement,
					"status":                       status,
					"project_award_date":           item.ProjectAwardDate,
					"proposal_opening_date":        item.ProposalOpeningDate,
					"proposal_submission_due_date": item.ProposalSubmissionDueDate,
					"proposal":                     item.ProposalId,
					"job_owner":                    item.JobOwnerId,
					"branch":                       item.Branch,
					"outstanding_balance":          item.OutstandingBalance,
					"outstanding_balance_date":     outstandingBalanceDate,
					"parent":                       item.Parent,
					"proposal_value":               item.ProposalValue,
					"time_and_materials":           item.TimeAndMaterials,
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

			// --- Load Job Time Allocations ---
			// Default hours may be 0 initially; upsert keyed by (job, division)
			jobTimeAllocInsertSQL := "INSERT INTO job_time_allocations (id, job, division, hours) VALUES ({:id}, {:job}, {:division}, {:hours})"
			jobTimeAllocBinder := func(item load.JobTimeAllocation) dbx.Params {
				return dbx.Params{
					"id":       item.Id,
					"job":      item.Job,
					"division": item.Division,
					"hours":    item.Hours,
				}
			}
			load.FromParquet(
				"./parquet/JobTimeAllocations.parquet",
				targetDatabase,
				"job_time_allocations",
				jobTimeAllocInsertSQL,
				jobTimeAllocBinder,
				true, // upsert
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

			// --- Load Client Notes ---
			// Import client notes from TurboClientNotes (exported via Firebase sync).
			// The uid field contains legacy_uid and needs to be converted to PocketBase uid via admin_profiles.
			// The jobId field is already a PocketBase job ID and can be used directly.
			clientNoteInsertSQL := `INSERT INTO client_notes (
				id, 
				created, 
				updated, 
				note, 
				client, 
				job, 
				job_not_applicable, 
				uid, 
				job_status_changed_to, 
				_imported
			) VALUES (
				{:id}, 
				{:created}, 
				{:updated}, 
				{:note}, 
				{:client_id}, 
				IIF({:job_id} = '', NULL, {:job_id}),
				{:job_not_applicable}, 
				(SELECT uid FROM admin_profiles WHERE legacy_uid = {:uid}),
				{:job_status_changed_to}, 
				true
			)`

			clientNoteBinder := func(item load.ClientNote) dbx.Params {
				return dbx.Params{
					"id":                    item.Id,
					"created":               item.Created,
					"updated":               item.Updated,
					"note":                  item.Note,
					"client_id":             item.ClientId,
					"job_id":                item.JobId,
					"job_not_applicable":    item.JobNotApplicable,
					"uid":                   item.Uid,
					"job_status_changed_to": item.JobStatusChangedTo,
				}
			}

			load.FromParquet(
				"./parquet/ClientNotes.parquet",
				targetDatabase,
				"client_notes",
				clientNoteInsertSQL,
				clientNoteBinder,
				true,
			)

		} // end jobs phase

		// =========================================================================
		// PHASE: USERS (--users)
		// Tables: users, admin_profiles, profiles, _externalAuths, user_claims, mileage_reset_dates
		// =========================================================================
		if *usersFlag {
			fmt.Println("Importing users phase: users, profiles, admin_profiles, user_claims, mileage_reset_dates...")

			// Delete all existing records before import (full replace)
			// Order doesn't matter since foreign keys are disabled during deletion
			err := deleteAllFromTables(targetDatabase, []string{
				"user_claims",
				"mileage_reset_dates",
				"_externalAuths",
				"admin_profiles",
				"profiles",
				"users",
			})
			if err != nil {
				log.Fatalf("Failed to clear users phase tables: %v", err)
			}

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

			// --- Load Admin Profiles ---
			// Define the specific SQL for the admin profiles table
			// default_charge_out_rate, opening_ov are type Decimal and so need to be case to float then divided by 100 (Decimal 6,2), opening_op needs to be divided by 10 (Decimal 5,1)
			// if work_week_hours is 0, set it to 40
			// default_charge_out_rate should be set to 50 if it is 0
			adminProfileInsertSQL := "INSERT INTO admin_profiles (uid, active, work_week_hours, salary, default_charge_out_rate, off_rotation_permitted, skip_min_time_check, opening_date, opening_op, opening_ov, payroll_id, untracked_time_off, time_sheet_expected, allow_personal_reimbursement, mobile_phone, job_title, personal_vehicle_insurance_expiry, default_branch, legacy_uid, _imported) VALUES ({:uid}, true, IIF({:work_week_hours} = 0, 40, {:work_week_hours}), {:salary}, IIF({:default_charge_out_rate} = 0, 50, CAST({:default_charge_out_rate} AS REAL) / 100), {:off_rotation_permitted}, IIF({:skip_min_time_check} IS false, 'no', 'on_next_bundle'), {:opening_date}, CAST({:opening_op} AS REAL) / 10, CAST({:opening_ov} AS REAL) / 100, {:payroll_id}, {:untracked_time_off}, {:time_sheet_expected}, {:allow_personal_reimbursement}, {:mobile_phone}, {:job_title}, {:personal_vehicle_insurance_expiry}, (SELECT id FROM branches WHERE code = {:default_branch}), {:legacy_uid}, true)"

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
					"legacy_uid":                        item.Id,
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

			// After loading user_claims, synthesize po_approver claim and props from Profiles
			if err := synthesizePoApproverProps(targetDatabase); err != nil {
				log.Fatalf("Failed to synthesize po_approver props: %v", err)
			}

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

		} // end users phase

		// =========================================================================
		// PHASE: TIME (--time)
		// Tables: time_sheets, time_entries, time_amendments
		// =========================================================================
		if *timeFlag {
			fmt.Println("Importing time phase: time_sheets, time_entries, time_amendments...")

			// Delete all existing records before import (full replace)
			// Order doesn't matter since foreign keys are disabled during deletion
			err := deleteAllFromTables(targetDatabase, []string{
				"time_entries",
				"time_amendments",
				"time_sheets",
			})
			if err != nil {
				log.Fatalf("Failed to clear time phase tables: %v", err)
			}

			// --- Load TimeSheets ---
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
					"committer":       item.ApproverUid, // set committer to approver uid since we don't have this data in the parquet file
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

		} // end time phase

		// =========================================================================
		// PHASE: EXPENSES (--expenses)
		// Tables: vendors, purchase_orders, expenses
		// =========================================================================
		if *expensesFlag {
			fmt.Println("Importing expenses phase: vendors, purchase_orders, expenses...")

			// Delete all existing records before import (full replace)
			// Order doesn't matter since foreign keys are disabled during deletion
			err := deleteAllFromTables(targetDatabase, []string{
				"expenses",
				"purchase_orders",
				"vendors",
			})
			if err != nil {
				log.Fatalf("Failed to clear expenses phase tables: %v", err)
			}

			// --- Load Vendors ---
			// Vendors.parquet now includes hybrid ID resolution:
			// - Vendors matching TurboVendors by name use the TurboVendor's PocketBase ID
			// - Other vendors get deterministic IDs from MD5 hash of name
			// - Turbo-only vendors (not in any expense) are also included
			// - alias and status fields are populated from TurboVendors when matched
			vendorInsertSQL := "INSERT INTO vendors (id, name, alias, status, _imported) VALUES ({:id}, {:name}, {:alias}, {:status}, true)"

			// Define the binder function for the Vendor type
			vendorBinder := func(item load.Vendor) dbx.Params {
				status := item.Status
				if status == "" {
					status = "Active"
				}
				return dbx.Params{
					"id":     item.Id,
					"name":   item.Name,
					"alias":  item.Alias,
					"status": status,
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
				`INSERT INTO purchase_orders (
				id,
				po_number,
				approved,
				second_approval,
				second_approver,
				priority_second_approver,
				closed,
				closer,
				cancelled,
				canceller,
				rejected,
				rejector,
				rejection_reason,
				type,
				frequency,
				status,
				closed_by_system,
				description,
				approver,
				date,
				end_date,
				vendor,
				uid,
				total,
				approval_total,
				payment_type,
				job,
				division,
				category,
				parent_po,
				branch,
				attachment,
				attachment_hash,
				_imported
			) VALUES (
				{:id},
				{:po_number},
				{:approved},
				{:second_approval},
				{:second_approver},
				{:priority_second_approver},
				{:closed},
				{:closer},
				{:cancelled},
				{:canceller},
				{:rejected},
				{:rejector},
				{:rejection_reason},
				{:type},
				{:frequency},
				{:status},
				{:closed_by_system},
				{:description},
				{:approver},
				{:date},
				{:end_date},
				{:vendor},
				{:uid},
				{:total},
				{:approval_total},
				{:payment_type},
				{:job},
				{:division},
				{:category},
				{:parent_po},
				{:branch},
				{:attachment},
				{:attachment_hash},
				true
			)`,
				func(item load.PurchaseOrder) dbx.Params {
					// Use actual values from parquet when available, fall back to defaults for legacy-derived POs
					approved := item.Approved
					if approved == "" {
						approved = fixedTimestamp
					}
					// For second_approval: use actual value from Turbo POs, or fixedTimestamp for legacy
					secondApproval := item.SecondApproval
					if secondApproval == "" {
						secondApproval = fixedTimestamp
					}
					// Legacy-derived POs have empty status; Turbo POs always include it.
					isDerived := item.Status == ""
					poType := item.Type
					if poType == "" {
						poType = "Normal"
					}
					status := item.Status
					if status == "" {
						status = "Closed"
					}
					// Only set closed when the status is Closed, and only auto-fill for derived POs.
					closed := item.Closed
					if status != "Closed" {
						closed = ""
					} else if isDerived && closed == "" {
						closed = fixedTimestamp
					}
					description := item.Description
					if description == "" {
						description = "Imported from Firebase Expenses"
					}
					return dbx.Params{
						"id":                       item.Id,
						"po_number":                item.PoNumber,
						"approved":                 approved,
						"second_approval":          secondApproval,
						"second_approver":          item.SecondApprover,          // May be empty for legacy POs
						"priority_second_approver": item.PrioritySecondApprover, // May be empty for legacy POs
						"closed":                   closed,
						"closer":                   item.Closer,                  // May be empty for legacy POs
						"cancelled":                item.Cancelled,
						"canceller":                item.Canceller,               // May be empty for legacy POs
						"rejected":                 item.Rejected,
						"rejector":                 item.Rejector,                // May be empty for legacy POs
						"rejection_reason":         item.RejectionReason,
						"type":                     poType,
						"frequency":                item.Frequency,
						"status":                   status,
						"closed_by_system":         1, // Not in MySQL, keep as default
						"description":              description,
						"approver":                 item.Approver,
						"date":                     item.Date,
						"end_date":                 item.EndDate,
						"vendor":                   item.Vendor,
						"uid":                      item.Uid,
						"total":                    item.Total,
						"approval_total":           item.ApprovalTotal,
						"payment_type":             item.PaymentType,
						"job":                      item.Job,
						"division":                 item.Division,
						"category":                 item.Category,                // PocketBase ID, may be empty
						"parent_po":                item.ParentPo,                // PocketBase ID, may be empty
						"branch":                   item.Branch,                  // PocketBase ID, may be empty
						"attachment":               item.Attachment,
						"attachment_hash":          item.AttachmentHash,
					}
				},
				true, // Enable upsert for idempotency
			)

			// Note: TurboPurchaseOrders are now merged during extraction in expensesToPurchaseOrders()
			// using hybrid ID resolution. purchase_orders.parquet includes:
			// - POs matching TurboPurchaseOrders by number (using Turbo's ID and fields)
			// - Derived POs from expenses (using generated IDs)
			// - Turbo-only POs (no matching expenses)

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
			attachment_hash,
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
			{:attachment_hash},
			{:cc_last_4_digits},
			{:allowance_types},
			{:purchase_order},
			true,
			{:approver},
			{:approved},
			{:committer},
			{:committed},
			{:committed_week_ending},
			{:imported} -- _imported: TRUE skips writeback, FALSE triggers writeback to Firebase
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

			// Counter for testing writeback: uses expenseImportedFalseCount constant (top of file)
			expenseCounter := 0

			// Define the binder function for the Expense type
			expenseBinder := func(item load.Expense) dbx.Params {
				expenseCounter++
				// _imported=false triggers writeback; controlled by expenseImportedFalseCount constant
				imported := expenseCounter > expenseImportedFalseCount

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
					"attachment_hash": func() string {
						// Extract hash from original Firebase Storage path: Expenses/{uid}/{hash}.{ext}
						if item.OriginalAttachment == "" {
							return ""
						}
						filename := path.Base(item.OriginalAttachment)          // e.g., "8f4e2d1a...b7.pdf"
						return strings.TrimSuffix(filename, path.Ext(filename)) // Remove extension to get hash
					}(),
					"cc_last_4_digits":      item.CCLast4Digits,
					"approver":              item.Approver,
					"approved":              item.Committed.Format("2006-01-02 15:04:05.000Z"),
					"committer":             item.Committer,
					"committed":             item.Committed.Format("2006-01-02 15:04:05.000Z"),
					"committed_week_ending": item.CommittedWeekEnding,
					"imported":              imported,
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

			// After importing expenses, compute Allowance/Meals totals using expense_rates
			if err := backfillAllowanceTotals(targetDatabase); err != nil {
				log.Fatalf("Failed to backfill allowance totals: %v", err)
			}

			// After importing expenses, compute Mileage totals using tiered mileage rates
			if err := backfillMileageTotals(targetDatabase); err != nil {
				log.Fatalf("Failed to backfill mileage totals: %v", err)
			}

		} // end expenses phase

		// Backfill branches after import completes
		if err := backfill.BackfillBranches(targetDatabase); err != nil {
			log.Fatalf("Failed to backfill branches: %v", err)
		}
	}

	if *attachmentsFlag {
		attachments.MigrateAttachments("./parquet/Expenses.parquet", "attachment", "destination_attachment", expenseCollectionId)
	}
}

// synthesizePoApproverProps ensures that:
// 1) Any user whose Profiles.customClaims contains 'tapr', 'vp', or 'smg' has a user_claims row for the 'po_approver' claim.
// 2) A po_approver_props row exists per such user_claim with max_amount and divisions based on the presence of tapr/vp/smg and default division.
//
// Mapping per _ISSUES.md:
// - tapr (and not vp or smg): max_amount = 500; divisions = [profile.default_division]
// - vp (and not smg): max_amount = 2500; divisions = [profile.default_division]
// - smg: max_amount = 250000; divisions = []
func synthesizePoApproverProps(dbPath string) error {
	// Read Profiles.parquet via DuckDB, then upsert into SQLite
	duck, err := sql.Open("duckdb", "")
	if err != nil {
		return fmt.Errorf("open duckdb: %w", err)
	}
	defer duck.Close()

	rows, err := duck.Query(`
		SELECT
			pocketbase_uid AS uid,
			LOWER(COALESCE(customClaims, '')) AS claims,
			pocketbase_defaultDivision AS default_division
		FROM read_parquet('./parquet/Profiles.parquet')
	`)
	if err != nil {
		return fmt.Errorf("read Profiles.parquet: %w", err)
	}
	defer rows.Close()

	type rowData struct {
		uid             string
		defaultDivision string
		maxAmount       float64
		divisionsJSON   string
	}

	var toUpsert []rowData
	for rows.Next() {
		var uid, claims, defaultDivision sql.NullString
		if err := rows.Scan(&uid, &claims, &defaultDivision); err != nil {
			return fmt.Errorf("scan profile: %w", err)
		}

		uidValue := ""
		if uid.Valid {
			uidValue = uid.String
		}
		claimsValue := ""
		if claims.Valid {
			claimsValue = claims.String
		}
		defaultDivisionValue := ""
		if defaultDivision.Valid {
			defaultDivisionValue = defaultDivision.String
		}

		hasTapr := strings.Contains(claimsValue, "tapr")
		hasVp := strings.Contains(claimsValue, "vp")
		hasSmg := strings.Contains(claimsValue, "smg")

		var maxAmount float64
		var divisionsJSON string
		switch {
		case hasSmg:
			maxAmount = 250000
			divisionsJSON = "[]"
		case hasVp && !hasSmg:
			maxAmount = 2500
		case hasTapr && !hasVp && !hasSmg:
			maxAmount = 500
		default:
			// not a candidate
		}

		if maxAmount == 0 {
			continue
		}

		// Cannot create approver props without a valid PocketBase user id.
		if uidValue == "" {
			continue
		}

		if divisionsJSON == "" {
			// Tapr/vp approvers require a default division; skip if missing.
			if defaultDivisionValue == "" && (hasTapr || hasVp) {
				continue
			}
			b, _ := json.Marshal([]string{defaultDivisionValue})
			divisionsJSON = string(b)
		}

		toUpsert = append(toUpsert, rowData{
			uid:             uidValue,
			defaultDivision: defaultDivisionValue,
			maxAmount:       maxAmount,
			divisionsJSON:   divisionsJSON,
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate profiles: %w", err)
	}

	if len(toUpsert) == 0 {
		return nil
	}

	sqliteDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer sqliteDB.Close()

	var poApproverClaimId string
	if err := sqliteDB.QueryRow(`SELECT id FROM claims WHERE name = 'po_approver'`).Scan(&poApproverClaimId); err != nil {
		return fmt.Errorf("fetch po_approver claim id: %w", err)
	}

	tx, err := sqliteDB.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ensureClaimStmt, err := tx.Prepare(`INSERT OR IGNORE INTO user_claims (uid, cid, _imported) VALUES (?, ?, 1)`)
	if err != nil {
		return fmt.Errorf("prepare ensureClaim: %w", err)
	}
	defer ensureClaimStmt.Close()

	getUserClaimIdStmt, err := tx.Prepare(`SELECT id FROM user_claims WHERE uid = ? AND cid = ?`)
	if err != nil {
		return fmt.Errorf("prepare getUserClaimId: %w", err)
	}
	defer getUserClaimIdStmt.Close()

	deletePropsStmt, err := tx.Prepare(`DELETE FROM po_approver_props WHERE user_claim = ?`)
	if err != nil {
		return fmt.Errorf("prepare deleteProps: %w", err)
	}
	defer deletePropsStmt.Close()

	insertPropsStmt, err := tx.Prepare(`INSERT INTO po_approver_props (user_claim, max_amount, divisions) VALUES (?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare insertProps: %w", err)
	}
	defer insertPropsStmt.Close()

	for _, r := range toUpsert {
		if _, err := ensureClaimStmt.Exec(r.uid, poApproverClaimId); err != nil {
			return fmt.Errorf("ensure user_claim: %w", err)
		}
		var userClaimId string
		if err := getUserClaimIdStmt.QueryRow(r.uid, poApproverClaimId).Scan(&userClaimId); err != nil {
			return fmt.Errorf("fetch user_claim id: %w", err)
		}
		if _, err := deletePropsStmt.Exec(userClaimId); err != nil {
			return fmt.Errorf("delete old props: %w", err)
		}
		if _, err := insertPropsStmt.Exec(userClaimId, r.maxAmount, r.divisionsJSON); err != nil {
			return fmt.Errorf("insert po_approver_props: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// backfillAllowanceTotals calculates and writes totals for Allowance/Meals
// expenses where total is missing or zero, using the effective rates at the
// expense date from the expense_rates table. This mirrors the logic used in
// reporting queries, but persists the computed value for downstream simplicity.
func backfillAllowanceTotals(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	updateSQL := `
        UPDATE expenses AS e
        SET total = COALESCE((
            SELECT
                (CASE WHEN e.allowance_types LIKE '%"Breakfast"%' THEN r.breakfast ELSE 0 END)
              + (CASE WHEN e.allowance_types LIKE '%"Lunch"%'     THEN r.lunch     ELSE 0 END)
              + (CASE WHEN e.allowance_types LIKE '%"Dinner"%'    THEN r.dinner    ELSE 0 END)
              + (CASE WHEN e.allowance_types LIKE '%"Lodging"%'   THEN r.lodging   ELSE 0 END)
            FROM expense_rates r
            WHERE r.effective_date = (
                SELECT MAX(i.effective_date)
                FROM expense_rates i
                WHERE i.effective_date <= e.date
            )
        ), e.total)
        WHERE e.payment_type IN ('Allowance','Meals')
          AND (e.total IS NULL OR e.total = 0);
    `

	if _, err := db.Exec(updateSQL); err != nil {
		return fmt.Errorf("update allowance totals: %w", err)
	}
	return nil
}

// backfillMileageTotals calculates and writes totals for Mileage expenses where
// total is missing or zero, using tiered mileage rates from expense_rates at the
// effective date of each expense, and the appropriate annual period defined by
// mileage_reset_dates. This mirrors the SQL used in reporting to ensure the same
// piecewise tiered calculation.
func backfillMileageTotals(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	updateSQL := `
        WITH rates_expanded AS (
            SELECT
                m.effective_date,
                CAST(t.key AS INTEGER) AS tier_lower,
                LEAD(CAST(t.key AS INTEGER)) OVER (
                    PARTITION BY m.effective_date
                    ORDER BY CAST(t.key AS INTEGER)
                ) AS tier_upper,
                CAST(t.value AS REAL) AS tier_rate
            FROM expense_rates m
            CROSS JOIN json_each(m.mileage) AS t
        ),
        CumulativeMileage AS (
            SELECT
                e.id,
                e.uid,
                e.date,
                e.distance,
                (
                    SELECT MAX(r.date)
                    FROM mileage_reset_dates r
                    WHERE r.date <= e.date
                ) AS reset_mileage_date,
                SUM(e.distance) OVER (
                    PARTITION BY e.uid, (
                        SELECT MAX(r.date)
                        FROM mileage_reset_dates r
                        WHERE r.date <= e.date
                    )
                    ORDER BY e.date, e.id
                ) AS end_distance,
                (
                    SELECT MAX(m.effective_date)
                    FROM expense_rates m
                    WHERE m.effective_date <= e.date
                ) AS effective_date
            FROM expenses e
            WHERE e.payment_type = 'Mileage'
              AND e.committed != ''
        ),
        base AS (
            SELECT
                cm.id,
                cm.uid,
                cm.date,
                cm.reset_mileage_date,
                cm.distance,
                cm.end_distance,
                cm.effective_date
            FROM CumulativeMileage cm
        ),
        overlaps AS (
            SELECT
                b.id,
                b.end_distance - b.distance AS start_distance,
                b.end_distance,
                r.tier_lower,
                COALESCE(r.tier_upper, 1e9) AS tier_upper,
                r.tier_rate
            FROM base b
            JOIN rates_expanded r
              ON r.effective_date = b.effective_date
            WHERE b.end_distance > r.tier_lower
              AND (r.tier_upper IS NULL OR (b.end_distance - b.distance) < r.tier_upper)
        ),
        tier_calcs AS (
            SELECT
                id,
                MAX(0,
                    MIN(end_distance, tier_upper)
                    - MAX(start_distance, tier_lower)
                ) AS overlap_km,
                tier_rate
            FROM overlaps
        ),
        mileage_totals AS (
            SELECT
                b.id,
                ROUND(COALESCE(
                    (SELECT SUM(overlap_km * tier_rate)
                     FROM tier_calcs tc
                     WHERE tc.id = b.id),
                    0
                ), 2) AS mileage_total
            FROM base b
        )
        UPDATE expenses AS e
        SET total = (SELECT mt.mileage_total FROM mileage_totals mt WHERE mt.id = e.id)
        WHERE e.payment_type = 'Mileage'
          AND (e.total IS NULL OR e.total = 0)
          AND e.committed != '';
    `

	if _, err := db.Exec(updateSQL); err != nil {
		return fmt.Errorf("update mileage totals: %w", err)
	}
	return nil
}

// deleteAllFromTables deletes all records from the specified tables.
// Foreign keys are temporarily disabled to avoid constraint violations during deletion.
func deleteAllFromTables(dbPath string, tables []string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Disable foreign keys during bulk delete
	_, _ = db.Exec("PRAGMA foreign_keys = OFF")
	defer db.Exec("PRAGMA foreign_keys = ON")

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	for _, tbl := range tables {
		result, err := tx.Exec("DELETE FROM " + tbl)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("delete from %s: %w", tbl, err)
		}
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			fmt.Printf("  Deleted %d records from %s\n", rowsAffected, tbl)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

// copyFile copies the contents and file mode from src to dst, overwriting dst if it exists
func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(out, in)
	if syncErr := out.Sync(); syncErr != nil && copyErr == nil {
		copyErr = syncErr
	}
	if closeErr := out.Close(); closeErr != nil && copyErr == nil {
		copyErr = closeErr
	}

	if copyErr != nil {
		return copyErr
	}

	if info, statErr := os.Stat(src); statErr == nil {
		_ = os.Chmod(dst, info.Mode())
	}

	return nil
}

// cleanupFreshDatabase removes test data from the freshly copied app database
func cleanupFreshDatabase(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	// Disable foreign keys during bulk delete
	_, _ = db.Exec("PRAGMA foreign_keys = OFF")
	defer db.Exec("PRAGMA foreign_keys = ON")

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	tables := []string{
		"users",
		"absorb_actions",
		"admin_profiles",
		"categories",
		"client_contacts",
		"client_notes",
		"clients",
		"expenses",
		"job_time_allocations",
		"jobs",
		"machine_secrets",
		"notifications",
		"po_approver_props",
		"profiles",
		"purchase_orders",
		"time_amendments",
		"time_entries",
		"time_sheet_reviewers",
		"time_sheets",
		"user_claims",
		"vendors",
	}

	for _, tbl := range tables {
		if _, err := tx.Exec("DELETE FROM " + tbl); err != nil {
			// If a table doesn't exist in this schema, skip it
			if strings.Contains(strings.ToLower(err.Error()), "no such table") {
				continue
			}
			_ = tx.Rollback()
			return fmt.Errorf("delete from %s: %w", tbl, err)
		}
	}

	// Remove test-only rate sheet fixture (used by rate_sheets_test.go)
	// Keep the real rate sheet (c41ofep525bcacj) and its entries intact
	if _, err := tx.Exec("DELETE FROM rate_sheets WHERE id = 'test_empty_sheet'"); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "no such table") {
			_ = tx.Rollback()
			return fmt.Errorf("delete test rate sheet: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
