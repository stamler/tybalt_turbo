import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

await runBrowserHarness(async (page, origin) => {
  const errors = [];
  const saves = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    if (request.method() === "PUT") {
      saves.push(request.postDataJSON());
      return route.fulfill({
        status: 400,
        contentType: "application/json",
        body: JSON.stringify({
          message: "Invalid completion date",
          data: { project_completion_date: { message: "Enter a valid completion date." } },
        }),
      });
    }
    const items = request.url().includes("job_time_allocations")
      ? [{ id: "allocation", division: "fy4i9poneukvq9u", hours: 10 }]
      : [];
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        page: 1,
        perPage: 500,
        totalPages: 1,
        totalItems: items.length,
        items,
      }),
    });
  });
  await page.goto(`${origin}/tests/fixtures/projectCompletionDate.html`);
  const date = page.locator('input[name="project_completion_date"]');
  const award = page.locator('input[name="project_award_date"]');
  const status = page.locator('select[name="status"]');
  const value = page.locator('input[name="project_value"]');
  await date.waitFor();
  assert.equal(await date.getAttribute("required"), "");
  assert.equal(await date.evaluate((input) => input.validity.valueMissing), true);
  assert.equal(await date.getAttribute("max"), null);
  assert.equal(await date.getAttribute("min"), "2025-01-15");
  assert.match(
    await page.locator('label[for="project_completion_date"]').getAttribute("title"),
    /estimate is acceptable/,
  );

  for (const value of ["Closed", "Cancelled"]) {
    await status.selectOption(value);
    assert.equal(await date.getAttribute("required"), null);
    assert.equal(await date.evaluate((input) => input.checkValidity()), true);
  }
  for (const state of ["Active", "Closed"]) {
    await status.selectOption(state);
    await value.fill("0");
    assert.equal(await value.evaluate((input) => input.validity.rangeUnderflow), true);
    await value.fill("");
    assert.equal(await value.evaluate((input) => input.validity.valueMissing), true);
    await value.fill("1");
    assert.equal(await value.evaluate((input) => input.checkValidity()), true);
  }
  await status.selectOption("Cancelled");
  await value.fill("0");
  assert.equal(await value.evaluate((input) => input.checkValidity()), true);
  await value.fill("100000");
  await status.selectOption("Active");
  assert.equal(await date.evaluate((input) => input.validity.valueMissing), true);
  await date.fill("2035-12-31");
  assert.equal(await date.evaluate((input) => input.checkValidity()), true);
  await page.getByTitle("Clear completion date", { exact: true }).click();
  assert.equal(await date.inputValue(), "");
  await date.fill("2025-01-14");
  assert.equal(await date.evaluate((input) => input.validity.rangeUnderflow), true);
  await date.fill("2025-01-15");
  assert.equal(await date.evaluate((input) => input.checkValidity()), true);
  await award.fill("2025-01-16");
  assert.equal(await date.getAttribute("min"), "2025-01-16");
  assert.equal(await date.evaluate((input) => input.validity.rangeUnderflow), true);
  await award.fill("");
  assert.equal(await date.getAttribute("min"), null);
  assert.equal(await date.evaluate((input) => input.checkValidity()), true);
  await award.fill("2025-01-15");
  await date.fill("2025-02-01");
  assert.equal(await date.evaluate((input) => input.checkValidity()), true);

  await page.locator('form[enctype="multipart/form-data"]').evaluate((form) => form.requestSubmit());
  await page.getByText("Enter a valid completion date.", { exact: true }).waitFor();
  assert.equal(saves.length, 1);
  assert.equal(saves[0].job.project_completion_date, "2025-02-01");

  await page.goto(`${origin}/tests/fixtures/projectCompletionDate.html?proposal`);
  await page.locator('input[name="proposal_opening_date"]').waitFor();
  assert.equal(await page.locator('input[name="project_completion_date"]').count(), 0);
  assert.deepEqual(errors, []);
});
console.log("Project completion date browser checks passed.");
