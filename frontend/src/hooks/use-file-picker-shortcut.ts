import { useEffect, type RefObject } from "react";

export function useFilePickerShortcut(input: RefObject<HTMLInputElement | null>, disabled: boolean) {
  useEffect(() => {
    const open = (event: KeyboardEvent) => {
      const target = event.target instanceof Element ? event.target : null;
      if (disabled || event.defaultPrevented || event.repeat || event.isComposing || event.ctrlKey || event.metaKey
        || !event.altKey || !event.shiftKey || event.key.toLowerCase() !== "u"
        || target?.closest('input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="textbox"]')
        || document.querySelector("dialog[open]")) return;
      event.preventDefault();
      input.current?.click();
    };
    window.addEventListener("keydown", open);
    return () => window.removeEventListener("keydown", open);
  }, [input, disabled]);
}
