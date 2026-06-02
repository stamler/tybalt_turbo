<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { pb } from "$lib/pocketbase";
  import { invalidateAll } from "$app/navigation";

  let { data } = $props();
  let openedHashes = $state(new Set<string>());
  let approving = $state<string | null>(null);
  let error = $state<string | null>(null);

  function markOpened(hash: string) {
    openedHashes = new Set([...openedHashes, hash]);
  }

  async function approve(item: any) {
    approving = item.id;
    error = null;
    try {
      await pb.send(`/api/jobs/${item.id}/project_authorization/approve`, {
        method: "POST",
        body: { project_authorization_doc_hash: item.project_authorization_doc_hash },
      });
      await invalidateAll();
    } catch (e: any) {
      error = e?.data?.message ?? e?.message ?? "Failed to approve PA document.";
    } finally {
      approving = null;
    }
  }
</script>

<div class="mx-auto space-y-4 p-4">
  <div class="flex items-center justify-between gap-2">
    <h1 class="text-2xl font-bold">Project Authorization Queue</h1>
  </div>

  {#if error}
    <div class="rounded-sm bg-red-100 p-3 text-sm text-red-800">{error}</div>
  {/if}

  <div class="overflow-x-auto">
    <table class="min-w-full border-collapse text-left text-sm">
      <thead>
        <tr class="border-b border-neutral-300">
          <th class="p-2">Job</th>
          <th class="p-2">Client</th>
          <th class="p-2">Manager</th>
          <th class="p-2">Branch</th>
          <th class="p-2">Hash</th>
          <th class="p-2">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each data.items as item}
          <tr class="border-b border-neutral-200">
            <td class="p-2">
              <a href={`/jobs/${item.id}/details`} class="font-semibold text-blue-600 hover:underline">
                {item.number}
              </a>
              <div class="text-neutral-600">{item.description}</div>
            </td>
            <td class="p-2">{item.client_name}</td>
            <td class="p-2">{item.manager_name}</td>
            <td class="p-2">{item.branch_code}</td>
            <td class="max-w-64 truncate p-2 font-mono text-xs" title={item.project_authorization_doc_hash}>
              {item.project_authorization_doc_hash}
            </td>
            <td class="p-2">
              <div class="flex items-center gap-2">
                <a
                  href={item.project_authorization_doc_url}
                  target="_blank"
                  rel="noreferrer"
                  class="text-blue-600 hover:underline"
                  onclick={() => markOpened(item.project_authorization_doc_hash)}
                >
                  Open PDF
                </a>
                <DsActionButton
                  action={() => approve(item)}
                  color="green"
                  loading={approving === item.id}
                  disabled={!openedHashes.has(item.project_authorization_doc_hash)}
                >
                  Approve
                </DsActionButton>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>

  {#if data.items.length === 0}
    <p class="text-neutral-600">No PA documents are pending review.</p>
  {/if}
</div>
