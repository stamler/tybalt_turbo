<script lang="ts">
  import TimeOffList from "$lib/components/TimeOffList.svelte";
  import type { PageData } from "./$types";
  let { data }: { data: PageData } = $props();
</script>

<TimeOffList items={data.items || []} header="All Time Off" />
