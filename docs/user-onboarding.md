# User Onboarding in Turbo

Audience: HR, Accounting, and the time-off manager

Verified against the current implementation on 2026-04-15.

## Summary

1. The new user must sign in to Turbo with Microsoft at least once.
2. That first sign-in creates the auth account and an `admin_profiles` record with a generated placeholder payroll ID in the `9xxxxxxxx` range.
3. That first sign-in does not create the user's `profiles` record.
4. The user must complete their own profile from the Account section by clicking their email address and saving their details.
5. HR then updates the user's Admin Profile, including replacing the placeholder payroll ID with the real payroll ID from Accounting.
6. If the user has opening time-off balances, the time-off manager sets `opening_date`, `opening_ov`, and `opening_op`.

## Recommended Procedure

1. Ask the new user to sign in to Turbo with Microsoft.
   This is required before HR can complete the rest of the setup, because the first Microsoft login creates the user's `admin_profiles` record.

2. Ask the new user to open Account and click their email address in the left sidebar.
   They should complete and save their profile details, especially:
   - given name
   - surname
   - manager
   - alternate manager, if needed
   - default division
   - default role, if used

3. HR opens `Admin Profiles`, finds the user, and updates the Admin Profile.
   At minimum, HR should replace the generated placeholder payroll ID with the real payroll ID from Accounting.
   HR should also review any HR-owned fields that apply to the employee, such as:
   - active
   - default branch
   - salary
   - time sheet expected
   - job title
   - default charge-out rate

4. If the user has opening time-off balances, the time-off manager updates the same Admin Profile.
   Enter:
   - `opening_date`
   - `opening_ov`
   - `opening_op`

## Important Notes

- If the user has never signed in, there will be no Admin Profile for HR to edit yet.
- The placeholder payroll ID is expected until HR or the time-off manager replaces it with the real payroll ID.
- `opening_date` must be a valid payroll Sunday.
- If `opening_ov` or `opening_op` is non-zero, `opening_date` is required.
- If the user has no opening time-off balances, `opening_date` can be left blank.
- In the current implementation, HR can also edit `opening_date`, `opening_ov`, and `opening_op`. If your business process assigns that responsibility to the time-off manager, keep that as an operational rule.
- In the current implementation, the time-off manager can also edit `payroll_id`. If you want Accounting and HR to remain the owners of payroll IDs, keep that as an operational rule.
