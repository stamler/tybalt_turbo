<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsActionButton from "./DSActionButton.svelte";
  import type { AbsorbActionsResponse } from "$lib/pocketbase-types";
  import { goto } from "$app/navigation";
  import { getAbsorbRedirectUrl } from "$lib/utilities";

  let {
    collectionName = undefined,
  }: {
    collectionName?: string;
  } = $props();

  let errors = $state<Record<string, { message: string }>>({});
  let items = $state<AbsorbActionsResponse[]>([]);
  let loading = $state<boolean>(true);

  // Track per-item confirmation states
  type ConfirmState = {
    undo: boolean;
    commit: boolean;
  };
  let confirms = $state<Record<string, ConfirmState>>({});

  function setConfirm(id: string, key: keyof ConfirmState, value: boolean) {
    const current = confirms[id] ?? { undo: false, commit: false };
    confirms = { ...confirms, [id]: { ...current, [key]: value } };
  }

  async function loadItems() {
    loading = true;
    try {
      const list = await pb.collection("absorb_actions").getFullList({
        sort: "-created",
        ...(collectionName ? { filter: `collection_name='${collectionName}'` } : {}),
      });
      items = list;
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "Failed to load absorb actions" } };
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    loadItems();
  });

  async function undoAbsorb(item: AbsorbActionsResponse) {
    try {
      await pb.send(`/api/${item.collection_name}/undo_absorb`, { method: "POST" });
      // After successful undo, redirect to appropriate page
      const url = await getAbsorbRedirectUrl(item.collection_name, item.target_id);
      goto(url);
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "Failed to undo absorb action" } };
      }
    } finally {
      setConfirm(item.id, "undo", false);
    }
  }

  async function commitAbsorb(item: AbsorbActionsResponse) {
    try {
      await pb.collection("absorb_actions").delete(item.id);
      // After successful commit, redirect to appropriate page
      const url = await getAbsorbRedirectUrl(item.collection_name, item.target_id);
      goto(url);
    } catch (error: unknown) {
      if (error instanceof Error) {
        errors = { global: { message: error.message } };
      } else {
        errors = { global: { message: "Failed to commit absorb action" } };
      }
    } finally {
      setConfirm(item.id, "commit", false);
    }
  }

  function formatCollectionLabel(name: string): string {
    if (name === "client_contacts") return "Client Contacts";
    if (name === "clients") return "Clients";
    if (name === "vendors") return "Vendors";
    return name;
  }

  function countAbsorbed(records: unknown): number | null {
    if (Array.isArray(records)) return records.length;
    return null;
  }

  function countUpdatedRefs(updated: unknown): number | null {
    try {
      if (updated && typeof updated === "object") {
        // updated[table][column][recordId] = oldValue
        const level1 = Object.values(updated as Record<string, any>);
        let count = 0;
        for (const table of level1) {
          for (const column of Object.values(table as Record<string, any>)) {
            count += Object.keys(column as Record<string, any>).length;
          }
        }
        return count;
      }
    } catch (_) {
      // ignore
    }
    return null;
  }

  function entriesOf(obj: unknown): [string, unknown][] {
    if (obj && typeof obj === "object") {
      return Object.entries(obj as Record<string, unknown>);
    }
    return [];
  }

  function formatPrevValue(value: unknown): string {
    if (
      value === null ||
      typeof value === "string" ||
      typeof value === "number" ||
      typeof value === "boolean"
    ) {
      return String(value);
    }
    try {
      return JSON.stringify(value);
    } catch (_) {
      return String(value);
    }
  }

  // Track expanded/collapsed state per item per table (collapsed by default)
  let expandedTables = $state<Record<string, Record<string, boolean>>>({});
  function isTableExpanded(itemId: string, tableName: string): boolean {
    return !!expandedTables[itemId]?.[tableName];
  }
  function setTableExpanded(itemId: string, tableName: string, value: boolean) {
    const current = expandedTables[itemId] ?? {};
    expandedTables = { ...expandedTables, [itemId]: { ...current, [tableName]: value } };
  }
  function toggleTable(itemId: string, tableName: string) {
    setTableExpanded(itemId, tableName, !isTableExpanded(itemId, tableName));
  }
</script>

<div class="flex w-full flex-col gap-4 p-4">
  <h1 class="text-xl font-bold">Absorb Actions</h1>

  {#if loading}
    <div class="text-neutral-500">Loading...</div>
  {:else if items.length === 0}
    <div class="rounded-sm border border-neutral-200 p-4 text-neutral-600">
      No pending absorb actions.
    </div>
  {:else}
    <ul class="flex flex-col gap-4">
      {#each items as item}
        <li class="rounded-lg border-2 border-yellow-500 bg-yellow-50 p-4">
          <div class="mb-2 flex items-center justify-between">
            <div>
              <h2 class="text-lg font-bold text-yellow-800">
                Pending Absorb — {formatCollectionLabel(item.collection_name)}
              </h2>
              <p class="text-sm text-yellow-800/80">Target ID: {item.target_id}</p>
            </div>
            <div class="text-xs text-yellow-800/70">
              Created: {new Date(item.created).toLocaleString()}
            </div>
          </div>

          <div class="mb-3 grid grid-cols-1 gap-2 md:grid-cols-3">
            <div class="rounded-sm bg-yellow-100 p-2 text-sm">
              <div class="font-semibold">Summary</div>
              <div>Absorbed records: {countAbsorbed(item.absorbed_records) ?? "n/a"}</div>
              <div>Updated references: {countUpdatedRefs(item.updated_references) ?? "n/a"}</div>
            </div>
            <div class="rounded-sm bg-yellow-100 p-2 text-sm md:col-span-2">
              <div class="font-semibold">Details</div>
              <div class="mt-1 grid grid-cols-1 gap-2 md:grid-cols-2">
                <div>
                  <div class="text-xs font-medium">Absorbed Records</div>
                  <pre class="mt-1 max-h-48 overflow-auto rounded-sm bg-yellow-200/50 p-2 text-xs">
{JSON.stringify(item.absorbed_records, null, 2)}
                  </pre>
                </div>
                <div>
                  <div class="text-xs font-medium">Updated References</div>
                  <div class="mt-1 flex flex-col gap-3">
                    {#each entriesOf(item.updated_references) as [tableName, columns]}
                      <div class="rounded-sm border border-yellow-200">
                        <button
                          class="flex w-full items-center justify-between bg-yellow-200/60 px-2 py-1 text-left text-xs font-semibold"
                          onclick={() => toggleTable(item.id, tableName as string)}
                          aria-expanded={isTableExpanded(item.id, tableName as string)}
                        >
                          <span>{tableName}</span>
                          <span>{isTableExpanded(item.id, tableName as string) ? "▾" : "▸"}</span>
                        </button>
                        {#if isTableExpanded(item.id, tableName as string)}
                          <div class="p-2">
                            {#each entriesOf(columns) as [columnName, rows]}
                              <div class="mb-3">
                                <div class="text-xs font-semibold">{columnName}</div>
                                <table class="mt-1 w-full table-auto border-collapse text-xs">
                                  <thead>
                                    <tr>
                                      <th class="border-b border-yellow-300 px-2 py-1 text-left"
                                        >row id</th
                                      >
                                      <th class="border-b border-yellow-300 px-2 py-1 text-left"
                                        >previous value</th
                                      >
                                    </tr>
                                  </thead>
                                  <tbody>
                                    {#each entriesOf(rows) as [rowId, prevValue]}
                                      <tr>
                                        <td class="border-b border-yellow-100 px-2 py-1 font-mono"
                                          >{rowId}</td
                                        >
                                        <td class="border-b border-yellow-100 px-2 py-1 break-all"
                                          >{formatPrevValue(prevValue)}</td
                                        >
                                      </tr>
                                    {/each}
                                  </tbody>
                                </table>
                              </div>
                            {/each}
                          </div>
                        {/if}
                      </div>
                    {/each}
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="flex flex-wrap gap-2">
            {#if confirms[item.id]?.undo}
              <div class="flex flex-col gap-2">
                <p class="font-bold text-red-600">Confirm undo? This cannot be reversed.</p>
                <div class="flex gap-2">
                  <DsActionButton action={() => undoAbsorb(item)} color="red"
                    >Confirm Undo</DsActionButton
                  >
                  <DsActionButton action={() => setConfirm(item.id, "undo", false)}
                    >Cancel</DsActionButton
                  >
                </div>
              </div>
            {:else if confirms[item.id]?.commit}
              <div class="flex flex-col gap-2">
                <p class="font-bold text-red-600">Confirm commit? This cannot be reversed.</p>
                <div class="flex gap-2">
                  <DsActionButton action={() => commitAbsorb(item)} color="red"
                    >Confirm Commit</DsActionButton
                  >
                  <DsActionButton action={() => setConfirm(item.id, "commit", false)}
                    >Cancel</DsActionButton
                  >
                </div>
              </div>
            {:else}
              <DsActionButton action={() => setConfirm(item.id, "undo", true)} color="yellow"
                >Undo</DsActionButton
              >
              <DsActionButton action={() => setConfirm(item.id, "commit", true)} color="green"
                >Commit</DsActionButton
              >
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  {#if errors.global}
    <div class="text-red-600">{errors.global.message}</div>
  {/if}
</div>
