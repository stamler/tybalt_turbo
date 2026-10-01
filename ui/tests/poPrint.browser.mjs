import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";

const fixtures = JSON.parse(
  await readFile(new URL("./fixtures/poPrint.json", import.meta.url), "utf8"),
);
const normalize = (text) => text.replace(/\s/g, " ");

await runBrowserHarness(async (page, origin) => {
  const errors = [],
    methods = [];
  let mode = "oneTime";
  const context = page.context();
  context.setDefaultTimeout(10000);
  context.on("page", (child) => child.on("pageerror", (error) => errors.push(error.message)));
  page.on("pageerror", (error) => errors.push(error.message));
  await context.route("**/logo.svg", async (route) =>
    route.fulfill({ response: await route.fetch({ url: `${origin}/static/logo.svg` }) }),
  );
  await context.route(
    (url) => url.pathname.startsWith("/pos/"),
    async (route) =>
      route.fulfill({
        response: await route.fetch({ url: `${origin}/tests/fixtures/poPrint.html` }),
      }),
  );
  await context.route("**/api/purchase_orders/visible/printpo", (route) => {
    methods.push(route.request().method());
    if (mode === "error")
      return route.fulfill({
        status: 500,
        contentType: "application/json",
        body: '{"message":"Test failure"}',
      });
    // This response models an expense committed while the popup was open.
    const fixture =
      mode === "changed" ? { ...fixtures.cumulative, print_max_amount: 6500 } : fixtures[mode];
    return route.fulfill({ contentType: "application/json", body: JSON.stringify(fixture) });
  });
  await page.goto(`${origin}/tests/fixtures/poPrint.html`);
  const dialog = page.getByRole("dialog", { name: "Print options" });
  const amount = dialog.getByLabel(/Amount to print/);
  const proceed = dialog.getByRole("button", { name: "Proceed" });
  const tax = dialog.getByLabel("Plus tax", { exact: true });
  const shipping = dialog.getByLabel("Plus shipping", { exact: true });
  const acknowledgement = dialog.getByLabel(/I understand/);
  const open = async () => {
    await page.getByRole("button", { name: "Print", exact: true }).click();
    await amount.waitFor();
  };
  const print = async () => {
    const opened = context.waitForEvent("page");
    await proceed.click();
    const child = await opened;
    await child.waitForURL((url) => url.pathname === "/pos/printpo/print");
    await child.waitForLoadState("domcontentloaded");
    assert.deepEqual([...new URL(child.url()).searchParams.keys()], ["print"]);
    return child;
  };

  await open();
  assert.equal(await amount.inputValue(), "10000.00");
  assert.equal(await tax.isChecked(), false);
  assert.equal(await shipping.isChecked(), false);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  assert.equal(context.pages().length, 1);
  await open();
  await page.keyboard.press("Escape");
  assert.equal(await dialog.isVisible(), false);
  assert.equal(
    await page
      .getByRole("button", { name: "Print", exact: true })
      .evaluate((button) => button === document.activeElement),
    true,
  );

  await open();
  await page.evaluate(() => {
    window.originalOpen = window.open;
    window.open = () => null;
  });
  await proceed.click();
  await dialog.getByText("Allow pop-ups for this site, then select Proceed again.").waitFor();
  assert.equal(await dialog.isVisible(), true);
  await page.evaluate(() => {
    window.open = window.originalOpen;
  });
  await dialog.getByRole("button", { name: "Cancel" }).click();

  for (const [plusTax, plusShipping, suffix] of [
    [false, false, ""],
    [true, false, " plus tax"],
    [false, true, " plus shipping"],
    [true, true, " plus tax and shipping"],
  ]) {
    await open();
    await amount.fill("8000");
    await tax.setChecked(plusTax);
    await shipping.setChecked(plusShipping);
    if (plusTax || plusShipping) {
      assert.equal(await proceed.isDisabled(), true);
      assert.match(normalize(await dialog.getByRole("alert").innerText()), /CAD 2,000.00/);
      await acknowledgement.check();
    }
    assert.equal(await proceed.isEnabled(), true);
    const expected = `CAD 8,000.00${suffix}`;
    assert.equal(normalize(await dialog.getByTestId("po-print-preview").innerText()), expected);
    if (plusTax && plusShipping) {
      await page.screenshot({ path: "/private/tmp/po-print-options-desktop.png" });
      await page.setViewportSize({ width: 390, height: 844 });
      assert.equal(await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth), true);
      await page.evaluate(
        () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
      );
      await page.screenshot({ path: "/private/tmp/po-print-options-mobile.png" });
      await page.setViewportSize({ width: 1000, height: 900 });
    }
    const child = await print();
    await child.waitForFunction(() => window.printCalls === 1);
    await child.waitForFunction(() => document.querySelector("article img")?.naturalWidth > 0);
    const document = child.locator("article");
    const content = normalize(await document.innerText());
    assert.ok(content.includes(expected));
    assert.doesNotMatch(
      content,
      /10,000|2,000|Maximum Authorized Total|must not exceed|I understand|on file/,
    );
    await child.evaluate(() => window.print());
    assert.equal(await child.evaluate(() => window.printCalls), 2);
    assert.equal(normalize(await document.innerText()), content);
    await child.emulateMedia({ media: "print" });
    assert.equal(await child.locator(".screen-only").isVisible(), false);
    await child.screenshot({ path: "/private/tmp/po-print-document.png", fullPage: true });
    await child.close();
  }

  await open();
  for (const invalid of ["", "0", "-1", "abc", "10000.01", "1.001", "1e3"]) {
    await amount.fill(invalid);
    assert.equal(await proceed.isDisabled(), true, invalid);
    await dialog.getByRole("alert").waitFor();
  }
  await amount.fill("10000");
  await tax.check();
  assert.equal(await proceed.isDisabled(), true);
  await amount.fill("8000");
  await acknowledgement.check();
  await amount.fill("7900");
  assert.equal(await acknowledgement.isChecked(), false);
  await amount.fill("8000");
  assert.equal(await acknowledgement.isChecked(), false);
  await acknowledgement.check();
  await shipping.check();
  assert.equal(await acknowledgement.isChecked(), false);
  await dialog.getByRole("button", { name: "Cancel" }).click();

  mode = "cumulative";
  await open();
  assert.equal(await amount.inputValue(), "7000.00");
  await dialog.getByText("Remaining available balance", { exact: true }).waitFor();
  await amount.fill("6000");
  await tax.check();
  await shipping.check();
  assert.match(normalize(await dialog.getByRole("alert").innerText()), /CAD 1,000.00/);
  await acknowledgement.check();
  mode = "changed";
  const stale = await print();
  await stale.getByRole("alert").waitFor();
  assert.match(await stale.getByRole("alert").innerText(), /available amount has changed/);
  assert.equal(await stale.locator("article").count(), 0);
  assert.equal(await stale.evaluate(() => window.printCalls), 0);
  await stale.close();

  for (const name of ["exhausted", "overdrawn", "closed"]) {
    mode = name;
    await open();
    assert.equal(await proceed.isDisabled(), true);
    await dialog.getByRole("button", { name: "Cancel" }).click();
  }
  mode = "error";
  await page.getByRole("button", { name: "Print", exact: true }).click();
  await dialog.getByText("Could not load the purchase order. Try again.").waitFor();
  assert.equal(await proceed.isDisabled(), true);
  mode = "recurring";
  await dialog.getByRole("button", { name: "Retry" }).click();
  await amount.waitFor();
  assert.equal(await amount.inputValue(), "500.00");
  await amount.fill("400");
  await tax.check();
  assert.match(normalize(await dialog.getByRole("alert").innerText()), /CAD 100.00 per period/);
  await acknowledgement.check();
  const recurring = await print();
  await recurring.waitForFunction(() => window.printCalls === 1);
  assert.match(
    normalize(await recurring.locator("article").innerText()),
    /CAD 400.00 plus tax \/ Monthly/,
  );
  assert.doesNotMatch(
    await recurring.locator("article").innerText(),
    /Maximum Authorized Total|6,000/,
  );
  await recurring.close();

  mode = "foreign";
  await open();
  assert.equal(await amount.inputValue(), "100.10");
  await amount.fill("100");
  await shipping.check();
  assert.match(normalize(await dialog.getByRole("alert").innerText()), /USD 0.10/);
  await dialog.getByRole("button", { name: "Cancel" }).click();

  mode = "oneTime";
  for (const query of [
    "?amount=8000",
    "?amount=10000.01&plusTax=0&plusShipping=0&limit=10000&acknowledged=0",
    "?amount=8000&plusTax=1&plusShipping=0&limit=10000&acknowledged=0",
  ]) {
    await page.goto(`${origin}/pos/printpo/print${query}`);
    await page.getByRole("alert").waitFor();
    assert.equal(await page.locator("article").count(), 0);
    assert.equal(await page.evaluate(() => window.printCalls), 0);
  }
  mode = "closed";
  await page.goto(`${origin}/pos/printpo/print`);
  await page.getByRole("alert").waitFor();
  assert.match(await page.getByRole("alert").innerText(), /^404:/);
  mode = "oneTime";
  await page.goto(`${origin}/pos/printpo/print`);
  await page.waitForFunction(() => window.printCalls === 1);
  assert.match(normalize(await page.locator("article").innerText()), /CAD 10,000.00/);
  assert.ok(methods.length > 0);
  assert.deepEqual([...new Set(methods)], ["GET"]);
  assert.deepEqual(errors, []);
});
console.log("PO print browser checks passed.");
