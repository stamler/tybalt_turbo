<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import L from "leaflet";
  import "leaflet/dist/leaflet.css";
  import { OpenLocationCode } from "open-location-code";
  import markerIcon2xUrl from "leaflet/dist/images/marker-icon-2x.png";
  import markerIconUrl from "leaflet/dist/images/marker-icon.png";
  import markerShadowUrl from "leaflet/dist/images/marker-shadow.png";
  import type { Snippet } from "svelte";

  // Fix Leaflet marker icon paths for Vite bundling
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  delete (L.Icon.Default.prototype as any)._getIconUrl;
  L.Icon.Default.mergeOptions({
    iconRetinaUrl: markerIcon2xUrl,
    iconUrl: markerIconUrl,
    shadowUrl: markerShadowUrl,
  });

  type OLCInstance = {
    decode: (code: string) => {
      latitudeCenter: number;
      longitudeCenter: number;
      latitudeLo: number;
      latitudeHi: number;
      longitudeLo: number;
      longitudeHi: number;
    };
    isValid: (code: string) => boolean;
    isFull: (code: string) => boolean;
  };

  let {
    items = [],
    tile,
    onViewportChange,
    showZoomControls = true,
  }: {
    // Accept any item shape; consumers provide a tile snippet and we only
    // require that items may have a `location` string used for marker placement.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    items: any[];
    tile: Snippet<[any]>;
    onViewportChange?: (v: {
      bounds: { north: number; south: number; east: number; west: number };
      zoom: number;
      center: { lat: number; lon: number };
    }) => void;
    showZoomControls?: boolean;
  } = $props();

  let mapElement: HTMLDivElement | null = null;
  let map: L.Map | null = null;
  let layerGroup: L.LayerGroup | null = null;
  const olc: OLCInstance = new (OpenLocationCode as unknown as { new (): OLCInstance })();
  let hasUserInteracted = $state(false);
  // Off-DOM rendered snippet nodes for popup content cloning
  // Why do we render the snippet off-DOM and clone it into Leaflet popups?
  // - Svelte 5 snippets/components cannot be reliably instantiated imperatively
  //   outside of the component tree (e.g., via `new Component({ target })`). Doing so
  //   can throw `component_api_invalid_new` and fail to render.
  // - Leaflet popups are created/manipulated outside Svelte’s lifecycle. Passing a
  //   static DOM node to Leaflet is the most stable approach.
  // Solution:
  // - We pre-render the provided `tile` snippet into hidden divs here (one per item),
  //   then clone that HTML and hand the clone to Leaflet’s `bindPopup`. This keeps
  //   the map integration robust while preserving the snippet API.
  let tileEls: Array<HTMLDivElement | null> = $state([]);

  function itemLatLng(item: any): L.LatLng | null {
    const code: string | undefined = item?.location as string | undefined;
    if (!code) return null;
    try {
      if (!olc.isValid(code) || !olc.isFull(code)) return null;
      const area = olc.decode(code);
      return L.latLng(area.latitudeCenter, area.longitudeCenter);
    } catch {
      return null;
    }
  }

  function renderMarkers() {
    if (!map) return;
    if (layerGroup) {
      layerGroup.clearLayers();
    } else {
      layerGroup = L.layerGroup().addTo(map);
    }

    // Compute padded bounds from all item markers
    const bounds: L.LatLngBounds | null = items.reduce(
      (acc: L.LatLngBounds | null, item) => {
        const ll = itemLatLng(item);
        if (!ll) return acc;
        if (!acc) return L.latLngBounds(ll, ll);
        acc.extend(ll);
        return acc;
      },
      null as L.LatLngBounds | null,
    );

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      const ll = itemLatLng(item);
      if (!ll) continue;
      const marker = L.marker(ll);
      const template = tileEls[i];
      const content = template
        ? (template.cloneNode(true) as HTMLElement)
        : document.createElement("div");
      // Remove any hidden classes inherited from the template container
      content.classList.remove("hidden");
      marker.bindPopup(content, { closeButton: true, autoPan: true, maxWidth: 320 });
      marker.addTo(layerGroup);
    }

    // Fit bounds comfortably if we have at least one marker
    // Only auto-fit before any user interaction (pan/zoom)
    if (bounds && !hasUserInteracted) {
      const pad = {
        paddingTopLeft: [20, 20] as [number, number],
        paddingBottomRight: [20, 20] as [number, number],
      };
      map.fitBounds(bounds, { ...pad, maxZoom: 16 });
    }
  }

  // Using Leaflet native popups; no custom overlay state needed

  // No background click handler necessary when using native popups

  let viewportTimer: number | null = null;

  function emitViewport() {
    if (!map || !onViewportChange) return;
    const b = map.getBounds();
    const z = map.getZoom();
    const c = map.getCenter();
    onViewportChange({
      bounds: { north: b.getNorth(), south: b.getSouth(), east: b.getEast(), west: b.getWest() },
      zoom: z,
      center: { lat: c.lat, lon: c.lng },
    });
  }

  onMount(() => {
    if (!mapElement) return;
    map = L.map(mapElement, { zoomControl: false, preferCanvas: true });
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
      maxZoom: 19,
      detectRetina: true,
    }).addTo(map);
    // Ensure the map has a valid center/zoom before any bounds or viewport logic
    // This prevents Leaflet errors when calling getBounds on an unset view
    map.setView([39.8283, -98.5795], 4);
    if (showZoomControls) {
      L.control.zoom({ position: "topleft" }).addTo(map);
    }
    setTimeout(() => map && map.invalidateSize(), 0);
    renderMarkers();
    // Notify initial viewport after first render
    setTimeout(() => emitViewport(), 0);

    // Mark when the user starts interacting so we stop auto-fitting
    map.on("movestart", () => {
      hasUserInteracted = true;
    });
    map.on("zoomstart", () => {
      hasUserInteracted = true;
    });

    // With native popups we don't need to track overlay position
    map.on("moveend zoomend", () => {
      if (viewportTimer) window.clearTimeout(viewportTimer);
      viewportTimer = window.setTimeout(() => emitViewport(), 250);
    });
  });

  onDestroy(() => {
    if (map) {
      map.remove();
      map = null;
    }
    layerGroup = null;
  });

  $effect(() => {
    renderMarkers();
  });
</script>

<div class="relative h-full w-full overflow-hidden rounded-sm border border-neutral-300">
  <div bind:this={mapElement} class="h-full w-full"></div>
  <!-- Hidden templates for popup content, one per item -->
  {#each items as it, i}
    <div bind:this={tileEls[i]} class="hidden">
      {@render tile(it)}
    </div>
  {/each}
</div>
