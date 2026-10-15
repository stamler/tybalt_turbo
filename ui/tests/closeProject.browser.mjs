import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

// "Close project" is in the job page's "⋯" menu for every Active project the
// user may close, imported or not. Its confirmation says what will happen to
// the referenced proposal and is disabled when the server would refuse.
await runBrowserHarness(async (page, origin) => {
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  let proposal = null;
  const closes = [];
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const send = (body, status = 200) =>
      route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
    if (path === "/api/realtime") return route.abort();
    if (path === "/api/jobs/proposal1") return send(proposal);
    if (path === "/api/jobs/project1/close" && request.method() === "POST") {
      closes.push(path);
      return send({ id: "project1", status: "Closed", project_note_created: true });
    }
    return send({});
  });

  const dialog = () => page.getByRole("dialog");
  const submit = () => dialog().getByRole("button", { name: "Close project", exact: true });
  async function openClose(query = "") {
    await page.goto(`${origin}/tests/fixtures/closeProject.html${query}`);
    await page.getByRole("button", { name: "More actions" }).click();
    await page.getByRole("button", { name: "Close project", exact: true }).click();
    await dialog().getByText("Close 26-1001").waitFor();
  }

  // A project that was not imported, with no proposal, closes after confirming.
  await openClose();
  await dialog().getByText("Its billing details do not need to be complete.").waitFor();
  assert.equal(await submit().isDisabled(), false);
  await submit().click();
  await page.waitForFunction(() => !document.querySelector("dialog[open]"));
  assert.deepEqual(closes, ["/api/jobs/project1/close"]);

  // An imported proposal that is still Submitted is awarded by the close.
  proposal = { id: "proposal1", number: "P26-0001", status: "Submitted", imported: true };
  await openClose("?proposal");
  await dialog().getByText("Proposal P26-0001 will be awarded").waitFor();
  assert.equal(await submit().isDisabled(), false);

  // An Awarded proposal needs no notice.
  proposal = { ...proposal, status: "Awarded", imported: false };
  await openClose("?proposal");
  await submit().waitFor();
  await page.waitForFunction(() => !document.body.textContent.includes("Checking the referenced"));
  assert.equal(await dialog().getByText("will be awarded").count(), 0);
  assert.equal(await submit().isDisabled(), false);

  // A proposal in any other state blocks the close, and the dialog says so.
  for (const blocked of [
    { status: "In Progress", imported: false },
    { status: "Not Awarded", imported: true },
  ]) {
    proposal = { ...proposal, ...blocked };
    await openClose("?proposal");
    await dialog().getByText("This project can't be closed yet").waitFor();
    await dialog().getByText(`Its proposal P26-0001 is ${blocked.status}.`).waitFor();
    assert.equal(await submit().isDisabled(), true, `${blocked.status} blocks the close`);
  }

  // The action is offered only for Active projects to holders of the job claim.
  for (const query of ["?status=Closed", "?noclaim"]) {
    await page.goto(`${origin}/tests/fixtures/closeProject.html${query}`);
    await page.getByText("26-1001").first().waitFor();
    const more = page.getByRole("button", { name: "More actions" });
    if (await more.count()) await more.click();
    assert.equal(
      await page.getByRole("button", { name: "Close project", exact: true }).count(),
      0,
      `no Close project for ${query}`,
    );
  }

  assert.equal(closes.length, 1);
  assert.deepEqual(errors, []);
});
console.log("Close project: menu visibility, proposal outcomes, and the close request checked.");
