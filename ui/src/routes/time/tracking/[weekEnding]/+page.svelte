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
  import DSTabBar, { type TabItem } from "$lib/components/DSTabBar.svelte";
  import { onMount } from "svelte";

  const weekEnding = $derived.by(() => $page.params.weekEnding);
  let rows = $state([] as any[]);
  let missing = $state([] as any[]);
  let notExpected = $state([] as any[]);
  let rejectModal: SvelteComponent;
  let isLoading = $state(false);

  // Tabs ----------------------------------------------------------------------
  let activeTab = $state<"sheets" | "missing" | "not_expected">("sheets");
  let tabs: TabItem[] = $derived([
    {
      label: `Sheets (${rows.length})`,
      href: "#sheets",
      active: activeTab === "sheets",
    },
    {
      label: `Missing (${missing.length})`,
      href: "#missing",
      active: activeTab === "missing",
    },
    {
      label: `Not Expected (${notExpected.length})`,
      href: "#not_expected",
      active: activeTab === "not_expected",
    },
  ]);

  async function init() {
    if (isLoading) return;
    isLoading = true;
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
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    if (weekEnding) {
      init();
    }
  });

  // Hash sync + persistence
  function setTabFromHash() {
    const hash = typeof window !== "undefined" ? window.location.hash : "";
    if (hash === "#missing") activeTab = "missing";
    else if (hash === "#not_expected") activeTab = "not_expected";
    else activeTab = "sheets";
    try {
      if (typeof window !== "undefined") {
        window.localStorage.setItem("tt_active_tab", activeTab);
      }
    } catch {}
  }
  onMount(() => {
    // restore last tab if no hash, else honor hash
    if (typeof window !== "undefined") {
      if (!window.location.hash) {
        try {
          const saved = window.localStorage.getItem("tt_active_tab") as typeof activeTab | null;
          if (saved) {
            activeTab = saved;
            window.location.hash = `#${saved}`;
          }
        } catch {}
      } else {
        setTabFromHash();
      }
      const handler = () => setTabFromHash();
      window.addEventListener("hashchange", handler);
      // initial data load after mount (also covered by $effect when weekEnding changes)
      init();
      return () => {
        window.removeEventListener("hashchange", handler);
      };
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

  // Utilities for Missing / Not Expected --------------------------------------
  function personName(p: any) {
    const gn = p?.given_name || "";
    const sn = p?.surname || "";
    return `${sn}${sn && gn ? ", " : ""}${gn}`.trim();
  }
  function emailOne(email: string) {
    if (!email) return;
    if (typeof window !== "undefined") {
      window.location.href = `mailto:${encodeURIComponent(email)}`;
    }
  }
  async function copy(text: string) {
    try {
      await navigator.clipboard.writeText(text);
      // small UX touch; keep silent to avoid noise
    } catch {}
  }
</script>

<RejectModal collectionName="time_sheets" bind:this={rejectModal} on:refresh={() => init()} />

<DSTabBar {tabs} />

<!-- Sheets Tab -->
<div class:hidden={activeTab !== "sheets"}>
  <DsList
    search={true}
    items={rows}
    groupField="phase"
    inListHeader={`Time Tracking for ${weekEnding}`}
  >
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
        <span class="text-sm text-gray-500"
          >Pending approval by {r.approver_name || r.approver}</span
        >
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
</div>

<!-- Missing Tab -->
<div class:hidden={activeTab !== "missing"} class="space-y-2">
  <DsList search={true} items={missing} inListHeader="Missing">
    {#snippet headline(m)}
      <span class="hover:underline">{personName(m) || m.email}</span>
    {/snippet}
    {#snippet actions(m)}
      <DsActionButton
        icon="mdi:email"
        color="blue"
        title="Email"
        action={() => emailOne(m.email)}
      />
      <DsActionButton
        icon="mdi:content-copy"
        color="gray"
        title="Copy email"
        action={() => copy(m.email)}
      />
    {/snippet}
  </DsList>
</div>

<!-- Not Expected Tab -->
<div class:hidden={activeTab !== "not_expected"} class="space-y-2">
  <DsList search={true} items={notExpected} inListHeader="Not Expected">
    {#snippet headline(n)}
      <span class="hover:underline">{personName(n) || n.email}</span>
    {/snippet}
  </DsList>
</div>
