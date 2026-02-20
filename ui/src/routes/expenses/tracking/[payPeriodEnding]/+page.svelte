<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { shortDate } from "$lib/utilities";
  import Icon from "@iconify/svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import { page } from "$app/stores";

  const payPeriodEnding = $derived.by(() => $page.params.payPeriodEnding);
  let rows = $state([] as any[]);

  async function init() {
    try {
      rows = await pb.send(`/api/expenses/tracking/${payPeriodEnding}`, { method: "GET" });
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load expenses");
    }
  }

  $effect(() => {
    if (payPeriodEnding) {
      init();
    }
  });
</script>

<DsList items={rows} groupField="phase" inListHeader={`Expenses for ${payPeriodEnding}`}>
  {#snippet groupHeader(label)}
    <span class="text-xs tracking-wide text-neutral-600 uppercase">{label}</span>
  {/snippet}
  {#snippet anchor(r)}
    <a href={`/expenses/${r.id}/details`} class="text-blue-600 hover:underline">{r.date}</a>
  {/snippet}
  {#snippet headline(r)}
    <span>{r.description}</span>
  {/snippet}
  {#snippet byline(r)}
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
          {#if r.vendor_alias}
            <span class="text-xs text-gray-500">({r.vendor_alias})</span>
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
  {#snippet line1(r)}
    <span>
      {r.given_name}
      {r.surname} / {r.division_code}
      {r.division_name}
    </span>
  {/snippet}
  {#snippet line2(r)}
    {#if r.job_number !== ""}
      <span class="flex items-center gap-1">
        {r.job_number} - {r.client_name}:
        {r.job_description}
        {#if r.category !== ""}
          <DsLabel color="teal">{r.category_name}</DsLabel>
        {/if}
      </span>
    {/if}
  {/snippet}
  {#snippet line3(r)}
    <span class="flex items-center gap-1 text-sm">
      {#if r.approved !== ""}
        <Icon icon="material-symbols:order-approve-outline" width="20px" class="inline-block" />
        {r.approver_name}
        ({shortDate(r.approved.split("T")[0], true)})
      {/if}
      {#if r.attachment}
        <a
          href={`${PUBLIC_POCKETBASE_URL}/api/files/expenses/${r.id}/${r.attachment}`}
          target="_blank"
        >
          <DsFileLink filename={r.attachment as string} />
        </a>
      {/if}
    </span>
  {/snippet}
</DsList>
