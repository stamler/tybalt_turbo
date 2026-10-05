import assert from "node:assert/strict";
import test from "node:test";
import {
  safeWorkflowReturn,
  keepClientWorkflowDraft,
  readClientWorkflowDraft,
  clearClientWorkflowDraft,
  selectClientWorkflowProfile,
} from "../src/lib/clientWorkflowNavigation.ts";

// Mimic Storage: stored entries are its enumerable keys, as Object.keys(sessionStorage) expects.
function storageMock(storage: Map<string, string>, blocked = () => false) {
  const methods = {
    setItem: (key: string, value: string) => {
      if (blocked()) throw new Error("Storage is full");
      storage.set(key, value);
    },
    getItem: (key: string) => storage.get(key) ?? null,
    removeItem: (key: string) => storage.delete(key),
  };
  return new Proxy(methods, {
    ownKeys: () => [...storage.keys()],
    getOwnPropertyDescriptor: (_, key) =>
      storage.has(String(key))
        ? { configurable: true, enumerable: true, value: storage.get(String(key)) }
        : undefined,
  });
}

test("workflow returns accept only local job forms", () => {
  for (const path of [
    "/jobs/add",
    "/jobs/add/parent1",
    "/jobs/add/from/proposal1",
    "/jobs/job1/edit",
  ])
    assert.equal(safeWorkflowReturn(path), path);
  for (const path of [
    "https://example.com/jobs/add",
    "//example.com/jobs/add",
    "/\\example.com/jobs/add",
    "/clients/client1/details",
    "/jobs",
    "/jobs/job1/delete",
    "/jobs/job1/details",
    "/jobs/value-approvals",
    "/job-requests/new",
    "/jobs/%2fexample.com/edit",
    "/jobs/add\n",
    null,
    undefined,
  ])
    assert.equal(safeWorkflowReturn(path), null);
});

test("copied profiles change only the matching unfinished form and preserve its draft", () => {
  const storage = new Map<string, string>();
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const originalStorage = Object.getOwnPropertyDescriptor(globalThis, "sessionStorage");
  const location = { href: "https://app.example/jobs/job1/edit" };
  let storageBlocked = false;
  Object.defineProperty(globalThis, "window", { configurable: true, value: { location } });
  Object.defineProperty(globalThis, "sessionStorage", {
    configurable: true,
    value: storageMock(storage, () => storageBlocked),
  });
  try {
    for (const [source, kind, field] of [
      ["/jobs/add", "job:new", "item"],
      ["/jobs/add/parent1", "job:new", "item"],
      ["/jobs/add/from/proposal1", "job:new", "item"],
      ["/jobs/job1/edit", "job:job1", "item"],
    ]) {
      location.href = "https://app.example" + source;
      const item = {
        client: "client1",
        invoicing_information: "original",
        description: "Unsaved scope",
        version: 4,
      };
      const draft = {
        [field]: item,
        allocations: [{ division: "division1", hours: 12 }],
        jobUpdated: "2026-10-01 12:00:00",
        originalAllocations: "original allocations",
        saved: "original saved baseline",
        fee: "fixed",
      };
      const workspace = new URL(
        keepClientWorkflowDraft(kind, "user1", draft, "client1", "contact1"),
        "https://app.example",
      );
      const destination = workspace.searchParams.get("workflow_return")!;
      location.href = "https://app.example/clients/client1/invoicing/profile1/edit";
      assert.equal(selectClientWorkflowProfile(destination, "user2", "client1", "copy"), null);
      assert.equal(selectClientWorkflowProfile(destination, "user1", "client2", "copy"), null);
      assert.equal(selectClientWorkflowProfile(destination, "", "client1", "copy"), null);
      assert.equal(selectClientWorkflowProfile(destination, "user1", "client1", ""), null);
      const otherPath = new URL(destination, "https://app.example");
      otherPath.pathname = "/jobs/other/edit";
      assert.equal(
        selectClientWorkflowProfile(
          otherPath.pathname + otherPath.search,
          "user1",
          "client1",
          "copy",
        ),
        null,
      );
      assert.equal(
        selectClientWorkflowProfile(
          "https://other.example" + destination,
          "user1",
          "client1",
          "copy",
        ),
        null,
      );
      assert.equal(selectClientWorkflowProfile("/jobs/add", "user1", "client1", "copy"), null);
      const oldNow = Date.now;
      Date.now = () => oldNow() + 5 * 60 * 60 * 1000;
      try {
        assert.equal(selectClientWorkflowProfile(destination, "user1", "client1", "copy"), null);
      } finally {
        Date.now = oldNow;
      }
      storageBlocked = true;
      assert.equal(selectClientWorkflowProfile(destination, "user1", "client1", "copy"), null);
      storageBlocked = false;
      location.href = "https://app.example" + destination;
      assert.deepEqual(
        readClientWorkflowDraft(kind, "user1"),
        draft,
        "failed selection leaves all draft data intact",
      );
      assert.equal(
        selectClientWorkflowProfile(destination, "user1", "client1", "copy"),
        destination,
      );
      const restored = readClientWorkflowDraft<typeof draft>(kind, "user1")!;
      assert.deepEqual(restored, {
        ...draft,
        [field]: { ...item, invoicing_information: "copy" },
        selectedProfile: "copy",
      });
      assert.equal(item.invoicing_information, "original", "the saved baseline remains unchanged");
      const token = new URL(destination, "https://app.example").searchParams.get("client_draft")!;
      const reloadCopy = JSON.parse(storage.get("client-workspace-draft:" + token)!);
      assert.equal(reloadCopy.payload[field].invoicing_information, "copy");
      clearClientWorkflowDraft();
    }

    // A reloaded tab reads the JSON copy without requiring an in-memory draft.
    const token = crypto.randomUUID();
    const destination = `/jobs/add?client_draft=${token}`;
    const envelope = {
      kind: "job:new",
      user: "user1",
      path: "/jobs/add",
      created: Date.now(),
      payload: {
        item: { client: "client1", invoicing_information: "original" },
        saved: "baseline",
      },
    };
    storage.set(
      "client-workspace-draft:" + token,
      JSON.stringify({ ...envelope, kind: "job:other" }),
    );
    assert.equal(selectClientWorkflowProfile(destination, "user1", "client1", "copy"), null);
    storage.set("client-workspace-draft:" + token, JSON.stringify(envelope));
    assert.equal(selectClientWorkflowProfile(destination, "user1", "client1", "copy"), destination);
    location.href = "https://app.example" + destination;
    assert.deepEqual(readClientWorkflowDraft("job:new", "user1"), {
      item: { client: "client1", invoicing_information: "copy" },
      saved: "baseline",
      selectedProfile: "copy",
    });
    clearClientWorkflowDraft();
  } finally {
    if (originalWindow) Object.defineProperty(globalThis, "window", originalWindow);
    else Reflect.deleteProperty(globalThis, "window");
    if (originalStorage) Object.defineProperty(globalThis, "sessionStorage", originalStorage);
    else Reflect.deleteProperty(globalThis, "sessionStorage");
  }
});

test("drafts preserve values, require the same user and source, and clear after save", () => {
  const storage = new Map<string, string>();
  const originalWindow = Object.getOwnPropertyDescriptor(globalThis, "window");
  const originalStorage = Object.getOwnPropertyDescriptor(globalThis, "sessionStorage");
  const location = { href: "https://app.example/jobs/add?parent=parent1" };
  Object.defineProperty(globalThis, "window", { configurable: true, value: { location } });
  Object.defineProperty(globalThis, "sessionStorage", {
    configurable: true,
    value: storageMock(storage),
  });
  try {
    const draft = {
      item: { description: "Unsaved scope", version: 4 },
      allocations: [{ hours: 12 }],
    };
    let sourceEntry = "";
    const href = keepClientWorkflowDraft(
      "job:new",
      "user1",
      draft,
      "client1",
      "contact1",
      "workspace",
      (url) => {
        sourceEntry = url;
      },
    );
    const workspace = new URL(href, "https://app.example");
    assert.equal(workspace.pathname, "/clients/client1/details");
    assert.equal(workspace.searchParams.get("project_contact"), "contact1");
    const returnUrl = workspace.searchParams.get("workflow_return")!;
    assert.equal(
      sourceEntry,
      returnUrl,
      "browser Back and the explicit return use the same draft token",
    );
    assert.ok(safeWorkflowReturn(returnUrl));
    assert.match(returnUrl, /parent=parent1/);
    location.href = "https://app.example" + returnUrl;
    assert.deepEqual(readClientWorkflowDraft("job:new", "user1"), draft);
    assert.equal(readClientWorkflowDraft("job:new", "user2"), null);
    assert.equal(readClientWorkflowDraft("job:job1", "user1"), null);
    location.href = location.href.replace("/jobs/add", "/jobs/other/edit");
    assert.equal(readClientWorkflowDraft("job:new", "user1"), null);
    location.href = "https://app.example" + returnUrl;
    const oldNow = Date.now;
    Date.now = () => oldNow() + 5 * 60 * 60 * 1000;
    try {
      assert.equal(readClientWorkflowDraft("job:new", "user1"), null);
    } finally {
      Date.now = oldNow;
    }
    const second = new URL(
      keepClientWorkflowDraft("job:new", "user1", draft, "client1", ""),
      "https://app.example",
    );
    assert.equal(
      readClientWorkflowDraft("job:new", "user1"),
      null,
      "the earlier history entry cannot restore a superseded draft",
    );
    location.href = "https://app.example" + second.searchParams.get("workflow_return");
    assert.deepEqual(readClientWorkflowDraft("job:new", "user1"), draft);
    clearClientWorkflowDraft();
    assert.equal(readClientWorkflowDraft("job:new", "user1"), null);
    assert.equal(storage.size, 0);

    const addUrl = new URL(
      keepClientWorkflowDraft("job:new", "user1", draft, "client1", "", "contact"),
      "https://app.example",
    );
    assert.equal(addUrl.pathname, "/clients/client1/contacts/add");
    assert.match(addUrl.searchParams.get("return_to")!, /^\/clients\/client1\/details\?/);
    const profileUrl = new URL(
      keepClientWorkflowDraft("job:new", "user1", draft, "client1", "contact1", "invoicing"),
      "https://app.example",
    );
    assert.equal(profileUrl.pathname, "/clients/client1/invoicing/add");
    assert.equal(profileUrl.searchParams.get("project_contact"), "contact1");
    const profileReturn = new URL(profileUrl.searchParams.get("return_to")!, "https://app.example");
    assert.equal(profileReturn.searchParams.get("project_contact"), "contact1");
    location.href = "https://app.example" + profileReturn.searchParams.get("workflow_return");
    assert.deepEqual(readClientWorkflowDraft("job:new", "user1"), draft);
  } finally {
    if (originalWindow) Object.defineProperty(globalThis, "window", originalWindow);
    else Reflect.deleteProperty(globalThis, "window");
    if (originalStorage) Object.defineProperty(globalThis, "sessionStorage", originalStorage);
    else Reflect.deleteProperty(globalThis, "sessionStorage");
  }
});
