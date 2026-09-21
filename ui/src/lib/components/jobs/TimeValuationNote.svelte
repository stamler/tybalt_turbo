<script lang="ts">
  import { resolve } from "$app/paths";

  let { rateSheet } = $props<{
    rateSheet?: { id: string; name: string; revision: number };
  }>();
</script>

<div class="mb-3 space-y-1 text-sm text-neutral-600">
  {#if rateSheet?.id}
    <p>
      Job rates come from the current rate sheet:
      <a
        class="text-blue-600 hover:underline"
        href={resolve("/rate-sheets/[id]/details", { id: rateSheet.id })}
      >
        {rateSheet.name} (rev. {rateSheet.revision})</a
      >.
    </p>
  {:else}
    <p>No rate sheet is assigned to this job. Employee default charge-out rates are used.</p>
  {/if}
  <p>
    Regular rates only. Includes committed and uncommitted time entries. Excludes meal hours and
    time amendments from values.
  </p>
  <p>
    Rate sheet means all priced hours use job rates. Estimated means some hours use employee default
    charge-out rates because no job rate matches. Incomplete means some hours have neither a
    matching job rate nor an employee default rate greater than zero.
  </p>
  <p>
    Known values include estimates and exclude unpriced hours. Percentages are unavailable when the
    total is incomplete or zero. Changes to job rates or employee defaults recalculate past values.
  </p>
</div>
