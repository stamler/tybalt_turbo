<script lang="ts">
  // Every DS component in each variant, with sample data. Point people and
  // models here (or at this file) to see what exists before writing markup.
  import { resolve } from "$app/paths";
  import { globalStore } from "$lib/stores/global";
  import DSActionButton from "$lib/components/DSActionButton.svelte";
  import DSCallout from "$lib/components/DSCallout.svelte";
  import DSCard from "$lib/components/DSCard.svelte";
  import DSCardGrid from "$lib/components/DSCardGrid.svelte";
  import DSCollapsible from "$lib/components/DSCollapsible.svelte";
  import DSCurrencyInput from "$lib/components/DSCurrencyInput.svelte";
  import DSDateInput from "$lib/components/DSDateInput.svelte";
  import DSField from "$lib/components/DSField.svelte";
  import DSFieldList from "$lib/components/DSFieldList.svelte";
  import DSFlyout from "$lib/components/DSFlyout.svelte";
  import DSList from "$lib/components/DSList.svelte";
  import DSMenuItem from "$lib/components/DSMenuItem.svelte";
  import DSModal from "$lib/components/DSModal.svelte";
  import DSPageHeader from "$lib/components/DSPageHeader.svelte";
  import DSPopover from "$lib/components/DSPopover.svelte";
  import DSSection from "$lib/components/DSSection.svelte";
  import DSSelector from "$lib/components/DSSelector.svelte";
  import DSStatusLabel from "$lib/components/DSStatusLabel.svelte";
  import DSTabBar from "$lib/components/DSTabBar.svelte";
  import DSTextInput from "$lib/components/DSTextInput.svelte";
  import DSToggle from "$lib/components/DSToggle.svelte";
  import DSTokenInput from "$lib/components/DSTokenInput.svelte";
  import DsCheck from "$lib/components/DsCheck.svelte";
  import DsEditingDisabledBanner from "$lib/components/DsEditingDisabledBanner.svelte";
  import DsFileLink from "$lib/components/DsFileLink.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";

  const isAdmin = $derived($globalStore.claims.includes("admin"));

  const chipColors = ["yellow", "green", "blue", "red", "orange", "purple", "cyan", "neutral"];
  const labelColors = [
    "neutral",
    "blue",
    "cyan",
    "teal",
    "green",
    "yellow",
    "amber",
    "orange",
    "red",
    "purple",
  ];
  const tones = ["neutral", "info", "success", "warning", "danger"] as const;
  const calloutTones = ["info", "success", "warning", "danger"] as const;
  const statuses = {
    job: ["Active", "Awarded", "Submitted", "In Progress", "Not Awarded", "Closed", "Cancelled"],
    po: ["Active", "Unapproved", "Rejected", "Closed", "Cancelled"],
    expense: ["Approved", "Committed", "Submitted", "Rejected"],
  } as const;

  // Sample form state. Nothing here is saved.
  const errors = { sample_error: { message: "This field shows its error here." } };
  let text = $state("Sample text");
  let errorText = $state("");
  let selected = $state("b");
  const selectorItems = [
    { id: "a", name: "Option A" },
    { id: "b", name: "Option B" },
  ];
  let date = $state("2026-10-15");
  let checked = $state(true);
  let toggle = $state("week");
  let tokens = $state(["civil", "survey"]);
  let amount = $state<number | string>(1250);
  let currency = $state("");

  const listItems = [
    { id: "1", name: "First record", detail: "Byline for the first record" },
    { id: "2", name: "Second record", detail: "Byline for the second record" },
  ];

  let modalSize = $state<"sm" | "md" | "lg">("md");
  let showModal = $state(false);
  let showLockedModal = $state(false);
  let showPopover = $state(false);
  let showDangerPopover = $state(false);
  let popoverError = $state<string | null>(null);
  let flyoutOpen = $state(false);
  let lastAction = $state("none yet");

  function openModal(size: "sm" | "md" | "lg") {
    modalSize = size;
    showModal = true;
  }

  function wait(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
</script>

<div class="mx-auto space-y-8 p-4">
  <DSPageHeader title="Components" subtitle="Every shared DS component in each variant">
    {#snippet context()}
      APIs and rules: descriptions/ui_design_system.md and ui/CLAUDE.md
    {/snippet}
  </DSPageHeader>

  {#if !isAdmin}
    <DSCallout tone="danger" title="Admins only">
      This page is visible to people with the admin claim.
    </DSCallout>
  {:else}
    <DSSection title="DSPageHeader">
      <DSCardGrid>
        <DSCard title="All parts" span="full">
          <DSPageHeader title="25-123" subtitle="Highway 7 bridge rehabilitation">
            {#snippet badges()}
              <DSStatusLabel status="Active" kind="job" />
              <DsLabel color="cyan">Civil</DsLabel>
            {/snippet}
            {#snippet context()}
              Subjob of <a class="link" href={resolve("/dev/components")}>25-100</a> · from proposal P25-033
            {/snippet}
            {#snippet actions()}
              <DSActionButton icon="mdi:edit-outline" title="Edit" color="blue" action={() => {}} />
              <DSActionButton
                icon="mdi:content-copy"
                title="Copy"
                color="green"
                action={() => {}}
              />
            {/snippet}
            {#snippet menu()}
              <DSMenuItem label="Change client" icon="mdi:swap-horizontal" action={() => {}} />
              <DSMenuItem label="Fast close" icon="mdi:lock-outline" action={() => {}} />
              <DSMenuItem
                label="Delete"
                icon="mdi:delete-outline"
                tone="danger"
                action={() => {}}
              />
            {/snippet}
          </DSPageHeader>
        </DSCard>
        <DSCard title="Title only">
          <DSPageHeader title="Jane Doe" />
        </DSCard>
        <DSCard title="Menu that renders no items (no ⋯ button)" span={2}>
          <DSPageHeader title="PO 2410-0042">
            {#snippet actions()}
              <DSActionButton icon="mdi:printer" title="Print" color="yellow" action={() => {}} />
            {/snippet}
            {#snippet menu()}
              {#if false}
                <DSMenuItem label="Never shown" action={() => {}} />
              {/if}
            {/snippet}
          </DSPageHeader>
        </DSCard>
      </DSCardGrid>
    </DSSection>

    <DSSection title="DSCardGrid and DSCard">
      <DSCardGrid>
        <DSCard title="With title and actions">
          {#snippet actions()}<a class="link" href={resolve("/dev/components")}>Open profile</a
            >{/snippet}
          Card content.
        </DSCard>
        <DSCard>No title row.</DSCard>
        <DSCard title="span: 1">One column.</DSCard>
        <DSCard title="span: 2" span={2}>Two columns from md up.</DSCard>
        <DSCard title="span: full" span="full">Every column.</DSCard>
      </DSCardGrid>
    </DSSection>

    <DSSection title="DSSection">
      {#snippet actions()}
        <DSActionButton icon="mdi:plus" title="Add" color="green" action={() => {}} />
      {/snippet}
      A page-level section with an h2 and optional actions, and no box. This block is one.
    </DSSection>

    <DSSection title="DSFieldList and DSField">
      <DSCardGrid>
        <DSCard title="columns: 1">
          <DSFieldList>
            <DSField label="Plain value" value="Acme Construction" />
            <DSField label="Number" value={42} />
            <DSField label="Link" value="P25-033" href="/dev/components" />
            <DSField label="With hint" value="$125,000.00" hint="as of Oct 1, 2026" />
            <DSField label="Empty, hidden" value="" />
            <DSField label="Empty with showEmpty" value={null} showEmpty />
          </DSFieldList>
        </DSCard>
        <DSCard title="columns: 2" span={2}>
          <DSFieldList columns={2}>
            <DSField label="Manager" value="Jane Doe" />
            <DSField label="Branch" value="Toronto" />
            <DSField label="Custom value (children)">
              <DsLabel color="cyan">Civil</DsLabel>
              <DsLabel color="teal">Survey</DsLabel>
            </DSField>
            <DSField label="Fixed price" value="Yes" />
          </DSFieldList>
        </DSCard>
      </DSCardGrid>
    </DSSection>

    <DSSection title="DSCallout">
      <div class="space-y-2">
        {#each calloutTones as tone (tone)}
          <DSCallout {tone} title={`tone="${tone}"`}>Body text explaining the notice.</DSCallout>
        {/each}
        <DSCallout tone="warning" title="With actions">
          A value change is waiting for review.
          {#snippet actions()}
            <DSActionButton variant="primary" action={() => {}}>Review</DSActionButton>
          {/snippet}
        </DSCallout>
        <DSCallout tone="info" icon={false}>No title and no icon.</DSCallout>
      </div>
    </DSSection>

    <DSSection title="DsEditingDisabledBanner">
      <DsEditingDisabledBanner message="Editing is disabled while the import runs." />
    </DSSection>

    <DSSection title="DSActionButton">
      <DSCardGrid>
        <DSCard title="variant: chip (default), text">
          <div class="flex flex-wrap gap-2">
            <DSActionButton action={() => {}}>Default</DSActionButton>
            {#each chipColors as color (color)}
              <DSActionButton {color} action={() => {}}>{color}</DSActionButton>
            {/each}
          </div>
        </DSCard>
        <DSCard title="variant: chip, icon (needs icon, title, color)">
          <div class="flex flex-wrap gap-2">
            {#each chipColors as color (color)}
              <DSActionButton icon="mdi:edit-outline" title={color} {color} action={() => {}} />
            {/each}
          </div>
        </DSCard>
        <DSCard title="chip: transparentBackground, loading, disabled">
          <div class="flex flex-wrap gap-2">
            <DSActionButton
              icon="mdi:delete"
              title="Transparent"
              color="red"
              transparentBackground
              action={() => {}}
            />
            <DSActionButton loading action={() => {}}>Loading</DSActionButton>
            <DSActionButton disabled action={() => {}}>Disabled</DSActionButton>
            <DSActionButton action={() => wait(1500)}>Click: auto-loading</DSActionButton>
          </div>
        </DSCard>
        <DSCard title="primary, danger, secondary, plain">
          <div class="flex flex-wrap items-center gap-2">
            <DSActionButton variant="primary" action={() => {}}>Primary</DSActionButton>
            <DSActionButton variant="danger" action={() => {}}>Danger</DSActionButton>
            <DSActionButton variant="secondary" action={() => {}}>Secondary</DSActionButton>
            <DSActionButton variant="plain" action={() => {}}>Plain</DSActionButton>
          </div>
        </DSCard>
        <DSCard title="Filled variants: icon, loading, disabled">
          <div class="flex flex-wrap items-center gap-2">
            <DSActionButton variant="primary" icon="mdi:check" action={() => {}}>
              Approve
            </DSActionButton>
            <DSActionButton variant="primary" action={() => wait(1500)}>
              Click: auto-loading
            </DSActionButton>
            <DSActionButton variant="danger" disabled action={() => {}}>Disabled</DSActionButton>
            <DSActionButton variant="plain" icon="mdi:download" action={() => {}}>
              Download
            </DSActionButton>
          </div>
        </DSCard>
      </DSCardGrid>
    </DSSection>

    <DSSection title="DsLabel">
      <DSCardGrid>
        <DSCard title="tone (when colour means something)">
          <div class="flex flex-wrap gap-2">
            {#each tones as tone (tone)}
              <DsLabel {tone}>{tone}</DsLabel>
            {/each}
          </div>
        </DSCard>
        <DSCard title="size: sm">
          <div class="flex flex-wrap gap-2">
            {#each tones as tone (tone)}
              <DsLabel {tone} size="sm">{tone}</DsLabel>
            {/each}
          </div>
        </DSCard>
        <DSCard title="style: inverted">
          <div class="flex flex-wrap gap-2">
            {#each tones as tone (tone)}
              <DsLabel {tone} style="inverted">{tone}</DsLabel>
            {/each}
          </div>
        </DSCard>
        <DSCard title="color (categorical, no meaning)" span="full">
          <div class="flex flex-wrap gap-2">
            {#each labelColors as color (color)}
              <DsLabel {color}>{color}</DsLabel>
            {/each}
          </div>
        </DSCard>
      </DSCardGrid>
    </DSSection>

    <DSSection title="DSStatusLabel">
      <DSFieldList columns={2}>
        {#each Object.entries(statuses) as [kind, list] (kind)}
          <DSField label={`kind="${kind}"`}>
            <div class="flex flex-wrap gap-2">
              {#each list as status (status)}
                <DSStatusLabel {status} kind={kind as "job" | "po" | "expense"} />
              {/each}
            </div>
          </DSField>
        {/each}
        <DSField label="Unknown status">
          <DSStatusLabel status="Something else" />
        </DSField>
      </DSFieldList>
    </DSSection>

    <DSSection title="DSModal and DSPopover">
      <div class="flex flex-wrap gap-2">
        <DSActionButton action={() => openModal("sm")}>Modal sm</DSActionButton>
        <DSActionButton action={() => openModal("md")}>Modal md</DSActionButton>
        <DSActionButton action={() => openModal("lg")}>Modal lg</DSActionButton>
        <DSActionButton action={() => (showLockedModal = true)}>Not dismissible</DSActionButton>
        <DSActionButton action={() => (showPopover = true)}>DSPopover</DSActionButton>
        <DSActionButton action={() => (showDangerPopover = true)}>DSPopover danger</DSActionButton>
      </div>
    </DSSection>

    <DSSection title="DSFlyout and DSMenuItem">
      <div class="flex flex-wrap items-start gap-4">
        <DSFlyout label="Sample menu" bind:open={flyoutOpen}>
          {#snippet trigger({ toggle })}
            <DSActionButton action={toggle}>Menu (align start)</DSActionButton>
          {/snippet}
          <DSMenuItem label="Run an action" icon="mdi:play" action={() => (lastAction = "run")} />
          <DSMenuItem label="Go to a link" icon="mdi:link" action="/dev/components" />
          <DSMenuItem
            label="Destructive action"
            icon="mdi:delete-outline"
            tone="danger"
            action={() => (lastAction = "delete")}
          />
        </DSFlyout>
        <DSFlyout label="Sample panel" align="end">
          {#snippet trigger({ toggle, open })}
            <DSActionButton action={toggle}
              >{open ? "Close" : "Open"} panel (align end)</DSActionButton
            >
          {/snippet}
          <div class="w-64 p-3">
            <DSFieldList>
              <DSField label="Any content" value="A flyout can hold fields, lists or history." />
            </DSFieldList>
          </div>
        </DSFlyout>
      </div>
      <DSFieldList>
        <DSField label="Last menu action" value={lastAction} />
      </DSFieldList>
    </DSSection>

    <DSSection title="DSTabBar">
      <DSTabBar
        tabs={[
          { label: "Active tab", href: "/dev/components", active: true },
          { label: "Other tab", href: "/dev/components" },
          { label: "Third tab", href: "/dev/components" },
        ]}
      />
    </DSSection>

    <DSSection title="DSCollapsible">
      <DSCollapsible title="Closed by default">Hidden content.</DSCollapsible>
      <DSCollapsible title="Open" collapsed={false}>Visible content.</DSCollapsible>
    </DSSection>

    <DSSection title="Inputs">
      <DSCardGrid>
        <DSCard title="DSTextInput" span={2}>
          <DSTextInput bind:value={text} {errors} fieldName="sample" uiName="Name" />
          <DSTextInput
            bind:value={errorText}
            {errors}
            fieldName="sample_error"
            uiName="With error"
          />
        </DSCard>
        <DSCard title="DSSelector">
          <DSSelector
            bind:value={selected}
            items={selectorItems}
            {errors}
            fieldName="sample"
            uiName="Choice"
            clear
          >
            {#snippet optionTemplate(item)}{item.name}{/snippet}
          </DSSelector>
        </DSCard>
        <DSCard title="DSDateInput">
          <DSDateInput bind:value={date} />
        </DSCard>
        <DSCard title="DsCheck">
          <DsCheck bind:value={checked} {errors} fieldName="sample" uiName="Checked" />
        </DSCard>
        <DSCard title="DSToggle">
          <DSToggle
            bind:value={toggle}
            label="Period"
            options={[
              { id: "week", label: "Week" },
              { id: "month", label: "Month" },
              { id: "year", label: "Year" },
            ]}
          />
        </DSCard>
        <DSCard title="DSTokenInput">
          <DSTokenInput bind:value={tokens} {errors} fieldName="sample" uiName="Tags" />
        </DSCard>
        <DSCard title="DSCurrencyInput" span={2}>
          <DSCurrencyInput bind:amount bind:currency {errors} />
        </DSCard>
        <DSCard title="DsFileLink">
          <div class="flex gap-2">
            <DsFileLink filename="report.pdf" />
            <DsFileLink filename="photo.jpg" />
            <DsFileLink filename="scan.png" />
          </div>
        </DSCard>
      </DSCardGrid>
    </DSSection>

    <DSSection title="DSList">
      <DSList items={listItems} inListHeader="Sample list">
        {#snippet headline(item)}{item.name}{/snippet}
        {#snippet byline(item)}{item.detail}{/snippet}
        {#snippet actions()}
          <DSActionButton icon="mdi:edit-outline" title="Edit" color="blue" action={() => {}} />
        {/snippet}
      </DSList>
    </DSSection>

    <DSCallout tone="info" title="Not shown here">
      DSAutoComplete and DSSearchList need a search index, DsFileSelect needs a record, and
      DSLocationPicker, DSItemsMap and DSJobsMap load a map. See their call sites.
    </DSCallout>
  {/if}
</div>

<DSModal bind:show={showModal} title={`DSModal size="${modalSize}"`} size={modalSize}>
  <p>
    Native dialog: Escape or a backdrop click closes it, focus stays inside, and the page behind is
    inert.
  </p>
  {#snippet footer()}
    <DSActionButton variant="secondary" action={() => (showModal = false)}>Cancel</DSActionButton>
    <DSActionButton variant="primary" action={() => (showModal = false)}>Done</DSActionButton>
  {/snippet}
</DSModal>

<DSModal
  bind:show={showLockedModal}
  title="Not dismissible"
  subtitle="dismissible is false, as while submitting"
  dismissible={false}
>
  <p>Escape and backdrop clicks do nothing. Use the button.</p>
  {#snippet footer()}
    <DSActionButton variant="primary" action={() => (showLockedModal = false)}>Close</DSActionButton
    >
  {/snippet}
</DSModal>

<DSPopover
  bind:show={showPopover}
  title="DSPopover"
  subtitle="A confirm or small form dialog on DSModal"
  error={popoverError}
  submitLabel="Submit with error"
  onSubmit={() => (popoverError = "Submitting shows errors here.")}
  onCancel={() => {
    popoverError = null;
    showPopover = false;
  }}
>
  <DSTextInput bind:value={text} {errors} fieldName="sample" uiName="Name" />
</DSPopover>

<DSPopover
  bind:show={showDangerPopover}
  title="Delete this sample?"
  subtitle="danger shows a red confirm button."
  submitLabel="Delete"
  danger
  onSubmit={async () => {
    await wait(1000);
    showDangerPopover = false;
  }}
>
  <p>Nothing is deleted.</p>
</DSPopover>
