<script lang="ts">
  import type { ClientWorkspaceData } from "$lib/clientWorkspaceLoad";
  import { untrack } from "svelte";
  import { navigating, page } from "$app/state";
  import { goto, replaceState } from "$app/navigation";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import HelpPopover from "$lib/components/jobs/TimeSummaryHelp.svelte";
  import DSTabBar from "$lib/components/DSTabBar.svelte";
  import ClientWorkspaceShell from "$lib/components/ClientWorkspaceShell.svelte";
  import ClientsEditor from "$lib/components/ClientsEditor.svelte";
  import ClientNotesSection from "$lib/components/ClientNotesSection.svelte";
  import ClientSaveNotice from "$lib/components/clients/ClientSaveNotice.svelte";
  import InvoicingInstructions from "$lib/components/clients/InvoicingInstructions.svelte";
  import { shortDate, formatCurrency } from "$lib/utilities";
  import { globalStore } from "$lib/stores/global";
  import {
    addressText,
    invoicingEmptyMessage,
    profileCreatorName,
    currentWorkspaceUrl,
    contactName,
    contactSearchText,
    countLabel,
    invoicingSearchText,
    parseClientWorkspaceQuery,
    savedListPosition,
    workspaceHref,
    type WorkspaceInvoicing,
    type WorkspaceContact,
    billingName,
  } from "$lib/clientWorkspace";
  import type { ClientsRecord, JobsRecord } from "$lib/pocketbase-types";

  const { data } = $props<{ data: ClientWorkspaceData }>();
  const workspaceUrl = $derived(currentWorkspaceUrl(data.client.id, page.url, page.state));
  const listState = $derived(parseClientWorkspaceQuery(workspaceUrl.searchParams));
  const canEdit = $derived($globalStore.claims.includes("job"));
  const canMerge = $derived($globalStore.claims.includes("absorb"));
  const contacts = $derived(data.client.contacts as WorkspaceContact[]);
  const profiles = $derived(data.client.invoicing_profiles as WorkspaceInvoicing[]);
  const jobs = $derived(data.jobs as JobsRecord[]);
  const filteredProfiles = $derived(
    listState.recipient
      ? profiles.filter((profile) => profile.contact === listState.recipient)
      : profiles,
  );
  const selectedRecipient = $derived(
    contacts.find((contact) => contact.id === listState.recipient),
  );
  let notice = $state<{
    clientId: string;
    kind: "contact" | "profile";
    id: string;
    highlighted: boolean;
  } | null>(null);
  const savedContact = $derived(
    notice && notice.clientId === data.client.id && notice.kind === "contact"
      ? contacts.find((contact) => contact.id === notice?.id)
      : undefined,
  );
  const savedProfile = $derived(
    notice && notice.clientId === data.client.id && notice.kind === "profile"
      ? profiles.find((profile) => profile.id === notice?.id)
      : undefined,
  );
  const contactPosition = $derived(
    savedListPosition(
      contacts,
      savedContact?.id,
      listState.contactsQuery,
      contactSearchText,
      listState.contactsPage,
      listState.contactsPerPage,
    ),
  );
  const profileRecipientMatches = $derived(
    !listState.recipient || savedProfile?.contact === listState.recipient,
  );
  const profilePosition = $derived(
    savedListPosition(
      profileRecipientMatches ? filteredProfiles : profiles,
      savedProfile?.id,
      listState.invoicingQuery,
      (profile) => invoicingSearchText(profile, contacts),
      listState.invoicingPage,
      listState.invoicingPerPage,
    ),
  );
  const savedContactHref = $derived(
    contactPosition && !contactPosition.visible
      ? href({
          contacts_q: contactPosition.clearSearch ? null : listState.contactsQuery,
          contacts_page: contactPosition.page,
        })
      : undefined,
  );
  const savedProfileHref = $derived(
    profilePosition && (!profileRecipientMatches || !profilePosition.visible)
      ? href({
          invoicing_q: profilePosition.clearSearch ? null : listState.invoicingQuery,
          recipient: profileRecipientMatches ? listState.recipient : null,
          invoicing_page: profilePosition.page,
        })
      : undefined,
  );

  // Wait for navigation to finish before replacing its history entry.
  // Consume the return marker so refresh and Back do not replay an old save.
  $effect(() => {
    if (navigating.to) return;
    const url = workspaceUrl;
    const clientId = data.client.id;
    const contactId = url.searchParams.get("saved_contact");
    const profileId = url.searchParams.get("saved_invoicing");
    const highlightId = url.searchParams.get("show_invoicing");
    if (!contactId && !profileId && !highlightId) return;
    // SvelteKit restores the hash URL in its queued focus-reset task.
    // Run after that task so it cannot restore the consumed save marker.
    const timer = setTimeout(() => {
      notice = {
        clientId,
        kind: contactId ? "contact" : "profile",
        id: contactId || profileId || highlightId!,
        highlighted: !contactId && !profileId && !!highlightId,
      };
      const cleaned = new URL(url);
      cleaned.searchParams.delete("saved_contact");
      cleaned.searchParams.delete("saved_invoicing");
      cleaned.searchParams.delete("show_invoicing");
      replaceWorkspaceUrl(`${cleaned.pathname}${cleaned.search}${cleaned.hash}`);
    }, 0);
    return () => clearTimeout(timer);
  });
  $effect(() => {
    const clientId = data.client.id;
    const editing = workspaceUrl.searchParams.has("edit");
    untrack(() => {
      if (notice && (notice.clientId !== clientId || editing)) notice = null;
    });
  });
  function showSaved(next: string) {
    replaceWorkspaceUrl(next);
    if (notice) notice = { ...notice };
  }
  const leadName = $derived(
    [data.client.lead_given_name, data.client.lead_surname].filter(Boolean).join(" ") ||
      "Not assigned",
  );

  function href(changes: Record<string, string | number | null>) {
    return workspaceHref(data.client.id, workspaceUrl.searchParams, changes, data.tab);
  }
  function editorHref(path: string) {
    const query = new URLSearchParams({ return_to: href({ edit: null }) });
    const projectContact = workspaceUrl.searchParams.get("project_contact");
    if (projectContact) query.set("project_contact", projectContact);
    return `/clients/${data.client.id}/${path}?${query}`;
  }
  function replaceWorkspaceUrl(next: string) {
    replaceState(next, { ...page.state, clientWorkspaceUrl: next });
  }
  function updateList(
    list: "contacts" | "invoicing",
    next: { searchTerm: string; page: number; perPage: number },
  ) {
    replaceWorkspaceUrl(
      href({
        [`${list}_q`]: next.searchTerm,
        [`${list}_page`]: next.page,
        [`${list}_per_page`]: next.perPage,
      }),
    );
  }
  function profileRecipient(profile: WorkspaceInvoicing) {
    return contactName(contacts.find((contact) => contact.id === profile.contact));
  }
  // "2 jobs · Created Oct 1, 2026 by Alex Adams", omitting an unknown creator.
  function profileUsage(profile: WorkspaceInvoicing) {
    const creator = profileCreatorName(profile);
    const created = profile.created
      ? `Created ${shortDate(profile.created, true)}${creator ? ` by ${creator}` : ""}`
      : "";
    return [countLabel(profile.job_count, "job"), created].filter(Boolean).join(" · ");
  }
  // Unnamed profiles are known by their recipient.
  function profileName(profile: WorkspaceInvoicing) {
    return profile.name || `Send invoices to ${profileRecipient(profile)}`;
  }
  function jobHref(view: string, requestedPage?: number) {
    return href({
      view: view === "projects" ? null : view,
      ...(requestedPage ? { [`${view}Page`]: requestedPage } : {}),
    });
  }
  function closeClientEditor(saved = false) {
    return goto(href({ edit: null }), { invalidateAll: saved });
  }
</script>

<ClientWorkspaceShell clientId={data.client.id} clientName={data.client.name} activeTab={data.tab}>
  {#if data.tab === "details"}
    {#if workspaceUrl.searchParams.get("edit") === "client" && canEdit}
      {#key data.client.id}
        <ClientsEditor
          data={{
            item: { ...data.client } as ClientsRecord,
            editing: true,
            id: data.client.id,
            client_contacts: [],
          }}
          onSaved={() => closeClientEditor(true)}
          onCancel={() => closeClientEditor()}
        />
      {/key}
    {:else}
      <section aria-label="Client information" class="space-y-3 rounded-sm bg-neutral-100 p-3">
        <div class="flex items-center justify-between gap-2">
          <h2 class="font-semibold">Client information</h2>
          {#if canEdit}<DsActionButton
              action={href({ edit: "client" })}
              icon="mdi:pencil"
              title="Edit client information"
              color="blue"
            />{/if}
        </div>
        <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {#if data.client.alias?.trim()}<div>
              <h3 class="text-sm font-semibold text-neutral-600">Alias</h3>
              <p>{data.client.alias}</p>
            </div>{/if}
          <div>
            <h3 class="text-sm font-semibold text-neutral-600">Address</h3>
            <p>{addressText(data.client) || "Not recorded"}</p>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-neutral-600">Phone</h3>
            <p>{data.client.phone || "Not recorded"}</p>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-neutral-600">Business Development Lead</h3>
            <p>{leadName}</p>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-neutral-600">Outstanding Balance</h3>
            <p>{formatCurrency(data.client.outstanding_balance ?? 0)}</p>
            {#if data.client.outstanding_balance_date}<p class="text-xs text-neutral-500">
                As of {shortDate(data.client.outstanding_balance_date, true)}
              </p>{/if}
          </div>
        </div>
      </section>
      <div class="grid min-w-0 items-start gap-6 xl:grid-cols-2">
        <section aria-label="Contacts" id="contacts" class="min-w-0 space-y-2">
          <div class="flex items-center justify-between gap-2">
            <h2 class="font-semibold">Contacts ({contacts.length})</h2>
            {#if canEdit}<DsActionButton
                action={editorHref("contacts/add")}
                icon="mdi:plus"
                title="Add contact"
                color="green"
              />{/if}
          </div>
          {#if savedContact}
            {#key notice}
              <ClientSaveNotice
                message="Contact saved."
                linkLabel="Show contact"
                href={savedContactHref}
                onShow={() => savedContactHref && showSaved(savedContactHref)}
                onExpire={() => (notice = null)}
              />
            {/key}
          {/if}
          <DsList
            items={contacts}
            highlightId={savedContact?.id}
            search
            pagination
            searchText={contactSearchText}
            searchPlaceholder="Filter contacts"
            searchTerm={listState.contactsQuery}
            page={listState.contactsPage}
            perPage={listState.contactsPerPage}
            onStateChange={(next) => updateList("contacts", next)}
            emptyMessage={contacts.length
              ? "No contacts match this filter."
              : "No contacts recorded."}
          >
            {#snippet headline(contact)}
              {#if canEdit}<a
                  class="text-blue-700 hover:underline"
                  href={editorHref(`contacts/${contact.id}/edit`)}>{contactName(contact)}</a
                >{:else}{contactName(contact)}{/if}
            {/snippet}
            {#snippet line1(contact)}<a
                class="break-all text-blue-700 hover:underline"
                href={`mailto:${contact.email}`}>{contact.email}</a
              >{#if contact.phone}<span class="ml-2 whitespace-nowrap text-neutral-600"
                  >{contact.phone}</span
                >{/if}{/snippet}
            {#snippet line2(contact)}
              <span class="text-xs text-neutral-600">{countLabel(contact.job_count, "job")}</span>
              {#if contact.profile_count > 0}<a
                  class="ml-2 text-xs text-blue-700 hover:underline"
                  href={href({ recipient: contact.id, invoicing_q: null, invoicing_page: 1 })}
                  >{countLabel(contact.profile_count, "invoicing profile")}</a
                >{/if}
            {/snippet}
            {#snippet actions(contact)}
              {#if canMerge}<DsActionButton
                  action={editorHref(`contacts/${contact.id}/absorb`)}
                  icon="mdi:merge"
                  title={`Merge contacts into ${contactName(contact)}`}
                  color="yellow"
                />{/if}
              {#if canEdit}<DsActionButton
                  action={editorHref(`contacts/${contact.id}/edit`)}
                  icon="mdi:pencil"
                  title={`Edit ${contactName(contact)}`}
                  color="blue"
                />{/if}
            {/snippet}
          </DsList>
        </section>
        <section aria-label="Invoicing profiles" id="invoicing" class="min-w-0 space-y-2">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-1">
              <h2 class="font-semibold">Invoicing profiles ({profiles.length})</h2>
              <HelpPopover title="About invoicing profiles" iconOnly>
                <p>
                  An invoicing profile sets who receives invoices for this client and any invoice
                  instructions. The billing address comes from that contact, or from the client if
                  the contact has no address.
                </p>
                <p>
                  An invoicing profile is required when creating or editing a project, including a
                  parent project. It is optional for proposals. Existing jobs can keep their current
                  details until edited. Many jobs can use the same profile.
                </p>
                <p>
                  Changes apply to all jobs that use the profile. Create a separate profile when one
                  job needs different invoice details.
                </p>
              </HelpPopover>
            </div>
            {#if canEdit && contacts.length > 0}<DsActionButton
                action={editorHref("invoicing/add")}
                icon="mdi:plus"
                title="Add invoicing profile"
                color="green"
              />{/if}
          </div>
          {#if listState.recipient}
            <p class="rounded-sm bg-neutral-100 p-2 text-sm">
              For {contactName(selectedRecipient)}
              <a
                class="ml-2 text-blue-700 underline"
                href={href({ recipient: null, invoicing_page: 1 })}>Show all recipients</a
              >
            </p>
          {/if}
          {#if savedProfile}
            {#key notice}
              <ClientSaveNotice
                message={notice?.highlighted ? "Profile highlighted." : "Profile saved."}
                linkLabel="Show profile"
                href={savedProfileHref}
                onShow={() => savedProfileHref && showSaved(savedProfileHref)}
                onExpire={() => (notice = null)}
              />
            {/key}
          {/if}
          <DsList
            items={filteredProfiles}
            highlightId={savedProfile?.id}
            search
            pagination
            searchText={(profile) => invoicingSearchText(profile, contacts)}
            searchPlaceholder="Filter invoicing profiles"
            searchTerm={listState.invoicingQuery}
            page={listState.invoicingPage}
            perPage={listState.invoicingPerPage}
            onStateChange={(next) => updateList("invoicing", next)}
            emptyMessage={profiles.length
              ? "No invoicing profiles match this filter."
              : invoicingEmptyMessage(canEdit, contacts.length > 0)}
          >
            {#snippet headline(profile)}
              {#if canEdit}<a
                  class="text-blue-700 hover:underline"
                  href={editorHref(`invoicing/${profile.id}/edit`)}>{profileName(profile)}</a
                >{:else}{profileName(profile)}{/if}
            {/snippet}
            {#snippet line1(profile)}
              {@const contact = contacts.find((row) => row.id === profile.contact)}
              {#if profile.name}
                <span class="text-sm">Send invoices to </span>
                {#if canEdit && contact}<a
                    class="text-blue-700 hover:underline"
                    href={editorHref(`contacts/${contact.id}/edit`)}>{contactName(contact)}</a
                  >{:else}{contactName(contact)}{/if}
              {/if}
              {#if data.client.alias?.trim()}<p class="text-sm">
                  Name on PAs and invoices: {billingName(data.client, profile)}
                </p>{/if}
            {/snippet}
            {#snippet line2(profile)}
              {#key profile.id}
                <InvoicingInstructions text={profile.invoicing_instructions || ""} />
              {/key}
            {/snippet}
            {#snippet line3(profile)}
              <p class="text-xs text-neutral-600">{profileUsage(profile)}</p>
            {/snippet}
            {#snippet actions(profile)}
              {#if canEdit}<DsActionButton
                  action={editorHref(`invoicing/${profile.id}/edit`)}
                  icon="mdi:pencil"
                  title={`Edit invoicing profile ${profile.name || `for ${profileRecipient(profile)}`}`}
                  color="blue"
                />{/if}
            {/snippet}
          </DsList>
        </section>
      </div>
    {/if}
  {:else if data.tab === "jobs"}
    <section class="space-y-2" aria-label="Jobs">
      <div class="overflow-x-auto">
        <DSTabBar
          tabs={[
            {
              label: `Projects (${data.counts.projects})`,
              href: jobHref("projects"),
              active: data.view === "projects",
            },
            {
              label: `Proposals (${data.counts.proposals})`,
              href: jobHref("proposals"),
              active: data.view === "proposals",
            },
            {
              label: `Jobs as owner (${data.counts.owner})`,
              href: jobHref("owner"),
              active: data.view === "owner",
            },
          ]}
        />
      </div>
      <DsList items={jobs}>
        {#snippet anchor(job)}<a
            href={`/jobs/${job.id}/details`}
            class="text-blue-600 hover:underline">{job.number}</a
          >{/snippet}
        {#snippet headline(job)}{job.description}{/snippet}
        {#snippet byline(job)}<span class="opacity-60"
            >{job.created ? shortDate(job.created) : ""}</span
          >{/snippet}
      </DsList>
      {#if !data.jobs.length}<p class="p-2 italic">No jobs in this list.</p>{/if}
      {#if data.totalPages > 1}
        <nav
          aria-label="Job pages"
          class="flex items-center justify-between gap-2 bg-neutral-100 p-2"
        >
          <span>Page {data.page} / {data.totalPages}</span>
          <div class="flex gap-2">
            {#each [{ label: "← Prev", target: data.page - 1 }, { label: "Next →", target: data.page + 1 }] as step}
              {#if step.target >= 1 && step.target <= data.totalPages}<a
                  class="rounded-sm bg-neutral-200 px-2 py-1 hover:bg-neutral-300"
                  href={jobHref(data.view, step.target)}>{step.label}</a
                >{:else}<span class="rounded-sm bg-neutral-200 px-2 py-1 opacity-40"
                  >{step.label}</span
                >{/if}
            {/each}
          </div>
        </nav>
      {/if}
    </section>
  {:else}
    {#key data.client.id}<ClientNotesSection
        clientId={data.client.id}
        collapsible={false}
        notes={data.notes}
        jobOptions={data.noteJobs}
        notesEndpoint={`/api/clients/${data.client.id}/notes`}
      />{/key}
  {/if}
</ClientWorkspaceShell>
