<script lang="ts">
  import { onMount } from "svelte";

  interface VersionInfo {
    name: string;
    version: string;
    build: number;
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

  interface ApiVersionResponse {
    name: string;
    version: string;
    build: number;
    fullVersion: string;
    gitCommit: string;
    gitCommitShort: string;
    gitBranch: string;
    buildTime: string;
  }

  async function fetchVersionInfo() {
    try {
      loading = true;
      error = null;

      // Try to fetch from the API first
      const response = await fetch("/api/version");
      if (response.ok) {
        const data: ApiVersionResponse = await response.json();
        versionInfo = {
          name: data.name,
          version: data.version,
          build: data.build,
          fullVersion: data.fullVersion,
          gitCommit: data.gitCommit,
          gitCommitShort: data.gitCommitShort,
          gitBranch: data.gitBranch,
          buildTime: data.buildTime,
        };
      } else {
        // Fallback to build-time constants if API fails
        const versionModule = await import("$lib/version");
        versionInfo = versionModule.VERSION_INFO;
      }
    } catch (err) {
      console.warn("Failed to fetch version from API, using build-time version:", err);
      try {
        // Fallback to build-time constants
        const versionModule = await import("$lib/version");
        versionInfo = versionModule.VERSION_INFO;
      } catch (buildErr) {
        console.error("Failed to load version info:", buildErr);
        error = "Version info unavailable";
      }
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchVersionInfo();
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
