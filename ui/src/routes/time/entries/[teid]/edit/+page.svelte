<script lang="ts">
  import flatpickr from "flatpickr";
  import { onMount } from "svelte";
  import type { PageData } from "./$types";
  import { globalStore } from "$lib/stores/global";
  import { pb } from "$lib/pocketbase";
  import type { BaseAuthStore } from "pocketbase";
  import { authStore } from "$lib/stores/auth";
  import type { TimeTypesRecord, DivisionsRecord, JobsRecord, TimeEntriesRecord } from "$lib/pocketbase-types";
  import { goto } from "$app/navigation";

  let { data }: { data: PageData } = $props();

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
  const defaultItem = {
    uid: "",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    time_type: "sdyfl3q7j7ap849",
    division: "vccd5fo56ctbigh",
    description: "",
    job: "",
    work_record: "",
    hours: 0,
    meals_hours: 0,
    payout_request_amount: 0,
    week_ending: "2006-01-02",
  };
  const itemId = (data.item?.id !== undefined ? data.item.id : "") as string;
  let item = $state((itemId.length > 1 ? data.item : { ...defaultItem }) as TimeEntriesRecord);

  // given a list of time type codes, return true if the item's time type is in
  // the list
  function hasTimeType(typelist: string[]) {
    if ($globalStore.timetypes.length === 0) {
      return false;
    }
    return typelist
      .map((c) => {
        const foundTimeType = $globalStore.timetypes.find((t) => t.code === c);
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
      await pb.collection("time_entries").update(itemId, item);

      // submission was successful, clear the errors
      errors = {};

      // clear the item
      item = { ...defaultItem };
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

<form class="flex flex-col items-center w-full gap-2 p-2">
  <span class="flex flex-col w-full gap-2">
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
  <span class="flex w-full gap-2">
    <label for="timetype">Time Type</label>
    <select name="timetype" bind:value={item.time_type}>
      {#each $globalStore.timetypes as TimeTypesRecord[] as t}
        <option value={t.id} selected={t.id === item.time_type}>{t.code} - {t.name}</option>
      {/each}
    </select>
  </span>
  {#if trainingTokensInDescriptionWhileRegularHours}
    <span class="flex w-full gap-2 text-red-600 bg-red-200">
      ^Should you choose training instead?
    </span>
  {/if}

  <!----------------------------------------------->
  <!-- FIELDS VISIBLE ONLY FOR R or RT TimeTypes -->
  <!----------------------------------------------->
  {#if isWorkTime}
    <span class="flex w-full gap-2">
      <label for="division">Division</label>
      <select name="division" bind:value={item.division}>
        {#each $globalStore.divisions as DivisionsRecord[] as d}
          <option value={d.id} selected={d.id === item.division}>{d.code} - {d.name}</option>
        {/each}
      </select>
    </span>
    <span class="flex w-full gap-2">
      <label for="job">Job</label>
      <select name="job" bind:value={item.job}>
        <option value="">No Job</option>
        {#each $globalStore.jobs as JobsRecord[] as j}
          <option value={j.id} selected={j.id === item.job}>{j.number} - {j.description}</option>
        {/each}
      </select>
    </span>
  {/if}

  {#if item.job && item.job !== "" && item.division && isWorkTime}
    <div class="flex flex-col w-full gap-2 {errors.work_record !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <label for="workRecord">Work Record</label>
        <input
          class="flex-1"
          type="text"
          name="workRecord"
          placeholder="Work Record"
          bind:value={item.work_record}
        />
      </span>
      {#if errors.work_record !== undefined}
        <span class="text-red-600">{errors.work_record.message}</span>
      {/if}
    </div>
  {/if}

  <!--------------------------------------------------->
  <!-- END FIELDS VISIBLE ONLY FOR R or RT TimeTypes -->
  <!--------------------------------------------------->

  <!-- TODO: The item.job === undefined below is predecated on the
  autocomplete clearing the property. Right now we are using a text field
  so it will never show up after being set once -->
  {#if !hasTimeType(["OR", "OW", "OTO"])}
    <div class="flex flex-col w-full gap-2 {errors.hours !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <label for="hours">Hours</label>
        <input
          class="flex-1"
          type="number"
          name="hours"
          bind:value={item.hours}
          step="0.5"
          min="0"
          max="18"
        />
      </span>
      {#if errors.hours !== undefined}
        <span class="text-red-600">{errors.hours.message}</span>
      {/if}
    </div>
  {/if}

  {#if item.division && isWorkTime}
    <div class="flex flex-col w-full gap-2 {errors.meals_hours !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <label for="mealsHours">Meals Hours</label>
        <input
          class="flex-1"
          type="number"
          name="mealsHours"
          bind:value={item.meals_hours}
          step="0.5"
          min="0"
          max="2"
        />
      </span>
      {#if errors.meals_hours !== undefined}
        <span class="text-red-600">{errors.meals_hours.message}</span>
      {/if}
    </div>
  {/if}

  {#if !hasTimeType(["OR", "OW", "OTO", "RB"])}
    <div class="flex flex-col w-full gap-2 {errors.description !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <label for="description">Description</label>
        <input
          class="flex-1"
          type="text"
          name="description"
          placeholder="Description (5 char minimum)"
          bind:value={item.description}
        />
      </span>
      {#if errors.description !== undefined}
        <span class="text-red-600">{errors.description.message}</span>
      {/if}
    </div>
  {/if}
  {#if jobNumbersInDescription}
    <span class="flex w-full gap-2 text-red-600 bg-red-200">
      Job numbers are not allowed in the description. Enter jobs numbers in the appropriate field
      and create one time entry per job.
    </span>
  {/if}

  {#if hasTimeType(["OTO"])}
    <div
      class="flex flex-col w-full gap-2 {errors.payout_request_amount !== undefined
        ? 'bg-red-200'
        : ''}"
    >
      <span class="flex w-full gap-2">
        $<input
          class="flex-1"
          type="number"
          name="payoutRequestAmount"
          placeholder="Amount"
          bind:value={item.payout_request_amount}
          step="0.01"
        />
      </span>
      {#if errors.payout_request_amount !== undefined}
        <span class="text-red-600">{errors.payout_request_amount.message}</span>
      {/if}
    </div>
  {/if}

  <div class="flex flex-col w-full gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      {#if !jobNumbersInDescription}
        <button type="button" onclick={save}> Save </button>
      {/if}
      <button type="button" onclick={() => goto('/time/entries/list')}> Cancel </button>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
