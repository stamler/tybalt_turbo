package load

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/pocketbase/dbx"
	_ "modernc.org/sqlite" // Import modernc SQLite driver for side-effect registration
)

/*
	Order of operations:
	1. ✅ Upload Clients.parquet to the sqlite database
	2. ✅ Upload Contacts.parquet to the sqlite database
	3. ✅ Upload Jobs.parquet to the sqlite database
	4. ✅ Upload Profiles.parquet to the sqlite database users table
	5. ✅ Upload Profiles.parquet to the sqlite database admin_profiles table
	6. ✅ Upload Profiles.parquet to the sqlite database profiles table
	7. ✅ Upload Profiles.parquet to the sqlite database _externalAuths table
	8. ✅ Upload TimeSheets.parquet to the sqlite database
	9. ✅ Upload TimeEntries.parquet to the sqlite database
	10. ✅ Upload TimeAmendments.parquet to the sqlite database
	11. Upload Expenses.parquet to the sqlite database (these reference jobs, profiles, and purchase orders) We may not do this because there aren't many purchase orders and we can archive the attachments.
*/

// --- Struct definitions for Parquet data ---
// TODO: Move these definitions to a more shared location if used outside this package.

type Client struct {
	Id                         string `parquet:"id"`
	Name                       string `parquet:"name"`
	BusinessDevelopmentLeadUid string `parquet:"businessDevelopmentLeadUid"` // Legacy Firebase UID, needs reverse conversion on import
}

type ClientContact struct {
	Id        string `parquet:"id"`
	Surname   string `parquet:"surname"`
	GivenName string `parquet:"givenName"`
	Email     string `parquet:"email"`
	Client    string `parquet:"client_id"`
}

type Job struct {
	Number                      string `parquet:"id"`
	AlternateManagerDisplayName string `parquet:"alternateManagerDisplayName"`
	AlternateManagerUid         string `parquet:"alternateManagerUid"`
	Categories                  string `parquet:"categories"`
	ClientName                  string `parquet:"client"`
	ClientContact               string `parquet:"clientContact"`
	Description                 string `parquet:"description"`
	Divisions                   string `parquet:"divisions"`
	DivisionsIds                string `parquet:"divisions_ids"`
	FnAgreement                 bool   `parquet:"fnAgreement"`
	HasTimeEntries              bool   `parquet:"hasTimeEntries"`
	JobOwner                    string `parquet:"jobOwner"`
	JobOwnerId                  string `parquet:"job_owner_id"`
	LastTimeEntryDate           string `parquet:"lastTimeEntryDate"`
	ManagerName                 string `parquet:"manager"`
	ManagerDisplayName          string `parquet:"managerDisplayName"`
	ManagerUid                  string `parquet:"managerUid"`
	ProjectAwardDate            string `parquet:"projectAwardDate"`
	Proposal                    string `parquet:"proposal"`
	ProposalId                  string `parquet:"proposal_id"`
	ProposalOpeningDate         string `parquet:"proposalOpeningDate"`
	ProposalSubmissionDueDate   string `parquet:"proposalSubmissionDueDate"`
	Status                      string `parquet:"status"`
	Timestamp                   string `parquet:"timestamp"`
	Id                          string `parquet:"pocketbase_id"`
	TClient                     string `parquet:"t_client"`
	TClientContact              string `parquet:"t_clientContact"`
	Client                      string `parquet:"client_id"`
	Contact                     string `parquet:"contact_id"`
	Manager                     string `parquet:"manager_id"`
	AlternateManagerId          string `parquet:"alternate_manager_id"`
	Branch                      string `parquet:"branch"`
	Parent                      string `parquet:"parent_id"`
}

type Profile struct {
	Id                             string  `parquet:"id"`                             // legacy Firebase UID
	Surname                        string  `parquet:"surname"`                        // profiles, users
	GivenName                      string  `parquet:"givenName"`                      // profiles, users
	OpeningDateTimeOff             string  `parquet:"openingDateTimeOff"`             // admin_profiles
	OpeningOP                      float64 `parquet:"openingOP"`                      // admin_profiles
	OpeningOV                      float64 `parquet:"openingOV"`                      // admin_profiles
	UntrackedTimeOff               bool    `parquet:"untrackedTimeOff"`               // admin_profiles
	DefaultChargeOutRate           float64 `parquet:"defaultChargeOutRate"`           // admin_profiles
	Email                          string  `parquet:"email"`                          // users
	MobilePhone                    string  `parquet:"mobilePhone"`                    // admin_profiles
	JobTitle                       string  `parquet:"jobTitle"`                       // admin_profiles
	AzureId                        string  `parquet:"azureId"`                        // _externalAuths
	Salary                         bool    `parquet:"salary"`                         // admin_profiles
	DefaultDivision                string  `parquet:"pocketbase_defaultDivision"`     // profiles
	ManagerId                      string  `parquet:"pocketbase_manager"`             // profiles
	TimeSheetExpected              bool    `parquet:"timeSheetExpected"`              // admin_profiles
	PayrollId                      string  `parquet:"payrollId"`                      // admin_profiles
	OffRotationPermitted           bool    `parquet:"offRotation"`                    // admin_profiles
	PersonalVehicleInsuranceExpiry string  `parquet:"personalVehicleInsuranceExpiry"` // admin_profiles
	AllowPersonalReimbursement     bool    `parquet:"allowPersonalReimbursement"`     // admin_profiles
	SkipMinTimeCheckOnNextBundle   bool    `parquet:"skipMinTimeCheckOnNextBundle"`   // admin_profiles
	WorkWeekHours                  float64 `parquet:"workWeekHours"`                  // admin_profiles
	AlternateManager               string  `parquet:"pocketbase_alternateManager"`    // profiles
	DoNotAcceptSubmissions         bool    `parquet:"doNotAcceptSubmissions"`         // profiles
	PocketbaseId                   string  `parquet:"pocketbase_id"`
	UserId                         string  `parquet:"pocketbase_uid"` // profiles, users, admin_profiles
	DefaultBranch                  string  `parquet:"defaultBranch"`  // admin_profiles
}

type Category struct {
	Id   string `parquet:"id"`
	Name string `parquet:"name"`
	Job  string `parquet:"job"`
}

type TimeSheet struct {
	Id            string  `parquet:"pocketbase_id"`
	Uid           string  `parquet:"pocketbase_uid"`
	WeekEnding    string  `parquet:"weekEnding"`
	ApproverUid   string  `parquet:"pocketbase_approver_uid"`
	PayrollId     string  `parquet:"payrollId"`
	Salary        bool    `parquet:"salary"`
	WorkWeekHours float64 `parquet:"workWeekHours"`
}

type TimeEntry struct {
	Id                  string  `parquet:"pocketbase_id"`
	UserId              string  `parquet:"pocketbase_uid"`
	Job                 string  `parquet:"pocketbase_jobid"`
	TimeType            string  `parquet:"timetype_id"`
	Division            string  `parquet:"division_id"`
	TimeSheet           string  `parquet:"pocketbase_tsid"`
	Date                string  `parquet:"date"`
	WeekEnding          string  `parquet:"week_ending"`
	WorkRecord          string  `parquet:"workrecord"`
	Hours               float64 `parquet:"hours"`
	JobHours            float64 `parquet:"jobHours"`
	MealsHours          float64 `parquet:"mealsHours"`
	Description         string  `parquet:"workDescription"`
	PayoutRequestAmount float64 `parquet:"payoutRequestAmount"`
	Category            string  `parquet:"category_id"`
}

type TimeAmendment struct {
	Id                  string    `parquet:"pocketbase_id"`
	Creator             string    `parquet:"pocketbase_creator_uid"`
	Committer           string    `parquet:"pocketbase_commit_uid"`
	Created             string    `parquet:"created"`
	CommittedWeekEnding string    `parquet:"committedWeekEnding"`
	User                string    `parquet:"pocketbase_uid"`
	PayrollId           string    `parquet:"payrollId"`
	WorkWeekHours       float64   `parquet:"workWeekHours"`
	Salary              bool      `parquet:"salary"`
	WeekEnding          string    `parquet:"weekEnding"`
	Date                string    `parquet:"date"`
	TimeType            string    `parquet:"timetype_id"`
	Division            string    `parquet:"division_id"`
	Job                 string    `parquet:"pocketbase_jobid"`
	WorkRecord          string    `parquet:"workrecord"`
	Hours               float64   `parquet:"hours"`
	JobHours            float64   `parquet:"jobHours"`
	MealsHours          float64   `parquet:"mealsHours"`
	Description         string    `parquet:"workDescription"`
	PayoutRequestAmount float64   `parquet:"payoutRequestAmount"`
	TimeSheet           string    `parquet:"pocketbase_tsid"`
	Category            string    `parquet:"category_id"`
	Committed           time.Time `parquet:"commitTime"`
}

type Vendor struct {
	Id   string `parquet:"id"`
	Name string `parquet:"name"`
}

type Expense struct {
	Id                  string    `parquet:"pocketbase_id"`
	Uid                 string    `parquet:"pocketbase_uid"`
	PayrollId           string    `parquet:"payrollId"`
	Division            string    `parquet:"division_id"`
	Job                 string    `parquet:"pocketbase_jobid"`
	Category            string    `parquet:"category_id"`
	Date                string    `parquet:"date"`
	PayPeriodEnding     string    `parquet:"payPeriodEnding"`
	Description         string    `parquet:"description"`
	Breakfast           bool      `parquet:"breakfast"`
	Lunch               bool      `parquet:"lunch"`
	Dinner              bool      `parquet:"dinner"`
	Lodging             bool      `parquet:"lodging"`
	Vendor              string    `parquet:"vendor_id"`
	Distance            float64   `parquet:"distance"`
	Total               float64   `parquet:"total"`
	PaymentType         string    `parquet:"paymentType"`
	Attachment          string    `parquet:"destination_attachment"`
	CCLast4Digits       string    `parquet:"ccLast4Digits_string"`
	PurchaseOrderNumber string    `parquet:"po"`
	PurchaseOrderId     string    `parquet:"purchase_order_id"`
	Approver            string    `parquet:"pocketbase_approver_uid"`
	Committer           string    `parquet:"pocketbase_commit_uid"`
	Committed           time.Time `parquet:"commitTime"`
	CommittedWeekEnding string    `parquet:"committedWeekEnding"`
}

type UserClaim struct {
	Uid string `parquet:"uid"`
	Cid string `parquet:"cid"`
}

type MileageResetDate struct {
	Id   string `parquet:"pocketbase_id"`
	Date string `parquet:"date"`
}

type PurchaseOrder struct {
	Id          string  `parquet:"id"`
	PoNumber    string  `parquet:"number"`
	Approver    string  `parquet:"approver"`
	Date        string  `parquet:"date"`
	Vendor      string  `parquet:"vendor"`
	Uid         string  `parquet:"uid"`
	Total       float64 `parquet:"total"`
	PaymentType string  `parquet:"payment_type"`
	Job         string  `parquet:"job"`
	Division    string  `parquet:"division"`
}

// JobTimeAllocation represents a single division-hours allocation for a job.
type JobTimeAllocation struct {
	Id       string  `parquet:"id"`
	Job      string  `parquet:"job"`
	Division string  `parquet:"division"`
	Hours    float64 `parquet:"hours"`
}

// FromParquet reads data from a Parquet file and inserts it into a SQLite table using a generic approach.
// T: The struct type corresponding to the Parquet file schema.
// parquetFilePath: Path to the input Parquet file.
// sqliteDBPath: Path to the target SQLite database file.
// sqliteTableName: Name of the target table (used for logging).
// insertSQL: The parameterized INSERT statement (e.g., "INSERT INTO table (col1, col2) VALUES ({:p1}, {:p2})").
// binder: A function that takes an item of type T and returns a dbx.Params map suitable for binding to insertSQL.
// upsert: If true, uses INSERT OR REPLACE to handle duplicate keys gracefully.
func FromParquet[T any](parquetFilePath string, sqliteDBPath string, sqliteTableName string, insertSQL string, binder func(item T) dbx.Params, upsert bool) {
	// --- Parquet Reading (Generic) ---
	items, err := parquet.ReadFile[T](parquetFilePath)
	if err != nil {
		panic(err)
	}

	// connect to the sqlite database with the dbx package
	db, err := dbx.Open("sqlite", sqliteDBPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Convert INSERT to INSERT OR REPLACE if upsert is true
	if upsert {
		// Use regex to handle whitespace before INSERT INTO
		re := regexp.MustCompile(`(?i)\s*INSERT\s+INTO`)
		insertSQL = re.ReplaceAllString(insertSQL, " INSERT OR REPLACE INTO")
	}

	fmt.Printf("Inserting %d items into %s...\n", len(items), sqliteTableName)
	insertedCount := 0
	failureCount := 0

	// Prepare the query once (more efficient)
	q := db.NewQuery(insertSQL)

	// Iterate through the items (still includes potentially bad first item)
	for _, item := range items {
		params := binder(item) // Use the provided binder function
		_, err := q.Bind(params).Execute()
		if err != nil {
			// Log error and continue? Or stop? For now, log and count failures.
			log.Printf("WARN: Failed to insert item into %s: %v. Item: %+v", sqliteTableName, err, item)
			failureCount++
			continue // Continue with the next item
		}
		insertedCount++
	}

	fmt.Printf("Finished insertion into %s: %d successful, %d failures\n", sqliteTableName, insertedCount, failureCount)

}
