<script lang="ts">
  import DSItemsMap from "./DSItemsMap.svelte";
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
  let tileEls: Array<HTMLDivElement | null> = $state([]);

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

<div class="h-full w-full">
  <DSItemsMap items={filtered} onViewportChange={handleViewportChange} showZoomControls={true}>
    {#snippet tile(item)}
      <div class="text-sm">
        <a href={`/jobs/${item.id}/details`} class="font-semibold hover:underline">
          {item.number ?? "Job"}
        </a>
        <div class="text-neutral-600">{item.description ?? ""}</div>
        <div class="mt-1 text-xs">{item.client ?? ""}</div>
      </div>
    {/snippet}
  </DSItemsMap>
</div>
