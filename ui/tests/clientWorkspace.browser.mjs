import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { runBrowserHarness } from "./browserHarness.mjs";
const fixture = JSON.parse(
  await readFile(new URL("./fixtures/clientWorkspace.json", import.meta.url), "utf8"),
);
const authRecord = {
  id: "workspaceuser",
  collectionName: "users",
  collectionId: "_pb_users_auth_",
  email: "workspace@example.test",
  verified: true,
};
const token = [
  "eyJhbGciOiJIUzI1NiJ9",
  Buffer.from(JSON.stringify({ id: authRecord.id, exp: 4102444800 })).toString("base64url"),
  "test-signature",
].join(".");
await runBrowserHarness(
  async (page, origin) => {
    page.setDefaultTimeout(15000);
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    let claims = ["job"];
    let notes = structuredClone(fixture.notes);
    let failNotesRefresh = false;
    const client = structuredClone(fixture.client);
    const writes = [];
    const profileWrites = [];
    await page.addInitScript(
      ({ token, record }) =>
        localStorage.setItem("pocketbase_auth", JSON.stringify({ token, record })),
      { token, record: authRecord },
    );
    await page.route("**/api/**", async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      const path = url.pathname;
      const send = (body, status = 200) =>
        route.fulfill({
          status,
          contentType: "application/json",
          headers: { "access-control-allow-origin": "*" },
          body: JSON.stringify(body),
        });
      const list = (items) => {
        const number = Number(url.searchParams.get("page") || 1);
        const perPage = Number(url.searchParams.get("perPage") || 500);
        return send({
          page: number,
          perPage,
          totalItems: items.length,
          totalPages: Math.max(1, Math.ceil(items.length / perPage)),
          items: items.slice((number - 1) * perPage, number * perPage),
        });
      };
      if (request.method() === "OPTIONS")
        return route.fulfill({
          status: 204,
          headers: {
            "access-control-allow-origin": "*",
            "access-control-allow-methods": "GET,POST,PATCH,DELETE",
            "access-control-allow-headers": "*",
          },
        });
      if (
        path === "/api/collections/client_invoicing_information/records" &&
        request.method() === "POST"
      ) {
        const payload = request.postDataJSON();
        const record = {
          id:
            payload.name === fixture.createdProfile.name
              ? fixture.createdProfile.id
              : "profile-copy",
          ...payload,
          created: fixture.createdProfile.created,
          creator: fixture.createdProfile.creator,
          creator_name: fixture.createdProfile.creator_name,
          job_count: 0,
        };
        profileWrites.push({ method: "POST", record, payload });
        client.invoicing_profiles.push(record);
        return send(record);
      }
      if (path.includes("/client_invoicing_information/records/") && request.method() === "PATCH") {
        profileWrites.push({ method: "PATCH", record: request.postDataJSON() });
        // Keep the mock server state current for the options refresh on form return.
        const record = client.invoicing_profiles.find((row) => row.id === path.split("/").at(-1));
        Object.assign(record, request.postDataJSON());
        return send(record);
      }
      if (path.endsWith("/users/auth-refresh")) return send({ token, record: authRecord });
      if (path.includes("/user_claims_summary/")) return send({ id: authRecord.id, claims });
      if (path.includes("/user_po_approver_profile/"))
        return send({ id: authRecord.id, claims: [], divisions: [] });
      if (path === "/api/users/defaults" || path === "/api/nav/badges") return send({});
      if (path === `/api/clients/${client.id}`) return send(client);
      if (path === "/api/clients/missing") return send({ message: "Client not found" }, 404);
      if (path === `/api/clients/${client.id}/notes`)
        return failNotesRefresh ? send({ message: "Notes refresh failed" }, 500) : send(notes);
      if (path === "/api/collections/client_notes/records" && request.method() === "POST") {
        assert.equal(request.postDataJSON().note, fixture.createdNote.note);
        notes = [fixture.createdNote, ...notes];
        return send(fixture.createdNote);
      }
      if (path === "/api/clients") return send([client]);
      if (path === "/api/clients/busdev-leads")
        return send([{ id: "lead1", given_name: "Casey", surname: "Manager" }]);
      if (path === "/api/collections/client_contacts/records" && request.method() === "POST") {
        // Model the new contact in the mock server so the return route refreshes its choices.
        const payload = request.postDataJSON();
        const record = { ...fixture.createdContact, ...payload };
        writes.push(payload);
        client.contacts.push(record);
        return send(record);
      }
      if (path.includes("/client_contacts/records/") && request.method() === "PATCH") {
        // Mutate this browser fixture only to model a successful server save and reload.
        const record = client.contacts.find((row) => row.id === path.split("/").at(-1));
        const update = request.postDataJSON();
        writes.push(update);
        Object.assign(record, update);
        return send(record);
      }
      if (path === "/api/collections/jobs/records") {
        const filter = url.searchParams.get("filter") || "";
        return list(
          filter.includes(" || ")
            ? [...fixture.jobs, ...fixture.owner]
            : filter.includes("job_owner=")
              ? fixture.owner
              : filter.includes("number ~")
                ? fixture.proposals
                : fixture.jobs,
        );
      }
      if (path.includes("/records/")) return send({});
      if (path.endsWith("/records")) return list([]);
      if (path === "/api/realtime") return send({});
      return send([]);
    });
    const root = `${origin}/clients/${client.id}/details`;
    const contacts = () => page.getByRole("region", { name: "Contacts", exact: true });
    const invoicing = () => page.getByRole("region", { name: "Invoicing profiles", exact: true });
    await page.goto(root);
    await contacts().getByRole("searchbox").waitFor();
    assert.match(await invoicing().innerText(), /Created Oct 1, 2026 by Alex Adams/);
    // Without recorded creation details, a profile shows only its usage.
    assert.match(await invoicing().innerText(), /^1 job$/m);
    for (const name of [
      "Short instruction example",
      "Long instruction example",
      "Multiline instruction example",
    ]) {
      const profile = client.invoicing_profiles.find((profile) => profile.name === name);
      await invoicing().getByRole("searchbox").fill(name);
      assert.equal(await invoicing().locator(".headline_wrapper").count(), 1);
      const text = invoicing().getByText(profile.invoicing_instructions, { exact: true });
      assert.equal(
        await invoicing().locator("details").count(),
        0,
        "instructions do not use a disclosure triangle",
      );
      if (name === "Short instruction example") {
        assert.equal(await text.count(), 1, "short instructions are rendered once");
        assert.equal(await text.isVisible(), true);
        assert.equal(
          await invoicing()
            .getByRole("button", { name: "Expand invoice instructions", exact: true })
            .count(),
          0,
        );
      } else {
        assert.equal(await text.count(), 0, "collapsed instructions contain only a preview");
        const expand = invoicing().getByRole("button", {
          name: "Expand invoice instructions",
          exact: true,
        });
        assert.equal(await expand.innerText(), "…");
        await expand.click();
        assert.equal(await text.count(), 1, "expanded instructions are rendered once");
        assert.equal(await text.isVisible(), true);
        const collapse = invoicing().getByRole("button", {
          name: "Collapse invoice instructions",
          exact: true,
        });
        if (name === "Long instruction example") {
          await page.setViewportSize({ width: 390, height: 844 });
          await collapse.scrollIntoViewIfNeeded();
          assert.equal(
            await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
            true,
          );
          await page.screenshot({
            path: "/tmp/invoicing-instructions-expanded.png",
            fullPage: false,
          });
          await page.setViewportSize({ width: 1000, height: 900 });
        }
        assert.equal(await collapse.getAttribute("aria-expanded"), "true");
        assert.equal(
          await collapse.evaluate((button) => button === document.activeElement),
          true,
          "expanding keeps keyboard focus on the toggle",
        );
        await page.keyboard.press("Enter");
        assert.equal(await text.count(), 0);
        assert.equal(await expand.getAttribute("aria-expanded"), "false");
        await page.keyboard.press("Enter");
        assert.equal(await text.isVisible(), true);
        await collapse.click();
      }
    }
    await invoicing().getByRole("searchbox").fill("");
    const profileHelp = invoicing().getByRole("button", {
      name: "About invoicing profiles",
      exact: true,
    });
    const helpPanel = page.getByRole("dialog", { name: "About invoicing profiles", exact: true });
    assert.equal(await helpPanel.isVisible(), false);
    await profileHelp.click();
    await helpPanel.waitFor();
    assert.match(await helpPanel.innerText(), /required when creating or editing a project/);
    assert.match(await helpPanel.innerText(), /parent project/);
    assert.match(await helpPanel.innerText(), /optional for proposals/);
    assert.match(await helpPanel.innerText(), /Changes apply to all jobs/);
    await helpPanel.getByRole("button", { name: "Close explanation" }).click();
    assert.equal(await helpPanel.isVisible(), false);
    await profileHelp.focus();
    await page.keyboard.press("Enter");
    await helpPanel.waitFor();
    await page.keyboard.press("Escape");
    await helpPanel.waitFor({ state: "hidden" });
    await page.screenshot({ path: "/tmp/client-workspace-production-desktop.png", fullPage: true });
    assert.equal(await contacts().locator(".headline_wrapper").count(), 20);
    assert.equal(await invoicing().locator(".headline_wrapper").count(), 20);
    assert.equal(
      await contacts().getByTitle("Merge contacts into Contact001 Sample", { exact: true }).count(),
      0,
      "job permission alone cannot merge contacts",
    );
    await page.setViewportSize({ width: 390, height: 844 });
    await profileHelp.click();
    await helpPanel.waitFor();
    const helpBounds = await helpPanel.boundingBox();
    assert.ok(
      helpBounds.x >= 0 && helpBounds.x + helpBounds.width <= 390,
      "profile help fits a narrow screen",
    );
    await helpPanel.getByRole("button", { name: "Close explanation" }).click();
    await page.screenshot({ path: "/tmp/client-workspace-production-mobile.png", fullPage: true });
    assert.equal(
      await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth),
      true,
      "workspace fits a narrow screen",
    );
    await page.setViewportSize({ width: 1300, height: 900 });

    assert.match(
      await page.getByRole("region", { name: "Client information" }).innerText(),
      /4,321\.50/,
    );
    await contacts().getByRole("button", { name: "Next →" }).click();
    assert.match(await contacts().innerText(), /Contact021/);
    assert.match(await invoicing().innerText(), /Setup001/);
    await contacts().getByRole("searchbox").fill("contact100@");
    assert.equal(await contacts().locator(".headline_wrapper").count(), 1);
    assert.match(await contacts().innerText(), /Contact100/);
    assert.equal(await contacts().getByRole("button", { name: "Next →" }).count(), 0);
    await invoicing().getByRole("searchbox").fill("reference 099");
    assert.match(await invoicing().innerText(), /Setup099/);
    await contacts().getByRole("searchbox").fill("");
    await contacts().getByRole("combobox").selectOption("10");
    await contacts().getByRole("button", { name: "Next →" }).click();
    await contacts().getByRole("link", { name: "Contact011 Sample", exact: true }).click();
    await page.getByRole("heading", { name: "Edit contact", exact: true }).waitFor();
    assert.match(new URL(page.url()).searchParams.get("return_to"), /contacts_page=2/);
    await page.getByLabel("Email (optional)", { exact: true }).fill("changed@example.test");
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await contacts().getByRole("searchbox").waitFor();
    assert.match(await contacts().innerText(), /Page 2 \/ 10/);
    assert.match(await contacts().innerText(), /changed@example.test/);
    assert.equal(await invoicing().getByRole("searchbox").inputValue(), "reference 099");
    assert.equal(await contacts().getByRole("status").innerText(), "Contact saved.");
    assert.equal(await contacts().getByRole("status").getByRole("link").count(), 0);
    assert.match(await contacts().locator('[data-highlighted="true"]').innerText(), /Contact011/);
    assert.equal(new URL(page.url()).searchParams.has("saved_contact"), false);
    assert.equal(writes.length, 1);
    await page.reload();
    await contacts().getByRole("searchbox").waitFor();
    assert.equal(await page.getByRole("status").count(), 0);
    assert.equal(await page.locator('[data-highlighted="true"]').count(), 0);
    assert.match(await contacts().innerText(), /Page 2 \/ 10/);
    assert.equal(await invoicing().getByRole("searchbox").inputValue(), "reference 099");
    await contacts().getByRole("link", { name: "Contact011 Sample", exact: true }).click();
    await page.getByRole("heading", { name: "Edit contact", exact: true }).waitFor();
    await page.goBack();
    await contacts().getByRole("searchbox").waitFor();
    assert.match(await contacts().innerText(), /Page 2 \/ 10/);
    await contacts().getByRole("searchbox").fill("Contact011");
    await contacts().getByRole("link", { name: "Contact011 Sample", exact: true }).click();
    await page.getByRole("heading", { name: "Edit contact", exact: true }).waitFor();
    await page.getByLabel("Given name", { exact: true }).fill("Renamed");
    await page.getByRole("button", { name: "Save contact", exact: true }).click();
    await contacts().getByRole("searchbox").waitFor();
    assert.equal(await contacts().locator(".headline_wrapper").count(), 0);
    assert.match(await contacts().getByRole("status").innerText(), /Contact saved/);
    assert.equal(await contacts().getByRole("status").getByRole("link").count(), 1);
    await contacts().getByRole("link", { name: "Show contact", exact: true }).click();
    await contacts().getByRole("link", { name: "Renamed Sample", exact: true }).waitFor();
    assert.equal(await contacts().getByRole("searchbox").inputValue(), "");
    assert.match(await contacts().innerText(), /Page 2 \/ 10/);
    assert.equal(await invoicing().getByRole("searchbox").inputValue(), "reference 099");
    assert.equal(await contacts().getByRole("status").getByRole("link").count(), 0);
    assert.match(await contacts().locator('[data-highlighted="true"]').innerText(), /Renamed/);

    // A return marker models a successful save without changing the stored fixtures.
    // Reveal must find the saved ID and preserve unrelated list state.
    for (const filter of [
      "invoicing_q=reference&invoicing_page=1",
      "invoicing_q=reference+001&invoicing_page=1",
      "recipient=contact00000001&invoicing_q=reference&invoicing_page=1",
    ]) {
      await page.goto(
        `${origin}/clients/${client.id}/details?contacts_q=Contact020&contacts_per_page=10&invoicing_per_page=20&${filter}&saved_invoicing=invoice00000099`,
      );
      const status = invoicing().getByRole("status");
      await status.waitFor();
      assert.match(await status.innerText(), /Profile saved/);
      assert.equal(await status.getByRole("link").count(), 1);
      assert.equal(await invoicing().locator('[data-highlighted="true"]').count(), 0);
      assert.equal(new URL(page.url()).searchParams.has("saved_invoicing"), false);
      if (filter.startsWith("recipient=")) {
        await status.getByRole("link", { name: "Show profile", exact: true }).focus();
        // The link must remain available beyond the normal expiry while focused.
        await page.waitForTimeout(6200);
        assert.equal(await status.isVisible(), true);
      }
      await status.getByRole("link", { name: "Show profile", exact: true }).click();
      await invoicing().getByRole("link", { name: "Setup099", exact: true }).waitFor();
      assert.match(await invoicing().locator('[data-highlighted="true"]').innerText(), /Setup099/);
      assert.equal(await status.getByRole("link").count(), 0);
      assert.equal(await contacts().getByRole("searchbox").inputValue(), "Contact020");
      // The filtered contact list is short enough to hide its controls; its page size is kept.
      assert.equal(new URL(page.url()).searchParams.get("contacts_per_page"), "10");
      assert.equal(new URL(page.url()).searchParams.get("invoicing_page"), "5");
      assert.equal(new URL(page.url()).searchParams.has("recipient"), false);
      assert.equal(
        await invoicing().getByRole("searchbox").inputValue(),
        filter.includes("reference+001") ? "" : "reference",
      );
    }
    // A visible profile needs no link; both the message and row highlight expire.
    await page.goto(
      `${origin}/clients/${client.id}/details?invoicing_q=reference+099&saved_invoicing=invoice00000099`,
    );
    await invoicing().getByRole("status").waitFor();
    assert.equal(await invoicing().getByRole("status").innerText(), "Profile saved.");
    assert.equal(await invoicing().getByRole("status").getByRole("link").count(), 0);
    assert.equal(await invoicing().locator('[data-highlighted="true"]').count(), 1);
    await page.mouse.move(0, 0);
    await invoicing().getByRole("status").waitFor({ state: "hidden", timeout: 10000 });
    assert.equal(await invoicing().locator('[data-highlighted="true"]').count(), 0);

    await page
      .locator(`a[href^="/clients/${client.id}/jobs"]`)
      .filter({ hasText: /^Jobs$/ })
      .click();
    await page.getByRole("link", { name: "Projects (25)", exact: true }).waitFor();
    await page.getByRole("link", { name: "Proposals (25)", exact: true }).click();
    await page
      .getByRole("region", { name: "Jobs", exact: true })
      .getByText("Proposal 1", { exact: true })
      .waitFor();
    assert.match(
      await page.getByRole("region", { name: "Jobs", exact: true }).innerText(),
      /Proposal 1/,
    );
    assert.equal(new URL(page.url()).searchParams.get("view"), "proposals");
    await page.getByRole("link", { name: "Jobs as owner (25)", exact: true }).click();
    await page
      .getByRole("region", { name: "Jobs", exact: true })
      .getByText("Owner job 1", { exact: true })
      .waitFor();
    assert.match(
      await page.getByRole("region", { name: "Jobs", exact: true }).innerText(),
      /Owner job 1/,
    );
    await page.getByRole("link", { name: "Notes", exact: true }).click();
    await page.waitForURL((url) => url.pathname === `/clients/${client.id}/notes`);
    assert.equal(new URL(page.url()).pathname, `/clients/${client.id}/notes`);
    await page.reload();
    const notesRegion = page.getByRole("region", { name: "Notes", exact: true });
    await notesRegion.getByRole("searchbox", { name: "Filter notes" }).waitFor();
    assert.equal(await notesRegion.locator("summary").count(), 0);
    assert.equal(await notesRegion.locator(".headline_wrapper").count(), 3);
    const rowColours = await notesRegion
      .locator("li.contents")
      .evaluateAll((rows) => rows.map((row) => getComputedStyle(row).backgroundColor));
    assert.notEqual(rowColours[0], rowColours[1], "note rows alternate colours");
    assert.equal(rowColours[0], rowColours[2]);
    assert.equal(
      await notesRegion.getByRole("link", { name: "26-001", exact: true }).getAttribute("href"),
      "/jobs/project1/details",
    );
    for (const [term, expected] of [
      ["INSPECTION", "Confirm the inspection"],
      ["Brown", "maintenance report"],
      ["reception@", "Reception closes"],
      ["26-001", "Confirm the inspection"],
      ["pumping", "Confirm the inspection"],
    ]) {
      await notesRegion.getByRole("searchbox").fill(term);
      assert.equal(await notesRegion.locator(".headline_wrapper").count(), 1);
      assert.match(await notesRegion.innerText(), new RegExp(expected));
    }
    await notesRegion.getByRole("searchbox").fill("no matching note");
    await notesRegion
      .getByRole("status")
      .filter({ hasText: "No notes match this filter." })
      .waitFor();
    await notesRegion.getByTitle("Add note", { exact: true }).click();
    await notesRegion.getByRole("button", { name: "Add Note", exact: true }).click();
    await notesRegion.getByText("Note is required", { exact: true }).waitFor();
    assert.equal(await notesRegion.getByRole("searchbox").inputValue(), "no matching note");
    await notesRegion.locator("textarea").fill(fixture.createdNote.note);
    await notesRegion.getByLabel("Not tied to a specific job").check();
    await notesRegion.getByRole("button", { name: "Add Note", exact: true }).click();
    await notesRegion.getByText(fixture.createdNote.note, { exact: true }).waitFor();
    assert.equal(await notesRegion.getByRole("searchbox").inputValue(), "");
    assert.equal(await notesRegion.locator("textarea").count(), 0);
    assert.equal(await notesRegion.locator(".headline_wrapper").count(), 4);
    failNotesRefresh = true;
    await notesRegion.getByTitle("Refresh notes", { exact: true }).click();
    await page.getByText(/error refreshing notes:/).waitFor();
    assert.equal(
      await notesRegion.locator(".headline_wrapper").count(),
      4,
      "failed refresh retains notes",
    );
    failNotesRefresh = false;
    notes = [];
    await notesRegion.getByTitle("Refresh notes", { exact: true }).click();
    await notesRegion.getByText("No notes yet.", { exact: true }).waitFor();
    notes = structuredClone(fixture.notes);
    await notesRegion.getByTitle("Refresh notes", { exact: true }).click();
    await notesRegion.getByText("Send the maintenance report.", { exact: true }).waitFor();
    await page.screenshot({ path: "/tmp/client-notes-list.png", fullPage: true });
    for (const [view, expected] of [
      ["projects", "Project 1"],
      ["proposals", "Proposal 1"],
      ["owner", "Owner job 1"],
    ]) {
      await page.goto(`${origin}/clients/${client.id}/jobs?view=${view}`);
      await page.getByRole("region", { name: "Jobs", exact: true }).waitFor();
      assert.match(
        await page.getByRole("region", { name: "Jobs", exact: true }).innerText(),
        new RegExp(expected),
      );
      const jobList = page.getByRole("region", { name: "Jobs", exact: true });
      assert.equal(await jobList.locator(".headline_wrapper").count(), 20);
      await jobList.getByRole("link", { name: "Next →", exact: true }).click();
      await jobList
        .getByRole("navigation", { name: "Job pages" })
        .getByText("Page 2 / 2", { exact: true })
        .waitFor();
      assert.equal(await jobList.locator(".headline_wrapper").count(), 5);
      await jobList.getByRole("link", { name: "← Prev", exact: true }).click();
      await jobList
        .getByRole("navigation", { name: "Job pages" })
        .getByText("Page 1 / 2", { exact: true })
        .waitFor();
      const pageKey = { projects: "projectsPage", proposals: "proposalsPage", owner: "ownerPage" }[
        view
      ];
      await page.goto(`${origin}/clients/${client.id}/jobs?view=${view}&${pageKey}=999`);
      await jobList
        .getByRole("navigation", { name: "Job pages" })
        .getByText("Page 2 / 2", { exact: true })
        .waitFor();
      assert.equal(
        await jobList.locator(".headline_wrapper").count(),
        5,
        "stale pages clamp to the last page",
      );
    }
    // Each tab is a direct route, including reload and browser history.
    await page.goto(`${origin}/clients/${client.id}/jobs?view=proposals`);
    await page
      .getByRole("region", { name: "Jobs", exact: true })
      .getByText("Proposal 1", { exact: true })
      .waitFor();
    await page.reload();
    await page
      .getByRole("region", { name: "Jobs", exact: true })
      .getByText("Proposal 1", { exact: true })
      .waitFor();
    await page.getByRole("link", { name: "Notes", exact: true }).click();
    await page.getByRole("searchbox", { name: "Filter notes" }).waitFor();
    await page.goBack();
    await page
      .getByRole("region", { name: "Jobs", exact: true })
      .getByText("Proposal 1", { exact: true })
      .waitFor();
    assert.equal(new URL(page.url()).pathname, `/clients/${client.id}/jobs`);
    await page.goForward();
    await page.getByRole("searchbox", { name: "Filter notes" }).waitFor();
    assert.equal(new URL(page.url()).pathname, `/clients/${client.id}/notes`);
    claims = ["absorb"];
    await page.goto(root);
    await contacts().getByRole("searchbox").waitFor();
    assert.equal(
      await contacts().getByTitle("Merge contacts into Contact001 Sample", { exact: true }).count(),
      1,
    );
    assert.equal(await contacts().locator('a[href*="/edit"]').count(), 0);
    assert.equal(await page.getByTitle("Add contact", { exact: true }).count(), 0);
    // Time-only users can inspect the workspace but cannot change shared records.
    claims = ["time"];
    await page.goto(root);
    await contacts().getByRole("searchbox").waitFor();
    assert.equal(await contacts().locator('a[href*="/edit"]').count(), 0);
    assert.equal(await invoicing().locator('a[href*="/edit"]').count(), 0);
    assert.equal(await page.getByTitle("Add contact", { exact: true }).count(), 0);
    assert.equal(await invoicing().getByTitle("Add invoicing profile", { exact: true }).count(), 0);
    for (const path of [
      "invoicing/add",
      `invoicing/${client.invoicing_profiles[0].id}/edit`,
      "contacts/add",
    ]) {
      await page.goto(`${origin}/clients/${client.id}/${path}`);
      await page.getByText(/The job permission is required/).waitFor();
    }
    // Change only mock response collections to cover clients at each setup stage.
    const existingContacts = client.contacts;
    client.invoicing_profiles = [];
    await page.goto(root);
    await invoicing()
      .getByText("This client has no invoicing profiles. Ask a client maintainer to create one.", {
        exact: true,
      })
      .waitFor();
    client.contacts = [];
    await page.goto(root);
    await invoicing()
      .getByText("This client has no invoicing profiles. Ask a client maintainer to create one.", {
        exact: true,
      })
      .waitFor();
    claims = ["job"];
    await page.goto(root);
    await invoicing()
      .getByText(
        "This client has no invoicing profiles. Add a client contact, then create a profile.",
        { exact: true },
      )
      .waitFor();
    assert.equal(await contacts().getByTitle("Add contact", { exact: true }).count(), 1);
    assert.equal(await invoicing().getByTitle("Add invoicing profile", { exact: true }).count(), 0);
    client.contacts = existingContacts;
    await page.goto(`${root}?project_contact=${client.contacts[1].id}`);
    await invoicing()
      .getByText(
        "This client has no invoicing profiles. Create one using an existing client contact.",
        { exact: true },
      )
      .waitFor();
    await invoicing().getByTitle("Add invoicing profile", { exact: true }).click();
    await page.getByRole("heading", { name: "Add invoicing profile", exact: true }).waitFor();
    await page.getByRole("button", { name: "Cancel", exact: true }).click();
    await invoicing().getByTitle("Add invoicing profile", { exact: true }).waitFor();
    assert.equal(profileWrites.length, 0, "cancelling creation must not write");
    await invoicing().getByTitle("Add invoicing profile", { exact: true }).click();
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    await page
      .getByLabel("Profile name (optional)", { exact: true })
      .fill(fixture.createdProfile.name);
    await page
      .getByLabel("Invoice instructions (optional)", { exact: true })
      .fill(fixture.createdProfile.invoicing_instructions);
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await invoicing()
      .getByRole("link", { name: fixture.createdProfile.name, exact: true })
      .waitFor();
    assert.equal(profileWrites.length, 1);
    assert.equal(
      profileWrites[0].payload.creator,
      undefined,
      "server supplies creator attribution",
    );
    await invoicing().getByText("by Workspace User", { exact: false }).waitFor();
    assert.match(await invoicing().innerText(), /Oct 2, 2026/);
    // Reuse from the workspace does not write or report a false save.
    await invoicing().getByTitle("Add invoicing profile", { exact: true }).click();
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    await page.getByRole("button", { name: "Use this profile", exact: true }).click();
    await invoicing()
      .getByRole("status")
      .getByText("Profile highlighted.", { exact: true })
      .waitFor();
    assert.equal(profileWrites.length, 1);
    assert.equal(new URL(page.url()).searchParams.has("show_invoicing"), false);
    assert.equal(await invoicing().getByText("Profile saved.", { exact: true }).count(), 0);
    claims = [];
    await page.goto(root);
    await invoicing().getByText("by Workspace User", { exact: false }).waitFor();
    assert.equal(await invoicing().getByTitle("Add invoicing profile", { exact: true }).count(), 0);
    assert.equal(await invoicing().locator('a[href*="/edit"]').count(), 0);

    claims = ["job"];
    await page.goto(`${origin}/clients/${client.id}/contacts/foreigncontact/edit`);
    await page.getByText("Contact not found for this client.", { exact: true }).waitFor();
    await page.goto(`${origin}/clients/missing/details`);
    await page.getByText("Client not found.", { exact: true }).waitFor();
    assert.equal(await page.getByRole("button", { name: "Save client", exact: true }).count(), 0);
    for (const path of ["edit", "billing"]) {
      const response = await page.goto(`${origin}/clients/${client.id}/${path}`);
      assert.equal(response.status(), 404, `${path} must be removed, not redirected`);
      assert.equal(new URL(page.url()).pathname, `/clients/${client.id}/${path}`);
    }
    assert.deepEqual(errors, []);
    console.log(
      "Client workspace real-route navigation, large lists, edit return, permissions and missing records passed",
    );
  },
  { app: true },
);
