<script lang="ts">
  import { onDestroy } from "svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import { vendors } from "$lib/stores/vendors";
  import VendorCreatePopover from "./VendorCreatePopover.svelte";
  import Portal from "./Portal.svelte";

  /*
   * VendorSelector centralizes all "pick or create a vendor" behavior used by
   * PO and Expense editors.
   *
   * What it does:
   * - Renders a vendor autocomplete bound to the caller's `value`.
   * - Optionally shows a green "add vendor" action when a vendor is not selected.
   * - Opens vendor creation UI and immediately selects the newly created vendor.
   *
   * Why it exists:
   * - Removes duplicated vendor-selection and vendor-creation wiring across forms.
   * - Keeps UX and validation behavior consistent in every editor that uses vendors.
   * - Prevents regressions where modal inputs can submit the parent editor form.
   *
   * Important implementation detail:
   * - The create popover is rendered through `Portal` so it is not nested inside
   *   the parent form DOM subtree.
   * - While the popover is open, the parent form is set `inert` (with lock
   *   ref-counting) so keyboard actions cannot accidentally submit the editor.
   *
   * Future reuse:
   * - This pattern can be copied for other "inline create" selectors (clients,
   *   contacts, jobs) where we need fast creation without leaving an edit flow.
   */
  // `vendors.init()` is idempotent, so each consumer can safely mount this component.
  vendors.init();

  let {
    value = $bindable(),
    errors,
    disabled = false,
    canCreate = true,
  }: {
    value: string;
    errors: any;
    disabled?: boolean;
    canCreate?: boolean;
  } = $props();

  let showAddVendorPopover = $state(false);
  let selectorRoot = $state<HTMLDivElement | null>(null);
  let lockedForm: HTMLFormElement | null = null;
  let formLockApplied = false;

  const FORM_LOCK_COUNT_ATTR = "data-popover-lock-count";
  const FORM_INERT_OWNED_ATTR = "data-popover-inert-owned";

  // This form lock protects against accidental parent-form submits while the
  // portal popover is active. Ref counting keeps it safe if multiple overlays
  // are active on the same form.
  function lockParentForm(parentForm: HTMLFormElement | null) {
    if (!parentForm || formLockApplied) return;

    const activeLocks =
      Number.parseInt(parentForm.getAttribute(FORM_LOCK_COUNT_ATTR) ?? "0", 10) || 0;
    const nextLocks = activeLocks + 1;
    parentForm.setAttribute(FORM_LOCK_COUNT_ATTR, String(nextLocks));
    if (activeLocks === 0 && !parentForm.hasAttribute("inert")) {
      parentForm.setAttribute("inert", "");
      parentForm.setAttribute(FORM_INERT_OWNED_ATTR, "true");
    }
    lockedForm = parentForm;
    formLockApplied = true;
  }

  function unlockParentForm() {
    if (!lockedForm || !formLockApplied) return;
    const parentForm = lockedForm;

    const activeLocks =
      Number.parseInt(parentForm.getAttribute(FORM_LOCK_COUNT_ATTR) ?? "0", 10) || 0;
    const nextLocks = Math.max(activeLocks - 1, 0);
    if (nextLocks === 0) {
      parentForm.removeAttribute(FORM_LOCK_COUNT_ATTR);
      if (parentForm.getAttribute(FORM_INERT_OWNED_ATTR) === "true") {
        parentForm.removeAttribute("inert");
        parentForm.removeAttribute(FORM_INERT_OWNED_ATTR);
      }
    } else {
      parentForm.setAttribute(FORM_LOCK_COUNT_ATTR, String(nextLocks));
    }
    formLockApplied = false;
    lockedForm = null;
  }

  $effect(() => {
    const parentForm = selectorRoot?.closest("form") ?? null;

    if (showAddVendorPopover) {
      if (formLockApplied && lockedForm !== parentForm) {
        unlockParentForm();
      }
      if (!formLockApplied) {
        lockParentForm(parentForm);
      }
    } else {
      unlockParentForm();
    }
  });

  onDestroy(() => {
    unlockParentForm();
  });

  function openAddVendorPopover() {
    showAddVendorPopover = true;
  }

  function handleVendorCreated(vendor: { id: string }) {
    value = vendor.id;
  }
</script>

{#if $vendors.index !== null}
  <div class="flex w-full items-end gap-1" bind:this={selectorRoot}>
    <div class="flex-1">
      <DsAutoComplete
        bind:value={value as string}
        index={$vendors.index}
        {errors}
        fieldName="vendor"
        uiName="Vendor"
        {disabled}
      >
        {#snippet resultTemplate(item)}{item.name} ({item.alias}){/snippet}
      </DsAutoComplete>
    </div>
    {#if canCreate && !value && !disabled}
      <DsActionButton
        action={openAddVendorPopover}
        icon="feather:plus-circle"
        color="green"
        title="Add new vendor"
      />
    {/if}
  </div>
{/if}

{#if showAddVendorPopover}
  <Portal>
    <VendorCreatePopover bind:show={showAddVendorPopover} onCreated={handleVendorCreated} />
  </Portal>
{/if}
