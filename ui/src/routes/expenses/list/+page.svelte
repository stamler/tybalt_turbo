<script lang="ts">
  import type { ExpensesListData } from "$lib/svelte-types";
  import ExpensesList from "$lib/components/ExpensesList.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import type { ExpensesAugmentedResponse } from "$lib/pocketbase-types";

  let { data }: { data: ExpensesListData } = $props();

  type ExpenseListMode = "not_committed" | "committed";

  let expenseListMode = $state<ExpenseListMode>("not_committed");

  const listModeOptions = [
    { id: "not_committed", label: "Uncommitted" },
    { id: "committed", label: "Committed" },
  ];

  const statusFilter = $derived((expense: ExpensesAugmentedResponse) => {
    if (expenseListMode === "committed") {
      return expense.committed !== "";
    }
    return expense.committed === "";
  });
</script>

<ExpensesList
  inListHeader={expenseListMode === "committed"
    ? "My Expenses - Committed"
    : "My Expenses - Uncommitted"}
  {data}
  endpoint="/api/expenses/list"
  filter={statusFilter}
  showLoadMore={expenseListMode === "committed"}
>
  {#snippet searchBarExtra()}
    <div class="flex items-center gap-2 max-[639px]:w-full max-[639px]:flex-wrap">
      <DSToggle
        bind:value={expenseListMode}
        options={listModeOptions}
        ariaLabel="Expense committed status filter"
      />
    </div>
  {/snippet}
</ExpensesList>
