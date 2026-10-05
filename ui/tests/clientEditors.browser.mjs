import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";

await runBrowserHarness(
  async (page, origin) => {
    page.setDefaultTimeout(15000);
    const errors = [];
    const writes = [];
    let failSave = false;
    let jobCount = 4;
    let failUsageRead = false;
    let holdSave = false;
    let releaseSave;
    let signalSaveHeld = () => {};
    let failDelete = false;
    page.on("pageerror", (error) => errors.push(error.message));
    await page.route("**/api/**", async (route) => {
      const request = route.request();
      const path = new URL(request.url()).pathname;
      const collection = path.match(/\/collections\/(\w+)\/records/);
      if (request.method() === "DELETE") {
        writes.push({ method: "DELETE", path });
        return route.fulfill(
          failDelete
            ? { status: 400, json: { message: "Contact is now in use." } }
            : { status: 204 },
        );
      }
      if (request.method() === "POST" || request.method() === "PATCH") {
        const body = request.postDataJSON();
        writes.push({ method: request.method(), path, body });
        if (holdSave)
          await new Promise((resolve) => {
            releaseSave = resolve;
            signalSaveHeld();
          });
        return route.fulfill(
          failSave
            ? {
                status: 400,
                json: {
                  message: "Save failed. Try again.",
                  data: { email: { message: "Email was rejected." } },
                },
              }
            : {
                json: {
                  id:
                    request.method() === "PATCH"
                      ? path.split("/").at(-1)
                      : collection?.[1] === "client_contacts"
                        ? "contact-new"
                        : "profile-new",
                  ...body,
                },
              },
        );
      }
      if (path === "/api/clients/client1")
        return route.fulfill(
          failUsageRead
            ? { status: 500, json: { message: "Unable to check profile usage." } }
            : {
                json: {
                  id: "client1",
                  name: "Example Client",
                  contacts: [],
                  invoicing_profiles: [{ id: "profile1", job_count: jobCount }],
                },
              },
        );
      if (path === "/api/clients/devleads")
        return route.fulfill({ json: [{ id: "lead1", given_name: "Business", surname: "Lead" }] });
      return route.fulfill({
        json: {
          id: "client1",
          items: [{ id: "lead1", given_name: "Business", surname: "Lead" }],
          page: 1,
          totalPages: 1,
        },
      });
    });
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?contact&new`);
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    assert.equal(await page.getByText("This field is required.", { exact: true }).count(), 2);
    assert.equal(writes.length, 0);
    await page.getByLabel("Given name", { exact: true }).fill("New");
    await page.getByLabel("Surname", { exact: true }).fill("Contact");
    await page.getByLabel("Email (optional)", { exact: true }).fill("not-email");
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await page.getByText("Enter a valid email address.", { exact: true }).waitFor();
    await page.getByLabel("Email (optional)", { exact: true }).fill("new@example.com");
    page.once("dialog", (dialog) => dialog.dismiss());
    const navigationCancelled = await page.evaluate(() => {
      let cancelled = false;
      for (const guard of window.navigationGuards || [])
        guard({
          cancel: () => {
            cancelled = true;
          },
        });
      return cancelled;
    });
    assert.equal(navigationCancelled, true, "route changes protect dirty contact edits");
    failSave = true;
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await page.getByText("Email was rejected.", { exact: true }).waitFor();
    assert.equal(await page.getByLabel("Given name", { exact: true }).inputValue(), "New");
    page.once("dialog", (dialog) => dialog.dismiss());
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    assert.equal(
      await page.evaluate(() => window.cancelled),
      0,
      "cancel keeps dirty form when discard is declined",
    );
    failSave = false;
    holdSave = true;
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await page.waitForFunction(() => document.querySelector("button[type=submit]").disabled);
    assert.equal(
      await page.evaluate(() => {
        let cancelled = false;
        for (const guard of window.navigationGuards || [])
          guard({
            cancel: () => {
              cancelled = true;
            },
          });
        return cancelled;
      }),
      true,
      "navigation is blocked until an in-flight save completes",
    );
    holdSave = false;
    releaseSave();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.at(-1).body.client, "client1");
    assert.equal(writes.at(-1).body.email, "new@example.com");
    assert.equal("id" in writes.at(-1).body, false);

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?contact&used`);
    await page.getByText("More actions", { exact: true }).click();
    assert.equal(
      await page.getByRole("button", { name: "Delete contact", exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      await page
        .getByRole("button", { name: "Merge contacts into this contact", exact: true })
        .count(),
      1,
    );
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?contact`);
    await page.getByText("More actions", { exact: true }).click();
    page.once("dialog", (dialog) => dialog.accept());
    failDelete = true;
    await page.getByRole("button", { name: "Delete contact", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Contact is now in use." }).waitFor();
    failDelete = false;
    page.once("dialog", (dialog) => dialog.accept());
    await page.getByRole("button", { name: "Delete contact", exact: true }).click();
    await page.waitForFunction(() => window.deleted === 1);

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?edit`);
    await page.getByLabel("Profile name (optional)", { exact: true }).fill("Capital projects");
    await page
      .getByLabel("Invoice instructions (optional)", { exact: true })
      .fill("Include our PO number.");
    await page.getByRole("button", { name: "Edit contact", exact: true }).click();
    await page.getByLabel("Email (optional)", { exact: true }).fill("alex-updated@example.com");
    page.once("dialog", (dialog) => dialog.dismiss());
    assert.equal(
      await page.evaluate(() => {
        let cancelled = false;
        for (const guard of window.navigationGuards || [])
          guard({
            cancel: () => {
              cancelled = true;
            },
          });
        return cancelled;
      }),
      true,
      "nested contact edits and parent invoice draft share one navigation guard",
    );
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await page.getByRole("heading", { name: "Edit invoicing profile", exact: true }).waitFor();
    assert.equal(
      await page.getByLabel("Profile name (optional)", { exact: true }).inputValue(),
      "Capital projects",
    );
    assert.equal(
      await page.getByLabel("Invoice instructions (optional)", { exact: true }).inputValue(),
      "Include our PO number.",
    );
    assert.match(
      await page.getByLabel("Send invoices to", { exact: true }).textContent(),
      /alex-updated@example.com/,
    );
    await page.getByRole("button", { name: "Add contact", exact: true }).click();
    await page.getByLabel("Given name", { exact: true }).fill("Billing");
    await page.getByLabel("Surname", { exact: true }).fill("Person");
    await page.getByLabel("Email (optional)", { exact: true }).fill("billing@example.com");
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await page.getByRole("heading", { name: "Edit invoicing profile", exact: true }).waitFor();
    assert.equal(
      await page.getByLabel("Send invoices to", { exact: true }).inputValue(),
      "contact-new",
    );
    await page.getByText("Uses the client address.", { exact: true }).waitFor();
    assert.equal(
      await page.getByLabel("Profile name (optional)", { exact: true }).inputValue(),
      "Capital projects",
    );
    const choice = page.getByRole("dialog", { name: "Save invoicing profile", exact: true });
    const beforeCancel = writes.length;
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await choice.waitFor();
    assert.match(await choice.innerText(), /4 jobs/);
    await choice.getByRole("button", { name: "Update shared profile", exact: true }).hover();
    await choice
      .getByText("Apply these changes everywhere this profile is used.", { exact: true })
      .waitFor();
    await choice.getByRole("button", { name: "Save as new profile", exact: true }).hover();
    await choice
      .getByText(
        "Keep the original profile unchanged and create a new profile from these fields.",
        { exact: true },
      )
      .waitFor();
    await choice.getByRole("button", { name: "Cancel", exact: true }).focus();
    await choice
      .getByText("Return to editing without saving. Your changes stay in the form.", {
        exact: true,
      })
      .waitFor();
    await choice.getByRole("button", { name: "Cancel", exact: true }).click();
    await choice.waitFor({ state: "hidden" });
    assert.equal(writes.length, beforeCancel, "cancelling sends no update");
    assert.equal(
      await page.getByLabel("Profile name (optional)", { exact: true }).inputValue(),
      "Capital projects",
    );
    assert.equal(
      await page.getByLabel("Send invoices to", { exact: true }).inputValue(),
      "contact-new",
    );
    assert.equal(
      await page.getByLabel("Invoice instructions (optional)", { exact: true }).inputValue(),
      "Include our PO number.",
    );
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await choice.waitFor();
    await page.keyboard.press("Escape");
    await choice.waitFor({ state: "hidden" });
    assert.equal(writes.length, beforeCancel);
    failUsageRead = true;
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Unable to check profile usage." }).waitFor();
    assert.equal(writes.length, beforeCancel);
    failUsageRead = false;
    jobCount = 7;
    failSave = true;
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await choice.waitFor();
    assert.match(await choice.innerText(), /7 jobs\./);
    await choice.getByRole("button", { name: "Update shared profile", exact: true }).click();
    await choice.getByRole("alert").filter({ hasText: "Save failed. Try again." }).waitFor();
    failSave = false;
    await choice.getByRole("button", { name: "Update shared profile", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.at(-1).method, "PATCH");
    assert.deepEqual(writes.at(-1).body, {
      client: "client1",
      name: "Capital projects",
      contact: "contact-new",
      fax: "",
      invoicing_instructions: "Include our PO number.",
    });

    for (const count of [0, 1, 2]) {
      jobCount = count;
      await page.goto(`${origin}/tests/fixtures/clientEditors.html?edit`);
      await page
        .getByLabel("Profile name (optional)", { exact: true })
        .fill(`Profile for ${count} jobs`);
      await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
      if (count > 1) {
        await choice.waitFor();
        assert.match(await choice.innerText(), /2 jobs\./);
        const beforeCopy = writes.length;
        failSave = true;
        await choice.getByRole("button", { name: "Save as new profile", exact: true }).click();
        await choice.getByRole("alert").filter({ hasText: "Save failed. Try again." }).waitFor();
        failSave = false;
        holdSave = true;
        const heldRequest = new Promise((resolve) => {
          signalSaveHeld = resolve;
        });
        await choice.getByRole("button", { name: "Save as new profile", exact: true }).click();
        await page.waitForFunction(() =>
          [...document.querySelectorAll("dialog button")].every((button) => button.disabled),
        );
        await heldRequest;
        await page.keyboard.press("Escape");
        assert.equal(await choice.isVisible(), true, "cannot close while saving");
        holdSave = false;
        releaseSave();
        await page.waitForFunction(() => window.saved.length === 1);
        assert.ok(
          writes.slice(beforeCopy).every((write) => write.method === "POST"),
          "copy never updates original",
        );
        assert.equal(await page.evaluate(() => window.savedModes[0]), "copy");
      } else {
        await page.waitForFunction(() => window.saved.length === 1);
        assert.equal(await choice.isVisible(), false);
        assert.equal(writes.at(-1).method, "PATCH");
        // Copy remains available even for an unused or single-job profile.
        await page.goto(`${origin}/tests/fixtures/clientEditors.html?edit`);
        await page
          .getByLabel("Profile name (optional)", { exact: true })
          .fill("Separate arrangement");
        await page.getByRole("button", { name: "Save as new profile", exact: true }).click();
        await page.waitForFunction(() => window.saved.length === 1);
        assert.equal(writes.at(-1).method, "POST");
        assert.equal(writes.at(-1).body.name, "Separate arrangement");
      }
    }
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?edit`);
    const beforeUnchanged = writes.length;
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(
      writes.length,
      beforeUnchanged,
      "unchanged fields must not overwrite another user's changes",
    );
    assert.equal(await choice.isVisible(), false, "unchanged fields need no confirmation");
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?edit&fail-return`);
    await page.getByLabel("Profile name (optional)", { exact: true }).fill("Saved once");
    const beforeReturnFailure = writes.length;
    await page.getByRole("button", { name: "Save as new profile", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Retry the return" }).waitFor();
    assert.equal(writes.length, beforeReturnFailure + 1);
    assert.equal(writes.at(-1).method, "POST");
    assert.equal(
      await page.getByRole("button", { name: "Save invoicing profile", exact: true }).count(),
      0,
    );
    await page.getByRole("button", { name: "Return to your work", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.length, beforeReturnFailure + 1, "return retry must not write again");
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?many`);
    assert.equal(
      await page.getByRole("button", { name: "Save as new profile", exact: true }).count(),
      0,
    );
    await page.getByLabel("Send invoices to", { exact: true }).fill("person100");
    await page
      .getByRole("button", { name: "Contact 100 — person100@example.com", exact: true })
      .click();
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.at(-1).body.contact, "contact100");
    assert.equal(await choice.isVisible(), false);

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?profiles`);
    assert.equal(await page.getByRole("button", { name: "Add contact", exact: true }).count(), 1);
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    assert.equal(await page.getByRole("button", { name: "Edit contact", exact: true }).count(), 1);
    assert.equal(
      await page.getByRole("button", { name: "Save as new profile", exact: true }).count(),
      0,
    );
    const existingProfiles = page.getByRole("region", {
      name: "Existing profiles for this contact",
      exact: true,
    });
    await existingProfiles.getByText("Capital projects", { exact: true }).waitFor();
    await existingProfiles.getByText("Maintenance", { exact: true }).waitFor();
    assert.equal(await existingProfiles.getByText("Other recipient", { exact: true }).count(), 0);
    await existingProfiles
      .getByText("Include the capital project number.\nSend a PDF invoice.", { exact: true })
      .waitFor();
    assert.equal(await existingProfiles.locator("summary").count(), 0);
    assert.equal(
      await existingProfiles
        .getByRole("button", { name: "Expand invoice instructions", exact: true })
        .count(),
      0,
      "short instructions use the shared single-text display without a disclosure",
    );
    await page
      .getByLabel("Profile name (optional)", { exact: true })
      .fill("Another capital profile");
    page.once("dialog", (dialog) => dialog.dismiss());
    const beforeReuse = writes.length;
    await existingProfiles
      .getByRole("button", { name: "Use this profile", exact: true })
      .first()
      .click();
    assert.equal(
      await page.getByLabel("Profile name (optional)", { exact: true }).inputValue(),
      "Another capital profile",
    );
    assert.equal(await page.evaluate(() => window.saved.length), 0);
    page.once("dialog", (dialog) => dialog.accept());
    await existingProfiles
      .getByRole("button", { name: "Use this profile", exact: true })
      .first()
      .click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.length, beforeReuse, "reusing a profile does not write a duplicate");
    assert.equal(await page.evaluate(() => window.saved[0].id), "capital-profile");
    assert.equal(await page.evaluate(() => window.savedModes[0]), "reuse");
    await page
      .getByRole("status")
      .filter({ hasText: "Using the existing invoicing profile." })
      .waitFor();

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?profiles`);
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    await page
      .getByLabel("Invoice instructions (optional)", { exact: true })
      .fill("Use a different project code.");
    failSave = true;
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Save failed. Try again." }).waitFor();
    assert.equal(
      await page.getByLabel("Invoice instructions (optional)", { exact: true }).inputValue(),
      "Use a different project code.",
    );
    failSave = false;
    const beforeCreate = writes.length;
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.length, beforeCreate + 1);
    assert.equal(writes.at(-1).method, "POST");
    assert.equal(writes.at(-1).body.contact, "contact1");
    assert.equal(writes.at(-1).body.client, "client1");
    assert.equal(writes.at(-1).body.invoicing_instructions, "Use a different project code.");
    assert.equal(await page.evaluate(() => window.savedModes[0]), "create");
    assert.equal(await choice.isVisible(), false, "new profiles need no shared-edit confirmation");

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?profiles-many`);
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    const beforeSearch = writes.length;
    await existingProfiles
      .getByPlaceholder("Find an existing profile", { exact: true })
      .fill("Capital 99");
    await existingProfiles
      .getByPlaceholder("Find an existing profile", { exact: true })
      .press("Enter");
    await existingProfiles.getByText("Capital 99", { exact: true }).waitFor();
    assert.equal(writes.length, beforeSearch, "Enter in reuse search must not create a profile");
    await existingProfiles.getByRole("button", { name: "Use this profile", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(await page.evaluate(() => window.saved[0].id), "capital-99");

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?time&edit`);
    assert.equal(
      await page.getByLabel("Profile name (optional)", { exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      await page.getByRole("button", { name: "Save invoicing profile", exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      await page.getByRole("button", { name: "Save as new profile", exact: true }).count(),
      0,
    );
    assert.equal(await page.getByRole("button", { name: "Add contact", exact: true }).count(), 0);
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?none`);
    assert.equal(
      await page.getByRole("button", { name: "Save invoicing profile", exact: true }).isDisabled(),
      true,
    );

    await page.goto(`${origin}/tests/fixtures/clientEditors.html?contact&readonly`);
    assert.equal(await page.getByLabel("Given name", { exact: true }).isDisabled(), true);
    assert.equal(
      await page.getByRole("button", { name: "Save contact", exact: true }).isDisabled(),
      true,
    );
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?client`);
    await page.getByRole("heading", { name: "Edit client information", exact: true }).waitFor();
    assert.equal(
      await page.getByRole("button", { name: /contact/i }).count(),
      0,
      "client editor cannot duplicate contact management",
    );
    await page.getByLabel("Name", { exact: true }).fill("");
    await page.getByRole("button", { name: "Save client", exact: true }).click();
    await page.getByText("This field is required.", { exact: true }).waitFor();
    await page.getByLabel("Name", { exact: true }).fill("Updated client");
    await page.getByLabel("Business development lead", { exact: true }).selectOption("lead1");
    failSave = true;
    await page.getByRole("button", { name: "Save client", exact: true }).click();
    await page.getByRole("alert").filter({ hasText: "Save failed. Try again." }).waitFor();
    assert.equal(await page.getByLabel("Name", { exact: true }).inputValue(), "Updated client");
    failSave = false;
    await page.getByRole("button", { name: "Save client", exact: true }).click();
    await page.waitForFunction(() => window.saved.length === 1);
    assert.equal(writes.at(-1).body.name, "Updated client");
    assert.equal(writes.at(-1).body.business_development_lead, "lead1");
    assert.equal("id" in writes.at(-1).body, false);
    assert.equal("client_contacts" in writes.at(-1).body, false);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto(`${origin}/tests/fixtures/clientEditors.html?edit`);
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      true,
    );
    await page.getByLabel("Profile name (optional)", { exact: true }).fill("Mobile profile");
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await choice.waitFor();
    const bounds = await choice.boundingBox();
    assert.ok(bounds.x >= 0 && bounds.x + bounds.width <= 390);
    await choice.getByRole("button", { name: "Save as new profile", exact: true }).focus();
    await page.screenshot({ path: "/tmp/client-invoicing-copy-mobile.png", fullPage: true });
    await choice.getByRole("button", { name: "Cancel", exact: true }).click();
    assert.deepEqual(errors, []);
  },
  {
    services: {
      "$app/navigation":
        "export const goto = (url) => { location.href = url; }; export const beforeNavigate = (guard) => { (window.navigationGuards ||= []).push(guard); }; export const invalidateAll = async () => {}; export const replaceState = (url, state) => history.replaceState(state, '', url);",
    },
  },
);
console.log("Client editor browser checks passed.");
