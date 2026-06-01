<script lang="ts" generics="Item extends { location?: string | null }">
  import { onMount, onDestroy } from "svelte";
  import L from "leaflet";
  import "leaflet/dist/leaflet.css";
  import "leaflet.markercluster";
  import "leaflet.markercluster/dist/MarkerCluster.css";
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
  type MarkerTone = "orange" | "teal";
  type MapMarkerOptions = L.MarkerOptions & { mapTone?: MarkerTone };

  let {
    items = [],
    tile,
    markerTone,
    markerTitle,
    onViewportChange,
    showZoomControls = true,
  }: {
    // Accept any item shape; consumers provide a tile snippet and we only
    // require that items may have a `location` string used for marker placement.
    items: Item[];
    tile: Snippet<[Item]>;
    markerTone?: (item: Item) => MarkerTone;
    markerTitle?: (item: Item) => string;
    onViewportChange?: (v: {
      bounds: { north: number; south: number; east: number; west: number };
      zoom: number;
      center: { lat: number; lon: number };
    }) => void;
    showZoomControls?: boolean;
  } = $props();

  let mapElement: HTMLDivElement | null = null;
  let map: L.Map | null = null;
  let layerGroup: L.MarkerClusterGroup | null = null;
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

  function itemLatLng(item: Item): L.LatLng | null {
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

  function clusterRadius(zoom: number): number {
    if (zoom >= 11) return 34;
    if (zoom >= 8) return 46;
    return 58;
  }

  function countLabel(count: number, singular: string, plural: string): string {
    return `${count} ${count === 1 ? singular : plural}`;
  }

  function clusterIcon(cluster: L.MarkerCluster): L.DivIcon {
    const childMarkers = cluster.getAllChildMarkers() as Array<
      L.Marker & { options: MapMarkerOptions }
    >;
    const total = childMarkers.length;
    const proposalCount = childMarkers.filter(
      (marker) => marker.options.mapTone === "orange",
    ).length;
    const projectCount = total - proposalCount;
    const proposalAngle = total > 0 ? (proposalCount / total) * 360 : 0;
    const dominantTone = proposalCount > projectCount ? "orange" : "teal";
    const mixClass = proposalCount > 0 && projectCount > 0 ? "ds-map-cluster--mixed" : "";
    const sizeClass =
      total >= 100 ? "ds-map-cluster--large" : total >= 25 ? "ds-map-cluster--medium" : "";
    const title = `${countLabel(total, "mapped job", "mapped jobs")}: ${countLabel(projectCount, "project", "projects")}, ${countLabel(proposalCount, "proposal", "proposals")}`;

    return L.divIcon({
      html: `<div class="ds-map-cluster ds-map-cluster--${dominantTone} ${mixClass} ${sizeClass}" style="--proposal-angle: ${proposalAngle}deg" title="${title}" aria-label="${title}"><span>${total}</span></div>`,
      className: "ds-map-cluster-icon",
      iconSize: total >= 100 ? [58, 58] : total >= 25 ? [50, 50] : [42, 42],
      iconAnchor: total >= 100 ? [29, 29] : total >= 25 ? [25, 25] : [21, 21],
    });
  }

  function markerOptions(item: Item): MapMarkerOptions {
    const tone = markerTone?.(item);
    const title = markerTitle?.(item).trim();
    const options: MapMarkerOptions = { mapTone: tone };
    if (title) {
      options.title = title;
      options.alt = title;
    }
    if (!tone) return options;

    const markerEl = document.createElement("div");
    markerEl.className = `ds-map-marker ds-map-marker--${tone}`;
    if (title) {
      markerEl.title = title;
      markerEl.setAttribute("aria-label", title);
    }

    options.icon = L.divIcon({
      html: markerEl,
      className: "ds-map-marker-icon",
      iconSize: [18, 18],
      iconAnchor: [9, 9],
      popupAnchor: [0, -9],
    });
    return options;
  }

  function renderMarkers() {
    if (!map) return;
    if (layerGroup) {
      layerGroup.clearLayers();
    } else {
      layerGroup = L.markerClusterGroup({
        chunkedLoading: true,
        maxClusterRadius: clusterRadius,
        showCoverageOnHover: false,
        spiderLegPolylineOptions: { color: "#525252", opacity: 0.55, weight: 1.5 },
        spiderfyDistanceMultiplier: 1.25,
        spiderfyOnMaxZoom: true,
        zoomToBoundsOnClick: true,
        iconCreateFunction: clusterIcon,
      }).addTo(map);
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
      const marker = L.marker(ll, markerOptions(item));
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
  {#each items as it, i (it)}
    <div bind:this={tileEls[i]} class="hidden">
      {@render tile(it)}
    </div>
  {/each}
</div>

<style>
  :global(.ds-map-marker-icon) {
    background: transparent;
    border: 0;
  }

  :global(.ds-map-marker) {
    box-sizing: border-box;
    width: 18px;
    height: 18px;
    border: 2px solid white;
    border-radius: 9999px;
    box-shadow:
      0 1px 3px rgb(0 0 0 / 0.4),
      0 0 0 1px rgb(0 0 0 / 0.12);
  }

  :global(.ds-map-marker--orange) {
    background-color: #c2410c;
  }

  :global(.ds-map-marker--teal) {
    background-color: #0f766e;
  }

  :global(.ds-map-cluster-icon) {
    background: transparent;
    border: 0;
  }

  :global(.ds-map-cluster) {
    --project-color: #0f766e;
    --proposal-color: #c2410c;
    display: grid;
    box-sizing: border-box;
    width: 42px;
    height: 42px;
    place-items: center;
    border: 2px solid white;
    border-radius: 9999px;
    background: var(--project-color);
    box-shadow:
      0 12px 26px rgb(0 0 0 / 0.3),
      0 0 0 1px rgb(0 0 0 / 0.18);
    color: #171717;
    font-size: 0.8rem;
    font-weight: 800;
    line-height: 1;
  }

  :global(.ds-map-cluster--medium) {
    width: 50px;
    height: 50px;
    font-size: 0.9rem;
  }

  :global(.ds-map-cluster--large) {
    width: 58px;
    height: 58px;
    font-size: 1rem;
  }

  :global(.ds-map-cluster--orange) {
    background: var(--proposal-color);
  }

  :global(.ds-map-cluster--mixed) {
    background: conic-gradient(
      var(--proposal-color) 0 var(--proposal-angle),
      var(--project-color) var(--proposal-angle) 360deg
    );
  }

  :global(.ds-map-cluster span) {
    display: grid;
    width: 66%;
    height: 66%;
    place-items: center;
    border-radius: 9999px;
    background: rgb(255 255 255 / 0.94);
    box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.45);
  }
</style>
