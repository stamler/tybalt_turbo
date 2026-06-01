<script lang="ts">
  import DSItemsMap from "./DSItemsMap.svelte";
  import { resolve } from "$app/paths";
  import { jobs } from "$lib/stores/jobs";
  import type { JobApiResponse } from "$lib/stores/jobs";
  import { OpenLocationCode } from "open-location-code";
  import { get } from "svelte/store";

  type OLCInstance = {
    encode: (lat: number, lon: number, codeLength?: number) => string;
    decode: (code: string) => {
      latitudeCenter: number;
      longitudeCenter: number;
      latitudeLo: number;
      latitudeHi: number;
      longitudeLo: number;
      longitudeHi: number;
    };
    shorten: (code: string, lat: number, lon: number) => string;
    isValid: (code: string) => boolean;
  };
  const olc: OLCInstance = new (OpenLocationCode as unknown as { new (): OLCInstance })();

  // Viewport -> items filter
  // We avoid prefix-based OLC filtering because it can exclude valid items near
  // viewport edges as zoom changes. Instead, decode each item's OLC center and
  // keep it if it falls within the current Leaflet bounds.
  type ViewBounds = { north: number; south: number; east: number; west: number };
  function isInBounds(lat: number, lon: number, b: ViewBounds): boolean {
    return lat <= b.north && lat >= b.south && lon <= b.east && lon >= b.west;
  }

  let filtered: JobApiResponse[] = $state([]);
  let lastBounds: ViewBounds | null = $state(null);

  function isProposal(item: JobApiResponse): boolean {
    return item.number?.startsWith("P") ?? false;
  }

  function jobMarkerTone(item: JobApiResponse): "orange" | "teal" {
    return isProposal(item) ? "orange" : "teal";
  }

  function jobMarkerTitle(item: JobApiResponse): string {
    const type = isProposal(item) ? "Proposal" : "Project";
    return `${type} ${item.number ?? "Job"}`;
  }

  function jobIdentifierClass(item: JobApiResponse): string {
    const tone = isProposal(item) ? "bg-orange-700" : "bg-teal-700";
    return `ds-job-map-popup-id inline-flex rounded-sm px-2 py-1 text-xs font-bold hover:underline ${tone}`;
  }

  function applyFilter(bounds: ViewBounds | null) {
    const state = get(jobs);
    const items = state.items as JobApiResponse[];
    if (!bounds) {
      filtered = items.filter((i) => typeof i.location === "string" && i.location);
      return;
    }
    filtered = items.filter((i) => {
      const code = i.location as string | undefined;
      if (!code) return false;
      try {
        const area = olc.decode(code);
        const lat = area.latitudeCenter;
        const lon = area.longitudeCenter;
        return isInBounds(lat, lon, bounds);
      } catch {
        return false;
      }
    });
  }

  // Init jobs store when mounted
  jobs.init();

  function handleViewportChange(v: { bounds: ViewBounds; zoom: number }) {
    lastBounds = v.bounds;
    applyFilter(lastBounds);
  }

  // Re-filter whenever jobs store updates
  jobs.subscribe(() => {
    applyFilter(lastBounds);
  });
</script>

<div class="relative h-full w-full">
  <DSItemsMap
    items={filtered}
    markerTone={jobMarkerTone}
    markerTitle={jobMarkerTitle}
    onViewportChange={handleViewportChange}
    showZoomControls={true}
  >
    {#snippet tile(item)}
      <div class="text-sm">
        <a href={resolve(`/jobs/${item.id}/details`)} class={jobIdentifierClass(item)}>
          {item.number ?? "Job"}
        </a>
        <div class="text-neutral-600">{item.description ?? ""}</div>
        <div class="mt-1 text-xs">{item.client ?? ""}</div>
      </div>
    {/snippet}
  </DSItemsMap>
  <div
    class="pointer-events-none absolute top-3 right-3 z-[1000] flex gap-3 rounded-sm border border-neutral-300 bg-white/95 px-3 py-2 text-xs font-medium text-neutral-700 shadow-sm backdrop-blur"
  >
    <span class="inline-flex items-center gap-1.5">
      <span class="size-2.5 rounded-full bg-teal-700"></span>
      Project
    </span>
    <span class="inline-flex items-center gap-1.5">
      <span class="size-2.5 rounded-full bg-orange-700"></span>
      Proposal
    </span>
  </div>
</div>

<style>
  :global(.leaflet-container a.ds-job-map-popup-id) {
    color: white;
  }
</style>
