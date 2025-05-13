package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"imports/extract"
	"imports/load"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb" // DuckDB driver (blank import for side-effect registration)
	"github.com/pocketbase/dbx"
	"google.golang.org/api/option"
)

)

// This file is used to run either an export or an import.

func main() {
	// Parse command line arguments
	exportFlag := flag.Bool("export", false, "Export data to Parquet files")
	importFlag := flag.Bool("import", false, "Import data from Parquet files")
	attachmentsFlag := flag.Bool("attachments", false, "Import attachments from GCS to S3")
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
				CAST({:payout_request_amount} AS REAL) / 100, 
				{:date}, 
				{:week_ending}, 
				{:tsid}, 
				{:category}
			)`

		// Define the binder function for the TimeEntry type
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

		// --- Load TimeAmendments ---
		// Define the specific SQL for the time_amendments table
		timeAmendmentInsertSQL := `
			INSERT INTO time_amendments (
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
				skip_tsid_check
			) VALUES (
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
				false
			)`

		// Define the binder function for the TimeAmendment type
		timeAmendmentBinder := func(item load.TimeAmendment) dbx.Params {
			return dbx.Params{
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
			"../app/test_pb_data/data.db",
			"time_amendments",      // Table name (for logging)
			timeAmendmentInsertSQL, // The specific INSERT SQL
			timeAmendmentBinder,    // The specific binder function
		)

		// --- Load Vendors ---
		// Define the specific SQL for the vendors table
		vendorInsertSQL := "INSERT INTO vendors (id, name, status) VALUES ({:id}, {:name}, {:status})"

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
			"../app/test_pb_data/data.db",
			"vendors",       // Table name (for logging)
			vendorInsertSQL, // The specific INSERT SQL
			vendorBinder,    // The specific binder function
		)

		// --- Load Expenses ---
		// Define the specific SQL for the expenses table
		expenseInsertSQL := `INSERT INTO expenses (
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
			committed_week_ending
		) VALUES (
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
			{:committed_week_ending}
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
				// "payroll_id":            item.PayrollId,
				// "breakfast":             item.Breakfast,
				// "lunch":                 item.Lunch,
				// "dinner":                item.Dinner,
				// "lodging":               item.Lodging,
				"purchase_order":        item.PurchaseOrderNumber, // This must change to id in the future
				"uid":                   item.Uid,
				"division":              item.Division,
				"job":                   item.Job,
				"category":              item.Category,
				"date":                  item.Date,
				"pay_period_ending":     item.PayPeriodEnding,
				"description":           item.Description,
				"vendor":                item.Vendor,
				"distance":              item.Distance,
				"allowance_types":       createAllowanceTypes(item),
				"total":                 item.Total,
				"payment_type":          item.PaymentType,
				"attachment":            item.Attachment,
				"cc_last_4_digits":      item.CCLast4Digits,
				"approver":              item.Approver,
				"approved":              nowString,
				"committer":             item.Committer,
				"committed":             item.Committed.Format("2006-01-02 15:04:05.000Z"),
				"committed_week_ending": item.CommittedWeekEnding,
			}
		}

		// Call the generic function, specifying the type and providing SQL + binder
		load.FromParquet(
			"./parquet/Expenses.parquet",
			"../app/test_pb_data/data.db",
			"expenses",       // Table name (for logging)
			expenseInsertSQL, // The specific INSERT SQL
			expenseBinder,    // The specific binder function
		)

		// --- Load User Claims ---
		// Define the specific SQL for the user_claims table
		userClaimInsertSQL := `INSERT INTO user_claims (uid, cid) VALUES ({:uid}, {:cid})`

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
			"../app/test_pb_data/data.db",
			"user_claims",      // Table name (for logging)
			userClaimInsertSQL, // The specific INSERT SQL
			userClaimBinder,    // The specific binder function
		)
	}

	if *attachmentsFlag {
		fmt.Println("Importing attachments from GCS to S3...")

		// --- Load Expenses attachments, transfering from Google Cloud Storage
		// to the local file system, renaming them correctly, then uploading them
		// to AWS S3.

		// 1. setup the Google Cloud Storage client
		ctx := context.Background()
		gcsClient, err := storage.NewClient(ctx, option.WithAPIKey(gcsAPIKey), option.WithCredentialsFile(gcsServiceAccountJSON))
		if err != nil {
			log.Fatalf("Failed to create Google Cloud Storage client: %v", err)
		}
		defer gcsClient.Close()
		_ = gcsClient.Bucket(gcsBucketName)

		// 2. setup the AWS S3 client
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(awsRegion),
			Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
		})
		if err != nil {
			log.Fatalf("Failed to create AWS session: %v", err)
		}

		s3Svc := s3.New(sess)
		_ = s3Svc // Placeholder to use s3Svc, remove when used in step 4

		// 3. from parquet/Expenses.parquet `SELECT attachment, destination_attachment FROM data WHERE attachment IS NOT NULL` into a slice of structs
		type ExpenseAttachment struct {
			Attachment            *string `parquet:"name=attachment, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
			DestinationAttachment *string `parquet:"name=destination_attachment, type=BYTE_ARRAY, convertedtype=UTF8, repetitiontype=OPTIONAL"`
		}

		var attachmentInfos []ExpenseAttachment

		// Setup DuckDB
		// The empty string for the DSN typically means an in-memory database for DuckDB.
		// The driver registers itself under the name "duckdb" upon import.
		db, err := sql.Open("duckdb", "")
		if err != nil {
			log.Fatalf("Failed to open DuckDB database: %v", err)
		}
		defer db.Close()

		if err := db.PingContext(ctx); err != nil { // Good practice to ping the database
			log.Fatalf("Failed to ping DuckDB: %v", err)
		}

		// Query Parquet file using DuckDB
		rows, err := db.QueryContext(ctx, `
			SELECT attachment, destination_attachment 
			FROM read_parquet('./parquet/Expenses.parquet') 
			WHERE attachment IS NOT NULL AND attachment != ''
		`)
		if err != nil {
			log.Fatalf("Failed to query Expenses.parquet with DuckDB: %v", err)
		}
		defer rows.Close()

		log.Println("Reading attachments from Expenses.parquet using DuckDB...")
		for rows.Next() {
			var ea ExpenseAttachment
			var attachment, destAttachment string
			if err := rows.Scan(&attachment, &destAttachment); err != nil {
				log.Printf("Warning: Failed to scan row from DuckDB result: %v. Skipping row.", err)
				continue
			}
			if attachment != "" {
				ea.Attachment = &attachment
			}
			if destAttachment != "" {
				ea.DestinationAttachment = &destAttachment
			}
			if ea.Attachment != nil {
				attachmentInfos = append(attachmentInfos, ea)
			}
		}
		if err := rows.Err(); err != nil {
			log.Fatalf("Error iterating DuckDB query results: %v", err)
		}

		log.Printf("Found %d expense attachments to process using DuckDB.", len(attachmentInfos))

		// 4. for each row in the slice
		for _, info := range attachmentInfos {
			if info.Attachment == nil || *info.Attachment == "" {
				log.Println("Skipping row with missing GCS attachment path.")
				continue
			}
			if info.DestinationAttachment == nil || *info.DestinationAttachment == "" {
				log.Println("Skipping row with missing S3 destination attachment path.")
				continue
			}

			gcsObjectPath := *info.Attachment
			s3ObjectKey := *info.DestinationAttachment

			log.Printf("Processing attachment: GCS: gs://%s/%s -> S3: s3://%s/%s", gcsBucketName, gcsObjectPath, awsS3BucketName, s3ObjectKey)

			//    1. download the attachment from Google Cloud Storage to the local file system
			gcsObject := gcsClient.Bucket(gcsBucketName).Object(gcsObjectPath)
			rc, err := gcsObject.NewReader(ctx)
			if err != nil {
				log.Printf("ERROR: Failed to create reader for GCS object gs://%s/%s: %v", gcsBucketName, gcsObjectPath, err)
				continue
			}

			// Create a temporary local file
			// Use a subdirectory in the current directory to avoid cluttering the root
			tempDir := "./temp_attachments"
			if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
				log.Printf("ERROR: Failed to create temporary directory %s: %v", tempDir, err)
				continue
			}
			localTempFilePath := filepath.Join(tempDir, filepath.Base(gcsObjectPath))
			localFile, err := os.Create(localTempFilePath)
			if err != nil {
				log.Printf("ERROR: Failed to create temporary file %s: %v", localTempFilePath, err)
				rc.Close() // Close GCS reader
				continue
			}

			if _, err := io.Copy(localFile, rc); err != nil {
				log.Printf("ERROR: Failed to download GCS object gs://%s/%s to %s: %v", gcsBucketName, gcsObjectPath, localTempFilePath, err)
				rc.Close()
				localFile.Close()
				os.Remove(localTempFilePath) // Attempt to clean up temp file
				continue
			}
			rc.Close()
			localFile.Close()
			log.Printf("Successfully downloaded gs://%s/%s to %s", gcsBucketName, gcsObjectPath, localTempFilePath)

			//    2. upload the file to AWS S3 using the destination_attachment path
			fileToUpload, err := os.Open(localTempFilePath)
			if err != nil {
				log.Printf("ERROR: Failed to open temporary file %s for S3 upload: %v", localTempFilePath, err)
				os.Remove(localTempFilePath) // Attempt to clean up temp file
				continue
			}

			_, err = s3Svc.PutObject(&s3.PutObjectInput{
				Bucket: aws.String(awsS3BucketName),
				Key:    aws.String(s3ObjectKey),
				Body:   fileToUpload,
			})
			fileToUpload.Close() // Close the file before attempting to remove it

			if err != nil {
				log.Printf("ERROR: Failed to upload %s to S3 bucket %s key %s: %v", localTempFilePath, awsS3BucketName, s3ObjectKey, err)
				os.Remove(localTempFilePath) // Attempt to clean up temp file
				continue
			}
			log.Printf("Successfully uploaded %s to S3 bucket %s key %s", localTempFilePath, awsS3BucketName, s3ObjectKey)

			// Clean up the temporary file
			if err := os.Remove(localTempFilePath); err != nil {
				log.Printf("WARNING: Failed to remove temporary file %s: %v", localTempFilePath, err)
			}
		}
	}
}
