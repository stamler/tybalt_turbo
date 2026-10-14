package migrations_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ganigeorgiev/fexpr"
	"github.com/pocketbase/pocketbase/core"
	_ "tybalt/migrations"
)

// PocketBase compiles every @collection reference in a rule into a LEFT JOIN
// with no ON condition, so several references form a cross product of whole
// tables. An AND-only rule stays cheap because PocketBase checks each
// reference with a correlated NOT EXISTS that rejects a denied record before
// the cross product is walked, and an allowed record stops at the first row.
// Any || disables that short circuit, even a branch on the record itself, and
// a denied request then walks the whole cross product. List and view rules
// select many records, so they get no short circuit at all.
//
// The rules are parsed with fexpr, the parser PocketBase uses, so comments are
// dropped and text inside quoted strings is never mistaken for a reference or
// an operator.

// collectionRuleProblems returns why rule could join whole tables, or nothing
// when it cannot. multiRecord marks list and view rules.
func collectionRuleProblems(rule string, multiRecord bool) ([]string, error) {
	groups, err := fexpr.Parse(rule)
	if err != nil {
		return nil, err
	}
	var refs, ors []string
	var walkToken func(fexpr.Token)
	walkToken = func(token fexpr.Token) {
		switch token.Type {
		case fexpr.TokenIdentifier:
			if strings.HasPrefix(token.Literal, "@collection.") {
				refs = append(refs, token.Literal)
			}
		case fexpr.TokenFunction:
			args, _ := token.Meta.([]fexpr.Token)
			for _, arg := range args {
				walkToken(arg)
			}
		}
	}
	var walkGroups func([]fexpr.ExprGroup)
	walkGroups = func(groups []fexpr.ExprGroup) {
		for _, group := range groups {
			if group.Join == fexpr.JoinOr {
				ors = append(ors, "|| "+formatRuleItem(group.Item))
			}
			switch item := group.Item.(type) {
			case fexpr.Expr:
				walkToken(item.Left)
				walkToken(item.Right)
			case []fexpr.ExprGroup:
				walkGroups(item)
			}
		}
	}
	walkGroups(groups)

	if len(refs) == 0 {
		return nil, nil
	}
	var problems []string
	if multiRecord {
		problems = append(problems, "list and view rules must not use @collection: "+strings.Join(refs, ", "))
	}
	if len(ors) > 0 {
		problems = append(problems, fmt.Sprintf("rules using @collection (%s) must not use ||: %s",
			strings.Join(refs, ", "), strings.Join(ors, "; ")))
	}
	return problems, nil
}

func formatRuleItem(item any) string {
	switch item := item.(type) {
	case fexpr.Expr:
		return strings.Join([]string{formatRuleToken(item.Left), string(item.Op), formatRuleToken(item.Right)}, " ")
	case []fexpr.ExprGroup:
		var parts []string
		for i, group := range item {
			if i > 0 {
				parts = append(parts, string(group.Join))
			}
			parts = append(parts, formatRuleItem(group.Item))
		}
		return "(" + strings.Join(parts, " ") + ")"
	}
	return fmt.Sprint(item)
}

func formatRuleToken(token fexpr.Token) string {
	switch token.Type {
	case fexpr.TokenText:
		return "'" + token.Literal + "'"
	case fexpr.TokenFunction:
		args, _ := token.Meta.([]fexpr.Token)
		formatted := make([]string, len(args))
		for i, arg := range args {
			formatted[i] = formatRuleToken(arg)
		}
		return token.Literal + "(" + strings.Join(formatted, ", ") + ")"
	}
	return token.Literal
}

type namedRule struct {
	name        string
	rule        *string
	multiRecord bool
}

func collectionRules(collection *core.Collection) []namedRule {
	rules := []namedRule{
		{"listRule", collection.ListRule, true},
		{"viewRule", collection.ViewRule, true},
		{"createRule", collection.CreateRule, false},
		{"updateRule", collection.UpdateRule, false},
		{"deleteRule", collection.DeleteRule, false},
	}
	if collection.IsAuth() {
		rules = append(rules,
			namedRule{"authRule", collection.AuthRule, false},
			namedRule{"manageRule", collection.ManageRule, false},
		)
	}
	return rules
}

// schemaRuleProblems checks every rule of every collection in app.
func schemaRuleProblems(t *testing.T, app core.App) []string {
	t.Helper()
	collections, err := app.FindAllCollections()
	if err != nil {
		t.Fatal(err)
	}
	var problems []string
	for _, collection := range collections {
		for _, rule := range collectionRules(collection) {
			if rule.rule == nil || *rule.rule == "" {
				continue
			}
			found, err := collectionRuleProblems(*rule.rule, rule.multiRecord)
			if err != nil {
				t.Fatalf("%s.%s does not parse: %v", collection.Name, rule.name, err)
			}
			for _, problem := range found {
				problems = append(problems, fmt.Sprintf("%s.%s: %s", collection.Name, rule.name, problem))
			}
		}
	}
	return problems
}

func TestApiRulesHaveNoUnconditionedCollectionJoins(t *testing.T) {
	app := newMigratedApp(t)
	for _, problem := range schemaRuleProblems(t, app) {
		t.Error(problem)
	}
}

// Appending a branch on the caller to the jobs deleteRule let a denied delete
// walk the cross product of its @collection joins until the query timed out.
func TestApiRuleCheckCatchesOrInJobsDeleteRule(t *testing.T) {
	app := newMigratedApp(t)
	jobs := migrationCollection(t, app, "jobs")
	if jobs.DeleteRule == nil || !strings.Contains(*jobs.DeleteRule, "@collection.") {
		t.Fatal("jobs deleteRule no longer uses @collection, so this test proves nothing")
	}
	rule := *jobs.DeleteRule + "\n|| @request.auth.id = 'x'"
	jobs.DeleteRule = &rule
	if err := app.Save(jobs); err != nil {
		t.Fatal(err)
	}

	for _, problem := range schemaRuleProblems(t, app) {
		if strings.HasPrefix(problem, "jobs.deleteRule: ") && strings.Contains(problem, "|| @request.auth.id = 'x'") {
			return
		}
	}
	t.Fatal("the jobs deleteRule with || was not flagged")
}

func TestCollectionRuleProblems(t *testing.T) {
	cases := []struct {
		name        string
		rule        string
		multiRecord bool
		want        []string
	}{
		{
			name: "and-only references with != are allowed in a delete rule",
			rule: `@request.auth.id != "" &&
// keep jobs that still have time
@collection.time_entries.job != id &&
@collection.expenses.job != id`,
		},
		{
			name: "or on the caller",
			rule: `@collection.time_entries.job != id && @collection.expenses.job != id || @request.auth.id = 'x'`,
			want: []string{"rules using @collection (@collection.time_entries.job, @collection.expenses.job) must not use ||: || @request.auth.id = 'x'"},
		},
		{
			name: "or on the record itself",
			rule: `@collection.time_entries.job != id || number = 'zzz'`,
			want: []string{"rules using @collection (@collection.time_entries.job) must not use ||: || number = 'zzz'"},
		},
		{
			name: "or nested in a group",
			rule: `@request.auth.id != "" && (@collection.time_entries.job != id || @collection.expenses.job != id)`,
			want: []string{"rules using @collection (@collection.time_entries.job, @collection.expenses.job) must not use ||: || @collection.expenses.job != id"},
		},
		{
			name: "or in a different group from the reference",
			rule: `(@request.auth.id = id || @request.auth.id = uid) && @collection.expenses.job != id`,
			want: []string{"rules using @collection (@collection.expenses.job) must not use ||: || @request.auth.id = uid"},
		},
		{
			name: "or without a reference",
			rule: `@request.auth.id = uid || @request.auth.user_claims_via_uid.cid.name ?= 'admin'`,
		},
		{
			name:        "reference in a list rule",
			rule:        `@request.auth.id != "" && @collection.expenses.uid ?= uid`,
			multiRecord: true,
			want:        []string{"list and view rules must not use @collection: @collection.expenses.uid"},
		},
		{
			name:        "reference inside a function call",
			rule:        `strftime('%Y', @collection.expenses.created) = '2026'`,
			multiRecord: true,
			want:        []string{"list and view rules must not use @collection: @collection.expenses.created"},
		},
		{
			name:        "list rule reference with or reports both",
			rule:        `@request.auth.id = uid || @collection.expenses.approver ?= @request.auth.id`,
			multiRecord: true,
			want: []string{
				"list and view rules must not use @collection: @collection.expenses.approver",
				"rules using @collection (@collection.expenses.approver) must not use ||: || @collection.expenses.approver ?= @request.auth.id",
			},
		},
		{
			name: "or and reference inside strings are text",
			rule: `name != 'a || b' && description != "@collection.expenses.uid" && @collection.expenses.job != id`,
		},
		{
			name:        "reference inside a string in a list rule is text",
			rule:        `name = '@collection.expenses.uid'`,
			multiRecord: true,
		},
		{
			name: "or and reference inside comments are ignored",
			rule: `@collection.expenses.job != id
// used to be || @request.auth.id = uid
&& @request.auth.id != ""`,
		},
		{
			name:        "reference inside a comment in a list rule is ignored",
			rule:        "@request.auth.id = uid\n// see @collection.expenses.uid",
			multiRecord: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := collectionRuleProblems(tc.rule, tc.multiRecord)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Errorf("problems differ\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}
