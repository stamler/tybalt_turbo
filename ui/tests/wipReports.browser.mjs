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
  const missingRateRequests = [];
  let failMissingRates = true;
  let emptyMissingRates = false;
  let releaseMissingRates;
  let waitForMissingRates = Promise.resolve();
  let empty = false;
  let employeeOnly = false;
  let percentageExamples = false;
  let remainingExamples = false;
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
    let items = empty
      ? []
      : remainingExamples
        ? fixture.remaining_examples
        : percentageExamples
          ? fixture.percentage_examples
          : employeeOnly
            ? fixture.employee_rates
            : fixture.items;
    const branch = url.searchParams.get("branch");
    if (branch) items = items.filter((row) => row.branch_id === branch);
    if (url.pathname.endsWith("division") && url.searchParams.get("require_time") !== "false")
      items = items.filter((row) => row.hours > 0);
    if (slowSouth && branch === "south") await new Promise((resolve) => setTimeout(resolve, 400));
    await route
      .fulfill({ contentType: "application/json", body: JSON.stringify({ ...fixture, items }) })
      .catch(() => {});
  });
  await page.route("**/api/jobs/*/wip/missing-rates?*", async (route) => {
    missingRateRequests.push(new URL(route.request().url()));
    if (failMissingRates) {
      failMissingRates = false;
      return route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"message":"Test failure"}',
      });
    }
    await waitForMissingRates;
    await route
      .fulfill({
        contentType: "application/json",
        body: JSON.stringify(
          emptyMissingRates ? { ...fixture.missing_rates, roles: [] } : fixture.missing_rates,
        ),
      })
      .catch(() => {});
  });
  await page.goto(`${origin}/tests/fixtures/wipReports.html`);
  await page.getByTestId("report-normal").waitFor();
  const wipButton = page.getByRole("button", { name: "WIP", exact: true });
  const wipMenu = page.getByRole("navigation", { name: "WIP views", exact: true });
  assert.equal(await wipButton.getAttribute("aria-expanded"), "false");
  assert.equal(await wipMenu.isVisible(), false);
  await wipButton.click();
  await wipMenu.waitFor();
  assert.deepEqual(await wipMenu.getByRole("link").allTextContents(), [
    "My WIP",
    "Branch",
    "Division",
  ]);
  assert.equal(
    await wipMenu.getByRole("link", { name: "My WIP", exact: true }).getAttribute("aria-current"),
    "page",
  );
  const buttonBox = await wipButton.boundingBox();
  const menuBox = await wipMenu.boundingBox();
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({ path: `${process.env.WIP_REPORT_SCREENSHOT}-menu-desktop.png` });
  assert.ok(
    menuBox.x > buttonBox.x + buttonBox.width,
    "The desktop menu opens to the right, outside sidebar overflow.",
  );
  await page.keyboard.press("ArrowDown");
  assert.equal(await page.locator(":focus").innerText(), "Branch");
  await page.keyboard.press("End");
  assert.equal(await page.locator(":focus").innerText(), "Division");
  await page.keyboard.press("Home");
  assert.equal(await page.locator(":focus").innerText(), "My WIP");
  await page.keyboard.press("Escape");
  assert.equal(await wipMenu.isVisible(), false);
  assert.equal(await wipButton.evaluate((el) => el === document.activeElement), true);
  await page.keyboard.press("ArrowRight");
  await wipMenu.waitFor();
  await page.keyboard.press("ArrowLeft");
  assert.equal(await wipMenu.isVisible(), false);
  await wipButton.click();
  await wipMenu.waitFor();
  await wipButton.click();
  assert.equal(await wipMenu.isVisible(), false, "Clicking the trigger again closes the menu.");
  await wipButton.click();
  await wipMenu.waitFor();
  await page.getByRole("combobox", { name: "Report", exact: true }).click();
  assert.equal(await wipMenu.isVisible(), false, "An outside click closes the menu.");
  await page.keyboard.press("Escape");
  const ranked = page.getByRole("table", { name: "Projects: ranked WIP", exact: true });
  assert.deepEqual(
    await ranked.locator("tbody tr").evaluateAll((rows) => rows.map((row) => row.dataset.testid)),
    ["report-over", "report-normal", "report-empty"],
  );
  assert.match(await page.getByTestId("report-over").innerText(), /130\.0%/);
  assert.equal(await ranked.getByRole("columnheader").last().innerText(), "Remaining");
  const remaining = (id) => page.getByTestId(`report-${id}`).locator("td").last();
  assert.equal(await remaining("over").innerText(), "-$15,000.00");
  assert.equal(
    await remaining("over").evaluate((el) => el.classList.contains("text-red-700")),
    true,
  );
  assert.equal(await remaining("normal").innerText(), "$35,000.00");
  assert.equal(
    await remaining("normal").evaluate((el) => el.classList.contains("text-red-700")),
    false,
  );
  assert.equal(await remaining("partial").innerText(), "—");
  const sort = page.getByRole("combobox", { name: "Sort by", exact: true });
  const beforeSort = requests.length;
  await sort.selectOption("remaining");
  assert.deepEqual(
    await ranked.locator("tbody tr").evaluateAll((rows) => rows.map((row) => row.dataset.testid)),
    ["report-over", "report-normal", "report-empty"],
  );
  assert.equal(requests.length, beforeSort, "sorting stays client-side");
  await sort.selectOption("percent");
  assert.deepEqual(
    await ranked.locator("tbody tr").evaluateAll((rows) => rows.map((row) => row.dataset.testid)),
    ["report-over", "report-normal", "report-empty"],
  );
  assert.equal(
    await page.getByTestId("report-partial").getByTestId("wip-percentage-fill").count(),
    0,
  );
  await page
    .getByTestId("report-normal")
    .getByRole("button", { name: "Estimated: Time value: value details", exact: true })
    .waitFor();
  const estimate = page
    .getByTestId("report-normal")
    .getByRole("button", { name: "Estimated: Time value: value details", exact: true });
  assert.equal(await estimate.innerText(), "≈");
  const estimateBounds = await estimate.boundingBox();
  const estimatedAmountBounds = await page
    .getByTestId("report-normal")
    .getByRole("cell")
    .nth(2)
    .getByText("$40,000.00", { exact: true })
    .boundingBox();
  assert.ok(
    estimateBounds.x + estimateBounds.width <= estimatedAmountBounds.x,
    "estimate precedes the time amount",
  );
  await estimate.click();
  const timeDetails = page.getByRole("dialog", { name: "Time value: value details", exact: true });
  await timeDetails.waitFor();
  const explanation = await timeDetails.innerText();
  assert.match(explanation, /60.00 hours have no role recorded/);
  assert.match(explanation, /40.00 hours have a role with no matching rate/);
  assert.match(explanation, /100.00 hours use employee default/);
  assert.doesNotMatch(explanation, /cannot be priced/);
  assert.equal(missingRateRequests.length, 0, "opening the popup must not fetch role details");
  const missingRates = timeDetails.getByRole("button", { name: "missing rates", exact: true });
  await missingRates.click();
  await timeDetails.getByRole("alert").waitFor();
  assert.equal(missingRateRequests.length, 1);
  assert.equal(missingRateRequests[0].pathname, "/api/jobs/normal/wip/missing-rates");
  assert.equal(missingRateRequests[0].searchParams.get("as_of"), fixture.items[0].as_of);
  waitForMissingRates = new Promise((resolve) => {
    releaseMissingRates = resolve;
  });
  await timeDetails.getByRole("button", { name: "Try again", exact: true }).click();
  await timeDetails.getByRole("status").waitFor();
  assert.equal(await missingRates.isDisabled(), true);
  const initialHeight = (await timeDetails.boundingBox()).height;
  releaseMissingRates();
  const rateSheetLink = timeDetails.getByRole("link", {
    name: "Example Standard Rates",
    exact: true,
  });
  await rateSheetLink.waitFor();
  assert.equal(await rateSheetLink.getAttribute("href"), "/rate-sheets/example-sheet/details");
  assert.match(
    await rateSheetLink.locator("..").innerText(),
    /40.00 hours have a role with no matching rate in the\s+Example Standard Rates\s+\(revision 2\)\s+rate sheet\./,
  );
  assert.equal(await missingRates.count(), 0, "loaded details replace the missing rates button");
  assert.doesNotMatch(await timeDetails.innerText(), /Rate sheet:|assigned rate sheet/);
  assert.match(await timeDetails.innerText(), /Intermediate Designer — 30.00 hours/);
  assert.match(await timeDetails.innerText(), /Summer Student — 10.00 hours/);
  assert.match(await timeDetails.innerText(), /revision 2/);
  assert.ok(
    (await timeDetails.boundingBox()).height > initialHeight,
    "popup expands for role details",
  );
  await page.waitForTimeout(50);
  const expandedBounds = await timeDetails.boundingBox();
  assert.ok(
    expandedBounds.y >= 0 && expandedBounds.y + expandedBounds.height <= 900,
    "expanded popup stays in the viewport",
  );
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-expanded.png`,
      fullPage: true,
    });
  await page.keyboard.press("Escape");
  await estimate.click();
  await rateSheetLink.waitFor();
  assert.equal(missingRateRequests.length, 2, "reopening the popup reuses its loaded details");
  await page.keyboard.press("Escape");
  assert.equal(
    await page
      .getByTestId("report-over")
      .getByRole("button", { name: /Time value: value details/ })
      .count(),
    0,
  );
  assert.equal(
    await page
      .getByTestId("report-empty")
      .getByRole("button", { name: /Time value: value details/ })
      .count(),
    0,
  );
  assert.equal(
    await page
      .getByTestId("report-partial")
      .getByRole("button", { name: /No rate sheet/ })
      .innerText(),
    "NR",
  );
  assert.equal(
    await page.getByRole("link", { name: "26-001", exact: true }).getAttribute("href"),
    "/jobs/normal/details",
  );
  assert.match(await page.locator("main").innerText(), /2 without project value/);
  assert.ok(
    await page
      .getByTestId("report-normal")
      .evaluate((row) => row.getBoundingClientRect().height <= 38),
    "rows fit on one line",
  );
  const detailButton = page.getByRole("button", { name: "Project details: 26-001", exact: true });
  assert.equal(
    await page.getByText("Bridge assessment", { exact: true }).count(),
    0,
    "closed project popups do not build their contents before first use",
  );
  await detailButton.focus();
  await page.keyboard.press("Enter");
  const details = page.getByRole("dialog", { name: "Project details: 26-001", exact: true });
  await details.waitFor();
  assert.match(await details.innerText(), /Bridge assessment/);
  assert.match(await details.innerText(), /Example Client/);
  assert.match(await details.innerText(), /Project Manager/);
  assert.match(await details.innerText(), /North/);
  await page.keyboard.press("Escape");
  assert.equal(await detailButton.evaluate((el) => el === document.activeElement), true);
  const search = page.getByRole("searchbox", { name: "Search projects" });
  const beforeSearch = requests.length;
  for (const [term, count] of [
    ["bRiDgE", 1],
    ["26-003", 1],
    ["example client", 6],
    ["project manager", 6],
    ["north", 4],
    ["north building", 1],
  ]) {
    await search.fill(term);
    assert.equal(await page.locator("tbody tr").count(), count);
  }
  await search.fill("no matching project");
  await page.getByText("No projects match your search.", { exact: true }).waitFor();
  assert.equal(await page.locator("table").count(), 0);
  await search.fill("");
  await page.getByTestId("report-normal").waitFor();
  assert.equal(requests.length, beforeSearch, "search must stay client-side");
  const before = requests.length;
  await page.getByLabel("Include active POs", { exact: true }).uncheck();
  assert.equal(await remaining("normal").innerText(), "$50,000.00");
  assert.equal(await remaining("over").innerText(), "$0.00");
  assert.equal(
    await remaining("over").evaluate((el) => el.classList.contains("text-red-700")),
    false,
  );
  await ranked.getByTestId("report-unknown-po").waitFor();
  assert.match(await ranked.getByTestId("report-unknown-po").innerText(), /50\.0%/);
  await page.getByLabel("Include committed expenses", { exact: true }).uncheck();
  await ranked.getByTestId("report-unknown-expense").waitFor();
  assert.equal(requests.length, before);
  assert.equal(await page.getByRole("columnheader", { name: "Expenses", exact: true }).count(), 0);
  assert.equal(await page.getByRole("columnheader", { name: "POs", exact: true }).count(), 0);
  assert.equal(await remaining("normal").innerText(), "$60,000.00");
  assert.equal(await page.getByTestId("report-normal").locator("td").count(), 5);
  await page.getByLabel("Include committed expenses", { exact: true }).check();
  await page.getByLabel("Include active POs", { exact: true }).check();
  assert.match(await page.getByTestId("report-normal").innerText(), /15,000\.00/);
  assert.equal(await ranked.getByRole("columnheader").count(), 8);
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
    for (const showAll of [false, true]) {
      await page.getByLabel("Show All UI", { exact: true }).setChecked(showAll);
      await wipButton.click();
      await wipMenu.waitFor();
      assert.equal(await wipMenu.getByRole("link").count(), access === "ordinary" ? 1 : 3);
      await page.keyboard.press("Escape");
    }
  }
  await page.getByLabel("Report", { exact: true }).selectOption("branch");
  await page.getByRole("table", { name: "North: ranked WIP", exact: true }).waitFor();
  await wipButton.click();
  await wipMenu.waitFor();
  assert.equal(
    await wipMenu.getByRole("link", { name: "Branch", exact: true }).getAttribute("aria-current"),
    "page",
  );
  assert.equal(await page.locator(":focus").innerText(), "Branch");
  await page.keyboard.press("Tab");
  assert.equal(await page.locator(":focus").innerText(), "Division");
  await page.keyboard.press("Tab");
  assert.equal(await wipMenu.isVisible(), false, "Tab can leave the menu.");
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

  waitForMissingRates = new Promise((resolve) => {
    releaseMissingRates = resolve;
  });
  await estimate.click();
  await timeDetails.getByRole("button", { name: "missing rates", exact: true }).click();
  await timeDetails.getByRole("status").waitFor();
  await page.getByLabel("Report", { exact: true }).selectOption("division");
  await page.getByTestId("report-normal").waitFor();
  releaseMissingRates();
  await estimate.click();
  assert.equal(
    await rateSheetLink.count(),
    0,
    "a stale role response must not enter the new report popup",
  );
  await page.keyboard.press("Escape");
  assert.equal(requests.at(-1).searchParams.get("division"), "civil");
  assert.equal(requests.at(-1).searchParams.get("require_time"), "true");
  assert.equal(await page.getByTestId("report-empty").count(), 0);
  await page.getByRole("checkbox", { name: "Must include CIV hours", exact: true }).uncheck();
  await page.getByTestId("report-empty").waitFor();
  assert.equal(requests.at(-1).searchParams.get("require_time"), "false");
  await page.getByLabel("Default division", { exact: true }).selectOption("");
  await page.getByText("Select a division to view its projects.", { exact: true }).waitFor();
  await page.getByRole("checkbox", { name: "Must include division hours", exact: true }).waitFor();
  const division = page.getByLabel("Division", { exact: true });
  await division.fill("STR");
  await division.press("ArrowDown");
  await division.press("Enter");
  await page.getByTestId("report-normal").waitFor();
  assert.equal(requests.at(-1).searchParams.get("division"), "structural");
  emptyMissingRates = true;
  await estimate.click();
  await timeDetails.getByRole("button", { name: "missing rates", exact: true }).click();
  await timeDetails
    .getByText("No missing role rates found. The job or rate sheet may have changed.", {
      exact: true,
    })
    .waitFor();
  await page.keyboard.press("Escape");
  emptyMissingRates = false;
  await page.getByRole("checkbox", { name: "Must include STR hours", exact: true }).waitFor();
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-desktop.png`,
      fullPage: true,
    });
  await page.setViewportSize({ width: 360, height: 800 });
  await wipButton.click();
  await wipMenu.waitFor();
  const mobileMenu = await wipMenu.boundingBox();
  assert.ok(mobileMenu.x >= 0 && mobileMenu.x + mobileMenu.width <= 360);
  assert.ok(mobileMenu.y >= 0 && mobileMenu.y + mobileMenu.height <= 800);
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({ path: `${process.env.WIP_REPORT_SCREENSHOT}-menu-mobile.png` });
  await page.keyboard.press("Escape");
  assert.equal(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    true,
  );
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-mobile.png`,
      fullPage: true,
    });
  employeeOnly = true;
  await page.getByLabel("Report", { exact: true }).selectOption("branch");
  const employeeRow = page.getByTestId("report-employee-rates");
  await employeeRow.waitFor();
  const timeCell = employeeRow.getByRole("cell").nth(2);
  assert.equal(
    await timeCell
      .getByRole("button", { name: "Estimated: Time value: value details", exact: true })
      .count(),
    0,
  );
  assert.equal(
    await employeeRow
      .getByRole("rowheader")
      .getByRole("button", { name: /No rate sheet/ })
      .count(),
    0,
  );
  const noSheet = timeCell.getByRole("button", { name: /No rate sheet/ });
  assert.equal(await noSheet.innerText(), "NR");
  const nrBounds = await noSheet.boundingBox();
  const amountBounds = await timeCell.getByText("$40,000.00", { exact: true }).boundingBox();
  assert.ok(nrBounds.x + nrBounds.width <= amountBounds.x, "NR precedes the time amount");
  await noSheet.click();
  const noSheetDialog = page.getByRole("dialog", { name: "Time uses employee defaults" });
  await noSheetDialog.waitFor();
  assert.match(await noSheetDialog.innerText(), /400.00 hours use employee default/);
  assert.doesNotMatch(
    await noSheetDialog.innerText(),
    /cannot be priced|no role recorded|no matching rate/,
  );
  await page.keyboard.press("Escape");
  remainingExamples = true;
  await page.getByLabel("Report", { exact: true }).selectOption("my");
  await page.getByTestId("report-small-over").waitFor();
  assert.equal(
    await ranked.locator("tbody tr").first().getAttribute("data-testid"),
    "report-small-over",
  );
  await sort.selectOption("remaining");
  assert.equal(
    await ranked.locator("tbody tr").first().getAttribute("data-testid"),
    "report-large-over",
  );
  await sort.selectOption("percent");
  assert.equal(
    await ranked.locator("tbody tr").first().getAttribute("data-testid"),
    "report-small-over",
  );
  remainingExamples = false;
  percentageExamples = true;
  await page.getByLabel("Report", { exact: true }).selectOption("division");
  await division.fill("STR");
  await division.press("ArrowDown");
  await division.press("Enter");
  await page.getByTestId("report-percent-540").waitFor();
  const colours = new Map();
  for (const value of [-10, 0, 25, 50, 75, 100, 130, 540]) {
    const row = page.getByTestId(`report-percent-${value}`);
    const cell = row.locator("td").nth(-2);
    assert.equal(await cell.innerText(), `${value.toFixed(1)}%`);
    const fill = cell.getByTestId("wip-percentage-fill");
    const display = await fill.evaluate((el) => ({
      share: el.getBoundingClientRect().width / el.parentElement.getBoundingClientRect().width,
      background: getComputedStyle(el).backgroundImage,
    }));
    assert.ok(Math.abs(display.share - Math.max(0, Math.min(value, 100)) / 100) < 0.01);
    colours.set(value, display.background);
    assert.ok(await row.evaluate((el) => el.getBoundingClientRect().height <= 38));
  }
  assert.equal(new Set([0, 25, 50, 75, 100, 130].map((value) => colours.get(value))).size, 6);
  assert.equal(colours.get(130), colours.get(540), "over-budget values use the same purple scale");
  await page.getByRole("region", { name: "Projects: ranked WIP", exact: true }).evaluate((el) => {
    el.scrollLeft = el.scrollWidth;
  });
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-percentages-mobile.png`,
      fullPage: true,
    });
  await page.setViewportSize({ width: 1400, height: 900 });
  if (process.env.WIP_REPORT_SCREENSHOT)
    await page.screenshot({
      path: `${process.env.WIP_REPORT_SCREENSHOT}-percentages.png`,
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
  // Use a small destination page to verify real link navigation without app authentication.
  await page.route("**/jobs/wip/*", (route) =>
    route.fulfill({ contentType: "text/html", body: "<h1>Selected WIP view</h1>" }),
  );
  for (const [label, mode] of [
    ["My WIP", "my"],
    ["Branch", "branch"],
    ["Division", "division"],
  ]) {
    await page.goto(`${origin}/tests/fixtures/wipReports.html`);
    await wipButton.click();
    await wipMenu.getByRole("link", { name: label, exact: true }).click();
    await page.waitForURL(`${origin}/jobs/wip/${mode}`);
    await page.getByRole("heading", { name: "Selected WIP view" }).waitFor();
  }
  assert.deepEqual(errors, []);
  console.log(
    "WIP report browser checks passed: ranking, controls, warnings, access, filters, defaults, retry, stale responses, empty states, and mobile layout.",
  );
});
