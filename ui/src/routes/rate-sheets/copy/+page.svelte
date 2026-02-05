<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { goto } from "$app/navigation";
  import type { PageData } from "./$types";
  import { untrack } from "svelte";

  let { data }: { data: PageData } = $props();

  const sourceSheet = untrack(() => data.sourceSheet);
  const sourceEntries = untrack(() => data.sourceEntries);
  const isRevising = untrack(() => data.isRevising);
  const nextRevision = untrack(() => data.nextRevision);

  let errors = $state({} as Record<string, { message: string }>);
  let saveError = $state("");
  let saving = $state(false);
  const entryError = (index: number, field: string) =>
    errors[`entries.${index}.${field}`] as { message: string } | undefined;
  const entriesError = () => errors.entries as { message: string } | undefined;

  // Form state - name is locked when revising
  let name = $state(isRevising ? sourceSheet.name : `Copy of ${sourceSheet.name}`);
  let effective_date = $state("");

  // Staged entries - editable copies of source entries, sorted by role name
  let stagedEntries = $state(
    sourceEntries
      .map((e) => ({
        role: e.role,
        roleName: e.expand?.role?.name ?? "Unknown",
        rate: e.rate,
        overtime_rate: e.overtime_rate,
      }))
      .sort((a, b) => a.roleName.localeCompare(b.roleName)),
  );

  // Overtime multiplier for bulk calculation
  let overtimeMultiplier = $state(1.3);

  // Apply overtime multiplier to all staged entries
  function applyOvertimeMultiplier() {
    stagedEntries = stagedEntries.map((entry) => ({
      ...entry,
      overtime_rate: Math.round(entry.rate * overtimeMultiplier * 100) / 100,
    }));
  }

  async function save(event: Event) {
    event.preventDefault();
    saveError = "";
    errors = {};

    // Validate
    if (!name.trim()) {
      errors = { name: { message: "Name is required" } };
      return;
    }
    if (!effective_date) {
      errors = { effective_date: { message: "Effective date is required" } };
      return;
    }

    saving = true;

    try {
      // Create rate sheet and all entries atomically via custom endpoint
      const response = await pb.send("/api/rate_sheets", {
        method: "POST",
        body: JSON.stringify({
          name: name.trim(),
          effective_date,
          revision: isRevising ? nextRevision : 0,
          entries: stagedEntries.map((entry) => ({
            role: entry.role,
            rate: entry.rate,
            overtime_rate: entry.overtime_rate,
          })),
        }),
      });

      // Success - redirect to details page
      saving = false;
      goto(`/rate-sheets/${response.id}/details`);
    } catch (error: any) {
      console.error("Failed to save:", error);
      saveError = error.data?.message ?? error.message ?? "Failed to create rate sheet";
      if (error.data?.data) {
        errors = error.data.data;
      }
      saving = false;
    }
  }
</script>

<div class="p-4">
  <h1 class="mb-4 text-xl font-bold">
    {isRevising ? "Revise Rate Sheet" : "New Rate Sheet"}
  </h1>
  <p class="mb-6 text-neutral-600">
    {isRevising
      ? `Creating revision ${nextRevision} of "${sourceSheet.name}"`
      : `Creating a new rate sheet based on "${sourceSheet.name}"`}
  </p>

  <form onsubmit={save} class="flex flex-col gap-4">
    <!-- Name field -->
    {#if isRevising}
      <div class="flex w-full flex-col gap-2">
        <span class="flex w-full items-center gap-2">
          <label for="name">Name</label>
          <input
            class="flex-1 cursor-not-allowed rounded-sm border border-neutral-300 bg-neutral-100 px-1 opacity-60"
            type="text"
            id="name"
            value={name}
            disabled
          />
          <span class="text-sm text-neutral-500 italic">revising</span>
        </span>
      </div>
    {:else}
      <DsTextInput bind:value={name} {errors} fieldName="name" uiName="Name" />
    {/if}

    <!-- Effective Date field -->
    <div
      class="flex w-full flex-col gap-2 {errors.effective_date !== undefined ? 'bg-red-200' : ''}"
    >
      <span class="flex w-full gap-2">
        <label for="effective_date">Effective Date</label>
        <input
          class="flex-1 rounded-sm border border-neutral-300 px-1"
          type="date"
          id="effective_date"
          name="effective_date"
          bind:value={effective_date}
        />
      </span>
      {#if errors.effective_date !== undefined}
        <span class="text-red-600">{errors.effective_date.message}</span>
      {/if}
    </div>

    <!-- Entries Table -->
    <div class="mt-4">
      <h2 class="mb-2 text-lg font-semibold">Rate Entries</h2>
      <p class="mb-3 text-sm text-neutral-600">
        Edit the rates below. All entries will be created when you save.
      </p>

      <div class="overflow-x-auto">
        <table class="w-full border-collapse">
          <thead>
            <!-- Multiplier row above headers -->
            <tr>
              <th></th>
              <th></th>
              <th class="px-3 py-2 text-left">
                <div class="flex items-center gap-1">
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
            </tr>
            <tr class="border-b border-neutral-300">
              <th class="px-3 py-2 text-left">Role</th>
              <th class="px-3 py-2 text-left">Rate</th>
              <th class="px-3 py-2 text-left">Overtime Rate</th>
            </tr>
          </thead>
          <tbody>
            {#each stagedEntries as entry, i}
              <tr class="border-b border-neutral-200">
                <td class="px-3 py-2 text-neutral-700">{entry.roleName}</td>
                <td class="px-3 py-2">
                  <input
                    type="number"
                    min="1"
                    step="1"
                    bind:value={stagedEntries[i].rate}
                    class={`w-28 rounded-sm border px-2 py-1 ${
                      entryError(i, "rate") ? "border-red-500 bg-red-50" : "border-neutral-300"
                    }`}
                  />
                  {#if entryError(i, "rate")}
                    <div class="mt-1 text-xs text-red-600">{entryError(i, "rate")?.message}</div>
                  {/if}
                </td>
                <td class="px-3 py-2">
                  <input
                    type="number"
                    min="1"
                    step="0.01"
                    bind:value={stagedEntries[i].overtime_rate}
                    class={`w-28 rounded-sm border px-2 py-1 ${
                      entryError(i, "overtime_rate")
                        ? "border-red-500 bg-red-50"
                        : "border-neutral-300"
                    }`}
                  />
                  {#if entryError(i, "overtime_rate")}
                    <div class="mt-1 text-xs text-red-600">
                      {entryError(i, "overtime_rate")?.message}
                    </div>
                  {/if}
                </td>
              </tr>
              {#if entryError(i, "role")}
                <tr class="border-b border-neutral-200">
                  <td class="px-3 pb-2 text-xs text-red-600" colspan="3">
                    {entryError(i, "role")?.message}
                  </td>
                </tr>
              {/if}
            {/each}
          </tbody>
        </table>
      </div>

      {#if stagedEntries.length === 0}
        <p class="mt-2 text-neutral-500">
          The source rate sheet has no entries. The new sheet will be created empty.
        </p>
      {/if}
      {#if entriesError()}
        <p class="mt-2 text-red-600">{entriesError()?.message}</p>
      {/if}
    </div>

    <!-- Save/Cancel buttons -->
    <div class="mt-4 flex items-center gap-2">
      <DsActionButton type="submit" disabled={saving}>
        {saving ? "Saving..." : "Save"}
      </DsActionButton>
      <DsActionButton action="/rate-sheets/list">Cancel</DsActionButton>

      {#if saveError}
        <span class="text-red-600">{saveError}</span>
      {/if}
    </div>
  </form>

  <!-- Back link -->
  <div class="mt-6">
    <a href="/rate-sheets/list" class="text-blue-600 hover:underline">
      &larr; Back to Rate Sheets
    </a>
  </div>
</div>
