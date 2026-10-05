<script lang="ts">
  let {
    canEdit,
    hasContacts,
    oncreate,
    onaddcontact,
  }: {
    canEdit: boolean;
    hasContacts: boolean;
    oncreate?: () => void;
    onaddcontact?: () => void;
  } = $props();
</script>

<p class="text-sm leading-relaxed font-normal text-neutral-600">
  This client has no invoicing profiles.
  {#if canEdit && hasContacts}
    <button
      type="button"
      class="text-sm text-blue-700 underline hover:text-blue-800"
      onclick={oncreate}>Create one</button
    > to continue.
  {:else if canEdit}
    <button
      type="button"
      class="text-sm text-blue-700 underline hover:text-blue-800"
      onclick={onaddcontact}>Add a contact</button
    > first, then create a profile.
  {:else}
    Ask a job maintainer to create an invoicing profile for this client.
  {/if}
</p>
