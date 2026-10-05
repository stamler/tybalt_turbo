import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import vm from "node:vm";
import ts from "typescript";

async function editorModules({
  claims = ["job"],
  fail,
  client = {
    id: "client1",
    contacts: [{ id: "contact1", client: "client1" }],
    invoicing_profiles: [{ id: "profile1", client: "client1" }],
  },
} = {}) {
  const calls = [];
  const context = vm.createContext({ URL, URLSearchParams });
  const modules = new Map();
  const values = {
    "@sveltejs/kit": {
      error: (status, message) => {
        throw Object.assign(new Error(message), { status });
      },
    },
    "$lib/pocketbase": {
      pb: {
        filter: (_expression, values) => `client=${values.client}`,
        collection: (name) => ({
          getFullList: async (options) => {
            calls.push({ collection: name, filter: options.filter });
            return client.contacts;
          },
        }),
        send: async (path) => {
          calls.push(path);
          if (fail) throw fail;
          return client;
        },
      },
    },
    "svelte/store": { get: () => ({ claims }) },
    "$lib/stores/global": {
      globalStore: {
        ensureClaims: async () => {
          calls.push("claims");
        },
      },
    },
  };
  for (const [name, exports] of Object.entries(values)) {
    modules.set(
      name,
      new vm.SyntheticModule(
        Object.keys(exports),
        function () {
          for (const [key, value] of Object.entries(exports)) this.setExport(key, value);
        },
        { context },
      ),
    );
  }
  for (const [name, path] of [
    ["$lib/clientWorkspace", "../src/lib/clientWorkspace.ts"],
    ["editors", "../src/lib/clientEditors.ts"],
    ["loader", "../src/lib/clientEditorLoad.ts"],
  ]) {
    const source = await readFile(new URL(path, import.meta.url), "utf8");
    const output = ts.transpileModule(source, {
      compilerOptions: { target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ESNext },
    }).outputText;
    modules.set(name, new vm.SourceTextModule(output, { context, identifier: name }));
  }
  for (const module of modules.values()) {
    if (module.status === "unlinked") await module.link((name) => modules.get(name));
    if (module.status !== "evaluated") await module.evaluate();
  }
  return { modules, calls };
}

test("editor loader requires job permission before requesting client data", async () => {
  const { modules, calls } = await editorModules({ claims: ["time"] });
  await assert.rejects(
    modules
      .get("loader")
      .namespace.loadClientEditor("client1", new URL("https://app.invalid/"), "contact"),
    { status: 403 },
  );
  assert.deepEqual(calls, ["claims"]);
});

test("editor loaders reject missing or different-client IDs without creating a record", async () => {
  const { modules } = await editorModules();
  const load = modules.get("loader").namespace.loadClientEditor;
  for (const kind of ["contact", "invoicing"]) {
    await assert.rejects(
      load("client1", new URL("https://app.invalid/"), kind, "other-client-record"),
      { status: 404 },
    );
  }
  const failure = Object.assign(new Error("Forbidden"), { status: 403 });
  const failed = await editorModules({ fail: failure });
  await assert.rejects(
    failed.modules
      .get("loader")
      .namespace.loadClientEditor("client1", new URL("https://app.invalid/"), "contact"),
    (error) => error.status === 403,
  );
});

test("editor loader retains safe workspace context and validates project contact", async () => {
  const { modules } = await editorModules();
  const load = modules.get("loader").namespace.loadClientEditor;
  const url = new URL("https://app.invalid/");
  url.searchParams.set(
    "return_to",
    "/clients/client1/details?contacts_q=Pat&contacts_page=3#contacts",
  );
  url.searchParams.set("project_contact", "contact1");
  const result = await load("client1", url, "invoicing", "profile1");
  assert.equal(result.record.id, "profile1");
  assert.equal(result.projectContact, "contact1");
  assert.equal(result.returnTo, "/clients/client1/details?contacts_q=Pat&contacts_page=3#contacts");
  url.searchParams.set("project_contact", "foreign");
  url.searchParams.set("return_to", "https://evil.invalid");
  const safe = await load("client1", url, "contact");
  assert.equal(safe.projectContact, "");
  assert.equal(safe.returnTo, "/clients/client1/details");
});

test("contact validation and editable fields prevent accidental record ownership changes", async () => {
  const { modules } = await editorModules();
  const helpers = modules.get("editors").namespace;
  assert.deepEqual(Object.keys(helpers.validateContact({})), ["given_name", "surname"]);
  assert.deepEqual(
    Object.keys(helpers.validateContact({ given_name: "Pat", surname: "Lee", email: "" })),
    [],
  );
  assert.ok(helpers.validateContact({ given_name: "Pat", surname: "Lee", email: "bad" }).email);
  assert.equal(
    Object.keys(
      helpers.validateContact({ given_name: "Pat", surname: "Lee", email: "pat@example.com" }),
    ).length,
    0,
  );
  const body = helpers.editableFields(
    { given_name: "Pat", id: "private", client: "foreign", job_count: 2 },
    helpers.contactFields,
  );
  assert.equal(body.given_name, "Pat");
  assert.equal("client" in body, false);
  assert.equal("id" in body, false);
  assert.equal("job_count" in body, false);
  assert.equal(helpers.contactInUse({ profile_count: 1 }), true);
  assert.equal(helpers.contactInUse({ job_count: 0, profile_count: 0 }), false);
});

test("saved record links retain list filters and replace stale save banners", async () => {
  const { modules } = await editorModules();
  const result = modules
    .get("editors")
    .namespace.savedReturnTo(
      "/clients/client1/details?contacts_q=Pat&contacts_page=3&saved_invoicing=old&edit=client",
      "contact",
      "contact2",
    );
  const url = new URL(result, "https://app.invalid");
  assert.equal(url.searchParams.get("contacts_q"), "Pat");
  assert.equal(url.searchParams.get("contacts_page"), "3");
  assert.equal(url.searchParams.get("saved_contact"), "contact2");
  assert.equal(url.searchParams.has("saved_invoicing"), false);
  assert.equal(url.searchParams.has("edit"), false);
  assert.equal(url.hash, "#contacts");
});

test("contact merge preserves absorb permission without granting job-only users access", async () => {
  const allowed = await editorModules({ claims: ["absorb"] });
  const result = await allowed.modules
    .get("loader")
    .namespace.loadClientMerge("client1", "contact1");
  assert.equal(result.client.id, "client1");
  assert.equal(result.contacts[0].id, "contact1");
  assert.deepEqual(allowed.calls, [
    "claims",
    "/api/clients/client1",
    { collection: "client_contacts", filter: "client=client1" },
  ]);
  const forbidden = await editorModules({ claims: ["job"] });
  await assert.rejects(
    forbidden.modules.get("loader").namespace.loadClientMerge("client1", "contact1"),
    { status: 403 },
  );
  assert.deepEqual(forbidden.calls, ["claims"]);
  await assert.rejects(
    allowed.modules.get("loader").namespace.loadClientMerge("client1", "foreign-contact"),
    { status: 404 },
  );
});

test("missing client API status remains a route 404", async () => {
  const missing = await editorModules({
    fail: Object.assign(new Error("Missing"), { status: 404 }),
  });
  await assert.rejects(
    missing.modules
      .get("loader")
      .namespace.loadClientEditor("missing", new URL("https://app.invalid/"), "contact"),
    { status: 404, message: "Client not found." },
  );
});

test("time-only users cannot create or edit client records", async () => {
  const { modules } = await editorModules({ claims: ["time"] });
  const load = modules.get("loader").namespace.loadClientEditor;
  for (const [kind, id] of [
    ["invoicing", undefined],
    ["invoicing", "profile1"],
    ["contact", "contact1"],
    ["contact", undefined],
  ]) {
    await assert.rejects(load("client1", new URL("https://app.invalid/"), kind, id), {
      status: 403,
    });
  }
});

test("profile creation rejects absent permissions and inactive-client API failures", async () => {
  const denied = await editorModules({ claims: [] });
  await assert.rejects(
    denied.modules
      .get("loader")
      .namespace.loadClientEditor("client1", new URL("https://app.invalid/"), "invoicing"),
    { status: 403 },
  );
  assert.deepEqual(denied.calls, ["claims"]);
  const inactive = await editorModules({
    claims: ["job"],
    fail: Object.assign(new Error("Active account required"), { status: 403 }),
  });
  await assert.rejects(
    inactive.modules
      .get("loader")
      .namespace.loadClientEditor("client1", new URL("https://app.invalid/"), "invoicing"),
    { status: 403 },
  );
});

test("contact merge retains only a safe same-client workspace return", async () => {
  const allowed = await editorModules({ claims: ["absorb"] });
  const load = allowed.modules.get("loader").namespace.loadClientMerge;
  const returnTo =
    "/clients/client1/details?contacts_q=Robin&workflow_return=%2Fjobs%2Fadd%3Fclient_draft%3Dtoken";
  const url = new URL("https://app.invalid/clients/client1/contacts/contact1/absorb");
  url.searchParams.set("return_to", returnTo);
  assert.equal((await load("client1", "contact1", url)).returnTo, returnTo);
  for (const unsafe of [
    "https://other.test/",
    "/clients/other/details",
    "/clients/client1/details/../edit",
  ]) {
    url.searchParams.set("return_to", unsafe);
    assert.equal((await load("client1", "contact1", url)).returnTo, "/clients/client1/details");
  }
  assert.equal((await load("client1", "contact1")).returnTo, undefined);
});
