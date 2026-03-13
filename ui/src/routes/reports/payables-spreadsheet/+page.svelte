<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { globalStore } from "$lib/stores/global";
  import type { PageData } from "./$types";
  import { downloadCSV } from "$lib/utilities";

  let { data }: { data: PageData } = $props();
  let monthlyYYMM = $state("");
  let copyFeedback = $state<Record<string, boolean>>({});
  let monthlyCopyFeedback = $state(false);
  const dates = $derived(data.dates);

  // Group dates into weekly rows (Mon-Sun). Most recent week first.
  function groupByWeek(dateStrings: string[]) {
    const weeks: { label: string; dates: string[] }[] = [];
    if (dateStrings.length === 0) return weeks;

    // Group by ISO week
    // eslint-disable-next-line svelte/prefer-svelte-reactivity
    const weekMap = new Map<string, string[]>();
    for (const ds of dateStrings) {
      const d = new Date(ds + "T00:00:00");
      // Get Monday of this week
      const day = d.getDay();
      const diff = d.getDate() - day + (day === 0 ? -6 : 1);
      // eslint-disable-next-line svelte/prefer-svelte-reactivity
      const monday = new Date(d);
      monday.setDate(diff);
      const key = monday.toISOString().slice(0, 10);
      if (!weekMap.has(key)) weekMap.set(key, []);
      weekMap.get(key)!.push(ds);
    }

    // Sort weeks descending (most recent first)
    const sortedKeys = [...weekMap.keys()].sort().reverse();
    for (const key of sortedKeys) {
      const mondayDate = new Date(key + "T00:00:00");
      // eslint-disable-next-line svelte/prefer-svelte-reactivity
      const sundayDate = new Date(mondayDate);
      sundayDate.setDate(sundayDate.getDate() + 6);
      const label = `Week of ${mondayDate.toLocaleDateString("en-CA", { month: "short", day: "numeric" })} - ${sundayDate.toLocaleDateString("en-CA", { month: "short", day: "numeric" })}`;
      weeks.push({ label, dates: weekMap.get(key)!.sort() });
    }

    return weeks;
  }

  function formatDate(ds: string) {
    const d = new Date(ds + "T00:00:00");
    return d.toLocaleDateString("en-CA", {
      weekday: "short",
      month: "short",
      day: "numeric",
    });
  }

  function setCopied(key: string) {
    copyFeedback = { ...copyFeedback, [key]: true };
    setTimeout(() => {
      copyFeedback = { ...copyFeedback, [key]: false };
    }, 2000);
  }

  function setMonthlyCopied() {
    monthlyCopyFeedback = true;
    setTimeout(() => {
      monthlyCopyFeedback = false;
    }, 2000);
  }

  async function fetchReportText(url: string) {
    const headers: Record<string, string> = {};
    if (pb.authStore.isValid) {
      headers["Authorization"] = pb.authStore.token;
    }

    const response = await fetch(url, { headers });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.text();
  }

  async function copyText(textPromise: Promise<string>) {
    if (
      typeof navigator !== "undefined" &&
      navigator.clipboard?.write &&
      typeof ClipboardItem !== "undefined"
    ) {
      await navigator.clipboard.write([
        new ClipboardItem({
          "text/plain": textPromise.then((text) => new Blob([text], { type: "text/plain" })),
        }),
      ]);
      return;
    }

    const text = await textPromise;

    if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return;
    }

    if (typeof document === "undefined") {
      throw new Error("Clipboard unavailable");
    }

    const el = document.createElement("textarea");
    el.value = text;
    el.setAttribute("readonly", "");
    el.style.position = "absolute";
    el.style.left = "-9999px";
    document.body.appendChild(el);
    el.select();
    document.execCommand("copy");
    document.body.removeChild(el);
  }

  async function fetchTSVAndCopy(dateStr: string) {
    const url = `${pb.baseUrl}/api/reports/payables_spreadsheet/${dateStr}?format=tsv`;
    try {
      await copyText(fetchReportText(url));
      setCopied(dateStr);
    } catch (error) {
      console.error("Failed to copy payables spreadsheet:", error);
      globalStore.addError("Failed to copy spreadsheet");
    }
  }

  async function fetchCSV(dateStr: string) {
    const url = `${pb.baseUrl}/api/reports/payables_spreadsheet/${dateStr}`;
    const fileName = `payables_spreadsheet_${dateStr}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchMonthlyCSV() {
    if (monthlyYYMM.length !== 4) return;
    const url = `${pb.baseUrl}/api/reports/payables_spreadsheet_monthly/${monthlyYYMM}`;
    const fileName = `payables_spreadsheet_monthly_${monthlyYYMM}.csv`;
    await downloadCSV(url, fileName);
  }

  async function fetchMonthlyTSVAndCopy() {
    if (monthlyYYMM.length !== 4) return;
    const url = `${pb.baseUrl}/api/reports/payables_spreadsheet_monthly/${monthlyYYMM}?format=tsv`;
    try {
      await copyText(fetchReportText(url));
      setMonthlyCopied();
    } catch (error) {
      console.error("Failed to copy monthly payables spreadsheet:", error);
      globalStore.addError("Failed to copy spreadsheet");
    }
  }

  const weeks = $derived(groupByWeek(dates));
</script>

<div class="mx-auto max-w-4xl p-4">
  <h1 class="mb-4 text-2xl font-bold">Payables Spreadsheet Downloads</h1>
  <div class="mb-6 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
    Daily download dates are based on approval timestamps stored in UTC and only appear once the
    full UTC day has completed. Approvals entered late in the local evening may therefore appear
    under the following day here.
  </div>

  {#if weeks.length === 0}
    <p class="text-gray-500">No approved POs found in the last 4 weeks.</p>
  {:else}
    {#each weeks as week (week.label)}
      <div class="mb-6">
        <h2 class="mb-2 text-lg font-semibold text-gray-700">{week.label}</h2>
        <div class="flex flex-wrap gap-2">
          {#each week.dates as dateStr (dateStr)}
            <div
              class="flex items-center gap-1 rounded-lg border border-gray-200 bg-white px-3 py-2 shadow-sm"
            >
              <span class="mr-2 text-sm font-medium">{formatDate(dateStr)}</span>
              <button
                class="rounded bg-blue-500 px-2 py-1 text-xs text-white hover:bg-blue-600"
                onclick={() => fetchTSVAndCopy(dateStr)}
              >
                {copyFeedback[dateStr] ? "Copied!" : "Copy"}
              </button>
              <button
                class="rounded bg-orange-500 px-2 py-1 text-xs text-white hover:bg-orange-600"
                onclick={() => fetchCSV(dateStr)}
              >
                CSV
              </button>
            </div>
          {/each}
        </div>
      </div>
    {/each}
  {/if}

  <div class="mt-8 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
    <h2 class="mb-3 text-lg font-semibold text-gray-700">Prefix Summary</h2>
    <div class="flex items-center gap-3">
      <label for="yymm" class="text-sm text-gray-600">PO Prefix (YYMM):</label>
      <input
        id="yymm"
        type="text"
        bind:value={monthlyYYMM}
        placeholder="e.g. 2603"
        maxlength={4}
        class="w-24 rounded border border-gray-300 px-2 py-1 text-sm"
      />
      <button
        class="rounded bg-blue-500 px-3 py-1 text-sm text-white hover:bg-blue-600 disabled:opacity-50"
        disabled={monthlyYYMM.length !== 4}
        onclick={fetchMonthlyTSVAndCopy}
      >
        {monthlyCopyFeedback ? "Copied!" : "Copy"}
      </button>
      <button
        class="rounded bg-orange-500 px-3 py-1 text-sm text-white hover:bg-orange-600 disabled:opacity-50"
        disabled={monthlyYYMM.length !== 4}
        onclick={fetchMonthlyCSV}
      >
        CSV
      </button>
    </div>
  </div>
</div>
