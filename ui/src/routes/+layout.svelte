<script lang="ts">
  import { browser } from "$app/environment";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { authStore } from "$lib/stores/auth";
  import { globalStore } from "$lib/stores/global";
  import { afterNavigate } from "$app/navigation";
  import "../app.css";

  // children is a function that we will call to render the current route
  // https://svelte-5-preview.vercel.app/docs/snippets#passing-snippets-to-components
  // Any content inside the component tags that is not a snippet declaration
  // implicitly becomes part of the children snippet
  let { children } = $props();

  afterNavigate((navigation) => {
    // afterNavigate may be called multiple times, but we only want to run once
    // per navigation event (i.e. when the URL changes) so we check the type of
    // the navigation event and don't run if it's the hydration event (enter)

    if (navigation.type === "enter") {
      return;
    }

    // refresh the global store if it's stale
    globalStore.refresh();
  });

  // route guards
  $effect(() => {
    if (browser && !$authStore?.isValid && $page.url.pathname !== "/login") {
      goto("/login");
    } else if (browser && $authStore?.isValid && $page.url.pathname === "/login") {
      goto("/time/entries/list");
    }
  });
</script>

<header class="flex h-10 w-screen items-center justify-between bg-neutral-700 px-3 text-white">
  <a href="/">Tybalt</a>
  <a href="/time/entries/add">New Entry</a>
  <a href="/time/entries/list">Entries</a>
  <a href="/jobs">Jobs</a>
  <a href="/timetypes">Time Types</a>
  <a href="/divisions">Divisions</a>
  <div class="h-full">
    {#if $authStore?.isValid}
      <!-- user is logged in -->
      <span class="flex h-full items-center gap-2">
        <span><a href="/profile/{$authStore?.model?.id}">{$authStore?.model?.email}</a></span>
        <button
          class="rounded bg-blue-500 px-2 py-1 text-sm text-white hover:bg-blue-700"
          onclick={authStore.logout}
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
      </span>
    {:else}
      <span class="flex h-full items-center gap-2">Not logged in</span>
    {/if}
  </div>
</header>
<main>
  {@render children()}
</main>
