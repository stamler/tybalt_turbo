package hooks

import (
	"errors"
	"net/http"
	"testing"
	"tybalt/errs"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func assertHookErrorCode(t *testing.T, err error, expectedStatus int, expectedField string, expectedCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error but got nil")
	}

	var hookErr *errs.HookError
	if !errors.As(err, &hookErr) {
		t.Fatalf("expected HookError, got %T: %v", err, err)
	}

	if hookErr.Status != expectedStatus {
		t.Fatalf("expected status %d, got %d", expectedStatus, hookErr.Status)
	}

	fieldErr, ok := hookErr.Data[expectedField]
	if !ok {
		t.Fatalf("expected field %q in error data, got %+v", expectedField, hookErr.Data)
	}

	if fieldErr.Code != expectedCode {
		t.Fatalf("expected code %q for field %q, got %q", expectedCode, expectedField, fieldErr.Code)
	}
}

func makeRecordRequestEvent(app core.App, record *core.Record, auth *core.Record) *core.RecordRequestEvent {
	return &core.RecordRequestEvent{
		RequestEvent: &core.RequestEvent{App: app, Auth: auth},
		Record:       record,
	}
}

func TestProcessPurchaseOrder_UpdateOwnershipGuards(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	ownerAuth, err := app.FindRecordById("users", "f2j5a8vk006baub")
	if err != nil {
		t.Fatalf("failed to load owner auth record: %v", err)
	}
	nonOwnerAuth, err := app.FindRecordById("users", "rzr98oadsp9qc11")
	if err != nil {
		t.Fatalf("failed to load non-owner auth record: %v", err)
	}

	t.Run("non-owner update rejected", func(t *testing.T) {
		record, err := app.FindRecordById("purchase_orders", "46efdq319b22480")
		if err != nil {
			t.Fatalf("failed to load purchase order: %v", err)
		}

		_, err = ProcessPurchaseOrder(app, makeRecordRequestEvent(app, record, nonOwnerAuth))
		assertHookErrorCode(t, err, http.StatusForbidden, "uid", "not_owner")
	})

	t.Run("uid immutable on update", func(t *testing.T) {
		record, err := app.FindRecordById("purchase_orders", "46efdq319b22480")
		if err != nil {
			t.Fatalf("failed to load purchase order: %v", err)
		}
		record.Set("uid", "rzr98oadsp9qc11")

		_, err = ProcessPurchaseOrder(app, makeRecordRequestEvent(app, record, ownerAuth))
		assertHookErrorCode(t, err, http.StatusBadRequest, "uid", "immutable_field")
	})

	t.Run("non-unapproved update rejected", func(t *testing.T) {
		record, err := app.FindRecordById("purchase_orders", "2plsetqdxht7esg")
		if err != nil {
			t.Fatalf("failed to load active purchase order: %v", err)
		}
		activeOwnerAuth, err := app.FindRecordById("users", "rzr98oadsp9qc11")
		if err != nil {
			t.Fatalf("failed to load active PO owner auth record: %v", err)
		}

		_, err = ProcessPurchaseOrder(app, makeRecordRequestEvent(app, record, activeOwnerAuth))
		assertHookErrorCode(t, err, http.StatusBadRequest, "status", "invalid_status")
	})
}

func TestProcessExpense_UpdateOwnershipGuards(t *testing.T) {
	app, err := tests.NewTestApp("../test_pb_data")
	if err != nil {
		t.Fatalf("failed to init test app: %v", err)
	}
	defer app.Cleanup()

	ownerAuth, err := app.FindRecordById("users", "f2j5a8vk006baub")
	if err != nil {
		t.Fatalf("failed to load owner auth record: %v", err)
	}
	nonOwnerAuth, err := app.FindRecordById("users", "rzr98oadsp9qc11")
	if err != nil {
		t.Fatalf("failed to load non-owner auth record: %v", err)
	}

	t.Run("non-owner update rejected", func(t *testing.T) {
		record, err := app.FindRecordById("expenses", "77i1224mudailrb")
		if err != nil {
			t.Fatalf("failed to load expense: %v", err)
		}

		err = ProcessExpense(app, makeRecordRequestEvent(app, record, nonOwnerAuth))
		assertHookErrorCode(t, err, http.StatusForbidden, "uid", "not_owner")
	})

	t.Run("uid immutable on update", func(t *testing.T) {
		record, err := app.FindRecordById("expenses", "77i1224mudailrb")
		if err != nil {
			t.Fatalf("failed to load expense: %v", err)
		}
		record.Set("uid", "rzr98oadsp9qc11")

		err = ProcessExpense(app, makeRecordRequestEvent(app, record, ownerAuth))
		assertHookErrorCode(t, err, http.StatusBadRequest, "uid", "immutable_field")
	})
}
