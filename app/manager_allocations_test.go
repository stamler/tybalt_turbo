package main

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

const allocationManagerEmail = "time@test.com"
const allocationManagerID = "rzr98oadsp9qc11"

func assignAllocationManager(t *testing.T, app *tests.TestApp, role string) {
	t.Helper()
	// Change only the assignment on the CSV fixture to test each supported role.
	if role != "manager" && role != "alternate_manager" {
		t.Fatal("invalid test role")
	}
	if _, err := app.DB().NewQuery("UPDATE jobs SET " + role + "={:uid} WHERE id={:job}").Bind(dbx.Params{"uid": allocationManagerID, "job": legacyWorkflowJob}).Execute(); err != nil {
		t.Fatal(err)
	}
}

func allocationSnapshot(t *testing.T, app *tests.TestApp, jobID string) []map[string]any {
	t.Helper()
	records, err := app.FindRecordsByFilter("job_time_allocations", "job={:job}", "division", 0, 0, dbx.Params{"job": jobID})
	if err != nil {
		t.Fatal(err)
	}
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		result = append(result, record.PublicExport())
	}
	return result
}

func allocationBody(rows ...map[string]any) map[string]any {
	return map[string]any{"allocations": rows}
}

func allocationRow(division string, hours float64) map[string]any {
	return map[string]any{"division": division, "hours": hours}
}

func TestManagerAllocationSave(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, role := range []string{"manager", "alternate_manager", "job_claim"} {
			t.Run(fmt.Sprintf("requests_%t/%s", enabled, role), func(t *testing.T) {
				app := newWorkflowTestApp(t, enabled)
				email := allocationManagerEmail
				if role == "job_claim" {
					// Keep the claim holder unassigned to prove the claim grants access.
					assignAllocationManager(t, app, "manager")
					email = workflowAuthor
				} else {
					assignAllocationManager(t, app, role)
				}
				// Keep the historical zero value and remove billing data. Allocation
				// edits must not require a project correction or a new approval PDF.
				clearLegacyJobBilling(t, app)
				before := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
				path := "/api/jobs/" + legacyWorkflowJob
				clientSetupJSON(t, app, http.MethodPut, path, email, allocationBody(allocationRow("fy4i9poneukvq9u", 12), allocationRow("vccd5fo56ctbigh", 0)), http.StatusOK)
				rows := allocationSnapshot(t, app, legacyWorkflowJob)
				if len(rows) != 2 || rows[0]["hours"] != float64(12) || rows[1]["hours"] != float64(0) {
					t.Fatalf("add/change failed: %#v", rows)
				}
				clientSetupJSON(t, app, http.MethodPut, path, email, allocationBody(allocationRow("vccd5fo56ctbigh", 7)), http.StatusOK)
				rows = allocationSnapshot(t, app, legacyWorkflowJob)
				if len(rows) != 1 || rows[0]["division"] != "vccd5fo56ctbigh" || rows[0]["hours"] != float64(7) {
					t.Fatalf("remove/change failed: %#v", rows)
				}
				after := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
				if after["_imported"] != false || before["updated"] == after["updated"] {
					t.Fatal("allocation change did not mark the job for writeback")
				}
				for _, field := range []string{"updated", "_imported"} {
					delete(before, field)
					delete(after, field)
				}
				if !reflect.DeepEqual(before, after) {
					t.Fatal("allocation save changed project fields")
				}
			})
		}
	}
}

func TestManagerAllocationRejectedSaveIsAtomic(t *testing.T) {
	cases := []struct {
		name   string
		body   map[string]any
		status int
	}{
		{"empty", allocationBody(), http.StatusBadRequest},
		{"empty_job_object", map[string]any{"job": map[string]any{}, "allocations": []map[string]any{allocationRow("fy4i9poneukvq9u", 15)}}, http.StatusForbidden},
		{"fractional", allocationBody(allocationRow("fy4i9poneukvq9u", 1.5)), http.StatusBadRequest},
		{"negative", allocationBody(allocationRow("fy4i9poneukvq9u", -1)), http.StatusBadRequest},
		{"duplicate", allocationBody(allocationRow("fy4i9poneukvq9u", 1), allocationRow("fy4i9poneukvq9u", 2)), http.StatusBadRequest},
		{"inactive_division", allocationBody(allocationRow("2rrfy6m2c8hazjy", 1)), http.StatusBadRequest},
		{"missing_division", allocationBody(allocationRow("missing", 1)), http.StatusBadRequest},
		{"blank_division", allocationBody(allocationRow("", 1)), http.StatusBadRequest},
	}
	for _, field := range []string{"description", "project_value", "manager", "alternate_manager", "invoicing_information", "id", "_workflow_authorized"} {
		body := allocationBody(allocationRow("vccd5fo56ctbigh", 99))
		body["job"] = map[string]any{field: "unauthorized"}
		cases = append(cases, struct {
			name   string
			body   map[string]any
			status int
		}{field, body, http.StatusForbidden})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			assignAllocationManager(t, app, "manager")
			before := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
			allocations := allocationSnapshot(t, app, legacyWorkflowJob)
			clientSetupJSON(t, app, http.MethodPut, "/api/jobs/"+legacyWorkflowJob, allocationManagerEmail, tc.body, tc.status)
			if !reflect.DeepEqual(before, workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()) || !reflect.DeepEqual(allocations, allocationSnapshot(t, app, legacyWorkflowJob)) {
				t.Fatal("rejected save changed stored data")
			}
		})
	}
}

func TestManagerAllocationAccessUsesCurrentState(t *testing.T) {
	for _, state := range []string{"unassigned", "removed_assignment", "inactive", "editing_disabled", "another_job", "cancelled_proposal"} {
		t.Run(state, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			if state != "unassigned" {
				assignAllocationManager(t, app, "alternate_manager")
			}
			// Obtain the token before the state change. Cached access must not grant a save.
			token := workflowToken(t, app, allocationManagerEmail)
			query := ""
			switch state {
			case "removed_assignment":
				query = "UPDATE jobs SET alternate_manager='' WHERE id='pawflegacy00001'"
			case "inactive":
				query = "UPDATE admin_profiles SET active=0 WHERE uid='rzr98oadsp9qc11'"
			case "cancelled_proposal":
				query = "UPDATE jobs SET manager='rzr98oadsp9qc11', status='Cancelled' WHERE id='test_prop_inprog'"
			case "editing_disabled":
				query = `UPDATE app_config SET value=json_set(value, '$.create_edit_absorb', json('false')) WHERE key='jobs'`
			}
			// These targeted fixture changes simulate permissions changing after the editor opens.
			if query != "" {
				if _, err := app.DB().NewQuery(query).Execute(); err != nil {
					t.Fatal(err)
				}
			}
			jobID := legacyWorkflowJob
			if state == "another_job" || state == "cancelled_proposal" {
				jobID = "test_prop_inprog"
			}
			before := workflowRecord(t, app, "jobs", jobID).PublicExport()
			allocations := allocationSnapshot(t, app, jobID)
			response := performJSONRequest(t, app, http.MethodPut, "/api/jobs/"+jobID, token, allocationBody(allocationRow("vccd5fo56ctbigh", 99)))
			status := http.StatusForbidden
			if state == "cancelled_proposal" {
				status = http.StatusBadRequest
			}
			mustStatus(t, response, status)
			if !reflect.DeepEqual(before, workflowRecord(t, app, "jobs", jobID).PublicExport()) || !reflect.DeepEqual(allocations, allocationSnapshot(t, app, jobID)) {
				t.Fatal("denied save changed stored data")
			}
		})
	}
}

func TestManagerAllocationNoOpAndPendingApproval(t *testing.T) {
	app := newWorkflowTestApp(t, true)
	assignAllocationManager(t, app, "alternate_manager")
	id := createValueChange(t, app, legacyWorkflowJob, 10000, "Allocation review")
	before := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
	allocations := allocationSnapshot(t, app, legacyWorkflowJob)
	clientSetupJSON(t, app, http.MethodPut, "/api/jobs/"+legacyWorkflowJob, allocationManagerEmail, allocationBody(allocationRow("fy4i9poneukvq9u", 10)), http.StatusOK)
	if !reflect.DeepEqual(before, workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()) || !reflect.DeepEqual(allocations, allocationSnapshot(t, app, legacyWorkflowJob)) {
		t.Fatal("no-op save changed stored data")
	}
	attachment := workflowRecord(t, app, "job_attachments", id).PublicExport()
	clientSetupJSON(t, app, http.MethodPut, "/api/jobs/"+legacyWorkflowJob, allocationManagerEmail, allocationBody(allocationRow("fy4i9poneukvq9u", 15)), http.StatusOK)
	clientSetupJSON(t, app, http.MethodPost, "/api/job_attachments/"+id+"/approve", workflowApprover, attachmentActionData(t, app, id), http.StatusConflict)
	if !reflect.DeepEqual(attachment, workflowRecord(t, app, "job_attachments", id).PublicExport()) || workflowRecord(t, app, "jobs", legacyWorkflowJob).GetFloat("project_value") != 0 {
		t.Fatal("allocation change or stale approval changed value evidence")
	}
}

func TestGenericAllocationWritesDenied(t *testing.T) {
	for _, role := range []string{"manager", "alternate_manager", "job_claim", "unassigned"} {
		t.Run(role, func(t *testing.T) {
			app := newWorkflowTestApp(t, true)
			email := allocationManagerEmail
			if role == "manager" || role == "alternate_manager" {
				assignAllocationManager(t, app, role)
			} else if role == "job_claim" {
				email = workflowAuthor
			}
			const base = "/api/collections/job_time_allocations/records"
			const record = base + "/pawflegacyalloc"
			before := allocationSnapshot(t, app, legacyWorkflowJob)
			targetBefore := allocationSnapshot(t, app, "test_prop_inprog")
			jobBefore := workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()
			for _, request := range []struct {
				method, path string
				body         map[string]any
			}{
				{http.MethodPost, base, map[string]any{"job": legacyWorkflowJob, "division": "vccd5fo56ctbigh", "hours": 20}},
				{http.MethodPatch, record, map[string]any{"hours": 777}},
				{http.MethodPatch, record, map[string]any{"division": "2rrfy6m2c8hazjy"}},
				{http.MethodPatch, record, map[string]any{"job": "test_prop_inprog", "division": "vccd5fo56ctbigh", "hours": 777}},
				{http.MethodDelete, record, nil},
			} {
				clientSetupJSON(t, app, request.method, request.path, email, request.body, http.StatusForbidden)
			}
			if !reflect.DeepEqual(before, allocationSnapshot(t, app, legacyWorkflowJob)) || !reflect.DeepEqual(targetBefore, allocationSnapshot(t, app, "test_prop_inprog")) || !reflect.DeepEqual(jobBefore, workflowRecord(t, app, "jobs", legacyWorkflowJob).PublicExport()) {
				t.Fatal("denied generic write changed allocations or job data")
			}
			// The editor still reads allocations through the collection API.
			clientSetupJSON(t, app, http.MethodGet, base, email, nil, http.StatusOK)
			clientSetupJSON(t, app, http.MethodGet, record, email, nil, http.StatusOK)
		})
	}
}
