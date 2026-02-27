# Rule Deltas: Current DB -> Up Migration

## `expenses`

```txt
BEFORE: (@request.body.uid:isset = false || uid = @request.body.uid)
AFTER:  @request.body.uid:changed = false
```

## `profiles`

```txt
BEFORE:
uid = @request.auth.id

AFTER:
uid = @request.auth.id &&
@request.body.uid:changed = false
```

## `purchase_orders`

```txt
BEFORE:
uid = @request.auth.id &&

status = 'Unapproved' &&
second_approval = ""

AFTER:
uid = @request.auth.id &&
@request.body.uid:changed = false &&

status = 'Unapproved' &&
second_approval = "" &&
```

## `time_amendments`

```txt
BEFORE:
@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&
committed = "" &&

AFTER:
@request.auth.id != "" &&
@request.auth.user_claims_via_uid.cid.name ?= 'tame' &&
committed = "" &&

@request.body.uid:changed = false &&
```

## `time_entries`

```txt
BEFORE:
uid = @request.auth.id && tsid = "" &&

AFTER:
uid = @request.auth.id && tsid = "" &&
@request.body.uid:changed = false &&
```

## `time_sheets`

```txt
BEFORE:
(rejected = true && rejector != "" && rejection_reason != "") || (rejected = false)

AFTER:
nil
```
