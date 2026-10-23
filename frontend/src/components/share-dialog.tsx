import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { ToastProvider } from "@/components/toast-provider";
import { ShareForm } from "@/components/share-form";

export function ShareDialog({ fileId, onClose }: { fileId: string; onClose: () => void }) {
  return <Modal label="Share file" onClose={onClose}>
    <div className="mb-5 flex items-center justify-between gap-3">
      <h2 className="text-xl font-semibold">Share file</h2>
      <Button variant="ghost" onClick={onClose} aria-label="Close share dialog">Close</Button>
    </div>
    <ToastProvider><ShareForm fileId={fileId} /></ToastProvider>
  </Modal>;
}
