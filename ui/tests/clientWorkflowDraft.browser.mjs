import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";
const fixture = JSON.parse(
  await readFile(new URL("./fixtures/clientWorkflowDraft.json", import.meta.url), "utf8"),
);
await runBrowserHarness(async (page, origin) => {
  page.setDefaultTimeout(15000);
  const errors = [],
    writes = [];
  let changedJob = false,
    changedAllocations = false;
  page.on("pageerror", (e) => errors.push(e.message));
  await page.route("**/api/**", async (route) => {
    const req = route.request(),
      url = new URL(req.url());
    const send = (body, status = 200) =>
      route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
    if (req.method() !== "GET") {
      writes.push(req.postData());
      return send({ message: "Keep the form open for the test." }, 400);
    }
    if (url.pathname === "/api/job_attachments/context")
      return send({
        job: { ...fixture.job, updated: changedJob ? "job-version-2" : fixture.job.updated },
        allocations: changedAllocations
          ? [{ division: "division1", hours: 10 }]
          : fixture.allocations,
        review_token: "new-related-token",
      });
    if (url.pathname === "/api/job_requests/options") return send(fixture.options);
    if (url.pathname === "/api/project_authorization/approvers")
      return send({ items: [{ id: "reviewer", name: "Reviewer", max_value: 99999 }] });
    return send({ items: [fixture.editing] });
  });
  await page.goto(`${origin}/tests/fixtures/clientWorkflowDraft.html`);
  const save = page.getByRole("button", { name: "Save proposed value", exact: true });
  await save.waitFor();
  await page.getByText("Selected PDF: kept-authorization.pdf", { exact: true }).waitFor();
  await page.getByText("Project and billing details", { exact: true }).click();
  await page.getByLabel("Project / scope of work").waitFor();
  assert.equal(
    await page.getByLabel("Project / scope of work").inputValue(),
    "Unsaved corrected scope",
  );
  assert.equal(await page.getByLabel("Hours", { exact: true }).inputValue(), "42");
  assert.equal(await page.getByLabel("Hours", { exact: true }).getAttribute("step"), "1");
  assert.equal(
    await page.getByLabel("Proposed project value (CAD, before tax)").inputValue(),
    "2400",
  );
  assert.match(
    await page.getByLabel("Invoicing profile", { exact: true }).innerText(),
    /Updated invoicing profile/,
  );
  await save.click();
  await page.getByText("Keep the form open for the test.", { exact: true }).waitFor();
  assert.equal(writes.length, 1);
  assert.match(writes[0], /new-related-token/);
  assert.doesNotMatch(writes[0], /old-related-token/);
  assert.match(writes[0], /Unsaved corrected scope/);
  assert.match(writes[0], /kept-authorization.pdf/);
  assert.match(writes[0], /name="version"\r\n\r\n7/);

  changedJob = true;
  await page.goto(`${origin}/tests/fixtures/clientWorkflowDraft.html`);
  await page.getByRole("alert").filter({ hasText: "The job or its allocations changed" }).waitFor();
  assert.equal(await save.isDisabled(), true);
  changedJob = false;
  changedAllocations = true;
  await page.goto(`${origin}/tests/fixtures/clientWorkflowDraft.html`);
  await page.getByRole("alert").filter({ hasText: "The job or its allocations changed" }).waitFor();
  assert.equal(await save.isDisabled(), true);
  changedAllocations = false;
  await page.goto(`${origin}/tests/fixtures/clientWorkflowDraft.html?reloaded`);
  await page.getByRole("alert").filter({ hasText: "Select the PDF again" }).waitFor();
  assert.equal(
    await page.getByText("Selected PDF: kept-authorization.pdf", { exact: true }).count(),
    0,
  );
  assert.deepEqual(errors, []);
});
console.log(
  "Client workflow return: draft/PDF preserved, current shared data reviewed, concurrent job/allocation changes blocked.",
);
