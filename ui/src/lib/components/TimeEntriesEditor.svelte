<script lang="ts">
  import flatpickr from "flatpickr";
  import { onMount } from "svelte";
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsAutoComplete from "./DSAutoComplete.svelte";
  import { authStore } from "$lib/stores/auth";
  import { goto } from "$app/navigation";
  import type { TimeEntriesPageData } from "$lib/svelte-types";
  import type { JobsRecord, TimeTypesRecord, DivisionsRecord } from "$lib/pocketbase-types";

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

  let calendarInput: HTMLInputElement;
  let errors = $state({} as any);
  let item = $state(data.item);

  // given a list of time type codes, return true if the item's time type is in
  // the list
  function hasTimeType(typelist: string[]) {
    if ($globalStore.time_types.length === 0) {
      return false;
    }
    return typelist
      .map((c) => {
        const foundTimeType = $globalStore.time_types.find((t) => t.code === c);
        return foundTimeType ? foundTimeType.id : null;
      })
      .includes(item.time_type);
  }

  async function save() {
    // set the uid from the authStore We do it here rather than the
    // backend because the
    // OnRecordBeforeCreateRequest("time_entries").Add() hook runs *AFTER*
    // schema validation (!)
    // https://github.com/pocketbase/pocketbase/discussions/2881 so we
    // need a correct value in the payload just to make it to the hook
    item.uid = $authStore?.model?.id;

    // set a dummy value for week_ending to satisfy the schema non-empty
    // requirement. This will be changed in the backend to the correct
    // value every time a record is saved
    item.week_ending = "2006-01-02";

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

  // initialize flatpickr on the onMount lifecycle event
  onMount(() => {
    flatpickr(calendarInput, {
      inline: true,
      minDate: "2024-06-01",
      // 2 months from now
      maxDate: new Date(new Date().setMonth(new Date().getMonth() + 3)),
      enableTime: false,
      dateFormat: "Y-m-d",
      defaultDate: item.date,
    });
  });
</script>

<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />

<form class="flex w-full flex-col items-center gap-2 p-2">
  <span class="flex w-full flex-col gap-2">
    <label for="date">Date</label>
    <input
      class="flex-1"
      type="text"
      name="date"
      placeholder="Date"
      bind:this={calendarInput}
      bind:value={item.date}
    />
  </span>
  {#snippet optionTemplate(item: TimeTypesRecord | DivisionsRecord)}
    {item.code} - {item.name}
  {/snippet}
  <DsSelector
    bind:value={item.time_type as string}
    items={$globalStore.time_types}
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
    <DsSelector
      bind:value={item.division as string}
      items={$globalStore.divisions}
      {errors}
      {optionTemplate}
      fieldName="division"
      uiName="Division"
    />
    {#snippet jobOptionTemplate(item: JobsRecord)}
      {item.number} - {item.description}
    {/snippet}
    {#if $globalStore.jobsIndex !== null}
      <DsAutoComplete
        bind:value={item.job as string}
        index={$globalStore.jobsIndex}
        {errors}
        fieldName="job"
        uiName="Job"
      >
        {#snippet resultTemplate(item)}{item.number} - {item.description}{/snippet}
      </DsAutoComplete>
    {/if}
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
        <button
          type="button"
          onclick={save}
          class="rounded-sm bg-yellow-200 px-1 hover:bg-yellow-300"
        >
          Save
        </button>
      {/if}
      <button type="button" onclick={() => goto("/time/entries/list")}> Cancel </button>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
