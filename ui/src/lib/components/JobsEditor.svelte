<script lang="ts">
  import { flatpickrAction, fetchClientContacts } from "$lib/utilities";
  import { pb } from "$lib/pocketbase";
  import DsTextInput from "$lib/components/DSTextInput.svelte";
  import DsSelector from "$lib/components/DSSelector.svelte";
  import DsAutoComplete from "$lib/components/DSAutoComplete.svelte";
  import { goto } from "$app/navigation";
  import type { JobsPageData } from "$lib/svelte-types";
  import DsActionButton from "./DSActionButton.svelte";
  import DSLocationPicker from "./DSLocationPicker.svelte";
  import DSPopover from "./DSPopover.svelte";
  import { profiles } from "$lib/stores/profiles";
  import { clients } from "$lib/stores/clients";
  import { jobs } from "$lib/stores/jobs";
  import { divisions } from "$lib/stores/divisions";
  import { rateSheets } from "$lib/stores/rateSheets";
  import { appConfig, jobsEditingEnabled } from "$lib/stores/appConfig";
  import DsCheck from "$lib/components/DsCheck.svelte";
  import { onMount, untrack } from "svelte";
  import type {
    BranchesResponse,
    ClientContactsResponse,
    DivisionsResponse,
    JobsRecord,
  } from "$lib/pocketbase-types";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { JobsStatusOptions } from "$lib/pocketbase-types";
  import { busdevLeads } from "$lib/stores/busdevLeads";
  let { data }: { data: JobsPageData } = $props();

  // initialize the stores, noop if already initialized
  clients.init();
  profiles.init();
  jobs.init();
  divisions.init();
  rateSheets.init();
  appConfig.init();

  let errors = $state({} as Record<string, { message: string }>);
  // Allow extra field `location` introduced by migration to be present on item
  let item = $state(
    untrack(() => data.item) as JobsRecord | (JobsRecord & Record<string, unknown>),
  );
  let categories = $state(untrack(() => data.categories));
  let client_contacts = $state([] as ClientContactsResponse[]);
  let clientContactsRequestId = 0;

  // Default status will be set reactively based on job type
  // For now, initialize to empty and let the $effect handle it
  const initialStatus = item.status;
  if (item.fn_agreement === undefined) {
    item.fn_agreement = false;
  }
  item.description = item.description ?? "";
  item.client = item.client ?? "";
  item.contact = item.contact ?? "";
  item.alternate_manager = item.alternate_manager ?? "";
  item.proposal = item.proposal ?? "";
  item.job_owner = item.job_owner ?? "";
  item.branch = item.branch ?? "";
  item.outstanding_balance = item.outstanding_balance ?? 0;
  item.outstanding_balance_date = item.outstanding_balance_date ?? "";
  item.authorizing_document = item.authorizing_document ?? "";
  item.client_po = item.client_po ?? "";
  item.client_reference_number = item.client_reference_number ?? "";
  item.proposal_value = item.proposal_value ?? 0;
  item.time_and_materials = item.time_and_materials ?? false;
  (item as Record<string, unknown>).rate_sheet = (item as Record<string, unknown>).rate_sheet ?? "";

  let newCategory = $state("");

  // Comment modal state for No Bid / Cancelled status changes
  let showStatusCommentModal = $state(false);
  let pendingStatus = $state<JobsStatusOptions | null>(null);
  let statusComment = $state("");
  let statusCommentError = $state<string | null>(null);
  let statusCommentSubmitting = $state(false);
  let previousStatus = $state<JobsStatusOptions | undefined>(item.status);
  let newCategories = $state([] as string[]);
  let categoriesToDelete = $state([] as string[]);

  let branches = $state([] as BranchesResponse[]);
  // Allocations editor state
  type AllocationRow = { division: string; hours: number };
  let allocations = $state([] as AllocationRow[]);

  // Add Client popover state
  let showAddClientPopover = $state(false);
  let newClientName = $state("");
  let newClientBusdevLead = $state("");
  let newClientSubmitting = $state(false);
  let newClientError = $state<string | null>(null);

  // Add Contact popover state
  let showAddContactPopover = $state(false);
  let newContactGivenName = $state("");
  let newContactSurname = $state("");
  let newContactEmail = $state("");
  let newContactSubmitting = $state(false);
  let newContactError = $state<string | null>(null);

  // Initialize busdevLeads store for the Add Client popover
  busdevLeads.init();

  const alternateManagerErrorMessage = "Alternate manager must be different from manager.";

  // Determine if this is a proposal based on number prefix or proposal dates
  const isProposal = $derived.by(() => {
    // If number starts with P, it's a proposal
    if (item.number?.startsWith("P")) return true;
    // For new jobs, check if proposal dates are set but no project award date
    const projectAwardDate = item.project_award_date ?? "";
    const proposalOpeningDate = item.proposal_opening_date ?? "";
    const proposalSubmissionDueDate = item.proposal_submission_due_date ?? "";
    if (
      projectAwardDate === "" &&
      (proposalOpeningDate !== "" || proposalSubmissionDueDate !== "")
    ) {
      return true;
    }
    return false;
  });

  // Check if this is a cancelled proposal (terminal state - no edits allowed)
  const isCancelledProposal = $derived(isProposal && item.status === JobsStatusOptions.Cancelled);

  // Status options filtered by job type
  // New proposals can only be "In Progress" or "Submitted" (no ID yet for comment-requiring statuses)
  const newProposalStatuses = [JobsStatusOptions["In Progress"], JobsStatusOptions.Submitted];
  const existingProposalStatuses = [
    JobsStatusOptions["In Progress"],
    JobsStatusOptions.Submitted,
    JobsStatusOptions.Awarded,
    JobsStatusOptions["Not Awarded"],
    JobsStatusOptions.Cancelled,
    JobsStatusOptions["No Bid"],
  ];
  const projectStatuses = [
    JobsStatusOptions.Active,
    JobsStatusOptions.Closed,
    JobsStatusOptions.Cancelled,
  ];

  const isNewJob = $derived(data.id === null);

  const statusOptionsList = $derived.by(() => {
    let statuses: JobsStatusOptions[];
    if (isProposal) {
      statuses = isNewJob ? newProposalStatuses : existingProposalStatuses;
    } else {
      statuses = projectStatuses;
    }
    return statuses.map((status) => ({ id: status, name: status }));
  });

  const authorizingDocumentOptions = [
    { id: "Unauthorized", name: "Unauthorized" },
    { id: "PO", name: "PO" },
    { id: "PA", name: "PA" },
  ];

  function jobLabel(job: Pick<JobApiResponse, "id" | "number" | "description">) {
    const numberPart = job.number?.trim();
    const descriptionPart = job.description?.trim();

    const displayNumber = numberPart && numberPart.length > 0 ? numberPart : job.id;
    const displayDescription =
      descriptionPart && descriptionPart.length > 0 ? ` — ${descriptionPart}` : "";

    return `${displayNumber}${displayDescription}`;
  }

  // Filter functions for autocomplete - applied to search results, not indices
  const proposalFilter = (job: JobApiResponse) => job.number?.startsWith("P") ?? false;
  const activeRateSheetFilter = (sheet: { active: boolean }) => sheet.active;

  // Hide proposal dates when:
  // 1. Creating a project from the dedicated route (loader sets _prefilled_from_proposal)
  // 2. Project award date has a value (job is a project)
  const hideProposalDates = $derived.by(
    () =>
      Boolean((item as unknown as Record<string, unknown>)._prefilled_from_proposal) ||
      Boolean(item.project_award_date),
  );

  // Hide project award date when either proposal date has a value (job is a proposal)
  const hideProjectDate = $derived.by(
    () => Boolean(item.proposal_opening_date) || Boolean(item.proposal_submission_due_date),
  );

  function setFieldError(fieldName: string, message: string) {
    if (errors[fieldName]?.message === message) {
      return;
    }
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

  function addAllocationRow() {
    allocations = [...allocations, { division: "", hours: 0 }];
  }
  function removeAllocationRow(index: number) {
    allocations = allocations.filter((_, i) => i !== index);
  }
  function setAllocationDivision(index: number, value: string | number) {
    const id = value.toString();
    // prevent duplicate divisions
    if (allocations.some((a, i) => i !== index && a.division === id)) {
      return;
    }
    allocations = allocations.map((row, i) => (i === index ? { ...row, division: id } : row));
  }

  onMount(async () => {
    try {
      branches = await pb.collection("branches").getFullList<BranchesResponse>({ sort: "name" });
      // Load allocations when editing
      if ((data as JobsPageData).editing && (data as JobsPageData).id) {
        const list = await pb
          .collection("job_time_allocations")
          .getFullList<{ id: string; job: string; division: string; hours: number }>({
            filter: `job="${(data as JobsPageData).id}"`,
          });
        allocations = list.map((r) => ({ division: r.division, hours: r.hours ?? 0 }));

        // Check for stored validation errors from "Create referencing project" flow
        if (typeof sessionStorage !== "undefined") {
          const storageKey = `proposal_validation_errors_${(data as JobsPageData).id}`;
          const storedErrors = sessionStorage.getItem(storageKey);
          if (storedErrors) {
            try {
              const parsedErrors = JSON.parse(storedErrors);
              // Convert from { field: { code, message } } to { field: { message } }
              const formattedErrors: Record<string, { message: string }> = {};
              for (const [field, error] of Object.entries(parsedErrors)) {
                if (error && typeof error === "object" && "message" in error) {
                  formattedErrors[field] = { message: (error as { message: string }).message };
                }
              }
              errors = formattedErrors;
            } catch (e) {
              console.error("Failed to parse stored validation errors", e);
            }
            // Clear the stored errors after reading
            sessionStorage.removeItem(storageKey);
          }
        }
      }
    } catch (error) {
      console.error("Failed to load branches", error);
    }
  });

  function formatContactName(contact: ClientContactsResponse) {
    const surname = contact.surname?.trim();
    const given = contact.given_name?.trim();
    if (surname && given) return `${surname}, ${given}`;
    if (surname) return surname;
    if (given) return given;
    return contact.email ?? contact.id;
  }

  // Watch for changes to the client and fetch contacts accordingly
  // When the client changes, fetch its contacts. The requestId guard ensures that
  // only the latest fetch updates the state, so rapid client switches don't apply
  // stale contact lists.
  $effect(() => {
    const clientId = item.client;
    const requestId = ++clientContactsRequestId;

    if (!clientId) {
      client_contacts = [];
      if (item.contact) {
        item.contact = "";
      }
      return;
    }

    fetchClientContacts(clientId)
      .then((contacts) => {
        if (requestId !== clientContactsRequestId) return;
        client_contacts = contacts;
        const hasSelectedContact = contacts.some((contact) => contact.id === item.contact);
        if (!hasSelectedContact) {
          item.contact = "";
        }
      })
      .catch((error) => {
        console.error("Failed to load client contacts", error);
        if (requestId !== clientContactsRequestId) return;
        client_contacts = [];
        if (item.contact) {
          item.contact = "";
        }
      });
  });

  $effect(() => {
    if (!item.manager || !item.alternate_manager) {
      if (errors.alternate_manager?.message === alternateManagerErrorMessage) {
        clearFieldError("alternate_manager");
      }
      return;
    }
    if (item.manager === item.alternate_manager) {
      setFieldError("alternate_manager", alternateManagerErrorMessage);
    } else if (errors.alternate_manager?.message === alternateManagerErrorMessage) {
      clearFieldError("alternate_manager");
    }
  });

  // Mirror backend behavior: when not PO, clear any provided client_po
  $effect(() => {
    if (item.authorizing_document !== "PO" && item.client_po) {
      item.client_po = "";
      if (errors.client_po) clearFieldError("client_po");
    }
  });

  // Set default status based on job type, and correct status if it becomes invalid
  // when job type changes (e.g., entering proposal dates changes project to proposal)
  $effect(() => {
    const validStatuses = isProposal
      ? isNewJob
        ? newProposalStatuses
        : existingProposalStatuses
      : projectStatuses;
    const currentStatusIsValid =
      item.status && validStatuses.includes(item.status as JobsStatusOptions);

    if (!currentStatusIsValid) {
      // Status is missing or invalid for current job type - set appropriate default
      item.status = isProposal ? JobsStatusOptions["In Progress"] : JobsStatusOptions.Active;
    }
  });

  // Watch for status changes to No Bid or Cancelled for proposals - require comment first
  $effect(() => {
    const currentStatus = item.status;
    if (!isProposal) {
      previousStatus = currentStatus;
      return;
    }

    const requiresComment =
      currentStatus === JobsStatusOptions["No Bid"] ||
      currentStatus === JobsStatusOptions.Cancelled;

    // If status changed to one that requires a comment, show modal and revert
    if (requiresComment && currentStatus !== previousStatus && !showStatusCommentModal) {
      pendingStatus = currentStatus;
      // Revert to previous status until comment is added
      if (previousStatus !== undefined) {
        item.status = previousStatus;
      }
      showStatusCommentModal = true;
      statusComment = "";
      statusCommentError = null;
    } else if (!requiresComment) {
      previousStatus = currentStatus;
    }
  });

  async function submitStatusComment() {
    if (!statusComment.trim()) {
      statusCommentError = "A comment is required";
      return;
    }
    if (!pendingStatus || !data.id || !item.client) {
      statusCommentError = "Unable to add comment - missing job or client data";
      return;
    }

    statusCommentSubmitting = true;
    statusCommentError = null;

    try {
      // Create the client note with the status change
      await pb.collection("client_notes").create({
        client: item.client,
        job: data.id,
        note: statusComment,
        job_status_changed_to: pendingStatus,
      });

      // Now set the status
      item.status = pendingStatus;
      previousStatus = pendingStatus;
      pendingStatus = null;
      showStatusCommentModal = false;
      statusComment = "";
    } catch (error: any) {
      statusCommentError = error?.response?.message ?? "Failed to add comment";
    } finally {
      statusCommentSubmitting = false;
    }
  }

  function cancelStatusChange() {
    showStatusCommentModal = false;
    pendingStatus = null;
    statusComment = "";
    statusCommentError = null;
  }

  async function save(event: Event) {
    event.preventDefault();

    errors = {};
    if (item.manager && item.alternate_manager && item.manager === item.alternate_manager) {
      setFieldError("alternate_manager", alternateManagerErrorMessage);
      return;
    }

    try {
      let jobId = data.id;

      if (data.editing && jobId !== null) {
        await pb.send(`/api/jobs/${jobId}`, {
          method: "PUT",
          body: { job: item, allocations },
        });
      } else {
        const resp = (await pb.send(`/api/jobs`, {
          method: "POST",
          body: { job: item, allocations },
        })) as { id: string };
        jobId = resp.id;
      }

      // Add new categories
      for (const categoryName of newCategories) {
        await pb.collection("categories").create(
          {
            job: jobId,
            name: categoryName.trim(),
          },
          { returnRecord: true },
        );
      }

      // Remove deleted categories
      for (const categoryId of categoriesToDelete) {
        await pb.collection("categories").delete(categoryId);
      }

      errors = {};
      // Check if we should redirect to create project page after fixing proposal errors
      if (data.editing && data.id !== null) {
        const redirectKey = `redirect_to_create_project_${data.id}`;
        const shouldRedirectToCreateProject =
          typeof sessionStorage !== "undefined" && sessionStorage.getItem(redirectKey) === "true";

        if (shouldRedirectToCreateProject) {
          sessionStorage.removeItem(redirectKey);
          // Also remove any stored validation errors
          sessionStorage.removeItem(`proposal_validation_errors_${data.id}`);
          goto(`/jobs/add/from/${data.id}?setAwardToday=true`);
        } else {
          goto("/jobs/list");
        }
      } else {
        goto(`/jobs/${jobId}/details`);
      }
    } catch (error: unknown) {
      // Handle special case where backend requires setting proposal to Awarded first
      const pocket = error as {
        data?: {
          data?: Record<string, { message: string; code?: string; data?: Record<string, string> }>;
        };
      };
      const hookErrors = pocket?.data?.data;
      const proposalErr = hookErrors?.proposal;
      const proposalId = proposalErr?.data?.proposal_id;
      if (proposalErr?.code === "proposal_not_awarded" && typeof proposalId === "string") {
        const proceed =
          typeof window !== "undefined" &&
          window.confirm(
            "The referenced proposal is not marked as Awarded. Set it to Awarded and continue?",
          );
        if (proceed) {
          // First, try to update the proposal to Awarded status
          try {
            await pb.collection("jobs").update(proposalId, { status: JobsStatusOptions.Awarded });
          } catch (proposalUpdateErr) {
            // Handle proposal update failure - show the actual error with link to edit proposal
            const proposalErrData = (
              proposalUpdateErr as {
                data?: { data?: Record<string, { message: string }> };
              }
            )?.data?.data;

            // Build error message with details from backend
            let errorDetails = "";
            if (proposalErrData) {
              errorDetails = Object.values(proposalErrData)
                .map((v) => v.message)
                .join("; ");
            }

            // Set a special error on the proposal field that includes a link
            errors = {
              proposal: {
                message: `Cannot set proposal to Awarded: ${errorDetails || "validation failed"}. Please edit the proposal to fix these issues.`,
                // Store the proposal ID for the link
                proposalId: proposalId,
              } as { message: string; proposalId?: string },
            };
            return;
          }

          // Proposal was successfully updated to Awarded, now retry the job creation
          try {
            let retryJobId = (data as JobsPageData).id;
            if ((data as JobsPageData).editing && retryJobId !== null) {
              await pb.send(`/api/jobs/${retryJobId}`, {
                method: "PUT",
                body: { job: item, allocations },
              });
            } else {
              const resp = (await pb.send(`/api/jobs`, {
                method: "POST",
                body: { job: item, allocations },
              })) as { id: string };
              retryJobId = resp.id;
            }
            // continue categories changes as usual
            for (const categoryName of newCategories) {
              await pb
                .collection("categories")
                .create({ job: retryJobId!, name: categoryName.trim() }, { returnRecord: true });
            }
            for (const categoryId of categoriesToDelete) {
              await pb.collection("categories").delete(categoryId);
            }
            errors = {};
            // Redirect to job details for new jobs, jobs list for edits
            if ((data as JobsPageData).editing && (data as JobsPageData).id !== null) {
              goto("/jobs/list");
            } else {
              goto(`/jobs/${retryJobId}/details`);
            }
            return;
          } catch (retryErr) {
            // fall through to display errors from retry
            const retryData = (
              retryErr as {
                data?: { data?: Record<string, { message: string }> };
              }
            )?.data?.data;
            errors = retryData ?? {};
            return;
          }
        }
      }
      const backendErrors = pocket?.data?.data as Record<string, { message: string }> | undefined;
      errors = backendErrors ?? {};
    }
  }

  async function addCategory() {
    if (newCategory.trim() === "") return;

    newCategories.push(newCategory.trim());
    newCategory = "";
  }

  async function removeCategory(categoryId: string) {
    categoriesToDelete.push(categoryId);
    categories = categories.filter((category) => category.id !== categoryId);
  }

  function preventDefault(fn: (event: Event) => void) {
    return (event: Event) => {
      event.preventDefault();
      fn(event);
    };
  }

  function cancel() {
    // Clean up any redirect flags if user cancels
    if (data.editing && data.id !== null && typeof sessionStorage !== "undefined") {
      sessionStorage.removeItem(`redirect_to_create_project_${data.id}`);
      sessionStorage.removeItem(`proposal_validation_errors_${data.id}`);
    }
    if (data.editing && data.id !== null) {
      goto(`/jobs/${data.id}/details`);
    } else {
      goto("/jobs/list");
    }
  }

  function openAddClientPopover() {
    newClientName = "";
    newClientBusdevLead = "";
    newClientError = null;
    showAddClientPopover = true;
  }

  function closeAddClientPopover() {
    showAddClientPopover = false;
    newClientName = "";
    newClientBusdevLead = "";
    newClientError = null;
  }

  async function createNewClient() {
    if (!newClientName.trim()) {
      newClientError = "Name is required";
      return;
    }
    if (!newClientBusdevLead.trim()) {
      newClientError = "Business Development Lead is required";
      return;
    }

    newClientSubmitting = true;
    newClientError = null;

    try {
      const createdClient = await pb.collection("clients").create({
        name: newClientName.trim(),
        business_development_lead: newClientBusdevLead,
      });

      // Select the newly created client
      // The realtime subscription will automatically update the store/index
      item.client = createdClient.id;

      closeAddClientPopover();
    } catch (error: any) {
      const hookErrors = error?.data?.data;
      if (hookErrors?.name?.message) {
        newClientError = hookErrors.name.message;
      } else if (hookErrors?.business_development_lead?.message) {
        newClientError = hookErrors.business_development_lead.message;
      } else {
        newClientError = error?.message ?? "Failed to create client";
      }
    } finally {
      newClientSubmitting = false;
    }
  }

  function openAddContactPopover() {
    newContactGivenName = "";
    newContactSurname = "";
    newContactEmail = "";
    newContactError = null;
    showAddContactPopover = true;
  }

  function closeAddContactPopover() {
    showAddContactPopover = false;
    newContactGivenName = "";
    newContactSurname = "";
    newContactEmail = "";
    newContactError = null;
  }

  // Get the selected client's name for display in the Add Contact popover
  const selectedClientName = $derived.by(() => {
    if (!item.client || !$clients.items) return "";
    const client = $clients.items.find((c: { id: string }) => c.id === item.client);
    return client?.name ?? "";
  });

  async function createNewContact() {
    if (!newContactGivenName.trim() && !newContactSurname.trim()) {
      newContactError = "At least given name or surname is required";
      return;
    }
    if (!item.client) {
      newContactError = "No client selected";
      return;
    }

    newContactSubmitting = true;
    newContactError = null;

    try {
      const createdContact = await pb.collection("client_contacts").create({
        client: item.client,
        given_name: newContactGivenName.trim(),
        surname: newContactSurname.trim(),
        email: newContactEmail.trim(),
      });

      // Add the new contact to the local client_contacts list
      client_contacts = [...client_contacts, createdContact];

      // Select the newly created contact
      item.contact = createdContact.id;

      closeAddContactPopover();
    } catch (error: any) {
      const hookErrors = error?.data?.data;
      if (hookErrors?.given_name?.message) {
        newContactError = hookErrors.given_name.message;
      } else if (hookErrors?.surname?.message) {
        newContactError = hookErrors.surname.message;
      } else if (hookErrors?.email?.message) {
        newContactError = hookErrors.email.message;
      } else {
        newContactError = error?.message ?? "Failed to create contact";
      }
    } finally {
      newContactSubmitting = false;
    }
  }
</script>

<svelte:head>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/flatpickr/dist/flatpickr.min.css" />
</svelte:head>

{#if !$jobsEditingEnabled}
  <div class="disabled-notice">
    <p>Job creation and editing is temporarily disabled during a system transition.</p>
    <p>Please check back later or contact an administrator if you need immediate assistance.</p>
  </div>
{:else if isCancelledProposal}
  <div class="disabled-notice">
    <p>This proposal has been cancelled and cannot be modified.</p>
    <p>Cancelled proposals are in a terminal state. No further changes are allowed.</p>
    <div class="mt-4">
      <a href="/jobs/{data.id}/details" class="text-blue-600 hover:underline"
        >← Back to job details</a
      >
    </div>
  </div>
{:else}
  <form
    class="flex w-full flex-col items-center gap-2 p-2"
    enctype="multipart/form-data"
    onsubmit={save}
  >
    <h1 class="w-full text-xl font-bold text-neutral-800">
      {#if data.editing && item.number}
        Editing {item.number}
      {:else}
        Create Job
      {/if}
    </h1>

    {#if (data as JobsPageData).existingReferencingProjects && (data as JobsPageData).existingReferencingProjects!.length > 0}
      <div class="w-full rounded-sm border border-yellow-400 bg-yellow-50 p-3 text-yellow-800">
        <p class="font-semibold">This proposal is already referenced by existing project(s):</p>
        <ul class="mt-1 list-inside list-disc">
          {#each (data as JobsPageData).existingReferencingProjects! as project}
            <li>
              <a
                href="/jobs/{project.id}/details"
                class="text-blue-600 underline hover:text-blue-800"
              >
                {project.number}
              </a>
            </li>
          {/each}
        </ul>
        <p class="mt-2 text-sm">
          Creating another project will result in multiple projects referencing the same proposal.
        </p>
      </div>
    {/if}

    {#if !hideProjectDate}
      <span
        class="flex w-full items-center gap-2 {errors.project_award_date !== undefined
          ? 'bg-red-200'
          : ''}"
      >
        <label for="project_award_date">Project Award Date</label>
        <input
          class="flex-1"
          type="text"
          name="project_award_date"
          placeholder="Project Award Date"
          use:flatpickrAction
          bind:value={item.project_award_date}
        />
        {#if item.project_award_date}
          <DsActionButton
            icon="mdi:close"
            title="Clear date"
            color="red"
            action={() => (item.project_award_date = "")}
          />
        {/if}
        {#if errors.project_award_date !== undefined}
          <span class="text-red-600">{errors.project_award_date.message}</span>
        {/if}
      </span>
    {/if}

    {#if !hideProposalDates}
      <span
        class="flex w-full items-center gap-2 {errors.proposal_opening_date !== undefined
          ? 'bg-red-200'
          : ''}"
      >
        <label for="proposal_opening_date">Proposal Opening Date</label>
        <input
          class="flex-1"
          type="text"
          name="proposal_opening_date"
          placeholder="Proposal Opening Date"
          use:flatpickrAction
          bind:value={item.proposal_opening_date}
        />
        {#if item.proposal_opening_date}
          <DsActionButton
            icon="mdi:close"
            title="Clear date"
            color="red"
            action={() => (item.proposal_opening_date = "")}
          />
        {/if}
        {#if errors.proposal_opening_date !== undefined}
          <span class="text-red-600">{errors.proposal_opening_date.message}</span>
        {/if}
      </span>

      <span
        class="flex w-full items-center gap-2 {errors.proposal_submission_due_date !== undefined
          ? 'bg-red-200'
          : ''}"
      >
        <label for="proposal_submission_due_date">Proposal Submission Due Date</label>
        <input
          class="flex-1"
          type="text"
          name="proposal_submission_due_date"
          placeholder="Proposal Submission Due Date"
          use:flatpickrAction
          bind:value={item.proposal_submission_due_date}
        />
        {#if item.proposal_submission_due_date}
          <DsActionButton
            icon="mdi:close"
            title="Clear date"
            color="red"
            action={() => (item.proposal_submission_due_date = "")}
          />
        {/if}
        {#if errors.proposal_submission_due_date !== undefined}
          <span class="text-red-600">{errors.proposal_submission_due_date.message}</span>
        {/if}
      </span>
    {/if}

    <div class="flex w-full flex-col gap-1 {errors.location !== undefined ? 'bg-red-200' : ''}">
      <label for="location">Location</label>
      <DSLocationPicker bind:value={item.location as string} {errors} fieldName="location" />
    </div>

    <!---
  <DsTextInput
    bind:value={item.number as string}
    {errors}
    fieldName="number"
    uiName="Number"
    disabled={true}
  />
  <p class="self-start text-xs text-neutral-600">Number is auto-assigned on creation.</p>
-->
    <DsTextInput
      bind:value={item.description as string}
      {errors}
      fieldName="description"
      uiName="Description"
    />

    {#if $clients.index !== null}
      <div class="flex w-full items-end gap-1">
        <div class="flex-1">
          <DsAutoComplete
            bind:value={item.client as string}
            index={$clients.index}
            {errors}
            fieldName="client"
            uiName="Client"
            disabled={(data as JobsPageData).editing === false &&
              (item as any).parent &&
              (item as any).parent !== ""}
          >
            {#snippet resultTemplate(client)}{client.name}{/snippet}
          </DsAutoComplete>
        </div>
        {#if !item.client}
          <DsActionButton
            action={openAddClientPopover}
            icon="feather:plus-circle"
            color="green"
            title="Add new client"
          />
        {/if}
      </div>
    {/if}

    <div class="flex w-full flex-col gap-1 {errors.contact !== undefined ? 'bg-red-200' : ''}">
      <div class="flex w-full items-end gap-1">
        <span class="flex flex-1 gap-2">
          <label for="contact">Client Contact</label>
          <select
            id="contact"
            name="contact"
            bind:value={item.contact}
            class="flex-1 rounded-sm border border-neutral-300 px-1"
            disabled={item.client === "" || client_contacts.length === 0}
          >
            <option value=""
              >{item.client === "" ? "Select a client first" : "Select a contact"}</option
            >
            {#each client_contacts as contact}
              <option value={contact.id}>{formatContactName(contact)}</option>
            {/each}
          </select>
        </span>
        {#if item.client && !item.contact}
          <DsActionButton
            action={openAddContactPopover}
            icon="feather:plus-circle"
            color="green"
            title="Add new contact"
          />
        {/if}
      </div>
      {#if errors.contact !== undefined}
        <span class="text-red-600">{errors.contact.message}</span>
      {/if}
    </div>

    {#if $profiles.index !== null}
      <DsAutoComplete
        bind:value={item.manager as string}
        index={$profiles.index}
        idField="uid"
        {errors}
        fieldName="manager"
        uiName="Manager"
      >
        {#snippet resultTemplate(item)}
          {item.surname}, {item.given_name}
        {/snippet}
      </DsAutoComplete>
    {/if}
    {#if $profiles.index !== null}
      <DsAutoComplete
        bind:value={item.alternate_manager as string}
        index={$profiles.index}
        idField="uid"
        {errors}
        fieldName="alternate_manager"
        uiName="Alternate Manager"
        excludeIds={item.manager ? [item.manager] : []}
      >
        {#snippet resultTemplate(item)}
          {item.surname}, {item.given_name}
        {/snippet}
      </DsAutoComplete>
    {/if}

    <DsCheck
      bind:value={item.fn_agreement as boolean}
      {errors}
      fieldName="fn_agreement"
      uiName="Has First Nation's agreement"
    />

    <DsSelector
      bind:value={item.status as string}
      items={statusOptionsList}
      {errors}
      fieldName="status"
      uiName="Status"
    >
      {#snippet optionTemplate(item)}{item.name}{/snippet}
    </DsSelector>
    {#if !isProposal}
      <p
        class="cursor-help self-start text-sm text-neutral-600"
        title="Use the status Closed if the purpose of this job is to act as a reporting container for many sub jobs. These “Parent” jobs can be billed to if their status is set to “Active”, however they are created as “Closed” by default. For example MTO retainers are usually created as Closed."
      >
        Creating a parent job? Use Closed.
      </p>
    {/if}

    {#if isProposal}
      <div
        class="flex w-full flex-col gap-1"
        class:bg-red-200={errors.proposal_value !== undefined}
      >
        <label class="text-sm font-semibold" for="proposal_value">Proposal Value ($)</label>
        <input
          id="proposal_value"
          name="proposal_value"
          type="number"
          class="rounded-sm border border-neutral-300 px-2 py-1"
          bind:value={item.proposal_value as number}
          min={0}
          step={1}
          disabled={isCancelledProposal}
        />
        {#if errors.proposal_value !== undefined}
          <span class="text-sm text-red-600">{errors.proposal_value.message}</span>
        {/if}
      </div>

      <DsCheck
        bind:value={item.time_and_materials as boolean}
        {errors}
        fieldName="time_and_materials"
        uiName="Time and Materials"
        disabled={isCancelledProposal}
      />
      <p class="self-start text-xs text-neutral-600">
        Proposals with status Submitted, Awarded, or Not Awarded must have a proposal value or be
        marked as Time and Materials. If both, interpret proposal value as a maximum.
      </p>
    {/if}

    <DsSelector
      bind:value={item.authorizing_document}
      items={authorizingDocumentOptions}
      {errors}
      fieldName="authorizing_document"
      uiName="Authorizing Document"
    >
      {#snippet optionTemplate(item)}{item.name}{/snippet}
    </DsSelector>
    {#if item.authorizing_document === "PO"}
      <DsTextInput bind:value={item.client_po} {errors} fieldName="client_po" uiName="Client PO" />
    {/if}
    <DsTextInput
      bind:value={item.client_reference_number}
      {errors}
      fieldName="client_reference_number"
      uiName="Client Reference Number"
    />

    {#if !isProposal}
      <div
        class="flex w-full flex-col gap-1"
        class:bg-red-200={errors.outstanding_balance !== undefined}
      >
        <label class="text-sm font-semibold" for="outstanding_balance">Outstanding Balance</label>
        <input
          id="outstanding_balance"
          name="outstanding_balance"
          type="number"
          class="rounded-sm border border-neutral-300 px-2 py-1"
          bind:value={item.outstanding_balance as number}
          min={0}
          step={0.01}
        />
        {#if errors.outstanding_balance !== undefined}
          <span class="text-sm text-red-600">{errors.outstanding_balance.message}</span>
        {/if}
      </div>
      {#if item.outstanding_balance_date}
        <p class="self-start text-sm text-neutral-600">
          Last updated: {item.outstanding_balance_date}
        </p>
      {/if}

      {#if $rateSheets.index !== null}
        <DsAutoComplete
          bind:value={(item as Record<string, unknown>).rate_sheet as string}
          index={$rateSheets.index}
          filter={activeRateSheetFilter}
          {errors}
          fieldName="rate_sheet"
          uiName="Rate Sheet"
        >
          {#snippet resultTemplate(sheet)}
            {sheet.name} (rev. {sheet.revision}) - {sheet.effective_date}
          {/snippet}
        </DsAutoComplete>
      {/if}
    {/if}

    {#if $jobs.index !== null}
      {@const proposalHasCustomError =
        errors.proposal !== undefined &&
        (errors.proposal as { message: string; proposalId?: string }).proposalId}
      {@const proposalErrors = proposalHasCustomError ? {} : errors}
      <DsAutoComplete
        bind:value={item.proposal as string}
        index={$jobs.index}
        filter={proposalFilter}
        errors={proposalErrors}
        fieldName="proposal"
        uiName="Proposal"
      >
        {#snippet resultTemplate(job)}{jobLabel(
            job as unknown as Pick<JobApiResponse, "id" | "number" | "description">,
          )}{/snippet}
      </DsAutoComplete>
    {/if}
    {#if errors.proposal !== undefined && (errors.proposal as { message: string; proposalId?: string }).proposalId}
      <div class="w-full rounded-sm border border-red-300 bg-red-50 p-3 text-red-800">
        <p class="font-semibold">Cannot create project from proposal</p>
        <p class="mt-1">{errors.proposal.message}</p>
        <p class="mt-2">
          <a
            href="/jobs/{(errors.proposal as { message: string; proposalId?: string })
              .proposalId}/edit"
            target="_blank"
            rel="noopener noreferrer"
            class="text-blue-600 underline hover:text-blue-800"
          >
            Edit the proposal in a new tab →
          </a>
        </p>
      </div>
    {/if}

    {#if $divisions.index !== null}
      <div class="flex w-full flex-col gap-2">
        <span class="font-semibold">Divisions</span>
        <div class="flex flex-col gap-2">
          {#each allocations as row, idx}
            <div class="flex items-center gap-2">
              <div class="min-w-[280px] flex-1">
                <DsAutoComplete
                  bind:value={row.division}
                  index={$divisions.index}
                  {errors}
                  fieldName={"allocation_division_" + idx}
                  uiName="Division"
                  choose={(id) => setAllocationDivision(idx, id)}
                >
                  {#snippet resultTemplate(item: DivisionsResponse)}{item.code} - {item.name}{/snippet}
                </DsAutoComplete>
              </div>
              <input
                class="w-28 rounded-sm border border-neutral-300 px-2 py-1"
                type="number"
                min={0}
                step={0.5}
                bind:value={row.hours}
                oninput={(e) => {
                  const v = parseFloat((e.target as HTMLInputElement).value || "0");
                  allocations = allocations.map((r, i) =>
                    i === idx ? { ...r, hours: isNaN(v) ? 0 : v } : r,
                  );
                }}
              />
              <button
                class="text-neutral-600"
                onclick={preventDefault(() => removeAllocationRow(idx))}
                title="Remove"
              >
                &times;
              </button>
            </div>
          {/each}
        </div>
        <div>
          <DsActionButton action={addAllocationRow} icon="feather:plus-circle" color="green"
            >Add division</DsActionButton
          >
        </div>
      </div>
    {/if}

    {#if $clients.index !== null}
      <DsAutoComplete
        bind:value={item.job_owner as string}
        index={$clients.index}
        {errors}
        fieldName="job_owner"
        uiName="Job Owner"
      >
        {#snippet resultTemplate(item)}{item.name}{/snippet}
      </DsAutoComplete>
    {/if}

    <div class="flex w-full gap-2 {errors.branch !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <label for="branch">Branch</label>
        <select
          id="branch"
          name="branch"
          bind:value={item.branch}
          class="flex-1 rounded-sm border border-neutral-300 px-1"
        >
          <option value="">Select a branch</option>
          {#each branches as branch}
            <option value={branch.id}>{branch.code ?? branch.name}</option>
          {/each}
        </select>
      </span>
      {#if errors.branch !== undefined}
        <span class="text-red-600">{errors.branch.message}</span>
      {/if}
    </div>

    <div class="flex w-full flex-col gap-2 {errors.categories !== undefined ? 'bg-red-200' : ''}">
      <label for="categories">Categories</label>
      <div class="flex flex-wrap gap-1">
        {#each categories as category}
          <span class="flex items-center rounded-full bg-neutral-200 px-2">
            <span>{category.name}</span>
            <button
              class="text-neutral-500"
              onclick={preventDefault(() => removeCategory(category.id))}
            >
              &times;
            </button>
          </span>
        {/each}
        {#each newCategories as categoryName}
          <span class="flex items-center rounded-full bg-neutral-200 px-2">
            <span>{categoryName}</span>
            <button
              class="text-neutral-500"
              onclick={preventDefault(
                () => (newCategories = newCategories.filter((name) => name !== categoryName)),
              )}
            >
              &times;
            </button>
          </span>
        {/each}
      </div>
      <div class="flex items-center gap-1">
        <DsTextInput
          bind:value={newCategory as string}
          {errors}
          fieldName="newCategory"
          uiName="Add Category"
        />
        <DsActionButton
          action={addCategory}
          icon="feather:plus-circle"
          color="green"
          title="Add Category"
        />
      </div>
      {#if errors.categories !== undefined}
        <span class="text-red-600">{errors.categories.message}</span>
      {/if}
    </div>

    <div class="flex w-full flex-col gap-2 {errors.global !== undefined ? 'bg-red-200' : ''}">
      <span class="flex w-full gap-2">
        <DsActionButton type="submit">Save</DsActionButton>
        <DsActionButton action={cancel}>Cancel</DsActionButton>
      </span>
      {#if errors.global !== undefined}
        <span class="text-red-600">{errors.global.message}</span>
      {/if}
    </div>
  </form>
{/if}

<!-- Status Comment Modal for No Bid / Cancelled -->
<DSPopover
  bind:show={showStatusCommentModal}
  title="{pendingStatus === 'No Bid' ? 'No Bid' : 'Cancel Proposal'} - Comment Required"
  subtitle="A comment is required to set the status to &quot;{pendingStatus}&quot;. Please provide a reason."
  error={statusCommentError}
  submitting={statusCommentSubmitting}
  submitLabel="Add Comment & Set Status"
  onSubmit={submitStatusComment}
  onCancel={cancelStatusChange}
>
  <textarea
    class="w-full rounded-sm border border-neutral-300 p-2"
    rows="4"
    placeholder="Enter your comment..."
    bind:value={statusComment}
    disabled={statusCommentSubmitting}
  ></textarea>
</DSPopover>

<!-- Add Client Popover -->
<DSPopover
  bind:show={showAddClientPopover}
  title="Add New Client"
  error={newClientError}
  submitting={newClientSubmitting}
  submitLabel="Create Client"
  onSubmit={createNewClient}
  onCancel={closeAddClientPopover}
>
  <div class="flex flex-col gap-1">
    <label class="text-sm font-semibold" for="new_client_name">Name</label>
    <input
      id="new_client_name"
      type="text"
      class="rounded-sm border border-neutral-300 px-2 py-1"
      placeholder="Client name"
      bind:value={newClientName}
      disabled={newClientSubmitting}
    />
  </div>

  <div class="flex flex-col gap-1">
    {#if $busdevLeads.items.length > 0}
      {#if $busdevLeads.items.length <= 10}
        <DsSelector
          bind:value={newClientBusdevLead}
          items={[{ id: "", given_name: "", surname: "" }, ...$busdevLeads.items]}
          errors={{}}
          fieldName="new_client_busdev_lead"
          uiName="Business Development Lead"
        >
          {#snippet optionTemplate(option)}
            {#if option.id === ""}
              -- select --
            {:else}
              {option.surname}{option.given_name ? `, ${option.given_name}` : ""}
            {/if}
          {/snippet}
        </DsSelector>
      {:else if $busdevLeads.index !== null}
        <DsAutoComplete
          bind:value={newClientBusdevLead}
          index={$busdevLeads.index}
          errors={{}}
          fieldName="new_client_busdev_lead"
          uiName="Business Development Lead"
          idField="id"
        >
          {#snippet resultTemplate(option)}
            {option.surname}{option.given_name ? `, ${option.given_name}` : ""}
          {/snippet}
        </DsAutoComplete>
      {/if}
    {:else if $busdevLeads.loading}
      <span class="text-sm font-semibold">Business Development Lead</span>
      <span class="text-sm text-neutral-500">Loading business development leads…</span>
    {:else}
      <span class="text-sm font-semibold">Business Development Lead</span>
      <span class="text-sm text-neutral-500">No eligible Business Development Leads found.</span>
    {/if}
  </div>
</DSPopover>

<!-- Add Contact Popover -->
<DSPopover
  bind:show={showAddContactPopover}
  title="Add New Contact"
  subtitle="Adding contact for: {selectedClientName}"
  error={newContactError}
  submitting={newContactSubmitting}
  submitLabel="Create Contact"
  onSubmit={createNewContact}
  onCancel={closeAddContactPopover}
>
  <div class="flex flex-col gap-1">
    <label class="text-sm font-semibold" for="new_contact_given_name">Given Name</label>
    <input
      id="new_contact_given_name"
      type="text"
      class="rounded-sm border border-neutral-300 px-2 py-1"
      placeholder="Given name"
      bind:value={newContactGivenName}
      disabled={newContactSubmitting}
    />
  </div>

  <div class="flex flex-col gap-1">
    <label class="text-sm font-semibold" for="new_contact_surname">Surname</label>
    <input
      id="new_contact_surname"
      type="text"
      class="rounded-sm border border-neutral-300 px-2 py-1"
      placeholder="Surname"
      bind:value={newContactSurname}
      disabled={newContactSubmitting}
    />
  </div>

  <div class="flex flex-col gap-1">
    <label class="text-sm font-semibold" for="new_contact_email">Email</label>
    <input
      id="new_contact_email"
      type="email"
      class="rounded-sm border border-neutral-300 px-2 py-1"
      placeholder="Email"
      bind:value={newContactEmail}
      disabled={newContactSubmitting}
    />
  </div>
</DSPopover>

<style>
  .disabled-notice {
    padding: 1.5rem;
    background-color: #fff3cd;
    border: 1px solid #ffc107;
    border-radius: 0.5rem;
    margin: 1rem;
    max-width: 600px;
  }

  .disabled-notice p {
    margin: 0.5rem 0;
    color: #856404;
  }
</style>
