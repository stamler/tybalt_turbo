package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"tybalt/internal/testutils"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPurchaseOrdersRoutes(t *testing.T) {
	unauthorizedToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	nonCloseToken, err := testutils.GenerateRecordToken("users", "fakemanager@fakesite.xyz")
	if err != nil {
		t.Fatal(err)
	}

	closeToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	poApproverToken, err := testutils.GenerateRecordToken("users", "fatt@mac.com")
	if err != nil {
		t.Fatal(err)
	}

	// Token for po_approver_tier2 user
	tier2Token, err := testutils.GenerateRecordToken("users", "tier2@poapprover.com")
	if err != nil {
		t.Fatal(err)
	}

	// Token for user with po_approver_tier3 claim
	po_approver_tier3Token, err := testutils.GenerateRecordToken("users", "hal@2005.com")
	if err != nil {
		t.Fatal(err)
	}

	// Get current date in UTC for approval timestamp validation
	currentDate := time.Now().UTC().Format("2006-01-02")
	currentPoPrefix := fmt.Sprintf("%d%02d-", time.Now().Year()%100, time.Now().Month())

	// Get approval tier values from the database
	app := testutils.SetupTestApp(t)
	tier1, tier2 := testutils.GetApprovalTiers(app)

	scenarios := []tests.ApiScenario{
		{
			Name:   fmt.Sprintf("authorized approver successfully approves PO below lowest threshold (%.0f)", tier1),
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/approve", // Using existing Unapproved PO with total 329.01
			Body:   strings.NewReader(`{}`),                        // No body needed for approval
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate),      // Should have today's date
				fmt.Sprintf(`"po_number":"%s`, currentPoPrefix), // Should start with current year
				`"status":"Active"`,                             // Status should be Active
				`"approver":"etysnrlup2f6bak"`,                  // Should be set to the approver's ID
			},
			ExpectedEvents: map[string]int{
				"OnRecordCreate":   1, // notification record is created
				"OnRecordUpdate":   1, // purchase order record is updated
				"OnRecordValidate": 2, // both records are validated
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "first approval of high-value PO leaves status as Unapproved",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/46efdq319b22480/approve", // Using existing Unapproved PO with total 862.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate), // Should have today's date
				`"status":"Unapproved"`,                    // Status should remain Unapproved
				`"po_number":""`,                           // No PO number yet
				`"approver":"etysnrlup2f6bak"`,             // Approver changes to match caller (fatt@mac.com)
				`"second_approver":""`,                     // No second approver yet
				`"second_approval":""`,                     // No second approval timestamp yet
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "first approval is blocked when second approval is required but no second approver can be assigned",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"second_approval_unassignable"`,
				`"message":"second approval is required, but no eligible second approver is available. Set a priority second approver before first approval."`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(t)
				_, err := app.NonconcurrentDB().NewQuery(`
					UPDATE purchase_orders
					SET
						approval_total = 2000000,
						total = 2000000,
						status = 'Unapproved',
						approved = '',
						second_approval = '',
						approver = '',
						second_approver = '',
						priority_second_approver = ''
					WHERE id = 'gal6e5la2fa4rpn'
				`).Execute()
				if err != nil {
					t.Fatal(err)
				}
				return app
			},
		},
		{
			Name:   "first approval succeeds when priority second approver is set even with no general second approver candidates",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate),
				`"status":"Unapproved"`,
				`"po_number":""`,
				`"approver":"etysnrlup2f6bak"`,
				`"priority_second_approver":"6bq4j0eb26631dy"`,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(t)
				_, err := app.NonconcurrentDB().NewQuery(`
					UPDATE purchase_orders
					SET
						approval_total = 2000000,
						total = 2000000,
						status = 'Unapproved',
						approved = '',
						second_approval = '',
						approver = '',
						second_approver = '',
						priority_second_approver = '6bq4j0eb26631dy'
					WHERE id = 'gal6e5la2fa4rpn'
				`).Execute()
				if err != nil {
					t.Fatal(err)
				}
				return app
			},
		},
		{
			Name:   "po_approver_tier2 claim holder completes both approvals of high-value PO in single call",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/46efdq319b22480/approve", // Using existing Unapproved PO with total 862.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": tier2Token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				fmt.Sprintf(`"approved":"%s`, currentDate),        // Should have today's date
				fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have same timestamp
				`"status":"Active"`,                               // Status should become Active
				fmt.Sprintf(`"po_number":"%s`, currentPoPrefix),   // Should get PO number
				`"approver":"6bq4j0eb26631dy"`,                    // caller becomes first approver
				`"second_approver":"6bq4j0eb26631dy"`,             // caller also becomes second approver
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate":   1, // purchase order record is updated
				"OnRecordCreate":   1, // notification record is created
				"OnRecordValidate": 2, // both records are validated
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "second approval of high-value PO completes approval process",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2blv18f40i2q373/approve", // Using PO with first approval and total 1022.69
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": tier2Token,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"approved":"2025-01-29 14:22:29.563Z"`,           // Should keep original first approval
				fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have today's date
				`"status":"Active"`,                               // Status should become Active
				fmt.Sprintf(`"po_number":"%s`, currentPoPrefix),   // Should get PO number
				`"approver":"wegviunlyr2jjjv"`,                    // Should keep original approver
				`"second_approver":"6bq4j0eb26631dy"`,             // Should be set to caller's ID
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1, // purchase order record is updated
				"OnRecordCreate":            1, // notification record is created
				"OnRecordValidate":          2, // both records are validated
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user cannot approve purchase order",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": unauthorizedToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to approve this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "po_approver cannot approve PO in unauthorized division",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/j6nn3v7s2s6d6u8/approve", // PO in division 2rrfy6m2c8hazjy
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // fatt@mac.com who can't approve purchase orders in this division
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to approve this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim can convert Active One-Time purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/make_cumulative",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"type":"Cumulative"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim cannot convert non-Active One-Time purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/gal6e5la2fa4rpn/make_cumulative",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_active"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim cannot convert Active non-One-Time purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/ly8xyzpuj79upq1/make_cumulative",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_one_time"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller without the payables_admin claim cannot convert Active One-Time purchase_orders to Cumulative",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/make_cumulative",
			Headers:        map[string]string{"Authorization": nonCloseToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_conversion"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim can cancel Active purchase_orders records with no expenses against them",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/cancel",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"Cancelled"`,
				fmt.Sprintf(`"cancelled":"%s`, currentDate),
				`"canceller":"tqqf7q0f3378rvp"`,
			},
			ExpectedEvents: map[string]int{
				"OnRecordUpdate": 1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller without the payables_admin claim cannot cancel Active purchase_orders records with no expenses against them",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/cancel",
			Headers:        map[string]string{"Authorization": nonCloseToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_cancellation"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelBeforeUpdate": 0,
				"OnModelAfterUpdate":  0,
				"OnBeforeApiError":    0,
				"OnAfterApiError":     0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "caller without the payables_admin claim cannot close Active Cumulative purchase_orders records",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/ly8xyzpuj79upq1/close",
			Headers:         map[string]string{"Authorization": nonCloseToken},
			ExpectedStatus:  http.StatusForbidden,
			ExpectedContent: []string{`"code":"unauthorized_closure","message":"you are not authorized to close purchase orders"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim can close Active Cumulative purchase_orders records",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/ly8xyzpuj79upq1/close",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"Closed"`,
				fmt.Sprintf(`"closed":"%s`, currentDate),
				`"closer":"tqqf7q0f3378rvp"`,
				`"closed_by_system":false`,
			},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "Active One-Time purchase_orders records cannot be closed",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/2plsetqdxht7esg/close",
			Headers:         map[string]string{"Authorization": closeToken},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"code":"invalid_po_type","message":"One-Time purchase orders may be cancelled but not manually closed"`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// TODO: Non-Active Cumulative purchase_orders records cannot be closed
		{
			Name:           "caller with the payables_admin claim can close Active Recurring purchase_orders records that have committed expenses",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/d8463q483f3da28/close", // PO 2025-0004 (Recurring with committed expense)
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"Closed"`,
				fmt.Sprintf(`"closed":"%s`, currentDate),
				`"closer":"tqqf7q0f3378rvp"`,
				`"closed_by_system":false`,
			},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:            "Active Recurring purchase_orders records without committed expenses cannot be closed manually",
			Method:          http.MethodPost,
			URL:             "/api/purchase_orders/rec5e5la2fa4rpn/close", // PO 2025-0003 (Recurring with no committed expenses)
			Headers:         map[string]string{"Authorization": closeToken},
			ExpectedStatus:  400,
			ExpectedContent: []string{`"code":"no_expenses","message":"only cumulative or recurring purchase orders with at least one associated expense may be closed manually. Cancel the purchase order instead."`},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 0,
				"OnAfterApiError":  0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies that a user cannot perform second approval without the required claims (po_approver_tier2/po_approver_tier3),
		   even if they have other valid permissions. Specifically, it tests that:

		   Test Data:
		   1. Purchase Order (2blv18f40i2q373):
		      - Division: vccd5fo56ctbigh
		      - Total: 1022.69 (above tier 1 limit, requiring second approval)
		      - Current Status: Unapproved
		      - Has first approval: Yes (timestamp: 2025-01-29 14:22:29.563Z)
		      - First approver: wegviunlyr2jjjv

		   2. User Attempting Second Approval (fatt@mac.com using poApproverToken):
		      - Has po_approver claim: Yes
		      - Authorized divisions: ["hcd86z57zjty6jo", "fy4i9poneukvq9u", "vccd5fo56ctbigh"]
		      - Has division permission: Yes (PO's division matches user's authorized divisions)
		      - Has po_approver_tier3 claim: No
		      - Has po_approver_tier2 claim: No

		   Expected Behavior:
		   - Request should fail with 403 Forbidden
		   - Error should indicate lack of required claim (not division permission)
		   - No changes should be made to the PO
		   - No events should be triggered

		   This test isolates the claim requirement by using a user who has all other necessary permissions
		   (division authorization, po_approver claim) but lacks the specific claims required for second approval.
		   This ensures the failure is specifically due to missing po_approver_tier3/po_approver_tier2 claims, not other permission issues.
		*/
		{
			Name:   "user with po_approver claim but without po_approver_tier3/po_approver_tier2 claims cannot perform second approval",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2blv18f40i2q373/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // fatt@mac.com who has division permission but no po_approver_tier2/po_approver_tier3 claims
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to perform second approval on this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user cannot second-approve PO with approval_total above their max_amount",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/q79eyq0uqrk6x2q/approve", // PO with total 3251.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": tier2Token,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_approval"`,
				`"message":"you are not authorized to perform second approval on this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   fmt.Sprintf("po_approver_tier3 claim holder can second-approve PO with value above tier 2 limit (%.0f)", tier2),
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/q79eyq0uqrk6x2q/approve", // PO with total 3251.12
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": po_approver_tier3Token, // hal@2005.com has max_amount of 1000000
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"approved":"2025-01-29 17:00:02.493Z"`,           // already approved, should not change
				fmt.Sprintf(`"second_approval":"%s`, currentDate), // Should have same timestamp
				`"status":"Active"`,                               // Status should become Active
				fmt.Sprintf(`"po_number":"%s`, currentPoPrefix),   // Should get PO number
				`"approver":"f2j5a8vk006baub"`,                    // approver does not change
				`"second_approver":"66ct66w380ob6w8"`,             // po_approver_tier3 holder becomes second approver
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1, // purchase order record is updated
				"OnRecordValidate":          2, // both records are validated
				"OnRecordCreate":            1, // notification record is created
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "approval attempt on already approved PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2plsetqdxht7esg/approve", // Already approved PO (2024-0008)
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // Using a valid approver token to isolate the already-approved check
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_unapproved"`,
				`"message":"only unapproved purchase orders can be approved"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "approval attempt on rejected PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/l9w1z13mm3srtoo/approve", // Rejected PO with rejection reason
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // Using a valid approver token to isolate the rejection check
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_rejected"`,
				`"message":"rejected purchase orders cannot be approved"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "approval attempt on cancelled PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/338568325487lo2/approve", // Cancelled PO (2025-0002)
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // Using a valid approver token to isolate the cancelled status check
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_unapproved"`,
				`"message":"only unapproved purchase orders can be approved"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "approval attempt on non-existent PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/nonexistent123/approve",
			Body:   strings.NewReader(`{}`),
			Headers: map[string]string{
				"Authorization": poApproverToken, // Using a valid approver token to isolate the not-found check
			},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"code":"po_not_found"`,
				`"message":"error fetching purchase order: sql: no rows in result set"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "authorized approver can reject an unapproved purchase order",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "Budget constraints"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Unapproved"`,
				fmt.Sprintf(`"rejected":"%s`, currentDate),
				`"rejection_reason":"Budget constraints"`,
				`"rejector":"etysnrlup2f6bak"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          2, // 1 for validate, 1 for notification
				"OnRecordCreate":            1, // notification created
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user cannot reject purchase order",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "Budget constraints"
			}`),
			Headers:        map[string]string{"Authorization": unauthorizedToken},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_rejection"`,
				`"message":"you are not authorized to reject this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "po_approver cannot reject PO in unauthorized division",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/j6nn3v7s2s6d6u8/reject", // PO in Municipal division (2rrfy6m2c8hazjy)
			Body: strings.NewReader(`{
				"rejection_reason": "Budget constraints"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken}, // fatt@mac.com who can't approve POs in Municipal division
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"code":"unauthorized_rejection"`,
				`"message":"you are not authorized to reject this purchase order"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		// Changes to GetPOApprovers in utilities/po_approvers.go break this test by
		// allowing the qualified second approver to reject an unapproved purchase
		// order even if it doesn't require second approval. This is because the
		// query now doesn't check whether the caller's max_amount <= the lowest
		// threshold. As a result, this test is commented out and replaced with a
		// test immediately below that makes the new behavior explicit. NB: The new
		// test will pass but that doesn't mean that app will necessarily show this
		// PO to the qualified second approver.
		/*
			{
				Name:   "qualified second approver cannot reject an unapproved purchase order if it doesn't require second approval",
				Method: http.MethodPost,
				URL:    "/api/purchase_orders/gal6e5la2fa4rpn/reject", // doesn't require second approval
				Body: strings.NewReader(`{
					"rejection_reason": "Budget constraints"
				}`),
				Headers:        map[string]string{"Authorization": tier2Token},
				ExpectedStatus: http.StatusForbidden,
				ExpectedContent: []string{
					`"code":"unauthorized_rejection"`,
					`"message":"you are not authorized to reject this purchase order"`,
				},
				ExpectedEvents: map[string]int{
					"*": 0,
				},
				TestAppFactory: testutils.SetupTestApp,
			},
		*/
		{
			Name:   "qualified second approver can reject an unapproved purchase order if it doesn't require second approval",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/reject", // doesn't require second approval
			Body: strings.NewReader(`{
				"rejection_reason": "Budget constraints"
			}`),
			Headers:        map[string]string{"Authorization": tier2Token},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"Unapproved"`,
				fmt.Sprintf(`"rejected":"%s`, currentDate),
				`"rejection_reason":"Budget constraints"`,
				`"rejector":"6bq4j0eb26631dy"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          2, // 1 for validate, 1 for notification
				"OnRecordCreate":            1, // notification created
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "qualified second approver can reject an unapproved purchase order if it requires second approval",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/46efdq319b22480/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "Budget constraints"
			}`),
			Headers:        map[string]string{"Authorization": tier2Token},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Unapproved"`,
				fmt.Sprintf(`"rejected":"%s`, currentDate),
				`"rejection_reason":"Budget constraints"`,
				`"rejector":"6bq4j0eb26631dy"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          2, // 1 for validate, 1 for notification
				"OnRecordCreate":            1, // notification created
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		/*
		   This test verifies an important aspect of the rejection authorization model:
		   that rejection permissions are intentionally simpler than approval permissions.

		   Specifically, it verifies that any user with basic approval rights (po_approver)
		   can reject a PO that's awaiting second approval, even if they aren't qualified
		   to perform that second approval themselves.

		   Test setup:
		   - Uses PO 2blv18f40i2q373 which:
		     * Has total of 1022.69 (above tier 1 limit, requiring second approval)
		     * Already has first approval (from wegviunlyr2jjjv)
		     * Is awaiting second approval
		   - Uses poApproverToken (fatt@mac.com) who:
		     * Has po_approver claim but NOT po_approver_tier3/po_approver_tier2 claims
		     * Cannot perform second approvals
		     * Can still reject because PO is in Unapproved state

		   This is by design - the rejection model is simpler because:
		   1. The PO isn't Active yet, so no downstream processes are affected
		   2. Any qualified approver should be able to reject if they spot issues
		   3. The approval state (first approval, pending second) doesn't matter
		      as long as the PO is still in an Unapproved state
		*/
		{
			Name:   "regular approver can reject PO awaiting second approval",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2blv18f40i2q373/reject", // PO with first approval but awaiting second approval
			Body: strings.NewReader(`{
				"rejection_reason": "Budget constraints"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken}, // Using regular approver, not a second approver
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"status":"Unapproved"`,
				fmt.Sprintf(`"rejected":"%s`, currentDate),
				`"rejection_reason":"Budget constraints"`,
				`"rejector":"etysnrlup2f6bak"`,
				`"approved":"2025-01-29 14:22:29.563Z"`, // Should preserve existing first approval
				`"approver":"wegviunlyr2jjjv"`,          // Should preserve existing approver
				`"second_approval":""`,                  // Should still have no second approval
				`"second_approver":""`,                  // Should still have no second approver
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          2, // 1 for validate, 1 for notification
				"OnRecordCreate":            1, // notification created
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "rejection attempt on already rejected PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/l9w1z13mm3srtoo/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "New rejection reason"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_rejected"`,
				`"message":"rejected purchase orders cannot be rejected again"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "rejection attempt on Active PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/2plsetqdxht7esg/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "Found issues"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_unapproved"`,
				`"message":"only unapproved purchase orders can be rejected"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "rejection attempt on non-existent PO fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/nonexistent123/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "Not going to work"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken}, // Using valid approver token to isolate the not-found check
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"code":"po_not_found"`,
				`"message":"error fetching purchase order: sql: no rows in result set"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "rejection with empty reason fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/reject",
			Body: strings.NewReader(`{
				"rejection_reason": ""
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_rejection_reason"`,
				`"message":"rejection reason must be at least 5 characters"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "rejection with too short reason fails",
			Method: http.MethodPost,
			URL:    "/api/purchase_orders/gal6e5la2fa4rpn/reject",
			Body: strings.NewReader(`{
				"rejection_reason": "no"
			}`),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_rejection_reason"`,
				`"message":"rejection reason must be at least 5 characters"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "rejection with blank body fails",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/gal6e5la2fa4rpn/reject",
			Body:           strings.NewReader(`{}`),
			Headers:        map[string]string{"Authorization": poApproverToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"invalid_request_body"`,
				`"message":"you must provide a rejection reason"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim cannot cancel non-Active purchase_orders records",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/338568325487lo2/cancel", // Already Cancelled PO (2025-0002)
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_not_active"`,
				`"message":"only active purchase orders can be cancelled"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "cancellation of non-existent purchase order returns 404",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/nonexistent123/cancel",
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"code":"po_not_found"`,
				`"message":"error fetching purchase order: sql: no rows in result set"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "purchase orders with associated expenses cannot be cancelled",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/ly8xyzpuj79upq1/cancel", // PO 2024-0009 with 4 expenses
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`"code":"po_has_expenses"`,
				`"message":"this purchase order has associated expenses and cannot be cancelled"`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "cancellation fails when expense query fails",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/2plsetqdxht7esg/cancel", // Using a known Active PO
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusInternalServerError,
			ExpectedContent: []string{
				`"code":"error_fetching_expenses"`,
				`"message":"error fetching expenses: SQL logic error: no such table: expenses (1); failed query: SELECT`,
			},
			ExpectedEvents: map[string]int{
				"*": 0,
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := testutils.SetupTestApp(t)

				// Break the expenses table after routes are registered
				app.OnServe().BindFunc(func(e *core.ServeEvent) error {
					_, err := app.NonconcurrentDB().NewQuery("ALTER TABLE expenses RENAME TO expenses_broken").Execute()
					if err != nil {
						t.Fatal(err)
					}
					return e.Next()
				})

				return app
			},
		},
		{
			Name:           "caller with the payables_admin claim can cancel Active Cumulative purchase_orders records with no expenses",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/y660i6a14ql2355/cancel", // Active Cumulative PO (2025-0001)
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"Cancelled"`,
				fmt.Sprintf(`"cancelled":"%s`, currentDate),
				`"canceller":"tqqf7q0f3378rvp"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:           "caller with the payables_admin claim can cancel Active Recurring purchase_orders records with no expenses",
			Method:         http.MethodPost,
			URL:            "/api/purchase_orders/rec5e5la2fa4rpn/cancel", // Active Recurring PO (2025-0003)
			Headers:        map[string]string{"Authorization": closeToken},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				`"status":"Cancelled"`,
				fmt.Sprintf(`"cancelled":"%s`, currentDate),
				`"canceller":"tqqf7q0f3378rvp"`,
			},
			ExpectedEvents: map[string]int{
				"OnModelAfterUpdateSuccess": 1,
				"OnModelUpdate":             1,
				"OnRecordUpdate":            1,
				"OnRecordValidate":          1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
