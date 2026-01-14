# Import Bug: Contact-Client Relationship Mismatch

**Date:** January 14, 2026  
**Status:** Fixed

## Problem Observed

The `syncToSQL` function in legacy Tybalt was failing with a foreign key constraint error when exporting `TurboClientContacts` to MySQL:

```
Error: Cannot add or update a child row: a foreign key constraint fails
```

Investigation revealed that several jobs had contacts whose parent client didn't match the job's client:

| Job | Job's Client | Contact's Client |
|-----|-------------|------------------|
| 22-063 | Thunder Bay Airport Authority | TBIAA |
| 23-609 | Northwest Company | North West Company |
| 24-418 | Critchley Hill Architecture | CHAI |
| P20-846 | Thunder Bay Airport Authority | TBIAA |
| P24-028 | Thunder Bay Airport Authority | TBIAA |
| P24-442 | Finnway | Finn Way General Contractor |
| P24-643 | Finnway | Finn Way General Contractor |

## Investigation Timeline

1. **Initial hypothesis:** Absorb operation might be corrupting relationships
2. **Checked absorb code:** ParentConstraint logic prevents absorbing contacts with different clients - code is correct
3. **Checked absorb_actions table:** Empty - no pending absorbs
4. **Examined contact records:** Contacts with `_imported = 1` were never modified in Turbo
5. **Checked legacy Firestore data:** Same contact name "TJ Ahvenniemi" appeared with 5 different client name variants
6. **Compared import audit:** Jobs_audit.parquet showed correct contact_id assignments, but production DB had wrong values
7. **Found the bug:** Import tool joins jobs to contacts by name only, not by (name, client)

## Root Cause

In `extract/jobs_to_clients_and_contacts.go`, the legacy import path (lines 346-348) joined jobs to contacts **by name only**:

```sql
UPDATE jobs SET contact_id = contacts.id 
FROM contacts 
WHERE jobs.t_clientContact = contacts.name;
```

When multiple contacts share the same name but belong to different clients (e.g., 5 "TJ Ahvenniemi" contacts for different Thunder Bay airport client variants), this UPDATE assigns all matching jobs to **whichever contact the database returns** - typically the same one regardless of which client the job actually belongs to.

### Example: TJ Ahvenniemi

The legacy Firestore had "TJ Ahvenniemi" as the contact for jobs with these different client name strings:
- TBIAA
- Thunder Bay Airport Authority  
- Thunder Bay International Airport Authority
- Thunder Bay International Airports Authority
- Thunder Bay Airport

The import created 5 separate TJ contacts (one per client variant). But the name-only join caused ALL jobs to get the same `contact_id` (TJ@TBIAA), regardless of their actual client.

Later, client absorb operations correctly consolidated the client variants into "Thunder Bay Airport Authority" and updated jobs' `client` fields. But the jobs' `contact` fields remained pointing to TJ@TBIAA, creating the mismatch.

## Fix Applied

Changed the join to include both contact name AND client:

```sql
UPDATE jobs SET contact_id = contacts.id 
FROM contacts 
WHERE jobs.t_clientContact = contacts.name 
  AND jobs.client_id = contacts.client_id;
```

This ensures each job gets the contact that belongs to the same client.

## Resolution

A fresh re-import from Firestore will create correct relationships. The 7 affected jobs in production will be fixed by the re-import.

## Lessons Learned

1. When entities have composite natural keys (name + parent), joins must include all key components
2. The absorb feature worked correctly - it couldn't fix pre-existing bad data from import
3. Test data may not expose this bug if it doesn't have duplicate names across different parents
