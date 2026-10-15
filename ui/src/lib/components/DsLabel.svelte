<script lang="ts" module>
  export type LabelTone = "neutral" | "info" | "success" | "warning" | "danger";
  // Categorical colours distinguish kinds of thing and carry no meaning. Use
  // tone when the colour means something.
  export type LabelColor =
    | "neutral"
    | "gray"
    | "blue"
    | "cyan"
    | "teal"
    | "green"
    | "yellow"
    | "amber"
    | "orange"
    | "red"
    | "purple";
</script>

<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    children,
    color,
    tone,
    title,
    style,
    size = "md",
    shade,
  }: {
    children: Snippet;
    color?: LabelColor | string;
    tone?: LabelTone;
    title?: string;
    style?: "inverted";
    size?: "md" | "sm";
    // A neutral shade from 0 (lightest) to 1 (darkest), for showing how much
    // of something each label has. It replaces color and tone.
    shade?: number;
  } = $props();

  // Full class strings so Tailwind sees every class.
  const colors: Record<LabelColor, { regular: string; inverted: string }> = {
    neutral: {
      regular: "border-neutral-300 bg-neutral-100 text-neutral-500",
      inverted: "bg-neutral-500 text-white",
    },
    gray: {
      regular: "border-gray-300 bg-gray-100 text-gray-500",
      inverted: "bg-gray-500 text-white",
    },
    blue: {
      regular: "border-blue-300 bg-blue-100 text-blue-500",
      inverted: "bg-blue-500 text-white",
    },
    cyan: {
      regular: "border-cyan-300 bg-cyan-100 text-cyan-500",
      inverted: "bg-cyan-500 text-white",
    },
    teal: {
      regular: "border-teal-300 bg-teal-100 text-teal-500",
      inverted: "bg-teal-500 text-white",
    },
    green: {
      regular: "border-green-300 bg-green-100 text-green-500",
      inverted: "bg-green-500 text-white",
    },
    yellow: {
      regular: "border-yellow-300 bg-yellow-100 text-yellow-500",
      inverted: "bg-yellow-500 text-white",
    },
    amber: {
      regular: "border-amber-300 bg-amber-100 text-amber-600",
      inverted: "bg-amber-500 text-white",
    },
    orange: {
      regular: "border-orange-300 bg-orange-100 text-orange-500",
      inverted: "bg-orange-500 text-white",
    },
    red: {
      regular: "border-red-300 bg-red-100 text-red-500",
      inverted: "bg-red-500 text-white",
    },
    purple: {
      regular: "border-purple-300 bg-purple-100 text-purple-500",
      inverted: "bg-purple-500 text-white",
    },
  };

  // Lightest to darkest, each with readable text.
  // Neutral 500 is skipped: neither dark nor white text reads well on it.
  const shades = [
    "border-neutral-300 bg-white text-neutral-700",
    "border-neutral-300 bg-neutral-100 text-neutral-800",
    "border-neutral-300 bg-neutral-200 text-neutral-800",
    "border-neutral-400 bg-neutral-300 text-neutral-900",
    "border-neutral-500 bg-neutral-400 text-neutral-950",
    "border-neutral-600 bg-neutral-600 text-white",
    "border-neutral-800 bg-neutral-800 text-white",
  ];

  const toneColors: Record<LabelTone, LabelColor> = {
    neutral: "neutral",
    info: "blue",
    success: "green",
    warning: "amber",
    danger: "red",
  };

  const sizes = {
    md: "px-2 py-1 text-sm",
    sm: "px-2 py-0.5 text-xs",
  } as const;

  const palette = $derived(
    colors[(tone ? toneColors[tone] : color) as LabelColor] ?? colors.neutral,
  );
  const regular = $derived(
    shade === undefined
      ? palette.regular
      : shades[Math.min(shades.length - 1, Math.max(0, Math.ceil(shade * shades.length) - 1))],
  );
</script>

{#if style === "inverted"}
  <span
    class="inline-flex items-center gap-1 rounded-full {palette.inverted} {sizes[size]}"
    {title}
  >
    {@render children()}
  </span>
{:else}
  <span class="inline-flex items-center gap-1 rounded-sm border {regular} {sizes[size]}" {title}>
    {@render children()}
  </span>
{/if}
