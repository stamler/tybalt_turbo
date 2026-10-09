package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tybalt/utilities"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func recordPublic(r *core.Record) map[string]any {
	if r == nil {
		return nil
	}
	return r.PublicExport()
}
func requestRelated(app core.App, r *core.Record) (map[string]any, error) {
	result := map[string]any{}
	relations := map[string]string{"client": "clients", "job_owner": "clients", "contact": "client_contacts", "manager": "users", "alternate_manager": "users", "creator": "users", "approver": "users", "branch": "branches", "rate_sheet": "rate_sheets", "parent": "jobs", "proposal": "jobs", "invoicing_information": "client_invoicing_information"}
	for key, collection := range relations {
		if id := r.GetString(key); id != "" {
			row, err := app.FindRecordById(collection, id)
			if err != nil {
				return nil, err
			}
			data := recordPublic(row)
			if collection == "users" {
				data = workflowPerson(app, row)
			}
			result[key] = data
		}
	}
	if profile, ok := result["invoicing_information"].(map[string]any); ok {
		if id, ok := profile["contact"].(string); ok && id != "" {
			row, err := app.FindRecordById("client_contacts", id)
			if err != nil {
				return nil, err
			}
			result["billing_contact"] = recordPublic(row)
		}
	}
	// Bill the client name that the invoicing profile chooses: official or alias.
	if id := r.GetString("client"); id != "" {
		client, err := app.FindRecordById("clients", id)
		if err != nil {
			return nil, err
		}
		var profile *core.Record
		if id := r.GetString("invoicing_information"); id != "" {
			if profile, err = app.FindRecordById("client_invoicing_information", id); err != nil {
				return nil, err
			}
		}
		result["bill_to_name"] = utilities.BillingName(client, profile)
	}
	rates := []map[string]any{}
	if id := r.GetString("rate_sheet"); id != "" {
		rows, err := app.FindRecordsByFilter("rate_sheet_entries", "rate_sheet={:id}", "role", 0, 0, dbx.Params{"id": id})
		if err != nil {
			return nil, err
		}
		for _, entry := range rows {
			role, err := app.FindRecordById("rate_roles", entry.GetString("role"))
			if err != nil {
				return nil, err
			}
			rates = append(rates, map[string]any{"id": entry.Id, "role": role.GetString("name"), "role_name": role.GetString("name"), "rate": entry.GetFloat("rate"), "overtime_rate": entry.GetFloat("overtime_rate"), "updated": entry.Get("updated")})
		}
	}
	result["rate_sheet_entries"] = rates
	return result, nil
}
func workflowPerson(app core.App, user *core.Record) map[string]any {
	data := map[string]any{"id": user.Id, "name": user.GetString("name"), "email": user.GetString("email"), "phone": ""}
	if p, err := app.FindFirstRecordByData("profiles", "uid", user.Id); err == nil {
		data["given_name"] = p.GetString("given_name")
		data["surname"] = p.GetString("surname")
		data["name"] = strings.TrimSpace(p.GetString("given_name") + " " + p.GetString("surname"))
		data["phone"] = p.GetString("phone")
	}
	return data
}

// requestReviewToken binds a review page to both saved data and one submission.
// submission_token is a random identity issued only by submit, not a password
// or a sequence number. Without it, recall/resubmit of unchanged data and PDF
// would recreate the same digest and accept the old page's checked checklist.
// Approval stores this digest on the attachment; creation checks that stored
// digest and the caller's digest against the current submission. Permission and
// workflow-state checks remain separate. A draft can have an empty submission
// token for viewing/printing, but approve/create must reject an empty token.
func requestReviewToken(app core.App, r *core.Record) (string, error) {
	related, err := requestRelated(app, r)
	if err != nil {
		return "", err
	}
	delete(related, "approver")
	delete(related, "creator")
	data := map[string]any{"version": r.GetInt("version"), "submission_token": r.GetString("submission_token"), "related": related}
	for _, key := range requestBusinessFields {
		data[key] = r.Get(key)
	}
	allocs, err := requestAllocations(app, r.Id)
	if err != nil {
		return "", err
	}
	values := []map[string]any{}
	for _, a := range allocs {
		d, err := app.FindRecordById("divisions", a.GetString("division"))
		if err != nil {
			return "", err
		}
		values = append(values, map[string]any{"division": d.Id, "hours": a.GetFloat("hours"), "active": d.GetBool("active")})
	}
	data["allocations"] = values
	a, err := requestAttachment(app, r.Id)
	if err != nil {
		return "", err
	}
	if a != nil {
		data["document_hash"] = a.GetString("document_hash")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
func requestDetailHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error { return respondRequestDetail(app, e, e.Request.PathValue("id")) }
}
func respondRequestDetail(app core.App, e *core.RequestEvent, id string) error {
	var result map[string]any
	err := app.RunInTransaction(func(tx core.App) error {
		r, err := loadVisibleRequest(tx, id, e.Auth)
		if err != nil {
			return err
		}
		a, err := requestAttachment(tx, id)
		if err != nil {
			return err
		}
		allocs, err := requestAllocations(tx, id)
		if err != nil {
			return err
		}
		related, err := requestRelated(tx, r)
		if err != nil {
			return err
		}
		token, err := requestReviewToken(tx, r)
		if err != nil {
			return err
		}
		enabled, err := utilities.IsJobRequestApprovalRequired(tx)
		if err != nil {
			return err
		}
		hasJob, err := utilities.HasClaim(tx, e.Auth, "job")
		if err != nil {
			return err
		}
		hasAdmin, err := utilities.HasClaim(tx, e.Auth, "admin")
		if err != nil {
			return err
		}
		hasPA, err := utilities.HasClaim(tx, e.Auth, "project_authorization")
		if err != nil {
			return err
		}
		editable := r.GetString("creator") == e.Auth.Id && (r.GetString("status") == "Draft" || r.GetString("status") == "Recalled" || r.GetString("status") == "Rejected")
		submitted := r.GetString("status") == "Awaiting approval" || r.GetString("status") == "Awaiting creation"
		_, authorityErr := utilities.ProjectAuthorizationLimit(tx, e.Auth.Id, r.GetFloat("project_value"))
		attachment := recordPublic(a)
		if a != nil {
			attachment["document_url"] = "/api/job_attachments/" + a.Id + "/document"
		}
		result = map[string]any{"request": r, "allocations": allocs, "attachment": attachment, "related": related, "review_token": token, "enabled": enabled, "permissions": map[string]bool{"can_edit": enabled && editable, "can_submit": enabled && editable && r.GetString("status") != "Rejected", "can_recall": enabled && submitted && r.GetString("creator") == e.Auth.Id, "can_reject": enabled && submitted && hasPA && r.GetString("approver") == e.Auth.Id, "can_approve": enabled && r.GetString("status") == "Awaiting approval" && r.GetString("approver") == e.Auth.Id && authorityErr == nil, "can_create": enabled && hasJob && r.GetString("status") == "Awaiting creation", "can_delete": hasAdmin && abandonedRequest(r)}}
		return nil
	})
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, result)
}
func requestQueueSelection(app core.App, auth *core.Record, queue string) (string, dbx.Params, error) {
	if err := workflowActive(app, auth); err != nil {
		return "", nil, err
	}
	params := dbx.Params{"uid": auth.Id}
	switch queue {
	case "creation":
		if err := requestEnabled(app); err != nil {
			return "", nil, err
		}
		if err := workflowClaim(app, auth, "job"); err != nil {
			return "", nil, err
		}
		return creationQueueFilter, params, nil
	case "approval":
		if err := workflowClaim(app, auth, "project_authorization"); err != nil {
			return "", nil, err
		}
		return "approver={:uid} && status='Awaiting approval'", params, nil
	case "abandoned":
		if err := workflowClaim(app, auth, "admin"); err != nil {
			return "", nil, err
		}
		params["cutoff"] = time.Now().UTC().AddDate(0, 0, -JOB_REQUEST_ABANDONMENT_DAYS).Format("2006-01-02 15:04:05.000Z")
		return "last_edited <= {:cutoff}", params, nil
	case "all":
		if err := workflowClaim(app, auth, "job"); err != nil {
			return "", nil, err
		}
		return "id != ''", params, nil
	case "", "mine":
		return "creator={:uid}", params, nil
	default:
		return "", nil, workflowError("Unknown request queue.")
	}
}
func requestListHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		filter, params, err := requestQueueSelection(app, e.Auth, e.Request.URL.Query().Get("queue"))
		if err != nil {
			return err
		}
		page, _ := strconv.Atoi(e.Request.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(e.Request.URL.Query().Get("perPage"))
		if limit < 1 || limit > 100 {
			limit = 50
		}
		var response map[string]any
		err = app.RunInTransaction(func(tx core.App) error {
			all, err := tx.FindRecordsByFilter("job_requests", filter, "-submitted,-created,id", 0, 0, params)
			if err != nil {
				return err
			}
			total := len(all)
			start := min((page-1)*limit, total)
			end := min(start+limit, total)
			items := []map[string]any{}
			for _, r := range all[start:end] {
				row := recordPublic(r)
				if c, err := tx.FindRecordById("clients", r.GetString("client")); err == nil {
					row["client_name"] = c.GetString("name")
				}
				items = append(items, row)
			}
			enabled, err := utilities.IsJobRequestApprovalRequired(tx)
			if err != nil {
				return err
			}
			response = map[string]any{"items": items, "totalItems": total, "page": page, "perPage": limit, "totalPages": (total + limit - 1) / limit, "enabled": enabled}
			return nil
		})
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, response)
	}
}
func requestOptionsHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := workflowActive(app, e.Auth); err != nil {
			return err
		}
		allowed := false
		for _, claim := range []string{"time", "job", "project_authorization", "admin"} {
			ok, err := utilities.HasClaim(app, e.Auth, claim)
			if err != nil {
				return err
			}
			allowed = allowed || ok
		}
		if !allowed {
			return apis.NewForbiddenError("You cannot use project requests.", nil)
		}
		result := map[string]any{}
		for key, col := range map[string]string{"clients": "clients", "contacts": "client_contacts", "invoicing_profiles": "client_invoicing_information", "branches": "branches", "rate_sheets": "rate_sheets", "divisions": "divisions"} {
			rows, err := app.FindAllRecords(col)
			if err != nil {
				return err
			}
			result[key] = rows
		}
		users, err := app.FindAllRecords("users")
		if err != nil {
			return err
		}
		people := []map[string]any{}
		for _, u := range users {
			active, err := utilities.IsUserActive(app, u.Id)
			if err != nil {
				return err
			}
			if active {
				people = append(people, workflowPerson(app, u))
			}
		}
		result["users"] = people
		jobs, err := app.FindAllRecords("jobs")
		if err != nil {
			return err
		}
		proposals := []map[string]any{}
		parents := []map[string]any{}
		for _, j := range jobs {
			row := map[string]any{}
			for _, key := range []string{"id", "number", "description", "client", "proposal", "parent", "status", "contact", "manager", "branch", "rate_sheet", "project_value", "time_and_materials", "invoicing_information", "location"} {
				row[key] = j.Get(key)
			}
			if strings.HasPrefix(j.GetString("number"), "P") {
				switch j.GetString("status") {
				case "In Progress", "Submitted", "Awarded":
					proposals = append(proposals, row)
				}
			} else if regexp.MustCompile(`^\d{2}-\d{3,4}$`).MatchString(j.GetString("number")) {
				parents = append(parents, row)
			}
		}
		result["proposals"] = proposals
		result["parents"] = parents
		enabled, err := utilities.IsJobRequestApprovalRequired(app)
		if err != nil {
			return err
		}
		result["enabled"] = enabled
		return e.JSON(http.StatusOK, result)
	}
}
func authorizationApproversHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := workflowActive(app, e.Auth); err != nil {
			return err
		}
		value, err := strconv.ParseFloat(e.Request.URL.Query().Get("value"), 64)
		if err != nil || !validWholeAmount(value, true) {
			return workflowError("Enter a positive whole-dollar project value.")
		}
		rows, err := app.FindRecordsByFilter("user_claims", "cid.name='project_authorization'", "uid", 0, 0)
		if err != nil {
			return err
		}
		items := []map[string]any{}
		for _, row := range rows {
			limit, err := utilities.ProjectAuthorizationLimit(app, row.GetString("uid"), value)
			if err != nil {
				continue
			}
			u, err := app.FindRecordById("users", row.GetString("uid"))
			if err != nil {
				return err
			}
			person := workflowPerson(app, u)
			person["max_value"] = limit
			items = append(items, person)
		}
		return e.JSON(http.StatusOK, map[string]any{"items": items})
	}
}
func requestDuplicatesHandler(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		r, err := loadVisibleRequest(app, e.Request.PathValue("id"), e.Auth)
		if err != nil {
			return err
		}
		rows, err := app.FindAllRecords("jobs")
		if err != nil {
			return err
		}
		items := []map[string]any{}
		for _, j := range rows {
			if strings.HasPrefix(j.GetString("number"), "P") {
				continue
			}
			reasons := duplicateReasons(r, j)
			if len(reasons) == 0 {
				continue
			}
			row := map[string]any{}
			for _, key := range []string{"id", "number", "description", "client", "status", "manager", "parent"} {
				row[key] = j.Get(key)
			}
			if parent, err := app.FindRecordById("jobs", j.GetString("parent")); err == nil {
				row["parent_number"] = parent.GetString("number")
			}
			row["reasons"] = reasons
			if c, err := app.FindRecordById("clients", j.GetString("client")); err == nil {
				row["client_name"] = c.GetString("name")
			}
			if u, err := app.FindRecordById("users", j.GetString("manager")); err == nil {
				row["manager_name"] = workflowPerson(app, u)["name"]
			}
			items = append(items, row)
		}
		pending := []map[string]any{}
		hasJob, err := utilities.HasClaim(app, e.Auth, "job")
		if err != nil {
			return err
		}
		if hasJob {
			requests, err := app.FindAllRecords("job_requests")
			if err != nil {
				return err
			}
			for _, other := range requests {
				if other.Id == r.Id {
					continue
				}
				reasons := duplicateReasons(r, other)
				if len(reasons) > 0 {
					row := recordPublic(other)
					row["reasons"] = reasons
					pending = append(pending, row)
				}
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"jobs": items, "requests": pending})
	}
}
func duplicateReasons(r, other *core.Record) []string {
	reasons := []string{}
	for _, pair := range [][2]string{{"client", "Same client"}, {"proposal", "Same proposal"}, {"parent", "Same parent"}, {"location", "Same location"}, {"client_po", "Same client PO"}, {"client_reference_number", "Same client reference"}} {
		if value := r.GetString(pair[0]); value != "" && value == other.GetString(pair[0]) {
			reasons = append(reasons, pair[1])
		}
	}
	a, b := strings.ToLower(strings.TrimSpace(r.GetString("description"))), strings.ToLower(strings.TrimSpace(other.GetString("description")))
	if a != "" && b != "" && (strings.Contains(a, b) || strings.Contains(b, a)) {
		reasons = append(reasons, "Similar description")
	}
	if r.GetString("parent") == other.Id {
		reasons = append(reasons, "Parent project")
	}
	return reasons
}

func addProjectWorkflowBadges(app core.App, auth *core.Record, counts map[string]int) error {
	if err := workflowActive(app, auth); err != nil {
		return nil
	}
	enabled, err := utilities.IsJobRequestApprovalRequired(app)
	if err != nil {
		return err
	}
	if enabled {
		hasJob, err := utilities.HasClaim(app, auth, "job")
		if err != nil {
			return err
		}
		if hasJob {
			filter, params, err := requestQueueSelection(app, auth, "creation")
			if err != nil {
				return err
			}
			rows, err := app.FindRecordsByFilter("job_requests", filter, "", 0, 0, params)
			if err != nil {
				return err
			}
			counts["/job-requests?queue=creation"] = len(rows)
		}
	}
	hasPA, err := utilities.HasClaim(app, auth, "project_authorization")
	if err != nil {
		return err
	}
	if hasPA {
		if enabled {
			rows, err := app.FindRecordsByFilter("job_requests", "approver={:uid} && status='Awaiting approval'", "", 0, 0, dbx.Params{"uid": auth.Id})
			if err != nil {
				return err
			}
			counts["/job-requests?queue=approval"] = len(rows)
		}
		rows, err := app.FindRecordsByFilter("job_attachments", "assigned_approver={:uid} && job!='' && status='Pending'", "", 0, 0, dbx.Params{"uid": auth.Id})
		if err != nil {
			return err
		}
		counts["/jobs/value-approvals"] = len(rows)
	}
	return nil
}
