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
  import { navSections } from "$lib/navConfig";

  // children is a function that we will call to render the current route
  // https://svelte-5-preview.vercel.app/docs/snippets#passing-snippets-to-components
  // Any content inside the component tags that is not a snippet declaration
  // implicitly becomes part of the children snippet
  let { children } = $props();
  let isSidebarOpen = $state(false);

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
      goto("/login");
    } else if (browser && $authStore?.isValid && $page.url.pathname === "/login") {
      goto("/time/entries/list");
    }
  });

  const toggleSidebar = () => {
    isSidebarOpen = !isSidebarOpen;
  };
</script>

<div class="h-screen w-screen overflow-hidden">
  <!-- Mobile header with hamburger -->
  <header
    class="flex h-10 w-full items-center justify-between bg-neutral-700 px-3 text-white lg:hidden"
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
    <span class="text-lg font-semibold">Tybalt</span>
    <div class="flex items-center gap-2">
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
  </header>

  <div class="flex h-[calc(100vh-2.5rem)] lg:h-screen">
    <!-- Sidebar -->
    <aside
      class={`fixed inset-y-0 left-0 w-screen transform bg-neutral-700 text-white transition-transform duration-300 ease-in-out lg:static lg:w-64 ${
        isSidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
      } z-30`}
    >
      <div class="h-full overflow-y-auto overflow-x-hidden">
        <!-- Desktop brand header -->
        <div class="hidden h-10 items-center justify-between px-4 text-lg font-semibold lg:flex">
          <a href="/" class="text-white">Tybalt</a>
          <VersionInfo />
        </div>
        <!-- Mobile close/header -->
        <div class="flex h-12 items-center justify-between px-4 lg:hidden">
          <span class="text-xl font-semibold">Tybalt</span>
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
                  class:justify-between={item.button}
                >
                  <a
                    href={item.href}
                    class="ml-4 flex h-full flex-grow items-center rounded pl-2 text-xl hover:bg-neutral-600 lg:text-sm"
                    >{item.label}</a
                  >
                  {#if item.button}
                    <div class="[&_svg]:h-8 [&_svg]:w-8 lg:[&_svg]:h-6 lg:[&_svg]:w-6">
                      <DsActionButton
                        action={item.button.action}
                        icon={item.button.icon}
                        title={item.button.title}
                        color={item.button.color}
                        transparentBackground
                      />
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
                <div class="[&_svg]:h-8 [&_svg]:w-8 lg:[&_svg]:h-6 lg:[&_svg]:w-6">
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
    <div class="flex-1 overflow-auto bg-white">
      <ErrorBar />
      {#if $navigating || $tasksLoading}
        <div class="sticky top-0 z-50 h-[4px] w-full animate-pulse bg-purple-500"></div>
      {/if}
      <div>
        {@render children()}
      </div>
    </div>

    <!-- Overlay for mobile -->
    {#if isSidebarOpen}
      <div
        class="fixed inset-0 z-20 bg-black bg-opacity-50 lg:hidden"
        onclick={toggleSidebar}
        aria-hidden="true"
      ></div>
    {/if}
  </div>
</div>
