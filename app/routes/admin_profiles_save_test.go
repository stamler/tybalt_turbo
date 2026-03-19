package routes

import (
	"testing"
	"tybalt/hooks"
	"tybalt/internal/testseed"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	adminProfilesSaveTestRecordID = "35i85kqy88hfsfc"
	adminProfilesSaveTestUID      = "f2j5a8vk006baub"
	adminClaimID                  = "cjna52siibr7zgq"
	corporateClaimID              = "corpclaim000001"
	corporateBranchID             = "corpbranch00001"
	defaultBranchID               = "80875lm27v8wgi4"
)

func TestSaveAdminProfileWithClaims_CommitsClaimsAndProfileTogether(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	authRecord, err := app.FindAuthRecordByEmail("users", "author@soup.com")
	if err != nil {
		t.Fatalf("failed to load auth record: %v", err)
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		_, saveErr := saveAdminProfileWithClaims(txApp, authRecord, adminProfilesSaveTestRecordID, saveAdminProfileWithClaimsRequest{
			AdminProfile: map[string]any{
				"uid":            adminProfilesSaveTestUID,
				"default_branch": corporateBranchID,
			},
			ClaimIDs: []string{adminClaimID, corporateClaimID},
		})
		return saveErr
	})
	if err != nil {
		t.Fatalf("expected save to succeed: %v", err)
	}

	adminProfile, err := app.FindRecordById("admin_profiles", adminProfilesSaveTestRecordID)
	if err != nil {
		t.Fatalf("failed to reload admin profile: %v", err)
	}
	if got := adminProfile.GetString("default_branch"); got != corporateBranchID {
		t.Fatalf("expected default_branch %q, got %q", corporateBranchID, got)
	}

	userClaims, err := app.FindRecordsByFilter(
		"user_claims",
		"uid={:uid} && cid={:cid}",
		"",
		0,
		0,
		dbx.Params{"uid": adminProfilesSaveTestUID, "cid": corporateClaimID},
	)
	if err != nil {
		t.Fatalf("failed to query user_claims: %v", err)
	}
	if len(userClaims) != 1 {
		t.Fatalf("expected 1 corporate user_claim after save, got %d", len(userClaims))
	}
}

func TestSaveAdminProfileWithClaims_RollsBackClaimChangesOnValidationFailure(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	t.Cleanup(app.Cleanup)
	hooks.AddHooks(app)
	AddRoutes(app)

	authRecord, err := app.FindAuthRecordByEmail("users", "author@soup.com")
	if err != nil {
		t.Fatalf("failed to load auth record: %v", err)
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		_, saveErr := saveAdminProfileWithClaims(txApp, authRecord, adminProfilesSaveTestRecordID, saveAdminProfileWithClaimsRequest{
			AdminProfile: map[string]any{
				"uid":            adminProfilesSaveTestUID,
				"default_branch": "missingbranch000",
			},
			ClaimIDs: []string{adminClaimID, corporateClaimID},
		})
		return saveErr
	})
	if err == nil {
		t.Fatal("expected save to fail")
	}

	adminProfile, err := app.FindRecordById("admin_profiles", adminProfilesSaveTestRecordID)
	if err != nil {
		t.Fatalf("failed to reload admin profile: %v", err)
	}
	if got := adminProfile.GetString("default_branch"); got != defaultBranchID {
		t.Fatalf("expected default_branch to remain %q after rollback, got %q", defaultBranchID, got)
	}

	userClaims, err := app.FindRecordsByFilter(
		"user_claims",
		"uid={:uid} && cid={:cid}",
		"",
		0,
		0,
		dbx.Params{"uid": adminProfilesSaveTestUID, "cid": corporateClaimID},
	)
	if err != nil {
		t.Fatalf("failed to query user_claims: %v", err)
	}
	if len(userClaims) != 0 {
		t.Fatalf("expected no corporate user_claim after rollback, got %d", len(userClaims))
	}
}
