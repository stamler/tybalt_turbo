<script lang="ts">
  import type { ExpensesPageData } from "$lib/svelte-types";
  import ExpensesEditor from "$lib/components/ExpensesEditor.svelte";
  let { data }: { data: ExpensesPageData } = $props();
</script>

<ExpensesEditor {data} />
