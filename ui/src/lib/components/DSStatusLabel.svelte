<script lang="ts">
  import DsLabel, { type LabelTone } from "./DsLabel.svelte";

  // The one map from record status to tone. Expenses have no status field;
  // pass the display status the page derives from its timestamps.
  let { status, kind }: { status: string; kind?: "job" | "po" | "expense" } = $props();

  const tonesByKind: Record<"job" | "po" | "expense", Record<string, LabelTone>> = {
    job: {
      Active: "success",
      Awarded: "success",
      Submitted: "warning",
      "In Progress": "warning",
      "Not Awarded": "danger",
      Closed: "neutral",
      Cancelled: "neutral",
      "No Bid": "neutral",
    },
    po: {
      Active: "success",
      Unapproved: "warning",
      Rejected: "danger",
      Closed: "neutral",
      Cancelled: "neutral",
    },
    expense: {
      Approved: "success",
      Committed: "success",
      Submitted: "warning",
      Rejected: "danger",
    },
  };

  // Without a kind, look the status up in every map; no status has a
  // different tone in two of them.
  const allTones: Record<string, LabelTone> = Object.assign({}, ...Object.values(tonesByKind));

  const tone = $derived((kind ? tonesByKind[kind] : allTones)[status] ?? "neutral");
</script>

<DsLabel {tone}>{status}</DsLabel>
