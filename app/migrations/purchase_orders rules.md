# Rules for purchase orders

## List/Search

```pocketbase
@request.auth.id = uid ||
@request.auth.id = approver ||
@request.auth.id = second_approver
```

## Create

```pocketbase
// the caller is authenticated
@request.auth.id != "" &&

// no po_number is submitted
@request.data.po_number:isset = false &&

// status is Unapproved
@request.data.status = "Unapproved" &&

// the uid is missing or is equal to the authenticated user's id
(@request.data.uid:isset = false || @request.data.uid = @request.auth.id) &&

// no rejection properties are submitted
@request.data.rejector:isset = false &&
@request.data.rejected:isset = false &&
@request.data.rejection_reason:isset = false &&

// no approval properties are submitted
@request.data.approved:isset = false &&
@request.data.approver:isset = false &&

// no second approver properties are submitted
@request.data.second_approver:isset = false &&
@request.data.second_approval:isset = false &&
@request.data.second_approver_claim:isset = false &&

// no cancellation properties are submitted
@request.data.cancelled:isset = false &&
@request.data.canceller:isset = false
```

## Update

```pocketbase
// only the creator can update the record
uid = @request.auth.id &&

// status is Unapproved
status = 'Unapproved' &&

// no po_number is submitted
(@request.data.po_number:isset = false || po_number = @request.data.po_number) &&

// no rejection properties are submitted
(@request.data.rejector:isset = false || rejector = @request.data.rejector) &&
(@request.data.rejected:isset = false || rejected = @request.data.rejected) &&
(@request.data.rejection_reason:isset = false || rejection_reason = @request.data.rejection_reason) &&

// no approval properties are submitted
(@request.data.approved:isset = false || approved = @request.data.approved) &&
(@request.data.approver:isset = false || approver = @request.data.approver) &&

// no second approver properties are submitted
(@request.data.second_approver:isset = false || second_approver = @request.data.second_approver) &&
(@request.data.second_approval:isset = false || second_approval = @request.data.second_approval) &&
(@request.data.second_approver_claim:isset = false || second_approver_claim = @request.data.second_approver_claim) &&

// no cancellation properties are submitted
(@request.data.cancelled:isset = false || cancelled = @request.data.cancelled) &&
(@request.data.canceller:isset = false || canceller = @request.data.canceller)
```
