<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { goto, invalidateAll, beforeNavigate, replaceState } from "$app/navigation";
  import { fullName } from "$lib/jobRequests";
  import { workflowError, type WorkflowRecord } from "$lib/clientEditors";
  import { untrack } from "svelte";
  import DSLocationPicker from "$lib/components/DSLocationPicker.svelte";
  import { globalStore } from "$lib/stores/global";
  import InvoicingProfileEmptyState from "$lib/components/clients/InvoicingProfileEmptyState.svelte";
  import { canCreateInvoicingProfile } from "$lib/clientWorkspace";
  import {
    type ClientWorkflowTarget,
    clearClientWorkflowDraft,
    keepClientWorkflowDraft,
    readClientWorkflowDraft,
  } from "$lib/clientWorkflowNavigation";
  let {
    request,
    allocations: initialAllocations = [],
    options,
    dirty = $bindable(false),
    onsaved,
  }: {
    request: WorkflowRecord;
    allocations?: WorkflowRecord[];
    options: WorkflowRecord;
    dirty?: boolean;
    onsaved?: () => Promise<void>;
  } = $props();
  const restored = untrack(() =>
    readClientWorkflowDraft<{
      item: WorkflowRecord;
      allocations: WorkflowRecord[];
      fee: string;
      saved: string;
    }>(`request:${request.id || "new"}`, pb.authStore.record?.id || ""),
  );
  let item = $state<WorkflowRecord>(untrack(() => ({ ...(restored?.item || request) })));
  let allocations = $state<WorkflowRecord[]>(
    untrack(() =>
      (restored?.allocations || initialAllocations).map((row) => ({
        division: row.division,
        hours: row.hours,
      })),
    ),
  );
  let fee = $state(
    untrack(
      () =>
        restored?.fee ??
        (request.time_and_materials_set ? (request.time_and_materials ? "tm" : "fixed") : ""),
    ),
  );
  let saved = $state(untrack(() => restored?.saved || JSON.stringify({ item, allocations, fee })));
  let openingClient = false;
  let saving = $state(false),
    error = $state("");
  let approvers = $state<WorkflowRecord[]>([]),
    approverError = $state("");
  const contacts = $derived(
    (options.contacts || []).filter((record: WorkflowRecord) => record.client === item.client),
  );
  const profiles = $derived(
    (options.invoicing_profiles || []).filter(
      (record: WorkflowRecord) => record.client === item.client,
    ),
  );
  const proposals = $derived(
    (options.proposals || []).filter(
      (record: WorkflowRecord) =>
        record.client === item.client &&
        ["In Progress", "Submitted", "Awarded"].includes(record.status),
    ),
  );
  $effect(() => {
    dirty = JSON.stringify({ item, allocations, fee }) !== saved;
  });
  $effect(() => {
    const value = item.project_value;
    let current = true;
    approvers = [];
    if (Number.isInteger(value) && value > 0) {
      pb.send(`/api/project_authorization/approvers?value=${value}`, {
        method: "GET",
        requestKey: null,
      })
        .then((response) => {
          if (current) {
            approvers = response.items;
            approverError = "";
          }
        })
        .catch((e) => {
          if (current) approverError = workflowError(e);
        });
    }
    return () => {
      current = false;
    };
  });
  beforeNavigate(({ cancel }) => {
    if (openingClient) return;
    if (dirty && !saving && !window.confirm("Discard unsaved request changes?")) cancel();
    else clearClientWorkflowDraft();
  });
  async function manageClient(target: ClientWorkflowTarget = "workspace", event?: MouseEvent) {
    if (event && (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)) return;
    event?.preventDefault();
    try {
      const href = keepClientWorkflowDraft(
        `request:${request.id || "new"}`,
        pb.authStore.record?.id || "",
        $state.snapshot({ item, allocations, fee, saved }),
        item.client,
        item.contact || "",
        target,
        (url) => replaceState(url, {}),
      );
      openingClient = true;
      await goto(href);
    } catch (e) {
      error = workflowError(e);
    } finally {
      openingClient = false;
    }
  }
  function clientChanged() {
    item.contact = "";
    item.invoicing_information = "";
    item.proposal = "";
  }
  export async function save(): Promise<boolean> {
    if (saving) return false;
    saving = true;
    error = "";
    const fields = [
      "description",
      "notes",
      "client",
      "contact",
      "manager",
      "alternate_manager",
      "branch",
      "location",
      "job_owner",
      "parent",
      "proposal",
      "project_status",
      "project_value",
      "project_award_date",
      "project_completion_date",
      "client_po",
      "client_reference_number",
      "fn_agreement_answer",
      "invoicing_information",
      "rate_sheet",
      "approver",
    ];
    const payload = Object.fromEntries(
      fields.filter((field) => item[field] !== undefined).map((field) => [field, item[field]]),
    );
    if (fee) {
      payload.time_and_materials = fee === "tm";
    }
    try {
      const result = await pb.send(item.id ? `/api/job_requests/${item.id}` : "/api/job_requests", {
        method: item.id ? "PATCH" : "POST",
        body: { version: item.version, request: payload, allocations },
      });
      saved = JSON.stringify({ item, allocations, fee });
      dirty = false;
      clearClientWorkflowDraft();
      if (!item.id) await goto(`/job-requests/${result.request?.id ?? result.id}`);
      else if (onsaved) await onsaved();
      else await invalidateAll();
      return true;
    } catch (e) {
      error = workflowError(e);
      return false;
    } finally {
      saving = false;
    }
  }
</script>

<form
  class="space-y-5"
  onsubmit={(event) => {
    event.preventDefault();
    save();
  }}
>
  {#if error}<p class="rounded border border-red-300 bg-red-50 p-3 text-red-800" role="alert">
      {error}
    </p>{/if}
  <fieldset disabled={saving} class="grid gap-4 md:grid-cols-2">
    <label class="md:col-span-2"
      >Project / scope of work<textarea rows="4" bind:value={item.description}></textarea></label
    >
    <label class="md:col-span-2"
      >Notes (optional)<textarea rows="3" maxlength="1000" bind:value={item.notes}></textarea><span
        class="text-xs font-normal text-neutral-600"
        >For example, a property's legal description. You can include the notes when printing the
        Project Authorization. They become the job's first note when the job is created. 10 to 1000
        characters.</span
      ></label
    >
    <label
      >Client<select bind:value={item.client} disabled={!!item.parent} onchange={clientChanged}
        ><option value="">Select client</option>{#each options.clients || [] as row}<option
            value={row.id}>{row.name}</option
          >{/each}</select
      ></label
    >
    <label
      >Project contact<select bind:value={item.contact}
        ><option value="">Select project contact</option>{#each contacts as row}<option
            value={row.id}>{fullName(row)} — {row.email}</option
          >{/each}</select
      ></label
    >
    {#if item.client && !profiles.length}
      <div class="flex flex-col gap-1 md:col-span-2">
        <h3 class="text-sm font-semibold">Invoicing profile</h3>
        <InvoicingProfileEmptyState
          canCreate={canCreateInvoicingProfile($globalStore.claims)}
          canManageContacts={$globalStore.claims.includes("job")}
          hasContacts={contacts.length > 0}
          oncreate={() => manageClient("invoicing")}
          onaddcontact={() => manageClient("contact")}
        />
      </div>
    {:else}
      <label
        >Invoicing profile<select bind:value={item.invoicing_information}
          ><option value="">Select invoicing profile</option>{#each profiles as row}<option
              value={row.id}
              >{row.name ||
                fullName(
                  (options.contacts || []).find(
                    (contact: WorkflowRecord) => contact.id === row.contact,
                  ),
                ) ||
                row.id}{row.invoicing_instructions
                ? ` — ${row.invoicing_instructions}`
                : ""}</option
            >{/each}</select
        ></label
      >
      {#if item.client}<a
          class="text-sm text-blue-700 underline"
          href={`/clients/${item.client}/details?project_contact=${encodeURIComponent(item.contact || "")}`}
          onclick={(event) => manageClient("workspace", event)}
          >Manage client contacts and invoicing</a
        >{/if}
    {/if}
    <label
      >Project owner (optional)<select bind:value={item.job_owner}
        ><option value="">No owner selected</option>{#each options.clients || [] as row}<option
            value={row.id}>{row.name}</option
          >{/each}</select
      ></label
    >
    <label
      >Project manager<select bind:value={item.manager}
        ><option value="">Select manager</option>{#each options.users || [] as row}<option
            value={row.id}>{fullName(row)}</option
          >{/each}</select
      ></label
    >
    <label
      >Alternate manager (optional)<select bind:value={item.alternate_manager}
        ><option value="">No alternate manager</option>{#each options.users || [] as row}<option
            value={row.id}>{fullName(row)}</option
          >{/each}</select
      ></label
    >
    <label
      >Branch<select bind:value={item.branch}
        ><option value="">Select branch</option>{#each options.branches || [] as row}<option
            value={row.id}>{row.name}</option
          >{/each}</select
      ></label
    >
    <label
      >Intended project status<select bind:value={item.project_status}
        ><option>Active</option><option>Closed</option></select
      ></label
    >
    <div class="md:col-span-2">
      <DSLocationPicker bind:value={item.location} errors={{}} fieldName="location" />
    </div>
    <label>Project award date<input type="date" bind:value={item.project_award_date} /></label>
    <label
      >Project completion date<input type="date" bind:value={item.project_completion_date} /></label
    >
    <label
      >Project value (CAD, before tax)<input
        type="number"
        min="1"
        step="1"
        bind:value={item.project_value}
      /></label
    >
    <label
      >Fee basis<select bind:value={fee}
        >{#if !request.time_and_materials_set}<option value="">Select fee basis</option>{/if}<option
          value="fixed">Fixed fee</option
        ><option value="tm">Time and materials</option></select
      ></label
    >
    <label
      >Rate sheet<select bind:value={item.rate_sheet}
        ><option value="">Select rate sheet</option
        >{#each (options.rate_sheets || []).filter((row: WorkflowRecord) => row.active) as row}<option
            value={row.id}>{row.name} — Rev. {row.revision}</option
          >{/each}</select
      ></label
    >
    <label
      >First Nations agreement<select bind:value={item.fn_agreement_answer}
        ><option value="">Select an answer</option><option>Yes</option><option>No</option></select
      ></label
    >
    <label
      >Proposal (optional)<select bind:value={item.proposal}
        ><option value="">No proposal</option>{#each proposals as row}<option value={row.id}
            >{row.number} — {row.description} ({row.status})</option
          >{/each}</select
      ></label
    >
    <label>Client PO (optional)<input bind:value={item.client_po} /></label>
    <label
      >Client reference number (optional)<input bind:value={item.client_reference_number} /></label
    >
    <label
      >Selected approver<select bind:value={item.approver}
        ><option value="">Select approver</option
        >{#if item.approver && !approvers.some((row) => row.id === item.approver)}<option
            value={item.approver}>Current approver — check authority</option
          >{/if}{#each approvers as row}<option value={row.id}
            >{row.name} — up to ${row.max_value.toLocaleString("en-CA")}</option
          >{/each}</select
      ></label
    >
  </fieldset>
  {#if approverError}<p role="alert" class="text-red-700">{approverError}</p>{/if}
  <fieldset disabled={saving} class="space-y-3">
    <legend class="font-semibold">Division allocations</legend>
    {#each allocations as allocation, index}<div class="flex items-end gap-3">
        <label class="grow"
          >Division<select bind:value={allocation.division}
            ><option value="">Select division</option
            >{#each (options.divisions || []).filter((row: WorkflowRecord) => row.active) as row}<option
                value={row.id}>{row.code} — {row.name}</option
              >{/each}</select
          ></label
        ><label>Hours<input type="number" min="0" step="1" bind:value={allocation.hours} /></label
        ><button
          type="button"
          class="rounded border px-3 py-2"
          onclick={() => (allocations = allocations.filter((_, i) => i !== index))}>Remove</button
        >
      </div>{/each}
    <button
      type="button"
      class="rounded border px-3 py-2"
      onclick={() => (allocations = [...allocations, { division: "", hours: 0 }])}
      >Add division</button
    >
  </fieldset>
  <p class="text-sm text-neutral-600">
    Drafts can be incomplete. Submission checks all required data, the PDF, allocations, and
    approval authority.
  </p>
  <button disabled={saving} class="rounded bg-blue-700 px-4 py-2 text-white" type="submit"
    >{saving ? "Saving…" : "Save request"}</button
  >
</form>

<style>
  label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-size: 0.9rem;
    font-weight: 600;
  }
  input,
  select,
  textarea {
    width: 100%;
    border: 1px solid #a3a3a3;
    border-radius: 0.2rem;
    padding: 0.5rem;
    background: white;
    color: #171717;
    font-weight: 400;
  }
  :disabled {
    opacity: 0.65;
  }
</style>
