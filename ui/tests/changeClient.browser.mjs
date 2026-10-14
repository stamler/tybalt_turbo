import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

// The Change Client dialog keeps Change Client disabled until it knows the
// project's subjobs and the user has confirmed them, and sends the subjobs it
// confirmed so the server can refuse a stale list.
await runBrowserHarness(async (page, origin) => {
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  let subjobs = [{ id: "sub1", number: "26-1001-01" }];
  let subjobLoads = 0;
  let failSubjobs = false;
  let holdSubjobs = null;
  let refuseNext = false;
  const posts = [];
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const send = (body, status = 200) =>
      route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
    const list = (items) =>
      send({ page: 1, perPage: 500, totalItems: items.length, totalPages: 1, items });
    if (path === "/api/realtime") return route.abort();
    if (path === "/api/clients")
      return send([
        { id: "oldclient", name: "Old Co", alias: "", contacts: [] },
        { id: "newclient", name: "New Co", alias: "", contacts: [] },
      ]);
    if (path === "/api/collections/jobs/records") {
      subjobLoads++;
      if (failSubjobs) return send({ message: "Jobs are unavailable" }, 500);
      if (holdSubjobs) {
        const rows = subjobs;
        holdSubjobs(() => list(rows));
        return;
      }
      return list(subjobs);
    }
    if (path === "/api/collections/client_contacts/records")
      return list([{ id: "contact1", given_name: "Pat", surname: "Lee", client: "newclient" }]);
    if (path === "/api/collections/client_invoicing_information/records")
      return list([{ id: "profile1", name: "Accounts", contact: "contact1", client: "newclient" }]);
    if (path === "/api/jobs/project1/change-client" && request.method() === "POST") {
      posts.push(request.postDataJSON());
      if (refuseNext) {
        refuseNext = false;
        return send(
          {
            status: 409,
            message: "The subjobs confirmed do not match this project's subjobs.",
            data: {
              subjobs: {
                code: "unconfirmed",
                message: "The subjobs confirmed do not match this project's subjobs.",
              },
            },
          },
          409,
        );
      }
      return send({ client: "newclient" });
    }
    return send({});
  });

  const changeButton = () => page.getByRole("button", { name: "Change Client", exact: true });
  const confirmation = () => page.getByRole("checkbox");
  const changedSelections = () => page.evaluate(() => window.changed);
  async function fillForm() {
    await page.getByPlaceholder("New client").fill("New");
    await page.getByRole("button", { name: "New Co", exact: true }).click();
    await page.locator("select").first().selectOption("contact1");
    await page.locator("select").nth(1).selectOption("profile1");
    await page.locator("textarea").fill("Entered under the architect instead of the owner.");
  }

  // While the subjobs load, Change Client is disabled.
  let release;
  holdSubjobs = (respond) => (release = respond);
  await page.goto(`${origin}/tests/fixtures/changeClient.html`);
  await page.getByText("Checking for subjobs…").waitFor();
  assert.equal(await changeButton().isDisabled(), true, "disabled while subjobs load");
  holdSubjobs = null;
  while (!release) await new Promise((resolve) => setTimeout(resolve, 20));
  await release();
  await page
    .getByText("I understand that the client will also be changed for 26-1001-01.")
    .waitFor();
  await fillForm();
  assert.equal(await changeButton().isDisabled(), true, "disabled until confirmed");
  await confirmation().check();
  assert.equal(await changeButton().isDisabled(), false, "enabled once confirmed");
  await changeButton().click();
  await page.waitForFunction(() => window.changed.length === 1);
  assert.deepEqual(posts.at(-1).subjobs, ["sub1"]);
  assert.deepEqual(await changedSelections(), [
    { client: "newclient", contact: "contact1", invoicing_information: "profile1" },
  ]);

  // A failed load says so, keeps Change Client disabled, and can be retried.
  failSubjobs = true;
  await page.goto(`${origin}/tests/fixtures/changeClient.html`);
  await page.getByText("Could not load this project's subjobs.").waitFor();
  assert.equal(await changeButton().isDisabled(), true, "disabled after a failed load");
  failSubjobs = false;
  await page.getByRole("button", { name: "Try again", exact: true }).click();
  await confirmation().waitFor();

  // When the subjobs changed after they were loaded, the server refuses, and
  // the dialog shows the current list unconfirmed instead of the stale one.
  await fillForm();
  await confirmation().check();
  subjobs = [...subjobs, { id: "sub2", number: "26-1001-02" }];
  refuseNext = true;
  await changeButton().click();
  await page
    .getByText("I understand that the client will also be changed for 26-1001-01, 26-1001-02.")
    .waitFor();
  assert.equal(await confirmation().isChecked(), false, "a reloaded list is not confirmed");
  assert.equal(await changeButton().isDisabled(), true);
  await page.getByText("The subjobs confirmed do not match this project's subjobs.").waitFor();
  await confirmation().check();
  await changeButton().click();
  await page.waitForFunction(() => window.changed.length === 1);
  assert.deepEqual(posts.at(-1).subjobs, ["sub1", "sub2"]);

  // Subjobs the opener already has are used without loading them.
  const loads = subjobLoads;
  await page.goto(`${origin}/tests/fixtures/changeClient.html?known`);
  await confirmation().waitFor();
  assert.equal(subjobLoads, loads, "known subjobs are not loaded again");

  // A project without subjobs needs no confirmation.
  subjobs = [];
  await page.goto(`${origin}/tests/fixtures/changeClient.html`);
  await page.getByText("Reason for the change").waitFor();
  await page.waitForFunction(() => !document.body.textContent.includes("Checking for subjobs"));
  assert.equal(await confirmation().count(), 0);
  assert.equal(await changeButton().isDisabled(), false);
  await fillForm();
  await changeButton().click();
  await page.waitForFunction(() => window.changed.length === 1);
  assert.deepEqual(posts.at(-1).subjobs, []);

  assert.deepEqual(errors, []);
});
console.log("Change client: subjob loading, confirmation, retry, and stale-list refusal checked.");
