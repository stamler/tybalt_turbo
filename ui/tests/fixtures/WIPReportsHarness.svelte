<script lang="ts">
  import NavFlyout from "../../src/lib/components/NavFlyout.svelte";
  import WIPReport from "../../src/lib/components/jobs/WIPReport.svelte";
  import { navSections } from "../../src/lib/navConfig";
  import type { WIPMode } from "../../src/lib/reports/wipReports";
  let mode = $state<WIPMode>("my");
  let access = $state("kpi");
  let showAllUi = $state(false);
  let defaultDivision = $state("civil");
  const claims = $derived(access === "kpi" || access === "admin" ? [access] : []);
  const manager = $derived(access === "manager");
  const wipNav = navSections
    .find((section) => section.title === "Business")!
    .items.find((item) => item.label === "WIP")!;
</script>

<div class="p-4">
  <label
    >Report <select aria-label="Report" bind:value={mode}
      ><option value="my">My WIP</option><option value="branch">Branch WIP</option><option
        value="division">Division WIP</option
      ></select
    ></label
  >
  <label
    >Access <select aria-label="Access" bind:value={access}
      ><option>kpi</option><option>admin</option><option>manager</option><option>ordinary</option
      ></select
    ></label
  >
  <label
    >Default division <select aria-label="Default division" bind:value={defaultDivision}
      ><option value="civil">Civil</option><option value="">None</option></select
    ></label
  >
  <nav aria-label="WIP navigation" class="flex h-12 w-64 overflow-hidden bg-neutral-700 text-white">
    <NavFlyout
      label={wipNav.label}
      items={wipNav.children!}
      pathname={`/jobs/wip/${mode}`}
      {claims}
      isBranchManager={manager}
      {showAllUi}
    />
  </nav>
  <label><input type="checkbox" bind:checked={showAllUi} />Show All UI</label>
</div>
{#key `${mode}:${defaultDivision}`}
  <WIPReport
    {mode}
    allowed={mode === "my" || access !== "ordinary"}
    branches={[
      { id: "north", name: "North" },
      { id: "south", name: "South" },
    ]}
    divisions={[
      { id: "civil", code: "CIV", name: "Civil" },
      { id: "structural", code: "STR", name: "Structural" },
    ]}
    initialBranch="north"
    initialDivision={defaultDivision}
  />
{/key}
