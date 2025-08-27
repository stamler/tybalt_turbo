<script lang="ts">
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsCheck from "$lib/components/DsCheck.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { flatpickrAction } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";
  import type { AdminProfilesSkipMinTimeCheckOptions } from "$lib/pocketbase-types";
  import type { AdminProfilesPageData } from "$lib/svelte-types";
  import { goto } from "$app/navigation";

  let { data }: { data: AdminProfilesPageData } = $props();

  let errors = $state({} as Record<string, { message: string }>);
  let item = $state({ ...data.item });

  async function save(event: Event) {
    event.preventDefault();
    try {
      if (data.editing && data.id) {
        await pb.collection("admin_profiles").update(data.id, item);
      } else {
        await pb.collection("admin_profiles").create(item);
      }
      errors = {};
      goto("/admin_profiles/list");
    } catch (error: any) {
      errors = error?.data?.data || {};
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
  <style>
    form {
      max-width: 900px;
    }
  </style>
</svelte:head>

<form class="flex w-full flex-col items-center gap-2 p-2" onsubmit={save}>
  <div class="grid w-full grid-cols-1 gap-2 md:grid-cols-2">
    <DsTextInput
      bind:value={item.work_week_hours as number}
      {errors}
      fieldName="work_week_hours"
      uiName="Work Week Hours"
      type="number"
      step={1}
      min={0}
    />

    <DsCheck bind:value={item.salary as boolean} {errors} fieldName="salary" uiName="Salary" />
    <DsCheck
      bind:value={item.off_rotation_permitted as boolean}
      {errors}
      fieldName="off_rotation_permitted"
      uiName="Off Rotation Permitted"
    />
    <DsCheck
      bind:value={item.untracked_time_off as boolean}
      {errors}
      fieldName="untracked_time_off"
      uiName="Untracked Time Off"
    />
    <DsCheck
      bind:value={item.time_sheet_expected as boolean}
      {errors}
      fieldName="time_sheet_expected"
      uiName="Time Sheet Expected"
    />
    <DsCheck
      bind:value={item.allow_personal_reimbursement as boolean}
      {errors}
      fieldName="allow_personal_reimbursement"
      uiName="Allow Personal Reimbursement"
    />

    <DsTextInput
      bind:value={item.default_charge_out_rate as number}
      {errors}
      fieldName="default_charge_out_rate"
      uiName="Default Charge Out Rate"
      type="number"
      step={0.01}
      min={0}
    />

    <DsSelector
      bind:value={item.skip_min_time_check as AdminProfilesSkipMinTimeCheckOptions}
      items={[
        { id: "no", name: "No" },
        { id: "on_next_bundle", name: "On Next Bundle" },
        { id: "yes", name: "Yes" },
      ]}
      {errors}
      fieldName="skip_min_time_check"
      uiName="Skip Min Time Check"
    >
      {#snippet optionTemplate(item)}{item.name}{/snippet}
    </DsSelector>

    <span class="flex w-full gap-2 {errors.opening_date !== undefined ? 'bg-red-200' : ''}">
      <label for="opening_date">Opening Date</label>
      <input
        class="flex-1"
        type="text"
        name="opening_date"
        placeholder="Opening Date"
        use:flatpickrAction
        bind:value={item.opening_date}
      />
      {#if errors.opening_date !== undefined}
        <span class="text-red-600">{errors.opening_date.message}</span>
      {/if}
    </span>

    <span
      class="flex w-full gap-2 {errors.personal_vehicle_insurance_expiry !== undefined
        ? 'bg-red-200'
        : ''}"
    >
      <label for="personal_vehicle_insurance_expiry">Personal Vehicle Insurance Expiry</label>
      <input
        class="flex-1"
        type="text"
        name="personal_vehicle_insurance_expiry"
        placeholder="Insurance Expiry"
        use:flatpickrAction
        bind:value={item.personal_vehicle_insurance_expiry}
      />
      {#if errors.personal_vehicle_insurance_expiry !== undefined}
        <span class="text-red-600">{errors.personal_vehicle_insurance_expiry.message}</span>
      {/if}
    </span>

    <DsTextInput
      bind:value={item.opening_op as number}
      {errors}
      fieldName="opening_op"
      uiName="Opening OP"
      type="number"
      step={0.1}
      min={0}
    />
    <DsTextInput
      bind:value={item.opening_ov as number}
      {errors}
      fieldName="opening_ov"
      uiName="Opening OV"
      type="number"
      step={0.01}
      min={0}
    />

    <DsTextInput
      bind:value={item.payroll_id as string}
      {errors}
      fieldName="payroll_id"
      uiName="Payroll ID"
    />
    <DsTextInput
      bind:value={item.mobile_phone as string}
      {errors}
      fieldName="mobile_phone"
      uiName="Mobile Phone"
    />
    <DsTextInput
      bind:value={item.job_title as string}
      {errors}
      fieldName="job_title"
      uiName="Job Title"
    />
  </div>

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/admin_profiles/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
