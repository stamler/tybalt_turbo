<script lang="ts">
  import { shortDate } from "$lib/utilities";
  import { onMount } from "svelte";
  import { untrack } from "svelte";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  let rateSheet = $state(untrack(() => data.rateSheet));
  let entries = $state(untrack(() => data.entries));

  // Sort entries by rate descending
  const sortedEntries = $derived([...entries].sort((a, b) => b.rate - a.rate));

  // Format currency values
  function formatRate(value: number): string {
    return `$${value.toFixed(2)}`;
  }

  // Trigger print dialog on mount
  onMount(() => {
    // Small delay to ensure the page is fully rendered
    const timer = setTimeout(() => window.print(), 300);
    return () => clearTimeout(timer);
  });
</script>

<div class="print-page mx-auto max-w-4xl p-8">
  <!-- Header -->
  <div class="mb-6 border-b-2 border-black pb-4">
    <h1 class="text-2xl font-bold text-black">
      {rateSheet.name}
      <span class="text-lg font-normal">— Rev. {rateSheet.revision}</span>
    </h1>
    <p class="mt-1 text-sm text-black">
      Effective: {shortDate(rateSheet.effective_date, true)}
    </p>
  </div>

  <!-- Rate Entries Table -->
  <table class="w-full border-collapse">
    <thead>
      <tr class="border-b-2 border-black">
        <th class="py-2 pr-4 text-left text-sm font-bold text-black uppercase">Role</th>
        <th class="py-2 pr-4 text-right text-sm font-bold text-black uppercase">Rate</th>
        <th class="py-2 text-right text-sm font-bold text-black uppercase">Overtime Rate</th>
      </tr>
    </thead>
    <tbody>
      {#each sortedEntries as entry (entry.id)}
        <tr class="border-b border-neutral-400">
          <td class="py-2 pr-4 text-black">{entry.expand?.role?.name ?? "Unknown"}</td>
          <td class="py-2 pr-4 text-right text-black">{formatRate(entry.rate)}</td>
          <td class="py-2 text-right text-black">{formatRate(entry.overtime_rate)}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  @page {
    margin: 2cm;
  }

  @media print {
    .print-page {
      max-width: 100% !important;
      padding: 0 !important;
      margin: 0 !important;
    }

    /* Force B&W - remove any color */
    * {
      color: black !important;
      background: white !important;
      -webkit-print-color-adjust: exact;
      print-color-adjust: exact;
    }
  }
</style>
