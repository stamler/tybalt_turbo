<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import { pb } from "$lib/pocketbase";
  import { invalidateAll } from "$app/navigation";
  import { globalStore } from "$lib/stores/global";
  import { pocketBaseFileHref } from "$lib/utilities";

  type Priority = "in_use" | "recent" | "dormant" | "all";
  type Tab = "missing" | "pending";

  type MissingItem = {
    id: string;
    number: string;
    description: string;
    client_name: string;
    manager_name: string;
    branch_code: string;
    project_authorization_state: "missing_pdf" | "missing_hash";
    time_entry_count: number;
    purchase_order_count: number;
    active_purchase_order_count: number;
    expense_count: number;
    latest_activity_date: string;
    can_upload: boolean;
  };

  type MissingResponse = {
    items: MissingItem[];
    counts: Record<Priority, number>;
    page: number;
    limit: number;
    total: number;
    total_pages: number;
    priority: Priority;
    pending_review_count: number;
  };

  type PendingItem = {
    id: string;
    number: string;
    description: string;
    client_name: string;
    manager_name: string;
    branch_code: string;
    status: string;
    project_authorization_doc: string;
    project_authorization_doc_url: string;
    project_authorization_doc_hash: string;
  };

  let { data } = $props();
  let activeTab = $state<Tab>("missing");
  let loadedMissing = $state<MissingResponse | null>(null);
  let missing = $derived(loadedMissing ?? data.missing);
  let pendingItems = $state<PendingItem[]>([]);
  let pendingLoaded = $state(false);
  let pendingLoading = $state(false);
  let missingLoading = $state(false);
  let uploading = $state<string | null>(null);
  let approving = $state<string | null>(null);
  let approveTarget = $state<PendingItem | null>(null);
  let approveConfirmError = $state<string | null>(null);
  let error = $state<string | null>(null);

  const priorities: { id: Priority; label: string }[] = [
    { id: "in_use", label: "In Use" },
    { id: "recent", label: "Recent" },
    { id: "dormant", label: "Dormant" },
    { id: "all", label: "All" },
  ];

  const canReviewPending = $derived($globalStore.claims.includes("accounting"));
  const pendingReviewCount = $derived(
    pendingLoaded ? pendingItems.length : (missing.pending_review_count ?? 0),
  );

  $effect(() => {
    if (activeTab === "pending" && canReviewPending && !pendingLoaded && !pendingLoading) {
      pendingLoaded = true;
      void loadPending();
    }
  });

  function fileHref(item: PendingItem) {
    return item.project_authorization_doc
      ? pocketBaseFileHref("jobs", item.id, item.project_authorization_doc)
      : "";
  }

  async function loadPending() {
    pendingLoading = true;
    error = null;
    try {
      const response = await pb.send("/api/jobs/project_authorization/pending", { method: "GET" });
      pendingItems = (response.items ?? []).map((item: PendingItem) => ({
        ...item,
        project_authorization_doc_url: fileHref(item),
      }));
    } catch (e: any) {
      error = e?.data?.message ?? e?.message ?? "Failed to load PA documents pending review.";
    } finally {
      pendingLoading = false;
    }
  }

  async function loadMissing(priority: Priority, page = 1) {
    missingLoading = true;
    error = null;
    try {
      loadedMissing = await pb.send(
        `/api/jobs/project_authorization/missing?priority=${priority}&page=${page}&limit=${missing.limit ?? 50}`,
        { method: "GET" },
      );
    } catch (e: any) {
      error = e?.data?.message ?? e?.message ?? "Failed to load missing PA jobs.";
    } finally {
      missingLoading = false;
    }
  }

  function openApproveConfirm(item: PendingItem) {
    approveTarget = item;
    approveConfirmError = null;
  }

  function closeApproveConfirm() {
    if (approving) return;
    approveTarget = null;
    approveConfirmError = null;
  }

  async function approve(item: PendingItem) {
    approving = item.id;
    approveConfirmError = null;
    try {
      await pb.send(`/api/jobs/${item.id}/project_authorization/approve`, {
        method: "POST",
        body: { project_authorization_doc_hash: item.project_authorization_doc_hash },
      });
      approveTarget = null;
      await loadPending();
      await globalStore.refreshAttentionCounts();
      await invalidateAll();
    } catch (e: any) {
      approveConfirmError = e?.data?.message ?? e?.message ?? "Failed to approve PA document.";
    } finally {
      approving = null;
    }
  }

  function confirmApproveTarget() {
    if (!approveTarget) return;
    void approve(approveTarget);
  }

  async function uploadProjectAuthorization(item: MissingItem, event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    uploading = item.id;
    error = null;
    try {
      const form = new FormData();
      form.append("project_authorization_doc", file);
      await pb.send(`/api/jobs/${item.id}/project_authorization_doc`, {
        method: "POST",
        body: form,
      });
      input.value = "";
      await loadMissing(missing.priority, missing.page);
      if (pendingLoaded) {
        await loadPending();
      }
      await globalStore.refreshAttentionCounts();
      await invalidateAll();
    } catch (e: any) {
      error =
        e?.data?.data?.project_authorization_doc?.message ??
        e?.data?.message ??
        e?.message ??
        "Failed to upload PA document.";
    } finally {
      uploading = null;
      input.value = "";
    }
  }

  function stateLabel(state: string) {
    return state === "missing_hash" ? "Missing hash" : "Missing PDF";
  }

  function usageSummary(item: MissingItem) {
    return [
      `${item.time_entry_count} time`,
      `${item.purchase_order_count} PO`,
      `${item.expense_count} expense`,
    ].join(" / ");
  }
</script>

<div class="mx-auto space-y-4 p-4">
  <div class="flex flex-wrap items-center justify-between gap-2">
    <h1 class="text-2xl font-bold">Project Authorizations</h1>
  </div>

  {#if error}
    <div class="rounded-sm bg-red-100 p-3 text-sm text-red-800">{error}</div>
  {/if}

  <div class="flex flex-wrap gap-2 border-b border-neutral-300">
    <button
      class={`px-3 py-2 text-sm font-semibold ${activeTab === "missing" ? "border-b-2 border-blue-600 text-blue-700" : "text-neutral-600"}`}
      onclick={() => (activeTab = "missing")}
    >
      Missing / Incomplete PDF
    </button>
    {#if canReviewPending}
      <button
        class={`px-3 py-2 text-sm font-semibold ${activeTab === "pending" ? "border-b-2 border-blue-600 text-blue-700" : "text-neutral-600"}`}
        onclick={() => (activeTab = "pending")}
      >
        Pending Accounting Review
        {#if pendingReviewCount > 0}
          <span class="ml-1 rounded-full bg-red-500 px-1.5 text-xs text-white">
            {pendingReviewCount}
          </span>
        {/if}
      </button>
    {/if}
  </div>

  {#if activeTab === "missing"}
    <div class="space-y-3">
      <div class="flex flex-wrap gap-2">
        {#each priorities as option}
          <button
            class={`rounded-sm border px-3 py-1.5 text-sm ${missing.priority === option.id ? "border-blue-600 bg-blue-50 text-blue-700" : "border-neutral-300 text-neutral-700 hover:bg-neutral-100"}`}
            onclick={() => loadMissing(option.id, 1)}
            disabled={missingLoading}
          >
            {option.label}
            <span class="ml-1 text-xs text-neutral-500">{missing.counts?.[option.id] ?? 0}</span>
          </button>
        {/each}
      </div>

      <div class="overflow-x-auto">
        <table class="min-w-full border-collapse text-left text-sm">
          <thead>
            <tr class="border-b border-neutral-300">
              <th class="p-2">Job</th>
              <th class="p-2">Client</th>
              <th class="p-2">Manager</th>
              <th class="p-2">Branch</th>
              <th class="p-2">PA State</th>
              <th class="p-2">Usage</th>
              <th class="p-2">Latest Activity</th>
              <th class="p-2">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each missing.items as item}
              <tr class="border-b border-neutral-200">
                <td class="p-2">
                  <a
                    href={`/jobs/${item.id}/details`}
                    class="font-semibold text-blue-600 hover:underline"
                  >
                    {item.number}
                  </a>
                  <div class="text-neutral-600">{item.description}</div>
                </td>
                <td class="p-2">{item.client_name}</td>
                <td class="p-2">{item.manager_name}</td>
                <td class="p-2">{item.branch_code}</td>
                <td class="p-2">{stateLabel(item.project_authorization_state)}</td>
                <td class="p-2">
                  {usageSummary(item)}
                  {#if item.active_purchase_order_count > 0}
                    <div class="text-xs text-neutral-500">
                      {item.active_purchase_order_count} active
                    </div>
                  {/if}
                </td>
                <td class="p-2">{item.latest_activity_date || "-"}</td>
                <td class="p-2">
                  {#if item.can_upload}
                    <label
                      class={`inline-flex cursor-pointer text-blue-600 hover:underline ${uploading !== null ? "pointer-events-none opacity-60" : ""}`}
                    >
                      {uploading === item.id ? "Uploading..." : "Upload PA"}
                      <input
                        type="file"
                        accept="application/pdf"
                        disabled={uploading !== null}
                        onchange={(event) => uploadProjectAuthorization(item, event)}
                        class="sr-only"
                      />
                    </label>
                  {:else}
                    <a href={`/jobs/${item.id}/details`} class="text-blue-600 hover:underline">
                      Open Job
                    </a>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      {#if missing.items.length === 0}
        <p class="text-neutral-600">No missing or incomplete PA PDFs in this segment.</p>
      {/if}

      <div class="flex flex-wrap items-center justify-between gap-2 text-sm">
        <div class="text-neutral-600">
          Page {missing.page} of {missing.total_pages || 1} ({missing.total} total)
        </div>
        <div class="flex gap-2">
          <button
            class="rounded-sm border border-neutral-300 px-3 py-1.5 disabled:opacity-50"
            disabled={missingLoading || missing.page <= 1}
            onclick={() => loadMissing(missing.priority, missing.page - 1)}
          >
            Previous
          </button>
          <button
            class="rounded-sm border border-neutral-300 px-3 py-1.5 disabled:opacity-50"
            disabled={missingLoading || missing.page >= missing.total_pages}
            onclick={() => loadMissing(missing.priority, missing.page + 1)}
          >
            Next
          </button>
        </div>
      </div>
    </div>
  {:else if canReviewPending}
    <div class="overflow-x-auto">
      <table class="min-w-full border-collapse text-left text-sm">
        <thead>
          <tr class="border-b border-neutral-300">
            <th class="p-2">Job</th>
            <th class="p-2">Client</th>
            <th class="p-2">Manager</th>
            <th class="p-2">Branch</th>
            <th class="p-2">Status</th>
            <th class="p-2">Hash</th>
            <th class="p-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each pendingItems as item}
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
              <td class="p-2">{item.status}</td>
              <td
                class="max-w-64 truncate p-2 font-mono text-xs"
                title={item.project_authorization_doc_hash}
              >
                {item.project_authorization_doc_hash}
              </td>
              <td class="p-2">
                <div class="flex items-center gap-2">
                  <a
                    href={item.project_authorization_doc_url}
                    target="_blank"
                    rel="noreferrer"
                    class="text-blue-600 hover:underline"
                  >
                    Open PDF
                  </a>
                  <DsActionButton
                    action={() => openApproveConfirm(item)}
                    color="green"
                    loading={approving === item.id}
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

    {#if pendingLoaded && pendingItems.length === 0}
      <p class="text-neutral-600">No PA documents are pending Accounting review.</p>
    {/if}
  {/if}

  {#if approveTarget}
    <DSPopover
      show={Boolean(approveTarget)}
      title="Approve Project Authorization"
      subtitle={`${approveTarget.number} - ${approveTarget.description}`}
      error={approveConfirmError}
      submitting={approving === approveTarget.id}
      submitLabel="Confirm Approval"
      onSubmit={confirmApproveTarget}
      onCancel={closeApproveConfirm}
    >
      <div class="space-y-3 text-sm text-neutral-700">
        <p>
          Confirming this approval records that Accounting has reviewed the uploaded PA PDF and
          accepts it as the authorization document for this job.
        </p>
        <p>
          This means you are agreeing the document meets the business criteria required to allow
          billing and related job activity against this project.
        </p>
      </div>
    </DSPopover>
  {/if}
</div>
