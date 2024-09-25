<script lang="ts">
  import type { PageData } from "./$types";
  import ExpensesEditor from "$lib/components/ExpensesEditor.svelte";
  let { data }: { data: PageData } = $props();
</script>

<ExpensesEditor {data} />
