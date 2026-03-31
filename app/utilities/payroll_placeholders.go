package utilities

import "fmt"

const generatedPayrollPlaceholderGlob = "9[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]"

// GeneratedPayrollPlaceholderSQLCondition returns a SQLite predicate that
// matches the generated placeholder payroll-id range for the given SQL
// expression.
func GeneratedPayrollPlaceholderSQLCondition(expr string) string {
	return fmt.Sprintf(
		"(LENGTH(COALESCE(%[1]s, '')) = 9 AND COALESCE(%[1]s, '') GLOB '%[2]s')",
		expr,
		generatedPayrollPlaceholderGlob,
	)
}
