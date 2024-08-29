# The Time Sheet Rejection System

## The RejectModal component

- implemented in ui/src/lib/components/ directory as RejectModal.svelte
- The RejectModal component is very similar to the existing ShareModal component
  which is found in ui/src/lib/components/ShareModal.svelte. The ShareModal
  component should be used as a template and modified as needed to create the
  RejectModal. The RejectModal is used to reject a timesheet by prompting the
  user to enter a rejection reason. Once the user enters the reason and clicks
  save, the reject-timesheet route is called with a payload containing the
  timesheet ID and the rejection reason. There is also a cancel button that will
  close the modal without making any changes.

## New API route

- implemented in the existing app/routes/routes.go file
- POST /api/reject-timesheet
  - This route will be used to reject a timesheet.
  - It will take in a JSON body with the following structure:
    - rejectionReason (string): The reason for rejecting the timesheet.
    - timeSheetId (string): The ID of the timesheet to reject.
  - like the approve-timesheet route, this route verifies that the requestor's
    auth id matches the approver column in the time_sheet record
  - If the timesheet is already locked, it throws an error
  - If the timesheet is not submitted it throws an error
  - Everything is performed in a transaction. See the approve-timesheet route
    for details of this. If the checks pass, the time_sheets record is updated
    as follows: The rejected field is set to true The rejection_reason is set as
    specified in the JSON body The rejector column is set to the id of the user
    calling the function.

## updates to ui/src/routes/time/sheets/list/+page.svelte

- a reject function is created in this file that behaves similarly to the
  approve function but instead of calling the rejection api directly it opens
  the RejectModal. This means the RejectModal will need to be imported like
  the ShareModal.
- The reject span is updated into a button which when clicked opens the reject
  modal with the appropriate id.

## other

If any other changes are needed to implement this system, create or update files
as needed. Add comments in the code as you see fit to make it clear what is
happening so it is maintainable and legible later
