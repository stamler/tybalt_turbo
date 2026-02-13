<script lang="ts">
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";

  let { data } = $props();

  const currency = new Intl.NumberFormat("en-CA", {
    style: "currency",
    currency: "CAD",
    minimumFractionDigits: 2,
  });

  function normalizeNumber(value: number | null | undefined): number {
    if (typeof value === "number") return Number.isFinite(value) ? value : 0;
    return 0;
  }

  function normalizeDivisions(value: null | string[] | undefined): string[] {
    return Array.isArray(value) ? value.filter((id): id is string => typeof id === "string") : [];
  }

  function hasPoApproverDetails(): boolean {
    if (!data.item) return false;
    const hasId =
      typeof data.item.po_approver_props_id === "string" && data.item.po_approver_props_id.trim() !== "";
    const divisions = normalizeDivisions(data.item.po_approver_divisions);
    return hasId || typeof data.item.po_approver_max_amount === "number" || divisions.length > 0;
  }

  function poApproverClaimLabel(): string {
    if (!data.item) return "po_approver";
    if (!hasPoApproverDetails()) return "po_approver";
    const divisions = normalizeDivisions(data.item.po_approver_divisions);
    if (divisions.length === 0) return "po_approver • All divisions";
    return `po_approver • ${divisions.length} division${divisions.length === 1 ? "" : "s"}`;
  }

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
      {#if data.item.active === false}
        <DsLabel color="red">Inactive</DsLabel>
      {/if}
      <DsActionButton
        action={`/admin_profiles/${data.item.id}/edit`}
        icon="mdi:pencil"
        title="Edit"
        color="blue"
      />
    </div>

    <section class="grid grid-cols-1 gap-2 md:grid-cols-2">
      <div class="flex gap-2">
        <span class="font-semibold">Active:</span>
        {data.item.active === false ? "No" : "Yes"}
      </div>
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
      <div class="flex gap-2">
        <span class="font-semibold">Record ID:</span>
        {data.item.id}
      </div>
      <div class="flex gap-2">
        <span class="font-semibold">UID:</span>
        {data.item.uid}
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
                  ? poApproverClaimLabel()
                  : claim.name}</DsLabel
              >
            </li>
          {/each}
        </ul>
      {:else}
        <p class="text-sm text-neutral-500">No claims assigned.</p>
      {/if}
    </section>

    {#if hasPoApproverDetails()}
      <section class="space-y-2">
        <h2 class="text-lg font-semibold">PO Approver Limits</h2>
        <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
          <div class="flex gap-2">
            <span class="font-semibold">Standard (No Job):</span>
            {currency.format(normalizeNumber(data.item.po_approver_max_amount))}
          </div>
          <div class="flex gap-2">
            <span class="font-semibold">Standard (With Job):</span>
            {currency.format(normalizeNumber(data.item.po_approver_project_max))}
          </div>
          <div class="flex gap-2">
            <span class="font-semibold">Sponsorship:</span>
            {currency.format(normalizeNumber(data.item.po_approver_sponsorship_max))}
          </div>
          <div class="flex gap-2">
            <span class="font-semibold">Staff and Social:</span>
            {currency.format(normalizeNumber(data.item.po_approver_staff_and_social_max))}
          </div>
          <div class="flex gap-2">
            <span class="font-semibold">Media and Event:</span>
            {currency.format(normalizeNumber(data.item.po_approver_media_and_event_max))}
          </div>
          <div class="flex gap-2">
            <span class="font-semibold">Computer:</span>
            {currency.format(normalizeNumber(data.item.po_approver_computer_max))}
          </div>
          <div class="flex gap-2">
            <span class="font-semibold">PO Approver Props ID:</span>
            {data.item.po_approver_props_id || "—"}
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          <span class="font-semibold">Divisions:</span>
          {#if normalizeDivisions(data.item.po_approver_divisions).length === 0}
            <span>All divisions</span>
          {:else}
            {#each normalizeDivisions(data.item.po_approver_divisions) as divisionId}
              <DsLabel color="purple">{divisionLabel(divisionId)}</DsLabel>
            {/each}
          {/if}
        </div>
      </section>
    {/if}
  </div>
{:else}
  <div class="p-4">Not found.</div>
{/if}
