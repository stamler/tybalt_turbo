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
  import { navSections } from "$lib/navConfig";

  // children is a function that we will call to render the current route
  // https://svelte-5-preview.vercel.app/docs/snippets#passing-snippets-to-components
  // Any content inside the component tags that is not a snippet declaration
  // implicitly becomes part of the children snippet
  let { children } = $props();
  let isSidebarOpen = $state(false);

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
        <button
          class="rounded bg-blue-500 px-2 py-1 text-sm text-white hover:bg-blue-700"
          onclick={authStore.logout}
          aria-label="Logout"
        >
          <svg
            class="h-6 w-6"
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="currentColor"
          >
            <path
              fill-rule="evenodd"
              d="M16.5 3.75a1.5 1.5 0 0 1 1.5 1.5v13.5a1.5 1.5 0 0 1-1.5 1.5h-6a1.5 1.5 0 0 1-1.5-1.5V15a.75.75 0 0 0-1.5 0v3.75a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3V5.25a3 3 0 0 0-3-3h-6a3 3 0 0 0-3 3V9A.75.75 0 1 0 9 9V5.25a1.5 1.5 0 0 1 1.5-1.5h6ZM5.78 8.47a.75.75 0 0 0-1.06 0l-3 3a.75.75 0 0 0 0 1.06l3 3a.75.75 0 0 0 1.06-1.06l-1.72-1.72H15a.75.75 0 0 0 0-1.5H4.06l1.72-1.72a.75.75 0 0 0 0-1.06Z"
              clip-rule="evenodd"
            />
          </svg>
        </button>
      {/if}
    </div>
  </header>

  <div class="flex h-[calc(100vh-2.5rem)] lg:h-screen">
    <!-- Sidebar -->
    <aside
      class={`fixed inset-y-0 left-0 w-64 transform bg-neutral-700 text-white transition-transform duration-300 ease-in-out lg:static ${
        isSidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
      } z-30`}
    >
      <div class="h-full overflow-y-auto">
        <div class="hidden h-10 items-center justify-between px-4 text-lg font-semibold lg:flex">
          <a href="/" class="text-white">Tybalt</a>
          <VersionInfo />
        </div>
        <nav class="mt-2 px-1">
          {#each navSections as section}
            <div class="mt-2">
              <p class="p-2 text-xs font-semibold uppercase text-neutral-400">{section.title}</p>
              {#each section.items as item}
                <div class="flex h-8 items-center pr-2" class:justify-between={item.button}>
                  <a
                    href={item.href}
                    class="ml-4 flex h-full flex-grow items-center rounded pl-2 hover:bg-neutral-600"
                    >{item.label}</a
                  >
                  {#if item.button}
                    <DsActionButton
                      action={item.button.action}
                      icon={item.button.icon}
                      title={item.button.title}
                      color={item.button.color}
                      transparentBackground
                    />
                  {:else}
                    <div class="w-8"></div>
                  {/if}
                </div>
              {/each}
            </div>
          {/each}
          {#if $authStore?.isValid}
            <div class="mt-2">
              <p class="p-2 text-xs font-semibold uppercase text-neutral-400">Account</p>
              <a
                href={`/profile/${$authStore?.model?.id}`}
                class="ml-4 block rounded pl-2 pr-2 hover:bg-neutral-600"
              >
                {$authStore?.model?.email}
              </a>
              <button
                class="ml-4 w-full rounded pl-2 pr-2 text-left text-red-400 hover:bg-neutral-600"
                onclick={authStore.logout}
              >
                Logout
              </button>
            </div>
          {/if}
        </nav>
      </div>
    </aside>

    <!-- Main content -->
    <div class="flex-1 overflow-auto bg-white">
      <ErrorBar />
      {#if $navigating}
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
