<script lang="ts">
  import { pb } from "$lib/pocketbase";
  import DSPopover from "./DSPopover.svelte";

  type CreatedVendor = {
    id: string;
    name: string;
    alias: string;
    status: string;
  };

  let {
    show = $bindable(),
    onCreated,
  }: {
    show: boolean;
    onCreated?: (vendor: CreatedVendor) => void;
  } = $props();

  let vendorName = $state("");
  let vendorAlias = $state("");
  let vendorSubmitting = $state(false);
  let vendorError = $state<string | null>(null);
  let wasOpen = $state(false);

  function resetForm() {
    vendorName = "";
    vendorAlias = "";
    vendorError = null;
  }

  $effect(() => {
    if (show && !wasOpen) {
      resetForm();
    }
    wasOpen = show;
  });

  function closePopover() {
    show = false;
    resetForm();
  }

  async function createVendor() {
    if (!vendorName.trim()) {
      vendorError = "Name is required";
      return;
    }

    vendorSubmitting = true;
    vendorError = null;
    try {
      const createdVendor = (await pb.collection("vendors").create({
        name: vendorName.trim(),
        alias: vendorAlias.trim(),
        status: "Active",
      })) as CreatedVendor;

      onCreated?.(createdVendor);
      closePopover();
    } catch (error: any) {
      const hookErrors = error?.data?.data;
      if (hookErrors?.name?.message) {
        vendorError = hookErrors.name.message;
      } else if (hookErrors?.alias?.message) {
        vendorError = hookErrors.alias.message;
      } else {
        vendorError = error?.message ?? "Failed to create vendor";
      }
    } finally {
      vendorSubmitting = false;
    }
  }
</script>

<DSPopover
  bind:show
  title="Add New Vendor"
  error={vendorError}
  submitting={vendorSubmitting}
  submitLabel="Create Vendor"
  onSubmit={createVendor}
  onCancel={closePopover}
>
  <div class="flex flex-col gap-1">
    <label class="text-sm font-semibold" for="new_vendor_name">Name</label>
    <input
      id="new_vendor_name"
      type="text"
      class="rounded-sm border border-neutral-300 px-2 py-1"
      placeholder="Vendor name"
      bind:value={vendorName}
      disabled={vendorSubmitting}
    />
  </div>

  <div class="flex flex-col gap-1">
    <label class="text-sm font-semibold" for="new_vendor_alias">Alias</label>
    <input
      id="new_vendor_alias"
      type="text"
      class="rounded-sm border border-neutral-300 px-2 py-1"
      placeholder="Vendor alias (optional)"
      bind:value={vendorAlias}
      disabled={vendorSubmitting}
    />
  </div>
</DSPopover>
