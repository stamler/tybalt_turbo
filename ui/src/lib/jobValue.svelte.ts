// Project-value changes for one job, or the signed-in approver's queue. The job
// page creates one instance per job, so a late response for an earlier job can
// only update that job's instance, never the one now shown.
import { pb } from "$lib/pocketbase";
import { invalidateAll } from "$app/navigation";
import { globalStore } from "$lib/stores/global";
import { workflowError, type WorkflowRecord } from "$lib/clientEditors";

export const valueCAD = (amount: unknown) =>
  new Intl.NumberFormat("en-CA", {
    style: "currency",
    currency: "CAD",
    maximumFractionDigits: 0,
  }).format(Number(amount) || 0);

// The date part of a stored timestamp, such as 2026-10-14.
export const valueDate = (timestamp: unknown) => String(timestamp ?? "").slice(0, 10);

export class JobValue {
  items = $state<WorkflowRecord[]>([]);
  loading = $state(true);
  error = $state("");
  busy = $state(false);
  // The proposal form: open for a new proposal (editing null) or a correction.
  proposing = $state(false);
  editing = $state<WorkflowRecord | null>(null);
  readonly jobId?: string;
  readonly queue: boolean;

  constructor(jobId?: string, queue = false) {
    this.jobId = jobId;
    this.queue = queue;
  }

  // Approved values, newest first. The approval sequence orders a job's values.
  approved = $derived(
    this.items
      .filter((item) => item.status === "Approved")
      .sort((a, b) => b.approval_sequence - a.approval_sequence),
  );
  // The approval that set the current value. The server marks it, since the
  // history can leave it out when the viewer may see only some approvals.
  current = $derived(this.approved.find((item) => item.current));
  // A pending or rejected change; a job has at most one.
  inProgress = $derived(this.items.find((item) => item.status !== "Approved"));

  async reload() {
    this.loading = true;
    try {
      const result = await pb.send(
        `/api/job_attachments?${this.queue ? "queue=approval" : `job=${this.jobId}`}`,
        { method: "GET", requestKey: null },
      );
      this.items = result.items;
    } catch (e) {
      this.error = workflowError(e);
    } finally {
      this.loading = false;
    }
  }

  // Reload this history and anything else on the page that shows the value.
  async refresh() {
    await this.reload();
    await Promise.all([globalStore.refreshAttentionCounts(), invalidateAll()]);
  }

  openProposal(item: WorkflowRecord | null) {
    this.editing = item;
    this.error = "";
    this.proposing = true;
  }

  // Approve, reject, or delete a change. Returns whether it succeeded.
  async act(
    item: WorkflowRecord,
    name: "approve" | "reject" | "delete",
    rejectionReason = "",
  ): Promise<boolean> {
    this.busy = true;
    this.error = "";
    try {
      await pb.send(`/api/job_attachments/${item.id}${name === "delete" ? "" : `/${name}`}`, {
        method: name === "delete" ? "DELETE" : "POST",
        body: {
          // Use the displayed revision for approval, rejection, and deletion.
          // A decision can change it even when the PDF hash stays the same.
          version: item.version,
          document_hash: item.document_hash,
          // A rejection records the job data shown here.
          review_token: item.current_review_token,
          rejection_reason: rejectionReason,
        },
      });
      await this.refresh();
      return true;
    } catch (e) {
      this.error = workflowError(e);
      return false;
    } finally {
      this.busy = false;
    }
  }

  // Open a change's PDF in a new tab, or save it as a file.
  async pdf(item: WorkflowRecord, download = false) {
    this.error = "";
    try {
      const response = await fetch(`${pb.baseURL}/api/job_attachments/${item.id}/document`, {
        headers: { Authorization: pb.authStore.token },
      });
      if (!response.ok) throw new Error("Cannot load this PDF.");
      const url = URL.createObjectURL(await response.blob());
      if (download) {
        const link = document.createElement("a");
        link.href = url;
        link.download = `${item.job_number || "project"} value ${valueDate(item.approved || item.uploaded)}.pdf`;
        link.click();
      } else window.open(url, "_blank");
      setTimeout(() => URL.revokeObjectURL(url), 60000);
    } catch (e) {
      this.error = workflowError(e);
    }
  }
}

// Project details that approval of a value change requires and the job lacks.
// The server checks the job in full when the change is saved; this catches the
// common gaps on older projects before the form is filled in. A missing
// authorizing document is not listed: the server treats the project as PA and
// approval records it.
export function valueBlockers(job: {
  status?: string;
  invoicing_information?: string;
  project_completion_date?: string;
  location?: string;
  branch_id?: string;
  allocations?: unknown[] | null;
}): string[] {
  const missing: string[] = [];
  if (!job.invoicing_information) missing.push("an invoicing profile");
  if (!job.allocations?.length) missing.push("division hours");
  if (job.status === "Active" && !job.project_completion_date)
    missing.push("a project completion date");
  if (!job.location) missing.push("a location");
  if (!job.branch_id) missing.push("a branch");
  return missing;
}
