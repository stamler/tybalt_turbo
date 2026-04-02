# Work Record Reconciliation

This change adds a read-only Work Records audit surface for reconciling time
entries against work records.

## Access

- The main nav shows `Work Records` under `Time Management`.
- The nav item is visible to holders of the `report` claim or the
  `work_record` claim.

## List Page

- Route: `/time/work-records`
- Data source: `GET /api/work_records`
- The page preloads all non-blank work records and searches them client-side.
- Search results are hidden until the user types at least 2 characters.
- The page uses fixed prefix tabs `K`, `Q`, and `F`, styled like the purchase
  order search toggles.
- Each result row shows:
  - the work record number
  - the count of time entries referencing that work record
- Search supports:
  - full values like `F34-142`
  - prefix-stripped forms like `34-142`
  - numeric-only fragments like `142`

Backend list query:

```sql
SELECT
  te.work_record AS work_record,
  SUBSTR(te.work_record, 1, 1) AS prefix,
  COUNT(*) AS entry_count
FROM time_entries te
WHERE te.work_record != ''
GROUP BY te.work_record
ORDER BY te.work_record ASC
```

The API also returns a derived `search_text` field used only for client-side
search matching.

## Detail Page

- Route: `/time/work-records/[workRecord]`
- Data source: `GET /api/work_records/{workRecord}`
- The page renders a sortable `ObjectTable`.
- Loaded fields include `id`, `uid`, `job_id`, and `timesheet_id`, but those
  columns are hidden in the UI.
- Visible columns are:
  - `Work Record`
  - `Week Ending`
  - `Employee`
  - `Hours`
  - `Job`
  - `Description`
- `Week Ending` links to `/time/sheets/{timesheet_id}/details` when a
  timesheet id is available.
- `Job` links to `/jobs/{job_id}/details` when a job id is available.

Backend detail query:

```sql
SELECT
  te.id AS id,
  te.work_record AS work_record,
  te.week_ending AS week_ending,
  te.uid AS uid,
  COALESCE(CAST(te.hours AS REAL), 0) AS hours,
  COALESCE(j.number, '') AS job_number,
  COALESCE(te.job, '') AS job_id,
  COALESCE(te.description, '') AS description,
  COALESCE(p.surname, '') AS surname,
  COALESCE(p.given_name, '') AS given_name,
  COALESCE(ts_exact.id, ts_fallback.id, '') AS timesheet_id
FROM time_entries te
LEFT JOIN jobs j
  ON te.job = j.id
LEFT JOIN profiles p
  ON te.uid = p.uid
LEFT JOIN time_sheets ts_exact
  ON ts_exact.id = te.tsid
LEFT JOIN time_sheets ts_fallback
  ON te.tsid = ''
 AND ts_fallback.uid = te.uid
 AND ts_fallback.week_ending = te.week_ending
WHERE te.work_record = {:work_record}
  AND te.work_record != ''
ORDER BY te.week_ending DESC
```

## Supporting Changes

- Added `GET /api/work_records`
- Added `GET /api/work_records/{workRecord}`
- Added an index on `time_entries.work_record`
- Extended `ObjectTable` to support:
  - display labels per column
  - per-column links while keeping sortable headers
- Added route tests covering:
  - `report` access
  - `work_record` access
  - unauthorized access
  - list counts and detail payload shape
