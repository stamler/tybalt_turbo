<script lang="ts">
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import DsCheck from "$lib/components/DsCheck.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import DsDateInput from "$lib/components/DSDateInput.svelte";
  import DsLabel from "$lib/components/DsLabel.svelte";
  import { DATE_INPUT_MIN, dateInputMaxMonthsAhead } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";
  import type {
    AdminProfilesAugmentedSkipMinTimeCheckOptions,
    AdminProfilesAugmentedResponse,
    ClaimsResponse,
    UserClaimsResponse,
    DivisionsResponse,
    PoApproverPropsResponse,
  } from "$lib/pocketbase-types";
  import type { AdminProfilesEditPageData } from "$lib/svelte-types";
  import { goto } from "$app/navigation";
  import { onMount, untrack } from "svelte";
  import type { SearchResult } from "minisearch";
  import { divisions as divisionsStore } from "$lib/stores/divisions";
  import { branches as branchesStore } from "$lib/stores/branches";
  import { globalStore } from "$lib/stores/global";
  import { resolve } from "$app/paths";

  const PO_APPROVER_CLAIM_NAME = "po_approver";
  const HR_EDITABLE_FIELDS = [
    "active",
    "allow_personal_reimbursement",
    "default_branch",
    "default_charge_out_rate",
    "job_title",
    "mobile_phone",
    "off_rotation_permitted",
    "payroll_id",
    "personal_vehicle_insurance_expiry",
    "salary",
    "skip_min_time_check",
    "time_sheet_expected",
  ] as const;
  const TIME_OFF_MANAGER_EDITABLE_FIELDS = ["opening_date", "opening_op", "opening_ov"] as const;

  let { data }: { data: AdminProfilesEditPageData & { divisions?: DivisionsResponse[] } } =
    $props();

  let errors = $state({} as Record<string, { message: string }>);
  let item = $state(untrack(() => ({ ...data.item })));
  const dateInputMax = dateInputMaxMonthsAhead(15);

  // Use shared divisions store for items and index
  const divisions = $derived.by(() => $divisionsStore.items as DivisionsResponse[]);
  const divisionsIndex = $derived.by(() => $divisionsStore.index);
  const isAdmin = $derived($globalStore.claims.includes("admin"));
  const hasHrClaim = $derived($globalStore.claims.includes("hr"));
  const hasTimeOffManagerClaim = $derived($globalStore.claims.includes("time_off_manager"));
  const isLimitedEditor = $derived(!isAdmin && (hasHrClaim || hasTimeOffManagerClaim));
  const canEditHrFields = $derived(isAdmin || hasHrClaim);
  const canEditOpeningFields = $derived(isAdmin || hasTimeOffManagerClaim);
  const effectiveSubjectClaimIds = $derived.by(() =>
    isAdmin ? stagedClaimIds : originalUserClaims.map((uc) => uc.cid),
  );
  const availableBranches = $derived.by(() =>
    $branchesStore.items.filter((branch) => {
      if (branch.id === item.default_branch) return true;
      if (!branch.allowed_claims || branch.allowed_claims.length === 0) return true;
      return branch.allowed_claims.some((claimId) => effectiveSubjectClaimIds.includes(claimId));
    }),
  );

  let allClaims = $state([] as ClaimsResponse[]);
  let originalUserClaims = $state([] as UserClaimsResponse[]);
  let stagedClaimIds = $state([] as string[]);
  let selectedClaimId = $state("");
  let poApproverUserClaimId = $state<string | null>(null);
  let claimsLoaded = $state(false);

  const initialPoApproverMaxAmount = untrack(() =>
    normalizeNumber(data.item.po_approver_max_amount),
  );
  const initialPoApproverProjectMax = untrack(() =>
    normalizeNumber(data.item.po_approver_project_max),
  );
  const initialPoApproverSponsorshipMax = untrack(() =>
    normalizeNumber(data.item.po_approver_sponsorship_max),
  );
  const initialPoApproverStaffAndSocialMax = untrack(() =>
    normalizeNumber(data.item.po_approver_staff_and_social_max),
  );
  const initialPoApproverMediaAndEventMax = untrack(() =>
    normalizeNumber(data.item.po_approver_media_and_event_max),
  );
  const initialPoApproverComputerMax = untrack(() =>
    normalizeNumber(data.item.po_approver_computer_max),
  );
  const initialPoApproverDivisions = untrack(() =>
    normalizeDivisions(data.item.po_approver_divisions),
  );
  const initialPoApproverPropsId = untrack(() => data.item.po_approver_props_id ?? null);

  let poApproverPropsId = $state(initialPoApproverPropsId);
  let poApproverMaxAmount = $state(initialPoApproverMaxAmount);
  let poApproverProjectMax = $state(initialPoApproverProjectMax);
  let poApproverSponsorshipMax = $state(initialPoApproverSponsorshipMax);
  let poApproverStaffAndSocialMax = $state(initialPoApproverStaffAndSocialMax);
  let poApproverMediaAndEventMax = $state(initialPoApproverMediaAndEventMax);
  let poApproverComputerMax = $state(initialPoApproverComputerMax);
  let originalApproverMaxAmount = $state(initialPoApproverMaxAmount);
  let originalApproverProjectMax = $state(initialPoApproverProjectMax);
  let originalApproverSponsorshipMax = $state(initialPoApproverSponsorshipMax);
  let originalApproverStaffAndSocialMax = $state(initialPoApproverStaffAndSocialMax);
  let originalApproverMediaAndEventMax = $state(initialPoApproverMediaAndEventMax);
  let originalApproverComputerMax = $state(initialPoApproverComputerMax);
  let poApproverDivisions = $state(initialPoApproverDivisions);
  let originalApproverDivisions = $state([...initialPoApproverDivisions]);
  let poApproverDivisionsSearch = $state("");

  const poApproverDivisionsError = $derived.by(
    () => errors.divisions ?? errors.po_approver_divisions ?? null,
  );
  const hasPoApproverClaim = $derived.by(() => {
    const claimId = getPoApproverClaimId();
    return claimId !== "" && stagedClaimIds.includes(claimId);
  });

  onMount(async () => {
    await branchesStore.init();
    if (isAdmin) {
      await Promise.all([reloadAllClaims(), reloadUserClaims()]);
      return;
    }
    await reloadUserClaims();
  });

  function normalizeNumber(value: unknown): number {
    if (typeof value === "number") {
      return Number.isFinite(value) ? value : 0;
    }
    if (typeof value === "string") {
      const parsed = Number(value);
      return Number.isFinite(parsed) ? parsed : 0;
    }
    return 0;
  }

  function normalizeDivisions(value: unknown): string[] {
    if (Array.isArray(value)) {
      return value.filter((id): id is string => typeof id === "string");
    }
    if (typeof value === "string" && value.trim().startsWith("[")) {
      try {
        const parsed = JSON.parse(value);
        if (Array.isArray(parsed)) {
          return parsed.filter((id): id is string => typeof id === "string");
        }
      } catch {
        // noop
      }
    }
    return [];
  }

  async function reloadAllClaims() {
    try {
      const list = await pb.collection("claims").getFullList<ClaimsResponse>({ sort: "name" });
      allClaims = list;
    } catch {
      // noop
    }
  }

  // divisions are loaded and indexed via the shared store

  function applyPoApproverValues(
    values: {
      id?: string | null;
      max_amount?: unknown;
      project_max?: unknown;
      sponsorship_max?: unknown;
      staff_and_social_max?: unknown;
      media_and_event_max?: unknown;
      computer_max?: unknown;
      divisions?: unknown;
    } | null,
  ) {
    if (!values) {
      resetPoApproverPropsState();
      return;
    }
    poApproverPropsId = values.id ?? "";
    poApproverMaxAmount = normalizeNumber(values.max_amount);
    originalApproverMaxAmount = poApproverMaxAmount;
    poApproverProjectMax = normalizeNumber(values.project_max);
    originalApproverProjectMax = poApproverProjectMax;
    poApproverSponsorshipMax = normalizeNumber(values.sponsorship_max);
    originalApproverSponsorshipMax = poApproverSponsorshipMax;
    poApproverStaffAndSocialMax = normalizeNumber(values.staff_and_social_max);
    originalApproverStaffAndSocialMax = poApproverStaffAndSocialMax;
    poApproverMediaAndEventMax = normalizeNumber(values.media_and_event_max);
    originalApproverMediaAndEventMax = poApproverMediaAndEventMax;
    poApproverComputerMax = normalizeNumber(values.computer_max);
    originalApproverComputerMax = poApproverComputerMax;
    poApproverDivisions = normalizeDivisions(values.divisions);
    originalApproverDivisions = [...poApproverDivisions];
  }

  function applyLoadedPoApproverProps(props: PoApproverPropsResponse | null) {
    if (!props) {
      applyPoApproverValues(null);
      return;
    }
    applyPoApproverValues({
      id: props.id,
      max_amount: props.max_amount,
      project_max: props.project_max,
      sponsorship_max: props.sponsorship_max,
      staff_and_social_max: props.staff_and_social_max,
      media_and_event_max: props.media_and_event_max,
      computer_max: props.computer_max,
      divisions: props.divisions,
    });
  }

  async function resolveAugmentedPoApproverValuesByUid() {
    if (!item?.uid) return null;
    try {
      const profile = await pb
        .collection("admin_profiles_augmented")
        .getFirstListItem<AdminProfilesAugmentedResponse>(`uid="${item.uid}"`);
      const id = profile.po_approver_props_id;
      if (typeof id !== "string" || id.trim() === "") {
        return null;
      }
      return {
        id,
        max_amount: profile.po_approver_max_amount,
        project_max: profile.po_approver_project_max,
        sponsorship_max: profile.po_approver_sponsorship_max,
        staff_and_social_max: profile.po_approver_staff_and_social_max,
        media_and_event_max: profile.po_approver_media_and_event_max,
        computer_max: profile.po_approver_computer_max,
        divisions: profile.po_approver_divisions,
      };
    } catch {
      return null;
    }
  }

  async function reloadPoApproverProps() {
    if (!poApproverUserClaimId) {
      applyPoApproverValues(null);
      return;
    }

    const augmentedValues = await resolveAugmentedPoApproverValuesByUid();
    if (augmentedValues) {
      applyPoApproverValues(augmentedValues);
    } else {
      applyPoApproverValues(null);
    }
  }

  async function resolveExistingPoApproverPropsIdByUid(): Promise<string | null> {
    const values = await resolveAugmentedPoApproverValuesByUid();
    return values?.id ?? null;
  }

  async function reloadUserClaims() {
    if (!item?.uid) {
      originalUserClaims = [];
      stagedClaimIds = [];
      poApproverUserClaimId = null;
      resetPoApproverPropsState();
      claimsLoaded = true;
      return;
    }

    try {
      const list = await pb.collection("user_claims").getFullList<UserClaimsResponse>({
        filter: `uid='${item.uid}'`,
        expand: "cid",
      });
      originalUserClaims = list;
      stagedClaimIds = list.map((uc) => uc.cid);
      const poEntry = list.find((uc) => uc.expand?.cid?.name === PO_APPROVER_CLAIM_NAME);
      poApproverUserClaimId = poEntry?.id ?? poApproverUserClaimId;
      await reloadPoApproverProps();
    } catch {
      originalUserClaims = [];
      stagedClaimIds = [];
      poApproverUserClaimId = null;
      resetPoApproverPropsState();
    } finally {
      claimsLoaded = true;
    }
  }

  // index building handled by the shared store

  function divisionDisplay(
    division: SearchResult | { id: string; code?: string | null; name?: string },
  ): string {
    const code = "code" in division ? division.code?.trim() : undefined;
    const name = "name" in division ? (division.name?.trim() ?? division.id) : division.id;
    return code && code.length > 0 ? `${code} — ${name}` : name;
  }

  function divisionLabel(divisionId: string): string {
    const division = divisions.find((d) => d.id === divisionId);
    if (!division) return divisionId;
    return divisionDisplay(division);
  }

  function getPoApproverClaimId(): string {
    const claim = allClaims.find((c) => c.name === PO_APPROVER_CLAIM_NAME);
    return claim?.id ?? "";
  }

  function syncPoApproverUserClaimId(claims = originalUserClaims) {
    const claimId = getPoApproverClaimId();
    const entry = claimId !== "" ? (claims.find((uc) => uc.cid === claimId) ?? null) : null;
    poApproverUserClaimId = entry?.id ?? null;
  }

  function availableClaims(): ClaimsResponse[] {
    const assignedIds = new Set(stagedClaimIds);
    return allClaims.filter((c) => !assignedIds.has(c.id));
  }

  async function addClaimById(cid: string) {
    if (!cid) return;
    if (!stagedClaimIds.includes(cid)) {
      stagedClaimIds = [...stagedClaimIds, cid];
    }
    if (cid === getPoApproverClaimId()) {
      clearFieldError("po_approver_divisions");
    }
    selectedClaimId = "";
  }

  async function removeUserClaim(cid: string) {
    stagedClaimIds = stagedClaimIds.filter((id) => id !== cid);
    if (cid === getPoApproverClaimId()) {
      clearFieldError("po_approver_divisions");
      poApproverUserClaimId = null;
      resetPoApproverPropsState();
    }
  }

  $effect(() => {
    if (selectedClaimId !== "") {
      addClaimById(selectedClaimId);
    }
  });

  $effect(() => {
    if (!claimsLoaded) {
      return;
    }
    if (!hasPoApproverClaim) {
      resetPoApproverPropsState();
      return;
    }

    if (!poApproverUserClaimId) {
      syncPoApproverUserClaimId(originalUserClaims);
      if (poApproverUserClaimId) {
        reloadPoApproverProps();
      }
    }
  });

  function claimNameFor(cid: string): string {
    const inAll = allClaims.find((c) => c.id === cid);
    if (inAll) return inAll.name;
    const inOriginal = originalUserClaims.find((uc) => uc.cid === cid)?.expand?.cid?.name;
    return inOriginal ?? cid;
  }

  function addDivisionById(id: string | number) {
    const divisionId = id.toString();
    if (poApproverDivisions.includes(divisionId)) {
      poApproverDivisionsSearch = "";
      return;
    }

    const division = divisions.find((d) => d.id === divisionId);
    if (!division) {
      setFieldError("po_approver_divisions", "Unable to add selected division.");
      return;
    }
    if (division.active === false) {
      setFieldError("po_approver_divisions", "Only active divisions can be selected.");
      return;
    }

    poApproverDivisions = [...poApproverDivisions, divisionId];
    poApproverDivisionsSearch = "";
    clearFieldError("po_approver_divisions");
  }

  function removeDivision(divisionId: string) {
    poApproverDivisions = poApproverDivisions.filter((id) => id !== divisionId);
    if (poApproverDivisions.length === 0) {
      clearFieldError("po_approver_divisions");
    }
  }

  function setFieldError(fieldName: string, message: string) {
    errors = {
      ...errors,
      [fieldName]: { message },
    };
  }

  function clearFieldError(fieldName: string) {
    if (errors[fieldName] === undefined) return;
    const nextErrors = { ...errors };
    delete nextErrors[fieldName];
    errors = nextErrors;
  }

  function resetPoApproverPropsState() {
    poApproverMaxAmount = 0;
    originalApproverMaxAmount = 0;
    poApproverProjectMax = 0;
    originalApproverProjectMax = 0;
    poApproverSponsorshipMax = 0;
    originalApproverSponsorshipMax = 0;
    poApproverStaffAndSocialMax = 0;
    originalApproverStaffAndSocialMax = 0;
    poApproverMediaAndEventMax = 0;
    originalApproverMediaAndEventMax = 0;
    poApproverComputerMax = 0;
    originalApproverComputerMax = 0;
    poApproverDivisions = [];
    originalApproverDivisions = [];
    poApproverDivisionsSearch = "";
    poApproverPropsId = "";
  }

  function poApproverClaimLabel(): string {
    if (!hasPoApproverClaim) return "po_approver";
    if (poApproverDivisions.length === 0) {
      return "po_approver • All divisions";
    }
    const divisionsLabel =
      poApproverDivisions.length === 1
        ? `${poApproverDivisions.length} division`
        : `${poApproverDivisions.length} divisions`;
    return `po_approver • ${divisionsLabel}`;
  }

  async function persistPoApproverProps() {
    const claimId = getPoApproverClaimId();
    const hasClaim = claimId !== "" && stagedClaimIds.includes(claimId);

    if (!hasClaim) {
      if (poApproverPropsId) {
        try {
          await pb.collection("po_approver_props").delete(poApproverPropsId);
        } catch {
          // noop
        }
        resetPoApproverPropsState();
      }
      return;
    }

    let userClaimId = poApproverUserClaimId;
    if (!userClaimId) {
      const createdClaim = await pb
        .collection("user_claims")
        .create<UserClaimsResponse>({ uid: item.uid, cid: claimId });
      userClaimId = createdClaim.id;
      poApproverUserClaimId = createdClaim.id;
      originalUserClaims = [...originalUserClaims, createdClaim];
      stagedClaimIds = [...new Set([...stagedClaimIds, claimId])];
    }

    const payload = {
      user_claim: userClaimId,
      max_amount: Number.isFinite(poApproverMaxAmount) ? poApproverMaxAmount : 0,
      project_max: Number.isFinite(poApproverProjectMax) ? poApproverProjectMax : 0,
      sponsorship_max: Number.isFinite(poApproverSponsorshipMax) ? poApproverSponsorshipMax : 0,
      staff_and_social_max: Number.isFinite(poApproverStaffAndSocialMax)
        ? poApproverStaffAndSocialMax
        : 0,
      media_and_event_max: Number.isFinite(poApproverMediaAndEventMax)
        ? poApproverMediaAndEventMax
        : 0,
      computer_max: Number.isFinite(poApproverComputerMax) ? poApproverComputerMax : 0,
      divisions: poApproverDivisions,
    };

    if (poApproverPropsId) {
      const divisionsChanged =
        JSON.stringify([...poApproverDivisions].sort()) !==
        JSON.stringify([...originalApproverDivisions].sort());
      const maxChanged = poApproverMaxAmount !== originalApproverMaxAmount;
      const projectChanged = poApproverProjectMax !== originalApproverProjectMax;
      const sponsorshipChanged = poApproverSponsorshipMax !== originalApproverSponsorshipMax;
      const staffAndSocialChanged =
        poApproverStaffAndSocialMax !== originalApproverStaffAndSocialMax;
      const mediaAndEventChanged = poApproverMediaAndEventMax !== originalApproverMediaAndEventMax;
      const computerChanged = poApproverComputerMax !== originalApproverComputerMax;

      if (
        divisionsChanged ||
        maxChanged ||
        projectChanged ||
        sponsorshipChanged ||
        staffAndSocialChanged ||
        mediaAndEventChanged ||
        computerChanged
      ) {
        const updated = await pb
          .collection("po_approver_props")
          .update<PoApproverPropsResponse>(poApproverPropsId, payload);
        applyLoadedPoApproverProps(updated);
      }
    } else {
      const existingPropsId = await resolveExistingPoApproverPropsIdByUid();
      if (existingPropsId) {
        poApproverPropsId = existingPropsId;
        const updated = await pb
          .collection("po_approver_props")
          .update<PoApproverPropsResponse>(existingPropsId, payload);
        applyLoadedPoApproverProps(updated);
      } else {
        const createdProps = await pb
          .collection("po_approver_props")
          .create<PoApproverPropsResponse>(payload);
        applyLoadedPoApproverProps(createdProps);
      }
    }
  }

  function adminProfilePayload() {
    if (!isLimitedEditor) return item;
    const fieldNames = [
      ...(hasHrClaim ? HR_EDITABLE_FIELDS : []),
      ...(hasTimeOffManagerClaim ? TIME_OFF_MANAGER_EDITABLE_FIELDS : []),
    ];
    return [...new Set(fieldNames)].reduce<Record<string, unknown>>((payload, fieldName) => {
      payload[fieldName] = item[fieldName];
      return payload;
    }, {});
  }

  function errorData(error: unknown): Record<string, { message: string }> {
    if (
      typeof error === "object" &&
      error !== null &&
      "data" in error &&
      typeof error.data === "object" &&
      error.data !== null &&
      "data" in error.data &&
      typeof error.data.data === "object" &&
      error.data.data !== null
    ) {
      return error.data.data as Record<string, { message: string }>;
    }

    return {};
  }

  async function save(event: Event) {
    event.preventDefault();
    try {
      if (isLimitedEditor && (!data.editing || !data.id)) {
        errors = {
          global: { message: "This role can only update existing admin profiles." },
        };
        return;
      }

      const payload = adminProfilePayload();

      if (isLimitedEditor) {
        if (data.editing && data.id) {
          await pb.collection("admin_profiles").update(data.id, payload);
        } else {
          await pb.collection("admin_profiles").create(payload);
        }
        errors = {};
        goto(resolve("/admin_profiles/list"));
        return;
      }

      const endpoint =
        data.editing && data.id
          ? `/api/admin_profiles/${data.id}/save_with_claims`
          : "/api/admin_profiles/save_with_claims";
      await pb.send(endpoint, {
        method: "POST",
        body: {
          admin_profile: payload,
          claim_ids: stagedClaimIds,
        },
      });

      await reloadUserClaims();
      if (item.uid === (pb.authStore.model?.id ?? "")) {
        await globalStore.refresh();
      }

      await persistPoApproverProps();

      errors = {};
      goto(resolve("/admin_profiles/list"));
    } catch (error: unknown) {
      errors = errorData(error);
      if (!errors.global) {
        errors.global = { message: "Failed to save changes. Please try again." };
      }
    }
  }
</script>

<svelte:head>
  <style>
    form {
      max-width: 900px;
    }
  </style>
</svelte:head>

<form
  class="flex w-full flex-col items-center gap-2 p-2 max-lg:[&_button]:text-base max-lg:[&_input]:text-base max-lg:[&_label]:text-base max-lg:[&_select]:text-base max-lg:[&_textarea]:text-base"
  onsubmit={save}
>
  <div class="w/full grid grid-cols-1 gap-2 md:grid-cols-2">
    {#if canEditHrFields}
      <DsCheck bind:value={item.active as boolean} {errors} fieldName="active" uiName="Active" />
    {/if}

    {#if isAdmin}
      <DsTextInput
        bind:value={item.work_week_hours as number}
        {errors}
        fieldName="work_week_hours"
        uiName="Work Week Hours"
        type="number"
        step={1}
        min={0}
      />
    {/if}

    {#if canEditHrFields}
      <DsCheck bind:value={item.salary as boolean} {errors} fieldName="salary" uiName="Salary" />
      <DsCheck
        bind:value={item.off_rotation_permitted as boolean}
        {errors}
        fieldName="off_rotation_permitted"
        uiName="Off Rotation Permitted"
      />
    {/if}
    {#if isAdmin}
      <DsCheck
        bind:value={item.untracked_time_off as boolean}
        {errors}
        fieldName="untracked_time_off"
        uiName="Untracked Time Off"
      />
    {/if}
    {#if canEditHrFields}
      <DsCheck
        bind:value={item.time_sheet_expected as boolean}
        {errors}
        fieldName="time_sheet_expected"
        uiName="Time Sheet Expected"
      />
      <DsCheck
        bind:value={item.allow_personal_reimbursement as boolean}
        {errors}
        fieldName="allow_personal_reimbursement"
        uiName="Allow Personal Reimbursement"
      />

      <DsTextInput
        bind:value={item.default_charge_out_rate as number}
        {errors}
        fieldName="default_charge_out_rate"
        uiName="Default Charge Out Rate"
        type="number"
        step={0.01}
        min={0}
      />

      <DsSelector
        bind:value={item.skip_min_time_check as AdminProfilesAugmentedSkipMinTimeCheckOptions}
        items={[
          { id: "no", name: "No" },
          { id: "on_next_bundle", name: "On Next Bundle" },
          { id: "yes", name: "Yes" },
        ]}
        {errors}
        fieldName="skip_min_time_check"
        uiName="Skip Min Time Check"
      >
        {#snippet optionTemplate(item)}{item.name}{/snippet}
      </DsSelector>
    {/if}

    {#if canEditOpeningFields}
      <span class="flex w-full gap-2 {errors.opening_date !== undefined ? 'bg-red-200' : ''}">
        <label for="opening_date">Opening Date</label>
        <DsDateInput
          class="flex-1"
          name="opening_date"
          min={DATE_INPUT_MIN}
          max={dateInputMax}
          bind:value={item.opening_date}
        />
        {#if errors.opening_date !== undefined}
          <span class="text-red-600">{errors.opening_date.message}</span>
        {/if}
      </span>
    {/if}

    {#if canEditHrFields}
      <span
        class="flex w-full gap-2 {errors.personal_vehicle_insurance_expiry !== undefined
          ? 'bg-red-200'
          : ''}"
      >
        <label for="personal_vehicle_insurance_expiry">Personal Vehicle Insurance Expiry</label>
        <DsDateInput
          class="flex-1"
          name="personal_vehicle_insurance_expiry"
          min={DATE_INPUT_MIN}
          max={dateInputMax}
          bind:value={item.personal_vehicle_insurance_expiry}
        />
        {#if errors.personal_vehicle_insurance_expiry !== undefined}
          <span class="text-red-600">{errors.personal_vehicle_insurance_expiry.message}</span>
        {/if}
      </span>
    {/if}

    {#if canEditOpeningFields}
      <DsTextInput
        bind:value={item.opening_op as number}
        {errors}
        fieldName="opening_op"
        uiName="Opening OP"
        type="number"
        step={0.1}
        min={0}
      />
      <DsTextInput
        bind:value={item.opening_ov as number}
        {errors}
        fieldName="opening_ov"
        uiName="Opening OV"
        type="number"
        step={0.01}
        min={0}
      />
    {/if}

    {#if canEditHrFields}
      <DsTextInput
        bind:value={item.payroll_id as string}
        {errors}
        fieldName="payroll_id"
        uiName="Payroll ID"
      />
      <DsTextInput
        bind:value={item.mobile_phone as string}
        {errors}
        fieldName="mobile_phone"
        uiName="Mobile Phone"
      />
      <DsTextInput
        bind:value={item.job_title as string}
        {errors}
        fieldName="job_title"
        uiName="Job Title"
      />

      <DsSelector
        bind:value={item.default_branch as string}
        items={availableBranches}
        {errors}
        fieldName="default_branch"
        uiName="Default Branch"
      >
        {#snippet optionTemplate(item)}{item.name}{/snippet}
      </DsSelector>
    {/if}
  </div>

  {#if isAdmin}
    <!-- Claims section -->
    <div class="mt-4 w-full space-y-4">
      <div class="space-y-2">
        <h2 class="text-lg font-semibold">Claims</h2>
        <div class="flex flex-row flex-wrap gap-2">
          {#each stagedClaimIds as cid (cid)}
            <span class="flex items-center gap-1">
              <DsLabel color={cid === getPoApproverClaimId() ? "purple" : "cyan"}
                >{cid === getPoApproverClaimId() ? poApproverClaimLabel() : claimNameFor(cid)}
                <DsActionButton
                  transparentBackground={true}
                  title="Remove claim"
                  color="red"
                  action={() => removeUserClaim(cid)}>x</DsActionButton
                >
              </DsLabel>
            </span>
          {/each}
          {#if stagedClaimIds.length === 0}
            <span class="text-sm text-neutral-500">No claims assigned.</span>
          {/if}
        </div>

        <DsSelector
          bind:value={selectedClaimId}
          items={[{ id: "", name: "-- add claim --" }, ...availableClaims()]}
          {errors}
          fieldName="claim_to_add"
          uiName="Add Claim"
          disabled={availableClaims().length === 0}
        >
          {#snippet optionTemplate(item)}{item.name}{/snippet}
        </DsSelector>
      </div>

      {#if hasPoApproverClaim}
        <section class="space-y-3 rounded-sm border border-neutral-200 bg-neutral-50 p-3">
          <h3 class="text-base font-semibold">PO Approver Settings</h3>

          <DsTextInput
            bind:value={poApproverMaxAmount as number}
            {errors}
            fieldName="po_approver_max_amount"
            uiName="Capital Max"
            type="number"
            min={0}
            step={0.01}
          />

          <DsTextInput
            bind:value={poApproverProjectMax as number}
            {errors}
            fieldName="po_approver_project_max"
            uiName="Project Max"
            type="number"
            min={0}
            step={0.01}
          />

          <DsTextInput
            bind:value={poApproverSponsorshipMax as number}
            {errors}
            fieldName="po_approver_sponsorship_max"
            uiName="Sponsorship Max"
            type="number"
            min={0}
            step={0.01}
          />

          <DsTextInput
            bind:value={poApproverStaffAndSocialMax as number}
            {errors}
            fieldName="po_approver_staff_and_social_max"
            uiName="Staff and Social Max"
            type="number"
            min={0}
            step={0.01}
          />

          <DsTextInput
            bind:value={poApproverMediaAndEventMax as number}
            {errors}
            fieldName="po_approver_media_and_event_max"
            uiName="Media and Event Max"
            type="number"
            min={0}
            step={0.01}
          />

          <DsTextInput
            bind:value={poApproverComputerMax as number}
            {errors}
            fieldName="po_approver_computer_max"
            uiName="Computer Max"
            type="number"
            min={0}
            step={0.01}
          />

          <div class="space-y-2">
            <p class="font-semibold">Divisions for PO approval</p>
            <div class="flex flex-wrap gap-2">
              {#each poApproverDivisions as divisionId (divisionId)}
                <DsLabel color="purple">
                  {divisionLabel(divisionId)}
                  <DsActionButton
                    transparentBackground={true}
                    title="Remove division"
                    color="red"
                    action={() => removeDivision(divisionId)}
                  >
                    x
                  </DsActionButton>
                </DsLabel>
              {/each}
              {#if poApproverDivisions.length === 0}
                <span class="text-sm text-neutral-500">All divisions</span>
              {/if}
            </div>

            {#if divisionsIndex}
              <DsAutoComplete
                bind:value={poApproverDivisionsSearch}
                index={divisionsIndex}
                {errors}
                fieldName="po_approver_division"
                uiName="Add Division"
                multi={true}
                choose={addDivisionById}
              >
                {#snippet resultTemplate(option)}
                  {divisionDisplay(option)}
                {/snippet}
              </DsAutoComplete>
            {:else}
              <span class="text-sm text-neutral-500">Loading divisions…</span>
            {/if}

            {#if poApproverDivisionsError}
              <span class="text-sm text-red-600">{poApproverDivisionsError.message}</span>
            {/if}
          </div>
        </section>
      {/if}
    </div>
  {/if}

  <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
    <span class="flex w-full gap-2">
      <DsActionButton type="submit">Save</DsActionButton>
      <DsActionButton action="/admin_profiles/list">Cancel</DsActionButton>
    </span>
    {#if errors.global !== undefined}
      <span class="text-red-600">{errors.global.message}</span>
    {/if}
  </div>
</form>
