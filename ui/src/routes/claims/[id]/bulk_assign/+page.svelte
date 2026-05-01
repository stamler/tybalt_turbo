<script lang="ts">
  import { goto, invalidateAll } from "$app/navigation";
  import DsList from "$lib/components/DSList.svelte";
  import { pb } from "$lib/pocketbase";
  import type { ClaimAssignableUser } from "$lib/svelte-types";

  let { data } = $props();

  let selectedUIDs = $state<string[]>([]);
  let saving = $state(false);
  let error = $state("");

  const selectedCount = $derived(selectedUIDs.length);

  function displayName(user: ClaimAssignableUser): string {
    const profileName = `${user.given_name} ${user.surname}`.trim();
    return profileName || user.name || user.email || user.username || user.id;
  }

  function isSelected(uid: string): boolean {
    return selectedUIDs.includes(uid);
  }

  function toggleUser(uid: string): void {
    if (isSelected(uid)) {
      selectedUIDs = selectedUIDs.filter((selectedUID) => selectedUID !== uid);
      return;
    }
    selectedUIDs = [...selectedUIDs, uid];
  }

  async function save(): Promise<void> {
    if (!data.item || selectedUIDs.length === 0 || saving) {
      return;
    }

    saving = true;
    error = "";
    try {
      await pb.send(`/api/claims/${data.item.id}/bulk_assign`, {
        method: "POST",
        body: { uids: selectedUIDs },
      });
      await invalidateAll();
      await goto(`/claims/${data.item.id}/details`);
    } catch (saveError) {
      console.error(`bulk assigning claim: ${saveError}`);
      error = "Could not assign this claim. Refresh and try again.";
    } finally {
      saving = false;
    }
  }
</script>

{#if data.error}
  <div class="p-4 text-red-600">{data.error}</div>
{:else if data.item}
  <div class="bg-neutral-100 px-3 py-2 text-sm text-neutral-600">
    {data.item.description}
  </div>
  {#if error}
    <div class="p-3 text-sm text-red-600">{error}</div>
  {/if}

  <DsList
    items={data.item.assignable_users}
    search={true}
    inListHeader="{data.item.name} users without claim"
  >
    {#snippet searchBarExtra()}
      <span class="text-sm whitespace-nowrap text-neutral-600">{selectedCount} selected</span>
      <button
        type="button"
        class="rounded-sm bg-blue-500 px-3 py-1 text-sm text-white hover:bg-blue-600 disabled:cursor-not-allowed disabled:opacity-50"
        disabled={selectedCount === 0 || saving}
        onclick={save}
      >
        {saving ? "Saving..." : "Save"}
      </button>
    {/snippet}
    {#snippet headline(user: ClaimAssignableUser)}
      {#if user.admin_profile_id}
        <a
          href={`/admin_profiles/${user.admin_profile_id}/details`}
          class="text-blue-600 hover:underline"
        >
          {displayName(user)}
        </a>
      {:else}
        <span>{displayName(user)}</span>
      {/if}
    {/snippet}
    {#snippet line1(user: ClaimAssignableUser)}
      <span class="text-sm text-neutral-600">{user.email || user.username}</span>
    {/snippet}
    {#snippet actions(user: ClaimAssignableUser)}
      <label class="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          class="h-5 w-5"
          checked={isSelected(user.id)}
          onchange={() => toggleUser(user.id)}
        />
      </label>
    {/snippet}
  </DsList>
{:else}
  <div class="p-4">Claim not found.</div>
{/if}
