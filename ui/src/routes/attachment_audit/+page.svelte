<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import { pb } from "$lib/pocketbase";
  import { downloadCSV } from "$lib/utilities";
  import Icon from "@iconify/svelte";
  import { onDestroy, onMount } from "svelte";

  interface AttachmentAuditRun {
    target_key: string;
    label: string;
    collection: string;
    field: string;
    status: "running" | "completed" | "failed" | "";
    requested_by: string;
    started_at: string;
    finished_at: string;
    total_records: number;
    referenced_records: number;
    matching_records: number;
    missing_records: number;
    orphaned_files: number;
    error: string;
    has_missing_report: boolean;
    has_orphaned_report: boolean;
  }

  interface AttachmentAuditTarget {
    key: string;
    label: string;
    collection: string;
    field: string;
    latest: AttachmentAuditRun | null;
  }

  interface DeleteOrphansResponse {
    deleted_files: number;
    skipped_referenced_files: number;
    already_missing_files: number;
    skipped_invalid_files: number;
    failed_files: number;
    refresh_error: string;
    latest: AttachmentAuditRun | null;
  }

  let targets = $state<AttachmentAuditTarget[]>([]);
  let loading = $state(true);
  let refreshingTargets = $state(new Set<string>());
  let deletingTargets = $state(new Set<string>());
  let errors = $state<Record<string, string>>({});
  let notices = $state<Record<string, string>>({});
  let pollHandle: ReturnType<typeof setInterval> | null = null;
  let openHeadingHelp = $state<AttachmentAuditHeading | null>(null);
  let deleteConfirmTarget = $state<AttachmentAuditTarget | null>(null);

  type AttachmentAuditHeading = "records" | "referenced" | "present" | "missing" | "orphaned";

  const headingDescriptions: Record<AttachmentAuditHeading, string> = {
    records: "All records scanned in this collection.",
    referenced: "Records where the attachment field contains a filename.",
    present: "Referenced records whose attachment file exists in configured storage.",
    missing: "Referenced records whose attachment file was not found in configured storage.",
    orphaned: "Files found in configured storage that are not referenced by any record.",
  };

  async function fetchTargets() {
    try {
      targets = (await pb.send("/api/attachment_audit/targets", {
        method: "GET",
      })) as AttachmentAuditTarget[];
      syncPolling();
    } catch (error: any) {
      errors = {
        global: error?.response?.message ?? "Failed to load attachment audit targets",
      };
    } finally {
      loading = false;
    }
  }

  function syncPolling() {
    const hasRunningTarget = targets.some((target) => target.latest?.status === "running");
    if (hasRunningTarget && pollHandle === null) {
      pollHandle = setInterval(fetchTargets, 2000);
    }
    if (!hasRunningTarget && pollHandle !== null) {
      clearInterval(pollHandle);
      pollHandle = null;
    }
  }

  async function refreshTarget(target: AttachmentAuditTarget) {
    errors = {};
    notices = {};
    refreshingTargets = new Set([...refreshingTargets, target.key]);
    try {
      const latest = (await pb.send(`/api/attachment_audit/targets/${target.key}/refresh`, {
        method: "POST",
      })) as AttachmentAuditRun;
      targets = targets.map((row) => (row.key === target.key ? { ...row, latest } : row));
      syncPolling();
    } catch (error: any) {
      errors = {
        [target.key]: error?.response?.message ?? "Failed to refresh attachment audit",
      };
    } finally {
      const next = new Set(refreshingTargets);
      next.delete(target.key);
      refreshingTargets = next;
    }
  }

  async function downloadReport(target: AttachmentAuditTarget, report: "missing" | "orphaned") {
    errors = {};
    try {
      await downloadCSV(
        `${pb.baseUrl}/api/attachment_audit/targets/${target.key}/${report}.csv`,
        `${target.key}_${report}.csv`,
      );
    } catch (error: any) {
      errors = {
        [target.key]: error?.message ?? "Failed to download attachment audit report",
      };
    }
  }

  function openDeleteConfirm(target: AttachmentAuditTarget) {
    errors = {};
    deleteConfirmTarget = target;
  }

  function closeDeleteConfirm() {
    deleteConfirmTarget = null;
  }

  async function deleteOrphans() {
    if (!deleteConfirmTarget) return;

    const target = deleteConfirmTarget;
    errors = {};
    notices = {};
    deletingTargets = new Set([...deletingTargets, target.key]);
    try {
      const response = (await pb.send(
        `/api/attachment_audit/targets/${target.key}/delete_orphans`,
        { method: "POST" },
      )) as DeleteOrphansResponse;

      targets = targets.map((row) =>
        row.key === target.key && response.latest ? { ...row, latest: response.latest } : row,
      );
      notices = {
        [target.key]: deleteOrphansSummary(response),
      };
      if (response.refresh_error) {
        errors = {
          [target.key]: `Deleted orphan cleanup completed, but refresh failed: ${response.refresh_error}`,
        };
      }
      closeDeleteConfirm();
    } catch (error: any) {
      errors = {
        [target.key]: error?.response?.message ?? "Failed to delete orphaned attachments",
      };
    } finally {
      const next = new Set(deletingTargets);
      next.delete(target.key);
      deletingTargets = next;
    }
  }

  function deleteOrphansSummary(response: DeleteOrphansResponse): string {
    const parts = [
      `${response.deleted_files} deleted`,
      `${response.skipped_referenced_files} skipped because they are now referenced`,
      `${response.already_missing_files} already missing`,
      `${response.skipped_invalid_files} invalid cached rows skipped`,
      `${response.failed_files} failed`,
    ];
    return `Orphan cleanup complete: ${parts.join(", ")}.`;
  }

  function formatDate(value: string): string {
    if (!value) return "Never";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return date.toLocaleString();
  }

  function statusLabel(run: AttachmentAuditRun | null): string {
    if (!run?.status) return "Not run";
    if (run.status === "completed") return "Completed";
    if (run.status === "running") return "Running";
    return "Failed";
  }

  function statusClasses(run: AttachmentAuditRun | null): string {
    if (run?.status === "completed") return "border-green-300 bg-green-50 text-green-800";
    if (run?.status === "running") return "border-blue-300 bg-blue-50 text-blue-800";
    if (run?.status === "failed") return "border-red-300 bg-red-50 text-red-800";
    return "border-neutral-300 bg-neutral-50 text-neutral-600";
  }

  function toggleHeadingHelp(heading: AttachmentAuditHeading) {
    openHeadingHelp = openHeadingHelp === heading ? null : heading;
  }

  onMount(fetchTargets);

  onDestroy(() => {
    if (pollHandle !== null) {
      clearInterval(pollHandle);
    }
  });
</script>

{#snippet auditHeading(heading: AttachmentAuditHeading, label: string)}
  <span class="relative inline-flex items-center justify-end gap-1">
    <span>{label}</span>
    <button
      type="button"
      class="inline-flex items-center text-slate-500 transition-colors hover:text-slate-700"
      aria-label={`${label} explanation`}
      aria-expanded={openHeadingHelp === heading}
      aria-haspopup="dialog"
      onclick={() => toggleHeadingHelp(heading)}
      onkeydown={(event) => event.key === "Escape" && (openHeadingHelp = null)}
    >
      <Icon icon="mdi:information-outline" width="15px" />
    </button>
    {#if openHeadingHelp === heading}
      <span
        class="absolute right-0 top-full z-20 mt-1 w-56 rounded-sm border border-sky-200 bg-sky-50 p-2 text-left text-xs font-normal normal-case text-sky-950 shadow-sm"
        role="tooltip"
      >
        {headingDescriptions[heading]}
      </span>
    {/if}
  </span>
{/snippet}

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <h1 class="text-2xl font-bold">Attachment Audit</h1>
    <DsActionButton action={fetchTargets} icon="mdi:refresh" title="Reload" color="blue" />
  </div>

  {#if errors.global}
    <div class="rounded-sm border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700">
      {errors.global}
    </div>
  {/if}

  {#if loading}
    <div class="text-neutral-500">Loading…</div>
  {:else}
    <div class="overflow-x-auto rounded-sm border border-neutral-300">
      <table class="min-w-full divide-y divide-neutral-200 text-sm">
        <thead class="bg-neutral-100 text-left text-xs font-semibold uppercase tracking-normal text-neutral-600">
          <tr>
            <th class="px-3 py-2">Target</th>
            <th class="px-3 py-2">Status</th>
            <th class="px-3 py-2 text-right">{@render auditHeading("records", "Records")}</th>
            <th class="px-3 py-2 text-right">{@render auditHeading("referenced", "Referenced")}</th>
            <th class="px-3 py-2 text-right">{@render auditHeading("present", "Present")}</th>
            <th class="px-3 py-2 text-right">{@render auditHeading("missing", "Missing")}</th>
            <th class="px-3 py-2 text-right">{@render auditHeading("orphaned", "Orphaned")}</th>
            <th class="px-3 py-2">Finished</th>
            <th class="px-3 py-2 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-200 bg-white">
          {#each targets as target (target.key)}
            {@const run = target.latest}
            <tr>
              <td class="px-3 py-2">
                <div class="font-semibold">{target.label}</div>
                <div class="font-mono text-xs text-neutral-500">
                  {target.collection}.{target.field}
                </div>
                {#if errors[target.key]}
                  <div class="mt-1 text-xs text-red-600">{errors[target.key]}</div>
                {/if}
                {#if notices[target.key]}
                  <div class="mt-1 text-xs text-green-700">{notices[target.key]}</div>
                {/if}
                {#if run?.status === "failed" && run.error}
                  <div class="mt-1 text-xs text-red-600">{run.error}</div>
                {/if}
              </td>
              <td class="px-3 py-2">
                <span class="inline-flex rounded-sm border px-2 py-1 text-xs {statusClasses(run)}">
                  {statusLabel(run)}
                </span>
              </td>
              <td class="px-3 py-2 text-right tabular-nums">{run?.total_records ?? 0}</td>
              <td class="px-3 py-2 text-right tabular-nums">{run?.referenced_records ?? 0}</td>
              <td class="px-3 py-2 text-right tabular-nums">{run?.matching_records ?? 0}</td>
              <td class="px-3 py-2 text-right tabular-nums">{run?.missing_records ?? 0}</td>
              <td class="px-3 py-2 text-right tabular-nums">{run?.orphaned_files ?? 0}</td>
              <td class="px-3 py-2 whitespace-nowrap">{formatDate(run?.finished_at ?? "")}</td>
              <td class="px-3 py-2">
                <div class="flex justify-end gap-1">
                  <DsActionButton
                    action={() => refreshTarget(target)}
                    icon="mdi:refresh"
                    title="Refresh Audit"
                    color="green"
                    loading={refreshingTargets.has(target.key)}
                  />
                  <DsActionButton
                    action={() => downloadReport(target, "missing")}
                    icon="mdi:file-alert-outline"
                    title="Download Missing Report"
                    color="yellow"
                    disabled={!run?.has_missing_report || (run?.missing_records ?? 0) === 0}
                  />
                  <DsActionButton
                    action={() => downloadReport(target, "orphaned")}
                    icon="mdi:file-search-outline"
                    title="Download Orphaned Report"
                    color="blue"
                    disabled={!run?.has_orphaned_report || (run?.orphaned_files ?? 0) === 0}
                  />
                  <DsActionButton
                    action={() => openDeleteConfirm(target)}
                    icon="mdi:delete-alert-outline"
                    title="Delete Orphaned Attachments"
                    color="red"
                    loading={deletingTargets.has(target.key)}
                    disabled={!run?.has_orphaned_report || (run?.orphaned_files ?? 0) === 0}
                  />
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

{#if deleteConfirmTarget}
  <DSPopover
    show={true}
    title="Delete Orphaned Attachments"
    subtitle={deleteConfirmTarget.label}
    submitLabel="Delete Orphans"
    submitting={deletingTargets.has(deleteConfirmTarget.key)}
    onSubmit={deleteOrphans}
    onCancel={closeDeleteConfirm}
  >
    <p class="text-sm text-neutral-700">
      This will use the latest cached orphaned report for {deleteConfirmTarget.label} as the basis
      for deletion. Before each file is deleted, the server will re-check current records and skip
      any file that is now referenced.
    </p>
    <p class="text-sm font-semibold text-red-700">
      Deleted files are removed from configured storage and cannot be restored from this screen.
    </p>
  </DSPopover>
{/if}
