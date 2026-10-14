import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

await runBrowserHarness(async (page, origin) => {
  const errors = [];
  const writes = [];
  let heldHistory;
  let delayHistory = false;
  page.on("pageerror", (error) => {
    errors.push(error.message);
    console.error(error.message);
  });
  const history = (job) => ({
    items: [
      {
        id: `${job}-pending`,
        job,
        status: "Pending",
        project_value: job === "jobA" ? 2000 : 3000,
        assigned_approver: "reviewer",
        version: 1,
        document_hash: `${job}-hash`,
      },
    ],
  });
  await page.route("**/api/**", async (route) => {
    const url = new URL(route.request().url());
    const send = (body) =>
      route.fulfill({ contentType: "application/json", body: JSON.stringify(body) });
    if (route.request().method() !== "GET") {
      writes.push(url.pathname);
      return send({});
    }
    if (url.pathname === "/api/job_attachments") {
      const job = url.searchParams.get("job");
      if (job === "jobA" && delayHistory) {
        heldHistory = () => send(history(job));
        return;
      }
      return send(history(job));
    }
    if (url.pathname === "/api/job_attachments/context")
      return send({
        job: {
          id: url.searchParams.get("job"),
          client: "client",
          description: "Project details",
          updated: "version1",
        },
        allocations: [],
        review_token: "reviewA",
      });
    if (url.pathname === "/api/job_requests/options")
      return send({
        clients: [],
        contacts: [],
        invoicing_profiles: [],
        users: [],
        branches: [],
        rate_sheets: [],
        divisions: [],
      });
    return send({ items: [], rows: [], totals: {}, filters: {}, totalItems: 0, totalPages: 1 });
  });
  const editor = page.getByRole("button", { name: "Save proposed value", exact: true });
  const correction = page.getByRole("button", {
    name: "Correct value change",
    exact: true,
  });
  const parent = page.locator("a[href='/jobs/jobB/details']");
  await page.goto(`${origin}/tests/fixtures/jobValueNavigation.html`);
  await correction.click();
  await editor.waitFor();
  // Navigate while the proposal modal is open, as a back or link navigation would.
  await page.evaluate(() => window.navigateTo("jobB"));
  await editor.waitFor({ state: "hidden" });
  await page.getByText("Description: Project B", { exact: true }).waitFor();
  assert.equal(await page.getByLabel("Proposed project value (CAD, before tax)").count(), 0);
  assert.deepEqual(writes, [], "navigation must not save the previous job's editor");
  await correction.click();
  assert.equal(
    await page.getByLabel("Proposed project value (CAD, before tax)").inputValue(),
    "3000",
  );

  delayHistory = true;
  await page.goto(`${origin}/tests/fixtures/jobValueNavigation.html`);
  await page.waitForFunction(() => document.querySelector("a[href='/jobs/jobB/details']"));
  for (let attempt = 0; attempt < 100 && !heldHistory; attempt++)
    await new Promise((resolve) => setTimeout(resolve, 50));
  assert.ok(heldHistory, "job A history request must be held before navigation");
  await page.getByText("Additional Details", { exact: true }).click();
  await parent.click();
  await correction.waitFor();
  await heldHistory();
  // Read after the old response has been processed, not just after it was sent.
  await page.waitForLoadState("networkidle");
  await correction.click();
  assert.equal(
    await page.getByLabel("Proposed project value (CAD, before tax)").inputValue(),
    "3000",
    "late job A history must not replace job B history",
  );
  assert.deepEqual(errors, []);
});
console.log("Job value navigation checks passed.");
