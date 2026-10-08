<script lang="ts">
  import type { ClientWorkflowTarget } from "$lib/clientWorkflowNavigation";
  import type { WorkflowRecord } from "$lib/clientEditors";
  import { fullName } from "$lib/jobRequests";
  import { canCreateInvoicingProfile } from "$lib/clientWorkspace";
  import { globalStore } from "$lib/stores/global";
  import DSLocationPicker from "$lib/components/DSLocationPicker.svelte";
  import InvoicingProfileEmptyState from "$lib/components/clients/InvoicingProfileEmptyState.svelte";
  let {
    job = $bindable(),
    allocations = $bindable(),
    options,
    originalRate = "",
    onmanageclient,
  }: {
    job: WorkflowRecord;
    allocations: WorkflowRecord[];
    options: WorkflowRecord;
    originalRate?: string;
    onmanageclient?: (target?: ClientWorkflowTarget) => void;
  } = $props();
  const contacts = $derived(
    (options.contacts || []).filter((row: WorkflowRecord) => row.client === job.client),
  );
  const profiles = $derived(
    (options.invoicing_profiles || []).filter((row: WorkflowRecord) => row.client === job.client),
  );
  const people = $derived(options.users || []);
</script>

<details
  class="space-y-3 rounded border p-3"
  open={!job.invoicing_information || !job.contact || !job.rate_sheet || !allocations.length}
>
  <summary class="cursor-pointer font-semibold">Project and billing details</summary>
  <p class="text-sm">
    Correct missing project details with this proposal. These detail changes are saved now. The
    effective project value stays unchanged until approval.
  </p>
  <div class="grid gap-3 md:grid-cols-2">
    <p class="md:col-span-2">
      Client: <strong
        >{options.clients?.find((row: WorkflowRecord) => row.id === job.client)?.name ||
          job.client}</strong
      >
    </p>
    <label class="md:col-span-2"
      >Project / scope of work<textarea rows="3" bind:value={job.description}></textarea></label
    >
    <label
      >Project contact<select bind:value={job.contact}
        ><option value="">Select contact</option>{#each contacts as row}<option value={row.id}
            >{fullName(row)} — {row.email}</option
          >{/each}</select
      ></label
    >
    {#if job.client && !profiles.length}
      <div class="flex flex-col gap-1 md:col-span-2">
        <h3 class="text-sm font-semibold">Invoicing profile</h3>
        <InvoicingProfileEmptyState
          canCreate={canCreateInvoicingProfile($globalStore.claims)}
          canManageContacts={$globalStore.claims.includes("job")}
          hasContacts={contacts.length > 0}
          oncreate={onmanageclient ? () => onmanageclient?.("invoicing") : undefined}
          onaddcontact={onmanageclient ? () => onmanageclient?.("contact") : undefined}
        />
      </div>
    {:else}
      <label
        >Invoicing profile<select
          aria-label="Invoicing profile"
          bind:value={job.invoicing_information}
          ><option value="">Select invoicing profile</option>{#each profiles as row}<option
              value={row.id}
              >{row.name ||
                fullName(
                  contacts.find((contact: WorkflowRecord) => contact.id === row.contact),
                )}{row.invoicing_instructions ? ` — ${row.invoicing_instructions}` : ""}</option
            >{/each}</select
        ></label
      >
      <button
        type="button"
        class="text-sm text-blue-700 underline md:col-span-2"
        onclick={() => onmanageclient?.()}>Manage client contacts and invoicing</button
      >
    {/if}
    <label
      >Project manager<select bind:value={job.manager}
        ><option value="">Select manager</option>{#each people as row}<option value={row.id}
            >{fullName(row)}</option
          >{/each}</select
      ></label
    >
    <label
      >Alternate manager (optional)<select bind:value={job.alternate_manager}
        ><option value="">No alternate manager</option>{#each people as row}<option value={row.id}
            >{fullName(row)}</option
          >{/each}</select
      ></label
    >
    <label
      >Branch<select bind:value={job.branch}
        ><option value="">Select branch</option>{#each options.branches || [] as row}<option
            value={row.id}>{row.name}</option
          >{/each}</select
      ></label
    >
    <label
      >Project owner (optional)<select bind:value={job.job_owner}
        ><option value="">No owner</option>{#each options.clients || [] as row}<option
            value={row.id}>{row.name}</option
          >{/each}</select
      ></label
    >
    <label
      >Project status<select bind:value={job.status}
        ><option>Active</option><option>Closed</option><option>Cancelled</option></select
      ></label
    >
    <label
      >Fee basis<select bind:value={job.time_and_materials}
        ><option value={false}>Fixed fee</option><option value={true}>Time and materials</option
        ></select
      ></label
    >
    <label
      >Rate sheet<select
        bind:value={job.rate_sheet}
        disabled={!!originalRate && !$globalStore.claims.includes("rate_sheet_revise")}
        ><option value="">Select rate sheet</option
        >{#each (options.rate_sheets || []).filter((row: WorkflowRecord) => row.active || row.id === originalRate) as row}<option
            value={row.id}>{row.name} — Rev. {row.revision}</option
          >{/each}</select
      ></label
    >
    <label
      >First Nations agreement<select bind:value={job.fn_agreement}
        ><option value={true}>Yes</option><option value={false}>No</option></select
      ></label
    >
    <label>Project award date<input type="date" bind:value={job.project_award_date} /></label>
    <label
      >Project completion date<input type="date" bind:value={job.project_completion_date} /></label
    >
    <label>Client PO (optional)<input bind:value={job.client_po} /></label>
    <label
      >Client reference number (optional)<input bind:value={job.client_reference_number} /></label
    >
    <div class="md:col-span-2">
      <DSLocationPicker bind:value={job.location} errors={{}} fieldName="location" />
    </div>
  </div>
  <h4 class="font-semibold">Division allocations</h4>
  {#each allocations as allocation, index}<div class="flex items-end gap-2">
      <label class="grow"
        >Division<select bind:value={allocation.division}
          ><option value="">Select division</option
          >{#each (options.divisions || []).filter((row: WorkflowRecord) => row.active) as row}<option
              value={row.id}>{row.code} — {row.name}</option
            >{/each}</select
        ></label
      ><label>Hours<input type="number" min="0" step="0.25" bind:value={allocation.hours} /></label
      ><button
        type="button"
        class="rounded border p-2"
        onclick={() => (allocations = allocations.filter((_, i) => i !== index))}>Remove</button
      >
    </div>{/each}
  <button
    type="button"
    class="rounded border px-3 py-2"
    onclick={() => (allocations = [...allocations, { division: "", hours: 0 }])}
    >Add division</button
  >
</details>

<style>
  label {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    font-weight: 600;
    font-size: 0.9rem;
  }
  input,
  select,
  textarea {
    width: 100%;
    border: 1px solid #a3a3a3;
    padding: 0.5rem;
    border-radius: 0.2rem;
    background: white;
    font-weight: 400;
  }
</style>
