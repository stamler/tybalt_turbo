# Authorization upload fixtures

These PDFs are inputs for the project authorization upload tests. They are not
preloaded attachment records. Tests submit them through the upload API and check
the saved file bytes and SHA-256 hash. The replacement has different content.

The seeded attachment and its CSV hash remain under
`files/job_attachments/paattachment001/authorization.pdf` and
`data/job_attachments.csv`.

Missing-file tests remove only the isolated test app's stored copy. They leave
these source files and the recorded hash unchanged, then check that submission,
approval, and job creation fail without changing workflow state. Matching
re-uploads restore missing objects without changing approval or rejection data.
