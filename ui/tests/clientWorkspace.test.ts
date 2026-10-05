import test from "node:test";
import assert from "node:assert/strict";
import {
  addressText,
  invoicingEmptyMessage,
  profileCreatorName,
  contactName,
  contactSearchText,
  invoicingSearchText,
  listPageSize,
  parseClientWorkspaceQuery,
  positivePage,
  safeReturnTo,
  savedListPosition,
  workspaceHref,
  type WorkspaceContact,
  type WorkspaceInvoicing,
} from "../src/lib/clientWorkspace.ts";

const query = (value = "") => new URLSearchParams(value);
test("workspace tabs have separate routes and job views use explicit filters", () => {
  for (const tab of ["details", "jobs", "notes"] as const) {
    const target = new URL(
      workspaceHref("client1", query("contacts_q=Pat"), {}, tab),
      "https://test.invalid",
    );
    assert.equal(target.pathname, `/clients/client1/${tab}`);
    assert.equal(target.searchParams.get("contacts_q"), "Pat");
    assert.equal(target.searchParams.has("tab"), false);
  }
  for (const view of ["projects", "proposals", "owner"]) {
    assert.equal(parseClientWorkspaceQuery(query(`view=${view}`)).view, view);
  }
  assert.equal(parseClientWorkspaceQuery(query()).view, "projects");
  assert.equal(parseClientWorkspaceQuery(query("view=unknown")).view, "projects");
});
test("invalid and unbounded list parameters use safe defaults", () => {
  for (const value of [null, "", "0", "-1", "NaN", "Infinity", "1.5", "9007199254740992"]) {
    assert.equal(positivePage(value), 1);
  }
  assert.equal(positivePage("5"), 5);
  for (const size of ["10", "20", "50"]) assert.equal(listPageSize(size), Number(size));
  for (const value of [null, "0", "100", "-20", "NaN"]) assert.equal(listPageSize(value), 20);
});

test("saved records are located by ID within the current filter and page", () => {
  const items = Array.from({ length: 65 }, (_, index) => ({
    id: `record-${index + 1}`,
    name: index % 2 === 0 ? "Alex Lee" : "Pat Lee",
  }));
  const searchText = (item: (typeof items)[number]) => item.name;
  const locate = (id: string | undefined, query = "", page = 1, perPage = 20) =>
    savedListPosition(items, id, query, searchText, page, perPage);

  assert.deepEqual(locate("record-1"), { visible: true, page: 1, clearSearch: false });
  assert.deepEqual(locate("record-41"), { visible: false, page: 3, clearSearch: false });
  assert.deepEqual(locate("record-41", "", 3), { visible: true, page: 3, clearSearch: false });
  assert.deepEqual(locate("record-41", "aLeX", 1), {
    visible: false,
    page: 2,
    clearSearch: false,
  });
  assert.deepEqual(locate("record-41", "ALEX", 2), {
    visible: true,
    page: 2,
    clearSearch: false,
  });
  assert.deepEqual(locate("record-41", "Pat", 2), {
    visible: false,
    page: 3,
    clearSearch: true,
  });
  assert.deepEqual(locate("record-41", "No matches", 1), {
    visible: false,
    page: 3,
    clearSearch: true,
  });
  // DSList does not trim the query, so spaces remain part of the filter.
  assert.equal(locate("record-41", " Alex")?.clearSearch, true);
  assert.deepEqual(locate("record-65", "", 999), {
    visible: true,
    page: 4,
    clearSearch: false,
  });
  assert.deepEqual(locate("record-65", "Alex", 999), {
    visible: true,
    page: 2,
    clearSearch: false,
  });
  for (const page of [0, -1, 1.5, Number.NaN, Number.POSITIVE_INFINITY, 2 ** 53])
    assert.deepEqual(locate("record-1", "", page), {
      visible: true,
      page: 1,
      clearSearch: false,
    });
  for (const perPage of [0, -1, 15, Number.NaN, Number.POSITIVE_INFINITY])
    assert.deepEqual(locate("record-41", "", 3, perPage), {
      visible: true,
      page: 3,
      clearSearch: false,
    });
  assert.deepEqual(locate("record-41", "", 5, 10), {
    visible: true,
    page: 5,
    clearSearch: false,
  });
  assert.deepEqual(locate("record-41", "", 1, 50), {
    visible: true,
    page: 1,
    clearSearch: false,
  });
  assert.equal(locate("missing"), null);
  assert.equal(locate(undefined), null);
  assert.equal(locate(""), null);
  assert.equal(savedListPosition([], "missing", "", searchText, 1, 20), null);
});
test("list state remains independent and editor return URLs retain it", () => {
  const original = query(
    "contacts_q=Alex&contacts_page=2&contacts_per_page=10&invoicing_q=capital&invoicing_page=3&invoicing_per_page=50&workflow_return=%2Fjobs%2Fadd",
  );
  const next = workspaceHref("client1", original, { contacts_q: "Pat & Jo", contacts_page: 1 });
  const state = parseClientWorkspaceQuery(new URL(next, "https://test.invalid").searchParams);
  assert.equal(state.contactsQuery, "Pat & Jo");
  assert.equal(state.contactsPage, 1);
  assert.equal(state.contactsPerPage, 10);
  assert.equal(state.invoicingQuery, "capital");
  assert.equal(state.invoicingPage, 3);
  assert.equal(state.invoicingPerPage, 50);
  assert.equal(original.get("contacts_q"), "Alex");
  assert.equal(safeReturnTo("client1", next), next);
  const afterSave = workspaceHref("client1", new URL(next, "https://test.invalid").searchParams, {
    saved_contact: "new-contact",
    edit: null,
  });
  assert.match(afterSave, /saved_contact=new-contact/);
  assert.match(afterSave, /invoicing_page=3/);
  assert.match(afterSave, /workflow_return=/);
});
test("return URLs cannot leave the same client workspace", () => {
  for (const value of [
    undefined,
    null,
    "",
    "https://evil.invalid/clients/client1/details",
    "//evil.invalid",
    "/clients/client2/details",
    "/clients/client1/details/../edit",
    "/clients/client1/details-more",
    "/jobs/add",
  ]) {
    assert.equal(safeReturnTo("client1", value), "/clients/client1/details");
  }
  assert.equal(
    safeReturnTo("client1", "/clients/client1/details?contacts_page=3#invoicing"),
    "/clients/client1/details?contacts_page=3#invoicing",
  );
});
test("contact and invoicing searches include the intended human-readable fields", () => {
  const contact: WorkspaceContact = {
    id: "contact1",
    client: "client1",
    given_name: "Alex",
    surname: "Lee",
    email: "accounts@example.test",
    phone: "555-0100",
    job_count: 3,
    profile_count: 2,
  };
  const profile: WorkspaceInvoicing = {
    id: "profile1",
    client: "client1",
    name: "Capital projects",
    contact: contact.id,
    fax: "",
    invoicing_instructions: "Include purchase order and send PDF",
    job_count: 3,
  };
  for (const term of ["Alex", "Lee", "accounts@example.test", "555-0100"])
    assert.ok(contactSearchText(contact).includes(term));
  for (const term of ["Capital", "purchase order", "Alex", "accounts@example.test"])
    assert.ok(invoicingSearchText(profile, [contact]).includes(term));
  assert.equal(
    invoicingSearchText(profile, []),
    "Capital projects Include purchase order and send PDF",
  );
  assert.equal(contactName(undefined), "Contact unavailable");
  assert.equal(
    addressText({ address: "1 Main", city: "Thunder Bay", country: "Canada" }),
    "1 Main, Thunder Bay, Canada",
  );
});

test("shallow URL state survives Back without leaking between clients or editor routes", async () => {
  const { currentWorkspaceUrl } = await import("../src/lib/clientWorkspace.ts");
  const base = new URL("https://test.invalid/clients/client1/details");
  const shallow = {
    clientWorkspaceUrl: "/clients/client1/details?contacts_page=4&invoicing_q=capital",
  };
  assert.equal(
    currentWorkspaceUrl("client1", base, shallow).searchParams.get("contacts_page"),
    "4",
  );
  assert.equal(currentWorkspaceUrl("client1", base, {}).href, base.href);
  const other = new URL("https://test.invalid/clients/client2/details");
  assert.equal(currentWorkspaceUrl("client1", other, shallow).href, other.href);
  const editor = new URL("https://test.invalid/clients/client1/contacts/add");
  assert.equal(currentWorkspaceUrl("client1", editor, shallow).href, editor.href);
});

test("empty profile lists point each user to an available next step", () => {
  assert.match(invoicingEmptyMessage(true, true), /Create one using an existing client contact/);
  assert.match(invoicingEmptyMessage(true, false), /Add a client contact, then create a profile/);
  for (const hasContacts of [true, false]) {
    assert.match(
      invoicingEmptyMessage(false, hasContacts),
      /Ask a client maintainer to create one/,
    );
  }
});

test("creator display uses recorded attribution without inventing legacy identity", () => {
  assert.equal(profileCreatorName({ creator: "user1", creator_name: " Alex Lee " }), "Alex Lee");
  assert.equal(
    profileCreatorName({ creator: "user1", creator_name: "alex@example.test" }),
    "alex@example.test",
  );
  assert.equal(profileCreatorName({}), "");
  assert.equal(profileCreatorName({ creator: "", creator_name: "Unverified name" }), "");
  assert.equal(profileCreatorName({ creator: "deleted-user", creator_name: "" }), "");
});
