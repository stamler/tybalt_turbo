package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"tybalt/internal/testutils"
	"tybalt/utilities"

	"github.com/pocketbase/pocketbase/tests"
)

func TestCreateMachineSecretAuth(t *testing.T) {
	// User with admin claim (author@soup.com has admin claim in test db)
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// User without admin claim (time@test.com does not have admin claim)
	nonAdminToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing Authorization header returns 401",
			Method:         http.MethodPost,
			URL:            "/api/machine_secrets/create",
			Body:           strings.NewReader(`{"days": 30, "role": "legacy_writeback"}`),
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without admin claim returns 403",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`{"days": 30, "role": "legacy_writeback"}`),
			Headers: map[string]string{
				"Authorization": nonAdminToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"status":403`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with admin claim and valid body returns 201",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`{"days": 30, "role": "legacy_writeback"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusCreated,
			ExpectedContent: []string{
				`"id":`,
				`"secret":`,
				`"expiry":`,
				`"role":"legacy_writeback"`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCreateMachineSecretValidation(t *testing.T) {
	// User with admin claim
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:   "zero days returns 400",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`{"days": 0, "role": "legacy_writeback"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Days must be a positive integer`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "negative days returns 400",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`{"days": -5, "role": "legacy_writeback"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Days must be a positive integer`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid role returns 400",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`{"days": 30, "role": "invalid_role"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Invalid role`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "invalid JSON returns 400",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`not valid json`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusBadRequest,
			ExpectedContent: []string{
				`Invalid JSON body`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestCreateMachineSecretTokenValidation(t *testing.T) {
	// This test verifies that the returned secret can be validated using ValidateMachineToken
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	var capturedSecret string

	scenarios := []tests.ApiScenario{
		{
			Name:   "create machine secret and capture response",
			Method: http.MethodPost,
			URL:    "/api/machine_secrets/create",
			Body:   strings.NewReader(`{"days": 30, "role": "legacy_writeback"}`),
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusCreated,
			ExpectedContent: []string{
				`"id":`,
				`"secret":`,
				`"role":"legacy_writeback"`,
			},
			TestAppFactory: testutils.SetupTestApp,
			AfterTestFunc: func(t testing.TB, app *tests.TestApp, res *http.Response) {
				// Parse response to get the secret
				var response struct {
					ID     string `json:"id"`
					Secret string `json:"secret"`
					Expiry string `json:"expiry"`
					Role   string `json:"role"`
				}
				if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				capturedSecret = response.Secret

				// Verify the secret can be validated
				if !utilities.ValidateMachineToken(app, response.Secret, "legacy_writeback") {
					t.Error("generated secret should validate successfully")
				}

				// Verify wrong secret fails
				if utilities.ValidateMachineToken(app, "wrong-secret", "legacy_writeback") {
					t.Error("wrong secret should not validate")
				}

				// Verify wrong role fails
				if utilities.ValidateMachineToken(app, response.Secret, "wrong_role") {
					t.Error("secret with wrong role should not validate")
				}
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}

	// Verify we captured a secret
	if capturedSecret == "" {
		t.Error("expected to capture a secret from the response")
	}
}

func TestListMachineSecretsAuth(t *testing.T) {
	// User with admin claim (author@soup.com has admin claim in test db)
	adminToken, err := testutils.GenerateRecordToken("users", "author@soup.com")
	if err != nil {
		t.Fatal(err)
	}

	// User without admin claim (time@test.com does not have admin claim)
	nonAdminToken, err := testutils.GenerateRecordToken("users", "time@test.com")
	if err != nil {
		t.Fatal(err)
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "missing Authorization header returns 401",
			Method:         http.MethodGet,
			URL:            "/api/machine_secrets/list",
			ExpectedStatus: http.StatusUnauthorized,
			ExpectedContent: []string{
				`"status":401`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user without admin claim returns 403",
			Method: http.MethodGet,
			URL:    "/api/machine_secrets/list",
			Headers: map[string]string{
				"Authorization": nonAdminToken,
			},
			ExpectedStatus: http.StatusForbidden,
			ExpectedContent: []string{
				`"status":403`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
		{
			Name:   "user with admin claim returns 200 with list",
			Method: http.MethodGet,
			URL:    "/api/machine_secrets/list",
			Headers: map[string]string{
				"Authorization": adminToken,
			},
			ExpectedStatus: http.StatusOK,
			ExpectedContent: []string{
				// The test database has at least one machine_secrets record
				`"id":`,
				`"role":`,
				`"expiry":`,
			},
			NotExpectedContent: []string{
				// Verify hash and salt are NOT exposed
				`"sha256_hash":`,
				`"salt":`,
				`"secret":`,
			},
			TestAppFactory: testutils.SetupTestApp,
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}
