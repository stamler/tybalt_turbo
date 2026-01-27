<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { shortDate } from "$lib/utilities";
  import ObjectTable from "$lib/components/ObjectTable.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { invalidateAll } from "$app/navigation";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  // Local state for entries and rate sheet (for reactivity after updates)
  // Using local variables to avoid "state_referenced_locally" warning
  const initialEntries = data.entries;
  const initialRateSheet = data.rateSheet;
  const initialRoles = data.allRoles;
  let entries = $state(initialEntries);
  let rateSheet = $state(initialRateSheet);
  let allRoles = $state(initialRoles);

  // Form state for adding new entry
  let newEntry = $state({
    role: "",
    rate: 0,
    overtime_rate: 0,
  });
  let saveError = $state("");
  let saving = $state(false);

  // Compute missing roles
  const missingRoles = $derived(
    allRoles.filter((role) => !entries.some((e) => e.role === role.id))
  );
  const isComplete = $derived(missingRoles.length === 0);

  // Transform entries for ObjectTable (flatten role name)
  const tableData = $derived(
    entries.map((e) => ({
      Role: e.expand?.role?.name ?? "Unknown",
      Rate: e.rate,
      "Overtime Rate": e.overtime_rate,
    }))
  );

  // Toggle active status
  async function toggleActive() {
    if (!isComplete) return;

    try {
      const newActive = !rateSheet.active;
      await pb.collection("rate_sheets").update(rateSheet.id, { active: newActive });
      rateSheet = { ...rateSheet, active: newActive };
    } catch (error: any) {
      console.error("Failed to update active status:", error);
      alert(error.data?.data?.active?.message ?? "Failed to update active status");
    }
  }

  // Save new entry
  async function saveEntry(event: Event) {
    event.preventDefault();
    if (!newEntry.role) {
      saveError = "Please select a role";
      return;
    }

    saving = true;
    saveError = "";

    try {
      const created = await pb.collection("rate_sheet_entries").create({
        rate_sheet: rateSheet.id,
        role: newEntry.role,
        rate: newEntry.rate,
        overtime_rate: newEntry.overtime_rate,
      });

      // Fetch the created entry with expanded role
      const expandedEntry = await pb.collection("rate_sheet_entries").getOne(created.id, {
        expand: "role",
      });

      // Add to local entries
      entries = [...entries, expandedEntry];

      // Reset form
      newEntry = { role: "", rate: 0, overtime_rate: 0 };
    } catch (error: any) {
      console.error("Failed to create entry:", error);
      saveError = error.data?.message ?? "Failed to create entry";
    } finally {
      saving = false;
    }
  }
</script>

<div class="p-4">
  <!-- Header -->
  <div class="mb-6">
    <h1 class="flex items-baseline gap-2 text-2xl font-bold">
      {rateSheet.name}
      <span class="text-base font-normal text-neutral-500">rev. {rateSheet.revision}</span>
    </h1>
    <p class="text-neutral-600">
      Effective: {shortDate(rateSheet.effective_date, true)}
    </p>
  </div>

  <!-- Action Buttons -->
  <div class="mb-4 flex gap-2">
    {#if rateSheet.active}
      <DsActionButton action={`/rate-sheets/add?revise=${rateSheet.id}`}>
        Revise
      </DsActionButton>
    {/if}
    <DsActionButton action={`/rate-sheets/copy?from=${rateSheet.id}`}>
      New from Template
    </DsActionButton>
  </div>

  <!-- Active Toggle -->
  <div class="mb-6">
    <label
      class="flex items-center gap-2"
      class:opacity-50={!isComplete}
      class:cursor-not-allowed={!isComplete}
    >
      <input
        type="checkbox"
        checked={rateSheet.active}
        disabled={!isComplete}
        onchange={toggleActive}
        class="h-5 w-5"
      />
      <span class="font-medium">
        {rateSheet.active ? "Active" : "Inactive"}
      </span>
      {#if !isComplete}
        <span class="text-sm text-neutral-500">
          (Cannot activate - missing {missingRoles.length} role{missingRoles.length === 1 ? "" : "s"})
        </span>
      {/if}
    </label>
  </div>

  <!-- Entries Table -->
  <div class="mb-6">
    <h2 class="mb-2 text-lg font-semibold">Rate Entries</h2>
    {#if entries.length > 0}
      <ObjectTable {tableData} />
    {:else}
      <p class="text-neutral-500">No entries yet. Add entries below to complete this rate sheet.</p>
    {/if}
  </div>

  <!-- Add Entry Form (when incomplete) -->
  {#if !isComplete}
    <div class="rounded border border-neutral-300 bg-neutral-50 p-4">
      <h3 class="mb-3 font-semibold">Add Entry</h3>
      <form onsubmit={saveEntry} class="flex flex-wrap items-end gap-4">
        <div class="flex flex-col">
          <label for="role" class="mb-1 text-sm font-medium">Role</label>
          <select
            id="role"
            bind:value={newEntry.role}
            class="rounded border border-neutral-300 px-3 py-2"
          >
            <option value="">Select a role...</option>
            {#each missingRoles as role}
              <option value={role.id}>{role.name}</option>
            {/each}
          </select>
        </div>

        <div class="flex flex-col">
          <label for="rate" class="mb-1 text-sm font-medium">Rate</label>
          <input
            id="rate"
            type="number"
            min="1"
            step="0.01"
            bind:value={newEntry.rate}
            class="w-32 rounded border border-neutral-300 px-3 py-2"
          />
        </div>

        <div class="flex flex-col">
          <label for="overtime_rate" class="mb-1 text-sm font-medium">Overtime Rate</label>
          <input
            id="overtime_rate"
            type="number"
            min="1"
            step="0.01"
            bind:value={newEntry.overtime_rate}
            class="w-32 rounded border border-neutral-300 px-3 py-2"
          />
        </div>

        <DsActionButton type="submit" disabled={saving}>
          {saving ? "Saving..." : "Add Entry"}
        </DsActionButton>
      </form>

      {#if saveError}
        <p class="mt-2 text-red-600">{saveError}</p>
      {/if}
    </div>
  {/if}

  <!-- Back link -->
  <div class="mt-6">
    <a href="/rate-sheets/list" class="text-blue-600 hover:underline">
      &larr; Back to Rate Sheets
    </a>
  </div>
</div>
