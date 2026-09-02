<script lang="ts">
  import { tick } from "svelte";
  import Portal from "$lib/components/Portal.svelte";
  import {
    buildBranchResourceFlow,
    getFlowEmployees,
    type EmployeeBranchHoursRow,
  } from "$lib/reports/branchResourceFlow";

  type MatrixMode = "hours" | "staff" | "work";
  type DialogView = "matrix" | "branch";
  type DetailSelection = {
    homeBranch: string;
    workBranch: string | null;
  };

  let {
    show = $bindable(false),
    columns,
    rows,
    startDate,
    endDate,
  } = $props<{
    show: boolean;
    columns: string[];
    rows: EmployeeBranchHoursRow[];
    startDate: string;
    endDate: string;
  }>();

  let view = $state<DialogView>("matrix");
  let matrixMode = $state<MatrixMode>("hours");
  let selectedBranch = $state("");
  let detailSelection = $state<DetailSelection | null>(null);
  let closeButton = $state<HTMLButtonElement | null>(null);
  let dialogElement = $state<HTMLDivElement | null>(null);

  const flow = $derived(buildBranchResourceFlow(columns, rows));
  const selectedSummary = $derived(flow.summaries[selectedBranch]);
  const selectedEmployees = $derived.by(() => {
    if (!detailSelection) return [];
    return getFlowEmployees(
      rows,
      flow.branchColumns,
      detailSelection.homeBranch,
      detailSelection.workBranch,
    );
  });
  const suppliedFlows = $derived.by(() =>
    flow.workBranches
      .filter((branch) => branch !== selectedBranch)
      .map((branch) => ({ branch, hours: flow.matrix[selectedBranch]?.[branch] ?? 0 }))
      .filter(({ hours }) => hours > 0)
      .sort((left, right) => right.hours - left.hours),
  );
  const receivedFlows = $derived.by(() =>
    flow.homeBranches
      .filter((branch) => branch !== selectedBranch)
      .map((branch) => ({ branch, hours: flow.matrix[branch]?.[selectedBranch] ?? 0 }))
      .filter(({ hours }) => hours > 0)
      .sort((left, right) => right.hours - left.hours),
  );
  const selectedFlowMaximum = $derived(
    Math.max(
      0,
      ...suppliedFlows.map(({ hours }) => hours),
      ...receivedFlows.map(({ hours }) => hours),
    ),
  );
  const matrixMaximum = $derived.by(() => {
    const values: number[] = [];
    for (const homeBranch of flow.homeBranches) {
      for (const workBranch of flow.workBranches) {
        values.push(matrixCellValue(homeBranch, workBranch));
      }
      const noJobValue = noJobCellValue(homeBranch);
      if (noJobValue !== null) values.push(noJobValue);
    }
    return Math.max(0, ...values);
  });
  const detailHours = $derived.by(() => {
    if (!detailSelection) return 0;
    return detailSelection.workBranch
      ? (flow.matrix[detailSelection.homeBranch]?.[detailSelection.workBranch] ?? 0)
      : (flow.noJobTotals[detailSelection.homeBranch] ?? 0);
  });

  $effect(() => {
    if (!show) return;
    if (!selectedBranch || !flow.homeBranches.includes(selectedBranch)) {
      selectedBranch =
        flow.homeBranches.find((branch) => {
          const summary = flow.summaries[branch];
          return summary && summary.staffJobHours + summary.workHours + summary.noJobHours > 0;
        }) ??
        flow.homeBranches[0] ??
        "";
    }
  });

  $effect(() => {
    if (!show || typeof document === "undefined") return;
    const previousActiveElement = document.activeElement;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    void tick().then(() => {
      if (show) closeButton?.focus();
    });
    return () => {
      document.body.style.overflow = previousOverflow;
      if (previousActiveElement instanceof HTMLElement) previousActiveElement.focus();
    };
  });

  function close() {
    show = false;
    detailSelection = null;
  }

  function handleDialogKeydown(event: KeyboardEvent) {
    if (event.key === "Escape") {
      close();
      return;
    }
    if (event.key !== "Tab" || !dialogElement) return;

    const focusableElements = Array.from(
      dialogElement.querySelectorAll<HTMLElement>(
        'button:not([disabled]), select:not([disabled]), [href], [tabindex]:not([tabindex="-1"])',
      ),
    ).filter((element) => element.offsetParent !== null);
    if (focusableElements.length === 0) return;

    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    if (event.shiftKey && document.activeElement === firstElement) {
      event.preventDefault();
      lastElement.focus();
    } else if (!event.shiftKey && document.activeElement === lastElement) {
      event.preventDefault();
      firstElement.focus();
    }
  }

  function handleBackdropClick(event: MouseEvent) {
    if (event.target === event.currentTarget) close();
  }

  function selectMatrixCell(homeBranch: string, workBranch: string | null) {
    detailSelection = { homeBranch, workBranch };
  }

  function selectBranch(branch: string) {
    selectedBranch = branch;
    detailSelection = null;
  }

  function matrixCellValue(homeBranch: string, workBranch: string): number {
    const hours = flow.matrix[homeBranch]?.[workBranch] ?? 0;
    if (matrixMode === "staff") {
      const total = (flow.staffJobTotals[homeBranch] ?? 0) + (flow.noJobTotals[homeBranch] ?? 0);
      return total > 0 ? hours / total : 0;
    }
    if (matrixMode === "work") {
      const total = flow.workTotals[workBranch] ?? 0;
      return total > 0 ? hours / total : 0;
    }
    return hours;
  }

  function noJobCellValue(homeBranch: string): number | null {
    const hours = flow.noJobTotals[homeBranch] ?? 0;
    if (matrixMode === "work") return null;
    if (matrixMode === "staff") {
      const total = (flow.staffJobTotals[homeBranch] ?? 0) + hours;
      return total > 0 ? hours / total : 0;
    }
    return hours;
  }

  function cellOpacity(value: number): string {
    if (value <= 0 || matrixMaximum <= 0) return "0";
    return String(0.14 + Math.sqrt(value / matrixMaximum) * 0.7);
  }

  function flowWidth(hours: number): string {
    if (hours <= 0 || selectedFlowMaximum <= 0) return "0%";
    return `${Math.max(8, (hours / selectedFlowMaximum) * 100)}%`;
  }

  function formatHours(value: number): string {
    return value.toLocaleString("en-CA", {
      minimumFractionDigits: 0,
      maximumFractionDigits: 2,
    });
  }

  function formatPercent(value: number): string {
    return value.toLocaleString("en-CA", {
      style: "percent",
      minimumFractionDigits: 0,
      maximumFractionDigits: 1,
    });
  }

  function formatMatrixValue(value: number | null): string {
    if (value === null) return "—";
    return matrixMode === "hours" ? formatHours(value) : formatPercent(value);
  }

  function matrixCellLabel(homeBranch: string, workBranch: string, value: number): string {
    const unit = matrixMode === "hours" ? `${formatHours(value)} hours` : formatPercent(value);
    return `${homeBranch} staff on ${workBranch} work: ${unit}`;
  }

  function detailTitle(selection: DetailSelection): string {
    return selection.workBranch
      ? `${selection.homeBranch} staff → ${selection.workBranch} work`
      : `${selection.homeBranch} staff → No-job time`;
  }
</script>

{#if show}
  <Portal>
    <div
      class="fixed inset-0 z-9999 flex items-center justify-center bg-black/55 p-2 sm:p-4"
      role="presentation"
      onclick={handleBackdropClick}
      onkeydown={handleDialogKeydown}
    >
      <div
        bind:this={dialogElement}
        class="flex max-h-[calc(100dvh-1rem)] w-full max-w-[96rem] flex-col overflow-hidden rounded-xl bg-white shadow-2xl sm:max-h-[calc(100dvh-2rem)]"
        role="dialog"
        aria-modal="true"
        aria-labelledby="branch-resource-flow-title"
      >
        <header
          class="flex items-start justify-between gap-4 border-b border-neutral-200 px-4 py-3 sm:px-6"
        >
          <div>
            <h2 id="branch-resource-flow-title" class="text-xl font-bold text-neutral-900">
              Branch Resource Flow
            </h2>
            <p class="mt-1 text-sm text-neutral-600">
              {startDate} through {endDate}, inclusive
            </p>
          </div>
          <button
            bind:this={closeButton}
            type="button"
            class="rounded-md px-3 py-1 text-2xl leading-none text-neutral-500 hover:bg-neutral-100 hover:text-neutral-900"
            aria-label="Close branch resource flow"
            onclick={close}>×</button
          >
        </header>

        <div
          class="flex flex-wrap items-center justify-between gap-3 border-b border-neutral-200 px-4 py-3 sm:px-6"
        >
          <div
            class="flex rounded-lg bg-neutral-100 p-1"
            role="tablist"
            aria-label="Resource flow view"
          >
            <button
              type="button"
              role="tab"
              aria-selected={view === "matrix"}
              class="rounded-md px-3 py-1.5 text-sm font-medium {view === 'matrix'
                ? 'bg-white text-blue-700 shadow-sm'
                : 'text-neutral-600 hover:text-neutral-900'}"
              onclick={() => (view = "matrix")}>Branch Matrix</button
            >
            <button
              type="button"
              role="tab"
              aria-selected={view === "branch"}
              class="rounded-md px-3 py-1.5 text-sm font-medium {view === 'branch'
                ? 'bg-white text-blue-700 shadow-sm'
                : 'text-neutral-600 hover:text-neutral-900'}"
              onclick={() => (view = "branch")}>Selected Branch</button
            >
          </div>

          {#if view === "matrix"}
            <div class="flex flex-wrap items-center gap-1" aria-label="Matrix value mode">
              {#each [{ value: "hours", label: "Hours" }, { value: "staff", label: "% of staff time" }, { value: "work", label: "% of work staffing" }] as option (option.value)}
                <button
                  type="button"
                  class="rounded-md border px-2.5 py-1 text-sm {matrixMode === option.value
                    ? 'border-blue-600 bg-blue-50 text-blue-800'
                    : 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-400'}"
                  aria-pressed={matrixMode === option.value}
                  onclick={() => (matrixMode = option.value as MatrixMode)}>{option.label}</button
                >
              {/each}
            </div>
          {:else}
            <label class="flex items-center gap-2 text-sm font-medium text-neutral-700">
              Branch
              <select
                class="rounded-md border border-neutral-300 bg-white px-2 py-1.5"
                value={selectedBranch}
                onchange={(event) => selectBranch(event.currentTarget.value)}
              >
                {#each flow.homeBranches as branch (branch)}
                  <option value={branch}>{branch}</option>
                {/each}
              </select>
            </label>
          {/if}
        </div>

        <div class="flex-1 overflow-y-auto px-4 py-4 sm:px-6">
          {#if view === "matrix"}
            <p class="mb-3 text-sm text-neutral-600">
              Rows are employee default branches. Columns are the branches where job hours were
              recorded.
            </p>
            <div
              class="mb-3 rounded-md border border-blue-100 bg-blue-50 px-3 py-2 text-sm text-blue-950"
              aria-live="polite"
            >
              {#if matrixMode === "hours"}
                Each cell is the number of R/RT job hours supplied by the row's staff branch to the
                column's work branch.
              {:else if matrixMode === "staff"}
                Each cell is its share of all qualifying time for the row's staff branch, including
                no-job time. Each row totals 100%.
              {:else}
                This answers: <strong>Who staffed each branch's work?</strong> Each cell is the share
                of all job hours for the column's work branch that came from the row's staff branch. Each
                work column totals 100%. No-job time is excluded.
              {/if}
            </div>
            <div class="overflow-x-auto rounded-lg border border-neutral-200">
              <table class="flow-matrix min-w-full border-collapse text-sm">
                <thead>
                  <tr>
                    <th class="sticky left-0 z-20 min-w-40 bg-neutral-100 px-3 py-2 text-left">
                      Staff branch ↓<br /><span class="font-normal text-neutral-500"
                        >Work branch →</span
                      >
                    </th>
                    {#each flow.workBranches as branch (branch)}
                      <th class="min-w-28 bg-neutral-100 px-2 py-2 text-center">{branch}</th>
                    {/each}
                    <th class="min-w-24 bg-neutral-100 px-2 py-2 text-right">Job total (h)</th>
                    <th class="min-w-24 bg-amber-50 px-2 py-2 text-right">No job</th>
                  </tr>
                </thead>
                <tbody>
                  {#each flow.homeBranches as homeBranch (homeBranch)}
                    {@const noJobValue = noJobCellValue(homeBranch)}
                    <tr>
                      <th
                        class="sticky left-0 z-10 border-t border-neutral-200 bg-white px-3 py-2 text-left"
                      >
                        {homeBranch}
                      </th>
                      {#each flow.workBranches as workBranch (workBranch)}
                        {@const value = matrixCellValue(homeBranch, workBranch)}
                        {@const rawHours = flow.matrix[homeBranch]?.[workBranch] ?? 0}
                        <td class="border-t border-l border-neutral-200 p-0 text-center">
                          <button
                            type="button"
                            class:local-flow-cell={homeBranch === workBranch}
                            class:cross-flow-cell={homeBranch !== workBranch}
                            class:text-white={matrixMaximum > 0 && value / matrixMaximum >= 0.45}
                            class="h-full min-h-11 w-full px-2 py-2 font-medium disabled:cursor-default"
                            style={`--flow-opacity: ${cellOpacity(value)}`}
                            disabled={rawHours <= 0}
                            aria-label={matrixCellLabel(homeBranch, workBranch, value)}
                            onclick={() => selectMatrixCell(homeBranch, workBranch)}
                            >{formatMatrixValue(value)}</button
                          >
                        </td>
                      {/each}
                      <td
                        class="border-t border-l border-neutral-200 px-2 py-2 text-right font-semibold"
                      >
                        {formatHours(flow.staffJobTotals[homeBranch] ?? 0)}
                      </td>
                      <td class="border-t border-l border-neutral-200 p-0 text-right">
                        <button
                          type="button"
                          class="no-job-flow-cell min-h-11 w-full px-2 py-2 text-right font-medium disabled:cursor-default disabled:bg-amber-50"
                          style={`--flow-opacity: ${cellOpacity(noJobValue ?? 0)}`}
                          disabled={noJobValue === null || (flow.noJobTotals[homeBranch] ?? 0) <= 0}
                          aria-label={`${homeBranch} no-job time: ${formatMatrixValue(noJobValue)}`}
                          onclick={() => selectMatrixCell(homeBranch, null)}
                          >{formatMatrixValue(noJobValue)}</button
                        >
                      </td>
                    </tr>
                  {/each}
                </tbody>
                <tfoot>
                  <tr class="border-t-2 border-neutral-300 bg-neutral-50 font-semibold">
                    <th class="sticky left-0 z-10 bg-neutral-50 px-3 py-2 text-left">
                      Work total (h)
                    </th>
                    {#each flow.workBranches as branch (branch)}
                      <td class="border-l border-neutral-200 px-2 py-2 text-center">
                        {formatHours(flow.workTotals[branch] ?? 0)}
                      </td>
                    {/each}
                    <td class="border-l border-neutral-200 px-2 py-2 text-right">
                      {formatHours(
                        Object.values(flow.staffJobTotals).reduce((sum, value) => sum + value, 0),
                      )}
                    </td>
                    <td class="border-l border-neutral-200 bg-amber-50 px-2 py-2 text-right">
                      {formatHours(
                        Object.values(flow.noJobTotals).reduce((sum, value) => sum + value, 0),
                      )}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
            <div class="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-sm text-neutral-600">
              <span
                ><span class="mr-1 inline-block h-3 w-3 bg-neutral-400/40"></span>Local work</span
              >
              <span
                ><span class="mr-1 inline-block h-3 w-3 bg-blue-600/40"></span>Cross-branch work</span
              >
              <span><span class="mr-1 inline-block h-3 w-3 bg-amber-500/40"></span>No-job R/RT</span
              >
              {#if matrixMode === "work"}<span
                  >No-job percentages do not apply to work staffing.</span
                >{/if}
            </div>
          {:else if selectedSummary}
            <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <article class="rounded-lg border border-neutral-200 p-3">
                <p class="text-sm text-neutral-500">Local work</p>
                <p class="mt-1 text-2xl font-semibold">
                  {formatHours(selectedSummary.localHours)} h
                </p>
                <p class="text-sm text-neutral-500">
                  {formatPercent(selectedSummary.localWorkShare)} of staff job hours
                </p>
              </article>
              <article class="rounded-lg border border-neutral-200 p-3">
                <p class="text-sm text-neutral-500">Staff supplied elsewhere</p>
                <p class="mt-1 text-2xl font-semibold">
                  {formatHours(selectedSummary.suppliedHours)} h
                </p>
                <p class="text-sm text-neutral-500">Work completed for other branches</p>
              </article>
              <article class="rounded-lg border border-neutral-200 p-3">
                <p class="text-sm text-neutral-500">Outside staff received</p>
                <p class="mt-1 text-2xl font-semibold">
                  {formatHours(selectedSummary.receivedHours)} h
                </p>
                <p class="text-sm text-neutral-500">
                  {formatPercent(selectedSummary.outsideStaffingShare)} of branch work
                </p>
              </article>
              <button
                type="button"
                class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-left hover:border-amber-400 disabled:cursor-default disabled:opacity-60 disabled:hover:border-amber-200"
                disabled={selectedSummary.noJobHours <= 0}
                onclick={() => selectMatrixCell(selectedBranch, null)}
              >
                <span class="text-sm text-neutral-600">No-job R/RT</span>
                <span class="mt-1 block text-2xl font-semibold"
                  >{formatHours(selectedSummary.noJobHours)} h</span
                >
                <span class="text-sm text-neutral-600">
                  {formatPercent(selectedSummary.noJobShare)} of qualifying time
                </span>
              </button>
            </div>

            <p class="mt-3 text-sm text-neutral-600">
              Resource balance: <strong
                >{selectedSummary.resourceBalance >= 0 ? "+" : ""}{formatHours(
                  selectedSummary.resourceBalance,
                )} h</strong
              >
              (outside staff received minus staff supplied elsewhere).
            </p>

            <div class="mt-5 grid gap-6 lg:grid-cols-2">
              <section>
                <h3 class="text-base font-semibold text-neutral-900">Our staff worked for</h3>
                <p class="mb-3 text-sm text-neutral-500">Select a branch to see the employees.</p>
                {#if suppliedFlows.length === 0}
                  <p class="rounded-md bg-neutral-50 p-3 text-sm text-neutral-500">
                    No cross-branch hours.
                  </p>
                {:else}
                  <div class="space-y-2">
                    {#each suppliedFlows as item (item.branch)}
                      <button
                        type="button"
                        class="group relative block min-h-12 w-full overflow-hidden rounded-md bg-neutral-100 text-left"
                        onclick={() => selectMatrixCell(selectedBranch, item.branch)}
                      >
                        <span
                          class="absolute inset-y-0 left-0 bg-blue-200 transition-all group-hover:bg-blue-300"
                          style:width={flowWidth(item.hours)}
                        ></span>
                        <span class="relative flex items-center justify-between gap-3 px-3 py-2">
                          <span class="font-medium">{item.branch}</span>
                          <span class="text-right">
                            <strong>{formatHours(item.hours)} h</strong>
                            <span class="block text-xs text-neutral-600">
                              {formatPercent(
                                selectedSummary.staffJobHours > 0
                                  ? item.hours / selectedSummary.staffJobHours
                                  : 0,
                              )} of staff job hours
                            </span>
                          </span>
                        </span>
                      </button>
                    {/each}
                  </div>
                {/if}
              </section>

              <section>
                <h3 class="text-base font-semibold text-neutral-900">
                  Other branches staffed our work
                </h3>
                <p class="mb-3 text-sm text-neutral-500">Select a branch to see the employees.</p>
                {#if receivedFlows.length === 0}
                  <p class="rounded-md bg-neutral-50 p-3 text-sm text-neutral-500">
                    No outside staff hours.
                  </p>
                {:else}
                  <div class="space-y-2">
                    {#each receivedFlows as item (item.branch)}
                      <button
                        type="button"
                        class="group relative block min-h-12 w-full overflow-hidden rounded-md bg-neutral-100 text-left"
                        onclick={() => selectMatrixCell(item.branch, selectedBranch)}
                      >
                        <span
                          class="absolute inset-y-0 right-0 bg-violet-200 transition-all group-hover:bg-violet-300"
                          style:width={flowWidth(item.hours)}
                        ></span>
                        <span class="relative flex items-center justify-between gap-3 px-3 py-2">
                          <span class="font-medium">{item.branch}</span>
                          <span class="text-right">
                            <strong>{formatHours(item.hours)} h</strong>
                            <span class="block text-xs text-neutral-600">
                              {formatPercent(
                                selectedSummary.workHours > 0
                                  ? item.hours / selectedSummary.workHours
                                  : 0,
                              )} of branch work
                            </span>
                          </span>
                        </span>
                      </button>
                    {/each}
                  </div>
                {/if}
              </section>
            </div>
          {/if}

          {#if detailSelection}
            <section class="mt-6 border-t border-neutral-200 pt-4" aria-live="polite">
              <div class="flex flex-wrap items-baseline justify-between gap-2">
                <div>
                  <h3 class="text-lg font-semibold text-neutral-900">
                    {detailTitle(detailSelection)}
                  </h3>
                  <p class="text-sm text-neutral-600">
                    {formatHours(detailHours)} hours across {selectedEmployees.length} employee{selectedEmployees.length ===
                    1
                      ? ""
                      : "s"}.
                  </p>
                </div>
                <button
                  type="button"
                  class="text-sm text-blue-700 hover:underline"
                  onclick={() => (detailSelection = null)}>Clear selection</button
                >
              </div>
              <div class="mt-3 max-h-72 overflow-auto rounded-lg border border-neutral-200">
                <table class="min-w-full text-sm">
                  <thead class="sticky top-0 bg-neutral-100 text-left">
                    <tr>
                      <th class="px-3 py-2">Payroll ID</th>
                      <th class="px-3 py-2">Surname</th>
                      <th class="px-3 py-2">Given name</th>
                      <th class="px-3 py-2">Default branch</th>
                      <th class="px-3 py-2 text-right">Hours</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each selectedEmployees as employee (`${employee.payrollId}-${employee.surname}-${employee.givenName}`)}
                      <tr class="border-t border-neutral-200 odd:bg-white even:bg-neutral-50">
                        <td class="px-3 py-2">{employee.payrollId}</td>
                        <td class="px-3 py-2">{employee.surname}</td>
                        <td class="px-3 py-2">{employee.givenName}</td>
                        <td class="px-3 py-2">{employee.defaultBranch}</td>
                        <td class="px-3 py-2 text-right font-medium"
                          >{formatHours(employee.hours)}</td
                        >
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
            </section>
          {/if}
        </div>
      </div>
    </div>
  </Portal>
{/if}

<style>
  .flow-matrix th,
  .flow-matrix td {
    white-space: nowrap;
  }

  .local-flow-cell {
    background-color: rgb(82 82 91 / var(--flow-opacity));
  }

  .cross-flow-cell {
    background-color: rgb(37 99 235 / var(--flow-opacity));
  }

  .no-job-flow-cell {
    background-color: rgb(245 158 11 / var(--flow-opacity));
  }

  .local-flow-cell:hover:not(:disabled),
  .cross-flow-cell:hover:not(:disabled),
  .no-job-flow-cell:hover:not(:disabled) {
    box-shadow: inset 0 0 0 2px rgb(30 64 175);
  }
</style>
