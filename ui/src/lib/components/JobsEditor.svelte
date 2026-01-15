<script lang="ts">
  import { flatpickrAction, fetchClientContacts } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { goto } from "$app/navigation";
  import type { JobsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import DSLocationPicker from "./DSLocationPicker.svelte";
  import { profiles } from "$lib/stores/profiles";
  import { clients } from "$lib/stores/clients";
  import { jobs } from "$lib/stores/jobs";
  import { divisions } from "$lib/stores/divisions";
  import { appConfig, jobsEditingEnabled } from "$lib/stores/appConfig";
  import DsCheck from "$lib/components/DsCheck.svelte";
  import MiniSearch from "minisearch";
  import { onMount } from "svelte";
  import type {
    BranchesResponse,
    ClientContactsResponse,
    DivisionsResponse,
    JobsRecord,
  } from "$lib/pocketbase-types";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { JobsStatusOptions } from "$lib/pocketbase-types";
  let { data }: { data: JobsPageData } = $props();

  // initialize the stores, noop if already initialized
  clients.init();
  profiles.init();
  jobs.init();
  divisions.init();
  appConfig.init();

  let errors = $state({} as Record<string, { message: string }>);
  // Allow extra field `location` introduced by migration to be present on item
  let item = $state(data.item as JobsRecord | (JobsRecord & Record<string, unknown>));
  let categories = $state(data.categories);
  let client_contacts = $state([] as ClientContactsResponse[]);
  let clientContactsRequestId = 0;

  if (item.status === undefined) {
    item.status = JobsStatusOptions.Active;
  }
  if (item.fn_agreement === undefined) {
    item.fn_agreement = false;
  }
  item.description = item.description ?? "";
  item.client = item.client ?? "";
  item.contact = item.contact ?? "";
  item.alternate_manager = item.alternate_manager ?? "";
  item.proposal = item.proposal ?? "";
  item.job_owner = item.job_owner ?? "";
  item.branch = item.branch ?? "";
  item.outstanding_balance = item.outstanding_balance ?? 0;
  item.outstanding_balance_date = item.outstanding_balance_date ?? "";
  item.authorizing_document = item.authorizing_document ?? "";
  item.client_po = item.client_po ?? "";
  item.client_reference_number = item.client_reference_number ?? "";

  let newCategory = $state("");
  let newCategories = $state([] as string[]);
  let categoriesToDelete = $state([] as string[]);

  let branches = $state([] as BranchesResponse[]);
  // Allocations editor state
  type AllocationRow = { division: string; hours: number };
  let allocations = $state([] as AllocationRow[]);

  const alternateManagerErrorMessage = "Alternate manager must be different from manager.";

  const statusOptionsList = Object.values(JobsStatusOptions).map((status) => ({
    id: status,
    name: status,
  }));

  const authorizingDocumentOptions = [
    { id: "Unauthorized", name: "Unauthorized" },
    { id: "PO", name: "PO" },
    { id: "PA", name: "PA" },
  ];

  function jobLabel(job: Pick<JobApiResponse, "id" | "number" | "description">) {
    const numberPart = job.number?.trim();
    const descriptionPart = job.description?.trim();

    const displayNumber = numberPart && numberPart.length > 0 ? numberPart : job.id;
    const displayDescription =
      descriptionPart && descriptionPart.length > 0 ? ` — ${descriptionPart}` : "";

    return `${displayNumber}${displayDescription}`;
  }

  const proposalsIndex = $derived.by(() => {
    if (!$jobs.items || $jobs.items.length === 0) return null;
    const proposals = ($jobs.items as JobApiResponse[]).filter((job) =>
      job.number?.startsWith("P"),
    );
    if (proposals.length === 0) return null;
    const index = new MiniSearch<JobApiResponse>({
      fields: ["number", "description", "client", "id"],
      storeFields: ["id", "number", "description", "client"],
    });
    index.addAll(proposals);
    return index;
  });

  // Hide proposal-only fields when creating a project from the dedicated route.
  // The loader for /jobs/add/from/[proposal] sets _prefilled_from_proposal on item.
  const hideProposalDates = $derived.by(() =>
    Boolean((item as unknown as Record<string, unknown>)._prefilled_from_proposal),
  );

  function setFieldError(fieldName: string, message: string) {
    if (errors[fieldName]?.message === message) {
      return;
    }
    errors = {
      ...errors,
      [fieldName]: { message },
    };
  }

  function clearFieldError(fieldName: string) {
    if (errors[fieldName] === undefined) return;
    const nextErrors = { ...errors };
    delete nextErrors[fieldName];
    errors = nextErrors;
  }

  function addAllocationRow() {
    allocations = [...allocations, { division: "", hours: 0 }];
  }
  function removeAllocationRow(index: number) {
    allocations = allocations.filter((_, i) => i !== index);
  }
  function setAllocationDivision(index: number, value: string | number) {
    const id = value.toString();
    // prevent duplicate divisions
    if (allocations.some((a, i) => i !== index && a.division === id)) {
      return;
    }
    allocations = allocations.map((row, i) => (i === index ? { ...row, division: id } : row));
  }

  onMount(async () => {
    try {
      branches = await pb.collection("branches").getFullList<BranchesResponse>({ sort: "name" });
      // Load allocations when editing
      if ((data as JobsPageData).editing && (data as JobsPageData).id) {
        const list = await pb
          .collection("job_time_allocations")
          .getFullList<{ id: string; job: string; division: string; hours: number }>({
            filter: `job="${(data as JobsPageData).id}"`,
          });
        allocations = list.map((r) => ({ division: r.division, hours: r.hours ?? 0 }));
      }
    } catch (error) {
      console.error("Failed to load branches", error);
    }
  });

  function formatContactName(contact: ClientContactsResponse) {
    const surname = contact.surname?.trim();
    const given = contact.given_name?.trim();
    if (surname && given) return `${surname}, ${given}`;
    if (surname) return surname;
    if (given) return given;
    return contact.email ?? contact.id;
  }

  // Watch for changes to the client and fetch contacts accordingly
  // When the client changes, fetch its contacts. The requestId guard ensures that
  // only the latest fetch updates the state, so rapid client switches don't apply
  // stale contact lists.
  $effect(() => {
    const clientId = item.client;
    const requestId = ++clientContactsRequestId;

    if (!clientId) {
      client_contacts = [];
      if (item.contact) {
        item.contact = "";
      }
      return;
    }

    fetchClientContacts(clientId)
      .then((contacts) => {
        if (requestId !== clientContactsRequestId) return;
        client_contacts = contacts;
        const hasSelectedContact = contacts.some((contact) => contact.id === item.contact);
        if (!hasSelectedContact) {
          item.contact = "";
        }
      })
      .catch((error) => {
        console.error("Failed to load client contacts", error);
        if (requestId !== clientContactsRequestId) return;
        client_contacts = [];
        if (item.contact) {
          item.contact = "";
        }
      });
  });

  $effect(() => {
    if (!item.manager || !item.alternate_manager) {
      if (errors.alternate_manager?.message === alternateManagerErrorMessage) {
        clearFieldError("alternate_manager");
      }
      return;
    }
    if (item.manager === item.alternate_manager) {
      setFieldError("alternate_manager", alternateManagerErrorMessage);
    } else if (errors.alternate_manager?.message === alternateManagerErrorMessage) {
      clearFieldError("alternate_manager");
    }
  });

  // Mirror backend behavior: when not PO, clear any provided client_po
  $effect(() => {
    if (item.authorizing_document !== "PO" && item.client_po) {
      item.client_po = "";
      if (errors.client_po) clearFieldError("client_po");
    }
  });

  async function save(event: Event) {
    event.preventDefault();

    errors = {};
    if (item.manager && item.alternate_manager && item.manager === item.alternate_manager) {
      setFieldError("alternate_manager", alternateManagerErrorMessage);
      return;
    }

    try {
      let jobId = data.id;

      if (data.editing && jobId !== null) {
        await pb.send(`/api/jobs/${jobId}`, {
          method: "PUT",
          body: { job: item, allocations },
        });
      } else {
        const resp = (await pb.send(`/api/jobs`, {
          method: "POST",
          body: { job: item, allocations },
        })) as { id: string };
        jobId = resp.id;
      }

      // Add new categories
      for (const categoryName of newCategories) {
        await pb.collection("categories").create(
          {
            job: jobId,
            name: categoryName.trim(),
          },
          { returnRecord: true },
        );
      }

      // Remove deleted categories
      for (const categoryId of categoriesToDelete) {
        await pb.collection("categories").delete(categoryId);
      }

      errors = {};
      // Redirect to job details for new jobs, jobs list for edits
      if (data.editing && data.id !== null) {
        goto("/jobs/list");
      } else {
        goto(`/jobs/${jobId}/details`);
      }
    } catch (error: unknown) {
      // Handle special case where backend requires setting proposal to Awarded first
      const pocket = error as {
        data?: {
          data?: Record<string, { message: string; code?: string; data?: Record<string, string> }>;
        };
      };
      const hookErrors = pocket?.data?.data;
      const proposalErr = hookErrors?.proposal;
      const proposalId = proposalErr?.data?.proposal_id;
      if (proposalErr?.code === "proposal_not_awarded" && typeof proposalId === "string") {
        const proceed =
          typeof window !== "undefined" &&
          window.confirm("The referenced proposal is Active. Set it to Awarded and continue?");
        if (proceed) {
          try {
            await pb.collection("jobs").update(proposalId, { status: JobsStatusOptions.Awarded });
            // retry create/update once
            let retryJobId = (data as JobsPageData).id;
            if ((data as JobsPageData).editing && retryJobId !== null) {
              await pb.collection("jobs").update(retryJobId, item);
            } else {
              const createdJob = await pb.collection("jobs").create(item);
              retryJobId = createdJob.id;
            }
            // continue categories changes as usual
            for (const categoryName of newCategories) {
              await pb
                .collection("categories")
                .create({ job: retryJobId!, name: categoryName.trim() }, { returnRecord: true });
            }
            for (const categoryId of categoriesToDelete) {
              await pb.collection("categories").delete(categoryId);
            }
            errors = {};
            // Redirect to job details for new jobs, jobs list for edits
            if ((data as JobsPageData).editing && (data as JobsPageData).id !== null) {
              goto("/jobs/list");
            } else {
              goto(`/jobs/${retryJobId}/details`);
            }
            return;
          } catch (retryErr) {
            // fall through to display errors from retry
            const retryData = (
              retryErr as {
                data?: { data?: Record<string, { message: string }> };
              }
            )?.data?.data;
            errors = retryData ?? {};
            return;
          }
        }
      }
      const backendErrors = pocket?.data?.data as Record<string, { message: string }> | undefined;
      errors = backendErrors ?? {};
    }
  }

  async function addCategory() {
    if (newCategory.trim() === "") return;

    newCategories.push(newCategory.trim());
    newCategory = "";
  }

  async function removeCategory(categoryId: string) {
    categoriesToDelete.push(categoryId);
    categories = categories.filter((category) => category.id !== categoryId);
  }

  function preventDefault(fn: (event: Event) => void) {
    return (event: Event) => {
      event.preventDefault();
      fn(event);
    };
  }

  function cancel() {
    if (data.editing && data.id !== null) {
      goto(`/jobs/${data.id}/details`);
    } else {
      goto("/jobs/list");
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
</svelte:head>

{#if !$jobsEditingEnabled}
  <div class="disabled-notice">
    <p>Job creation and editing is temporarily disabled during a system transition.</p>
    <p>Please check back later or contact an administrator if you need immediate assistance.</p>
  </div>
{:else}
  <form
    class="flex w-full flex-col items-center gap-2 p-2"
    enctype="multipart/form-data"
    onsubmit={save}
  >
  <span class="flex w-full gap-2 {errors.project_award_date !== undefined ? 'bg-red-200' : ''}">
    <label for="project_award_date">Project Award Date</label>
    <input
      class="flex-1"
      type="text"
      name="project_award_date"
      placeholder="Project Award Date"
      use:flatpickrAction
      bind:value={item.project_award_date}
    />
    {#if errors.project_award_date !== undefined}
      <span class="text-red-600">{errors.project_award_date.message}</span>
    {/if}
  </span>

  {#if !hideProposalDates}
    <span
      class="flex w-full gap-2 {errors.proposal_opening_date !== undefined ? 'bg-red-200' : ''}"
    >
      <label for="proposal_opening_date">Proposal Opening Date</label>
      <input
        class="flex-1"
        type="text"
        name="proposal_opening_date"
        placeholder="Proposal Opening Date"
        use:flatpickrAction
        bind:value={item.proposal_opening_date}
      />
      {#if errors.proposal_opening_date !== undefined}
        <span class="text-red-600">{errors.proposal_opening_date.message}</span>
      {/if}
    </span>

    <span
      class="flex w-full gap-2 {errors.proposal_submission_due_date !== undefined
        ? 'bg-red-200'
        : ''}"
    >
      <label for="proposal_submission_due_date">Proposal Submission Due Date</label>
      <input
        class="flex-1"
        type="text"
        name="proposal_submission_due_date"
        placeholder="Proposal Submission Due Date"
        use:flatpickrAction
        bind:value={item.proposal_submission_due_date}
      />
      {#if errors.proposal_submission_due_date !== undefined}
        <span class="text-red-600">{errors.proposal_submission_due_date.message}</span>
      {/if}
    </span>
  {/if}

  <div class="flex w-full flex-col gap-1 {errors.location !== undefined ? 'bg-red-200' : ''}">
    <label for="location">Location</label>
    <DSLocationPicker bind:value={item.location as string} {errors} fieldName="location" />
  </div>

  <!---
  <DsTextInput
    bind:value={item.number as string}
    {errors}
    fieldName="number"
    uiName="Number"
    disabled={true}
  />
  <p class="self-start text-xs text-neutral-600">Number is auto-assigned on creation.</p>
-->
  <DsTextInput
    bind:value={item.description as string}
    {errors}
    fieldName="description"
    uiName="Description"
  />

  {#if $clients.index !== null}
    <DsAutoComplete
      bind:value={item.client as string}
      index={$clients.index}
      {errors}
      fieldName="client"
      uiName="Client"
      disabled={(data as JobsPageData).editing === false &&
        (item as any).parent &&
        (item as any).parent !== ""}
    >
      {#snippet resultTemplate(client)}{client.name}{/snippet}
    </DsAutoComplete>
  {/if}

  <div class="flex w-full gap-2 {errors.contact !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <label for="contact">Client Contact</label>
      <select
        id="contact"
        name="contact"
        bind:value={item.contact}
        class="flex-1 rounded border border-neutral-300 px-1"
        disabled={item.client === "" || client_contacts.length === 0}
      >
        <option value="">{item.client === "" ? "Select a client first" : "Select a contact"}</option
        >
        {#each client_contacts as contact}
          <option value={contact.id}>{formatContactName(contact)}</option>
        {/each}
      </select>
    </span>
    {#if errors.contact !== undefined}
      <span class="text-red-600">{errors.contact.message}</span>
    {/if}
  </div>

  {#if $profiles.index !== null}
    <DsAutoComplete
      bind:value={item.manager as string}
      index={$profiles.index}
      idField="uid"
      {errors}
      fieldName="manager"
      uiName="Manager"
    >
      {#snippet resultTemplate(item)}
        {item.surname}, {item.given_name}
      {/snippet}
    </DsAutoComplete>
  {/if}
  {#if $profiles.index !== null}
    <DsAutoComplete
      bind:value={item.alternate_manager as string}
      index={$profiles.index}
      idField="uid"
      {errors}
      fieldName="alternate_manager"
      uiName="Alternate Manager"
      excludeIds={item.manager ? [item.manager] : []}
    >
      {#snippet resultTemplate(item)}
        {item.surname}, {item.given_name}
      {/snippet}
    </DsAutoComplete>
  {/if}

  <DsCheck
    bind:value={item.fn_agreement as boolean}
    {errors}
    fieldName="fn_agreement"
    uiName="Has First Nation's agreement"
  />

  <DsSelector
    bind:value={item.status as string}
    items={statusOptionsList}
    {errors}
    fieldName="status"
    uiName="Status"
  >
    {#snippet optionTemplate(item)}{item.name}{/snippet}
  </DsSelector>
  <p
    class="cursor-help self-start text-sm text-neutral-600"
    title="Use the status Closed if the purpose of this job is to act as a reporting container for many sub jobs. These “Parent” jobs can be billed to if their status is set to “Active”, however they are created as “Closed” by default. For example MTO retainers are usually created as Closed."
  >
    Creating a parent job? Use Closed.
  </p>

  <DsSelector
    bind:value={item.authorizing_document}
    items={authorizingDocumentOptions}
    {errors}
    fieldName="authorizing_document"
    uiName="Authorizing Document"
  >
    {#snippet optionTemplate(item)}{item.name}{/snippet}
  </DsSelector>
  {#if item.authorizing_document === "PO"}
    <DsTextInput bind:value={item.client_po} {errors} fieldName="client_po" uiName="Client PO" />
  {/if}
  <DsTextInput
    bind:value={item.client_reference_number}
    {errors}
    fieldName="client_reference_number"
    uiName="Client Reference Number"
  />

  <div
    class="flex w-full flex-col gap-1"
    class:bg-red-200={errors.outstanding_balance !== undefined}
  >
    <label class="text-sm font-semibold" for="outstanding_balance">Outstanding Balance</label>
    <input
      id="outstanding_balance"
      name="outstanding_balance"
      type="number"
      class="rounded border border-neutral-300 px-2 py-1"
      bind:value={item.outstanding_balance as number}
      min={0}
      step={0.01}
    />
    {#if errors.outstanding_balance !== undefined}
      <span class="text-sm text-red-600">{errors.outstanding_balance.message}</span>
    {/if}
  </div>
  {#if item.outstanding_balance_date}
    <p class="self-start text-sm text-neutral-600">
      Last updated: {item.outstanding_balance_date}
    </p>
  {/if}

  {#if proposalsIndex !== null}
    <DsAutoComplete
      bind:value={item.proposal as string}
      index={proposalsIndex}
      {errors}
      fieldName="proposal"
      uiName="Proposal"
    >
      {#snippet resultTemplate(job)}{jobLabel(
          job as unknown as Pick<JobApiResponse, "id" | "number" | "description">,
        )}{/snippet}
    </DsAutoComplete>
  {/if}

  {#if $divisions.index !== null}
    <div class="flex w-full flex-col gap-2">
      <span class="font-semibold">Divisions</span>
      <div class="flex flex-col gap-2">
        {#each allocations as row, idx}
          <div class="flex items-center gap-2">
            <div class="min-w-[280px] flex-1">
              <DsAutoComplete
                bind:value={row.division}
                index={$divisions.index}
                {errors}
                fieldName={"allocation_division_" + idx}
                uiName="Division"
                choose={(id) => setAllocationDivision(idx, id)}
              >
                {#snippet resultTemplate(item: DivisionsResponse)}{item.code} - {item.name}{/snippet}
              </DsAutoComplete>
            </div>
            <input
              class="w-28 rounded border border-neutral-300 px-2 py-1"
              type="number"
              min={0}
              step={0.5}
              bind:value={row.hours}
              oninput={(e) => {
                const v = parseFloat((e.target as HTMLInputElement).value || "0");
                allocations = allocations.map((r, i) =>
                  i === idx ? { ...r, hours: isNaN(v) ? 0 : v } : r,
                );
              }}
            />
            <button
              class="text-neutral-600"
              onclick={preventDefault(() => removeAllocationRow(idx))}
              title="Remove"
            >
              &times;
            </button>
          </div>
        {/each}
      </div>
      <div>
        <DsActionButton action={addAllocationRow} icon="feather:plus-circle" color="green"
          >Add division</DsActionButton
        >
      </div>
    </div>
  {/if}

  {#if $clients.index !== null}
    <DsAutoComplete
      bind:value={item.job_owner as string}
      index={$clients.index}
      {errors}
      fieldName="job_owner"
      uiName="Job Owner"
    >
      {#snippet resultTemplate(item)}{item.name}{/snippet}
    </DsAutoComplete>
  {/if}

  <div class="flex w-full gap-2 {errors.branch !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <label for="branch">Branch</label>
      <select
        id="branch"
        name="branch"
        bind:value={item.branch}
        class="flex-1 rounded border border-neutral-300 px-1"
      >
        <option value="">Select a branch</option>
        {#each branches as branch}
          <option value={branch.id}>{branch.code ?? branch.name}</option>
        {/each}
      </select>
    </span>
    {#if errors.branch !== undefined}
      <span class="text-red-600">{errors.branch.message}</span>
    {/if}
  </div>

  <div class="flex w-full flex-col gap-2 {errors.categories !== undefined ? 'bg-red-200' : ''}">
    <label for="categories">Categories</label>
    <div class="flex flex-wrap gap-1">
      {#each categories as category}
        <span class="flex items-center rounded-full bg-neutral-200 px-2">
          <span>{category.name}</span>
          <button
            class="text-neutral-500"
            onclick={preventDefault(() => removeCategory(category.id))}
          >
            &times;
          </button>
        </span>
      {/each}
      {#each newCategories as categoryName}
        <span class="flex items-center rounded-full bg-neutral-200 px-2">
          <span>{categoryName}</span>
          <button
            class="text-neutral-500"
            onclick={preventDefault(
              () => (newCategories = newCategories.filter((name) => name !== categoryName)),
            )}
          >
            &times;
          </button>
        </span>
      {/each}
    </div>
    <div class="flex items-center gap-1">
      <DsTextInput
        bind:value={newCategory as string}
        {errors}
        fieldName="newCategory"
        uiName="Add Category"
      />
      <DsActionButton
        action={addCategory}
        icon="feather:plus-circle"
        color="green"
        title="Add Category"
      />
    </div>
    {#if errors.categories !== undefined}
      <span class="text-red-600">{errors.categories.message}</span>
    {/if}
  </div>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action={cancel}>Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
  </form>
{/if}

<style>
  .disabled-notice {
    padding: 1.5rem;
    background-color: #fff3cd;
    border: 1px solid #ffc107;
    border-radius: 0.5rem;
    margin: 1rem;
    max-width: 600px;
  }

  .disabled-notice p {
    margin: 0.5rem 0;
    color: #856404;
  }
</style>
