// queue_events.go contains immediate event-driven notification queue functions.
//
// These paths build event payloads/recipients for rejection and sharing events,
// then create and send notifications immediately via shared fan-out helpers.
package notifications

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// QueueTimesheetRejectedNotifications creates immediate notifications for a
// rejected timesheet.
//
// Recipients include the employee, rejector (if different), and the employee's
// manager (if distinct). Send errors are logged and do not fail the overall
// business operation.
func QueueTimesheetRejectedNotifications(app core.App, timesheet *core.Record, rejectorUID, reason string) error {
	employeeUID := timesheet.GetString("uid")
	weekEnding := timesheet.GetString("week_ending")

	employeeName, employeeProfile, err := getProfileDisplayName(app, employeeUID)
	if err != nil {
		app.Logger().Error(
			"error finding employee profile",
			"employee_uid", employeeUID,
			"error", err,
		)
		return fmt.Errorf("error finding employee profile: %v", err)
	}

	rejectorName, _, err := getProfileDisplayName(app, rejectorUID)
	if err != nil {
		app.Logger().Error(
			"error finding rejector profile",
			"rejector_uid", rejectorUID,
			"error", err,
		)
		return fmt.Errorf("error finding rejector profile: %v", err)
	}

	managerUID := employeeProfile.GetString("manager")

	data := map[string]any{
		"EmployeeName":    employeeName,
		"WeekEnding":      weekEnding,
		"RejectorName":    rejectorName,
		"RejectionReason": reason,
		"ActionURL":       BuildActionURL(app, fmt.Sprintf("/time/sheets/%s/details", timesheet.Id)),
	}

	recipients := []string{employeeUID}
	if rejectorUID != employeeUID {
		recipients = append(recipients, rejectorUID)
	}
	if managerUID != "" && managerUID != rejectorUID && managerUID != employeeUID {
		recipients = append(recipients, managerUID)
	}

	createdCount := createAndSendToRecipients(
		app,
		"timesheet_rejected",
		recipients,
		data,
		true,
		"",
		map[string]any{"timesheet_id": timesheet.Id},
	)

	app.Logger().Info(
		"created timesheet rejection notifications",
		"timesheet_id", timesheet.Id,
		"created_count", createdCount,
		"recipient_count", len(recipients),
	)

	return nil
}

// QueueExpenseRejectedNotifications creates immediate notifications for a
// rejected expense.
//
// Recipients include the employee, rejector (if different), and the employee's
// manager (if distinct). The payload includes expense date/amount and rejection
// context for template rendering.
func QueueExpenseRejectedNotifications(app core.App, expense *core.Record, rejectorUID, reason string) error {
	employeeUID := expense.GetString("uid")
	expenseDate := expense.GetString("date")
	expenseTotal := expense.GetFloat("total")
	expenseAmount := fmt.Sprintf("$%.2f", expenseTotal)

	employeeName, employeeProfile, err := getProfileDisplayName(app, employeeUID)
	if err != nil {
		app.Logger().Error(
			"error finding employee profile",
			"employee_uid", employeeUID,
			"error", err,
		)
		return fmt.Errorf("error finding employee profile: %v", err)
	}

	rejectorName, _, err := getProfileDisplayName(app, rejectorUID)
	if err != nil {
		app.Logger().Error(
			"error finding rejector profile",
			"rejector_uid", rejectorUID,
			"error", err,
		)
		return fmt.Errorf("error finding rejector profile: %v", err)
	}

	managerUID := employeeProfile.GetString("manager")

	data := map[string]any{
		"EmployeeName":    employeeName,
		"ExpenseDate":     expenseDate,
		"ExpenseAmount":   expenseAmount,
		"RejectorName":    rejectorName,
		"RejectionReason": reason,
		"ActionURL":       BuildActionURL(app, fmt.Sprintf("/expenses/%s/details", expense.Id)),
	}

	recipients := []string{employeeUID}
	if rejectorUID != employeeUID {
		recipients = append(recipients, rejectorUID)
	}
	if managerUID != "" && managerUID != rejectorUID && managerUID != employeeUID {
		recipients = append(recipients, managerUID)
	}

	createdCount := createAndSendToRecipients(
		app,
		"expense_rejected",
		recipients,
		data,
		true,
		"",
		map[string]any{"expense_id": expense.Id},
	)

	app.Logger().Info(
		"created expense rejection notifications",
		"expense_id", expense.Id,
		"created_count", createdCount,
		"recipient_count", len(recipients),
	)

	return nil
}

// QueueTimesheetSharedNotifications creates immediate notifications for newly
// added timesheet viewers.
//
// It no-ops when no new viewers are provided and sends one notification per new
// viewer with sharer, employee, and week-ending context.
func QueueTimesheetSharedNotifications(app core.App, timesheet *core.Record, sharerUID string, newViewerUIDs []string) error {
	if len(newViewerUIDs) == 0 {
		return nil
	}

	employeeUID := timesheet.GetString("uid")
	weekEnding := timesheet.GetString("week_ending")

	sharerName, _, err := getProfileDisplayName(app, sharerUID)
	if err != nil {
		app.Logger().Error(
			"error finding sharer profile",
			"sharer_uid", sharerUID,
			"error", err,
		)
		return fmt.Errorf("error finding sharer profile: %v", err)
	}

	employeeName, _, err := getProfileDisplayName(app, employeeUID)
	if err != nil {
		app.Logger().Error(
			"error finding employee profile",
			"employee_uid", employeeUID,
			"error", err,
		)
		return fmt.Errorf("error finding employee profile: %v", err)
	}

	data := map[string]any{
		"UserName":     sharerName,
		"EmployeeName": employeeName,
		"WeekEnding":   weekEnding,
		"ActionURL":    BuildActionURL(app, fmt.Sprintf("/time/sheets/%s/details", timesheet.Id)),
	}

	createdCount := createAndSendToRecipients(
		app,
		"timesheet_shared",
		newViewerUIDs,
		data,
		true,
		"",
		map[string]any{"timesheet_id": timesheet.Id},
	)

	app.Logger().Info(
		"created timesheet shared notifications",
		"timesheet_id", timesheet.Id,
		"created_count", createdCount,
	)

	return nil
}
