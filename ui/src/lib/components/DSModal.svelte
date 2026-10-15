<script lang="ts">
  import type { Snippet } from "svelte";

  // The one modal shell. A native <dialog> opened with showModal() sits in the
  // browser's top layer, traps focus, makes the page inert and closes on
  // Escape, so no z-index or hand-built backdrop is needed.
  let {
    show = $bindable(),
    title,
    subtitle = "",
    size = "md",
    // Escape and backdrop clicks close it. Pass false while submitting.
    dismissible = true,
    // Called instead of setting show to false when the person dismisses the
    // modal, so the owner decides what dismissing means.
    onclose,
    children,
    footer,
  }: {
    show: boolean;
    title: string;
    subtitle?: string;
    size?: "sm" | "md" | "lg";
    dismissible?: boolean;
    onclose?: () => void;
    children: Snippet;
    footer?: Snippet;
  } = $props();

  const titleId = $props.id();

  const sizeClasses = {
    sm: "max-w-md",
    md: "max-w-lg",
    lg: "max-w-2xl",
  } as const;

  let dialog: HTMLDialogElement | undefined = $state();
  // A drag that starts inside the panel and ends on the backdrop is not a
  // backdrop click.
  let pressedBackdrop = false;

  $effect(() => {
    if (!dialog) return;
    if (show && !dialog.open) dialog.showModal();
    else if (!show && dialog.open) dialog.close();
  });

  function dismiss() {
    if (!dismissible) return;
    if (onclose) onclose();
    else show = false;
  }

  function handleCancel(event: Event) {
    event.preventDefault();
    dismiss();
  }

  // The browser can still close the dialog itself (e.g. a repeated Escape).
  // Keep it open when it must stay, otherwise treat it as a dismissal.
  function handleClose() {
    if (!show || !dialog) return;
    if (dismissible) dismiss();
    if (show && !dialog.open) dialog.showModal();
  }

  function handlePointerDown(event: PointerEvent) {
    pressedBackdrop = event.target === dialog;
  }

  function handleClick(event: MouseEvent) {
    if (pressedBackdrop && event.target === dialog) dismiss();
    pressedBackdrop = false;
  }
</script>

<!-- Escape is handled by the dialog's cancel event; the click is only for the backdrop. -->
<dialog
  bind:this={dialog}
  aria-labelledby={titleId}
  class="m-auto w-[calc(100%-2rem)] {sizeClasses[
    size
  ]} rounded-lg bg-white p-0 text-left shadow-xl backdrop:bg-black/50"
  oncancel={handleCancel}
  onclose={handleClose}
  onpointerdown={handlePointerDown}
  onclick={handleClick}
>
  {#if show}
    <div class="p-6">
      <h2 id={titleId} class="mb-2 text-xl font-bold">{title}</h2>
      {#if subtitle}
        <p class="mb-4 text-sm text-neutral-600">{subtitle}</p>
      {/if}
      {@render children()}
      {#if footer}
        <div class="mt-4 flex flex-wrap justify-end gap-2">
          {@render footer()}
        </div>
      {/if}
    </div>
  {/if}
</dialog>
