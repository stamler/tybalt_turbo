// In this file we define utility functions for hooks and routes

package utilities

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"sort"
	"strconv"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

// These constants are used to determine whether an expense is within the
// allowed percentage or value of the total of a purchase order. The lesser of
// the two limits is used to determine if the expense is valid.
const MAX_PURCHASE_ORDER_EXCESS_PERCENT = 0.05
const MAX_PURCHASE_ORDER_EXCESS_VALUE = 100.0

func IsValidDate(value interface{}) error {
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

// DateStringMin returns a validation.RuleFunc that validates that the value is
// a date string that is on or after the date string passed to the function or,
// if max is true, on or before the date string passed to the function.
func DateStringLimit(limit time.Time, max bool) validation.RuleFunc {
	return func(value interface{}) error {
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

// approverHasDivisionPermission returns true if the payload of the user_claims
// collection record corresponding to the approverId is null or is a JSON list
// of strings where one list element is the divisionId. Returns false otherwise.
// To accomplish this, it must first get the id of the claim that has a name of
// 'po_approver' and then use that id to load the user_claims record
// corresponding to the approverId and the po_approver claim id. This record is
// then checked to see if its payload is null or a JSON list of strings where
// one list element is the divisionId.
func ApproverHasDivisionPermission(app core.App, approverId string, divisionId string) validation.RuleFunc {
	return func(value interface{}) error {
		poApproverClaim, err := app.Dao().FindFirstRecordByFilter("claims", "name = {:claimName}", dbx.Params{
			"claimName": "po_approver",
		})
		if err != nil {
			return validation.NewError("validation_invalid_claim", "po_approver claim not found")
		}
		userClaimsRecord, err := app.Dao().FindFirstRecordByFilter("user_claims", "uid = {:uid} && cid = {:cid}", dbx.Params{
			"uid": approverId,
			"cid": poApproverClaim.Id,
		})
		if err != nil {
			return validation.NewError("validation_invalid_approver", "approver does not have a po_approver claim")
		}

		// payload is a JSON list of strings. Load it into a []string slice
		divisionIds := []string{}
		divisionIdsJson := userClaimsRecord.Get("payload").(types.JsonRaw)

		// if divisionIdsJson is null, "{}", or "[]" then all divisions are allowed
		if divisionIdsJson.String() == "null" || divisionIdsJson.String() == "[]" || divisionIdsJson.String() == "{}" {
			log.Printf("divisionIdsJson is null, [], or {}, so all divisions are allowed")
			return nil
		}

		jsonErr := json.Unmarshal(divisionIdsJson, &divisionIds)
		if jsonErr != nil {
			return validation.NewError("validation_invalid_division_payload", "division payload is not a valid JSON list of strings")
		}

		if !slices.Contains(divisionIds, divisionId) {
			return validation.NewError("validation_invalid_division", "approver does not have permission for the specified division")
		}
		return nil
	}
}

func IsPositiveMultipleOfPointFive() validation.RuleFunc {
	return func(value interface{}) error {
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
	return func(value interface{}) error {
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
func CalculateMileageTotal(app core.App, expenseRecord *models.Record, expenseRateRecord *models.Record) (float64, error) {
	distance := expenseRecord.GetFloat("distance")
	// check if the distance is an integer
	if distance != float64(int(distance)) {
		return 0, errors.New("distance must be an integer for mileage expenses")
	}

	startDate, err := GetAnnualPayrollPeriodStartDate(app, expenseRecord.GetString("date"))
	if err != nil {
		return 0, err
	}

	// the mileage property on the expense_rate record is a JSON object with
	// keys that represent the lower bound of the distance band and a value
	// that represents the rate for that distance band. We extract the mileage
	// property JSON string into a map[string]interface{} and then set the
	// total field on the expense record.
	var mileageRates map[string]float64
	mileageRatesRaw := expenseRateRecord.Get("mileage")

	if jsonRawData, ok := mileageRatesRaw.(types.JsonRaw); ok {
		if err := json.Unmarshal(jsonRawData, &mileageRates); err != nil {
			return 0, err
		}
	} else {
		return 0, fmt.Errorf("mileage data is not of type types.JsonRaw")
	}

	// Mileage rates are stored in a map[string]interface{} with the keys
	// representing the lower bound of the distance band and the value
	// representing the rate for that distance band. We need to find the
	// rate for the distance band that the expense record's distance
	// property falls into. The keys are strings representing the lower
	// bound in kilometres.

	// extract all the keys and turn them into an ordered slice of ints
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

	// TODO: determine which distance band applies to the expense record by
	// figuring out the total cumulative mileage already used in the annual period
	// and use the appropriate rate. This expense could end up spanning multiple
	// distance bands if the employee has already accumulated enough mileage in
	// the current annual period. In this case we need to break the distance
	// into two parts: the part that applies to the first distance band and the
	// part that applies to the second distance band. We then multiply each part
	// by the appropriate rate and sum the results.

	// First we query the SUM of all mileage expenses that between startDate
	// inclusive and expenseDate exclusive. We exclude the expenseDate because
	// the mileage expense hasn't yet been updated in the database prior to
	// the above query.
	// TODO: Restrict expenses to one Mileage entry per day similar to OR entries in validate_time_entries.go?
	type SumResult struct {
		TotalMileage float64 `db:"total_mileage"`
	}
	results := []SumResult{}
	app.Dao().DB().NewQuery("SELECT COALESCE(SUM(distance), 0) AS total_mileage FROM expenses WHERE payment_type = {:paymentType} AND date >= {:startDate} AND date < {:expenseDate}").Bind(dbx.Params{
		"paymentType": "Mileage",
		"startDate":   startDate,
		"expenseDate": expenseRecord.GetString("date"),
	}).All(&results)

	// total mileage is the sum of all mileage expenses in the annual period
	// prior to the date of this expense.
	totalMileage := int(results[0].TotalMileage)

	// totalMileage represents the total mileage already used in the annual
	// period. Now we need to determine which distance band(s) apply to the
	// expense record by finding the largest distance band that is less than
	// the total mileage already used in the annual period. If the distance
	// is greater than the next distance band minus the total mileage already
	// used in the annual period, we need to break the distance into two parts:
	// the part that applies to the first distance band and the part that
	// applies to the second distance band. We then multiply each part by the
	// appropriate rate and sum the results.

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
		// perform the conversion to float64 at the last possible moment to avoid
		// potential issues with float64 arithmetic.
		return float64(int(distance)*int(expenseRate*1000)) / 1000, nil
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

					// perform the arithmetic in integers to avoid issues with float64
					// arithmetic then convert to float64 at the last possible moment
					return float64(lowerDistanceBandMileage*lowerDistanceBandRateX1000+upperDistanceBandMileage*upperDistanceBandRateX1000) / 1000, nil

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

					return float64(totalExpense) / 1000, nil
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
	payrollYearEndDatesRecord, err := app.Dao().FindRecordsByFilter("payroll_year_end_dates", "date < {:date}", "-date", 1, 0, dbx.Params{
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
func HasClaim(dao *daos.Dao, uid string, name string) (bool, error) {
	userClaims, err := dao.FindRecordsByFilter(
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

func GetExpenseRateRecord(app core.App, expenseRecord *models.Record) (*models.Record, error) {
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
	expenseRateRecords, findErr := app.Dao().FindRecordsByFilter("expense_rates", "effective_date <= {:expenseDate}", "-effective_date", 1, 0, dbx.Params{
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

func CalculateRecurringPurchaseOrderTotalValue(app core.App, purchaseOrderRecord *models.Record) (int, float64, error) {
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
	var occurrences float64

	switch frequency {
	case "Weekly":
		occurrences = daysDiff / 7
	case "Biweekly":
		occurrences = daysDiff / 14
	case "Monthly":
		occurrences = daysDiff / 30 // Approximation
	default:
		return 0, 0, fmt.Errorf("invalid frequency: %s", frequency)
	}

	// calculate totalValue using the integer value of occurrences
	totalValue := total * float64(int(occurrences))
	return int(occurrences), totalValue, nil
}

// return true if the recurring purchase order has been exhausted, false otherwise
func RecurringPurchaseOrderExhausted(app core.App, purchaseOrderRecord *models.Record) (bool, error) {
	// TODO: implement issue #13, check if an expense has been committed for each
	// recurrence of the PO and set the Status to Closed if so, otherwise doing nothing.

	// Count the number of committed expenses for the purchase order
	query := app.Dao().DB().NewQuery("SELECT COUNT(*) AS count FROM expenses WHERE purchase_order = {:purchaseOrder}")
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
func CumulativeTotalExpensesForPurchaseOrder(app core.App, purchaseOrderRecord *models.Record, committedOnly bool) (float64, error) {
	existingExpensesTotal := 0.0
	query := app.Dao().DB().NewQuery("SELECT COALESCE(SUM(total), 0) AS total FROM expenses WHERE purchase_order = {:purchaseOrder}")
	if committedOnly {
		query = app.Dao().DB().NewQuery("SELECT COALESCE(SUM(total), 0) AS total FROM expenses WHERE purchase_order = {:purchaseOrder} AND committed != ''")
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
