<script lang="ts">
  import DsList from "$lib/components/DSList.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import { pb } from "$lib/pocketbase";
  import { onMount } from "svelte";

  interface MachineSecret {
    id: string;
    role: string;
    expiry: string;
  }

  interface NewSecretResponse {
    id: string;
    secret: string;
    expiry: string;
    role: string;
  }

  let secrets = $state<MachineSecret[]>([]);
  let loading = $state(true);
  let creating = $state(false);
  let errors = $state<Record<string, { message: string }>>({});

  // Form state
  let role = $state("legacy_writeback");
  let days = $state("30");

  // Newly created secret (shown once)
  let newSecret = $state<NewSecretResponse | null>(null);
  let copied = $state(false);

  const roleOptions = [{ id: "legacy_writeback", name: "legacy_writeback" }];

  async function fetchSecrets() {
    loading = true;
    try {
      const response = await pb.send("/api/machine_secrets/list", { method: "GET" });
      secrets = response as MachineSecret[];
    } catch (error: any) {
      console.error("Failed to fetch secrets:", error);
    } finally {
      loading = false;
    }
  }

  async function createSecret() {
    errors = {};
    const daysNum = parseInt(days, 10);

    if (isNaN(daysNum) || daysNum <= 0) {
      errors = { days: { message: "Days must be a positive number" } };
      return;
    }

    creating = true;
    try {
      const response = await pb.send("/api/machine_secrets/create", {
        method: "POST",
        body: JSON.stringify({ days: daysNum, role }),
        headers: { "Content-Type": "application/json" },
      });
      newSecret = response as NewSecretResponse;
      copied = false;
      // Refresh the list
      await fetchSecrets();
    } catch (error: any) {
      console.error("Failed to create secret:", error);
      errors = { global: { message: error?.response?.message || "Failed to create secret" } };
    } finally {
      creating = false;
    }
  }

  async function deleteSecret(id: string) {
    try {
      await pb.collection("machine_secrets").delete(id);
      // Refresh the list
      await fetchSecrets();
      // Clear the new secret display if it was the one deleted
      if (newSecret?.id === id) {
        newSecret = null;
      }
    } catch (error: any) {
      console.error("Failed to delete secret:", error);
      errors = { global: { message: error?.response?.message || "Failed to delete secret" } };
    }
  }

  async function copyToClipboard() {
    if (newSecret?.secret) {
      try {
        await navigator.clipboard.writeText(newSecret.secret);
        copied = true;
      } catch (error) {
        console.error("Failed to copy to clipboard:", error);
      }
    }
  }

  function formatExpiry(expiry: string): string {
    if (!expiry) return "";
    // Handle both ISO format and database format
    const date = new Date(expiry);
    if (isNaN(date.getTime())) return expiry;
    return date.toLocaleDateString();
  }

  function isExpired(expiry: string): boolean {
    if (!expiry) return false;
    const date = new Date(expiry);
    return date < new Date();
  }

  onMount(() => {
    fetchSecrets();
  });
</script>

<div class="flex flex-col gap-4 p-4">
  <h1 class="text-2xl font-bold">Machine Secrets</h1>

  <!-- List of existing secrets -->
  {#if loading}
    <div class="text-neutral-500">Loading...</div>
  {:else}
    <DsList items={secrets} inListHeader="Existing Secrets">
      {#snippet anchor({ id })}
        <span class="font-mono text-xs text-neutral-500">{id.slice(0, 8)}...</span>
      {/snippet}
      {#snippet headline({ role })}
        {role}
      {/snippet}
      {#snippet line1({ expiry })}
        <span class={isExpired(expiry) ? "text-red-600" : ""}>
          Expires: {formatExpiry(expiry)}
          {#if isExpired(expiry)}
            <span class="font-semibold">(expired)</span>
          {/if}
        </span>
      {/snippet}
      {#snippet actions(item)}
        <DsActionButton
          action={() => deleteSecret(item.id)}
          icon="feather:trash-2"
          title="Delete"
          color="red"
        />
      {/snippet}
    </DsList>
  {/if}

  <!-- Create new secret form -->
  <div class="rounded border border-neutral-300 bg-neutral-50 p-4">
    <h2 class="mb-3 text-lg font-semibold">Create New Secret</h2>
    <div class="flex flex-wrap items-end gap-4">
      <div class="flex flex-col gap-1">
        <label for="role-select" class="text-sm font-medium">Role</label>
        <select
          id="role-select"
          bind:value={role}
          class="rounded border border-neutral-300 px-2 py-1"
        >
          {#each roleOptions as option}
            <option value={option.id}>{option.name}</option>
          {/each}
        </select>
      </div>
      <div class="flex flex-col gap-1">
        <label for="days-input" class="text-sm font-medium">Lifetime (days)</label>
        <input
          id="days-input"
          type="number"
          bind:value={days}
          min="1"
          class="w-24 rounded border border-neutral-300 px-2 py-1 {errors.days ? 'border-red-500 bg-red-50' : ''}"
        />
        {#if errors.days}
          <span class="text-xs text-red-600">{errors.days.message}</span>
        {/if}
      </div>
      <DsActionButton action={createSecret} loading={creating} color="green">
        Create Secret
      </DsActionButton>
    </div>
    {#if errors.global}
      <div class="mt-2 text-red-600">{errors.global.message}</div>
    {/if}
  </div>

  <!-- Display newly created secret -->
  {#if newSecret}
    <div class="rounded border-2 border-green-500 bg-green-50 p-4">
      <div class="mb-2 flex items-center gap-2">
        <span class="font-semibold text-green-800">New Secret Created</span>
        <span class="text-sm text-green-600">(copy now - won't be shown again)</span>
      </div>
      <div class="flex items-center gap-2">
        <code class="flex-1 rounded bg-white px-3 py-2 font-mono text-lg">{newSecret.secret}</code>
        <DsActionButton
          action={copyToClipboard}
          icon={copied ? "feather:check" : "feather:clipboard"}
          title={copied ? "Copied!" : "Copy to clipboard"}
          color={copied ? "green" : "yellow"}
        />
      </div>
      <div class="mt-2 text-sm text-green-700">
        <span class="font-medium">Role:</span> {newSecret.role} |
        <span class="font-medium">Expires:</span> {formatExpiry(newSecret.expiry)}
      </div>
    </div>
  {/if}
</div>
