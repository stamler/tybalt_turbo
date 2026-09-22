import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";
const fixtures = JSON.parse(
  await readFile(new URL("./fixtures/wip.json", import.meta.url), "utf8"),
);

await runBrowserHarness(async (page, origin) => {
  const errors = [];
  let requests = 0;
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/jobs/**/wip", async (route) => {
    requests++;
    const scenario = new URL(route.request().url()).pathname.split("/")[3];
    if (scenario === "error")
      return route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"message":"test failure"}',
      });
    if (scenario === "slow") await new Promise((resolve) => setTimeout(resolve, 350));
    await route
      .fulfill({
        contentType: "application/json",
        body: JSON.stringify(fixtures[scenario === "slow" ? "over" : scenario]),
      })
      .catch(() => {});
  });
  await page.goto(`${origin}/tests/fixtures/wip.html`);
  const total = page.getByTestId("wip-total");
  const percent = page.getByTestId("wip-percent");
  await total.waitFor();
  assert.equal(await total.innerText(), "$65,000.00");
  assert.equal(await percent.innerText(), "65.0%");
  assert.match(await page.getByTestId("wip-time").innerText(), /40\.0%/);
  const timeHelp = page.getByRole("button", { name: "How time value is calculated", exact: true });
  await timeHelp.focus();
  await page.keyboard.press("Enter");
  const timeExplanation = page.getByRole("dialog", {
    name: "How time value is calculated",
    exact: true,
  });
  await timeExplanation.waitFor();
  assert.match(
    await timeExplanation.innerText(),
    /Hours × job rate, with employee defaults where needed/,
  );
  await page.keyboard.press("Escape");
  assert.equal(await timeHelp.evaluate((el) => el === document.activeElement), true);
  const initialRequests = requests;
  await page.getByLabel("Include committed expenses", { exact: true }).uncheck();
  assert.equal(await total.innerText(), "$55,000.00");
  assert.equal(await percent.innerText(), "55.0%");
  assert.match(await page.getByTestId("wip-expenses").innerText(), /Excluded/);
  assert.match(await page.getByTestId("wip-pos").innerText(), /15,000\.00/);
  await page.getByLabel("Include active POs", { exact: true }).uncheck();
  assert.equal(await total.innerText(), "$40,000.00");
  await page.getByLabel("Include committed expenses", { exact: true }).check();
  assert.equal(await total.innerText(), "$50,000.00");
  await page.getByLabel("Include active POs", { exact: true }).check();
  assert.equal(requests, initialRequests, "inclusion toggles do not change source balances");
  await page.getByRole("button", { name: "Show breakdown chart" }).click();
  await page.getByRole("img", { name: "Doughnut chart of included WIP components" }).waitFor();
  assert.match(
    await page.getByRole("figure", { name: "Share of included total", exact: true }).innerText(),
    /61\.5%/,
  );
  assert.equal(await page.locator("svg circle").count(), 3);
  await page.getByLabel("Include committed expenses", { exact: true }).uncheck();
  assert.equal(await page.locator("svg circle").count(), 2);
  await page.getByLabel("Include committed expenses", { exact: true }).check();
  const help = page.getByRole("button", { name: "How WIP is calculated", exact: true });
  await help.focus();
  await page.keyboard.press("Enter");
  await page.getByRole("dialog").waitFor();
  await page.keyboard.press("Escape");
  assert.equal(await help.evaluate((el) => el === document.activeElement), true);
  if (process.env.WIP_SCREENSHOT)
    await page.screenshot({ path: `${process.env.WIP_SCREENSHOT}-desktop.png`, fullPage: true });

  async function scenario(name) {
    const response = page.waitForResponse((res) => res.url().endsWith(`/api/jobs/${name}/wip`));
    await page.getByLabel("Scenario").selectOption(name);
    await response;
    if (name !== "error") await total.waitFor();
  }
  await scenario("over");
  assert.equal(await percent.innerText(), "130.0%");
  assert.match(await page.getByTestId("wip-balance").innerText(), /15,000\.00 over project value/);
  await scenario("partial");
  assert.equal(await percent.innerText(), "—");
  assert.equal(await page.getByTestId("wip-budget-bar").count(), 0);
  await page
    .getByRole("button", { name: "Partial: Time value: value details", exact: true })
    .click();
  assert.match(await page.getByRole("dialog").innerText(), /100\.00 hours cannot be priced/);
  await page.keyboard.press("Escape");
  await scenario("unknown-po");
  assert.equal(await percent.innerText(), "—");
  await page.getByLabel("Include active POs", { exact: true }).uncheck();
  assert.equal(await percent.innerText(), "50.0%");
  await scenario("unknown-expense");
  await page.getByLabel("Include committed expenses", { exact: true }).uncheck();
  assert.equal(await percent.innerText(), "55.0%");
  await scenario("zero-project");
  assert.equal(await percent.innerText(), "—");
  assert.equal(await total.innerText(), "$65,000.00");
  await page.getByRole("button", { name: "Show breakdown chart" }).click();
  await page.getByRole("img", { name: "Doughnut chart of included WIP components" }).waitFor();
  await scenario("empty");
  assert.equal(await percent.innerText(), "0.0%");
  await page.getByRole("button", { name: "Show breakdown chart" }).click();
  assert.equal(await page.locator("svg circle").count(), 0);
  await scenario("negative");
  assert.equal(await page.getByTestId("wip-budget-bar").count(), 0);
  await scenario("error");
  await page.getByRole("alert").waitFor();
  const beforeRetry = requests;
  await page.getByRole("button", { name: "Try again" }).click();
  await page.getByRole("alert").waitFor();
  assert.equal(requests, beforeRetry + 1);
  await page.getByLabel("Scenario").selectOption("slow");
  await page.getByRole("status").waitFor();
  await scenario("normal");
  await page.waitForTimeout(450);
  assert.equal(await percent.innerText(), "65.0%");
  await page.setViewportSize({ width: 360, height: 800 });
  await page.getByRole("button", { name: "Show breakdown chart" }).click();
  assert.equal(
    await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
    true,
  );
  if (process.env.WIP_SCREENSHOT)
    await page.screenshot({ path: `${process.env.WIP_SCREENSHOT}-mobile.png`, fullPage: true });
  assert.deepEqual(errors, []);
  console.log(
    "WIP browser checks passed: toggles, denominators, charts, over-budget and incomplete values, keyboard help, retry, stale responses, and mobile layout.",
  );
});
