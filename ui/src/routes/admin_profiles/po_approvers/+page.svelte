<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import { divisions as divisionsStore } from "$lib/stores/divisions";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import Icon from "@iconify/svelte";
  import type {
    AdminProfilesAugmentedResponse,
    DivisionsResponse,
  } from "$lib/pocketbase-types";
  import type { SearchResult } from "minisearch";
  import type { PageData } from "./$types";
  import { onMount, untrack } from "svelte";

  let { data }: { data: PageData } = $props();

  let items = $state(untrack(() => data.items as AdminProfilesAugmentedResponse[]));

  const isAdmin = $derived($globalStore.claims.includes("admin"));

  // Divisions store
  const divisions = $derived.by(() => $divisionsStore.items as DivisionsResponse[]);
  const divisionsIndex = $derived.by(() => $divisionsStore.index);

  onMount(async () => {
    await divisionsStore.init();
  });

  // Search
  let searchTerm = $state("");

  // Column sorting
  type AmountField =
    | "max_amount"
    | "project_max"
    | "sponsorship_max"
    | "staff_and_social_max"
    | "media_and_event_max"
    | "computer_max";
  type SortColumn = "name" | AmountField;
  let sortColumn = $state<SortColumn>("name");
  let sortOrder = $state<"asc" | "desc">("asc");

  function toggleSort(column: SortColumn) {
    if (sortColumn === column) {
      sortOrder = sortOrder === "asc" ? "desc" : "asc";
    } else {
      sortColumn = column;
      sortOrder = "asc";
    }
  }

  const itemFieldMap: Record<AmountField, string> = {
    max_amount: "po_approver_max_amount",
    project_max: "po_approver_project_max",
    sponsorship_max: "po_approver_sponsorship_max",
    staff_and_social_max: "po_approver_staff_and_social_max",
    media_and_event_max: "po_approver_media_and_event_max",
    computer_max: "po_approver_computer_max",
  };

  const filteredItems = $derived.by(() => {
    const term = searchTerm.toLowerCase().trim();
    if (!term) return items;
    return items.filter(
      (i) =>
        i.given_name.toLowerCase().includes(term) || i.surname.toLowerCase().includes(term),
    );
  });

  const sortedItems = $derived(
    [...filteredItems].sort((a, b) => {
      let comparison = 0;
      if (sortColumn === "name") {
        comparison =
          (a.surname + a.given_name).localeCompare(b.surname + b.given_name);
      } else {
        const aVal = normalizeNumber((a as Record<string, unknown>)[itemFieldMap[sortColumn]]);
        const bVal = normalizeNumber((b as Record<string, unknown>)[itemFieldMap[sortColumn]]);
        comparison = aVal - bVal;
      }
      return sortOrder === "asc" ? comparison : -comparison;
    }),
  );

  // Edited entries keyed by po_approver_props_id
  type EditedApproverEntry = {
    max_amount: number;
    project_max: number;
    sponsorship_max: number;
    staff_and_social_max: number;
    media_and_event_max: number;
    computer_max: number;
    divisions: string[];
  };

  let editedEntries = $state<Record<string, EditedApproverEntry>>({});
  let savingEntryId = $state<string | null>(null);
  let deletingEntryId = $state<string | null>(null);
  let entryErrors = $state<Record<string, string>>({});
  let divisionEditRowId = $state<string | null>(null);
  let divisionSearchValue = $state("");
  let showDeleteConfirm = $state(false);
  let deleteConfirmError = $state<string | null>(null);
  let pendingDelete = $state<{
    itemId: string;
    propsId: string;
    userClaimId: string;
    fullName: string;
  } | null>(null);

  // Helpers
  function normalizeNumber(value: unknown): number {
    if (typeof value === "number") {
      return Number.isFinite(value) ? value : 0;
    }
    if (typeof value === "string") {
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : 0;
    }
    return 0;
  }

  function normalizeDivisions(value: unknown): string[] {
    if (Array.isArray(value)) {
      return value.filter((id): id is string => typeof id === "string");
    }
    if (typeof value === "string" && value.trim().startsWith("[")) {
      try {
        const parsed = JSON.parse(value);
        if (Array.isArray(parsed)) {
          return parsed.filter((id): id is string => typeof id === "string");
        }
      } catch {
        // noop
      }
    }
    return [];
  }

  function divisionCode(divisionId: string): string {
    const division = divisions.find((d) => d.id === divisionId);
    if (!division) return divisionId;
    return division.code || divisionId;
  }

  function divisionName(divisionId: string): string {
    const division = divisions.find((d) => d.id === divisionId);
    if (!division) return "";
    return division.name || "";
  }

  function divisionDisplay(
    division: SearchResult | { id: string; code?: string | null; name?: string },
  ): string {
    const code = "code" in division ? division.code?.trim() : undefined;
    const name = "name" in division ? (division.name?.trim() ?? division.id) : division.id;
    return code && code.length > 0 ? `${code} — ${name}` : name;
  }

  // Initialize edited state for a row if not already present
  function ensureEdited(item: AdminProfilesAugmentedResponse): EditedApproverEntry {
    const propsId = item.po_approver_props_id!;
    if (!editedEntries[propsId]) {
      editedEntries[propsId] = {
        max_amount: normalizeNumber(item.po_approver_max_amount),
        project_max: normalizeNumber(item.po_approver_project_max),
        sponsorship_max: normalizeNumber(item.po_approver_sponsorship_max),
        staff_and_social_max: normalizeNumber(item.po_approver_staff_and_social_max),
        media_and_event_max: normalizeNumber(item.po_approver_media_and_event_max),
        computer_max: normalizeNumber(item.po_approver_computer_max),
        divisions: normalizeDivisions(item.po_approver_divisions),
      };
    }
    return editedEntries[propsId];
  }

  // Get current value (edited or original)
  function getAmount(
    item: AdminProfilesAugmentedResponse,
    field: keyof EditedApproverEntry,
  ): number {
    const propsId = item.po_approver_props_id!;
    if (editedEntries[propsId]) {
      return editedEntries[propsId][field] as number;
    }
    const fieldMap: Record<string, string> = {
      max_amount: "po_approver_max_amount",
      project_max: "po_approver_project_max",
      sponsorship_max: "po_approver_sponsorship_max",
      staff_and_social_max: "po_approver_staff_and_social_max",
      media_and_event_max: "po_approver_media_and_event_max",
      computer_max: "po_approver_computer_max",
    };
    return normalizeNumber((item as Record<string, unknown>)[fieldMap[field as string]]);
  }

  function getDivisions(item: AdminProfilesAugmentedResponse): string[] {
    const propsId = item.po_approver_props_id!;
    if (editedEntries[propsId]) {
      return editedEntries[propsId].divisions;
    }
    return normalizeDivisions(item.po_approver_divisions);
  }

  // Update amount field
  function updateAmount(
    item: AdminProfilesAugmentedResponse,
    field: keyof EditedApproverEntry,
    value: number,
  ) {
    ensureEdited(item);
    const propsId = item.po_approver_props_id!;
    (editedEntries[propsId][field] as number) = value;
    editedEntries = { ...editedEntries };
  }

  // Check if a row has been modified
  function isEntryModified(item: AdminProfilesAugmentedResponse): boolean {
    const propsId = item.po_approver_props_id!;
    const edits = editedEntries[propsId];
    if (!edits) return false;

    const amountFields = [
      ["max_amount", "po_approver_max_amount"],
      ["project_max", "po_approver_project_max"],
      ["sponsorship_max", "po_approver_sponsorship_max"],
      ["staff_and_social_max", "po_approver_staff_and_social_max"],
      ["media_and_event_max", "po_approver_media_and_event_max"],
      ["computer_max", "po_approver_computer_max"],
    ] as const;

    for (const [editField, itemField] of amountFields) {
      if (edits[editField] !== normalizeNumber((item as Record<string, unknown>)[itemField])) {
        return true;
      }
    }

    const originalDivs = normalizeDivisions(item.po_approver_divisions);
    if (edits.divisions.length !== originalDivs.length) return true;
    if (edits.divisions.some((d, i) => d !== originalDivs[i])) return true;

    return false;
  }

  // Remove a division from a row
  function removeDivision(item: AdminProfilesAugmentedResponse, divisionId: string) {
    ensureEdited(item);
    const propsId = item.po_approver_props_id!;
    editedEntries[propsId].divisions = editedEntries[propsId].divisions.filter(
      (id) => id !== divisionId,
    );
    editedEntries = { ...editedEntries };
  }

  // Add a division to a row
  function addDivision(item: AdminProfilesAugmentedResponse, id: string | number) {
    const divisionId = id.toString();
    ensureEdited(item);
    const propsId = item.po_approver_props_id!;
    if (editedEntries[propsId].divisions.includes(divisionId)) {
      divisionSearchValue = "";
      return;
    }
    editedEntries[propsId].divisions = [...editedEntries[propsId].divisions, divisionId];
    editedEntries = { ...editedEntries };
    divisionSearchValue = "";
  }

  // Save a single row
  async function saveEntry(item: AdminProfilesAugmentedResponse) {
    const propsId = item.po_approver_props_id!;
    const edits = editedEntries[propsId];
    if (!edits) return;

    savingEntryId = propsId;
    delete entryErrors[propsId];
    entryErrors = { ...entryErrors };

    try {
      const response = await pb.collection("po_approver_props").update(propsId, {
        max_amount: edits.max_amount,
        project_max: edits.project_max,
        sponsorship_max: edits.sponsorship_max,
        staff_and_social_max: edits.staff_and_social_max,
        media_and_event_max: edits.media_and_event_max,
        computer_max: edits.computer_max,
        divisions: edits.divisions,
      });

      // Update local items with saved values
      items = items.map((i) => {
        if (i.po_approver_props_id === propsId) {
          return {
            ...i,
            po_approver_max_amount: response.max_amount,
            po_approver_project_max: response.project_max,
            po_approver_sponsorship_max: response.sponsorship_max,
            po_approver_staff_and_social_max: response.staff_and_social_max,
            po_approver_media_and_event_max: response.media_and_event_max,
            po_approver_computer_max: response.computer_max,
            po_approver_divisions: response.divisions,
          };
        }
        return i;
      });

      // Clear edited state
      delete editedEntries[propsId];
      editedEntries = { ...editedEntries };
      if (divisionEditRowId === propsId) {
        divisionEditRowId = null;
      }
    } catch (error: unknown) {
      console.error("Failed to update po_approver_props:", error);
      const msg =
        error && typeof error === "object" && "data" in error
          ? ((error as { data?: { message?: string } }).data?.message ?? "Failed to update")
          : "Failed to update";
      entryErrors[propsId] = msg;
      entryErrors = { ...entryErrors };
    } finally {
      savingEntryId = null;
    }
  }

  // Cancel edits for a row
  function cancelEntry(item: AdminProfilesAugmentedResponse) {
    const propsId = item.po_approver_props_id!;
    delete editedEntries[propsId];
    editedEntries = { ...editedEntries };
    delete entryErrors[propsId];
    entryErrors = { ...entryErrors };
    if (divisionEditRowId === propsId) {
      divisionEditRowId = null;
    }
  }

  function clearRowState(propsId: string) {
    delete editedEntries[propsId];
    editedEntries = { ...editedEntries };
    delete entryErrors[propsId];
    entryErrors = { ...entryErrors };
    if (divisionEditRowId === propsId) {
      divisionEditRowId = null;
    }
  }

  function closeDeleteConfirm() {
    if (deletingEntryId !== null) return;
    showDeleteConfirm = false;
    deleteConfirmError = null;
    pendingDelete = null;
  }

  function openDeleteConfirm(item: AdminProfilesAugmentedResponse) {
    const propsId = item.po_approver_props_id!;
    const userClaimId = item.po_approver_user_claim_id;
    if (typeof userClaimId !== "string" || userClaimId.trim() === "") {
      entryErrors[propsId] = "No user claim is linked to this approver record.";
      entryErrors = { ...entryErrors };
      return;
    }

    delete entryErrors[propsId];
    entryErrors = { ...entryErrors };
    pendingDelete = {
      itemId: item.id,
      propsId,
      userClaimId,
      fullName: `${item.given_name} ${item.surname}`.trim(),
    };
    deleteConfirmError = null;
    showDeleteConfirm = true;
  }

  async function deleteEntry() {
    if (!pendingDelete) return;
    const { propsId, userClaimId, itemId } = pendingDelete;
    deletingEntryId = propsId;
    delete entryErrors[propsId];
    entryErrors = { ...entryErrors };
    deleteConfirmError = null;

    try {
      await pb.collection("user_claims").delete(userClaimId);

      items = items.filter((i) => i.id !== itemId);
      clearRowState(propsId);
      showDeleteConfirm = false;
      pendingDelete = null;
    } catch (error: unknown) {
      console.error("Failed to delete po_approver user_claim:", error);
      const msg =
        error && typeof error === "object" && "data" in error
          ? ((error as { data?: { message?: string } }).data?.message ?? "Failed to delete")
          : "Failed to delete";
      entryErrors[propsId] = msg;
      entryErrors = { ...entryErrors };
      deleteConfirmError = msg;
    } finally {
      deletingEntryId = null;
    }
  }

  const amountColumns = [
    { field: "max_amount" as const, label: "Capital" },
    { field: "project_max" as const, label: "Project" },
    { field: "sponsorship_max" as const, label: "Sponsor" },
    { field: "staff_and_social_max" as const, label: "Staff/Social" },
    { field: "media_and_event_max" as const, label: "Media/Event" },
    { field: "computer_max" as const, label: "Computer" },
  ];

  // Errors object for DsAutoComplete (empty, just needs the shape)
  const noErrors: Record<string, { message: string }> = {};
</script>

{#if !isAdmin}
  <div class="p-4">
    <p class="text-red-600">Access denied. You must be an admin to view this page.</p>
  </div>
{:else}
  <div class="p-4">
    <div class="mb-6">
      <h1 class="text-2xl font-bold">PO Approvers</h1>
      <p class="text-neutral-600">
        Bulk edit approval limits and division assignments for all PO approvers.
      </p>
    </div>

    <div class="mb-4 flex items-center gap-2">
      <input
        type="text"
        placeholder="search..."
        bind:value={searchTerm}
        class="flex-1 rounded-sm border border-neutral-300 px-2 py-1 text-base"
      />
      <span class="text-sm text-neutral-600">{sortedItems.length} of {items.length} approvers</span>
    </div>

    {#if items.length === 0}
      <p class="text-neutral-500">No PO approvers found.</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="border-b border-neutral-300">
              <th class="pr-4 pb-2 text-left">
                <button class="hover:underline" onclick={() => toggleSort("name")}>
                  Name
                  {#if sortColumn === "name"}
                    <Icon
                      icon={sortOrder === "asc" ? "mdi:sort-ascending" : "mdi:sort-descending"}
                      class="inline w-4"
                    />
                  {/if}
                </button>
              </th>
              {#each amountColumns as col}
                <th class="pr-2 pb-2 text-right">
                  <button class="hover:underline" onclick={() => toggleSort(col.field)}>
                    {col.label}
                    {#if sortColumn === col.field}
                      <Icon
                        icon={sortOrder === "asc" ? "mdi:sort-ascending" : "mdi:sort-descending"}
                        class="inline w-4"
                      />
                    {/if}
                  </button>
                </th>
              {/each}
              <th class="pr-4 pb-2 text-left">Divisions</th>
              <th class="pb-2 text-left">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each sortedItems as item (item.id)}
              {@const propsId = item.po_approver_props_id!}
              <tr class="border-b border-neutral-200">
                <td class="py-2 pr-4 whitespace-nowrap">
                  {item.given_name}
                  {item.surname}
                </td>
                {#each amountColumns as col}
                  <td class="py-2 pr-2 text-right">
                    <input
                      type="number"
                      min="0"
                      step="0.01"
                      class="w-20 rounded-sm border border-neutral-300 px-1 py-1 text-right text-sm"
                      value={getAmount(item, col.field)}
                      oninput={(e) =>
                        updateAmount(item, col.field, parseFloat(e.currentTarget.value) || 0)}
                    />
                  </td>
                {/each}
                <td class="py-2 pr-4">
                  <div class="flex flex-wrap items-center gap-1">
                    {#each getDivisions(item) as divisionId}
                      <DsLabel color="purple" title={divisionName(divisionId)}>
                        {divisionCode(divisionId)}
                        <button
                          type="button"
                          class="ml-1 text-red-500 hover:text-red-700"
                          title="Remove division"
                          onclick={() => removeDivision(item, divisionId)}
                        >
                          x
                        </button>
                      </DsLabel>
                    {/each}
                    {#if getDivisions(item).length === 0}
                      <span class="text-xs text-neutral-400">All</span>
                    {/if}
                    {#if divisionEditRowId === propsId && divisionsIndex}
                      <div class="min-w-40">
                        <DsAutoComplete
                          bind:value={divisionSearchValue}
                          index={divisionsIndex}
                          errors={noErrors}
                          fieldName="division"
                          uiName="Division"
                          multi={true}
                          excludeIds={getDivisions(item)}
                          choose={(id) => addDivision(item, id)}
                        >
                          {#snippet resultTemplate(option)}
                            {divisionDisplay(option)}
                          {/snippet}
                        </DsAutoComplete>
                      </div>
                    {:else}
                      <button
                        type="button"
                        class="text-xs text-blue-600 hover:text-blue-800"
                        onclick={() => {
                          divisionSearchValue = "";
                          divisionEditRowId = propsId;
                        }}
                      >
                        + add
                      </button>
                    {/if}
                  </div>
                </td>
                <td class="py-2">
                  {#if isEntryModified(item)}
                    <div class="flex items-center gap-2">
                      <button
                        type="button"
                        class="rounded-sm bg-blue-500 px-2 py-1 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
                        disabled={savingEntryId === propsId}
                        onclick={() => saveEntry(item)}
                      >
                        {savingEntryId === propsId ? "Saving..." : "Save"}
                      </button>
                      <button
                        type="button"
                        class="rounded-sm bg-neutral-200 px-2 py-1 text-sm text-neutral-700 hover:bg-neutral-300"
                        disabled={savingEntryId === propsId}
                        onclick={() => cancelEntry(item)}
                      >
                        Cancel
                      </button>
                      {#if entryErrors[propsId]}
                        <span class="text-sm text-red-600">{entryErrors[propsId]}</span>
                      {/if}
                    </div>
                  {:else}
                    <div class="flex items-center gap-2">
                      <DsActionButton
                        action={() => openDeleteConfirm(item)}
                        icon="mdi:delete"
                        title="Delete"
                        color="red"
                        loading={deletingEntryId === propsId}
                        disabled={deletingEntryId !== null}
                      />
                      {#if entryErrors[propsId]}
                        <span class="text-sm text-red-600">{entryErrors[propsId]}</span>
                      {/if}
                    </div>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

    {/if}

    <div class="mt-6">
      <a href="/admin_profiles/list" class="text-blue-600 hover:underline">
        &larr; Back to Admin Profiles
      </a>
    </div>
  </div>

  <DSPopover
    bind:show={showDeleteConfirm}
    title="Remove PO Approver"
    subtitle={pendingDelete
      ? `This will remove the po_approver claim for ${pendingDelete.fullName} and cascade-delete the related approver props.`
      : ""}
    error={deleteConfirmError}
    submitting={deletingEntryId !== null}
    submitLabel="Delete"
    onSubmit={deleteEntry}
    onCancel={closeDeleteConfirm}
  >
    <div class="rounded-sm border border-amber-300 bg-amber-50 p-3 text-amber-900">
      <p class="font-semibold">Confirm removal</p>
      <p class="mt-1 text-sm">
        This action removes PO approver access for this user.
      </p>
    </div>
  </DSPopover>
{/if}
