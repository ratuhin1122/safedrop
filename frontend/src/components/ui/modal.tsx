import { useEffect, useRef, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export function Modal({ children, label, onClose, sheet = false }: {
  children: ReactNode; label: string; onClose: () => void; sheet?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const dialog = ref.current!;
    const focus = document.activeElement as HTMLElement | null;
    dialog.showModal();


    return () => { dialog.close(); focus?.focus(); };
  }, []);
  return <dialog ref={ref} aria-label={label} onCancel={(event) => { event.preventDefault(); onClose(); }}
    onClick={(event) => { if (event.target === event.currentTarget) onClose(); }}
    className={cn("max-h-[90dvh] w-[calc(100%_-_2rem)] max-w-md overflow-y-auto rounded-xl border bg-card p-6 text-foreground shadow-xl backdrop:bg-black/40",
      sheet && "fixed inset-x-0 bottom-0 top-auto m-0 max-h-[90dvh] w-full max-w-none rounded-b-none sm:inset-y-0 sm:left-auto sm:right-0 sm:max-h-dvh sm:w-[30rem] sm:rounded-none")}>
    <div>{children}</div>
  </dialog>;
}
