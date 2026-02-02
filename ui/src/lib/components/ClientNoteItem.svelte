<script lang="ts">
  import { formatDateTime } from "$lib/utilities";

  let {
    created,
    message,
    author,
    job,
  }: {
    created: string;
    message: string;
    author: {
      email?: string | null;
      given_name?: string | null;
      surname?: string | null;
    };
    job?: {
      id: string;
      number?: string | null;
    } | null;
  } = $props();

  function authorLabel() {
    const surname = (author.surname ?? "").trim();
    const given = (author.given_name ?? "").trim();
    if (surname || given) {
      return `${surname}${surname && given ? ", " : ""}${given}`.trim();
    }
    return (author.email ?? "Unknown").trim() || "Unknown";
  }
</script>

<li class="rounded-sm border border-neutral-200 p-3">
  <div class="flex items-center justify-between text-sm">
    <span class="font-semibold">{authorLabel()}</span>
    <span class="text-neutral-500">{formatDateTime(created)}</span>
  </div>
  <p class="mt-1 text-neutral-700">{message}</p>
  {#if job?.id && job?.number}
    <a class="text-sm text-blue-600 hover:underline" href={`/jobs/${job.id}/details`}>
      {job.number}
    </a>
  {/if}
</li>
