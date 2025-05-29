#!/bin/bash

# Wrapper script for payroll expense report preparation
# Calls the script with appropriate parameters

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"$SCRIPT_DIR/old_report_prep.sh" "expenses_payroll" "expense"
"$SCRIPT_DIR/old_report_prep.sh" "expenses_weekly" "expense"
"$SCRIPT_DIR/old_report_prep.sh" "time_weekly" "weekly_time"
"$SCRIPT_DIR/old_report_prep.sh" "time_summary" "time_summary"
"$SCRIPT_DIR/old_report_prep.sh" "time_payroll/week1" "payroll_time"
"$SCRIPT_DIR/old_report_prep.sh" "time_payroll/week2" "payroll_time"