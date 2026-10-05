package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func decodeJSONResponseForTest[T any](t *testing.T, rec *httptest.ResponseRecorder, status int, label string) T {
	t.Helper()
	var response T
	if rec.Code != status {
		t.Fatalf("%s status = %d, want %d; body=%s", label, rec.Code, status, rec.Body.String())
	}
	if status != http.StatusOK {
		return response
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode %s response: %v", label, err)
	}
	return response
}
