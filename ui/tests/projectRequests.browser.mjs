import assert from "node:assert/strict";
import { runBrowserHarness } from "./browserHarness.mjs";
await runBrowserHarness(
  async (page, origin) => {
    const saves = [];
    const valueActions = [];
    const profileSaves = [];
    const valueSaves = [];
    let emptyValueProfiles = false;
    let attachmentItems = [
      {
        id: "initial",
        job: "job1",
        status: "Approved",
        project_value: 120000,
        previous_project_value: 0,
        approval_max_value: 200000,
        approval_sequence: 1,
        initial_approval: true,
        approver: "reviewer",
      },
      {
        id: "pending",
        job: "job1",
        status: "Pending",
        project_value: 150000,
        assigned_approver: "reviewer",
        version: 4,
        document_hash: "hash123",
      },
    ];
    const errors = [];
    page.on("pageerror", (error) => errors.push(error.message));
    await page.addInitScript(() => {
      window.print = () => {
        window.printCalled = true;
      };
    });
    await page.route("**/api/**", (route) => {
      const path = new URL(route.request().url()).pathname;
      const legacyJob = {
        id: "job1",
        updated: "2026-09-30 12:00:00.000Z",
        client: "client1",
        number: "24-001",
        description: "Legacy project",
        contact: "contact1",
        invoicing_information: "",
        manager: "reviewer",
        branch: "branch1",
        rate_sheet: "rate1",
        project_value: 0,
        status: "Active",
        location: "86VHQPRM+5C",
        project_award_date: "2026-09-01",
        project_completion_date: "2026-12-01",
        fn_agreement: false,
        time_and_materials: true,
      };
      if (path === "/api/job_attachments/context")
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({
            job: legacyJob,
            allocations: [{ id: "allocation1", division: "division1", hours: 20 }],
            review_token: "job-review-v1",
          }),
        });
      if (path.includes("/collections/jobs/records/job1"))
        return route.fulfill({ contentType: "application/json", body: JSON.stringify(legacyJob) });
      if (path.includes("/collections/job_time_allocations/records"))
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({
            page: 1,
            totalPages: 1,
            items: [{ id: "allocation1", division: "division1", hours: 20 }],
          }),
        });
      if (path === "/api/job_requests/options")
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({
            clients: [{ id: "client1", name: "Example Client" }],
            contacts: [
              { id: "contact1", client: "client1", name: "Alex Client", email: "alex@example.com" },
            ],
            invoicing_profiles: emptyValueProfiles
              ? []
              : [{ id: "profile1", client: "client1", contact: "contact1" }],
            users: [{ id: "reviewer", name: "Reviewer" }],
            branches: [{ id: "branch1", name: "Thunder Bay" }],
            rate_sheets: [{ id: "rate1", name: "Standard", revision: 1, active: true }],
            divisions: [{ id: "division1", name: "Engineering", code: "ENG", active: true }],
          }),
        });
      if (path === "/api/project_authorization/approvers")
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({
            items: [{ id: "reviewer", name: "Reviewer", max_value: 200000 }],
          }),
        });
      if (path === "/api/job_attachments" && route.request().method() === "POST") {
        valueSaves.push(route.request().postData());
        attachmentItems = [
          {
            id: "newpending",
            job: "job1",
            status: "Pending",
            project_value: 500,
            assigned_approver: "reviewer",
            version: 1,
            document_hash: "hash456",
          },
        ];
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify(attachmentItems[0]),
        });
      }
      if (
        route.request().url().includes("client_invoicing_information") &&
        route.request().method() === "POST"
      ) {
        profileSaves.push(route.request().postDataJSON());
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({ id: "profile1" }),
        });
      }
      if (route.request().url().includes("/api/job_attachments")) {
        if (route.request().method() === "POST") {
          valueActions.push(route.request().postDataJSON());
          attachmentItems = [
            attachmentItems[0],
            {
              ...attachmentItems[1],
              status: "Approved",
              approval_sequence: 2,
              previous_project_value: 120000,
              approval_max_value: 200000,
            },
          ];
        }
        return route.fulfill({
          contentType: "application/json",
          body: JSON.stringify({ items: attachmentItems }),
        });
      }
      if (route.request().method() === "PATCH") {
        const body = route.request().postDataJSON();
        saves.push(body);
        if (body.request.project_value === 130000)
          return route.fulfill({
            status: 400,
            contentType: "application/json",
            body: JSON.stringify({ message: "Selected approver limit is insufficient." }),
          });
      }
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify({ items: [] }),
      });
    });
    await page.goto(`${origin}/tests/fixtures/projectRequests.html`);
    await page.getByText("For the client: Example Client", { exact: true }).waitFor();
    assert.equal(await page.locator(".signature-line").count(), 8);
    assert.equal(await page.getByText("DRAFT - NOT FOR SIGNATURE", { exact: true }).count(), 0);
    assert.match(
      await page.locator("article").innerText(),
      /Project value \(CAD, before tax\): \$120,000/,
    );
    assert.equal(await page.locator(".appendix").count(), 0);
    assert.match(await page.title(), /Project Authorization - request12345678/);
    assert.equal(await page.evaluate(() => window.printCalled), true);
    const text = await page.locator("article").innerText();
    for (const internal of ["approval limit", "allocations", "Draft ->", "labour cost"])
      assert.equal(text.includes(internal), false);
    await page.screenshot({ path: "/tmp/project-request-complete.png", fullPage: true });
    await page.emulateMedia({ media: "print" });
    await page.pdf({ path: "/tmp/project-request-complete.pdf", preferCSSPageSize: true });
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?tm`);
    await page.getByText("Client billing rate sheet — CAD", { exact: true }).waitFor();
    assert.match(await page.locator(".rates").innerText(), /Engineer/);
    assert.match(await page.locator(".rates").innerText(), /\$180.00/);
    assert.match(
      await page.locator("article").innerText(),
      /For time and materials work, the attached rate sheet applies\./,
    );
    await page.pdf({ path: "/tmp/project-request-tm.pdf", preferCSSPageSize: true });
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?draft`);
    await page.getByText("DRAFT - NOT FOR SIGNATURE", { exact: true }).waitFor();
    assert.equal(await page.locator(".signature-line").count(), 0);
    assert.equal(
      await page.locator(".value").innerText(),
      "Project value (CAD, before tax): Not provided",
    );
    await page.screenshot({ path: "/tmp/project-request-draft.png", fullPage: true });
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?long&tm`);
    await page.getByText("END OF SCOPE", { exact: false }).waitFor();
    assert.equal(await page.locator(".signature-line").count(), 8);
    await page.pdf({ path: "/tmp/project-request-long.pdf", preferCSSPageSize: true });
    await page.emulateMedia({ media: "screen" });
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?editor&no-contacts`);
    await page
      .getByText("A job maintainer must add a client contact before you can create a profile.", {
        exact: false,
      })
      .waitFor();
    assert.equal(
      await page.getByRole("combobox", { name: "Invoicing profile", exact: true }).count(),
      0,
    );
    assert.equal(
      await page
        .getByRole("link", { name: "Manage client contacts and invoicing", exact: true })
        .count(),
      0,
    );
    assert.equal(await page.getByRole("button", { name: /Create one/ }).count(), 0);
    assert.equal(await page.getByRole("button", { name: /Add a contact/ }).count(), 0);
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?editor&viewer`);
    await page
      .getByText("Ask a job maintainer to create an invoicing profile for this client.", {
        exact: false,
      })
      .waitFor();
    assert.equal(await page.getByRole("button", { name: /Create one/ }).count(), 0);
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?editor`);
    await page.getByRole("heading", { name: "Invoicing profile", exact: true }).waitFor();
    await page
      .getByLabel("Project / scope of work", { exact: true })
      .fill("Keep this draft while creating invoices.");
    assert.equal(
      await page.getByRole("combobox", { name: "Invoicing profile", exact: true }).count(),
      0,
    );
    assert.equal(
      await page
        .getByRole("link", { name: "Manage client contacts and invoicing", exact: true })
        .count(),
      0,
    );
    await page.evaluate(() => history.replaceState({}, "", "/job-requests/request12345678"));
    await page.getByRole("button", { name: /Create one/ }).click();
    await page.waitForFunction(() => !!window.lastNavigation);
    const createProfileUrl = new URL(await page.evaluate(() => window.lastNavigation), origin);
    assert.equal(createProfileUrl.pathname, "/clients/client1/invoicing/add");
    const createWorkspace = new URL(createProfileUrl.searchParams.get("return_to"), origin);
    const requestReturn = new URL(createWorkspace.searchParams.get("workflow_return"), origin);
    assert.equal(requestReturn.pathname, "/job-requests/request12345678");
    const requestDraft = await page.evaluate(
      (token) => JSON.parse(sessionStorage.getItem(`client-workspace-draft:${token}`)),
      requestReturn.searchParams.get("client_draft"),
    );
    assert.equal(requestDraft.payload.item.description, "Keep this draft while creating invoices.");
    assert.equal(requestDraft.payload.item.contact, "contact1");
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?editor&job&no-contacts`);
    await page.evaluate(() => history.replaceState({}, "", "/job-requests/request12345678"));
    await page.getByRole("button", { name: /Add a contact/ }).click();
    await page.waitForFunction(() => !!window.lastNavigation);
    assert.equal(
      new URL(await page.evaluate(() => window.lastNavigation), origin).pathname,
      "/clients/client1/contacts/add",
    );
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?editor&profiles`);
    await page.getByRole("button", { name: "Save request", exact: true }).waitFor();
    assert.equal(
      await page.getByRole("combobox", { name: "Invoicing profile", exact: true }).count(),
      1,
    );
    assert.match(
      await page
        .getByRole("link", { name: "Manage client contacts and invoicing" })
        .getAttribute("href"),
      /project_contact=contact1/,
    );
    const value = page.getByLabel("Project value (CAD, before tax)");
    await value.fill("1.25");
    assert.equal(await value.evaluate((input) => input.validity.stepMismatch), true);
    await value.fill("120000");
    assert.equal(await value.evaluate((input) => input.checkValidity()), true);
    assert.deepEqual(
      await page.getByLabel("Intended project status").locator("option").allTextContents(),
      ["Active", "Closed"],
    );
    await page.getByRole("button", { name: "Save request", exact: true }).click();
    await page.waitForFunction(
      () => document.querySelector("button[type=submit]")?.textContent === "Save request",
    );
    assert.equal(saves.length, 1);
    assert.equal(saves[0].request.time_and_materials, false);
    assert.equal(Object.hasOwn(saves[0].request, "time_and_materials_set"), false);
    assert.equal(saves[0].request.rate_sheet, "rate1");
    assert.equal(saves[0].version, 3);
    await value.fill("130000");
    await page.getByRole("button", { name: "Save request", exact: true }).click();
    await page
      .getByRole("alert")
      .filter({ hasText: "Selected approver limit is insufficient." })
      .waitFor();
    assert.equal(await value.inputValue(), "130000", "failed edits preserve form input");
    await page.setViewportSize({ width: 390, height: 844 });
    await page.screenshot({ path: "/tmp/project-request-form-mobile.png", fullPage: true });
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?values`);
    await page.getByRole("button", { name: "Approve value", exact: true }).waitFor();
    assert.match(await page.locator("section").innerText(), /Effective project value: \$120,000/);
    assert.match(await page.locator("section").innerText(), /Pending · \$150,000/);
    assert.equal(
      await page.getByRole("button", { name: "Approve value", exact: true }).isDisabled(),
      true,
    );
    assert.equal(
      await page.getByRole("button", { name: "Delete pending change", exact: true }).count(),
      1,
    );
    assert.equal(
      await page
        .getByRole("button", { name: "Delete latest approval and restore value", exact: true })
        .count(),
      0,
    );
    await page.getByLabel("I reviewed this PDF and its full proposed value.").check();
    await page.getByRole("button", { name: "Approve value", exact: true }).click();
    await page
      .getByRole("button", { name: "Delete latest approval and restore value", exact: true })
      .waitFor();
    assert.equal(valueActions.length, 1);
    assert.equal(valueActions[0].version, 4);
    assert.equal(valueActions[0].document_hash, "hash123");
    assert.equal(
      await page
        .getByRole("button", { name: "Delete latest approval and restore value", exact: true })
        .count(),
      1,
      "initial approval has no delete action",
    );
    attachmentItems = [];
    emptyValueProfiles = true;
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?values&legacy`);
    await page.getByRole("button", { name: "Propose a new project value", exact: true }).click();
    await page.getByRole("heading", { name: "Invoicing profile", exact: true }).waitFor();
    assert.equal(
      await page.getByRole("combobox", { name: "Invoicing profile", exact: true }).count(),
      0,
    );
    assert.equal(
      await page
        .getByRole("button", { name: "Manage client contacts and invoicing", exact: true })
        .count(),
      0,
    );
    await page.getByLabel("Proposed project value (CAD, before tax)").fill("750");
    await page.evaluate(() => history.replaceState({}, "", "/jobs/job1/details"));
    await page.getByRole("button", { name: /Create one/ }).click();
    await page.waitForFunction(() => !!window.lastNavigation);
    const valueProfileUrl = new URL(await page.evaluate(() => window.lastNavigation), origin);
    assert.equal(valueProfileUrl.pathname, "/clients/client1/invoicing/add");
    const valueWorkspace = new URL(valueProfileUrl.searchParams.get("return_to"), origin);
    const valueReturn = new URL(valueWorkspace.searchParams.get("workflow_return"), origin);
    const valueDraft = await page.evaluate(
      (token) => JSON.parse(sessionStorage.getItem(`client-workspace-draft:${token}`)),
      valueReturn.searchParams.get("client_draft"),
    );
    assert.equal(valueDraft.kind, "job-value:job1");
    assert.equal(valueDraft.payload.value, 750);
    assert.equal(valueDraft.payload.correctionJob.client, "client1");
    emptyValueProfiles = false;
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?values&legacy`);
    await page.getByRole("button", { name: "Propose a new project value", exact: true }).click();
    await page.getByLabel("Invoicing profile", { exact: true }).selectOption("profile1");
    await page.getByLabel("Proposed project value (CAD, before tax)").fill("500");
    await page.getByLabel("Selected approver", { exact: true }).selectOption("reviewer");
    await page.getByLabel("Authorization PDF", { exact: true }).setInputFiles({
      name: "authorization.pdf",
      mimeType: "application/pdf",
      buffer: Buffer.from("%PDF-1.4\nTest document\n%%EOF"),
    });
    await page.getByRole("button", { name: "Save proposed value", exact: true }).click();
    await page
      .getByRole("button", { name: "Save proposed value", exact: true })
      .waitFor({ state: "hidden" });
    assert.equal(valueSaves.length, 1);
    const part = (name) =>
      valueSaves[0].match(new RegExp(`name="${name}"\\r\\n\\r\\n([^\\r]+)`))?.[1];
    const updates = JSON.parse(part("job_updates"));
    assert.equal(updates.invoicing_information, "profile1");
    for (const forbidden of ["client", "project_value", "number"])
      assert.equal(Object.hasOwn(updates, forbidden), false);
    assert.equal(part("job_review_token"), "job-review-v1");
    assert.equal(part("job_updated"), "2026-09-30 12:00:00.000Z");
    assert.equal(part("project_value"), "500");
    assert.deepEqual(JSON.parse(part("allocations")), [{ division: "division1", hours: 20 }]);
    assert.match(await page.locator("section").innerText(), /Effective project value: \$0/);
    await page.goto(`${origin}/tests/fixtures/projectRequests.html?billing`);
    await page.getByRole("button", { name: "Same as project contact", exact: true }).click();
    assert.equal(
      await page.getByLabel("Send invoices to", { exact: true }).inputValue(),
      "contact1",
    );
    await page.getByRole("button", { name: "Save invoicing profile", exact: true }).click();
    await page
      .getByRole("button", { name: "Save invoicing profile", exact: true })
      .waitFor({ state: "hidden" });
    assert.deepEqual(profileSaves[0], {
      name: "",
      billing_name: "",
      client: "client1",
      contact: "contact1",
      fax: "",
      invoicing_instructions: "",
    });
    assert.deepEqual(errors, []);
  },
  {
    services: {
      "$app/navigation":
        "export const goto = (url) => { window.lastNavigation = url; }; export const beforeNavigate = () => {}; export const invalidateAll = async () => {}; export const replaceState = (url, state) => history.replaceState(state, '', url);",
    },
  },
);
console.log("Project request browser checks passed.");
