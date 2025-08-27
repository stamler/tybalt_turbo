<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsList from "$lib/components/DSList.svelte";
  import type { AdminProfilesResponse } from "$lib/pocketbase-types";
  let { data } = $props();
  type AdminProfilesAugmented = AdminProfilesResponse & { given_name: string; surname: string };
  const items = $derived(data.items as AdminProfilesAugmented[]);
</script>

<DsList {items} search={true} inListHeader="Staff">
  {#snippet anchor(item: AdminProfilesAugmented)}
    <a href={`/admin_profiles/${item.id}/details`} class="text-blue-600 hover:underline">
      {item.surname}, {item.given_name}
    </a>
  {/snippet}

  {#snippet headline(item: AdminProfilesAugmented)}
    <span class="flex items-center gap-2">
      {item.job_title || "-"}
    </span>
  {/snippet}

  {#snippet byline(item: AdminProfilesAugmented)}
    <span class="opacity-60">Payroll: {item.payroll_id || "—"}</span>
  {/snippet}

  {#snippet line1(item: AdminProfilesAugmented)}
    <span class="flex gap-2">
      <span>{item.salary ? "Salary" : "Hourly"}</span>
      <span>Charge Out Rate: {item.default_charge_out_rate}</span>
    </span>
  {/snippet}

  {#snippet actions(item: AdminProfilesAugmented)}
    <DsActionButton
      action={`/admin_profiles/${item.id}/edit`}
      icon="mdi:edit-outline"
      title="Edit"
      color="blue"
    />
  {/snippet}
</DsList>
