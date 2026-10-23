import { createContext, useContext } from "react";

export type ToastInput = { title: string; description?: string; variant?: "default" | "destructive"; action?: { label: string; onClick: () => void }; onDismiss?: () => void };
export const ToastContext = createContext<((input: ToastInput) => void) | null>(null);

export function useToast() {
  const toast = useContext(ToastContext);
  if (!toast) throw new Error("useToast must be used inside ToastProvider.");
  return { toast };
}
