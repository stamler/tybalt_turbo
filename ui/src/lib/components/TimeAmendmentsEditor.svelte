<script lang="ts">
  import { flatpickrAction, fetchCategories } from "$lib/utilities";
  import { jobs } from "$lib/stores/jobs";
  import { divisions } from "$lib/stores/divisions";
  import { timeTypes } from "$lib/stores/time_types";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsCheck from "$lib/components/DsCheck.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsAutoComplete from "./DSAutoComplete.svelte";
  import DsActionButton from "./DSActionButton.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { TimeAmendmentsPageData } from "$lib/svelte-types";
  import { onMount } from "svelte";
  import MiniSearch from "minisearch";
  import type {
    TimeTypesRecord,
    DivisionsRecord,
    CategoriesResponse,
    ProfilesResponse,
    BranchesResponse,
  } from "$lib/pocketbase-types";

  // initialize the stores, noop if already initialized
  jobs.init();
  divisions.init();
  timeTypes.init();

  let { data }: { data: TimeAmendmentsPageData } = $props();

  // index all profiles for the autocomplete
  let profilesIndex = $state(null as MiniSearch<ProfilesResponse> | null);

  onMount(async () => {
    const profiles = await pb.collection("profiles").getFullList<ProfilesResponse>({
      // filter: pb.filter('tsid=""'),
      sort: "surname,given_name",
    });
    profilesIndex = new MiniSearch<ProfilesResponse>({
      fields: ["uid", "given_name", "surname"],
      storeFields: ["uid", "given_name", "surname"],
    });
    profilesIndex.addAll(profiles as ProfilesResponse[]);
    try {
      const list = await pb.collection("branches").getFullList<BranchesResponse>({ sort: "name" });
      branches = list;
      // If branch is required and not yet set, default to the first branch
      if (((item as any).branch === undefined || (item as any).branch === "") && list.length > 0) {
        (item as any).branch = list[0].id;
      }
    } catch (e) {
      // noop
    }
  });

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
  let item = $state(data.item);

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
  let branches = $state([] as BranchesResponse[]);

  // Watch for changes to the job and fetch categories accordingly
  $effect(() => {
    if (item.job) {
      fetchCategories(item.job).then((c) => (categories = c));
    }
  });

  async function save() {
    // set the creator from the authStore We do it here rather than the
    // backend because the
    // OnRecordBeforeCreateRequest("time_amendments").Add() hook runs *AFTER*
    // schema validation (!)
    // https://github.com/pocketbase/pocketbase/discussions/2881 so we
    // need a correct value in the payload just to make it to the hook
    item.creator = $authStore?.model?.id;

    // set a dummy value for week_ending to satisfy the schema non-empty
    // requirement. This will be changed in the backend to the correct
    // value every time a record is saved
    item.week_ending = "2006-01-02";

    // if the job is empty, set the category and work_record to empty strings
    if (item.job === "") {
      item.category = "";
      item.work_record = "";
    }

    try {
      if (data.editing && data.id !== null) {
        // update the item
        await pb.collection("time_amendments").update(data.id, item);
      } else {
        // create a new item
        await pb.collection("time_amendments").create(item);
      }

      // submission was successful, clear the errors
      errors = {};

      // redirect to the pending page
      goto("/time/amendments/pending");
    } catch (error: any) {
      errors = error.data.data;
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
</svelte:head>

<form class="flex w-full flex-col items-center gap-2 p-2">
  <!-- Input the user here with an autocomplete inpux-->
  {#if profilesIndex !== null}
    <DsAutoComplete
      bind:value={item.uid as string}
      index={profilesIndex}
      {errors}
      fieldName="uid"
      uiName="Staff Member"
      idField="uid"
    >
      {#snippet resultTemplate(item)}{item.given_name} {item.surname}{/snippet}
    </DsAutoComplete>
  {/if}

  <span class="flex w-full gap-2">
    <label for="date">Date</label>
    <input
      class="flex-1"
      type="text"
      name="date"
      placeholder="Date"
      use:flatpickrAction
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
  <DsSelector
    bind:value={(item as any).branch as string}
    items={branches}
    {errors}
    fieldName="branch"
    uiName="Branch"
  >
    {#snippet optionTemplate(item: BranchesResponse)}{item.name}{/snippet}
  </DsSelector>
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

  <DsCheck
    bind:value={item.skip_tsid_check as boolean}
    {errors}
    fieldName="skip_tsid_check"
    uiName="Do not check for existing time_sheets record (don't do this unless you know what you are doing)"
  />

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      {#if !jobNumbersInDescription}
        <DsActionButton action={save}>Save</DsActionButton>
      {/if}
      <DsActionButton action="/time/amendments/pending">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
