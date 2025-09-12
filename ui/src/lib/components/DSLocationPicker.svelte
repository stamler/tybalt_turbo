<script module>
  let idCounter = $state(0);
</script>

<script lang="ts">
  import DsActionButton from "./DSActionButton.svelte";
  // We use the well-known Open Location Code (Plus Codes) library.
  // encode(lat, lon) -> full global Plus Code (typically 10 characters with '+').
  import { OpenLocationCode } from "open-location-code";
  import { onMount } from "svelte";
  // Leaflet map for picking a location visually
  import L from "leaflet";
  import "leaflet/dist/leaflet.css";
  // Ensure marker icons render correctly under Vite by importing assets directly
  // These imports let the bundler resolve URLs properly at runtime.
  import markerIcon2xUrl from "leaflet/dist/images/marker-icon-2x.png";
  import markerIconUrl from "leaflet/dist/images/marker-icon.png";
  import markerShadowUrl from "leaflet/dist/images/marker-shadow.png";
  // Ensure Leaflet uses our absolute URLs rather than trying to build paths
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  delete (L.Icon.Default.prototype as any)._getIconUrl;
  L.Icon.Default.mergeOptions({
    iconRetinaUrl: markerIcon2xUrl,
    iconUrl: markerIconUrl,
    shadowUrl: markerShadowUrl,
  });
  // Local type shims for open-location-code if types are not present
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  type OLCEncode = (lat: number, lon: number, codeLength?: number) => string;

  // get an id for this instance from the counter in the module context then
  // increment it so the next instance gets a different id
  const thisId = idCounter;
  idCounter += 1;

  let {
    value = $bindable(),
    errors,
    fieldName,
    disabled = false,
    readonly = false,
  }: {
    value: string;
    errors: Record<string, { message: string }>;
    fieldName: string;
    disabled?: boolean;
    readonly?: boolean;
  } = $props();

  let loading = $state(false);
  let code = $state(value ?? "");
  const olcInstance = new OpenLocationCode();
  let copied = $state(false);

  const PLUS_CODE_REGEX = /^[23456789CFGHJMPQRVWX]{8}\+[23456789CFGHJMPQRVWX]{2,3}$/; // matches PocketBase validation

  const isValid = $derived.by(() => PLUS_CODE_REGEX.test(code ?? ""));

  function applyCode(next: string, updateMarker: boolean = true) {
    const normalized = (next ?? "").trim().toUpperCase();
    code = normalized;
    value = normalized;
    if (updateMarker && map && PLUS_CODE_REGEX.test(normalized)) {
      try {
        const area = olcInstance.decode(normalized);
        setMarker(area.latitudeCenter, area.longitudeCenter);
      } catch {
        // ignore
      }
    }
  }

  async function useCurrentLocation() {
    if (disabled || typeof navigator === "undefined" || !navigator.geolocation) return;
    loading = true;
    try {
      const position = await new Promise<GeolocationPosition>((resolve, reject) =>
        navigator.geolocation.getCurrentPosition(resolve, reject, {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 0,
        }),
      );
      const lat = position.coords.latitude;
      const lon = position.coords.longitude;
      // Let the library choose the default precision (usually 10 characters with 2 after '+').
      const newCode = olcInstance.encode(lat, lon);
      applyCode(newCode);
    } catch (err) {
      console.error("Geolocation error", err);
    } finally {
      loading = false;
    }
  }

  async function copyToClipboard() {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(code);
      } else {
        const el = document.createElement("textarea");
        el.value = code;
        el.setAttribute("readonly", "");
        el.style.position = "absolute";
        el.style.left = "-9999px";
        document.body.appendChild(el);
        el.select();
        document.execCommand("copy");
        document.body.removeChild(el);
      }
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {}
  }

  // --- Map setup ---
  let mapElement: HTMLDivElement | null = null;
  let map: any = null;
  let marker: any = null;

  function setMarker(lat: number, lon: number, animate: boolean = true) {
    if (!map) return;
    const latLng = L.latLng(lat, lon);
    if (!marker) {
      marker = L.marker(latLng, { draggable: !readonly }).addTo(map);
      if (!readonly) {
        marker.on("dragend", () => {
          const pos = marker!.getLatLng();
          applyCode(olcInstance.encode(pos.lat, pos.lng), false);
        });
      }
    } else {
      marker.setLatLng(latLng);
    }
    const currentZoom = typeof map.getZoom === "function" ? map.getZoom() : undefined;
    const zoom =
      typeof currentZoom === "number" && isFinite(currentZoom) && currentZoom > 0
        ? Math.max(currentZoom, 14)
        : 14;
    // Pan/zoom to the new position
    if (animate && typeof (map as any).flyTo === "function") {
      (map as any).flyTo(latLng, zoom, { animate: true, duration: 0.35 });
    } else {
      map.setView(latLng, zoom, { animate });
    }
    setTimeout(() => map && typeof map.invalidateSize === "function" && map.invalidateSize(), 0);
  }

  onMount(() => {
    if (!mapElement) return;
    map = L.map(mapElement, { zoomControl: true, preferCanvas: true });
    L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
      attribution: "&copy; OpenStreetMap contributors",
      maxZoom: 19,
    }).addTo(map);
    // Ensure map sizes correctly after mount
    setTimeout(() => map && map.invalidateSize(), 0);
    // Initial view: try existing value from parent; if invalid, default to USA centroid
    try {
      if (code && olcInstance.isValid(code) && olcInstance.isFull(code)) {
        const area = olcInstance.decode(code);
        setMarker(area.latitudeCenter, area.longitudeCenter, false);
      } else {
        map.setView([39.8283, -98.5795], 4);
      }
    } catch {
      map.setView([39.8283, -98.5795], 4);
    }

    if (!readonly) {
      map.on("click", (e: any) => {
        setMarker(e.latlng.lat, e.latlng.lng, true);
        applyCode(olcInstance.encode(e.latlng.lat, e.latlng.lng), false);
      });
    }

    // Also update on map move end if we already have a marker
    map.on("moveend", () => {
      if (marker) {
        try {
          const pos = marker.getLatLng();
          marker.setLatLng(pos); // re-attach to layer order
          if (marker._icon) marker._icon.style.zIndex = "10000";
        } catch {}
      }
    });
  });

  // Input is display-only; location is set via the map

  // Keep local code in sync if parent changes the bound `value` externally
  $effect(() => {
    if (value !== code) {
      applyCode(value);
    }
  });
</script>

<div class="flex w-full flex-col gap-2" class:bg-red-200={errors[fieldName] !== undefined}>
  <span class="flex w-full gap-2">
    <input
      id={`location-input-${thisId}`}
      name={fieldName}
      class="flex-1 rounded border border-neutral-300 px-1 {disabled ? 'opacity-50' : ''} {disabled
        ? 'cursor-not-allowed'
        : ''}"
      type="text"
      placeholder="8-char + 2-3-char Plus Code (e.g. 849VCWC8+R9)"
      bind:value={code}
      oninput={(e) => ((e.target as HTMLInputElement).value = code)}
      readonly
      {disabled}
    />
    {#if !readonly}
      <DsActionButton
        action={useCurrentLocation}
        icon="mdi:crosshairs-gps"
        color="blue"
        title="Use current location"
        {loading}
      />
    {/if}
    <DsActionButton
      action={copyToClipboard}
      icon="mdi:content-copy"
      color="yellow"
      title={copied ? "Copied" : "Copy"}
    />
  </span>
  {#if errors[fieldName] !== undefined}
    <span class="text-red-600">{errors[fieldName].message}</span>
  {/if}

  <div class="h-64 w-full overflow-hidden rounded border border-neutral-300">
    <div bind:this={mapElement} class="h-full w-full"></div>
  </div>
</div>
