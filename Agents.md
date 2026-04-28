# AGENTS.md

## Repository expectations

## Testing

1. Write comprehensive testing around all new backend features (go code in the app/ directory)
2. Test at least both the happy path and failure scenarios, and any other specific cases that you think are necessary
3. Use fixtures whenever possible, in the csv files, for testing.
4. Update the datapackage.json if necessary
5. Don't seed new data within the test code
6. Never do a new testseed dump, always surgically update the csv files to provide the required data
7. Favour new data for new tests, (append-only) rather than editing existing data in place
8. If we have to modify fixture data at test time within test code it must be clearly documented

## Database Migrations

1. When creating a new database migration, always call `date +%s` to get the prefix for the new migration. This is because chronological order matters when migrations are applied.
2. Never edit a migration that's on the origin/main branch because these are already deployed and that would cause these migrations to not be applied or to have unpredictable outcomes.
3. You may edit a migration on any other branch besides main.
4. Migrations should always use the Pocketbase format when defining schema (adding columns, removing columns, changing column types, adding indices, adding collections etc.) This is because these changes are persisted to the _collections collection so full pocketbase schema dumps will reflect the changes.
5. SQL can be used in migrations to data modifications such as initially populating a new column or deriving its value from a different column because these aren't schema changes and are only ever applied one time.

## Git

1. If asked to write a commit message, never commit unless explicitly told to do so. Always show the commit message in markdown that is able to be copied (i.e. in a text fence so the user can copy it). Don't hard wrap lines in the commit message.
2. When you make changes to files, never stage your changes unless explicitly told to do so. This applies even if the files you changed were already staged.

## Review

1. When asked to review code, always always give each finding a distinct number so the user can unambiguously reference your findings
