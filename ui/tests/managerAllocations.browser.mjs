import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

await runBrowserHarness(async (page, origin) => {
  const errors = [];
  const saves = [];
  let failSave = false;
  let failLoad = false;
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/jobs/*/details", (route) =>
    route.fulfill({
      contentType: "text/html",
      body: "<p>Job details</p>",
    }),
  );
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    if (request.method() === "PUT") {
      assert.match(new URL(request.url()).pathname, /^\/api\/jobs\/[^/]+$/);
      saves.push(request.postDataJSON());
      return route.fulfill({
        status: failSave ? 403 : 200,
        contentType: "application/json",
        body: JSON.stringify(
          failSave ? { message: "Your allocation editing permission has changed." } : { id: "job" },
        ),
      });
    }
    assert.equal(request.method(), "GET", "allocation saves must not change categories");
    const pathname = new URL(request.url()).pathname;
    if (failLoad && pathname.includes("job_time_allocations")) {
      return route.fulfill({ status: 500, contentType: "application/json", body: "{}" });
    }
    const items = pathname.includes("job_time_allocations")
      ? [{ id: "allocation", division: "fy4i9poneukvq9u", hours: 10 }]
      : pathname.includes("/divisions/")
        ? [
            { id: "fy4i9poneukvq9u", code: "MD", name: "Drilling Services", active: true },
            { id: "vccd5fo56ctbigh", code: "CI", name: "Information Technology", active: true },
          ]
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

  for (const role of ["manager", "alternate_manager"]) {
    await page.goto(`${origin}/tests/fixtures/managerAllocations.html?role=${role}`);
    await page.getByRole("heading", { name: /Edit allocations for/ }).waitFor();
    const firstHours = page.getByRole("spinbutton", { name: "Hours for division 1" });
    await firstHours.waitFor();
    assert.equal(await firstHours.inputValue(), "10");
    for (const name of ["description", "project_value", "status", "manager", "branch"]) {
      assert.equal(
        await page.locator(`[name="${name}"]`).count(),
        0,
        `${name} must not be editable`,
      );
    }
    assert.equal(await page.getByText("Categories", { exact: true }).count(), 0);
    await firstHours.fill("12");
    await page.getByRole("button", { name: "Add division" }).click();
    await page.getByPlaceholder("Division", { exact: true }).fill("Information");
    await page.getByText("CI - Information Technology", { exact: true }).click();
    await page.getByRole("spinbutton", { name: "Hours for division 2" }).fill("7");
    await page.getByTitle("Remove", { exact: true }).first().click();
    await page.getByRole("button", { name: "Save", exact: true }).click();
    await page.waitForURL("**/jobs/*/details");
    assert.deepEqual(saves.at(-1), { allocations: [{ division: "vccd5fo56ctbigh", hours: 7 }] });
  }

  await page.goto(`${origin}/tests/fixtures/managerAllocations.html`);
  await page.getByRole("spinbutton", { name: "Hours for division 1" }).waitFor();
  await page.getByTitle("Remove", { exact: true }).click();
  const savedCount = saves.length;
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByText("At least one division allocation is required.", { exact: true }).waitFor();
  assert.equal(saves.length, savedCount);

  failSave = true;
  await page.goto(`${origin}/tests/fixtures/managerAllocations.html`);
  await page.getByRole("spinbutton", { name: "Hours for division 1" }).waitFor();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page
    .getByText("Your allocation editing permission has changed.", { exact: true })
    .waitFor();
  assert.match(page.url(), /managerAllocations.html/);
  assert.deepEqual(saves.at(-1), { allocations: [{ division: "fy4i9poneukvq9u", hours: 10 }] });

  failLoad = true;
  await page.goto(`${origin}/tests/fixtures/managerAllocations.html`);
  await page.getByRole("button", { name: "Add division" }).click();
  await page.getByPlaceholder("Division", { exact: true }).fill("Information");
  await page.getByText("CI - Information Technology", { exact: true }).click();
  const savesBeforeLoadFailure = saves.length;
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page
    .getByText("Allocations have not loaded. Reload the page before saving.", { exact: true })
    .waitFor();
  assert.equal(saves.length, savesBeforeLoadFailure);

  await page.goto(`${origin}/tests/fixtures/managerAllocations.html?role=unrelated`);
  await page
    .getByText("The job permission is required to edit or create a job.", { exact: true })
    .waitFor();
  assert.equal(await page.getByRole("button", { name: "Save", exact: true }).count(), 0);
  assert.deepEqual(errors, []);
});
console.log("Manager allocation browser checks passed.");
