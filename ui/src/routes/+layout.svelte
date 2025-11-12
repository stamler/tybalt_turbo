<script lang="ts">
  import { browser } from "$app/environment";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { authStore } from "$lib/stores/auth";
  import { globalStore } from "$lib/stores/global";
  import { afterNavigate } from "$app/navigation";
  import "../app.css";
  import ErrorBar from "$lib/components/ErrorBar.svelte";
  import DsActionButton from "$lib/components/DSActionButton.svelte";
  import VersionInfo from "$lib/components/VersionInfo.svelte";
  import { navigating } from "$app/stores";
  import { tasks } from "$lib/stores/tasks";
  import { navSections as allNavSections } from "$lib/navConfig";

  // children is a function that we will call to render the current route
  // https://svelte-5-preview.vercel.app/docs/snippets#passing-snippets-to-components
  // Any content inside the component tags that is not a snippet declaration
  // implicitly becomes part of the children snippet
  let { children } = $props();
  let isSidebarOpen = $state(false);

  let navSections = $derived(
    $globalStore.showAllUi
      ? allNavSections
      : allNavSections
          .map((section) => {
            // Filter items within each section
            const filteredItems = section.items
              .filter((item) => {
                if (
                  item.href.startsWith("/time/tracking") ||
                  item.href.startsWith("/expenses/tracking")
                ) {
                  return $globalStore.claims.includes("report");
                }
                if (
                  item.href.startsWith("/time/sheets/pending") ||
                  item.href.startsWith("/time/sheets/approved") ||
                  item.href.startsWith("/expenses/pending") ||
                  item.href.startsWith("/expenses/approved") ||
                  item.href.startsWith("/pos/pending")
                ) {
                  return $globalStore.claims.includes("tapr");
                }
                if (item.href.startsWith("/absorb/actions")) {
                  return $globalStore.claims.includes("absorb");
                }
                if (item.href.startsWith("/reports/expense/queue")) {
                  return $globalStore.claims.includes("commit");
                }
                if (item.href.startsWith("/time/amendments")) {
                  return (
                    $globalStore.claims.includes("tame") || $globalStore.claims.includes("report")
                  );
                }
                if (
                  item.href.startsWith("/admin_profiles") ||
                  item.href.startsWith("/timetypes") ||
                  item.href.startsWith("/divisions")
                ) {
                  return $globalStore.claims.includes("admin");
                }
                return true; // Keep item if no specific claim is required
              })
              .map((item) => {
                const gatedButtons = (item.buttons || []).filter((btn) => {
                  if (
                    btn.action === "/jobs/add" ||
                    btn.action === "/jobs/unused" ||
                    btn.action === "/jobs/stale" ||
                    btn.action === "/jobs/latest" ||
                    btn.action === "/clients/add"
                  ) {
                    return $globalStore.claims.includes("job");
                  }
                  if (btn.action === "/vendors/add") {
                    return $globalStore.claims.includes("payables_admin");
                  }
                  return true;
                });
                return { ...item, buttons: gatedButtons };
              });

            // Return the section with filtered items
            return { ...section, items: filteredItems };
          })
          .filter((section) => {
            // Hide the entire "Reports" section if the user lacks the 'report' claim
            if (section.title === "Reports") {
              return $globalStore.claims.includes("report");
            }
            // Hide sections that become empty after filtering
            return section.items.length > 0;
          }),
  );

  // Helper store that reflects whether any task is running
  const tasksLoading = { subscribe: tasks.showTasks };

  afterNavigate((navigation) => {
    // afterNavigate may be called multiple times, but we only want to run once
    // per navigation event (i.e. when the URL changes) so we check the type of
    // the navigation event and don't run if it's the hydration event (enter)

    if (navigation.type === "enter") {
      return;
    }

    // refresh the global store if it's stale
    globalStore.refresh();
    // Close sidebar on navigation on mobile
    if (window.innerWidth < 1024) {
      isSidebarOpen = false;
    }
  });

  // route guards
  $effect(() => {
    if (browser && !$authStore?.isValid && $page.url.pathname !== "/login") {
      goto(`/login?redirect=${encodeURIComponent($page.url.pathname + $page.url.search)}`);
    } else if (browser && $authStore?.isValid && $page.url.pathname === "/login") {
      const redirectUrl = sessionStorage.getItem("redirectUrl");
      if (redirectUrl) {
        sessionStorage.removeItem("redirectUrl");
        goto(redirectUrl);
      } else {
        goto("/");
      }
    }
  });

  const toggleSidebar = () => {
    isSidebarOpen = !isSidebarOpen;
  };
</script>

<div class="h-screen w-screen overflow-hidden">
  <!-- Mobile header with hamburger -->
  <header
    class="relative flex h-10 w-full items-center justify-between bg-neutral-700 px-3 text-white lg:hidden"
  >
    <button onclick={toggleSidebar} class="text-white" aria-label="Toggle Menu">
      <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M4 6h16M4 12h16M4 18h16"
        />
      </svg>
    </button>
    <span class="absolute left-1/2 -translate-x-1/2 transform text-lg font-semibold">𝕋𝕌ℝ𝔹𝕆</span>
    <div class="ml-auto flex items-center gap-2">
      <VersionInfo />
      {#if $authStore?.isValid}
        <div class="[&_svg]:h-8 [&_svg]:w-8 lg:[&_svg]:h-6 lg:[&_svg]:w-6">
          <DsActionButton
            action={authStore.logout}
            icon="feather:log-out"
            title="Logout"
            color="red"
            transparentBackground
          />
        </div>
      {/if}
    </div>
    {#if $navigating || $tasksLoading}
      <!-- Mobile loading bar sits at bottom of the header -->
      <div
        class="absolute bottom-0 left-0 right-0 h-[4px] animate-pulse bg-purple-500 lg:hidden"
      ></div>
    {/if}
  </header>

  <div class="flex h-[calc(100vh-2.5rem)] lg:h-screen">
    <!-- Sidebar -->
    <aside
      class={`fixed inset-y-0 left-0 w-screen transform bg-neutral-700 text-white transition-transform duration-300 ease-in-out lg:static lg:w-64 ${
        isSidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
      } z-[1100]`}
    >
      <div class="h-full overflow-y-auto overflow-x-hidden">
        <!-- Desktop brand header -->
        <div class="hidden h-10 items-center justify-between px-4 text-lg font-semibold lg:flex">
          <a href="/" class="text-white">𝕋𝕌ℝ𝔹𝕆</a>
          <VersionInfo />
        </div>
        <!-- Mobile close/header -->
        <div class="flex h-12 items-center justify-between px-4 lg:hidden">
          <span class="text-xl font-semibold">𝕋𝕌ℝ𝔹𝕆</span>
          <button onclick={toggleSidebar} aria-label="Close Menu">
            <svg class="h-8 w-8" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
        <nav class="mt-2 px-1">
          {#each navSections as section}
            <div class="mt-2">
              <p class="p-2 text-lg font-semibold uppercase text-neutral-400 lg:text-xs">
                {section.title}
              </p>
              {#each section.items as item}
                <div
                  class="flex h-12 items-center pr-4 lg:h-8 lg:pr-2"
                  class:justify-between={item.buttons}
                >
                  <a
                    href={item.href}
                    class="ml-4 flex h-full flex-grow items-center rounded pl-2 text-xl hover:bg-neutral-600 lg:text-sm"
                    >{item.label}</a
                  >
                  {#if item.buttons}
                    <div
                      class="flex items-center gap-1 [&_svg]:h-8 [&_svg]:w-8 lg:[&_svg]:h-6 lg:[&_svg]:w-6"
                    >
                      {#each item.buttons as btn}
                        <DsActionButton
                          action={btn.action}
                          icon={btn.icon}
                          title={btn.title}
                          color={btn.color}
                          transparentBackground
                        />
                      {/each}
                    </div>
                  {:else}
                    <div class="w-8"></div>
                  {/if}
                </div>
              {/each}
            </div>
          {/each}
          {#if $authStore?.isValid}
            <div class="mb-6 mt-2">
              <p class="p-2 text-lg font-semibold uppercase text-neutral-400 lg:text-xs">Account</p>
              <div class="flex h-12 items-center justify-between pr-4 lg:h-8 lg:pr-2">
                <a
                  href={`/profile/${$authStore?.model?.id}`}
                  class="ml-4 flex h-full flex-grow items-center rounded pl-2 text-xl hover:bg-neutral-600 lg:text-sm"
                >
                  {$authStore?.model?.email}
                </a>
                <div
                  class="flex items-center gap-1 [&_svg]:h-8 [&_svg]:w-8 lg:[&_svg]:h-6 lg:[&_svg]:w-6"
                >
                  <DsActionButton
                    action={() => globalStore.toggleShowAllUi()}
                    icon={$globalStore.showAllUi ? "mdi:eye-off-outline" : "mdi:eye-outline"}
                    title={$globalStore.showAllUi ? "Hide All UI" : "Show All UI"}
                    color={$globalStore.showAllUi ? "gray" : "blue"}
                    transparentBackground
                  />
                  <DsActionButton
                    action={authStore.logout}
                    icon="feather:log-out"
                    title="Logout"
                    color="red"
                    transparentBackground
                  />
                </div>
              </div>
            </div>
          {/if}
        </nav>
      </div>
    </aside>

    <!-- Main content -->
    <div class="relative flex-1 overflow-auto bg-white">
      <ErrorBar />
      {#if $navigating || $tasksLoading}
        <!-- Desktop loading bar overlays content without shifting layout -->
        <div
          class="absolute left-0 right-0 top-0 z-50 hidden h-[4px] animate-pulse bg-purple-500 lg:block"
        ></div>
      {/if}
      <div class="h-full">
        {@render children()}
      </div>
    </div>

    <!-- Overlay for mobile -->
    {#if isSidebarOpen}
      <div
        class="fixed inset-0 z-[1000] bg-black bg-opacity-50 lg:hidden"
        onclick={toggleSidebar}
        aria-hidden="true"
      ></div>
    {/if}
  </div>
</div>
