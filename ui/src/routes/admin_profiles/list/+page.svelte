<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import Icon from "@iconify/svelte";
  import type { AdminProfilesAugmentedResponse } from "$lib/pocketbase-types";
  let { data } = $props();
  const items = $derived(data.items as AdminProfilesAugmentedResponse[]);

  function normalizeDivisions(value: null | string[] | undefined): string[] {
    return Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : [];
  }

  function poApproverLabel(item: AdminProfilesAugmentedResponse): string | null {
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
</script>

<DsList {items} search={true} inListHeader="Staff">
  {#snippet searchBarExtra()}
    <a
      href="/admin_profiles/po_approvers"
      class="flex items-center gap-1 rounded-sm bg-neutral-200 px-3 py-1 text-sm text-gray-700 hover:bg-neutral-300"
    >
      <Icon icon="mdi:edit-outline" class="text-base" /> PO Approvers
    </a>
  {/snippet}

  {#snippet anchor(item: AdminProfilesAugmentedResponse)}
    <div class="flex flex-col">
      <a
        href={`/admin_profiles/${item.id}/details`}
        class={item.active === false
          ? "text-blue-400 hover:underline"
          : "text-blue-600 hover:underline"}
      >
        {item.given_name}
        {item.surname}
      </a>
      {#if item.active === false}
        <span class="self-center text-xs font-medium text-red-500">Inactive</span>
      {/if}
    </div>
  {/snippet}

  {#snippet headline(item: AdminProfilesAugmentedResponse)}
    <span class={`flex items-center gap-2 ${item.active === false ? "opacity-50" : ""}`}>
      {item.job_title || "-"}
    </span>
  {/snippet}

  {#snippet byline(item: AdminProfilesAugmentedResponse)}
    <span class={item.active === false ? "opacity-30" : "opacity-60"}>
      {#if item.mobile_phone && item.mobile_phone.trim() !== ""}
        Mobile: {item.mobile_phone}
        {#if item.payroll_id && item.payroll_id.trim() !== ""}
          •
        {/if}
      {/if}
      Payroll: {item.payroll_id || "—"}
    </span>
  {/snippet}

  {#snippet line1(item: AdminProfilesAugmentedResponse)}
    <span class={`flex items-center gap-2 ${item.active === false ? "opacity-50" : ""}`}>
      <span>{item.salary ? "Salary" : "Hourly"}</span>
      <span>Charge Out Rate: {item.default_charge_out_rate}</span>
      {#if poApproverLabel(item)}
        <DsLabel color="purple">{poApproverLabel(item)}</DsLabel>
      {/if}
    </span>
  {/snippet}

  {#snippet actions(item: AdminProfilesAugmentedResponse)}
    <DsActionButton
      action={`/admin_profiles/${item.id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
  {/snippet}
</DsList>
