<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import { pb } from "$lib/pocketbase";
  import { invalidateAll } from "$app/navigation";
  import { globalStore } from "$lib/stores/global";
  import { pocketBaseFileHref, shortDate } from "$lib/utilities";

  type Priority = "in_use" | "recent" | "dormant" | "all";
  type Tab = "missing" | "pending" | "rejected";

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
    rejected_count: number;
  };

  type PendingItem = {
    id: string;
    number: string;
    description: string;
    client_po: string;
    client_name: string;
    manager_name: string;
    branch_code: string;
    status: string;
    project_authorization_doc: string;
    project_authorization_doc_url: string;
    project_authorization_doc_hash: string;
  };

  type RejectedItem = {
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
    pa_rejector_name: string;
    pa_rejected: string;
    pa_rejection_reason: string;
    can_upload: boolean;
  };

  let { data } = $props();
  let activeTab = $state<Tab>("missing");
  let loadedMissing = $state<MissingResponse | null>(null);
  let missing = $derived(loadedMissing ?? data.missing);
  let pendingItems = $state<PendingItem[]>([]);
  let pendingLoaded = $state(false);
  let pendingLoading = $state(false);
  let rejectedItems = $state<RejectedItem[]>([]);
  let rejectedLoaded = $state(false);
  let rejectedLoading = $state(false);
  let missingLoading = $state(false);
  let uploadCertifications = $state<Record<string, boolean>>({});
  let uploading = $state<string | null>(null);
  let approving = $state<string | null>(null);
  let approveTarget = $state<PendingItem | null>(null);
  let approveConfirmError = $state<string | null>(null);
  let rejecting = $state<string | null>(null);
  let rejectTarget = $state<PendingItem | null>(null);
  let rejectionReason = $state("");
  let rejectConfirmError = $state<string | null>(null);
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
  const rejectedCount = $derived(
    rejectedLoaded ? rejectedItems.length : (missing.rejected_count ?? 0),
  );

  $effect(() => {
    if (activeTab === "pending" && canReviewPending && !pendingLoaded && !pendingLoading) {
      pendingLoaded = true;
      void loadPending();
    }
    if (activeTab === "rejected" && !rejectedLoaded && !rejectedLoading) {
      rejectedLoaded = true;
      void loadRejected();
    }
  });

  function fileHref(item: Pick<PendingItem, "id" | "project_authorization_doc">) {
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

  async function loadRejected() {
    rejectedLoading = true;
    error = null;
    try {
      const response = await pb.send("/api/jobs/project_authorization/rejected", { method: "GET" });
      rejectedItems = (response.items ?? []).map((item: RejectedItem) => ({
        ...item,
        project_authorization_doc_url: fileHref(item),
      }));
    } catch (e: any) {
      error = e?.data?.message ?? e?.message ?? "Failed to load rejected PA documents.";
    } finally {
      rejectedLoading = false;
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

  function openRejectConfirm(item: PendingItem) {
    rejectTarget = item;
    rejectionReason = "";
    rejectConfirmError = null;
  }

  function closeRejectConfirm() {
    if (rejecting) return;
    rejectTarget = null;
    rejectionReason = "";
    rejectConfirmError = null;
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

  async function reject(item: PendingItem) {
    const reason = rejectionReason.trim();
    if (reason.length < 4) {
      rejectConfirmError = "Rejection reason must be at least 4 characters long.";
      return;
    }
    rejecting = item.id;
    rejectConfirmError = null;
    try {
      await pb.send(`/api/jobs/${item.id}/project_authorization/reject`, {
        method: "POST",
        body: {
          project_authorization_doc_hash: item.project_authorization_doc_hash,
          rejection_reason: reason,
        },
      });
      rejectTarget = null;
      rejectionReason = "";
      await loadPending();
      if (rejectedLoaded) {
        await loadRejected();
      }
      await globalStore.refreshAttentionCounts();
      await invalidateAll();
    } catch (e: any) {
      rejectConfirmError = e?.data?.message ?? e?.message ?? "Failed to reject PA document.";
    } finally {
      rejecting = null;
    }
  }

  function confirmRejectTarget() {
    if (!rejectTarget) return;
    void reject(rejectTarget);
  }

  function setUploadCertification(id: string, certified: boolean) {
    uploadCertifications = { ...uploadCertifications, [id]: certified };
  }

  async function uploadProjectAuthorization(item: { id: string }, event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (!uploadCertifications[item.id]) {
      error = "Confirm that the PDF contains a completed TBT Engineering Project Authorization Form.";
      input.value = "";
      return;
    }
    uploading = item.id;
    error = null;
    try {
      const form = new FormData();
      form.append("project_authorization_doc", file);
      form.append("project_authorization_certified", "true");
      await pb.send(`/api/jobs/${item.id}/project_authorization_doc`, {
        method: "POST",
        body: form,
      });
      input.value = "";
      setUploadCertification(item.id, false);
      await loadMissing(missing.priority, missing.page);
      if (pendingLoaded) {
        await loadPending();
      }
      if (rejectedLoaded) {
        await loadRejected();
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
    <button
      class={`px-3 py-2 text-sm font-semibold ${activeTab === "rejected" ? "border-b-2 border-blue-600 text-blue-700" : "text-neutral-600"}`}
      onclick={() => (activeTab = "rejected")}
    >
      Rejected PA Documents
      {#if rejectedCount > 0}
        <span class="ml-1 rounded-full bg-red-500 px-1.5 text-xs text-white">
          {rejectedCount}
        </span>
      {/if}
    </button>
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
                    <div class="flex max-w-sm flex-col gap-2">
                      <label class="flex items-start gap-2 text-xs text-neutral-700">
                        <input
                          type="checkbox"
                          checked={Boolean(uploadCertifications[item.id])}
                          disabled={uploading !== null}
                          onchange={(event) =>
                            setUploadCertification(
                              item.id,
                              (event.currentTarget as HTMLInputElement).checked,
                            )}
                          class="mt-0.5"
                        />
                        <span>
                          I certify that the attached PDF contains a completed TBT Engineering
                          Project Authorization Form. The PDF may also include additional supporting
                          documentation.
                        </span>
                      </label>
                      <label
                        class={`inline-flex cursor-pointer text-blue-600 hover:underline ${uploading !== null || !uploadCertifications[item.id] ? "pointer-events-none opacity-60" : ""}`}
                      >
                        {uploading === item.id ? "Uploading..." : "Upload PA"}
                        <input
                          type="file"
                          accept="application/pdf"
                          disabled={uploading !== null || !uploadCertifications[item.id]}
                          onchange={(event) => uploadProjectAuthorization(item, event)}
                          class="sr-only"
                        />
                      </label>
                    </div>
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
  {:else if activeTab === "pending" && canReviewPending}
    <div class="overflow-x-auto">
      <table class="min-w-full border-collapse text-left text-sm">
        <thead>
          <tr class="border-b border-neutral-300">
            <th class="p-2">Job</th>
            <th class="p-2">Client</th>
            <th class="p-2">Client PO</th>
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
              <td class="p-2">{item.client_po || "-"}</td>
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
                    disabled={rejecting !== null}
                  >
                    Approve
                  </DsActionButton>
                  <DsActionButton
                    action={() => openRejectConfirm(item)}
                    color="red"
                    loading={rejecting === item.id}
                    disabled={approving !== null}
                  >
                    Reject
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
  {:else if activeTab === "rejected"}
    <div class="overflow-x-auto">
      <table class="min-w-full border-collapse text-left text-sm">
        <thead>
          <tr class="border-b border-neutral-300">
            <th class="p-2">Job</th>
            <th class="p-2">Client</th>
            <th class="p-2">Manager</th>
            <th class="p-2">Branch</th>
            <th class="p-2">Rejected By</th>
            <th class="p-2">Reason</th>
            <th class="p-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each rejectedItems as item}
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
              <td class="p-2">
                {item.pa_rejector_name || "-"}
                {#if item.pa_rejected}
                  <div class="text-xs text-neutral-500">{shortDate(item.pa_rejected, true)}</div>
                {/if}
              </td>
              <td class="max-w-md p-2">{item.pa_rejection_reason || "-"}</td>
              <td class="p-2">
                <div class="flex flex-wrap items-center gap-2">
                  <a
                    href={item.project_authorization_doc_url}
                    target="_blank"
                    rel="noreferrer"
                    class="text-blue-600 hover:underline"
                  >
                    Open PDF
                  </a>
                  {#if item.can_upload}
                    <div class="flex max-w-sm flex-col gap-2">
                      <label class="flex items-start gap-2 text-xs text-neutral-700">
                        <input
                          type="checkbox"
                          checked={Boolean(uploadCertifications[item.id])}
                          disabled={uploading !== null}
                          onchange={(event) =>
                            setUploadCertification(
                              item.id,
                              (event.currentTarget as HTMLInputElement).checked,
                            )}
                          class="mt-0.5"
                        />
                        <span>
                          I certify that the attached PDF contains a completed TBT Engineering
                          Project Authorization Form. The PDF may also include additional supporting
                          documentation.
                        </span>
                      </label>
                      <label
                        class={`inline-flex cursor-pointer text-blue-600 hover:underline ${uploading !== null || !uploadCertifications[item.id] ? "pointer-events-none opacity-60" : ""}`}
                      >
                        {uploading === item.id ? "Uploading..." : "Upload Replacement"}
                        <input
                          type="file"
                          accept="application/pdf"
                          disabled={uploading !== null || !uploadCertifications[item.id]}
                          onchange={(event) => uploadProjectAuthorization(item, event)}
                          class="sr-only"
                        />
                      </label>
                    </div>
                  {:else}
                    <a href={`/jobs/${item.id}/details`} class="text-blue-600 hover:underline">
                      Open Job
                    </a>
                  {/if}
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    {#if rejectedLoaded && rejectedItems.length === 0}
      <p class="text-neutral-600">No PA documents have been rejected.</p>
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

  {#if rejectTarget}
    <DSPopover
      show={Boolean(rejectTarget)}
      title="Reject Project Authorization"
      subtitle={`${rejectTarget.number} - ${rejectTarget.description}`}
      error={rejectConfirmError}
      submitting={rejecting === rejectTarget.id}
      submitLabel="Reject PA"
      onSubmit={confirmRejectTarget}
      onCancel={closeRejectConfirm}
    >
      <div class="space-y-3 text-sm text-neutral-700">
        <p>
          Rejecting this PA records that Accounting reviewed the uploaded PDF and found it
          incomplete or unacceptable. The uploader will be notified with the reason below.
        </p>
        <label class="flex flex-col gap-1">
          <span class="font-semibold">Rejection Reason</span>
          <textarea
            bind:value={rejectionReason}
            class="min-h-28 rounded-sm border border-neutral-300 p-2"
          ></textarea>
        </label>
      </div>
    </DSPopover>
  {/if}
</div>
