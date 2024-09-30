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

func calculateMileageTotal(distance float64, expenseRateRecord *models.Record) (float64, error) {
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

	// for now just use the first rate in the list as a proof of concept
	var expenseRate float64
	if len(distanceBands) > 0 {
		firstBand := strconv.Itoa(distanceBands[0])
		rate, ok := mileageRates[firstBand].(float64)
		if !ok {
			return 0, fmt.Errorf("invalid rate for distance band %s", firstBand)
		}
		expenseRate = rate
	} else {
		return 0, errors.New("no mileage rates found")
	}

	return distance * expenseRate, nil
}
