import * as Toast from "@radix-ui/react-toast";
import { useCallback, useRef, useState, type ReactNode } from "react";
import { Icon } from "@/components/icons";
import { ToastContext, type ToastInput } from "@/hooks/use-toast";
import { cn } from "@/lib/utils";

type Notice = ToastInput & { id: number };

export function ToastProvider({ children }: { children: ReactNode }) {
  const [notices, setNotices] = useState<Notice[]>([]);
  const next = useRef(0);
  const toast = useCallback((input: ToastInput) => {
    setNotices((items) => [...items, { ...input, id: next.current++ }]);
  }, []);
  const dismiss = (id: number) => setNotices((items) => items.filter((item) => item.id !== id));

  return <ToastContext.Provider value={toast}>
    <Toast.Provider duration={6000} swipeDirection="right" label="Notification">
      {children}
      {notices.map((notice) => <Toast.Root key={notice.id} defaultOpen
        onOpenChange={(open) => { if (!open) { notice.onDismiss?.(); dismiss(notice.id); } }}
        type={notice.variant === "destructive" ? "foreground" : "background"}
        className={cn("flex items-start gap-4 rounded-lg border bg-card p-5 text-card-foreground shadow-lg",
          notice.variant === "destructive" && "border-destructive")}>
        <div className="min-w-0 flex-1 break-words">
          <Toast.Title className="text-sm font-semibold">{notice.title}</Toast.Title>
          {notice.description && <Toast.Description className="mt-1 text-sm text-muted-foreground">
            {notice.description}
          </Toast.Description>}
        </div>
        {notice.action && <Toast.Action asChild altText={notice.action.label}>
          <button className="min-h-11 rounded-full px-3 text-sm font-medium focus-visible:outline-2 focus-visible:outline-ring" onClick={notice.action.onClick}>{notice.action.label}</button>
        </Toast.Action>}
        <Toast.Close aria-label="Dismiss notification" className="-m-2 grid size-11 shrink-0 place-items-center rounded-full hover:bg-accent">
          <Icon name="close" className="size-4" />
        </Toast.Close>
      </Toast.Root>)}
      <Toast.Viewport label="Notifications (press F8)" className="fixed inset-x-0 bottom-0 z-50 m-0 flex max-h-dvh list-none flex-col gap-3 overflow-y-auto p-4 outline-none sm:left-auto sm:w-96 sm:p-6" />
    </Toast.Provider>
  </ToastContext.Provider>;
}
