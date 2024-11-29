package main

import (
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/routes"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestAbsorbRoutes(t *testing.T) {
	userToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	bookKeeperToken, err := testutils.GenerateRecordToken("users", "book@keeper.com")
	if err != nil {
		t.Fatal(err)
	}

	// Generate an invalid token to test auth record retrieval failure
	invalidToken := "invalid_token_format"

	// Create a custom test app factory for testing unsupported collection
	unsupportedCollectionTestApp := func(t *testing.T) *tests.TestApp {
		app, err := tests.NewTestApp("./test_pb_data")
		if err != nil {
			t.Fatal(err)
		}

		// Add a route with an unsupported collection
		app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
			e.Router.AddRoute(echo.Route{
				Method:  http.MethodPost,
				Path:    "/api/test_unsupported/:id/absorb",
				Handler: routes.CreateAbsorbRecordsHandler(app, "unsupported_collection"),
				Middlewares: []echo.MiddlewareFunc{
					apis.RequireRecordAuth("users"),
				},
			})
			return nil
		})

		return app
	}

	// Create a custom test app factory for testing claim check failure
	claimCheckFailureTestApp := func(t *testing.T) *tests.TestApp {
		app, err := tests.NewTestApp("./test_pb_data")
		if err != nil {
			t.Fatal(err)
		}

		// Add routes with the broken claims table
		app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
			// Break the claims table
			_, err := app.DB().NewQuery("ALTER TABLE claims RENAME TO claims_broken").Execute()
			if err != nil {
				t.Fatal(err)
			}

			e.Router.AddRoute(echo.Route{
				Method:  http.MethodPost,
				Path:    "/api/clients/:id/absorb",
				Handler: routes.CreateAbsorbRecordsHandler(app, "clients"),
				Middlewares: []echo.MiddlewareFunc{
					apis.RequireRecordAuth("users"),
				},
			})
			return nil
		})

		return app
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "unauthorized request",
			Method:         http.MethodPost,
			Url:            "/api/clients/lb0fnenkeyitsny/absorb",
			Body:           strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			ExpectedStatus: 401,
			ExpectedContent: []string{
				`"message":"The request requires valid record authorization token to be set."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid request body",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`invalid json`),
			RequestHeaders: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Invalid request body."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "empty ids list",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": []}`),
			RequestHeaders: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"No IDs provided to absorb."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unauthorized user (no absorb claim)",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			RequestHeaders: map[string]string{
				"Authorization": userToken,
			},
			ExpectedStatus: 403,
			ExpectedContent: []string{
				`"message":"User does not have permission to absorb records."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "authorized user (has absorb claim)",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			RequestHeaders: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"message":"Successfully absorbed 2 records into lb0fnenkeyitsny"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid auth token format",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			RequestHeaders: map[string]string{
				"Authorization": invalidToken,
			},
			ExpectedStatus: 401,
			ExpectedContent: []string{
				`"message":"The request requires valid record authorization token to be set."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "absorb non-existent records",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["definitely_nonexistent_1", "definitely_nonexistent_2"]}`),
			RequestHeaders: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"code":404,"message":"Failed to find record to absorb."`,
			},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 1,
				"OnAfterApiError":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "absorb into non-existent target",
			Method: http.MethodPost,
			Url:    "/api/clients/definitely_nonexistent_target/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			RequestHeaders: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 404,
			ExpectedContent: []string{
				`"code":404,"message":"Failed to find target record."`,
			},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 1,
				"OnAfterApiError":  1,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "unsupported collection",
			Method: http.MethodPost,
			Url:    "/api/test_unsupported/test_id/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["test1", "test2"]}`),
			RequestHeaders: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"code":500,"message":"Failed to absorb records."`,
			},
			ExpectedEvents: map[string]int{
				"OnBeforeApiError": 1,
				"OnAfterApiError":  1,
			},
			TestAppFactory: unsupportedCollectionTestApp,
		},
		{
			Name:   "absorb record into itself",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["lb0fnenkeyitsny"]}`),
			RequestHeaders: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Cannot absorb a record into itself."`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "claim check failure",
			Method: http.MethodPost,
			Url:    "/api/clients/lb0fnenkeyitsny/absorb",
			Body:   strings.NewReader(`{"ids_to_absorb": ["eldtxi3i4h00k8r", "pqpd90fqd5ohjcs"]}`),
			RequestHeaders: map[string]string{
				"Authorization": bookKeeperToken,
			},
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"message":"Failed to check user claims."`,
			},
			TestAppFactory: claimCheckFailureTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
