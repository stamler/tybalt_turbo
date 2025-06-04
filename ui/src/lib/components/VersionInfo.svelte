<script lang="ts">
  import { onMount } from "svelte";

  /**
   * VersionInfo Component - Simple version display
   *
   * Production: Fetches version from /api/version (backend with real git info)
   * Development: Reads version.json + shows ".dev" suffix
   *
   * Displays as a clickable button (e.g. "v0.1.4") that shows details popup
   */

  interface VersionInfo {
    name: string;
    version: string;
    build: number | string;
    fullVersion: string;
    gitCommit: string;
    gitCommitShort: string;
    gitBranch: string;
    buildTime: string;
  }

  let versionInfo = $state<VersionInfo | null>(null);
  let showDetails = $state(false);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function fetchVersionInfo() {
    const isProduction = typeof window !== "undefined" && window.location.hostname !== "localhost";

    try {
      if (isProduction) {
        // Production: Get version from backend API
        const response = await fetch("/api/version");
        if (response.ok) {
          versionInfo = await response.json();
          return;
        }
        throw new Error("API failed");
      } else {
        // Development: Read version.json and add ".dev"
        const response = await fetch("/version.json");
        if (response.ok) {
          const data = await response.json();
          if (!data.version || !data.name) {
            throw new Error("Invalid version.json");
          }
          versionInfo = {
            name: data.name,
            version: data.version,
            build: "dev",
            fullVersion: `${data.version}.dev`,
            gitCommit: "local-development",
            gitCommitShort: "local",
            gitBranch: "dev",
            buildTime: new Date().toISOString(),
          };
          return;
        }
        throw new Error("version.json not found");
      }
    } catch (err) {
      console.warn("Version fetch failed:", err);
      error = "Version unavailable";
    }
  }

  onMount(() => {
    fetchVersionInfo().finally(() => {
      loading = false;
    });
  });

  function toggleDetails() {
    showDetails = !showDetails;
  }

  function formatBuildTime(buildTime: string): string {
    try {
      return new Date(buildTime).toLocaleDateString();
    } catch {
      return buildTime;
    }
  }
</script>

{#if loading}
  <div class="flex items-center text-xs text-neutral-400">
    <span>Loading...</span>
  </div>
{:else if error}
  <div class="flex items-center text-xs text-red-400">
    <span>{error}</span>
  </div>
{:else if versionInfo}
  <div class="relative">
    <button
      onclick={toggleDetails}
      class="flex items-center gap-1 text-xs text-neutral-400 transition-colors hover:text-neutral-200"
      title="Click for version details"
    >
      <span>v{versionInfo.fullVersion}</span>
      <svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
        />
      </svg>
    </button>

    {#if showDetails}
      <div
        class="absolute right-0 top-full z-50 mt-1 w-56 rounded-md border border-neutral-600 bg-neutral-800 p-3 text-xs text-white shadow-lg lg:fixed lg:right-4 lg:top-12 lg:w-48"
      >
        <div class="space-y-1">
          <div class="border-b border-neutral-600 pb-1 font-semibold">{versionInfo.name}</div>
          <div><span class="text-neutral-400">Version:</span> {versionInfo.version}</div>
          <div><span class="text-neutral-400">Build:</span> {versionInfo.build}</div>
          <div>
            <span class="text-neutral-400">Git:</span>
            {versionInfo.gitCommitShort} ({versionInfo.gitBranch})
          </div>
          <div>
            <span class="text-neutral-400">Built:</span>
            {formatBuildTime(versionInfo.buildTime)}
          </div>
        </div>
        <div class="mt-2 border-t border-neutral-600 pt-2">
          <button
            onclick={() => navigator.clipboard.writeText(versionInfo?.gitCommit || "")}
            class="text-xs text-blue-400 underline hover:text-blue-300"
            title="Copy full commit hash"
          >
            Copy commit hash
          </button>
        </div>
      </div>
    {/if}
  </div>
{/if}

<!-- Click outside to close -->
{#if showDetails}
  <div class="fixed inset-0 z-40" onclick={() => (showDetails = false)} aria-hidden="true"></div>
{/if}
