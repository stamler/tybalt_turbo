// In this file we define utility functions for the hooks.

package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/types"
)

func isValidDate(value interface{}) error {
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
func generateWeekEnding(date string) (string, error) {
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
func generatePayPeriodEnding(date string) (string, error) {
	weekEnding, err := generateWeekEnding(date)
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

func isPositiveMultipleOfPointFive() validation.RuleFunc {
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

func isPositiveMultipleOfPointZeroOne() validation.RuleFunc {
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
// distance: the distance of the expense
// expenseDate: the date of the expense
// startDate: the start date of the annual period derived from the payroll_year_end_dates collection
// expenseRateRecord: the expense rate record retrieved from the expense_rates collection
func calculateMileageTotal(app *pocketbase.PocketBase, distance float64, startDate string, expenseDate string, expenseRateRecord *models.Record) (float64, error) {
	// the mileage property on the expense_rate record is a JSON object with
	// keys that represent the lower bound of the distance band and a value
	// that represents the rate for that distance band. We extract the mileage
	// property JSON string into a map[string]interface{} and then set the
	// total field on the expense record.
	var mileageRates map[string]interface{}
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

	// TODO: we're getting some rounding errors as a result of float64 arithmetic
	// that we should address.

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
		"expenseDate": expenseDate,
	}).All(&results)

	// total mileage is the sum of all mileage expenses in the annual period
	// prior to the date of this expense.
	totalMileage := results[0].TotalMileage

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
		if float64(band) <= totalMileage {
			lowerDistanceBand = band
			// upperDistanceBand is always at least as large as lowerDistanceBand
			upperDistanceBand = band
		}
		if float64(band) <= totalMileage+distance {
			upperDistanceBand = band
		} else {
			break
		}
	}

	// If the lower and upper distance bands are the same, we can use the
	// mileage rate for that distance band and simply multiply the distance by
	// the rate.
	if lowerDistanceBand == upperDistanceBand {
		expenseRate := mileageRates[strconv.Itoa(lowerDistanceBand)].(float64)
		return distance * expenseRate, nil
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
					lowerDistanceBandRate := mileageRates[strconv.Itoa(lowerDistanceBand)].(float64)
					upperDistanceBandRate := mileageRates[strconv.Itoa(upperDistanceBand)].(float64)
					lowerDistanceBandMileage := float64(upperDistanceBand) - totalMileage
					upperDistanceBandMileage := distance - lowerDistanceBandMileage
					return lowerDistanceBandMileage*lowerDistanceBandRate + upperDistanceBandMileage*upperDistanceBandRate, nil

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

					var totalExpense float64
					remainingDistance := distance
					currentMileage := totalMileage

					// Handle the lowest distance band
					lowestBandRate := mileageRates[strconv.Itoa(lowerDistanceBand)].(float64)
					lowestBandMileage := float64(distanceBands[i+1]) - currentMileage
					totalExpense += lowestBandMileage * lowestBandRate
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
						if float64(distanceBands[j+1]) > currentMileage+remainingDistance {
							break
						}
						middleBandRate := mileageRates[strconv.Itoa(distanceBands[j])].(float64)
						middleBandMileage := float64(distanceBands[j+1] - distanceBands[j])
						totalExpense += middleBandMileage * middleBandRate
						remainingDistance -= middleBandMileage
						currentMileage += middleBandMileage
					}

					// Handle the highest distance band
					highestBandRate := mileageRates[strconv.Itoa(upperDistanceBand)].(float64)
					totalExpense += remainingDistance * highestBandRate

					return totalExpense, nil
				}
			}
		}

		return 0, errors.New("no mileage rates found for the expense record")
	}
}

// when given a date string in the format "YYYY-MM-DD", return the date string
// representing the first day of the annual payroll period.
func getAnnualPayrollPeriodStartDate(app *pocketbase.PocketBase, date string) (string, error) {
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
