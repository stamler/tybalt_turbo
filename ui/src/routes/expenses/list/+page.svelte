<script lang="ts">
  import type { ExpensesPageData } from "$lib/svelte-types";
  import ExpensesList from "$lib/components/ExpensesList.svelte";
  let { data }: { data: ExpensesPageData } = $props();
</script>

<ExpensesList inListHeader="My Expenses" {data} />
