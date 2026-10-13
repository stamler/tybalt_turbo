<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { invalidateAll } from "$app/navigation";
  import { resolve } from "$app/paths";
  import { workflowError } from "$lib/clientEditors";
  import { wholeCAD } from "$lib/jobRequests";
  import { globalStore } from "$lib/stores/global";
  let { data } = $props();
  // Saved limits by claim assignment. An approver without one cannot approve.
  const limits = $derived(
    Object.fromEntries(data.props.map((prop) => [prop.user_claim, prop.max_value])),
  );
  // Unsaved edits by claim assignment; a row without one shows its saved limit.
  let drafts = $state<Record<string, number | null>>({}),
    saving = $state(""),
    errors = $state<Record<string, string>>({}),
    saved = $state<Record<string, string>>({});
  const valueFor = (id: string) => (id in drafts ? drafts[id] : (limits[id] ?? null));
  const valid = (value: number | null | undefined): value is number =>
    typeof value === "number" && Number.isInteger(value) && value >= 1;
  const holderName = (holder: (typeof data.holders)[number]) =>
    `${holder.given_name} ${holder.surname}`.trim() || holder.uid;
  function edit(id: string, value: number | null) {
    drafts[id] = value;
    saved[id] = "";
    errors[id] = "";
  }
  async function save(holder: (typeof data.holders)[number], event: Event) {
    event.preventDefault();
    const id = holder.user_claim_id,
      value = valueFor(id);
    if (!valid(value)) return;
    saving = id;
    saved[id] = "";
    errors[id] = "";
    const prop = data.props.find((row) => row.user_claim === id);
    try {
      const body = { user_claim: id, max_value: value };
      if (prop) await pb.collection("project_authorization_props").update(prop.id, body);
      else await pb.collection("project_authorization_props").create(body);
      await invalidateAll();
      delete drafts[id];
      saved[id] = `Saved. ${holderName(holder)} can now approve projects up to ${wholeCAD(value)}.`;
    } catch (e) {
      errors[id] = workflowError(e);
    } finally {
      saving = "";
    }
  }
</script>

<div class="mx-auto max-w-4xl space-y-4 p-4">
  <h1 class="text-2xl font-bold">Project approval limits</h1>
  <p>
    Each person with the project_authorization claim can approve projects and value changes up to
    their limit, in whole CAD dollars before tax. Someone without a limit cannot approve.
    <a class="text-blue-700 underline" href={resolve(`/claims/${data.claimId}/details`)}
      >Assign the claim to more people</a
    >.
  </p>
  {#if $globalStore.claims.includes("admin")}{#each data.holders as holder (holder.user_claim_id)}
      {@const id = holder.user_claim_id}
      {@const current = limits[id]}
      {@const value = valueFor(id)}
      <form class="space-y-2 rounded border p-4" onsubmit={(event) => save(holder, event)}>
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <h2 class="font-semibold">{holderName(holder)}</h2>
          {#if current}<p class="text-sm">
              Current limit: <strong>{wholeCAD(current)}</strong>
            </p>{:else}<p class="text-sm text-amber-800">No limit set. Cannot approve yet.</p>{/if}
        </div>
        <div class="flex flex-wrap items-end gap-3">
          <label class="grow text-sm"
            >Maximum project value (CAD, before tax)<input
              class="mt-1 block w-full rounded border p-2"
              type="number"
              min="1"
              step="1"
              placeholder="e.g. 250000"
              required
              bind:value={() => value, (next) => edit(id, next)}
            /></label
          ><button
            type="submit"
            disabled={saving !== "" || !valid(value) || value === current}
            class="rounded bg-blue-700 px-4 py-2 text-white disabled:opacity-40"
            >{saving === id ? "Saving…" : "Save limit"}</button
          >
        </div>
        {#if valid(value) && value !== current}<p class="text-sm text-neutral-600">
            Will save as {wholeCAD(value)}.
          </p>{/if}
        {#if saved[id]}<p class="text-sm text-green-800" role="status">{saved[id]}</p>{/if}
        {#if errors[id]}<p class="text-sm text-red-800" role="alert">{errors[id]}</p>{/if}
      </form>
    {:else}<p>No one holds the project_authorization claim yet.</p>{/each}{:else}<p>
      Only an admin can maintain approval limits.
    </p>{/if}
</div>
