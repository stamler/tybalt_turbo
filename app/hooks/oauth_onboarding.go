package hooks

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"tybalt/constants"
	"unicode"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbauth "github.com/pocketbase/pocketbase/tools/auth"
)

type microsoftOnboardingData struct {
	GivenName         string
	Surname           string
	DisplayName       string
	Mail              string
	UserPrincipalName string
	ProviderID        string
}

// microsoftOnboardingDataFromAuthUser extracts the specific Microsoft values we
// rely on during auth-time onboarding.
//
// PocketBase gives us a normalized AuthUser, but Microsoft's most useful values
// still live in RawUser and are not always mapped the way this app needs. We
// pull them into one small struct so the main hook can read like business logic
// instead of a pile of map lookups and fallbacks.
func microsoftOnboardingDataFromAuthUser(user *pbauth.AuthUser) microsoftOnboardingData {
	if user == nil {
		return microsoftOnboardingData{}
	}

	return microsoftOnboardingData{
		GivenName:         oauthRawString(user.RawUser, "givenName"),
		Surname:           oauthRawString(user.RawUser, "surname"),
		DisplayName:       firstNonEmpty(strings.TrimSpace(user.Name), oauthRawString(user.RawUser, "displayName")),
		Mail:              oauthRawString(user.RawUser, "mail"),
		UserPrincipalName: oauthRawString(user.RawUser, "userPrincipalName"),
		ProviderID:        strings.TrimSpace(user.Id),
	}
}

func oauthRawString(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}

	value, ok := raw[key]
	if !ok {
		return ""
	}

	s, ok := value.(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(s)
}

func (d microsoftOnboardingData) fullName() string {
	if d.GivenName != "" && d.Surname != "" {
		return strings.TrimSpace(d.GivenName + " " + d.Surname)
	}

	return strings.TrimSpace(d.DisplayName)
}

func (d microsoftOnboardingData) fallbackEmail() string {
	return firstNonEmpty(d.Mail, d.UserPrincipalName)
}

// ensureMicrosoftUserOnboarded performs the automatic post-auth work that is
// safe to do from Microsoft identity data alone.
//
// We intentionally keep this narrow. `admin_profiles` can be created from
// application defaults plus the user id, but `profiles` cannot because the
// business schema requires fields like `manager` that Microsoft does not
// provide. So "successful onboarding" here means the auth record exists and
// admin_profiles exists; collecting richer profile data happens later in-app.
//
// This helper also defines the boundary between Microsoft-owned identity data
// and app-owned business data:
// - Microsoft may seed users.name / users.email / users.username at first login
// - profiles is user/business-owned and not inferred here
// - admin_profiles is app-owned and only ensured to exist
//
// Planned follow-up: users.name and users.username should likely move from
// "seed on first login" to "sync on Microsoft login" so later directory
// changes are reflected in the PocketBase auth record as well.
func ensureMicrosoftUserOnboarded(app core.App, userRecord *core.Record) error {
	if userRecord == nil {
		return fmt.Errorf("missing user record for onboarding")
	}

	return ensureUserAdminProfile(app, userRecord.Id)
}

// ensureUserAdminProfile is idempotent and safe to call when the auth flow has
// decided a missing admin profile should be repaired.
//
// The old onboarding flow created admin_profiles with the shared payroll id
// "999999". That worked once and then failed for every later user because the
// column is unique. This helper first checks whether the row already exists and
// otherwise creates it with app defaults plus a generated placeholder payroll
// id that should be unique for each user.
//
// We intentionally verify collision cases by re-querying the database instead
// of parsing the save error text. PocketBase may normalize unique violations
// into structured validation errors, so string matching on err.Error() is too
// brittle for concurrency-sensitive onboarding code.
func ensureUserAdminProfile(app core.App, uid string) error {
	if uid == "" {
		return fmt.Errorf("missing user id for admin profile onboarding")
	}

	_, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err == nil {
		return nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed checking admin profile for %s: %w", uid, err)
	}

	adminProfiles, err := app.FindCollectionByNameOrId("admin_profiles")
	if err != nil {
		return err
	}

	for attempt := range 100 {
		payrollID := generatedPayrollPlaceholder(uid, attempt)
		record := core.NewRecord(adminProfiles)
		record.Set("uid", uid)
		record.Set("active", true)
		record.Set("work_week_hours", constants.DEFAULT_WORK_WEEK_HOURS)
		record.Set("default_charge_out_rate", constants.DEFAULT_CHARGE_OUT_RATE)
		record.Set("skip_min_time_check", "no")
		record.Set("salary", false)
		record.Set("untracked_time_off", false)
		record.Set("time_sheet_expected", false)
		record.Set("default_branch", constants.DEFAULT_BRANCH_ID)
		record.Set("payroll_id", payrollID)

		saveErr := app.Save(record)
		if saveErr == nil {
			return nil
		}

		existingByUID, findErr := findAdminProfileByUID(app, uid)
		if findErr != nil {
			return fmt.Errorf("failed checking admin profile creation for %s: %w", uid, findErr)
		}
		if existingByUID != nil {
			return nil
		}

		existingByPayrollID, findErr := findAdminProfileByPayrollID(app, payrollID)
		if findErr != nil {
			return fmt.Errorf("failed checking payroll placeholder collision for %s: %w", uid, findErr)
		}
		if existingByPayrollID != nil {
			continue
		}

		return fmt.Errorf("failed to create admin profile for %s: %w", uid, saveErr)
	}

	return fmt.Errorf("failed to create unique payroll id placeholder for %s", uid)
}

func findAdminProfileByUID(app core.App, uid string) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter("admin_profiles", "uid={:uid}", dbx.Params{"uid": uid})
	if err == nil {
		return record, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

func findAdminProfileByPayrollID(app core.App, payrollID string) (*core.Record, error) {
	record, err := app.FindFirstRecordByFilter("admin_profiles", "payroll_id={:payroll_id}", dbx.Params{"payroll_id": payrollID})
	if err == nil {
		return record, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return nil, err
}

// generatedPayrollPlaceholder deterministically maps a PocketBase user id into
// a reserved 9xxxxxxxx placeholder range.
//
// Attempt 0 is the stable hash-derived value. If that rare value collides with
// another user, later attempts linearly probe forward through the same range.
// This gives us repeatable values without reintroducing the old shared-id bug.
func generatedPayrollPlaceholder(uid string, attempt int) string {
	sum := sha256.Sum256([]byte(uid))
	offset := int(binary.BigEndian.Uint32(sum[:4]) % 100000000)
	value := 900000000 + ((offset + attempt) % 100000000)
	return strconv.Itoa(value)
}

// buildUniqueMicrosoftUsername turns Microsoft identity data into a valid,
// unique application username.
//
// The preferred source is the local-part of `mail` or `userPrincipalName`, so
// `alice.jones@tbte.ca` becomes `alice.jones`. If that is missing or sanitizes
// to nothing, we fall back to a provider-id-based username. If the candidate is
// already taken, we append `-2`, `-3`, and so on until we find a free value.
//
// This is intentionally a creation-time concern, not a per-login sync rule.
// Even if Microsoft later changes the user's email or UPN, we currently keep
// the existing app username stable once it has been created.
//
// Planned follow-up: revisit this policy so Microsoft identity changes can
// update users.username in a controlled way, together with any downstream code
// that currently treats username as effectively immutable.
func buildUniqueMicrosoftUsername(app core.App, collection *core.Collection, data microsoftOnboardingData, excludeRecordID string) (string, error) {
	usernameField, ok := collection.Fields.GetByName("username").(*core.TextField)
	if !ok || usernameField == nil {
		return "", fmt.Errorf("users.username field is not configured as text")
	}

	maxLength := usernameField.Max
	if maxLength <= 0 {
		maxLength = 150
	}

	base := normalizeUsernameBase(localPart(firstNonEmpty(data.Mail, data.UserPrincipalName)), maxLength)
	if base == "" {
		base = normalizeUsernameBase("u-"+shortProviderIdentifier(data.ProviderID), maxLength)
	}
	if base == "" {
		base = "u-user"
	}

	for attempt := range 100 {
		candidate := withUsernameSuffix(base, attempt, maxLength)
		if usernameField.ValidatePlainValue(candidate) != nil {
			continue
		}

		exists, err := usernameExists(app, candidate, excludeRecordID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to allocate unique username from %q", base)
}

// localPart returns everything before the first `@`. We use it for both email
// addresses and userPrincipalName values.
func localPart(identity string) string {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return ""
	}

	at := strings.Index(identity, "@")
	if at < 0 {
		return identity
	}
	if at == 0 {
		return ""
	}

	return identity[:at]
}

func shortProviderIdentifier(providerID string) string {
	normalized := normalizeUsernameBase(providerID, 12)
	normalized = strings.Trim(normalized, "-.")
	if normalized == "" {
		return "user"
	}
	return normalized
}

// normalizeUsernameBase reshapes arbitrary Microsoft identity strings into a
// conservative username base that satisfies the existing PocketBase field
// validator.
//
// The transformation is intentionally simple and stable:
// - lowercase
// - keep letters, digits, `_`, `.`, `-`
// - replace everything else with `-`
// - collapse repeated invalid runs
// - strip bad leading/trailing punctuation
// - prefix with `u` if the first rune would still be invalid
// - trim to the configured field length
func normalizeUsernameBase(raw string, maxLength int) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ""
	}

	var builder strings.Builder
	lastWasDash := false
	for _, r := range raw {
		var out rune
		switch {
		case r >= 'a' && r <= 'z':
			out = r
		case r >= '0' && r <= '9':
			out = r
		case r == '_' || r == '.' || r == '-':
			out = r
		default:
			out = '-'
		}

		if out == '-' {
			if builder.Len() == 0 || lastWasDash {
				continue
			}
			lastWasDash = true
		} else {
			lastWasDash = false
		}

		builder.WriteRune(out)
	}

	normalized := strings.Trim(builder.String(), "-.")
	if normalized == "" {
		return ""
	}

	if !isValidUsernameFirstRune(rune(normalized[0])) {
		normalized = "u" + normalized
	}

	if maxLength > 0 && len(normalized) > maxLength {
		normalized = normalized[:maxLength]
		normalized = strings.TrimRight(normalized, "-.")
	}

	if normalized == "" {
		return ""
	}

	if !isValidUsernameFirstRune(rune(normalized[0])) {
		if maxLength == 1 {
			return "u"
		}
		normalized = "u" + normalized
		if maxLength > 0 && len(normalized) > maxLength {
			normalized = normalized[:maxLength]
		}
	}

	return normalized
}

func isValidUsernameFirstRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// withUsernameSuffix applies the collision suffix while preserving the field
// length constraint. If we need to shorten the base to make room for `-2`,
// `-3`, etc., we trim first and then clean up trailing punctuation.
func withUsernameSuffix(base string, attempt int, maxLength int) string {
	if attempt == 0 {
		return trimUsernameToLength(base, maxLength)
	}

	suffix := "-" + strconv.Itoa(attempt+1)
	trimmedBase := base
	if maxLength > 0 && len(trimmedBase)+len(suffix) > maxLength {
		trimmedBase = trimmedBase[:maxLength-len(suffix)]
		trimmedBase = strings.TrimRight(trimmedBase, "-.")
	}
	if trimmedBase == "" {
		trimmedBase = "u"
	}

	return trimUsernameToLength(trimmedBase+suffix, maxLength)
}

func trimUsernameToLength(username string, maxLength int) string {
	if maxLength > 0 && len(username) > maxLength {
		username = username[:maxLength]
	}
	return strings.TrimRight(username, "-.")
}

// usernameExists uses a case-insensitive lookup so collision handling matches
// how usernames are effectively treated by users and the database.
func usernameExists(app core.App, username string, excludeRecordID string) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM users
		WHERE username = {:username} COLLATE NOCASE
	`
	params := dbx.Params{"username": username}
	if excludeRecordID != "" {
		query += ` AND id != {:exclude}`
		params["exclude"] = excludeRecordID
	}

	if err := app.DB().NewQuery(query).Bind(params).Row(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
