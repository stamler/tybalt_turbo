import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

await runBrowserHarness(async (page, origin) => {
  let items = ["A", "B", "C"].map((id) => ({
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
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/**", (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    if (path.endsWith("/approve")) {
      actions.push({ path, body: request.postDataJSON() });
      items = items.filter((item) => path !== `/api/job_attachments/${item.id}/approve`);
      // While A was approved, B was corrected with a replacement PDF. C is
      // unchanged.
      if (path === "/api/job_attachments/A/approve") {
        items = items.map((item) =>
          item.id === "B" ? { ...item, version: 2, document_hash: "hashB2" } : item,
        );
      }
      return route.fulfill({ contentType: "application/json", body: "{}" });
    }
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({ items: path === "/api/job_attachments" ? items : [] }),
    });
  });
  await page.goto(`${origin}/tests/fixtures/projectRequests.html?values&queue`);
  const card = (id) => page.locator("article").filter({ hasText: `Project ${id}` });
  const reviewed = (id) => card(id).getByRole("checkbox");
  const reason = (id) => card(id).getByLabel("Rejection reason", { exact: true });
  const approve = (id) => card(id).getByRole("button", { name: "Approve value", exact: true });

  await reviewed("A").check();
  await reviewed("B").check();
  await reason("B").fill("B is missing its signature.");
  await reviewed("C").check();
  await approve("A").click();
  await card("A").waitFor({ state: "hidden" });

  // B's revision changed, so its confirmation and reason no longer apply.
  assert.equal(await reviewed("B").isChecked(), false);
  assert.equal(await approve("B").isDisabled(), true);
  assert.equal(await reason("B").inputValue(), "");
  // C's revision did not, so its confirmation stays.
  assert.equal(await reviewed("C").isChecked(), true);

  await reviewed("B").check();
  await approve("B").click();
  await card("B").waitFor({ state: "hidden" });
  assert.deepEqual(
    actions.map((action) => [action.path, action.body.version, action.body.document_hash]),
    [
      ["/api/job_attachments/A/approve", 1, "hashA"],
      ["/api/job_attachments/B/approve", 2, "hashB2"],
    ],
  );
  assert.deepEqual(errors, []);
});
console.log("Value review checks passed: a new revision clears its confirmation and reason.");
