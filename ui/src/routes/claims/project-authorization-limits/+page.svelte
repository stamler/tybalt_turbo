<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import { invalidateAll } from "$app/navigation";
  import { fullName } from "$lib/jobRequests";
  import { workflowError } from "$lib/clientEditors";
  import { globalStore } from "$lib/stores/global";
  let { data } = $props();
  let values = $state<Record<string, number>>({}),
    busy = $state(false),
    error = $state("");
  $effect(() => {
    values = Object.fromEntries(
      data.claims.map((claim) => [
        claim.id,
        data.props.find((prop) => prop.user_claim === claim.id)?.max_value || 0,
      ]),
    );
  });
  async function save(id: string, event: Event) {
    event.preventDefault();
    error = "";
    busy = true;
    const prop = data.props.find((row) => row.user_claim === id);
    try {
      const body = { user_claim: id, max_value: values[id] };
      if (prop) await pb.collection("project_authorization_props").update(prop.id, body);
      else await pb.collection("project_authorization_props").create(body);
      await invalidateAll();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
</script>

<div class="mx-auto max-w-4xl space-y-4 p-4">
  <h1 class="text-2xl font-bold">Project approval limits</h1>
  <p>
    Assign the project_authorization claim through the user's profile, then set a positive
    whole-dollar CAD limit here. A claim with no valid limit grants no authority.
  </p>
  {#if error}<p class="text-red-800" role="alert">{error}</p>{/if}
  {#if $globalStore.claims.includes("admin")}{#each data.claims as claim}<form
        class="flex flex-wrap items-end gap-3 rounded border p-4"
        onsubmit={(event) => save(claim.id, event)}
      >
        <label class="grow font-semibold"
          >{fullName(data.profiles.find((profile) => profile.uid === claim.uid)) || claim.uid}<span
            class="mt-2 block text-sm font-normal">Maximum project value (CAD, before tax)</span
          ><input
            class="mt-1 w-full rounded border p-2"
            type="number"
            min="1"
            step="1"
            required
            bind:value={values[claim.id]}
          /></label
        ><button type="submit" disabled={busy} class="rounded bg-blue-700 px-4 py-2 text-white"
          >Save limit</button
        >
      </form>{:else}<p>No users hold the project_authorization claim.</p>{/each}{:else}<p>
      Only an admin can maintain approval limits.
    </p>{/if}
</div>
