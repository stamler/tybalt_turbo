import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";

const fixture = JSON.parse(
  await readFile(new URL("./fixtures/poApprovalLimits.json", import.meta.url), "utf8"),
);

await runBrowserHarness(async (page, origin) => {
  const errors = [];
  const methods = [];
  let mode = "normal";
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/purchase_orders/approval_limits", async (route) => {
    methods.push(route.request().method());
    if (mode === "error" || mode === "forbidden") {
      return route.fulfill({
        status: mode === "error" ? 500 : 403,
        contentType: "application/json",
        body: '{"message":"Test failure"}',
      });
    }
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(mode === "empty" ? { ...fixture, items: [] } : fixture),
    });
  });
  await page.goto(`${origin}/tests/fixtures/poApprovalLimits.html`);
  const table = page.getByRole("table");
  await table.waitFor();
  assert.equal(await table.locator("tbody tr").count(), 4);
  const bailey = table.getByRole("row", { name: /Bailey Baker/ });
  assert.match(await bailey.innerText(), /\$500\.00/);
  assert.match(await bailey.innerText(), /\$0\.00/);
  assert.equal(await bailey.getByTitle("ENG — Engineering", { exact: true }).innerText(), "ENG");
  assert.match(
    await table.getByRole("row", { name: /Casey Carter/ }).innerText(),
    /Not configured/,
  );
  assert.equal(await page.locator('input[type="number"]').count(), 0);
  assert.equal(await page.getByRole("button", { name: /save|delete|edit/i }).count(), 0);

  await page.getByRole("searchbox", { name: "Search", exact: true }).fill("  BAILEY BAKER  ");
  assert.equal(await table.locator("tbody tr").count(), 1);
  await page.getByRole("searchbox", { name: "Search", exact: true }).fill("no match");
  await page.getByText("No approvers match your search and division.").waitFor();
  await page.getByRole("searchbox", { name: "Search", exact: true }).fill("");
  await page.getByLabel("Division", { exact: true }).selectOption("eng");
  assert.equal(await table.locator("tbody tr").count(), 2);
  await page.getByLabel("Division", { exact: true }).selectOption("admin");
  assert.equal(await table.locator("tbody tr").count(), 2);
  assert.match(await table.locator("tbody tr").first().innerText(), /Avery Able/);
  await page.getByLabel("Division", { exact: true }).selectOption("");
  // These orders come from the fixture values, not the page's comparator.
  // Casey has no settings and must stay last in both directions.
  const amountSortCases = [
    {
      label: "Project",
      descending: ["Drew Dover", "Avery Able", "Bailey Baker", "Casey Carter"],
      ascending: ["Bailey Baker", "Avery Able", "Drew Dover", "Casey Carter"],
    },
    {
      label: "Capital",
      descending: ["Avery Able", "Bailey Baker", "Drew Dover", "Casey Carter"],
      ascending: ["Drew Dover", "Bailey Baker", "Avery Able", "Casey Carter"],
    },
    {
      label: "Staff and Social",
      descending: ["Drew Dover", "Avery Able", "Bailey Baker", "Casey Carter"],
      ascending: ["Bailey Baker", "Avery Able", "Drew Dover", "Casey Carter"],
    },
    {
      label: "Sponsorship",
      descending: ["Avery Able", "Drew Dover", "Bailey Baker", "Casey Carter"],
      ascending: ["Bailey Baker", "Drew Dover", "Avery Able", "Casey Carter"],
    },
    {
      label: "Media/Advertising",
      descending: ["Avery Able", "Drew Dover", "Bailey Baker", "Casey Carter"],
      ascending: ["Bailey Baker", "Drew Dover", "Avery Able", "Casey Carter"],
    },
    {
      label: "Computers/Software",
      descending: ["Drew Dover", "Avery Able", "Bailey Baker", "Casey Carter"],
      ascending: ["Bailey Baker", "Avery Able", "Drew Dover", "Casey Carter"],
    },
  ];
  const rowNames = () => table.locator("tbody th").allTextContents();
  for (const [index, expected] of amountSortCases.entries()) {
    const header = table.locator("thead th").nth(index + 1);
    assert.equal((await header.innerText()).trim(), expected.label);
    await header.getByRole("button").click();
    assert.equal(await header.getAttribute("aria-sort"), "descending");
    assert.deepEqual(
      (await rowNames()).map((name) => name.trim()),
      expected.descending,
    );
    await header.getByRole("button").click();
    assert.equal(await header.getAttribute("aria-sort"), "ascending");
    assert.deepEqual(
      (await rowNames()).map((name) => name.trim()),
      expected.ascending,
    );
  }
  await table.getByRole("button", { name: "Divisions", exact: true }).click();
  assert.deepEqual(
    (await rowNames()).map((name) => name.trim()),
    ["Drew Dover", "Avery Able", "Bailey Baker", "Casey Carter"],
  );
  await table.getByRole("button", { name: "Divisions ↑", exact: true }).click();
  assert.deepEqual(
    (await rowNames()).map((name) => name.trim()),
    ["Bailey Baker", "Avery Able", "Drew Dover", "Casey Carter"],
  );
  await table.getByRole("button", { name: "Name", exact: true }).click();
  assert.match((await rowNames())[0], /Avery Able/);
  await table.getByRole("button", { name: "Name ↑", exact: true }).click();
  assert.match((await rowNames())[0], /Drew Dover/);
  await table.getByRole("button", { name: "Name ↓", exact: true }).click();
  await page.getByRole("button", { name: "Help: How limits apply" }).click();
  await page.getByRole("dialog", { name: "How limits apply" }).waitFor();
  await page.getByText("Capital: $500.00", { exact: true }).waitFor();
  await page.keyboard.press("Escape");
  assert.equal(await page.getByRole("dialog").count(), 0);
  const sections = JSON.parse(await page.getByTestId("nav-config").textContent());
  const business = sections.find((section) => section.title === "Business").items;
  const approversIndex = business.findIndex((item) => item.label === "PO Approvers");
  assert.equal(business[approversIndex - 1].label, "Absorb Actions");
  assert.equal(business[approversIndex + 1].label, "Admin Profiles");
  assert.equal((await page.getByTestId("nav-access").textContent()).trim(), "true");
  assert.equal(
    sections
      .flatMap((section) => section.items)
      .filter((item) => item.href === "/pos/approval-limits").length,
    1,
  );
  // Match the list pages: aligned controls and row stripes without cell borders.
  const toolbar = page.getByRole("searchbox", { name: "Search" }).locator("..");
  const controlHeights = await toolbar
    .locator("input, select, button")
    .evaluateAll((controls) => controls.map((control) => control.getBoundingClientRect().height));
  assert.deepEqual(controlHeights, [36, 36, 36]);
  const rowStyles = await table.locator("tbody tr").evaluateAll((rows) =>
    rows.map((row) => ({
      background: getComputedStyle(row).backgroundColor,
      border: getComputedStyle(row).borderBottomWidth,
    })),
  );
  assert.notEqual(rowStyles[0].background, rowStyles[1].background);
  assert.equal(rowStyles[0].background, rowStyles[2].background);
  assert.ok(rowStyles.every((row) => row.border === "0px"));
  await page.screenshot({
    path: "/private/tmp/tybalt-po-approval-limits-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  assert.equal(
    await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth),
    true,
    "only the table should scroll horizontally on mobile",
  );
  // Keep names visible when a narrow screen needs horizontal scrolling.
  const region = page.getByRole("region", { name: "PO approval limits", exact: true });
  const firstName = table.locator("tbody th").first();
  const beforeScroll = await firstName.boundingBox();
  await region.evaluate((element) => {
    element.scrollLeft = element.scrollWidth;
  });
  const afterScroll = await firstName.boundingBox();
  assert.ok(
    Math.abs(beforeScroll.x - afterScroll.x) < 1,
    "names must stay fixed while amounts scroll",
  );
  assert.equal(await firstName.innerText(), "Avery Able");
  await region.evaluate((element) => {
    element.scrollLeft = 0;
  });
  assert.deepEqual(
    await toolbar
      .locator("input, select, button")
      .evaluateAll((controls) => controls.map((control) => control.getBoundingClientRect().height)),
    [36, 36, 36],
  );
  await page.screenshot({
    path: "/private/tmp/tybalt-po-approval-limits-mobile.png",
    fullPage: true,
  });

  mode = "error";
  await page.getByRole("button", { name: "Refresh" }).click();
  await page.getByRole("alert").waitFor();
  assert.match(await page.getByRole("alert").innerText(), /Could not load/);
  assert.equal(await table.count(), 0);
  mode = "normal";
  await page.getByRole("button", { name: "Retry" }).click();
  await table.waitFor();
  mode = "forbidden";
  await page.getByRole("button", { name: "Refresh" }).click();
  await page.getByText("You do not have permission to view PO approval limits.").waitFor();
  assert.equal(await table.count(), 0);
  mode = "empty";
  await page.getByRole("button", { name: "Retry" }).click();
  await page.getByText("No active PO approvers found.").waitFor();
  const beforeDenied = methods.length;
  await page.getByLabel("Test access").selectOption("none");
  await page
    .getByText("You need the report or po_approver claim to view PO approval limits.")
    .waitFor();
  assert.equal(methods.length, beforeDenied, "unauthorized page must not load the report");
  assert.equal((await page.getByTestId("nav-access").textContent()).trim(), "false");
  mode = "normal";
  await page.getByLabel("Test access").selectOption("admin");
  await page
    .getByText("You need the report or po_approver claim to view PO approval limits.")
    .waitFor();
  assert.equal(methods.length, beforeDenied, "admin alone must not load the report");
  assert.equal((await page.getByTestId("nav-access").textContent()).trim(), "false");
  await page.getByLabel("Test access").selectOption("report,admin");
  await table.waitFor();
  assert.equal(await table.locator("tbody tr").count(), 4);
  for (const access of ["po_approver", "report,po_approver"]) {
    await page.getByLabel("Test access").selectOption("none");
    await page.getByRole("alert").waitFor();
    await page.getByLabel("Test access").selectOption(access);
    await table.waitFor();
    assert.equal(await table.locator("tbody tr").count(), 4);
    assert.equal((await page.getByTestId("nav-access").textContent()).trim(), "true");
  }
  assert.deepEqual([...new Set(methods)], ["GET"]);
  assert.deepEqual(errors, []);
});
console.log("PO approval limits browser checks passed.");
