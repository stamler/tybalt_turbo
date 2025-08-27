<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  let { data } = $props();
</script>

{#if data.item}
  <div class="mx-auto space-y-6 p-4">
    <div class="flex items-center gap-2">
      <h1 class="text-2xl font-bold">
        {#if data.item.surname || data.item.given_name}
          {data.item.surname}{#if data.item.given_name},
          {/if}{data.item.given_name}
        {:else}
          {data.item.job_title || "-"}
        {/if}
      </h1>
      <DsActionButton
        action={`/admin_profiles/${data.item.id}/edit`}
        icon="mdi:pencil"
        title="Edit"
        color="blue"
      />
    </div>

    <section class="grid grid-cols-1 gap-2 md:grid-cols-2">
      <div class="flex gap-2">
        <span class="font-semibold">Payroll ID:</span>
        {data.item.payroll_id || "—"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Job Title:</span>
        {data.item.job_title || "—"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Mobile Phone:</span>
        {data.item.mobile_phone || "—"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Work Week Hours:</span>
        {data.item.work_week_hours}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Default Charge Out Rate:</span>
        {data.item.default_charge_out_rate}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Salary:</span>
        {data.item.salary ? "Yes" : "No"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Off Rotation Permitted:</span>
        {data.item.off_rotation_permitted ? "Yes" : "No"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Untracked Time Off:</span>
        {data.item.untracked_time_off ? "Yes" : "No"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Time Sheet Expected:</span>
        {data.item.time_sheet_expected ? "Yes" : "No"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Allow Personal Reimbursement:</span>
        {data.item.allow_personal_reimbursement ? "Yes" : "No"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Skip Min Time Check:</span>
        {data.item.skip_min_time_check}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Opening Date:</span>
        {data.item.opening_date || "—"}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Opening OP:</span>
        {data.item.opening_op}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Opening OV:</span>
        {data.item.opening_ov}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">Personal Vehicle Insurance Expiry:</span>
        {data.item.personal_vehicle_insurance_expiry || "—"}
      </div>
    </section>
  </div>
{:else}
  <div class="p-4">Not found.</div>
{/if}
