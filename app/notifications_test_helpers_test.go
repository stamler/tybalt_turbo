package main

import (
	"testing"

	"tybalt/internal/testutils"
	"tybalt/notifications"

	"github.com/pocketbase/pocketbase/tests"
)

func setupTestAppWithSynchronousImmediateNotifications(tb testing.TB) *tests.TestApp {
	tb.Helper()

	app := testutils.SetupTestApp(tb)
	notifications.SetSendNotificationAsyncForTest(app, false)
	return app
}
