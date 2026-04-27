# AGENTS.md

## Repository expectations

## Testing

- write comprehensive testing around all new features
- test at least both the happy path and failure scenarios
- use fixtures whenever possible, in the csv files for testing
- update the datapackage.json if necessary
- don't seed new data within the test code
- don't do a new testseed dump, always surgically update the csv files to provide the required data
- favour new data for new tests, (append-only) rather than editing existing data in place
- if we have to modify fixture data at test time within test code it must be clearly documented
