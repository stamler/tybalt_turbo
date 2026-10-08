import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

await runBrowserHarness(async (page, origin) => {
  let items = ["A", "B"].map((id) => ({
    id,
    job: `job${id}`,
    job_number: `Project ${id}`,
    status: "Pending",
    project_value: 10000,
    assigned_approver: "reviewer",
    version: 1,
    document_hash: `hash${id}`,
    current_review_token: `token${id}`,
  }));
  const actions = [];
  const errors = [];
  let failRejection = true;
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/**", (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path.endsWith("/reject")) {
      actions.push({ path, body: request.postDataJSON() });
      if (failRejection) {
        failRejection = false;
        return route.fulfill({
          status: 400,
          contentType: "application/json",
          body: JSON.stringify({ message: "Rejection failed. Try again." }),
        });
      }
      items = items.filter((item) => path !== `/api/job_attachments/${item.id}/reject`);
      return route.fulfill({ contentType: "application/json", body: "{}" });
    }
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ items: path === "/api/job_attachments" ? items : [] }),
    });
  });
  await page.goto(`${origin}/tests/fixtures/projectRequests.html?values&queue`);
  const card = (id) => page.locator("article").filter({ hasText: `Project ${id}` });
  const reason = (id) => card(id).getByLabel("Rejection reason", { exact: true });
  const reject = (id) => card(id).getByRole("button", { name: "Reject", exact: true });
  await reason("A").fill("A needs the signed appendix.");
  assert.equal(await reason("B").inputValue(), "");
  assert.equal(await reject("B").isDisabled(), true);
  await reason("B").fill("B has the wrong value.");
  assert.equal(await reason("A").inputValue(), "A needs the signed appendix.");

  await reject("A").click();
  await page.getByRole("alert").filter({ hasText: "Rejection failed. Try again." }).waitFor();
  assert.equal(await reason("A").inputValue(), "A needs the signed appendix.");
  assert.equal(await reason("B").inputValue(), "B has the wrong value.");

  await reject("A").click();
  await card("A").waitFor({ state: "hidden" });
  assert.equal(await reason("B").inputValue(), "B has the wrong value.");
  await reject("B").click();
  await card("B").waitFor({ state: "hidden" });
  assert.deepEqual(
    actions,
    ["A", "A", "B"].map((id) => ({
      path: `/api/job_attachments/${id}/reject`,
      body: {
        version: 1,
        document_hash: `hash${id}`,
        review_token: `token${id}`,
        rejection_reason: id === "A" ? "A needs the signed appendix." : "B has the wrong value.",
      },
    })),
  );
  assert.deepEqual(errors, []);
});
console.log("Value rejection checks passed: separate reasons, failed retry, and queue refresh.");
