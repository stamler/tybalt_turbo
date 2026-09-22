// Exercise real components and styles with only framework services stubbed.
// Set PLAYWRIGHT_MODULE and PLAYWRIGHT_CHANNEL for an external installation.
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { createServer } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
const require = createRequire(import.meta.url);
const root = fileURLToPath(new URL("../", import.meta.url));
const services = {
  "$app/paths": "export const resolve = (route, params) => route.replace('[id]', params.id);",
  "$app/navigation": "export const goto = (url) => { location.href = url; };",
  "$env/static/public": "export const PUBLIC_POCKETBASE_URL = location.origin;",
  "summary-test-global":
    "import { writable } from 'svelte/store'; export const globalStore = writable({claims:[]});",
};

export async function runBrowserHarness(check) {
  const { chromium } = require(process.env.PLAYWRIGHT_MODULE || "playwright");
  const cacheDir = await mkdtemp(path.join(tmpdir(), "tybalt-browser-vite-"));
  let server, browser;
  try {
    server = await createServer({
      root,
      configFile: false,
      cacheDir,
      plugins: [
        {
          name: "test-services",
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
    await server.listen();
    browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL });
    const page = await browser.newPage({ viewport: { width: 1000, height: 900 } });
    await check(page, `http://127.0.0.1:${server.httpServer.address().port}`);
  } finally {
    await browser?.close();
    await server?.close();
    await rm(cacheDir, { recursive: true, force: true });
  }
}
