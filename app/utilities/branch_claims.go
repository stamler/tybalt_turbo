package utilities

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strings"
	"tybalt/errs"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// EnsureUserCanUseBranch enforces branch-level claim restrictions for the
// target user. A branch with no allowed_claims is unrestricted.
func EnsureUserCanUseBranch(app core.App, branchID string, userID string, fieldName string) error {
	branchID = strings.TrimSpace(branchID)
	if branchID == "" {
		return nil
	}

	branch, err := app.FindRecordById("branches", branchID)
	if err != nil || branch == nil {
		return &errs.HookError{
			Status:  http.StatusBadRequest,
			Message: "branch lookup failed",
			Data: map[string]errs.CodeError{
				fieldName: {
					Code:    "invalid_branch",
					Message: "specified branch could not be found",
				},
			},
		}
	}

	allowedClaimIDs := normalizeStringList(branch.Get("allowed_claims"))
	if len(allowedClaimIDs) == 0 {
		return nil
	}

	userClaimRecords, err := app.FindRecordsByFilter(
		"user_claims",
		"uid={:uid}",
		"",
		0,
		0,
		dbx.Params{"uid": userID},
	)
	if err != nil {
		return &errs.HookError{
			Status:  http.StatusInternalServerError,
			Message: "branch claim validation failed",
			Data: map[string]errs.CodeError{
				fieldName: {
					Code:    "branch_claim_check_failed",
					Message: "unable to validate branch claim requirements",
				},
			},
		}
	}

	for _, record := range userClaimRecords {
		if slices.Contains(allowedClaimIDs, strings.TrimSpace(record.GetString("cid"))) {
			return nil
		}
	}

	requiredClaims := make([]string, 0, len(allowedClaimIDs))
	for _, claimID := range allowedClaimIDs {
		claim, claimErr := app.FindRecordById("claims", claimID)
		if claimErr == nil && claim != nil && strings.TrimSpace(claim.GetString("name")) != "" {
			requiredClaims = append(requiredClaims, strings.TrimSpace(claim.GetString("name")))
			continue
		}
		requiredClaims = append(requiredClaims, claimID)
	}
	sort.Strings(requiredClaims)

	return &errs.HookError{
		Status:  http.StatusBadRequest,
		Message: "branch claim requirement not met",
		Data: map[string]errs.CodeError{
			fieldName: {
				Code: "branch_claim_required",
				Message: fmt.Sprintf(
					"%s requires at least one of these claims: %s",
					branch.GetString("name"),
					strings.Join(requiredClaims, ", "),
				),
			},
		},
	}
}

func normalizeStringList(value any) []string {
	switch v := value.(type) {
	case []string:
		return slices.DeleteFunc(slices.Clone(v), func(item string) bool {
			return strings.TrimSpace(item) == ""
		})
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				continue
			}
			str = strings.TrimSpace(str)
			if str != "" {
				result = append(result, str)
			}
		}
		return result
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		var result []string
		if err := json.Unmarshal([]byte(v), &result); err == nil {
			return slices.DeleteFunc(result, func(item string) bool {
				return strings.TrimSpace(item) == ""
			})
		}
		return nil
	default:
		return nil
	}
}
