import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";
const readFixture = async (name) =>
  JSON.parse(await readFile(new URL(`./fixtures/${name}.json`, import.meta.url), "utf8"));
const fixture = await readFixture("jobBilling");
const job = await readFixture("projectCompletionDate");
const token = [
  "eyJhbGciOiJIUzI1NiJ9",
  Buffer.from(JSON.stringify({ id: fixture.auth.id, exp: 4102444800 })).toString("base64url"),
  "test-signature",
].join(".");
await runBrowserHarness(
  async (page, origin) => {
    page.setDefaultTimeout(15000);
    const errors = [],
      jobWrites = [],
      profileWrites = [];
    let claims = ["job", "absorb"],
      failProfiles = false,
      failJob = false;
    const client = structuredClone(fixture.client);
    let pendingAbsorb = null;
    let failAbsorb = false;
    let failAbsorbLoad = false;
    const absorbWrites = [];
    page.on("pageerror", (error) => errors.push(error.message));
    page.on("dialog", (dialog) => dialog.accept());
    await page.addInitScript(
      ({ token, record }) =>
        localStorage.setItem("pocketbase_auth", JSON.stringify({ token, record })),
      { token, record: fixture.auth },
    );
    await page.route("**/api/**", async (route) => {
      const request = route.request(),
        url = new URL(request.url()),
        path = url.pathname;
      const send = (body, status = 200) =>
        route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) });
      const list = (items) =>
        send({ page: 1, perPage: 500, totalItems: items.length, totalPages: 1, items });
      if (path.endsWith("/users/auth-refresh")) return send({ token, record: fixture.auth });
      if (path.includes("/user_claims_summary/")) return send({ id: fixture.auth.id, claims });
      if (path.includes("/user_po_approver_profile/"))
        return send({ id: fixture.auth.id, claims: [], divisions: [] });
      if (path === "/api/users/defaults" || path === "/api/nav/badges") return send({});
      if (path === "/api/clients") return send([client, fixture.otherClient]);
      if (path === `/api/clients/${client.id}`) return send(client);
      if (path.endsWith("/notes") || path === "/api/clients/busdev-leads") return send([]);
      if (path === "/api/jobs") return send([]);
      if (path === `/api/jobs/${job.id}` && request.method() === "PUT") {
        jobWrites.push(request.postDataJSON());
        return send({ message: "Keep the draft open for inspection." }, 400);
      }
      if (path === `/api/collections/jobs/records/${job.id}`)
        return failJob ? send({ message: "Access denied" }, 403) : send(job);
      if (path === "/api/collections/client_invoicing_information/records") {
        if (request.method() === "POST") {
          const payload = request.postDataJSON();
          const record = {
            ...fixture.profile,
            ...payload,
            id: profileWrites.length ? fixture.copiedProfile.id : fixture.createdProfile.id,
          };
          profileWrites.push(payload);
          client.invoicing_profiles.push(record);
          return send(record);
        }
        if (failProfiles) return send({ message: "Profile service failed" }, 500);
        return list(
          (url.searchParams.get("filter") || "").includes(fixture.otherClient.id)
            ? []
            : client.invoicing_profiles,
        );
      }
      if (path === `/api/client_contacts/${client.contacts[0].id}/absorb`) {
        if (failAbsorb) return send({ message: "Merge failed for this test." }, 400);
        absorbWrites.push(request.postDataJSON());
        pendingAbsorb = structuredClone(fixture.absorbAction);
        // Model the server removing the merged contact, then restoring it on undo.
        client.contacts = client.contacts.filter(
          (contact) => contact.id !== fixture.client.contacts[1].id,
        );
        return send({});
      }
      if (path === "/api/client_contacts/undo_absorb") {
        client.contacts.push(structuredClone(fixture.client.contacts[1]));
        pendingAbsorb = null;
        return send({});
      }
      if (
        path === `/api/collections/absorb_actions/records/${fixture.absorbAction.id}` &&
        request.method() === "DELETE"
      ) {
        pendingAbsorb = null;
        return route.fulfill({ status: 204 });
      }
      if (path.endsWith("/absorb_actions/records")) {
        if (pendingAbsorb && failAbsorbLoad)
          return send({ message: "Pending merge unavailable for this test." }, 500);
        return list(pendingAbsorb ? [pendingAbsorb] : []);
      }
      if (path.endsWith("/client_contacts/records")) return list(client.contacts);
      if (path.endsWith("/job_time_allocations/records")) return list(fixture.allocations);
      if (path.endsWith("/divisions/records")) return list(fixture.divisions);
      if (path.endsWith("/branches/records")) return list(fixture.branches);
      if (path.endsWith("/profiles/records")) return list(fixture.profiles);
      if (path.includes("/records/")) return send({});
      if (path.endsWith("/records")) return list([]);
      return send([]);
    });
    const edit = `${origin}/jobs/${job.id}/edit`;
    const description = () => page.getByLabel("Description", { exact: true });
    const invoice = () => page.locator('select[name="invoicing_information"]');
    const hours = () => page.getByRole("spinbutton", { name: "Hours for division 1" });
    const manage = () =>
      page.getByRole("button", { name: "Manage client contacts and invoicing", exact: true });
    const assertDraft = async (profileId) => {
      await invoice().waitFor();
      assert.equal(await description().inputValue(), "Unsaved job scope");
      assert.equal(await page.getByLabel("Project Value").inputValue(), "123456");
      assert.equal(await hours().inputValue(), "12");
      assert.equal(await invoice().inputValue(), profileId);
      assert.equal(
        await invoice().getAttribute("required"),
        null,
        "the form reports its own errors",
      );
      assert.equal(jobWrites.length, 0, "client setup must not save the job draft");
    };
    const selectedNotice = () =>
      page
        .getByRole("status")
        .filter({ hasText: "Invoicing profile selected. Save the job to keep it." });
    await page.goto(edit);
    await page.getByRole("button", { name: "Create one", exact: true }).waitFor();
    await page
      .getByText("Division hours can still be saved without an invoicing profile.", { exact: true })
      .waitFor();
    assert.equal(await invoice().count(), 0, "empty profile list has no selector");
    assert.equal(await manage().count(), 0, "empty profile list has one action");
    await description().fill("Unsaved job scope");
    await page.getByLabel("Project Value").fill("123456");
    await hours().fill("12.5");
    assert.equal(await hours().evaluate((input) => input.validity.stepMismatch), true);
    await hours().fill("12");
    assert.equal(await hours().getAttribute("step"), "1");
    await page.getByRole("button", { name: "Create one", exact: true }).click();
    await page.getByRole("heading", { name: "Add invoicing profile", exact: true }).waitFor();
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    await page.getByLabel("Profile name (optional)").fill(fixture.createdProfile.name);
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await assertDraft(fixture.createdProfile.id);
    await selectedNotice().waitFor();
    assert.equal(profileWrites[0].client, client.id);
    assert.equal(profileWrites[0].contact, client.contacts[0].id);
    await manage().click();
    await page.getByRole("region", { name: "Contacts", exact: true }).waitFor();
    await page.goBack();
    await assertDraft(fixture.createdProfile.id);
    await manage().click();
    await page
      .getByTitle(`Edit invoicing profile ${fixture.createdProfile.name}`, { exact: true })
      .click();
    await page.getByLabel("Profile name (optional)").fill(fixture.copiedProfile.name);
    await page.getByRole("button", { name: "Save as new profile", exact: true }).click();
    await assertDraft(fixture.copiedProfile.id);
    await manage().click();
    await page.getByTitle("Add invoicing profile", { exact: true }).click();
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    await page
      .getByRole("region", { name: "Existing profiles for this contact" })
      .getByRole("button", { name: "Use this profile", exact: true })
      .first()
      .click();
    await assertDraft(fixture.createdProfile.id);
    assert.equal(profileWrites.length, 2, "reuse must not create or update a profile");
    // A contact merge must keep the unsaved job and the pending merge controls accessible.
    const openMergeFromContact = async () => {
      await manage().click();
      await page.getByTitle("Edit Robin Recipient", { exact: true }).click();
      await page.getByText("More actions", { exact: true }).click();
      await page
        .getByRole("button", { name: "Merge contacts into this contact", exact: true })
        .click();
      await page.getByRole("heading", { name: "Target Record", exact: true }).waitFor();
      assert.match(new URL(page.url()).searchParams.get("return_to"), /workflow_return=/);
      await page.getByRole("link", { name: "Return to the job form", exact: true }).waitFor();
    };
    await openMergeFromContact();
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await page.getByRole("link", { name: "Return to the job form", exact: true }).click();
    await assertDraft(fixture.createdProfile.id);
    for (const resolution of ["Undo", "Commit"]) {
      await openMergeFromContact();
      await page
        .getByLabel("Select Record", { exact: true })
        .selectOption(fixture.client.contacts[1].id);
      if (resolution === "Undo") {
        failAbsorb = true;
        await page.getByRole("button", { name: "Absorb", exact: true }).click();
        await page.getByText("Merge failed for this test.", { exact: true }).waitFor();
        assert.equal(absorbWrites.length, 0);
        await page.getByRole("link", { name: "Return to the job form", exact: true }).waitFor();
        failAbsorb = false;
      }
      failAbsorbLoad = resolution === "Undo";
      await page.getByRole("button", { name: "Absorb", exact: true }).click();
      if (failAbsorbLoad) {
        await page.getByText("Pending merge unavailable for this test.", { exact: true }).waitFor();
        assert.equal(
          await page.getByRole("button", { name: "Absorb", exact: true }).count(),
          0,
          "a completed merge must not be submitted again when its review fails to load",
        );
        await page.getByRole("link", { name: "Return to the job form", exact: true }).waitFor();
        failAbsorbLoad = false;
        await page.reload();
      }
      await page
        .getByRole("heading", { name: "Pending Absorb — Client Contacts", exact: true })
        .waitFor();
      assert.equal(
        new URL(page.url()).pathname,
        `/clients/${client.id}/contacts/${client.contacts[0].id}/absorb`,
      );
      await page.getByRole("link", { name: "Return to the job form", exact: true }).waitFor();
      await page.getByRole("button", { name: resolution, exact: true }).click();
      await page.getByRole("button", { name: `Confirm ${resolution}`, exact: true }).click();
      await page.getByRole("region", { name: "Contacts", exact: true }).waitFor();
      await page.getByRole("link", { name: "Return to the job form", exact: true }).click();
      await assertDraft(fixture.createdProfile.id);
    }
    assert.deepEqual(absorbWrites, [
      { ids_to_absorb: [fixture.client.contacts[1].id] },
      { ids_to_absorb: [fixture.client.contacts[1].id] },
    ]);
    await page.getByLabel("Project Completion Date", { exact: false }).fill("2027-01-01");
    await page
      .locator('form[enctype="multipart/form-data"]')
      .evaluate((form) => form.requestSubmit());
    await page.getByText("Keep the draft open for inspection.", { exact: true }).waitFor();
    assert.equal(jobWrites[0].job.project_value, 123456, "existing values stay editable");
    assert.equal(jobWrites[0].job.client, client.id, "job payload retains the editable client");
    assert.equal(jobWrites[0].job.invoicing_information, fixture.createdProfile.id);
    assert.equal(jobWrites[0].allocations[0].hours, 12);
    const clientField = page
      .locator("label")
      .filter({ hasText: /^Client$/ })
      .locator("..");
    await clientField.getByRole("button", { name: "Clear", exact: true }).click();
    await page.getByRole("textbox", { name: "Client", exact: true }).fill(fixture.otherClient.name);
    await page.getByRole("button", { name: fixture.otherClient.name, exact: true }).click();
    await page.getByText("This client has no invoicing profiles.", { exact: false }).waitFor();
    assert.equal(await invoice().count(), 0, "switching clients must not retain a foreign profile");
    failProfiles = true;
    await page.goto(edit);
    await page
      .getByRole("alert")
      .filter({ hasText: "Could not load invoicing profiles" })
      .waitFor();
    failProfiles = false;
    await page.getByRole("button", { name: "Try again", exact: true }).click();
    await invoice().waitFor();
    // Profiles that read the same, created on the same day, are told apart by time.
    client.invoicing_profiles = ["10", "15"].map((hour) => ({
      ...fixture.profile,
      id: `sameday${hour}0000000`,
      created: `2026-10-01 ${hour}:00:00.000Z`,
    }));
    await page.goto(edit);
    await invoice().waitFor();
    const labels = await invoice().locator("option").allTextContents();
    assert.equal(new Set(labels).size, labels.length, "every profile option reads differently");
    claims = ["time"];
    client.invoicing_profiles = [];
    // An existing project without a profile. Active projects need a completion
    // date before any save, as on main.
    job.invoicing_information = "";
    job.project_completion_date = "2027-01-01";
    await page.goto(edit);
    await page
      .getByText("Ask a job maintainer to create an invoicing profile for this client.", {
        exact: false,
      })
      .waitFor();
    assert.equal(await page.getByRole("button", { name: "Create one", exact: true }).count(), 0);
    assert.equal(await manage().count(), 0);
    // Changing only division hours needs no profile and sends only the allocations.
    const writesBefore = jobWrites.length;
    await hours().fill("14");
    await page
      .locator('form[enctype="multipart/form-data"]')
      .evaluate((form) => form.requestSubmit());
    await page.getByText("Keep the draft open for inspection.", { exact: true }).waitFor();
    assert.equal(jobWrites.length, writesBefore + 1);
    assert.equal(jobWrites.at(-1).job, undefined, "an allocation-only save sends no job fields");
    assert.equal(jobWrites.at(-1).allocations[0].hours, 14);
    // Any other change still requires a profile.
    await description().fill("Changed scope");
    await page
      .locator('form[enctype="multipart/form-data"]')
      .evaluate((form) => form.requestSubmit());
    await page
      .getByText("This client needs an invoicing profile before project changes can be saved.", {
        exact: true,
      })
      .waitFor();
    assert.equal(jobWrites.length, writesBefore + 1);
    failJob = true;
    await page.goto(edit);
    await page.getByText("Could not load this job.", { exact: true }).waitFor();
    assert.equal(await page.getByRole("heading", { name: "Create Job", exact: true }).count(), 0);
    assert.deepEqual(errors, []);
  },
  { app: true },
);
console.log(
  "Job billing: create/copy/reuse return, draft and whole-hour allocations retained, permissions and load failures checked.",
);
