<script lang="ts">
  import RequestEditor from "$lib/components/jobs/RequestEditor.svelte";
  let { data } = $props();
</script>

<div class="mx-auto max-w-5xl space-y-4 p-4">
  <h1 class="text-2xl font-bold">New Project Request</h1>
  <p>Save a draft, print the agreement, then upload the signed PDF and submit the request.</p>
  <RequestEditor request={data.request} options={data.options} />
</div>
