<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { shortDate } from "$lib/utilities";
  import { globalStore } from "$lib/stores/global";
  import ObjectTable from "$lib/components/ObjectTable.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsToggle from "$lib/components/DSToggle.svelte";
  import Icon from "@iconify/svelte";
  import type { PageData } from "./$types";
  import { untrack } from "svelte";

  let { data }: { data: PageData } = $props();

  // Local state for entries and rate sheet (for reactivity after updates)
  let entries = $state(untrack(() => data.entries));
  let rateSheet = $state(untrack(() => data.rateSheet));
  let allRoles = $state(untrack(() => data.allRoles));

  // Admin check for editing capability
  const isAdmin = $derived($globalStore.claims.includes("admin"));

  // Download entries as CSV sorted by rate descending
  function downloadCsv() {
    const sorted = [...entries].sort((a, b) => b.rate - a.rate);
    const rows = [
      ["Role", "Rate", "Overtime Rate"],
      ...sorted.map((e) => [
        e.expand?.role?.name ?? "Unknown",
        e.rate.toFixed(2),
        e.overtime_rate.toFixed(2),
      ]),
    ];
    const csv = rows.map((r) => r.map((v) => `"${v}"`).join(",")).join("\n");
    const blob = new Blob([csv], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${rateSheet.name} rev${rateSheet.revision}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }

  // Track edited entries by id -> { rate, overtime_rate }
  let editedEntries = $state<Record<string, { rate: number; overtime_rate: number }>>({});
  let savingEntryId = $state<string | null>(null);
  let entryErrors = $state<Record<string, string>>({});

  // Overtime multiplier for bulk calculation
  let overtimeMultiplier = $state(1.3);

  // Column sorting state
  type SortColumn = "role" | "rate" | "overtime_rate";
  let sortColumn = $state<SortColumn>("role");
  let sortOrder = $state<"asc" | "desc">("asc");

  function toggleSort(column: SortColumn) {
    if (sortColumn === column) {
      sortOrder = sortOrder === "asc" ? "desc" : "asc";
    } else {
      sortColumn = column;
      sortOrder = "asc";
    }
  }

  // Form state for adding new entry
  let newEntry = $state({
    role: "",
    rate: 0,
    overtime_rate: 0,
  });
  let saveError = $state("");
  let saving = $state(false);
  let toggleError = $state("");

  // Check if an entry has been modified (values differ from original)
  function isEntryModified(entry: any): boolean {
    const edits = editedEntries[entry.id];
    if (!edits) return false;
    return edits.rate !== entry.rate || edits.overtime_rate !== entry.overtime_rate;
  }

  // Get the current value for an entry field (edited or original)
  function getEntryValue(entry: any, field: "rate" | "overtime_rate"): number {
    if (editedEntries[entry.id]) {
      return editedEntries[entry.id][field];
    }
    return entry[field];
  }

  // Update an entry field in the edited state
  function updateEntryField(entry: any, field: "rate" | "overtime_rate", value: number) {
    if (!editedEntries[entry.id]) {
      // Initialize with current values
      editedEntries[entry.id] = {
        rate: entry.rate,
        overtime_rate: entry.overtime_rate,
      };
    }
    editedEntries[entry.id][field] = value;
    // Trigger reactivity
    editedEntries = { ...editedEntries };
  }

  // Save a single edited entry
  async function saveEditedEntry(entryId: string) {
    const edits = editedEntries[entryId];
    if (!edits) return;

    savingEntryId = entryId;
    delete entryErrors[entryId];
    entryErrors = { ...entryErrors };

    try {
      const response = await pb.send(`/api/rate_sheet_entries/${entryId}`, {
        method: "PUT",
        body: JSON.stringify({
          rate: edits.rate,
          overtime_rate: edits.overtime_rate,
        }),
      });

      // Update local entries with new values
      entries = entries.map((e) =>
        e.id === entryId ? { ...e, rate: response.rate, overtime_rate: response.overtime_rate } : e,
      );

      // Remove from edited state
      delete editedEntries[entryId];
      editedEntries = { ...editedEntries };
    } catch (error: any) {
      console.error("Failed to update entry:", error);
      entryErrors[entryId] = error.data?.message ?? "Failed to update";
      entryErrors = { ...entryErrors };
    } finally {
      savingEntryId = null;
    }
  }

  // Cancel edits for an entry
  function cancelEntryEdit(entryId: string) {
    delete editedEntries[entryId];
    editedEntries = { ...editedEntries };
    delete entryErrors[entryId];
    entryErrors = { ...entryErrors };
  }

  // Apply overtime multiplier to all entries
  function applyOvertimeMultiplier() {
    const newEdits: Record<string, { rate: number; overtime_rate: number }> = { ...editedEntries };

    for (const entry of entries) {
      const currentRate = editedEntries[entry.id]?.rate ?? entry.rate;
      const calculatedOvertime = Math.round(currentRate * overtimeMultiplier * 100) / 100;

      newEdits[entry.id] = {
        rate: currentRate,
        overtime_rate: calculatedOvertime,
      };
    }

    editedEntries = newEdits;
  }

  // Sort entries by selected column
  const sortedEntries = $derived(
    [...entries].sort((a, b) => {
      let comparison = 0;
      if (sortColumn === "role") {
        const nameA = a.expand?.role?.name ?? "";
        const nameB = b.expand?.role?.name ?? "";
        comparison = nameA.localeCompare(nameB);
      } else if (sortColumn === "rate") {
        comparison = a.rate - b.rate;
      } else if (sortColumn === "overtime_rate") {
        comparison = a.overtime_rate - b.overtime_rate;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    }),
  );

  // Compute missing roles
  const missingRoles = $derived(
    allRoles.filter((role) => !entries.some((e) => e.role === role.id)),
  );
  const isComplete = $derived(missingRoles.length === 0);

  // Transform entries for ObjectTable (flatten role name)
  const tableData = $derived(
    entries.map((e) => ({
      Role: e.expand?.role?.name ?? "Unknown",
      Rate: e.rate,
      "Overtime Rate": e.overtime_rate,
    })),
  );

  // Toggle value for DSToggle (string-based)
  let activeToggleValue = $state(untrack(() => (rateSheet.active ? "active" : "inactive")));
  let isToggling = $state(false);

  // React to toggle value changes
  $effect(() => {
    const newValue = activeToggleValue;
    const currentActive = rateSheet.active;
    const shouldBeActive = newValue === "active";

    // Only call API if value actually changed and we're not already toggling
    if (shouldBeActive !== currentActive && !isToggling) {
      isToggling = true;
      toggleError = "";

      const endpoint = shouldBeActive
        ? `/api/rate_sheets/${rateSheet.id}/activate`
        : `/api/rate_sheets/${rateSheet.id}/deactivate`;

      pb.send(endpoint, { method: "POST" })
        .then((response) => {
          rateSheet = { ...rateSheet, active: response.active };
        })
        .catch((error: any) => {
          console.error("Failed to update active status:", error);
          toggleError = error.data?.message ?? "Failed to update status";
          // Revert toggle value on error
          activeToggleValue = currentActive ? "active" : "inactive";
        })
        .finally(() => {
          isToggling = false;
        });
    }
  });

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
    <div class="flex items-baseline gap-2">
      <h1 class="text-2xl font-bold">
        {rateSheet.name}
        <span class="text-base font-normal text-neutral-500">rev. {rateSheet.revision}</span>
      </h1>
      <DsActionButton
        action={() => window.open(`/rate-sheets/${rateSheet.id}/print`, '_blank')}
        icon="mdi:printer"
        title="Print"
        color="gray"
      />
      <DsActionButton
        action={downloadCsv}
        icon="mdi:download"
        title="Download CSV"
        color="gray"
      />
    </div>
    <p class="text-neutral-600">
      Effective: {shortDate(rateSheet.effective_date, true)}
    </p>
  </div>

  <!-- Action Buttons -->
  <div class="mb-4 flex gap-2">
    {#if rateSheet.active}
      <DsActionButton action={`/rate-sheets/copy?revise=${rateSheet.id}`}>Revise</DsActionButton>
    {/if}
    <DsActionButton action={`/rate-sheets/copy?from=${rateSheet.id}`}>
      Use as Template
    </DsActionButton>
  </div>

  <!-- Active Toggle -->
  <div class="mb-6 flex items-center gap-4">
    <div
      class:opacity-50={!isComplete || isToggling}
      class:pointer-events-none={!isComplete || isToggling}
    >
      <DsToggle
        bind:value={activeToggleValue}
        options={[
          { id: "inactive", label: "Inactive" },
          { id: "active", label: "Active" },
        ]}
      />
    </div>
    {#if !isComplete}
      <span class="text-sm text-neutral-500">
        (Cannot activate - missing {missingRoles.length} role{missingRoles.length === 1 ? "" : "s"})
      </span>
    {/if}
    {#if toggleError}
      <span class="text-sm text-red-600">{toggleError}</span>
    {/if}
  </div>

  <!-- Entries Table -->
  <div class="mb-6">
    <h2 class="mb-2 text-lg font-semibold">Rate Entries</h2>
    {#if entries.length > 0}
      {#if isAdmin}
        <!-- Editable table for admins -->
        <div class="overflow-x-auto">
          <table class="w-full">
            <thead>
              <!-- Multiplier row above headers -->
              <tr>
                <th></th>
                <th></th>
                <th class="pr-4 pb-2 text-right">
                  <div class="flex items-center justify-end gap-1">
                    <input
                      type="number"
                      min="0.01"
                      step="0.01"
                      bind:value={overtimeMultiplier}
                      class="w-16 rounded-sm border border-neutral-300 px-2 py-1 text-right text-sm"
                      title="Overtime multiplier"
                    />
                    <button
                      type="button"
                      class="rounded-sm bg-neutral-200 px-2 py-1 text-sm text-neutral-700 hover:bg-neutral-300"
                      onclick={applyOvertimeMultiplier}
                      title="Apply multiplier to all overtime rates"
                    >
                      &times;
                    </button>
                  </div>
                </th>
                <th></th>
              </tr>
              <tr class="border-b border-neutral-300">
                <th class="pr-4 pb-2 text-left">
                  <button class="hover:underline" onclick={() => toggleSort("role")}>
                    Role
                    {#if sortColumn === "role"}
                      <Icon
                        icon={sortOrder === "asc" ? "mdi:sort-ascending" : "mdi:sort-descending"}
                        class="inline w-4"
                      />
                    {/if}
                  </button>
                </th>
                <th class="pr-4 pb-2 text-right">
                  <button class="hover:underline" onclick={() => toggleSort("rate")}>
                    Rate
                    {#if sortColumn === "rate"}
                      <Icon
                        icon={sortOrder === "asc" ? "mdi:sort-ascending" : "mdi:sort-descending"}
                        class="inline w-4"
                      />
                    {/if}
                  </button>
                </th>
                <th class="pr-4 pb-2 text-right">
                  <button class="hover:underline" onclick={() => toggleSort("overtime_rate")}>
                    Overtime Rate
                    {#if sortColumn === "overtime_rate"}
                      <Icon
                        icon={sortOrder === "asc" ? "mdi:sort-ascending" : "mdi:sort-descending"}
                        class="inline w-4"
                      />
                    {/if}
                  </button>
                </th>
                <th class="pb-2 text-left">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each sortedEntries as entry (entry.id)}
                <tr class="border-b border-neutral-200">
                  <td class="py-2 pr-4">{entry.expand?.role?.name ?? "Unknown"}</td>
                  <td class="py-2 pr-4 text-right">
                    <input
                      type="number"
                      min="1"
                      step="1"
                      class="w-24 rounded-sm border border-neutral-300 px-2 py-1 text-right"
                      value={getEntryValue(entry, "rate")}
                      oninput={(e) =>
                        updateEntryField(entry, "rate", parseFloat(e.currentTarget.value) || 0)}
                    />
                  </td>
                  <td class="py-2 pr-4 text-right">
                    <input
                      type="number"
                      min="1"
                      step="0.01"
                      class="w-24 rounded-sm border border-neutral-300 px-2 py-1 text-right"
                      value={getEntryValue(entry, "overtime_rate")}
                      oninput={(e) =>
                        updateEntryField(
                          entry,
                          "overtime_rate",
                          parseFloat(e.currentTarget.value) || 0,
                        )}
                    />
                  </td>
                  <td class="py-2">
                    {#if isEntryModified(entry)}
                      <div class="flex items-center gap-2">
                        <button
                          type="button"
                          class="rounded-sm bg-blue-500 px-2 py-1 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
                          disabled={savingEntryId === entry.id}
                          onclick={() => saveEditedEntry(entry.id)}
                        >
                          {savingEntryId === entry.id ? "Saving..." : "Save"}
                        </button>
                        <button
                          type="button"
                          class="rounded-sm bg-neutral-200 px-2 py-1 text-sm text-neutral-700 hover:bg-neutral-300"
                          disabled={savingEntryId === entry.id}
                          onclick={() => cancelEntryEdit(entry.id)}
                        >
                          Cancel
                        </button>
                        {#if entryErrors[entry.id]}
                          <span class="text-sm text-red-600">{entryErrors[entry.id]}</span>
                        {/if}
                      </div>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {:else}
        <!-- Read-only table for non-admins -->
        <ObjectTable {tableData} />
      {/if}
    {:else}
      <p class="text-neutral-500">No entries yet. Add entries below to complete this rate sheet.</p>
    {/if}
  </div>

  <!-- Add Entry Form (when incomplete) -->
  {#if !isComplete}
    <div class="rounded-sm border border-neutral-300 bg-neutral-50 p-4">
      <h3 class="mb-3 font-semibold">Add Entry</h3>
      <form onsubmit={saveEntry} class="flex flex-wrap items-end gap-4">
        <div class="flex flex-col">
          <label for="role" class="mb-1 text-sm font-medium">Role</label>
          <select
            id="role"
            bind:value={newEntry.role}
            class="rounded-sm border border-neutral-300 px-3 py-2"
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
            class="w-32 rounded-sm border border-neutral-300 px-3 py-2"
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
            class="w-32 rounded-sm border border-neutral-300 px-3 py-2"
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
