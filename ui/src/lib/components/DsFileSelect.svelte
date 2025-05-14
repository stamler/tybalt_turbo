<script context="module">
  let idCounter = $state(0);
</script>

<script lang="ts" generics="T">
  import Icon from "@iconify/svelte";
  import { PUBLIC_POCKETBASE_URL } from "$env/static/public";
  import type { BaseSystemFields } from "$lib/pocketbase-types";
  import DsFileLink from "./DsFileLink.svelte";
  import DsActionButton from "./DSActionButton.svelte";

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

  let newFileName = $state("");

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

    {#if record[fieldName as keyof T]}
      {#if isBaseSystemFields(record) && newFileName === ""}
        <!-- The file is already stored on the server, so we can link to it
        directly unless the file name is different than the original file name
        which happens when the user changes the file -->
        <span>
          <a
            href={`${PUBLIC_POCKETBASE_URL}/api/files/${record.collectionId}/${record.id}/${record[fieldName as keyof T]}`}
            target="_blank"
          >
            <DsFileLink filename={record[fieldName as keyof T] as string} />
          </a>
        </span>
      {:else}
        <!-- The file is not stored on the server, so we can show the name but
        not link to it directly -->
        <span class="flex items-center gap-1">
          {newFileName}
          <span class="flex gap-1 text-neutral-500 hover:text-neutral-800">
            {#if newFileName.toLowerCase().endsWith(".pdf")}
              <Icon icon="bxs:file-pdf" width="24px" />
            {:else if newFileName.toLowerCase().endsWith(".jpeg") || newFileName
                .toLowerCase()
                .endsWith(".jpg")}
              <Icon icon="simple-icons:jpeg" width="24px" />
            {:else if newFileName.toLowerCase().endsWith(".png")}
              <Icon icon="bxs:file-png" width="24px" />
            {/if}
            <!-- {newFileName} -->
          </span>
        </span>
      {/if}
      <!-- use the 'as any' type assertion to tell the compiler that we know
      what we're doing with the assignment -->
      <DsActionButton
        action={() => ((record as any)[fieldName as keyof T] = "")}
        icon="mdi:delete"
        title="Remove"
        color="red"
      />
    {:else}
      <input
        id="attachment"
        type="file"
        onchange={(e) => {
          const target = e.target as HTMLInputElement;
          if (target.files) {
            newFileName = target.files[0].name;
            /*
              This line produces an ownership_invalid_mutation svelte error in the
              console at runtime. It's possible this is an issue with Svelte 5, or
              perhaps we can try to reimplement this component in a way that
              binds the field itself to the file input rather than the record and
              fieldName.
            */
            (record as any)[fieldName as keyof T] = target.files[0];
          }
        }}
        name="attachment"
      />
    {/if}
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}
</div>
