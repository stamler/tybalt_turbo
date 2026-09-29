import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";
const fixture = JSON.parse(
  await readFile(new URL("./fixtures/wipReports.json", import.meta.url), "utf8"),
);

await runBrowserHarness(async (page, origin) => {
  const errors = [];
  const requests = [];
  let failNext = false;
  let empty = false;
  let slowSouth = false;
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/wip/**", async (route) => {
    const url = new URL(route.request().url());
    requests.push(url);
    if (failNext) {
      failNext = false;
      return route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"message":"Test failure"}',
      });
    }
    let items = empty ? [] : fixture.items;
    const branch = url.searchParams.get("branch");
    if (branch) items = items.filter((row) => row.branch_id === branch);
    if (url.pathname.endsWith("division") && url.searchParams.get("require_time") !== "false")
      items = items.filter((row) => row.hours > 0);
    if (slowSouth && branch === "south") await new Promise((resolve) => setTimeout(resolve, 400));
    await route
      .fulfill({ contentType: "application/json", body: JSON.stringify({ ...fixture, items }) })
      .catch(() => {});
  });
  await page.goto(`${origin}/tests/fixtures/wipReports.html`);
  await page.getByTestId("report-normal").waitFor();
  assert.equal(
    await page.getByRole("navigation", { name: "WIP navigation" }).getByRole("link").count(),
    3,
  );
  const ranked = page.getByRole("table", { name: "Projects: ranked WIP", exact: true });
  assert.deepEqual(
    await ranked.locator("tbody tr").evaluateAll((rows) => rows.map((row) => row.dataset.testid)),
    ["report-over", "report-normal", "report-empty"],
  );
  assert.match(await page.getByTestId("report-over").innerText(), /130\.0%/);
  assert.match(await page.getByTestId("report-normal").innerText(), /Estimated/);
  assert.equal(
    await page.getByRole("link", { name: "26-001", exact: true }).getAttribute("href"),
    "/jobs/normal/details",
  );
  assert.match(await page.locator("main").innerText(), /2 active projects excluded/);
  const before = requests.length;
  await page.getByLabel("Include active POs", { exact: true }).uncheck();
  await ranked.getByTestId("report-unknown-po").waitFor();
  assert.match(await ranked.getByTestId("report-unknown-po").innerText(), /50\.0%/);
  await page.getByLabel("Include committed expenses", { exact: true }).uncheck();
  await ranked.getByTestId("report-unknown-expense").waitFor();
  assert.equal(requests.length, before);
  assert.match(await page.getByTestId("report-normal").innerText(), /15,000\.00/);
  await page.getByLabel("Include committed expenses", { exact: true }).check();
  await page.getByLabel("Include active POs", { exact: true }).check();
  const help = page
    .getByTestId("report-partial")
    .getByRole("button", { name: "Partial: Time value: value details", exact: true });
  await help.focus();
  await page.keyboard.press("Enter");
  await page.getByRole("dialog", { name: "Time value: value details", exact: true }).waitFor();
  assert.match(
    await page.getByRole("dialog", { name: "Time value: value details", exact: true }).innerText(),
    /cannot be priced/,
  );
  await page.keyboard.press("Escape");
  assert.equal(await help.evaluate((el) => el === document.activeElement), true);
  await page
    .getByTestId("report-partial")
    .getByRole("button", { name: /No rate sheet/ })
    .click();
  await page.getByRole("dialog", { name: "Time uses employee defaults" }).waitFor();
  await page.keyboard.press("Escape");

  for (const access of ["ordinary", "manager", "admin", "kpi"]) {
    await page.getByLabel("Access", { exact: true }).selectOption(access);
    assert.equal(
      await page.getByRole("navigation", { name: "WIP navigation" }).getByRole("link").count(),
      access === "ordinary" ? 1 : 3,
    );
  }
  await page.getByLabel("Report", { exact: true }).selectOption("branch");
  await page.getByRole("table", { name: "North: ranked WIP", exact: true }).waitFor();
  assert.equal(requests.at(-1).searchParams.get("branch"), "north");
  await page.getByLabel("Branch", { exact: true }).selectOption("");
  await page.getByRole("heading", { name: "South", exact: true }).waitFor();
  await page.getByRole("heading", { name: "North", exact: true }).waitFor();
  await page.getByLabel("Access", { exact: true }).selectOption("ordinary");
  await page.getByRole("alert").waitFor();
  assert.equal(await page.locator("table").count(), 0);
  await page.getByLabel("Access", { exact: true }).selectOption("manager");
  await page.getByTestId("report-over").waitFor();

  failNext = true;
  await page.getByLabel("Branch", { exact: true }).selectOption("south");
  await page.getByRole("alert").waitFor();
  await page.getByRole("button", { name: "Try again", exact: true }).click();
  await page.getByTestId("report-over").waitFor();
  await page.getByLabel("Branch", { exact: true }).selectOption("north");
  await page.getByTestId("report-normal").waitFor();
  slowSouth = true;
  await page.getByLabel("Branch", { exact: true }).selectOption("south");
  await page.getByRole("status").waitFor();
  await page.getByLabel("Branch", { exact: true }).selectOption("north");
  await page.getByTestId("report-normal").waitFor();
  await page.waitForTimeout(500);
  assert.equal(await page.getByTestId("report-over").count(), 0);

  await page.getByLabel("Report", { exact: true }).selectOption("division");
  await page.getByTestId("report-normal").waitFor();
  assert.equal(requests.at(-1).searchParams.get("division"), "civil");
  assert.equal(requests.at(-1).searchParams.get("require_time"), "true");
  assert.equal(await page.getByTestId("report-empty").count(), 0);
  await page.getByLabel("Only jobs with recorded work hours in this division").uncheck();
  await page.getByTestId("report-empty").waitFor();
  assert.equal(requests.at(-1).searchParams.get("require_time"), "false");
  await page.getByLabel("Default division", { exact: true }).selectOption("");
  await page.getByText("Select a division to view its projects.", { exact: true }).waitFor();
  const division = page.getByLabel("Division", { exact: true });
  await division.fill("STR");
  await division.press("ArrowDown");
  await division.press("Enter");
  await page.getByTestId("report-normal").waitFor();
  assert.equal(requests.at(-1).searchParams.get("division"), "structural");
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-desktop.png`,
      fullPage: true,
    });
  await page.setViewportSize({ width: 360, height: 800 });
  assert.equal(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    true,
  );
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-mobile.png`,
      fullPage: true,
    });
  empty = true;
  await page.getByLabel("Report", { exact: true }).selectOption("my");
  await page
    .getByText(
      "No projects to show. You are not the manager of any active projects with a project value greater than zero.",
      { exact: true },
    )
    .waitFor();
  assert.deepEqual(errors, []);
  console.log(
    "WIP report browser checks passed: ranking, controls, warnings, access, filters, defaults, retry, stale responses, empty states, and mobile layout.",
  );
});
