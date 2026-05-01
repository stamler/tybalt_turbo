<script lang="ts">
  import { invalidateAll } from "$app/navigation";
  import { resolve } from "$app/paths";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import Icon from "@iconify/svelte";
  import type { AdminProfilesAugmentedResponse } from "$lib/pocketbase-types";
  let { data } = $props();
  type AdminProfileListItem = AdminProfilesAugmentedResponse & {
    legacy_uid?: string;
    provider_count?: number;
    email?: string;
    name?: string;
    can_toggle_active?: boolean;
  };
  const items = $derived(data.items as AdminProfileListItem[]);
  let activeStatus = $state<"active" | "inactive">("active");
  const activeStatusOptions = [
    { id: "active", label: "Active" },
    { id: "inactive", label: "Inactive" },
  ];
  const filteredItems = $derived(
    items.filter((item) => (activeStatus === "active" ? item.active !== false : item.active === false)),
  );
  const isAdmin = $derived($globalStore.claims.includes("admin"));
  const hasHrClaim = $derived($globalStore.claims.includes("hr"));
  const hasTimeOffManagerClaim = $derived($globalStore.claims.includes("time_off_manager"));
  const hasItClaim = $derived($globalStore.claims.includes("it"));
  const canViewDetails = $derived(isAdmin || hasHrClaim || hasTimeOffManagerClaim);
  const canSetActive = $derived(isAdmin || hasItClaim || hasHrClaim);

  function normalizeDivisions(value: null | string[] | undefined): string[] {
    return Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : [];
  }

  function poApproverLabel(item: AdminProfileListItem): string | null {
    const divisions = normalizeDivisions(item.po_approver_divisions);
    const hasProps =
      (typeof item.po_approver_props_id === "string" && item.po_approver_props_id.trim() !== "") ||
      typeof item.po_approver_max_amount === "number" ||
      divisions.length > 0;
    if (!hasProps) return null;
    const divisionsPart =
      divisions.length === 0
        ? "All divisions"
        : `${divisions.length} division${divisions.length === 1 ? "" : "s"}`;
    return `po_approver • ${divisionsPart}`;
  }

  function displayName(item: AdminProfileListItem): string {
    const profileName = `${item.given_name ?? ""} ${item.surname ?? ""}`.trim();
    return profileName || item.name || item.email || item.uid;
  }

  function canLinkDetails(item: AdminProfileListItem): boolean {
    return canViewDetails && item.provider_count === undefined;
  }

  function canToggleActive(item: AdminProfileListItem): boolean {
    return canSetActive && item.can_toggle_active === true;
  }

  async function setActive(item: AdminProfileListItem, active: boolean): Promise<void> {
    await pb.send(`/api/admin_profiles/${item.id}/active`, {
      method: "POST",
      body: { active },
    });
    item.active = active;
    await invalidateAll();
  }
</script>

<DsList
  items={filteredItems}
  search={true}
  inListHeader={activeStatus === "active" ? "Active Staff" : "Inactive Staff"}
>
  {#snippet searchBarExtra()}
    <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
      <DSToggle
        bind:value={activeStatus}
        options={activeStatusOptions}
        ariaLabel="Admin profile active state filter"
      />
      {#if isAdmin}
        <a
          href={resolve("/admin_profiles/po_approvers")}
          class="flex items-center gap-1 rounded-sm bg-neutral-200 px-3 py-1 text-sm text-gray-700 hover:bg-neutral-300"
        >
          <Icon icon="mdi:edit-outline" class="text-base" /> PO Approvers
        </a>
      {/if}
    </div>
  {/snippet}

  {#snippet anchor(item: AdminProfileListItem)}
    <div class="flex flex-col">
      {#if canLinkDetails(item)}
        <a
          href={resolve(`/admin_profiles/${item.id}/details`)}
          class={item.active === false
            ? "text-blue-400 hover:underline"
            : "text-blue-600 hover:underline"}
        >
          {displayName(item)}
        </a>
      {:else}
        <span class={item.active === false ? "text-neutral-400" : ""}>{displayName(item)}</span>
      {/if}
      {#if item.active === false}
        <span class="self-center text-xs font-medium text-red-500">Inactive</span>
      {/if}
    </div>
  {/snippet}

  {#snippet headline(item: AdminProfileListItem)}
    <span class={`flex items-center gap-2 ${item.active === false ? "opacity-50" : ""}`}>
      {item.job_title || (hasItClaim ? item.email || "-" : "-")}
    </span>
  {/snippet}

  {#snippet byline(item: AdminProfileListItem)}
    <span class={item.active === false ? "opacity-30" : "opacity-60"}>
      {#if hasItClaim && !canViewDetails}
        Legacy UID: {item.legacy_uid || "—"} • Providers: {item.provider_count ?? 0}
      {:else if item.mobile_phone && item.mobile_phone.trim() !== ""}
        Mobile: {item.mobile_phone}
        {#if item.payroll_id && item.payroll_id.trim() !== ""}
          •
        {/if}
        Payroll: {item.payroll_id || "—"}
      {:else}
        Payroll: {item.payroll_id || "—"}
      {/if}
    </span>
  {/snippet}

  {#snippet line1(item: AdminProfileListItem)}
    <span class={`flex items-center gap-2 ${item.active === false ? "opacity-50" : ""}`}>
      {#if hasItClaim && !canViewDetails}
        <span>UID: {item.uid}</span>
      {:else}
        <span>{item.salary ? "Salary" : "Hourly"}</span>
        <span>Charge Out Rate: {item.default_charge_out_rate}</span>
        {#if isAdmin && poApproverLabel(item)}
          <DsLabel color="purple">{poApproverLabel(item)}</DsLabel>
        {/if}
      {/if}
    </span>
  {/snippet}

  {#snippet actions(item: AdminProfileListItem)}
    <div class="flex items-center gap-1">
      {#if canToggleActive(item)}
        {#if item.active === false}
          <DsActionButton
            action={() => setActive(item, true)}
            icon="mdi:account-check-outline"
            title="Activate"
            color="green"
          />
        {:else}
          <DsActionButton
            action={() => setActive(item, false)}
            icon="mdi:account-cancel-outline"
            title="Deactivate"
            color="red"
          />
        {/if}
      {/if}
      <DsActionButton
        action={`/admin_profiles/${item.id}/edit`}
        icon="mdi:edit-outline"
        title="Edit"
        color="blue"
      />
    </div>
  {/snippet}
</DsList>
