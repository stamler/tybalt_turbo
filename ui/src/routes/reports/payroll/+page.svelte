<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import type { PageData } from "./$types";
  import type { TimeReportWeekEndingsResponse } from "$lib/pocketbase-types";
  import { shortDate, hoursWorked, jobs, divisions, payoutRequests } from "$lib/utilities";

  let { data }: { data: PageData } = $props();
  let weekEndings = $state(data.items);

  async function fetchTimeReport(weekEnding: string, week: number) {
    try {
      // Construct the full URL
      const url = `${pb.baseUrl}/api/reports/payroll_time/${weekEnding}/${week}`;

      // Prepare headers, including Authorization if the user is logged in
      const headers: HeadersInit = {};
      if (pb.authStore.isValid) {
        headers["Authorization"] = pb.authStore.token;
      }

      // Use standard fetch API
      const response = await fetch(url, {
        method: "GET",
        headers: headers,
      });

      if (!response.ok) {
        throw new Error(`HTTP error ${response.status}: ${await response.text()}`);
      }

      // Get the response body as text
      const csvString = await response.text();

      // --- DEBUGGING (Optional: Keep for now) ---
      console.log("Type of response text:", typeof csvString);
      console.log(
        "Response value text:",
        csvString ? csvString.substring(0, 500) + "..." : "EMPTY",
      ); // Log first 500 chars
      // --- END DEBUGGING ---

      if (typeof csvString !== "string") {
        throw new Error("Received non-string response from server.");
      }

      // Create a Blob from the string response
      const timeReportCSV = new Blob([csvString], { type: "text/csv" });

      // Create an object URL from the Blob
      const blobUrl = URL.createObjectURL(timeReportCSV);
      window.open(blobUrl, "_blank");
      // Consider revoking the URL after a delay
      // setTimeout(() => URL.revokeObjectURL(blobUrl), 100);
    } catch (error) {
      console.error("Error fetching time report:", error);
      // Optionally: Show an error message to the user
    }
  }
</script>

{#snippet anchor({ week_ending }: TimeReportWeekEndingsResponse)}
  {week_ending}
{/snippet}
{#snippet headline()}
  Payroll
{/snippet}
{#snippet actions({ week_ending }: TimeReportWeekEndingsResponse)}
  Week 1
  <DsActionButton
    action={() => {
      fetchTimeReport(week_ending, 1);
    }}
    icon="mdi:download"
    title="Download"
    color="orange"
  />
{/snippet}

<!-- Show the list of items here -->
{#if weekEndings}
  <DsList items={weekEndings} search={true} {anchor} {headline} {actions} />
{/if}
