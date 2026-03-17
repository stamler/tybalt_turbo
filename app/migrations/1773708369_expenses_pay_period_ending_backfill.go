package migrations

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type expensePayPeriodBackfillRow struct {
	ID                  string `db:"id"`
	Date                string `db:"date"`
	Committed           string `db:"committed"`
	CommittedWeekEnding string `db:"committed_week_ending"`
}

// This migration normalizes pay_period_ending for locally-owned expense rows
// after the application changed from "date-derived on every save" to
// "server-managed and assigned at commit time".
//
// Historical context:
//   - Older expense behavior populated pay_period_ending during ordinary
//     create/update flows by deriving it directly from the expense date.
//   - New behavior leaves pay_period_ending blank until the expense is
//     committed, then assigns it using commit-time payroll bucketing logic.
//   - A preceding schema/rule migration updated validation to make
//     pay_period_ending optional and server-managed, but existing rows still
//     needed data cleanup to match the new semantics.
//
// Scope:
//   - Only expenses with _imported = false are touched.
//   - Imported rows are intentionally excluded because they originate from an
//     external system and may follow different synchronization/backfill rules.
//
// Normalization rules applied here:
//   1. Local, uncommitted expenses:
//      pay_period_ending is cleared to the empty string. Under the new model,
//      draft/submitted/approved expenses should not advertise a payroll bucket
//      before commit has actually happened.
//   2. Local, committed expenses:
//      pay_period_ending is recomputed using the same commit-time bucketing
//      rule now used by the application when an expense is committed.
//
// Data requirements and fallbacks:
//   - The preferred source of truth for committed rows is
//     committed_week_ending.
//   - Some historical rows may have committed populated while
//     committed_week_ending is blank. For those rows, this migration derives a
//     week ending from the committed timestamp's date portion so the backfill
//     can still complete.
//
// Why the helper logic is duplicated here instead of importing utilities:
//   - The test app imports the migrations package when seeding PocketBase.
//   - Importing tybalt/utilities from this migration creates an import cycle in
//     tests because utilities tests depend on the seeded test app.
//   - Keeping small, migration-local helper functions avoids that cycle and
//     makes the migration self-contained and stable over time.
//
// Rollback note:
//   - The down migration is intentionally a no-op.
//   - Once rows are normalized to the new semantics, there is no reliable way
//     to reconstruct the previous, date-derived pay_period_ending values for
//     every record without reintroducing the old business rule.
func init() {
	m.Register(func(app core.App) error {
		return app.RunInTransaction(func(txApp core.App) error {
			if _, err := txApp.DB().NewQuery(`
				UPDATE expenses
				SET pay_period_ending = ''
				WHERE _imported = 0
				  AND COALESCE(committed, '') = ''
				  AND COALESCE(pay_period_ending, '') != ''
			`).Execute(); err != nil {
				return err
			}

			var rows []expensePayPeriodBackfillRow
			if err := txApp.DB().NewQuery(`
				SELECT
					id,
					date,
					COALESCE(committed, '') AS committed,
					COALESCE(committed_week_ending, '') AS committed_week_ending
				FROM expenses
				WHERE _imported = 0
				  AND COALESCE(committed, '') != ''
			`).All(&rows); err != nil {
				return err
			}

			for _, row := range rows {
				weekEnding := row.CommittedWeekEnding
				if weekEnding == "" {
					committedDate, err := committedDateOnly(row.Committed)
					if err != nil {
						return fmt.Errorf("derive committed date for expense %s: %w", row.ID, err)
					}

					weekEnding, err = generateWeekEndingForMigration(committedDate)
					if err != nil {
						return fmt.Errorf("derive committed week ending for expense %s: %w", row.ID, err)
					}
				}

				payPeriodEnding, err := generateCommittedPayPeriodEndingForMigration(row.Date, weekEnding)
				if err != nil {
					return fmt.Errorf("generate pay period ending for expense %s: %w", row.ID, err)
				}

				if _, err := txApp.DB().NewQuery(`
					UPDATE expenses
					SET pay_period_ending = {:pay_period_ending}
					WHERE id = {:id}
				`).Bind(dbx.Params{
					"id":                row.ID,
					"pay_period_ending": payPeriodEnding,
				}).Execute(); err != nil {
					return err
				}
			}

			return nil
		})
	}, func(app core.App) error {
		return nil
	})
}

func committedDateOnly(committed string) (string, error) {
	trimmed := strings.TrimSpace(committed)
	if len(trimmed) < len(time.DateOnly) {
		return "", fmt.Errorf("invalid committed value %q", committed)
	}

	dateOnly := trimmed[:len(time.DateOnly)]
	if _, err := time.Parse(time.DateOnly, dateOnly); err != nil {
		return "", err
	}

	return dateOnly, nil
}

func generateWeekEndingForMigration(date string) (string, error) {
	t, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return "", err
	}

	for t.Weekday() != time.Saturday {
		t = t.AddDate(0, 0, 1)
	}

	return t.Format(time.DateOnly), nil
}

func generatePayPeriodEndingForMigration(date string) (string, error) {
	weekEnding, err := generateWeekEndingForMigration(date)
	if err != nil {
		return "", err
	}

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

	if int(intervalHours)%24 != 0 {
		return "", fmt.Errorf("interval hours is not a multiple of 24")
	}

	return weekEndingTime.AddDate(0, 0, 7).Format(time.DateOnly), nil
}

func generateCommittedPayPeriodEndingForMigration(expenseDate string, committedWeekEnding string) (string, error) {
	payPeriodEnding, err := generatePayPeriodEndingForMigration(committedWeekEnding)
	if err != nil {
		return "", err
	}

	if payPeriodEnding == committedWeekEnding {
		return committedWeekEnding, nil
	}

	committedWeekEndingTime, err := time.Parse(time.DateOnly, committedWeekEnding)
	if err != nil {
		return "", err
	}
	expenseDateTime, err := time.Parse(time.DateOnly, expenseDate)
	if err != nil {
		return "", err
	}

	previousPayPeriodEnding := committedWeekEndingTime.AddDate(0, 0, -7)
	if !expenseDateTime.After(previousPayPeriodEnding) {
		return previousPayPeriodEnding.Format(time.DateOnly), nil
	}

	return committedWeekEndingTime.AddDate(0, 0, 7).Format(time.DateOnly), nil
}
