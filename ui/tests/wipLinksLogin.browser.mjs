import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

// Use the real SvelteKit layout and login page. No credentials or backend are needed.
await runBrowserHarness(
  async (page, origin) => {
    await page.route("**/api/**", (route) =>
      route.fulfill({
        status: 401,
        contentType: "application/json",
        body: '{"message":"Not authenticated"}',
      }),
    );
    for (const target of [
      "/jobs/wip/branch#branch=north",
      "/jobs/wip/division#division=structural",
      "/jobs/wip/branch#branch=",
      "/jobs/wip/division?source=link#division=",
      "/jobs/wip/my",
    ]) {
      await page.goto(`${origin}${target}`);
      await page.waitForURL(
        (url) => url.pathname === "/login" && url.searchParams.get("redirect") === target,
      );
      await page.waitForFunction(
        (expected) => sessionStorage.getItem("redirectUrl") === expected,
        target,
      );
      assert.equal(new URL(page.url()).searchParams.get("redirect"), target);
      assert.equal(await page.evaluate(() => sessionStorage.getItem("redirectUrl")), target);
    }
    console.log(
      "WIP login links passed: branch, division, empty selections, query strings, and default route.",
    );
  },
  { app: true },
);
