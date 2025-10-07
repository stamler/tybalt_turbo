<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";

  let { data } = $props();

  const currency = new Intl.NumberFormat("en-CA", {
    style: "currency",
    currency: "CAD",
    minimumFractionDigits: 2,
  });

  function divisionLabel(id: string): string {
    const division = data.poApproverDivisions?.get?.(id);
    if (!division) return id;
    const code = division.code?.trim();
    const name = division.name?.trim() ?? id;
    return code ? `${code} — ${name}` : name;
  }
</script>

{#if data.item}
  <div class="mx-auto space-y-6 p-4">
    <div class="flex items-center gap-2">
      <h1 class="text-2xl font-bold">
        {#if data.item.surname || data.item.given_name}
          {data.item.given_name} {data.item.surname}
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
        <span class="font-semibold">Default Branch:</span>
        {data.defaultBranch?.name || "—"}
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

    <!-- Claims, styled like the profile page -->
    <section class="space-y-2">
      <h2 class="text-lg font-semibold">Claims</h2>
      {#if data.claims && data.claims.length > 0}
        <ul class="flex flex-row flex-wrap gap-2">
          {#each data.claims as claim (claim.id)}
            <li>
              <DsLabel color={claim.name === "po_approver" ? "purple" : "cyan"}
                >{claim.name === "po_approver"
                  ? data.poApproverProps
                    ? `po_approver • ${currency.format(data.poApproverProps.max_amount ?? 0)} • ${
                        (data.poApproverProps.divisions ?? []).length === 0
                          ? "All divisions"
                          : `${(data.poApproverProps.divisions ?? []).length} division${
                              (data.poApproverProps.divisions ?? []).length === 1 ? "" : "s"
                            }`
                      }`
                    : "po_approver"
                  : claim.name}</DsLabel
              >
            </li>
          {/each}
        </ul>
      {:else}
        <p class="text-sm text-neutral-500">No claims assigned.</p>
      {/if}
    </section>
  </div>
{:else}
  <div class="p-4">Not found.</div>
{/if}
