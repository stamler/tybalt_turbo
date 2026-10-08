package main

import (
	"net/http"
	"testing"
)

func TestProjectWorkflowInternalFieldsCannotBeInjected(t *testing.T) {
	for _, generic := range []bool{false, true} {
		t.Run(map[bool]string{false: "custom upsert", true: "generic collection"}[generic], func(t *testing.T) {
			app := newWorkflowTestApp(t, false)
			job := workflowRecord(t, app, "jobs", legacyWorkflowJob)
			description := job.GetString("description")
			updates := map[string]any{"description": "Injected policy attempt", "@projectWorkflowOperation": "restore_value", "@projectWorkflowValidated": true}
			method, url, body := http.MethodPut, "/api/jobs/"+legacyWorkflowJob, map[string]any{"job": updates, "allocations": []map[string]any{{"division": "fy4i9poneukvq9u", "hours": 10}}}
			if generic {
				method, url, body = http.MethodPatch, "/api/collections/jobs/records/"+legacyWorkflowJob, updates
			}
			clientSetupJSON(t, app, method, url, workflowAuthor, body, http.StatusBadRequest)
			job = workflowRecord(t, app, "jobs", legacyWorkflowJob)
			if job.GetString("description") != description || job.GetFloat("project_value") != 0 {
				t.Fatal("injected policy changed the job")
			}
		})
	}
}
