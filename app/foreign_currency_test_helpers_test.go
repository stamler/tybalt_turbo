package main

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/hook"
)

func performTestAPIRequest(
	t testing.TB,
	app *tests.TestApp,
	method string,
	path string,
	body io.Reader,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("failed to create router: %v", err)
	}

	recorder := httptest.NewRecorder()
	serveEvent := new(core.ServeEvent)
	serveEvent.App = app
	serveEvent.Router = baseRouter

	err = app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error {
		e.Router.Bind(&hook.Handler[*core.RequestEvent]{
			Func: func(re *core.RequestEvent) error {
				return re.Next()
			},
			Priority: -9999,
		})

		if body == nil {
			body = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, body)
		req.Header.Set("content-type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		mux, err := e.Router.BuildMux()
		if err != nil {
			t.Fatalf("failed to build router mux: %v", err)
		}
		mux.ServeHTTP(recorder, req)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to trigger serve event: %v", err)
	}

	return recorder
}

func mustReadBody(t testing.TB, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	return recorder.Body.String()
}

func mustStatus(t testing.TB, recorder *httptest.ResponseRecorder, want int) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("expected status %d, got %d; body=%s", want, recorder.Code, recorder.Body.String())
	}
}
