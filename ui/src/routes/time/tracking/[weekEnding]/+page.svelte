<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import RejectModal from "$lib/components/RejectModal.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { shortDate } from "$lib/utilities";
  import type { SvelteComponent } from "svelte";
  import { page } from "$app/stores";

  const weekEnding = $derived.by(() => $page.params.weekEnding);
  let rows = $state([] as any[]);
  let missing = $state([] as any[]);
  let notExpected = $state([] as any[]);
  let rejectModal: SvelteComponent;

  async function init() {
    try {
      const [listRes, missingRes, notExpectedRes] = await Promise.all([
        pb.send(`/api/time_sheets/tracking/weeks/${weekEnding}`, { method: "GET" }),
        pb.send(`/api/time_sheets/tracking/weeks/${weekEnding}/missing`, { method: "GET" }),
        pb.send(`/api/time_sheets/tracking/weeks/${weekEnding}/not_expected`, { method: "GET" }),
      ]);
      rows = listRes;
      missing = missingRes;
      notExpected = notExpectedRes;
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Failed to load timesheets");
    }
  }

  $effect(() => {
    if (weekEnding) {
      init();
    }
  });

  async function commit(id: string) {
    try {
      await pb.send(`/api/time_sheets/${id}/commit`, { method: "POST" });
      await init();
    } catch (error: any) {
      globalStore.addError(error?.response?.error || "Commit failed");
    }
  }

  function openReject(id: string) {
    // @ts-ignore exported function on the component instance
    rejectModal.openModal(id);
  }
</script>

<RejectModal collectionName="time_sheets" bind:this={rejectModal} on:refresh={() => init()} />

<details class="mb-4">
  <summary class="cursor-pointer font-semibold text-neutral-700">Missing</summary>
  <div class="mt-2">
    {#if missing.length === 0}
      <div class="text-sm text-neutral-500">None</div>
    {:else}
      <ul class="list-disc pl-5">
        {#each missing as m}
          <li>
            <a href={`mailto:${m.email}`} class="text-blue-600 hover:underline">
              {m.surname || m.given_name ? `${m.surname}, ${m.given_name}` : m.email}
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</details>

<details class="mb-4">
  <summary class="cursor-pointer font-semibold text-neutral-700">Not Expected</summary>
  <div class="mt-2">
    {#if notExpected.length === 0}
      <div class="text-sm text-neutral-500">None</div>
    {:else}
      <ul class="list-disc pl-5">
        {#each notExpected as n}
          <li>
            <a href={`mailto:${n.email}`} class="text-blue-600 hover:underline">
              {n.surname || n.given_name ? `${n.surname}, ${n.given_name}` : n.email}
            </a>
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</details>

<DsList items={rows} groupField="phase" inListHeader={`Time Tracking for ${weekEnding}`}>
  {#snippet groupHeader(label)}
    <span class="text-xs uppercase tracking-wide text-neutral-600">{label}</span>
  {/snippet}
  {#snippet headline(r)}
    <a href={`/time/sheets/${r.id}/details`} class="underline">{r.surname}, {r.given_name}</a>
    {#if r.rejected !== ""}
      <DsLabel
        color="red"
        title={`Rejected on ${shortDate(r.rejected.split("T")[0], true)}${r.rejector_name ? ` by ${r.rejector_name}` : ""}`}
        >Rejected</DsLabel
      >
    {/if}
  {/snippet}
  {#snippet line1(r)}
    {#if r.submitted && r.approved !== ""}
      <span class="text-sm text-gray-500"
        >Approved by {r.approver_name || r.approver} on {shortDate(
          r.approved.split("T")[0],
          true,
        )}</span
      >
    {:else if r.submitted && r.approved === ""}
      <span class="text-sm text-gray-500">Pending approval by {r.approver_name || r.approver}</span>
    {/if}
  {/snippet}
  {#snippet line2(r)}
    {#if r.committed !== ""}
      <span class="text-sm text-gray-500"
        >Committed by {r.committer_name || r.committer} on {shortDate(
          r.committed.split("T")[0],
          true,
        )}</span
      >
    {/if}
  {/snippet}
  {#snippet line3(r)}
    {@const segments = [
      r.total_hours_worked > 0 ? `Hours: ${r.total_hours_worked}` : "",
      r.total_stat > 0 ? `Stat: ${r.total_stat}` : "",
      r.total_ppto > 0 ? `PPTO: ${r.total_ppto}` : "",
      r.total_vacation > 0 ? `Vac: ${r.total_vacation}` : "",
      r.total_sick > 0 ? `Sick: ${r.total_sick}` : "",
      r.total_to_bank > 0 ? `Bank: ${r.total_to_bank}` : "",
      r.total_ot_payout_request > 0 ? `OT Payout: ${r.total_ot_payout_request}` : "",
      r.total_bereavement > 0 ? `Bereavement: ${r.total_bereavement}` : "",
      r.total_days_off_rotation > 0 ? `Off Rotation Days: ${r.total_days_off_rotation}` : "",
    ].filter(Boolean)}
    {#if segments.length}
      <div class="text-xs text-neutral-600">
        {#each segments as seg, i}
          {#if i > 0}
            ·
          {/if}{seg}
        {/each}
      </div>
    {/if}
  {/snippet}
  {#snippet actions(r)}
    {#if r.phase === "Approved" && r.rejected === ""}
      <DsActionButton action={() => commit(r.id)}>Commit</DsActionButton>
    {/if}
    {#if r.phase !== "Committed" && r.rejected === ""}
      <DsActionButton action={() => openReject(r.id)}>Reject</DsActionButton>
    {/if}
  {/snippet}
</DsList>
