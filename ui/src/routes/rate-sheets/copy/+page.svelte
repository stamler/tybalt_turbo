<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { goto } from "$app/navigation";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  const sourceSheet = data.sourceSheet;
  const sourceEntries = data.sourceEntries;

  let errors = $state({} as Record<string, { message: string }>);
  let saveError = $state("");
  let saving = $state(false);

  // Form state
  let name = $state(`Copy of ${sourceSheet.name}`);
  let effective_date = $state("");

  // Staged entries - editable copies of source entries
  let stagedEntries = $state(
    sourceEntries.map((e) => ({
      role: e.role,
      roleName: e.expand?.role?.name ?? "Unknown",
      rate: e.rate,
      overtime_rate: e.overtime_rate,
    }))
  );

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
    let createdSheetId: string | null = null;

    try {
      // Step 1: Create the rate sheet
      const createdSheet = await pb.collection("rate_sheets").create({
        name: name.trim(),
        effective_date,
        revision: 0,
        active: false,
      });
      createdSheetId = createdSheet.id;

      // Step 2: Create all entries
      const entryPromises = stagedEntries.map((entry) =>
        pb.collection("rate_sheet_entries").create({
          rate_sheet: createdSheetId,
          role: entry.role,
          rate: entry.rate,
          overtime_rate: entry.overtime_rate,
        })
      );

      await Promise.all(entryPromises);

      // 100% success - redirect to details page
      goto(`/rate-sheets/${createdSheetId}/details`);
    } catch (error: any) {
      console.error("Failed to save:", error);

      // Check if rate sheet was created but entries failed
      if (createdSheetId) {
        saveError = `Rate sheet created but some entries failed. Visit the details page to add missing entries.`;
        // Provide link but don't auto-redirect
      } else {
        saveError = error.data?.message ?? error.message ?? "Failed to create rate sheet";
        if (error.data?.data) {
          errors = error.data.data;
        }
      }
    } finally {
      saving = false;
    }
  }
</script>

<div class="p-4">
  <h1 class="mb-4 text-xl font-bold">New Rate Sheet from Template</h1>
  <p class="mb-6 text-neutral-600">
    Creating a new rate sheet based on "{sourceSheet.name}"
  </p>

  <form onsubmit={save} class="flex flex-col gap-4">
    <!-- Name field -->
    <DsTextInput bind:value={name} {errors} fieldName="name" uiName="Name" />

    <!-- Effective Date field -->
    <div class="flex w-full flex-col gap-2 {errors.effective_date !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <label for="effective_date">Effective Date</label>
        <input
          class="flex-1 rounded border border-neutral-300 px-1"
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
                    min="0"
                    step="0.01"
                    bind:value={stagedEntries[i].rate}
                    class="w-28 rounded border border-neutral-300 px-2 py-1"
                  />
                </td>
                <td class="px-3 py-2">
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    bind:value={stagedEntries[i].overtime_rate}
                    class="w-28 rounded border border-neutral-300 px-2 py-1"
                  />
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      {#if stagedEntries.length === 0}
        <p class="mt-2 text-neutral-500">
          The source rate sheet has no entries. The new sheet will be created empty.
        </p>
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
