// Run with an installed Playwright, or set PLAYWRIGHT_MODULE to its module path.
// Only framework services are stubbed; the real components, SDK, CSV helper,
// styles, and native popover controls run in Chromium.
import assert from "node:assert/strict";
import { readFile, mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createServer } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
const require = createRequire(import.meta.url);
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");
const root = fileURLToPath(new URL("../", import.meta.url));
const fixtures = JSON.parse(
  await readFile(new URL("./fixtures/timeSummary.json", import.meta.url), "utf8"),
);
const services = {
  "$app/paths": "export const resolve = (route, params) => route.replace('[id]', params.id);",
  "$app/navigation": "export const goto = (url) => { location.href = url; };",
  "$env/static/public": "export const PUBLIC_POCKETBASE_URL = location.origin;",
  "summary-test-global":
    "import { writable } from 'svelte/store'; export const globalStore = writable({claims:[]});",
};
const cacheDir = await mkdtemp(path.join(tmpdir(), "tybalt-summary-vite-"));
const server = await createServer({
  root,
  configFile: false,
  cacheDir,
  plugins: [
    {
      name: "summary-test-services",
      enforce: "pre",
      resolveId(id) {
        if (id in services) return `\0${id}`;
      },
      load(id) {
        return services[id.slice(1)];
      },
    },
    svelte({ configFile: false }),
  ],
  resolve: {
    alias: [
      { find: "$lib/stores/global", replacement: "summary-test-global" },
      { find: "$lib", replacement: path.join(root, "src/lib") },
    ],
  },
  server: { host: "127.0.0.1", port: 0 },
});
let browser;
try {
  await server.listen();
  const address = server.httpServer.address();
  browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL });
  const page = await browser.newPage({ viewport: { width: 1000, height: 760 } });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/jobs/**", async (route) => {
    const url = new URL(route.request().url());
    const scenario = url.pathname.split("/")[3];
    if (scenario === "error")
      return route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"message":"test error"}',
      });
    if (scenario === "slow") await new Promise((resolve) => setTimeout(resolve, 350));
    const rows = url.pathname.includes("/divisions/")
      ? fixtures.divisions
      : fixtures[scenario === "slow" ? "unpriced" : scenario];
    await route
      .fulfill({ status: 200, contentType: "application/json", body: JSON.stringify(rows) })
      .catch(() => {});
  });
  await page.goto(`http://127.0.0.1:${address.port}/tests/fixtures/timeSummary.html`);
  await page.getByRole("table").waitFor();
  if (process.env.SUMMARY_SCREENSHOT)
    await page.screenshot({
      path: process.env.SUMMARY_SCREENSHOT.replace(/\.png$/, "-desktop.png"),
      fullPage: true,
    });
  assert.deepEqual(await page.locator("thead th").allTextContents(), [
    "Staff member",
    "Hours",
    "Value",
    "Share",
  ]);
  assert.equal(await page.locator("tbody tr").count(), 3);
  assert.equal(await page.locator("tfoot tr").count(), 1);
  assert.match(await page.locator("tfoot").innerText(), /2,700\.00/);
  assert.equal(await page.getByRole("button", { name: /^(Estimated|Partial):/ }).count(), 0);
  const help = page.getByRole("button", { name: "How time values are calculated", exact: true });
  await help.focus();
  await page.keyboard.press("Enter");
  await page.getByRole("dialog", { name: "How time values are calculated" }).waitFor();
  await page.keyboard.press("Escape");
  assert.equal(await help.evaluate((el) => el === document.activeElement), true);
  await help.click();
  await page.getByLabel("Scenario").click();
  assert.equal(await page.getByRole("dialog").count(), 0);
  await page.keyboard.press("Escape");

  await page.getByRole("button", { name: "Value", exact: true }).click();
  assert.match(await page.locator("tbody tr").first().innerText(), /Casey Patel/);
  await page.getByRole("button", { name: "Alex Morgan", exact: true }).click();
  assert.equal(await page.locator("tbody tr").count(), 1);
  assert.match(await page.locator("tfoot").innerText(), /Visible total/);
  assert.match(await page.locator("tfoot").innerText(), /33\.3%/);
  const downloadEvent = page.waitForEvent("download");
  await page.getByRole("button", { name: "Download full summary CSV" }).click();
  const download = await downloadEvent;
  const csv = await readFile(await download.path(), "utf8");
  assert.match(csv, /estimated_hours/);
  assert.match(csv, /rate_sheet_revision/);
  assert.match(csv, /Blair/);
  await page.getByRole("button", { name: "Remove Staff member filter" }).click();

  async function scenario(value) {
    await page.getByLabel("Scenario").selectOption(value);
    if (value !== "error" && value !== "empty") await page.getByRole("table").waitFor();
  }
  await scenario("mixed");
  const estimated = page.getByRole("button", {
    name: "Estimated: Alex Morgan: employee rates used",
    exact: true,
  });
  await estimated.click();
  const estimatedDialog = page.getByRole("dialog", {
    name: "Alex Morgan: employee rates used",
    exact: true,
  });
  await estimatedDialog.waitFor();
  assert.equal(await estimatedDialog.evaluate((el) => getComputedStyle(el).textAlign), "left");
  assert.equal(await estimatedDialog.evaluate((el) => getComputedStyle(el).fontWeight), "400");
  assert.match(await estimatedDialog.innerText(), /2\.00 of 8\.00/);
  await page.getByRole("button", { name: "Close explanation", exact: true }).click();
  await scenario("no-sheet");
  assert.equal(await page.getByRole("button", { name: /^Estimated:/ }).count(), 0);
  await page.getByRole("button", { name: /^No rate sheet · employee rates used/ }).click();
  assert.match(await page.getByRole("dialog").innerText(), /24\.00 hours/);
  await page.keyboard.press("Escape");
  await scenario("partial");
  assert.match(await page.locator("tfoot").innerText(), /Known total/);
  await page
    .getByRole("button", { name: "Partial: Alex Morgan: unpriced hours", exact: true })
    .click();
  assert.match(await page.getByRole("dialog").innerText(), /2\.00 of 8\.00/);
  await page.keyboard.press("Escape");
  await scenario("unpriced");
  assert.doesNotMatch(await page.locator("table").innerText(), /\$0\.00/);
  await scenario("empty");
  await page.getByText("No time entries found for this date range.").waitFor();
  await scenario("error");
  await page.getByRole("alert").waitFor();
  assert.equal(await page.getByRole("table").count(), 0);
  await page.getByLabel("Scenario").selectOption("slow");
  await page.getByRole("status").waitFor();
  await scenario("normal");
  await page.waitForTimeout(450);
  assert.match(await page.locator("tfoot").innerText(), /2,700\.00/);
  await page.getByLabel("Start date").fill("");
  await page.getByText("Please select a start and end date.").waitFor();
  await page.getByLabel("Start date").fill("2026-09-01");
  await page.getByRole("table").waitFor();
  await page.getByLabel("Summary").selectOption("divisions");
  await page.getByRole("columnheader", { name: "Division", exact: true }).waitFor();
  assert.equal(await page.locator("tbody tr").count(), 2);

  await page.setViewportSize({ width: 360, height: 740 });
  await help.click();
  await page.getByRole("dialog").waitFor();
  const box = await page.getByRole("dialog").boundingBox();
  assert.ok(
    box.x >= 0 && box.x + box.width <= 360 && box.y >= 0 && box.y + box.height <= 740,
    "popover fits mobile viewport",
  );
  assert.equal(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    true,
  );
  await page.keyboard.press("Escape");
  await page.getByLabel("Summary").selectOption("staff");
  await scenario("mixed");
  await estimated.click();
  await page.getByRole("dialog").waitFor();
  const rowBox = await page.getByRole("dialog").boundingBox();
  assert.ok(rowBox.x >= 0 && rowBox.x + rowBox.width <= 360, "row help fits viewport");
  if (process.env.SUMMARY_SCREENSHOT)
    await page.screenshot({ path: process.env.SUMMARY_SCREENSHOT, fullPage: true });
  assert.deepEqual(errors, []);
  console.log(
    "Time summary browser checks passed: layout, sorting, filters, CSV, all pricing states, keyboard help, dismiss controls, load failures, stale responses, and mobile layout.",
  );
} finally {
  await browser?.close();
  await server.close();
  await rm(cacheDir, { recursive: true, force: true });
}
