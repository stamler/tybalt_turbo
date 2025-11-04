<script lang="ts">
  import type { ExpensesListData } from "$lib/svelte-types";
  import ExpensesList from "$lib/components/ExpensesList.svelte";
  let { data }: { data: ExpensesListData } = $props();
</script>

<ExpensesList inListHeader="Expenses Pending My Approval" {data} endpoint="/api/expenses/pending" />
