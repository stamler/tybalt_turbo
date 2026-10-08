package utilities

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"tybalt/errs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const JOB_REQUEST_ABANDONMENT_DAYS = 60

// MaxProjectAuthorizationValue is the largest whole dollar value that JSON clients retain exactly.
const MaxProjectAuthorizationValue = 9007199254740991

// ProjectAuthorizationLimit checks current authority for the full proposed value.
func ProjectAuthorizationLimit(app core.App, userID string, value float64) (limit int, result error) {
	defer func() {
		if result != nil {
			result = &errs.HookError{Status: http.StatusBadRequest, Message: result.Error()}
		}
	}()
	if value <= 0 || value > MaxProjectAuthorizationValue || math.IsNaN(value) || math.IsInf(value, 0) || math.Trunc(value) != value {
		return 0, fmt.Errorf("project value must be a positive whole dollar amount")
	}
	active, err := IsUserActive(app, userID)
	if err != nil {
		return 0, err
	}
	if !active {
		return 0, fmt.Errorf("approver must be active")
	}
	assignment, err := app.FindFirstRecordByFilter("user_claims", "uid = {:uid} && cid.name = 'project_authorization'", dbx.Params{"uid": userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("approver must have project_authorization authority")
		}
		return 0, err
	}
	props, err := app.FindFirstRecordByData("project_authorization_props", "user_claim", assignment.Id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("approver has no value limit")
		}
		return 0, err
	}
	maxValue := props.GetFloat("max_value")
	if maxValue <= 0 || maxValue > MaxProjectAuthorizationValue || math.IsNaN(maxValue) || math.IsInf(maxValue, 0) || math.Trunc(maxValue) != maxValue || maxValue < value {
		return 0, fmt.Errorf("approver limit does not cover the full project value")
	}
	return int(maxValue), nil
}

// ValidateRequestContactEmails requires a request's project contact and billing
// contact to each have an email address or be marked as having none. Contact
// email is otherwise optional, because many clients have no email address.
func ValidateRequestContactEmails(app core.App, request *core.Record) error {
	type requestContact struct{ label, id string }
	contacts := []requestContact{{"project contact", request.GetString("contact")}}
	if profile, err := app.FindRecordById("client_invoicing_information", request.GetString("invoicing_information")); err == nil {
		contacts = append(contacts, requestContact{"billing contact", profile.GetString("contact")})
	}
	for _, contact := range contacts {
		c, err := app.FindRecordById("client_contacts", contact.id)
		if err != nil {
			return &errs.HookError{Status: http.StatusBadRequest, Message: fmt.Sprintf("The %s was not found.", contact.label)}
		}
		if strings.TrimSpace(c.GetString("email")) == "" && !c.GetBool("no_email") {
			return &errs.HookError{
				Status:  http.StatusBadRequest,
				Message: fmt.Sprintf("Add an email address for the %s, or mark the contact as having no email address.", contact.label),
			}
		}
	}
	return nil
}
