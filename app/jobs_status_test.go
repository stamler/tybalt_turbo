// Package main contains API scenario tests for job status validation.
//
// These tests are in the main package (not hooks) to avoid an import cycle:
//   - testutils imports hooks (to call hooks.AddHooks)
//   - if hooks_test imported testutils, we'd have: hooks_test -> testutils -> hooks
//
// The tests in hooks/jobs_test.go can exist because they don't use testutils -
// they test internal functions directly without the full hook setup.
//
// This pattern matches client_notes_test.go which also uses API scenarios.
package main

import (
	"net/http"
	"strings"
	"testing"

	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/tests"
)

// =============================================================================
// Proposal Status Validation Tests
// =============================================================================

// TestProposalStatus_CancelledIsTerminal verifies that cancelled proposals cannot be modified.
//
// Test data: test_prop_cancelled (P24-0802) is a proposal with status "Cancelled"
func TestProposalStatus_CancelledIsTerminal(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "updating cancelled proposal fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_cancelled",
			Body: strings.NewReader(`{
				"description": "Trying to change description"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"cancelled_proposal_immutable"`,
				`"cancelled proposals cannot be modified"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestProposalStatus_DisallowedStatuses verifies that proposals cannot have Active or Closed status.
//
// Test data: test_prop_inprog (P24-0801) is a proposal with status "In Progress"
func TestProposalStatus_DisallowedStatuses(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "proposal cannot have Active status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Active"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
				`"proposals may be In Progress, Submitted, Awarded, Not Awarded, Cancelled or No Bid"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal cannot have Closed status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Closed"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestProjectStatus_DisallowedStatuses verifies that projects cannot have proposal-only statuses.
//
// Test data: cjf0kt0defhq480 (24-321) is a project with status "Active"
func TestProjectStatus_DisallowedStatuses(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "project cannot have Submitted status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"status": "Submitted"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
				`"projects may be Active, Closed or Cancelled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "project cannot have In Progress status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"status": "In Progress"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "project cannot have No Bid status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"status": "No Bid"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "project cannot have Awarded status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"status": "Awarded"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "project cannot have Not Awarded status",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/cjf0kt0defhq480",
			Body: strings.NewReader(`{
				"status": "Not Awarded"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_type"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestProposalStatus_ValueRequirement verifies that proposals with Submitted/Awarded/Not Awarded
// status must have proposal_value > 0 or time_and_materials = true.
//
// Test data: test_prop_inprog (P24-0801) is a proposal with proposal_value=0 and time_and_materials=false
func TestProposalStatus_ValueRequirement(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "proposal Submitted without value or T&M fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Submitted"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"value_required_for_status"`,
				`"proposals with status Submitted, Awarded, or Not Awarded must have a proposal value or be marked as time and materials"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal Awarded without value or T&M fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Awarded"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"value_required_for_status"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal Not Awarded without value or T&M fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Not Awarded"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"value_required_for_status"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal Submitted with proposal_value succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Submitted",
				"proposal_value": 50000
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Submitted"`,
				`"proposal_value":50000`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal Submitted with time_and_materials succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Submitted",
				"time_and_materials": true
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Submitted"`,
				`"time_and_materials":true`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestProposalStatus_CommentRequirement verifies that transitioning to No Bid or Cancelled
// requires a client_note with matching job_status_changed_to.
//
// Test data:
//   - test_prop_inprog (P24-0801) has no matching client_notes for No Bid
//   - test_prop_with_nobid_note (P24-0804) has a client_note with job_status_changed_to="No Bid"
//   - test_prop_with_cancel_note (P24-0805) has a client_note with job_status_changed_to="Cancelled"
func TestProposalStatus_CommentRequirement(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "proposal No Bid without comment fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "No Bid"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"comment_required_for_status"`,
				`"a comment must be added before setting status to No Bid"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal Cancelled without comment fails",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_inprog",
			Body: strings.NewReader(`{
				"status": "Cancelled"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"comment_required_for_status"`,
				`"a comment must be added before setting status to Cancelled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal No Bid with matching comment succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_with_nobid_note",
			Body: strings.NewReader(`{
				"status": "No Bid"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"No Bid"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "proposal Cancelled with matching comment succeeds",
			Method: http.MethodPatch,
			URL:    "/api/collections/jobs/records/test_prop_with_cancel_note",
			Body: strings.NewReader(`{
				"status": "Cancelled"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Cancelled"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

// TestNewProposal_StatusRestrictions verifies that new proposals can only have
// "In Progress" or "Submitted" status. Other statuses like "Awarded" are not allowed
// for new proposals because they require the job ID to exist (for comment requirements)
// or represent later workflow states.
//
// Test data: Uses existing client (ee3xvodl583b61o), contact (235g6k01xx3sdjk),
// manager (f2j5a8vk006baub), and branch (1r7r6hyp681vi15) from test_pb_data/data.db
func TestNewProposal_StatusRestrictions(t *testing.T) {
	recordToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "new proposal with Submitted status and proposal_value succeeds",
			Method: http.MethodPost,
			URL:    "/api/collections/jobs/records",
			Body: strings.NewReader(`{
				"description": "Test New Proposal Submitted",
				"client": "ee3xvodl583b61o",
				"contact": "235g6k01xx3sdjk",
				"manager": "f2j5a8vk006baub",
				"branch": "1r7r6hyp681vi15",
				"proposal_opening_date": "2024-12-01",
				"proposal_submission_due_date": "2024-12-15",
				"status": "Submitted",
				"proposal_value": 50000,
				"location": "87G8Q2GX+HV"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Submitted"`,
				`"proposal_value":50000`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "new proposal with Awarded status fails",
			Method: http.MethodPost,
			URL:    "/api/collections/jobs/records",
			Body: strings.NewReader(`{
				"description": "Test New Proposal Awarded",
				"client": "ee3xvodl583b61o",
				"contact": "235g6k01xx3sdjk",
				"manager": "f2j5a8vk006baub",
				"branch": "1r7r6hyp681vi15",
				"proposal_opening_date": "2024-12-01",
				"proposal_submission_due_date": "2024-12-15",
				"status": "Awarded",
				"proposal_value": 50000,
				"location": "87G8Q2GX+HV"
			}`),
			Headers: map[string]string{
				"Authorization": recordToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"code":"invalid_status_for_new_proposal"`,
				`"new proposals can only have status In Progress or Submitted"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
