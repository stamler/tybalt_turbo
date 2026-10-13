<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { resolve } from "$app/paths";
  import { page } from "$app/state";
  import { replaceState } from "$app/navigation";
  import { SvelteSet } from "svelte/reactivity";
  import Icon from "@iconify/svelte";
  import { globalStore } from "$lib/stores/global";
  import { getApiErrorMessage } from "$lib/errors";
  import { authStore } from "$lib/stores/auth";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { allExpensePaymentTypeOptions } from "$lib/expensePaymentTypes";
  import { formatCurrency, formatCurrencyAmount, formatJobLabel, shortDate } from "$lib/utilities";
  import { isUpcomingPayDate, payPeriodStart } from "$lib/payPeriods";

  type Period = {
    pay_period_ending: string;
    pay_date: string;
    expense_count: number;
    total: number;
  };

  type Expense = {
    id: string;
    date: string;
    payment_type: string;
    description: string;
    distance: number;
    job_number: string;
    job_description: string;
    client_name: string;
    currency_code: string;
    foreign_currency_total: number;
    amount: number;
  };

  let periods = $state([] as Period[]);
  let loaded = $state(false);
  const open = new SvelteSet<string>();
  let expenses = $state({} as Record<string, Expense[]>);
  let loading = $state({} as Record<string, boolean>);

  function paymentTypeLabel(paymentType: string) {
    if (paymentType === "Meals") return "Allowance";
    return (
      allExpensePaymentTypeOptions.find((option) => option.id === paymentType)?.name ?? paymentType
    );
  }

  async function loadExpenses(payPeriodEnding: string) {
    if (expenses[payPeriodEnding] !== undefined || loading[payPeriodEnding]) return;
    loading[payPeriodEnding] = true;
    try {
      expenses[payPeriodEnding] = await pb.send(`/api/expenses/reimbursements/${payPeriodEnding}`, {
        method: "GET",
      });
    } catch (error: unknown) {
      globalStore.addError(getApiErrorMessage(error, "Failed to load reimbursed expenses"));
    } finally {
      loading[payPeriodEnding] = false;
    }
  }

  // Mirror the open periods into the URL so returning from an expense's
  // details page restores the same view. An empty ?open= means the user
  // collapsed everything, which is different from no parameter (the default).
  function syncUrl() {
    // Keys are YYYY-MM-DD dates, so the comma list needs no escaping.
    const openList = [...open].sort().join(",");
    // The path is resolved; the rule cannot see through the appended query.
    // eslint-disable-next-line svelte/no-navigation-without-resolve
    replaceState(`${resolve("/expenses/reimbursements")}?open=${openList}`, page.state);
  }

  function toggle(payPeriodEnding: string) {
    if (open.has(payPeriodEnding)) {
      open.delete(payPeriodEnding);
    } else {
      open.add(payPeriodEnding);
      loadExpenses(payPeriodEnding);
    }
    syncUrl();
  }

  async function init() {
    if (!$authStore?.isValid) {
      return;
    }

    try {
      periods = await pb.send("/api/expenses/reimbursements", { method: "GET" });
    } catch (error: unknown) {
      globalStore.addError(getApiErrorMessage(error, "Failed to load reimbursements"));
    } finally {
      loaded = true;
    }

    const requested = page.url.searchParams.get("open");
    const initial =
      requested === null
        ? // Default to the most recent pay day that has already been paid.
          periods.filter((p) => !isUpcomingPayDate(p.pay_date)).slice(0, 1)
        : periods.filter((p) => requested.split(",").includes(p.pay_period_ending));
    for (const p of initial) {
      open.add(p.pay_period_ending);
      loadExpenses(p.pay_period_ending);
    }
  }

  init();
</script>

<ul>
  <li class="border-b border-blue-300 bg-blue-200 px-2 text-neutral-700">
    My Reimbursements by Pay Day
  </li>
  {#if loaded && periods.length === 0}
    <li class="p-2 text-neutral-600">No committed expenses have been reimbursed to you yet.</li>
  {/if}
  {#each periods as p (p.pay_period_ending)}
    {@const isOpen = open.has(p.pay_period_ending)}
    <li class="border-b border-neutral-300 odd:bg-neutral-100 even:bg-neutral-200">
      <button
        type="button"
        class="flex w-full items-center gap-3 p-2 text-left hover:bg-neutral-50"
        aria-expanded={isOpen}
        aria-controls={`period-${p.pay_period_ending}`}
        onclick={() => toggle(p.pay_period_ending)}
      >
        <Icon
          icon="mdi:chevron-right"
          width="24px"
          class="shrink-0 transition-transform {isOpen ? 'rotate-90' : ''}"
        />
        <span class="flex min-w-0 flex-col">
          <span class="flex flex-wrap items-center gap-2">
            <span class="font-bold">{shortDate(p.pay_date, true)}</span>
            <span class="font-bold">{formatCurrency(p.total)}</span>
            <span class="text-sm">{p.expense_count} expense{p.expense_count === 1 ? "" : "s"}</span>
            {#if isUpcomingPayDate(p.pay_date)}
              <DsLabel color="yellow" title="Expenses committed before pay day may still be added"
                >Upcoming</DsLabel
              >
            {/if}
          </span>
          <span class="text-sm text-neutral-600">
            Pay period {shortDate(payPeriodStart(p.pay_period_ending), true)} to
            {shortDate(p.pay_period_ending, true)}
          </span>
        </span>
      </button>

      {#if isOpen}
        <ul id={`period-${p.pay_period_ending}`} class="ml-11 border-l border-neutral-300 pb-2">
          {#if expenses[p.pay_period_ending] === undefined}
            <li class="px-3 py-1 text-sm text-neutral-600">
              {loading[p.pay_period_ending] ? "Loading..." : "Could not load expenses."}
            </li>
          {:else}
            {#each expenses[p.pay_period_ending] as e (e.id)}
              <li class="flex flex-col px-3 py-1 text-sm">
                <span class="flex flex-wrap items-center gap-2">
                  <a
                    href={resolve(`/expenses/${e.id}/details`)}
                    class="text-blue-600 hover:underline">{e.date}</a
                  >
                  <span class="font-semibold">{e.description}</span>
                  <span>
                    {#if e.payment_type === "Mileage"}{e.distance} km /{/if}
                    {formatCurrency(e.amount)}
                  </span>
                  {#if e.currency_code !== "CAD"}
                    <span class="text-xs text-gray-500"
                      >({formatCurrencyAmount(e.foreign_currency_total, e.currency_code)})</span
                    >
                  {/if}
                  <DsLabel color="cyan">{paymentTypeLabel(e.payment_type)}</DsLabel>
                </span>
                {#if e.job_number !== ""}
                  <span class="text-neutral-600">{formatJobLabel(e)}</span>
                {/if}
              </li>
            {/each}
          {/if}
        </ul>
      {/if}
    </li>
  {/each}
</ul>
