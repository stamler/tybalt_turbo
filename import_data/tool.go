package main

import (
	"flag"
	"fmt"
	"imports/extract"
	"imports/load"
	"strings"

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
		adminProfileInsertSQL := "INSERT INTO admin_profiles (uid, work_week_hours, salary, default_charge_out_rate, off_rotation_permitted, skip_min_time_check, opening_date, opening_op, opening_ov, payroll_id) VALUES ({:uid}, IIF({:work_week_hours} = 0, 40, {:work_week_hours}), {:salary}, IIF({:default_charge_out_rate} = 0, 50, CAST({:default_charge_out_rate} AS REAL) / 100), {:off_rotation_permitted}, IIF({:skip_min_time_check} IS false, 'no', 'on_next_bundle'), {:opening_date}, CAST({:opening_op} AS REAL) / 10, CAST({:opening_ov} AS REAL) / 100, {:payroll_id})"

		// Define the binder function for the Admin type
		adminProfileBinder := func(item load.Profile) dbx.Params {
			return dbx.Params{
				"uid":                     item.UserId,
				"work_week_hours":         item.WorkWeekHours,
				"salary":                  item.Salary,
				"default_charge_out_rate": item.DefaultChargeOutRate,
				"off_rotation_permitted":  item.OffRotationPermitted,
				"skip_min_time_check":     item.SkipMinTimeCheckOnNextBundle,
				"opening_date":            item.OpeningDateTimeOff,
				"opening_op":              item.OpeningOP,
				"opening_ov":              item.OpeningOV,
				"payroll_id":              item.PayrollId,
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
	}
}
