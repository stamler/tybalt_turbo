/*
SQLite query adapted from descriptions/Reports.md#payrollReport-FoldedAmendments.sql
Generates a weekly payroll report matching the structure of the original MySQL query.
Requires one parameter: the week ending date (Saturday) as 'YYYY-MM-DD'.
Differences from MySQL version:
- Uses SQLite functions (strftime, JSON_GROUP_ARRAY, COALESCE, CASE WHEN).
- Joins tables based on the SQLite schema (app/sqlite_schema.sql).
- Assumes boolean fields (salary) are stored as INTEGER (0 or 1).
- Manager name logic: Uses the amender's name if amendments exist for the week,
  otherwise uses the timesheet approver's name, consistent with the original SQL description.
- hasAmendmentsForWeeksEnding outputs '[]' (empty JSON array) if no amendments exist,
  otherwise a JSON array of relevant amendment week ending dates.
*/

-- Common Table Expression (CTE) to find amendment week endings for the target committed week
WITH AmendmentWeeks AS (
    SELECT
        ta.uid,
        JSON_GROUP_ARRAY(ta.week_ending) as hasAmendmentsForWeeksEnding
    FROM (
        -- Ensure we only group distinct week endings per user for the target committed week
        SELECT DISTINCT ta_inner.uid, ta_inner.week_ending
        FROM time_amendments ta_inner
        WHERE ta_inner.committed_week_ending = {:weekEnding} -- Param 1: Target WeekEnding (YYYY-MM-DD)
    ) ta
    GROUP BY ta.uid
),
-- CTE combining Amendments (X) and regular Entries (Z)
BASE AS (
    -- X: Time Amendments relevant to the committed week
    SELECT
        ap.payroll_id AS payrollId,
        ap.work_week_hours AS workWeekHours,
        ta.committed_week_ending AS reportWeekEnding, -- Use committed_week_ending as the key date for amendments
        p_user.surname,
        p_user.given_name AS givenName,
        -- Use committer's name as manager for amendments
        COALESCE(p_commit.given_name || ' ' || p_commit.surname, 'Unknown Committer') AS manager,
        ta.meals_hours AS meals,
        ta.payout_request_amount AS payoutRequestAmount,
        ap.salary,
        ta.uid AS primaryUid,
        tt.code AS timetype,
        ta.hours,
        aw.hasAmendmentsForWeeksEnding -- Joined from AmendmentWeeks CTE
    FROM time_amendments ta
    JOIN admin_profiles ap ON ta.uid = ap.uid
    JOIN profiles p_user ON ta.uid = p_user.uid
    JOIN time_types tt ON ta.time_type = tt.id
    LEFT JOIN profiles p_commit ON ta.committer = p_commit.uid -- Join for committer name
    LEFT JOIN AmendmentWeeks aw ON ta.uid = aw.uid             -- Left Join ensures all amendments for the committed week are included
    WHERE ta.committed_week_ending = {:weekEnding} -- Param 2: Target WeekEnding (YYYY-MM-DD)

    UNION ALL

    -- Z: Regular Time Entries for the week
    SELECT
        ap.payroll_id AS payrollId,
        ap.work_week_hours AS workWeekHours,
        ts.week_ending AS reportWeekEnding, -- Use timesheet week_ending as the key date for entries
        p_user.surname,
        p_user.given_name AS givenName,
        -- Use approver's name as manager for regular entries
        COALESCE(p_approve.given_name || ' ' || p_approve.surname, 'Unknown Approver') AS manager,
        te.meals_hours AS meals,
        te.payout_request_amount AS payoutRequestAmount,
        ap.salary,
        te.uid AS primaryUid,
        tt.code AS timetype,
        te.hours,
        NULL AS hasAmendmentsForWeeksEnding -- Regular entries don't contribute to this
    FROM time_entries te
    JOIN time_sheets ts ON te.tsid = ts.id
    JOIN admin_profiles ap ON ts.uid = ap.uid
    JOIN profiles p_user ON ts.uid = p_user.uid
    JOIN time_types tt ON te.time_type = tt.id
    LEFT JOIN profiles p_approve ON ts.approver = p_approve.uid -- Join for approver name
    WHERE ts.week_ending = {:weekEnding} -- Param 3: Target WeekEnding (YYYY-MM-DD)
),
-- CTE to aggregate BASE data by user for the reporting week
MIDDLE AS (
    SELECT
        -- Select columns needed for the next stage + grouping keys
        primaryUid,
        reportWeekEnding, -- Keep the week ending date associated with the group
        -- Aggregate values using appropriate functions
        -- Use MAX for single-value fields to get non-NULL value if amendments/entries exist
        MAX(payrollId) as payrollId,
        MAX(surname) as surname,
        MAX(givenName) as givenName,
        MAX(manager) as manager, -- Takes amender name if amendments exist, else approver name
        MAX(salary) as salary,
        MAX(workWeekHours) as workWeekHours,
        MAX(hasAmendmentsForWeeksEnding) as hasAmendmentsForWeeksEnding,
        -- Sum numerical fields, coalescing NULLs to 0
        SUM(COALESCE(meals, 0)) AS mealsSum,
        SUM(CASE WHEN timetype = 'OR' THEN 1 ELSE 0 END) AS daysOffRotation,
        SUM(CASE WHEN timetype IN ('R', 'RT') THEN COALESCE(hours, 0) ELSE 0 END) AS hoursWorked,
        SUM(CASE WHEN timetype = 'OB' THEN COALESCE(hours, 0) ELSE 0 END) AS Bereavement,
        SUM(CASE WHEN timetype = 'OH' THEN COALESCE(hours, 0) ELSE 0 END) AS Stat,
        SUM(CASE WHEN timetype = 'OP' THEN COALESCE(hours, 0) ELSE 0 END) AS PPTO,
        SUM(CASE WHEN timetype = 'OS' THEN COALESCE(hours, 0) ELSE 0 END) AS Sick,
        SUM(CASE WHEN timetype = 'OV' THEN COALESCE(hours, 0) ELSE 0 END) AS Vacation,
        SUM(CASE WHEN timetype = 'RB' THEN COALESCE(hours, 0) ELSE 0 END) AS overtimeHoursToBank,
        SUM(COALESCE(payoutRequestAmount, 0)) AS overtimePayoutRequested
    FROM BASE
    -- Group by user, ensuring all entries/amendments for that user in the week are combined
    GROUP BY primaryUid
),
-- CTE to calculate derived hour values based on aggregated data
FINAL AS (
    SELECT
        *,
        -- Calculate Salary hours over 44
        CASE
            WHEN salary = 1 AND hoursWorked > 44 THEN hoursWorked - 44
            ELSE 0
        END AS salaryHoursOver44,
        -- Calculate Adjusted hours worked (considering salary cap logic)
        CASE
            WHEN salary = 1 THEN -- Salaried employees
                CASE
                    -- Cap total work+stat+bereavement at workWeekHours
                    WHEN hoursWorked + COALESCE(Stat, 0) + COALESCE(Bereavement, 0) > workWeekHours
                    THEN workWeekHours - COALESCE(Stat, 0) - COALESCE(Bereavement, 0)
                    ELSE hoursWorked -- Otherwise, use actual hours worked
                END
            ELSE -- Hourly employees
                CASE
                    -- Cap regular hours at 44 for hourly
                    WHEN hoursWorked > 44 THEN 44
                    ELSE hoursWorked
                END
        END AS adjustedHoursWorked,
        -- Calculate Total overtime hours (applies only to hourly)
        CASE
            WHEN salary = 0 AND hoursWorked > 44 THEN hoursWorked - 44
            ELSE 0
        END AS totalOvertimeHours
    FROM MIDDLE
)
-- Final SELECT to format and output the report columns
SELECT
    payrollId,
    strftime('%Y %b %d', reportWeekEnding) AS weekEnding,
    surname,
    givenName,
    (givenName || ' ' || surname) AS name,
    manager,
    COALESCE(mealsSum, 0) AS meals,
    COALESCE(daysOffRotation, 0) AS "days off rotation",
    COALESCE(hoursWorked, 0) AS "hours worked",
    salaryHoursOver44,
    COALESCE(adjustedHoursWorked, 0) AS adjustedHoursWorked,
    totalOvertimeHours AS "total overtime hours",
    -- Calculate Overtime hours to pay (hourly only, considers banked hours)
    CASE
        WHEN salary = 0 THEN -- Hourly only
            CASE
                -- Subtract banked hours from total overtime
                WHEN overtimeHoursToBank > 0 THEN totalOvertimeHours - overtimeHoursToBank
                ELSE totalOvertimeHours
            END
        ELSE 0 -- Salary employees are not paid overtime this way
    END AS "overtime hours to pay",
    COALESCE(Bereavement, 0) AS Bereavement,
    COALESCE(Stat, 0) AS "Stat Holiday",
    COALESCE(PPTO, 0) AS PPTO,
    COALESCE(Sick, 0) AS Sick,
    COALESCE(Vacation, 0) AS Vacation,
    COALESCE(overtimeHoursToBank, 0) AS "overtime hours to bank",
    COALESCE(overtimePayoutRequested, 0) AS "Overtime Payout Requested",
    -- Use '[]' for null/empty amendment list for valid JSON
    COALESCE(hasAmendmentsForWeeksEnding, '[]') AS hasAmendmentsForWeeksEnding,
    salary
FROM FINAL
-- Order results consistently
ORDER BY LENGTH(payrollId), payrollId;