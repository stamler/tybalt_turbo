<script lang="ts">
  import type { ExpensesListData } from "$lib/svelte-types";
  import ExpensesList from "$lib/components/ExpensesList.svelte";
  let { data }: { data: ExpensesListData } = $props();
</script>

<ExpensesList inListHeader="My Expenses" {data} endpoint="/api/expenses/list" />
