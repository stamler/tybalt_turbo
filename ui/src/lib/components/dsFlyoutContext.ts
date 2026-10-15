import { getContext, setContext } from "svelte";

// Lets a DSMenuItem close the DSFlyout it sits in and tell it that it exists.
export interface DSFlyoutContext {
  close: () => void;
  register: () => () => void;
}

const key = Symbol("ds-flyout");

export function setFlyoutContext(context: DSFlyoutContext) {
  setContext(key, context);
}

export function getFlyoutContext(): DSFlyoutContext | undefined {
  return getContext<DSFlyoutContext | undefined>(key);
}
