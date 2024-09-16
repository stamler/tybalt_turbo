<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  import Icon from "@iconify/svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import type { BaseSystemFields } from "$lib/pocketbase-types";

  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let {
    record = $bindable(),
    errors,
    fieldName,
    uiName,
  }: {
    record: T;
    errors: Record<string, { message: string }>;
    fieldName: string;
    uiName: string;
  } = $props();

  // typeguard for BaseSystemFields to narrow the type of record
  const isBaseSystemFields = (obj: any): obj is BaseSystemFields => {
    return obj && typeof obj === "object" && "collectionId" in obj && "id" in obj;
  };
</script>

<div class="flex w-full flex-col gap-2 {errors[fieldName] !== undefined ? 'bg-red-200' : ''}">
  <span class="flex w-full gap-2">
    <label for={`file-input-${thisId}`}>{uiName}</label>
    <!-- if there is an existing attachment, display it here and provide a way
    to replace or remove it -->

    {#if isBaseSystemFields(record) && record[fieldName as keyof T]}
      <span>
        <a
          href={`${PUBLIC_POCKETBASE_URL}/api/files/${record.collectionId}/${record.id}/${record[fieldName as keyof T]}`}
          target="_blank"
        >
          {typeof record[fieldName as keyof T] === "object"
            ? record[fieldName as keyof T].name
            : record[fieldName as keyof T]}
          <Icon
            icon="bxs:file-pdf"
            width="24px"
            class="inline-block text-neutral-500 hover:text-neutral-800"
          />
        </a>
      </span>
      <button type="button" onclick={() => (record[fieldName as keyof T] = null)}>
        <Icon
          icon="feather:x-circle"
          width="24px"
          class="inline-block text-neutral-500 hover:text-neutral-800"
        />
      </button>
    {:else}
      <input
        id="attachment"
        type="file"
        onchange={(e) => (record[fieldName as keyof T] = e.target?.files[0])}
        name="attachment"
      />
    {/if}
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
