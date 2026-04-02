<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { pocketBaseFileHref, shortDate, trimmedOrEmpty } from "$lib/utilities";
  import Icon from "@iconify/svelte";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import { page } from "$app/stores";

  const committedWeekEnding = $derived.by(
    () => ($page.params as Record<string, string | undefined>).committedWeekEnding,
  );

  type Row = {
    id: string;
    uid: string;
    given_name: string;
    surname: string;
    attachment: string;
    date: string;
    [key: string]: any;
  };
  type GroupedRow = Row & { employee_group: string };

  let rows = $state([] as Row[]);

  function parseIsoDate(dateString: string) {
    return new Date(`${dateString}T12:00:00`);
  }

  const weekStart = $derived.by(() => {
    if (!committedWeekEnding) return "";
    const date = parseIsoDate(committedWeekEnding);
    date.setDate(date.getDate() - 6);
    return date.toISOString().slice(0, 10);
  });

  const attachmentCount = $derived.by(
    () => rows.filter((row) => typeof row.attachment === "string" && row.attachment !== "").length,
  );

  const groupedRows = $derived.by((): GroupedRow[] => {
    const counts = new Map<string, number>();

    for (const row of rows) {
      const key = row.uid || `${row.surname}, ${row.given_name}`;
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }

    return rows.map((row) => {
      const key = row.uid || `${row.surname}, ${row.given_name}`;
      const displayName = `${row.given_name} ${row.surname}`.trim();
      return {
        ...row,
        employee_group: `${displayName} (${counts.get(key) ?? 0})`,
      };
    });
  });

  async function init() {
    try {
      rows = await pb.send(`/api/expenses/tracking/${committedWeekEnding}`, { method: "GET" });
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load expenses");
    }
  }

  $effect(() => {
    if (committedWeekEnding) {
      init();
    }
  });
</script>

<div class="mb-3 rounded-md border border-neutral-300 bg-neutral-50 px-4 py-3">
  <div class="text-sm text-neutral-700">
    {shortDate(weekStart || committedWeekEnding || "", true)} to
    {shortDate(committedWeekEnding || "", true)} ({rows.length}, {attachmentCount} with attachments)
  </div>
</div>

<DsList
  items={groupedRows}
  groupField="employee_group"
  inListHeader={`Committed Expenses for ${shortDate(committedWeekEnding || "", true)}`}
>
  {#snippet anchor(r: GroupedRow)}
    <a href={`/expenses/${r.id}/details`} class="text-blue-600 hover:underline">{r.date}</a>
  {/snippet}
  {#snippet headline(r: GroupedRow)}
    <span>{r.description}</span>
  {/snippet}
  {#snippet byline(r: GroupedRow)}
    <span class="flex items-center gap-2 text-sm">
      {#if r.rejected !== ""}
        <DsLabel
          color="red"
          title={`${shortDate(r.rejected.split("T")[0], true)}: ${r.rejection_reason}`}
        >
          <Icon icon="mdi:cancel" width="20px" class="inline-block" />
          {r.rejector_name}
        </DsLabel>
      {/if}

      {#if r.payment_type === "Mileage"}
        {r.distance} km / ${r.total?.toFixed(2)}
      {:else}
        ${r.total?.toFixed(2)}
      {/if}
      {#if r.vendor}
        <span class="flex items-center gap-0">
          <Icon icon="mdi:store" width="20px" class="inline-block" />
          {r.vendor_name}
          {#if trimmedOrEmpty(r.vendor_alias)}
            <span class="text-xs text-gray-500">({trimmedOrEmpty(r.vendor_alias)})</span>
          {/if}
        </span>
      {/if}
      {#if r.payment_type === "CorporateCreditCard"}
        <DsLabel color="cyan">
          <Icon icon="mdi:credit-card-outline" width="20px" class="inline-block" />
          **** {r.cc_last_4_digits}
        </DsLabel>
      {/if}
    </span>
  {/snippet}
  {#snippet line1(r: GroupedRow)}
    <span>
      {r.given_name}
      {r.surname} / {r.division_code}
      {r.division_name}
    </span>
  {/snippet}
  {#snippet line2(r: GroupedRow)}
    {#if r.job_number !== ""}
      <span class="flex items-center gap-1">
        {r.job_number} - {r.client_name}:
        {r.job_description}
        {#if r.category !== "" && r.category_name}
          <DsLabel color="teal">{r.category_name}</DsLabel>
        {/if}
      </span>
    {/if}
  {/snippet}
  {#snippet line3(r: GroupedRow)}
    <span class="flex items-center gap-1 text-sm">
      {#if r.approved !== ""}
        <Icon icon="material-symbols:order-approve-outline" width="20px" class="inline-block" />
        {r.approver_name}
        ({shortDate(r.approved.split("T")[0], true)})
      {/if}
      {#if r.attachment}
        <a href={pocketBaseFileHref("expenses", r.id, r.attachment)} target="_blank">
          <DsFileLink filename={r.attachment as string} />
        </a>
      {/if}
    </span>
  {/snippet}
</DsList>
