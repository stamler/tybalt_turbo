import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";
import ts from "typescript";
import { get, writable } from "svelte/store";

// Run the real loader and store with controlled API timing and no browser server.
async function requestPage() {
  let finishClaims;
  const summary = new Promise((resolve) => {
    finishClaims = resolve;
  });
  let summaryCalls = 0;
  const pb = {
    send: async () => ({ enabled: true }),
    filter: () => "uid='creator'",
    collection: (name) => ({
      getOne: async () => {
        if (name !== "user_claims_summary") return {};
        summaryCalls++;
        return summary;
      },
      getFullList: async () => [],
    }),
  };
  const context = vm.createContext({ console });
  const modules = new Map();
  for (const [name, values] of [
    ["svelte/store", { get, writable }],
    ["$lib/pocketbase", { pb }],
    ["$lib/stores/auth", { authStore: writable({ isValid: true, model: { id: "creator" } }) }],
    [
      "@sveltejs/kit",
      {
        error: (status, message) => {
          throw Object.assign(new Error(message), { status });
        },
      },
    ],
  ]) {
    modules.set(
      name,
      new vm.SyntheticModule(
        Object.keys(values),
        function () {
          for (const [key, value] of Object.entries(values)) this.setExport(key, value);
        },
        { context },
      ),
    );
  }
  for (const [name, path] of [
    ["$lib/stores/global", "../src/lib/stores/global.ts"],
    ["page", "../src/routes/job-requests/new/+page.ts"],
  ]) {
    const source = await readFile(new URL(path, import.meta.url), "utf8");
    const { outputText } = ts.transpileModule(source, {
      compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
    });
    modules.set(name, new vm.SourceTextModule(outputText, { context, identifier: name }));
  }
  const page = modules.get("page");
  await page.link((name) => {
    assert.ok(modules.has(name), `Unexpected dependency: ${name}`);
    return modules.get(name);
  });
  await page.evaluate();
  return {
    load: () => page.namespace.load({ url: new URL("https://example.test/job-requests/new") }),
    store: modules.get("$lib/stores/global").namespace.globalStore,
    finish: (claims) => finishClaims({ id: "creator", claims }),
    calls: () => summaryCalls,
  };
}

for (const backgroundRefresh of [false, true]) {
  for (const permitted of [false, true]) {
    test(`waits for claims: permitted=${permitted}, background refresh=${backgroundRefresh}`, async () => {
      const page = await requestPage();
      if (backgroundRefresh) await page.store.refresh();
      let settled = false;
      const result = page.load().then(
        (value) => {
          settled = true;
          return { value };
        },
        (error) => {
          settled = true;
          return { error };
        },
      );
      await new Promise(setImmediate);
      assert.equal(settled, false, "permission must not use the initial empty claims list");
      assert.equal(page.calls(), 1, "reuse a claims request already in progress");
      page.finish(permitted ? ["time"] : ["job"]);
      const response = await result;
      if (permitted) {
        assert.equal(response.error, undefined);
        assert.equal(response.value.request.project_status, "Active");
      } else {
        assert.equal(response.error?.status, 403);
        assert.match(response.error.message, /time permission/);
      }
    });
  }
}
