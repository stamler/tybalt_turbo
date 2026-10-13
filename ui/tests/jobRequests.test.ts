import test from "node:test";
import assert from "node:assert/strict";
import {
  canViewJobCreateQueue,
  printMissing,
  contactEmail,
  billingAddress,
  wholeCAD,
  emptyChecklist,
  requestChecklist,
  requestChecklistVersion,
  type RequestDetail,
} from "../src/lib/jobRequests.ts";
const complete = (): RequestDetail => ({
  request: {
    id: "request1",
    version: 2,
    description: "Field inspection\nPrepare a report",
    location: "86VHQPRM+5C",
    project_value: 120000,
    time_and_materials: false,
    time_and_materials_set: true,
  },
  allocations: [],
  review_token: "",
  permissions: {},
  related: {
    client: {
      id: "client1",
      name: "Example Client",
      address: "1 Main Street",
      city: "Thunder Bay",
      province_state: "ON",
    },
    contact: { given_name: "Alex", surname: "Client", email: "alex@example.com" },
    billing_contact: { given_name: "Pat", surname: "Billing", email: "pat@example.com" },
    rate_sheet: {
      id: "rate1",
      active: true,
      name: "Standard",
      revision: 3,
      effective_date: "2026-01-01",
    },
  },
});
test("complete unsigned draft is ready before internal approval, PDF and number", () => {
  const d = complete();
  assert.deepEqual(printMissing(d), []);
  assert.equal(d.attachment, undefined);
  assert.equal(d.request.number, undefined);
});
test("optional contacts, owner, references and addresses do not block signature readiness", () => {
  const d = complete();
  delete d.related.client.address;
  delete d.related.client.city;
  delete d.related.client.province_state;
  assert.deepEqual(printMissing(d), []);
  d.related.billing_contact = d.related.contact;
  assert.deepEqual(printMissing(d), []);
});
test("each required client value is independently checked", () => {
  const mutations: ((d: RequestDetail) => void)[] = [
    (d) => (d.related.client.name = ""),
    (d) => (d.request.description = " "),
    (d) => (d.request.location = "bad"),
    (d) => (d.related.contact.given_name = d.related.contact.surname = ""),
    (d) => (d.related.contact.email = "invalid"),
    (d) => (d.related.billing_contact.given_name = d.related.billing_contact.surname = ""),
    (d) => (d.related.billing_contact.email = "invalid"),
    (d) => (d.request.project_value = 0),
    (d) => (d.request.project_value = 1.5),
    (d) => (d.request.time_and_materials_set = false),
    (d) => delete d.related.rate_sheet,
  ];
  for (const mutate of mutations) {
    const d = complete();
    mutate(d);
    assert.equal(printMissing(d).length, 1);
  }
});
test("a contact marked as having no email address is ready for signature", () => {
  const d = complete();
  d.related.contact.email = "";
  assert.deepEqual(printMissing(d), ["Project contact email"]);
  d.related.contact.no_email = true;
  assert.deepEqual(printMissing(d), []);
  assert.equal(contactEmail(d.related.contact), "No email address");
  assert.equal(contactEmail(d.related.billing_contact), d.related.billing_contact.email);
});
test("T&M requires matching available billing rates", () => {
  const d = complete();
  d.request.time_and_materials = true;
  assert.deepEqual(printMissing(d), ["Client billing rates"]);
  d.related.rate_sheet_entries = [{ role_name: "Engineer", rate: 180, overtime_rate: 270 }];
  assert.deepEqual(printMissing(d), []);
  d.related.rate_sheet_entries[0].rate = undefined;
  assert.deepEqual(printMissing(d), ["Client billing rates"]);
});
test("billing address selects one complete source without mixing partial addresses", () => {
  const client = { address: "1 Main Street", city: "Thunder Bay", province_state: "ON" };
  assert.equal(billingAddress({}, client), "1 Main Street, Thunder Bay, ON");
  assert.equal(billingAddress({ city: "Ottawa" }, client), "Ottawa");
});
test("unknown and fractional values are not printed as zero or rounded", () => {
  for (const value of [undefined, null, "120000", 0, -1, 10.2])
    assert.equal(wholeCAD(value), "Not provided");
  assert.equal(wholeCAD(120000), "$120,000");
});
test("all seven named confirmations start unchecked on each review", () => {
  const first = emptyChecklist();
  assert.equal(requestChecklistVersion, 2);
  assert.equal(Object.keys(first).length, 7);
  assert.equal(first.rate_sheet_correct, false);
  assert.deepEqual(Object.keys(first), Object.keys(requestChecklist));
  assert.ok(Object.values(first).every((value) => value === false));
  first.pdf_sufficient = true;
  assert.equal(emptyChecklist().pdf_sufficient, false);
});

test("creation queue navigation requires current server eligibility even when count is zero", () => {
  const counts = { "/job-requests?queue=creation": 0 };
  assert.equal(canViewJobCreateQueue(["job"], true, counts), true);
  assert.equal(canViewJobCreateQueue(["job"], true, {}), false);
  assert.equal(canViewJobCreateQueue(["job"], false, counts), false);
  assert.equal(canViewJobCreateQueue(["admin"], true, counts), false);
});
