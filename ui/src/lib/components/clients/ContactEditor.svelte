<script lang="ts">
  import { beforeNavigate, goto } from "$app/navigation";
  import { safeReturnTo } from "$lib/clientWorkspace";
  import { untrack } from "svelte";
  import { pb } from "$lib/pocketbase";
  import { clients } from "$lib/stores/clients";
  import { globalStore } from "$lib/stores/global";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import PostalFields from "./PostalFields.svelte";
  import {
    contactFields,
    contactInUse,
    editableFields,
    validateContact,
    type FieldErrors,
  } from "$lib/clientEditors";
  import { workflowError, type WorkflowRecord } from "$lib/clientEditors";

  let {
    clientId,
    record = {},
    onSaved,
    onCancel,
    onDeleted,
    guardNavigation = true,
    returnTo,
    dirty = $bindable(false),
    busy = $bindable(false),
  }: {
    clientId: string;
    record?: WorkflowRecord;
    onSaved: (record: WorkflowRecord) => void | Promise<void>;
    onCancel: () => void;
    onDeleted?: () => void | Promise<void>;
    guardNavigation?: boolean;
    returnTo?: string;
    dirty?: boolean;
    busy?: boolean;
  } = $props();
  let item = $state(untrack(() => editableFields(record, contactFields)));
  let saved = $state(untrack(() => JSON.stringify(item)));
  let errors = $state<FieldErrors>({});
  let error = $state("");
  let savedForNavigation = false;
  const inUse = $derived(contactInUse(record));
  $effect(() => {
    dirty = JSON.stringify(item) !== saved;
  });
  beforeNavigate(({ cancel }) => {
    if (!guardNavigation) return;
    if (busy && !savedForNavigation) {
      cancel();
      return;
    }
    if (dirty && !window.confirm("Discard unsaved contact changes?")) cancel();
  });
  function cancelEdit() {
    if (dirty && !window.confirm("Discard unsaved contact changes?")) return;
    saved = JSON.stringify(item);
    dirty = false;
    onCancel();
  }
  async function save(event: Event) {
    event.preventDefault();
    if (busy || !$globalStore.claims.includes("job")) return;
    errors = validateContact(item);
    if (Object.keys(errors).length) return;
    busy = true;
    savedForNavigation = false;
    error = "";
    try {
      const body: WorkflowRecord = { ...editableFields(item, contactFields), client: clientId };
      body.email = body.email.trim();
      const result = record.id
        ? await pb.collection("client_contacts").update(record.id, body)
        : await pb.collection("client_contacts").create(body);
      saved = JSON.stringify(item);
      dirty = false;
      // Related record changes do not send a client realtime event.
      void clients.refresh(clientId).catch(() => {});
      savedForNavigation = true;
      await onSaved({ ...record, ...result });
    } catch (e: any) {
      errors = e?.data?.data || {};
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
  async function deleteContact() {
    if (busy || inUse || !$globalStore.claims.includes("job") || !record.id) return;
    if (!window.confirm("Delete this contact?")) return;
    busy = true;
    savedForNavigation = false;
    error = "";
    try {
      await pb.collection("client_contacts").delete(record.id);
      saved = JSON.stringify(item);
      dirty = false;
      void clients.refresh(clientId).catch(() => {});
      savedForNavigation = true;
      await onDeleted?.();
    } catch (e) {
      error = workflowError(e);
    } finally {
      busy = false;
    }
  }
</script>

<form class="flex min-w-0 flex-col gap-4 [&_input]:min-w-0 [&_select]:min-w-0" onsubmit={save}>
  <h2 class="text-xl font-bold">{record.id ? "Edit contact" : "Add contact"}</h2>
  {#if record.id}
    <p class="text-sm text-neutral-600">
      Used by {record.job_count || 0} jobs and {record.profile_count || 0} invoicing profiles. Changes
      apply wherever this contact is used.
    </p>
  {/if}
  {#if error}<p role="alert" class="text-red-700">{error}</p>{/if}
  <fieldset
    disabled={busy || !$globalStore.claims.includes("job")}
    class="flex min-w-0 flex-col gap-3"
  >
    <DsTextInput bind:value={item.given_name} {errors} fieldName="given_name" uiName="Given name" />
    <DsTextInput bind:value={item.surname} {errors} fieldName="surname" uiName="Surname" />
    <DsTextInput bind:value={item.email} {errors} fieldName="email" uiName="Email (optional)" />
    <DsTextInput bind:value={item.phone} {errors} fieldName="phone" uiName="Phone (optional)" />
    <details
      open={Boolean(
        record.address ||
        record.city ||
        record.province_state ||
        record.postal_code ||
        record.country,
      )}
    >
      <summary class="cursor-pointer">Postal address (optional)</summary>
      <div class="mt-3 flex flex-col gap-3"><PostalFields bind:item {errors} /></div>
    </details>
  </fieldset>
  <div class="flex flex-wrap gap-2">
    <DsActionButton type="submit" loading={busy} disabled={!$globalStore.claims.includes("job")}
      >Save contact</DsActionButton
    >
    <DsActionButton action={cancelEdit} disabled={busy}>Cancel</DsActionButton>
  </div>
  {#if record.id && onDeleted && $globalStore.claims.includes("job")}
    <details>
      <summary class="cursor-pointer">More actions</summary>
      <div class="mt-3 flex flex-wrap gap-2">
        {#if $globalStore.claims.includes("absorb")}
          <DsActionButton
            action={() => {
              const query = new URLSearchParams({ return_to: safeReturnTo(clientId, returnTo) });
              return goto(`/clients/${clientId}/contacts/${record.id}/absorb?${query}`);
            }}
            disabled={busy}>Merge contacts into this contact</DsActionButton
          >
        {/if}
        <DsActionButton action={deleteContact} disabled={busy || inUse} color="red"
          >Delete contact</DsActionButton
        >
      </div>
      {#if inUse}<p class="mt-2 text-sm text-neutral-600">
          This contact is in use and cannot be deleted.
          {#if $globalStore.claims.includes("absorb")}Use merge to combine duplicate contacts.{/if}
        </p>{/if}
    </details>
  {/if}
</form>
