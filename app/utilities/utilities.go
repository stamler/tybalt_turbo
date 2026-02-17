// In this file we define utility functions for hooks and routes

package utilities

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"time"
	"tybalt/errs"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// IsUserActive checks if a user has an admin_profiles record with active=true.
// Returns (false, nil) if the user has no admin_profiles record or active=false.
// Returns (false, err) if there's a database error (callers should handle as 500).
// This is used to prevent inactive users from being assigned to manager/approver fields.
func IsUserActive(app core.App, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	adminProfile, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No admin_profiles record = treat as inactive
			return false, nil
		}
		// Real DB error - propagate it
		return false, err
	}
	return adminProfile.GetBool("active"), nil
}

func IsValidDate(value any) error {
	s, _ := value.(string)
	if _, err := time.Parse(time.DateOnly, s); err != nil {
		return validation.NewError("validation_invalid_date", s+" is not a valid date")
	}
	return nil
}

// generate the week ending date from the date property. The week ending date is
// the Saturday immediately following the date property. If the argument is
// already a Saturday, it is returned unchanged. The date property is a string
// in the format "YYYY-MM-DD".
func GenerateWeekEnding(date string) (string, error) {
	// parse the date string
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return "", err
	}

	// add days to the date until it is a Saturday
	for t.Weekday() != time.Saturday {
		t = t.AddDate(0, 0, 1)
	}

	// return the date as a string
	return t.Format(time.DateOnly), nil
}

// generate the pay period ending date from the date property. This is almost
// exactly the same as generateWeekEnding except that the value returned is
// every second Saturday rather than every Saturday. Thus we will call
// generateWeekEnding then check if the result is correct. If it isn't we'll
// return the date 7 days later.
func GeneratePayPeriodEnding(date string) (string, error) {
	weekEnding, err := GenerateWeekEnding(date)
	if err != nil {
		return "", err
	}

	// check if the day difference between the weekEnding and the epoch pay period
	// ending (August 31, 2024) has a remainder of 0 when divided by 14 (modulo
	// 14). If the remainder is zero, return the week ending. If not, return the
	// week ending plus 7 days. Remember date is a string in the format
	// "YYYY-MM-DD" and we don't want to worry about time zones and such.
	epochPayPeriodEnding, err := time.Parse(time.DateOnly, "2024-08-31")
	if err != nil {
		return "", err
	}

	weekEndingTime, err := time.Parse(time.DateOnly, weekEnding)
	if err != nil {
		return "", err
	}

	intervalHours := weekEndingTime.Sub(epochPayPeriodEnding).Hours()
	if int(intervalHours/24)%14 == 0 {
		return weekEnding, nil
	}

	// check that there isn't an hour error caused by time zone differences
	if int(intervalHours)%24 != 0 {
		return "", fmt.Errorf("interval hours is not a multiple of 24")
	}

	return weekEndingTime.AddDate(0, 0, 7).Format(time.DateOnly), nil
}

// DateStringLimit returns a validation.RuleFunc that validates that the value is
// a date string that is on or after the date string passed to the function or,
// if max is true, on or before the date string passed to the function.
func DateStringLimit(limit time.Time, max bool) validation.RuleFunc {
	return func(value any) error {
		date, _ := value.(string)
		dateAsTime, err := time.Parse(time.DateOnly, date)
		if err != nil {
			return err
		}
		if max {
			if dateAsTime.After(limit) {
				return validation.NewError("validation_invalid_date", date+" is too late")
			}
		} else {
			if dateAsTime.Before(limit) {
				return validation.NewError("validation_invalid_date", date+" is too early")
			}
		}
		return nil
	}
}

// PoApproverPropsHasDivisionPermission returns a validation function that checks if the
// provided user ID (as the value parameter) has permission to approve purchase
// orders for the specified division with the given claim. Permission is granted if
// either:
// 1. The po_approver_props record's divisions property is missing or
// 2. The po_approver_props record's divisions property contains the specified divisionId
func PoApproverPropsHasDivisionPermission(app core.App, claimId string, divisionId string) validation.RuleFunc {
	return func(value any) error {
		// fast fail if the value is nil
		if value == nil {
			return validation.NewError("value_required", "value is required")
		}
		// fast fail if the value is an empty string
		if value == "" {
			return validation.NewError("value_required", "value is required")
		}
		userId, _ := value.(string)
		type ClaimResult struct {
			HasClaim bool `db:"has_claim"`
		}
		var result ClaimResult
		app.DB().NewQuery(`
		  SELECT COUNT(*) > 0 AS has_claim 
			FROM user_claims u
			INNER JOIN po_approver_props p ON p.user_claim = u.id
			WHERE u.uid = {:userId} 
			AND u.cid = {:claimId}
			AND (JSON_ARRAY_LENGTH(p.divisions) = 0 OR EXISTS (SELECT 1 FROM JSON_EACH(p.divisions) WHERE value = {:divisionId}))
		`).Bind(dbx.Params{
			"userId":     userId,
			"claimId":    claimId,
			"divisionId": divisionId,
		}).One(&result)
		if !result.HasClaim {
			return validation.NewError("validation_no_claim", "user does not have the required claim")
		}
		return nil
	}
}

func IsPositiveMultipleOfPointFive() validation.RuleFunc {
	return func(value any) error {
		s, _ := value.(float64)
		if s == 0 {
			return nil
		}
		if s < 0 {
			return validation.NewError("validation_negative_number", "must be a positive number")
		}
		// return error is s is not a multiple of 0.5
		if s/0.5 != float64(int(s/0.5)) {
			return validation.NewError("validation_not_multiple_of_point_five", "must be a multiple of 0.5")
		}
		return nil
	}
}

func IsPositiveMultipleOfPointZeroOne() validation.RuleFunc {
	return func(value any) error {
		s, _ := value.(float64)
		if s == 0 {
			return nil
		}
		if s < 0 {
			return validation.NewError("validation_negative_number", "must be a positive number")
		}
		// return error is s is not a multiple of 0.1
		if s/0.01 != float64(int(s/0.01)) {
			return validation.NewError("validation_not_multiple_of_point_zero_one", "must be a multiple of 0.01")
		}
		return nil
	}
}

// arguments:
// expenseRecord: the expense record
// expenseRateRecord: the expense rate record retrieved from the expense_rates collection
func CalculateMileageTotal(app core.App, expenseRecord *core.Record, expenseRateRecord *core.Record) (float64, error) {
	distance := expenseRecord.GetFloat("distance")
	// check if the distance is an integer
	if distance != float64(int(distance)) {
		return 0, errors.New("distance must be an integer for mileage expenses")
	}

	// Determine reset period start. Historically we used payroll_year_end_dates to
	// bound the annual period. To align with reporting/backfill SQL, we now use
	// mileage_reset_dates and select the last reset date on or before the expense
	// date. This mirrors the partitioning used in reports (per user, per reset).
	expenseDate := expenseRecord.GetString("date")
	type DateResult struct {
		Date string `db:"date"`
	}
	reset := DateResult{}
	// Use COALESCE to avoid NULL scan issues when there are no reset rows <= date.
	// We intentionally coalesce to the empty string and handle the fallback below
	// to keep the behavior explicit in code.
	if err := app.DB().NewQuery(`SELECT COALESCE(MAX(date), '') AS date FROM mileage_reset_dates WHERE date <= {:expenseDate}`).
		Bind(dbx.Params{"expenseDate": expenseDate}).One(&reset); err != nil {
		return 0, fmt.Errorf("fetch mileage reset date: %w", err)
	}
	resetDate := reset.Date
	if resetDate == "" {
		// If no reset date exists, start from a very early date
		resetDate = "0001-01-01"
	}

	// Compute cumulative mileage prior to this expense for the same user and within
	// the reset period. We include same-day rows but break ties deterministically
	// by id (rows with smaller id are considered earlier), matching the SQL logic
	// used by reports/backfill.
	uid := expenseRecord.GetString("uid")
	type SumResult struct {
		TotalMileage float64 `db:"total_mileage"`
	}
	res := SumResult{}
	if err := app.DB().NewQuery(`
		SELECT COALESCE(SUM(distance), 0) AS total_mileage
		FROM expenses
		WHERE payment_type = 'Mileage'
		  AND committed != ''
		  AND uid = {:uid}
		  AND date >= {:resetDate}
		  AND (
		        date < {:expenseDate}
		     OR (date = {:expenseDate} AND id < {:expenseId})
		  )`).
		Bind(dbx.Params{
			"uid":         uid,
			"resetDate":   resetDate,
			"expenseDate": expenseDate,
			"expenseId":   expenseRecord.Id,
		}).One(&res); err != nil {
		return 0, fmt.Errorf("sum prior mileage: %w", err)
	}
	priorMileage := int(res.TotalMileage)

	// the mileage property on the expense_rate record is a JSON object with
	// keys that represent the lower bound of the distance band and a value
	// that represents the rate for that distance band. We extract the mileage
	// property JSON string into a map[string]float64 for quick lookups.
	var mileageRates map[string]float64
	mileageRatesRaw := expenseRateRecord.Get("mileage")

	if jsonRawData, ok := mileageRatesRaw.(types.JSONRaw); ok {
		if err := json.Unmarshal(jsonRawData, &mileageRates); err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("mileage data is not of type types.JSONRaw")
	}

	// extract all the keys (lower bounds in kilometres) and turn them into an
	// ordered slice of ints so that we can reason about the contiguous bands.
	var distanceBands []int
	for distanceBand := range mileageRates {
		distanceBandInt, err := strconv.Atoi(distanceBand)
		if err != nil {
			return 0, err
		}
		distanceBands = append(distanceBands, distanceBandInt)
	}

	if len(distanceBands) == 0 {
		return 0, errors.New("no mileage rates found")
	}

	// sort the distance bands
	sort.Ints(distanceBands)

	// Compute the start and end cumulative distances for this expense:
	//   start = priorMileage; end = priorMileage + distance.
	// Then split [start, end) across tier intervals and sum overlap × rate.
	totalMileage := priorMileage

	// Find the bounding distance bands that the expense record's distance
	// property falls into. The lower distance band is the largest distance band
	// that is less than the total mileage already used in the annual period. The
	// upper distance band is the largest distance band that is less than the
	// total mileage already used in the annual period plus the distance of the
	// expense.
	var lowerDistanceBand int
	var upperDistanceBand int
	for _, band := range distanceBands {
		if band <= totalMileage {
			lowerDistanceBand = band
			// upperDistanceBand is always at least as large as lowerDistanceBand
			upperDistanceBand = band
		}
		if band <= totalMileage+int(distance) {
			upperDistanceBand = band
		} else {
			break
		}
	}

	// If the lower and upper distance bands are the same, we can use the
	// mileage rate for that distance band and simply multiply the distance by
	// the rate.
	if lowerDistanceBand == upperDistanceBand {
		expenseRate := mileageRates[strconv.Itoa(lowerDistanceBand)]
		amount := float64(int(distance)) * expenseRate
		// Round to 2 decimals for currency: multiply to cents, round, divide back
		return math.Round(amount*100) / 100, nil
	} else {
		// If the lower and upper distance bands are different, There are two possible scenarios:
		// 1. The expense record's distance property spans two distance bands.
		// 2. The expense record's distance property spans three or more distance bands.

		for i, band := range distanceBands {
			if lowerDistanceBand == band {
				if upperDistanceBand == distanceBands[i+1] {
					// Scenario 1: The expense record's distance property spans two distance
					// bands. This means the index of the lower distance band is exactly one
					// less than the index of the upper distance band. The distance bands are already sorted in ascending order. We need to calculate how
					// much mileage lies in the first band and multiply it by the rate for the
					// first band. Then we need to calculate how much mileage lies in the second
					// band and multiply it by the rate for the second band. Finally, we sum
					// these two amounts to get the total mileage expense and return it.
					lowerDistanceBandRateX1000 := int(mileageRates[strconv.Itoa(lowerDistanceBand)] * 1000)
					upperDistanceBandRateX1000 := int(mileageRates[strconv.Itoa(upperDistanceBand)] * 1000)
					lowerDistanceBandMileage := upperDistanceBand - totalMileage
					upperDistanceBandMileage := int(distance) - lowerDistanceBandMileage

					amount := float64(lowerDistanceBandMileage*lowerDistanceBandRateX1000+upperDistanceBandMileage*upperDistanceBandRateX1000) / 1000
					// Round to 2 decimals for currency: multiply to cents, round, divide back
					return math.Round(amount*100) / 100, nil

				} else {
					// Scenario 2: The expense record's distance property spans three or
					// more distance bands. We need to calculate how much mileage lies in
					// the lowest distance band and how much mileage lies in the highest
					// distance band. For each of the middle distance bands, we need to
					// calculate how much mileage lies in each of them but this is just
					// the next distance band minus that distance band. We then multiply
					// each of these amounts by the rate for the corresponding distance
					// band. We then sum these amounts to get the total mileage expense
					// and return it.

					var totalExpense int
					remainingDistance := int(distance)
					currentMileage := totalMileage

					// Handle the lowest distance band
					lowestBandRateX1000 := int(mileageRates[strconv.Itoa(lowerDistanceBand)] * 1000)
					lowestBandMileage := distanceBands[i+1] - currentMileage
					totalExpense += lowestBandMileage * lowestBandRateX1000
					remainingDistance -= lowestBandMileage
					currentMileage += lowestBandMileage

					// Handle middle distance bands. The condition exists to prevent
					// out-of-bounds errors and simultaneously allow the code after the
					// for loop to execute for the highest distance band.
					for j := i + 1; j < len(distanceBands)-1; j++ {
						// If the next distance band is greater than the remaining
						// distance plus the current mileage, we break out of the loop
						// because the highest distance band cannot be greater than the
						// remaining distance plus the current mileage.
						if distanceBands[j+1] > currentMileage+remainingDistance {
							break
						}
						middleBandRateX1000 := int(mileageRates[strconv.Itoa(distanceBands[j])] * 1000)
						middleBandMileage := distanceBands[j+1] - distanceBands[j]
						totalExpense += middleBandMileage * middleBandRateX1000
						remainingDistance -= middleBandMileage
						currentMileage += middleBandMileage
					}

					// Handle the highest distance band
					highestBandRateX1000 := int(mileageRates[strconv.Itoa(upperDistanceBand)] * 1000)
					totalExpense += remainingDistance * highestBandRateX1000

					// Round to 2 decimals for currency: multiply to cents, round, divide back
					return math.Round((float64(totalExpense)/1000)*100) / 100, nil
				}
			}
		}

		return 0, errors.New("no mileage rates found for the expense record")
	}
}

// when given a date string in the format "YYYY-MM-DD", return the date string
// representing the first day of the annual payroll period.
func GetAnnualPayrollPeriodStartDate(app core.App, date string) (string, error) {
	// First we need to determine the current annual period. To do this we use
	// the expenseDate to find the date property of the payroll_year_end_dates
	// collection record that is less than the expenseDate. We then use day
	// after this date as the startDate argument in the calculateMileageTotal
	// function. (Since the payroll year end dates are the last day of the
	// year, we need to use the day after this date for the start of the
	// current annual period.)
	payrollYearEndDatesRecord, err := app.FindRecordsByFilter("payroll_year_end_dates", "date < {:date}", "-date", 1, 0, dbx.Params{
		"date": date,
	})
	if err != nil {
		return "", err
	}
	if len(payrollYearEndDatesRecord) == 0 {
		return "", errors.New("no payroll year end date record found for the given date")
	}
	payrollYearEndDate := payrollYearEndDatesRecord[0].GetString("date")
	payrollYearEndDateAsTime, parseErr := time.Parse(time.DateOnly, payrollYearEndDate)
	if parseErr != nil {
		return "", parseErr
	}
	startDate := payrollYearEndDateAsTime.AddDate(0, 0, 1)
	return startDate.Format(time.DateOnly), nil
}

// this function returns true if the user with uid has the claim with the
// specified name and false otherwise
func HasClaim(app core.App, auth *core.Record, name string) (bool, error) {
	// fast fail if the auth record is nil
	if auth == nil {
		return false, nil
	}
	return HasClaimByUserID(app, auth.Id, name)
}

// HasClaimByUserID returns true if the user with the specified uid has the claim name.
func HasClaimByUserID(app core.App, uid string, name string) (bool, error) {
	if uid == "" {
		return false, nil
	}
	userClaims, err := app.FindRecordsByFilter(
		"user_claims",
		"uid={:uid} && cid.name={:name}",
		"",
		1,
		0,
		dbx.Params{
			"uid":  uid,
			"name": name,
		},
	)
	if err != nil {
		return false, err
	}

	return len(userClaims) > 0, nil
}

func GetExpenseRateRecord(app core.App, expenseRecord *core.Record) (*core.Record, error) {
	// Expense rates are stored in the expense_rates collection in PocketBase.
	// The records have an effective_date property that designates the date the
	// rate is effective. We must fetch the appropriate record from the
	// expense_rates collection based on the expense record's date property.
	expenseDate := expenseRecord.GetString("date")
	expenseDateAsTime, parseErr := time.Parse(time.DateOnly, expenseDate)
	if parseErr != nil {
		return nil, parseErr
	}

	// fetch the expense rate record from the expense_rates collection
	expenseRateRecords, findErr := app.FindRecordsByFilter("expense_rates", "effective_date <= {:expenseDate}", "-effective_date", 1, 0, dbx.Params{
		"expenseDate": expenseDateAsTime.Format("2006-01-02"),
	})
	if findErr != nil {
		return nil, findErr
	}

	// if there are no expense rate records, return an error
	if len(expenseRateRecords) == 0 {
		return nil, errors.New("no expense rate record found for the given date")
	}
	return expenseRateRecords[0], nil
}

func CalculateRecurringPurchaseOrderTotalValue(app core.App, purchaseOrderRecord *core.Record) (int, float64, error) {
	if purchaseOrderRecord.GetDateTime("end_date").IsZero() {
		return 0, 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "end_date is required for recurring purchase orders",
			Data: map[string]errs.CodeError{
				"end_date": {
					Code:    "value_required",
					Message: "end_date is required for recurring purchase orders",
				},
			},
		}
	}
	if purchaseOrderRecord.GetString("frequency") == "" || purchaseOrderRecord.Get("frequency") == nil {
		return 0, 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "frequency is required for recurring purchase orders",
			Data: map[string]errs.CodeError{
				"frequency": {
					Code:    "value_required",
					Message: "frequency is required for recurring purchase orders",
				},
			},
		}
	}

	startDateString := purchaseOrderRecord.GetString("date")
	startDate, parseErr := time.Parse(time.DateOnly, startDateString)
	if parseErr != nil {
		return 0, 0, parseErr
	}
	total := purchaseOrderRecord.GetFloat("total")
	endDateString := purchaseOrderRecord.GetString("end_date")
	endDate, parseErr := time.Parse(time.DateOnly, endDateString)
	if parseErr != nil {
		return 0, 0, parseErr
	}
	frequency := purchaseOrderRecord.GetString("frequency")
	daysDiff := endDate.Sub(startDate).Hours() / 24

	// error if daysDiff is negative or zero
	if daysDiff <= 0 {
		return 0, 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "end_date is before start_date",
			Data: map[string]errs.CodeError{
				"end_date": {Code: "end_date_not_after_start_date", Message: "end_date must be after start_date"},
			},
		}
	}

	var occurrences float64

	switch frequency {
	case "Weekly":
		occurrences = daysDiff / 7
	case "Biweekly":
		occurrences = daysDiff / 14
	case "Monthly":
		occurrences = daysDiff / 30 // Approximation
	default:
		return 0, 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "invalid frequency",
			Data: map[string]errs.CodeError{
				"frequency": {Code: "invalid_frequency", Message: "invalid frequency value"},
			},
		}
	}

	// error if occurrences is less than 2
	if occurrences < 2 {
		return 0, 0, &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "recurring purchase order must occur at least twice",
			Data: map[string]errs.CodeError{
				"global": {Code: "fewer_than_two_occurrences", Message: "recurring purchase order must occur at least twice, adjust either the end_date or the frequency"},
			},
		}
	}

	// calculate totalValue using the integer value of occurrences
	totalValue := total * float64(int(occurrences))
	return int(occurrences), totalValue, nil
}

// return true if the recurring purchase order has been exhausted, false otherwise
func RecurringPurchaseOrderExhausted(app core.App, purchaseOrderRecord *core.Record) (bool, error) {
	// TODO: implement issue #13, check if an expense has been committed for each
	// recurrence of the PO and set the Status to Closed if so, otherwise doing nothing.

	// Count only committed expenses for the purchase order.
	query := app.DB().NewQuery("SELECT COUNT(*) AS count FROM expenses WHERE purchase_order = {:purchaseOrder} AND committed != ''")
	query.Bind(dbx.Params{"purchaseOrder": purchaseOrderRecord.Id})
	type CountResult struct {
		Count int `db:"count"`
	}
	result := CountResult{}
	if err := query.One(&result); err != nil {
		return false, err
	}
	committedExpensesCount := result.Count

	// Calculate the total number of expenses allowed for the purchase order
	maxExpenses, _, err := CalculateRecurringPurchaseOrderTotalValue(app, purchaseOrderRecord)
	if err != nil {
		return false, err
	}

	// return true if the committed expenses count is not less than the maximum
	// allowed, false otherwise
	return !(committedExpensesCount < maxExpenses), nil
}

// return the total of all expenses associated with the purchase order. If
// committedOnly is true, return the total of all committed expenses only. This
// function DOES NOT check if the purchaseOrderRecord is of type Cumulative.
// TODO: test this thoroughly.
func CumulativeTotalExpensesForPurchaseOrder(app core.App, purchaseOrderRecord *core.Record, committedOnly bool) (float64, error) {
	existingExpensesTotal := 0.0
	query := app.DB().NewQuery("SELECT COALESCE(SUM(total), 0) AS total FROM expenses WHERE purchase_order = {:purchaseOrder}")
	if committedOnly {
		query = app.DB().NewQuery("SELECT COALESCE(SUM(total), 0) AS total FROM expenses WHERE purchase_order = {:purchaseOrder} AND committed != ''")
	}
	query.Bind(dbx.Params{"purchaseOrder": purchaseOrderRecord.Id})
	type TotalResult struct {
		Total float64 `db:"total"`
	}
	result := TotalResult{}
	if err := query.One(&result); err != nil {
		return 0, err
	}
	existingExpensesTotal = result.Total
	return existingExpensesTotal, nil
}

// FindRequiredApproverClaimIdForPOAmount takes a purchase order amount and
// returns the claim ID that should be used for full approval based on the
// po_approval_tiers table. A claim ID is always returned unless the amount
// exceeds the maximum tier's limit or the po_approval_tiers table is empty.
func FindRequiredApproverClaimIdForPOAmount(app core.App, amount float64) (string, error) {
	// Find all tiers with max_amount >= amount, ordered by max_amount ascending
	// This will give us the smallest tier that can handle this amount
	tiers, err := app.FindRecordsByFilter(
		"po_approval_tiers",
		"max_amount >= {:amount}",
		"max_amount",
		1, // limit to 1 result (the lowest tier that qualifies)
		0,
		dbx.Params{
			"amount": amount,
		},
	)

	if err != nil {
		return "", fmt.Errorf("error finding approval tier: %v", err)
	}

	// If no tier is found, return an empty string
	if len(tiers) == 0 {
		return "", nil
	}

	// Return the claim ID for the appropriate tier
	claim := tiers[0].GetString("claim")
	return claim, nil
}

// GetBoundClaimIdAndMaxAmount returns the claim ID and max_amount of the
// tier with the lowest or highest max_amount in the po_approval_tiers table.
// If highest is true, the tier with the highest max_amount is returned,
// otherwise the tier with the lowest max_amount is returned.
func GetBoundClaimIdAndMaxAmount(app core.App, highest bool) (string, float64, error) {
	order := "max_amount"
	if highest {
		order = "-max_amount"
	}
	tiers, err := app.FindRecordsByFilter(
		"po_approval_tiers",
		"",
		order,
		1,
		0,
	)

	if err != nil {
		return "", 0, fmt.Errorf("error finding approval tier: %v", err)
	}

	if len(tiers) == 0 {
		return "", 0, fmt.Errorf("no approval tiers found")
	}

	return tiers[0].GetString("claim"), tiers[0].GetFloat("max_amount"), nil
}

// RecordHasMeaningfulChanges returns true when any non-auto-managed field on an
// existing record differs from its original value.
//
// The fields "created" and "updated" are always ignored. Callers can provide
// additional fields to ignore via extraSkipFields.
func RecordHasMeaningfulChanges(record *core.Record, extraSkipFields ...string) bool {
	original := record.Original()

	// Skip auto-managed fields by default.
	skipFields := map[string]bool{
		"updated": true,
		"created": true,
	}
	for _, fieldName := range extraSkipFields {
		skipFields[fieldName] = true
	}

	for _, fieldName := range record.Collection().Fields.FieldNames() {
		if skipFields[fieldName] {
			continue
		}
		if fmt.Sprintf("%v", record.Get(fieldName)) != fmt.Sprintf("%v", original.Get(fieldName)) {
			return true
		}
	}
	return false
}

// MarkImportedFalseIfChanged checks if any field (other than auto-managed fields)
// has changed on an existing record, and if so sets _imported to false. This
// ensures that records edited locally get written back to the legacy system.
// This function should only be called for updates (not creates).
func MarkImportedFalseIfChanged(record *core.Record) {
	if RecordHasMeaningfulChanges(record, "_imported") {
		record.Set("_imported", false)
	}
}

// MarkJobNotImported marks a single job as locally modified for writeback and
// updates its timestamp so updatedAfter filters include it.
func MarkJobNotImported(app core.App, jobID string) error {
	if jobID == "" {
		return nil
	}
	_, err := app.DB().NewQuery(`
		UPDATE jobs
		SET _imported = false,
		    updated = strftime('%Y-%m-%d %H:%M:%fZ', 'now')
		WHERE id = {:jobID}
	`).Bind(dbx.Params{"jobID": jobID}).Execute()
	return err
}

// MarkReferencingJobsNotImported marks jobs that reference refID through the
// specified relation column as locally modified for writeback.
func MarkReferencingJobsNotImported(app core.App, column string, refID string) error {
	if refID == "" {
		return nil
	}

	allowedColumns := map[string]bool{
		"client":    true,
		"job_owner": true,
		"contact":   true,
	}
	if !allowedColumns[column] {
		return fmt.Errorf("invalid jobs reference column: %s", column)
	}

	query := fmt.Sprintf(`
		UPDATE jobs
		SET _imported = false,
		    updated = strftime('%%Y-%%m-%%d %%H:%%M:%%fZ', 'now')
		WHERE %s = {:refID}
	`, column)
	_, err := app.DB().NewQuery(query).Bind(dbx.Params{"refID": refID}).Execute()
	return err
}

// TableHasImportedColumn checks if the given table has an _imported column.
// This is used to determine whether to set _imported = false during operations
// that use direct SQL updates (bypassing PocketBase hooks).
func TableHasImportedColumn(app core.App, tableName string) (bool, error) {
	var columns []struct {
		Name string `db:"name"`
	}
	err := app.DB().NewQuery(fmt.Sprintf("PRAGMA table_info(%s)", tableName)).All(&columns)
	if err != nil {
		return false, fmt.Errorf("error checking table columns: %w", err)
	}
	for _, col := range columns {
		if col.Name == "_imported" {
			return true, nil
		}
	}
	return false, nil
}

// ValidateMachineToken checks if a token matches any unexpired machine_secrets
// record with the specified role. Returns true if the token is valid.
func ValidateMachineToken(app core.App, token string, role string) bool {
	records, err := app.FindRecordsByFilter(
		"machine_secrets",
		"role = {:role} && expiry > {:now}",
		"", // sort
		0,  // limit (0 = all)
		0,  // offset
		dbx.Params{
			"role": role,
			"now":  time.Now().UTC().Format("2006-01-02 15:04:05"),
		},
	)
	if err != nil || len(records) == 0 {
		return false
	}

	for _, record := range records {
		salt := record.GetString("salt")
		storedHash := record.GetString("sha256_hash")
		h := sha256.New()
		h.Write([]byte(salt + token))
		if hex.EncodeToString(h.Sum(nil)) == storedHash {
			return true
		}
	}
	return false
}
