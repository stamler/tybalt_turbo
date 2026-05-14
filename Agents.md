# AGENTS.md

## Repository expectations

## Test fixtures

1. Use fixtures whenever possible, in the csv files, for testing.
2. Update the datapackage.json if necessary
3. Don't seed new data within the test code
4. Never do a new testseed dump, always surgically update the csv files to provide the required data
5. Favour new data for new tests, (append-only) rather than editing existing data in place
6. If we have to modify fixture data at test time within test code it must be clearly documented

## Database Migrations

1. When creating a new database migration, always call `date +%s` to get the prefix for the new migration. This is because chronological order matters when migrations are applied.
2. Never edit a migration that's on the origin/main branch because these are already deployed and that would cause these migrations to not be applied or to have unpredictable outcomes.
3. You may edit a migration on any other branch besides main.
4. Migrations should always use the Pocketbase format when defining schema (adding columns, removing columns, changing column types, adding indices, adding collections etc.) This is because these changes are persisted to the _collections collection so full pocketbase schema dumps will reflect the changes.
5. SQL can be used in migrations to data modifications such as initially populating a new column or deriving its value from a different column because these aren't schema changes and are only ever applied one time.
