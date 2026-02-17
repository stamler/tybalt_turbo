<script lang="ts">
  import { untrack } from "svelte";
  import {
    DATE_INPUT_MIN,
    applyDefaultDivisionOnce,
    applyDefaultRoleOnce,
    createJobCategoriesSync,
    dateInputMaxMonthsAhead,
  } from "$lib/utilities";
  import { divisions } from "$lib/stores/divisions";
  import { jobs } from "$lib/stores/jobs";
  import { rateRoles } from "$lib/stores/rateRoles";
  import { timeTypes } from "$lib/stores/time_types";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsDateInput from "$lib/components/DSDateInput.svelte";
  import DsAutoComplete from "./DSAutoComplete.svelte";
  import DsActionButton from "./DSActionButton.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { TimeEntriesPageData } from "$lib/svelte-types";
  import type { TimeTypesRecord, DivisionsRecord, CategoriesResponse } from "$lib/pocketbase-types";

  // initialize the stores, noop if already initialized
  jobs.init();
  divisions.init();
  timeTypes.init();
  rateRoles.init();

  let { data }: { data: TimeEntriesPageData } = $props();

  const trainingTokensInDescriptionWhileRegularHours = $derived.by(() => {
    if (item.time_type !== undefined && item.description !== undefined) {
      const lowercase = item.description.toLowerCase().trim();
      const lowercaseTokens = lowercase.split(/\s+/);
      return (
        hasTimeType(["R"]) &&
        (["training", "train", "orientation", "course", "whmis", "learning"].some((token) =>
          lowercaseTokens.includes(token),
        ) ||
          ["working at heights", "first aid"].some((token) => lowercase.includes(token)))
      );
    }
  });

  const jobNumbersInDescription = $derived.by(() => {
    if (item.description !== undefined) {
      const lowercase = item.description.toLowerCase().trim();
      // look for any instances of XX-YYY where XX is a number between 15 and
      // 40 and YYY is a zero-padded number between 1 and 999 then return true
      // if any are found
      return /(1[5-9]|2[0-9]|3[0-9]|40)-(\d{3})/.test(lowercase);
    }
  });

  const isWorkTime = $derived(hasTimeType(["R", "RT"]));

  let errors = $state({} as any);
  // Use untrack to capture initial value for form state (component recreates on navigation)
  let item = $state(untrack(() => data.item));
  const dateInputMax = dateInputMaxMonthsAhead(15);

  // Track allowed division IDs for the selected job (null = show all divisions, empty Set = job has no allocations)
  let jobDivisionIds = $state<Set<string> | null>(null);
  let jobHasNoAllocations = $derived(jobDivisionIds !== null && jobDivisionIds.size === 0);
  let jobAllocationsRequestId = 0;

  // Fetch job divisions when job changes
  $effect(() => {
    const jobId = item.job;
    if (!jobId || jobId === "") {
      // Invalidate any in-flight allocation request so stale responses are ignored.
      jobAllocationsRequestId++;
      jobDivisionIds = null; // No job = show all divisions
      return;
    }

    const requestId = ++jobAllocationsRequestId;
    // Fetch job details to get allocations.
    // /api/jobs/:id does not include allocations, so we must use /details.
    pb.send(`/api/jobs/${jobId}/details`, { method: "GET" })
      .then((job: { allocations?: Array<{ division?: { id?: string } | string }> }) => {
        if (requestId !== jobAllocationsRequestId) return;

        const allocationDivisionIds =
          job.allocations
            ?.map((allocation) =>
              typeof allocation.division === "string"
                ? allocation.division
                : allocation.division?.id,
            )
            .filter((id): id is string => !!id) ?? [];

        if (allocationDivisionIds.length > 0) {
          jobDivisionIds = new Set(allocationDivisionIds);
        } else {
          jobDivisionIds = new Set(); // Job has no allocations = empty set (will show no divisions)
        }
      })
      .catch(() => {
        if (requestId !== jobAllocationsRequestId) return;
        jobDivisionIds = null;
      });
  });

  // Default division from caller's profile if creating and empty
  $effect(() => applyDefaultDivisionOnce(item, data.editing));

  // Default role from caller's profile if creating, has a job, and role is empty
  $effect(() => {
    if (item.job && item.job !== "") {
      applyDefaultRoleOnce(item, data.editing);
    }
  });

  // given a list of time type codes, return true if the item's time type is in
  // the list
  function hasTimeType(typelist: string[]) {
    if ($timeTypes.items.length === 0) {
      return false;
    }
    return typelist
      .map((c) => {
        const foundTimeType = $timeTypes.items.find((t) => t.code === c);
        return foundTimeType ? foundTimeType.id : null;
      })
      .includes(item.time_type);
  }

  let categories = $state([] as CategoriesResponse[]);
  const syncCategoriesForJob = createJobCategoriesSync((rows) => {
    categories = rows;
  });

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    syncCategoriesForJob(item.job);
  });

  async function save() {
    // set the uid from the authStore We do it here rather than the
    // backend because the
    // OnRecordBeforeCreateRequest("time_entries").Add() hook runs *AFTER*
    // schema validation (!)
    // https://github.com/pocketbase/pocketbase/discussions/2881 so we
    // need a correct value in the payload just to make it to the hook
    const userId = $authStore?.model?.id;
    if (!userId) {
      errors = { global: { message: "You must be logged in to save time entries" } };
      return;
    }
    item.uid = userId;

    // set a dummy value for week_ending to satisfy the schema non-empty
    // requirement. This will be changed in the backend to the correct
    // value every time a record is saved
    item.week_ending = "2006-01-02";

    // if the job is empty, clear job-related fields
    if (item.job === "") {
      item.category = "";
      item.work_record = "";
      item.role = "";
    }

    try {
      if (data.editing && data.id !== null) {
        // update the item
        await pb.collection("time_entries").update(data.id, item);
      } else {
        // create a new item
        await pb.collection("time_entries").create(item);
      }

      // submission was successful, clear the errors
      errors = {};

      // redirect to the list page
      goto("/time/entries/list");
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<form
  class="flex w-full flex-col items-center gap-2 p-2 max-lg:[&_button]:text-base max-lg:[&_input]:text-base max-lg:[&_label]:text-base max-lg:[&_select]:text-base max-lg:[&_textarea]:text-base"
>
  <h1 class="w-full text-xl font-bold text-neutral-800">
    {#if data.editing}
      Editing Time Entry
    {:else}
      Create Time Entry
    {/if}
  </h1>

  <span class="flex w-full gap-2">
    <label for="date">Date</label>
    <DsDateInput
      class="flex-1"
      name="date"
      min={DATE_INPUT_MIN}
      max={dateInputMax}
      bind:value={item.date}
    />
  </span>
  {#snippet optionTemplate(item: TimeTypesRecord | DivisionsRecord)}
    {item.code} - {item.name}
  {/snippet}
  <DsSelector
    bind:value={item.time_type as string}
    items={$timeTypes.items}
    {errors}
    {optionTemplate}
    fieldName="time_type"
    uiName="Time Type"
  />
  {#if trainingTokensInDescriptionWhileRegularHours}
    <span class="flex w-full gap-2 bg-red-200 text-red-600">
      ^Should you choose training instead?
    </span>
  {/if}

  <!----------------------------------------------->
  <!-- FIELDS VISIBLE ONLY FOR R or RT TimeTypes -->
  <!----------------------------------------------->
  {#if isWorkTime}
    {#if $divisions.index !== null}
      <DsAutoComplete
        bind:value={item.division as string}
        index={$divisions.index}
        {errors}
        fieldName="division"
        uiName="Division"
        filter={jobDivisionIds ? (div) => jobDivisionIds!.has(div.id) : undefined}
      >
        {#snippet resultTemplate(item)}{item.code} - {item.name}{/snippet}
      </DsAutoComplete>
    {/if}
    {#if $jobs.index !== null}
      <DsAutoComplete
        bind:value={item.job as string}
        index={$jobs.index}
        {errors}
        fieldName="job"
        uiName="Job"
      >
        {#snippet resultTemplate(item)}{item.number} - {item.description}{/snippet}
      </DsAutoComplete>
    {/if}
    {#if jobHasNoAllocations}
      <span class="flex w-full gap-2 bg-red-200 text-red-600">
        This job has no division allocations configured. Contact an administrator.
      </span>
    {/if}
    {#if item.job && item.job !== "" && $rateRoles.index !== null}
      <DsAutoComplete
        bind:value={item.role as string}
        index={$rateRoles.index}
        {errors}
        fieldName="role"
        uiName="Role"
      >
        {#snippet resultTemplate(item)}{item.name}{/snippet}
      </DsAutoComplete>
    {/if}
  {/if}

  {#if item.job !== "" && categories.length > 0}
    <DsSelector
      bind:value={item.category as string}
      items={categories}
      {errors}
      fieldName="category"
      uiName="Category"
      clear={true}
    >
      {#snippet optionTemplate(item: CategoriesResponse)}
        {item.name}
      {/snippet}
    </DsSelector>
  {/if}

  {#if item.job && item.job !== "" && item.division && isWorkTime}
    <DsTextInput
      bind:value={item.work_record as string}
      {errors}
      fieldName="work_record"
      uiName="Work Record"
    />
  {/if}

  <!--------------------------------------------------->
  <!-- END FIELDS VISIBLE ONLY FOR R or RT TimeTypes -->
  <!--------------------------------------------------->

  <!-- TODO: The item.job === undefined below is predecated on the
  autocomplete clearing the property. Right now we are using a text field
  so it will never show up after being set once -->
  {#if !hasTimeType(["OR", "OW", "OTO"])}
    <DsTextInput
      bind:value={item.hours as number}
      {errors}
      fieldName="hours"
      uiName="Hours"
      type="number"
      step={0.5}
      min={0}
      max={18}
    />
  {/if}

  {#if item.division && isWorkTime}
    <DsTextInput
      bind:value={item.meals_hours as number}
      {errors}
      fieldName="meals_hours"
      uiName="Meals Hours"
      type="number"
      step={0.5}
      min={0}
      max={3}
    />
  {/if}

  {#if !hasTimeType(["OR", "OW", "OTO", "RB"])}
    <DsTextInput
      bind:value={item.description as string}
      {errors}
      fieldName="description"
      uiName="Description"
    />
  {/if}
  {#if jobNumbersInDescription}
    <span class="flex w-full gap-2 bg-red-200 text-red-600">
      Job numbers are not allowed in the description. Enter jobs numbers in the appropriate field
      and create one time entry per job.
    </span>
  {/if}

  {#if hasTimeType(["OTO"])}
    <DsTextInput
      bind:value={item.payout_request_amount as number}
      {errors}
      fieldName="payout_request_amount"
      uiName="$"
      type="number"
    />
  {/if}

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      {#if !jobNumbersInDescription}
        <DsActionButton action={save}>Save</DsActionButton>
      {/if}
      <DsActionButton action="/time/entries/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
