<script lang="ts">
  let {
    includeExpenses = $bindable(true),
    includePOs = $bindable(true),
  }: { includeExpenses?: boolean; includePOs?: boolean } = $props();
</script>

<div class="flex flex-wrap gap-x-6 gap-y-3">
  <label class="flex items-center gap-2"
    ><input type="checkbox" bind:checked={includeExpenses} />Include committed expenses</label
  >
  <label class="flex items-center gap-2"
    ><input type="checkbox" bind:checked={includePOs} />Include active POs</label
  >
</div>
