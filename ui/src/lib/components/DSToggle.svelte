<script lang="ts">
  // Option shape has two modes:
  // 1) compact labels only
  // 2) labels + descriptions (when showOptionDescriptions=true)
  type ToggleOption = { id: string; label: string };
  type ToggleOptionWithDescription = ToggleOption & { description: string };

  // Accessibility contract:
  // - caller must provide either ariaLabel/ariaLabelledBy
  // - or provide a visible label prop (used as aria-label fallback)
  type AccessibleName =
    | { ariaLabel: string; ariaLabelledBy?: string; label?: string }
    | { ariaLabel?: string; ariaLabelledBy: string; label?: string }
    | { ariaLabel?: string; ariaLabelledBy?: string; label: string };

  // Discriminated props enforce "all descriptions or none".
  type ToggleProps =
    | (AccessibleName & {
        value: string;
        options: ToggleOption[];
        showOptionDescriptions?: false;
        fullWidth?: boolean;
      })
    | (AccessibleName & {
        value: string;
        options: ToggleOptionWithDescription[];
        showOptionDescriptions: true;
        fullWidth?: boolean;
      });

  let {
    value = $bindable(),
    options,
    showOptionDescriptions = false,
    fullWidth = false,
    label,
    ariaLabel,
    ariaLabelledBy,
  }: ToggleProps = $props();

  const hasDescription = (
    option: ToggleOption | ToggleOptionWithDescription,
  ): option is ToggleOptionWithDescription => "description" in option;

  const canShowOptionDescriptions = $derived.by(
    () => showOptionDescriptions && options.every(hasDescription),
  );

  // shared state used by keyboard navigation and selected-description rendering
  const selectedIndex = $derived.by(() => options.findIndex((option) => option.id === value));
  const hasIntegratedMeta = $derived.by(() => Boolean(label) || canShowOptionDescriptions);
  let optionsRailRef = $state<HTMLDivElement | null>(null);
  const selectedOption = $derived.by(() =>
    canShowOptionDescriptions
      ? (options.find((option) => option.id === value) as ToggleOptionWithDescription | undefined)
      : undefined,
  );
  const computedAriaLabel = $derived.by(() => ariaLabel ?? label ?? undefined);
  const computedAriaLabelledBy = $derived.by(() =>
    label ? undefined : (ariaLabelledBy ?? undefined),
  );
  const useResponsiveGrid = $derived.by(() => fullWidth && options.length > 2);

  // Guard misuse in development: when descriptions are enabled, every option
  // must define a description key for consistent layout semantics.
  $effect(() => {
    if (showOptionDescriptions && !canShowOptionDescriptions) {
      console.error(
        "DSToggle requires a description for every option when showOptionDescriptions=true.",
      );
    }
  });

  function focusOption(index: number): void {
    if (!optionsRailRef || index < 0 || index >= options.length) return;
    const buttons = optionsRailRef.querySelectorAll('button[role="radio"]');
    const button = buttons[index];
    if (button instanceof HTMLButtonElement) button.focus();
  }

  function selectByIndex(index: number): void {
    if (index < 0 || index >= options.length) return;
    value = options[index].id;
  }

  // Radio-like keyboard behavior:
  // arrows cycle, Home/End jump, then focus follows selection.
  function handleOptionKeydown(event: KeyboardEvent, optionIndex: number): void {
    if (options.length === 0) return;
    let nextIndex = optionIndex;

    switch (event.key) {
      case "ArrowRight":
      case "ArrowDown":
        nextIndex = (optionIndex + 1) % options.length;
        break;
      case "ArrowLeft":
      case "ArrowUp":
        nextIndex = (optionIndex - 1 + options.length) % options.length;
        break;
      case "Home":
        nextIndex = 0;
        break;
      case "End":
        nextIndex = options.length - 1;
        break;
      default:
        return;
    }

    event.preventDefault();
    selectByIndex(nextIndex);
    focusOption(nextIndex);
  }
</script>

<!-- Root group frame.
  When label or descriptions are present, we render an integrated field shell.
  Otherwise the control stays compact for list/header use cases. -->
<div
  class="inline-flex flex-col rounded-sm {fullWidth ? 'w-full' : 'w-fit'} {hasIntegratedMeta
    ? 'overflow-hidden border border-neutral-300 bg-neutral-100 p-2'
    : ''}"
  role="radiogroup"
  aria-label={computedAriaLabel}
  aria-labelledby={computedAriaLabelledBy}
>
  {#if label}<span class="mb-1 text-sm text-neutral-800 max-lg:text-base">{label}</span>{/if}
  <!-- Options rail:
    - Desktop: horizontal segmented buttons
    - Mobile (max-lg) for full-width 3+ options: 2-column grid -->
  <div
    bind:this={optionsRailRef}
    class="overflow-hidden rounded-sm border border-neutral-300 bg-neutral-300 {fullWidth
      ? 'w-full'
      : 'w-fit'} {useResponsiveGrid
      ? 'max-lg:grid max-lg:auto-rows-fr max-lg:grid-cols-2 max-lg:gap-px lg:flex'
      : 'flex'}"
  >
    {#each options as option, index}
      <button
        type="button"
        onclick={() => (value = option.id)}
        onkeydown={(event) => handleOptionKeydown(event, index)}
        role="radio"
        aria-checked={value === option.id}
        tabindex={value === option.id || (selectedIndex === -1 && index === 0) ? 0 : -1}
        class="min-w-0 px-3 py-1 text-sm leading-tight max-lg:text-base {fullWidth
          ? useResponsiveGrid
            ? 'lg:flex-1'
            : 'flex-1'
          : ''} {index > 0
          ? useResponsiveGrid
            ? 'lg:border-l lg:border-neutral-300'
            : 'border-l border-neutral-300'
          : ''} text-center whitespace-nowrap {value === option.id
          ? 'bg-blue-500 text-white'
          : 'bg-white text-gray-700 hover:bg-neutral-100'}"
      >
        <span class="block whitespace-nowrap">{option.label}</span>
      </button>
    {/each}
  </div>
  <!-- Selected-option helper text lives inside the control to keep behavior
    consistent across call sites. -->
  {#if canShowOptionDescriptions && selectedOption}
    <div class="mt-1 pl-1 text-sm leading-snug text-neutral-600 max-lg:text-base">
      {selectedOption.description}
    </div>
  {/if}
</div>
