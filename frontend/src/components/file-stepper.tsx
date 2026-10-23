import type { FileItem } from "@/api/schemas";
import { fileSteps, stepLabels, rejectionMessage } from "@/lib/file-status";
import { cn } from "@/lib/utils";

export function FileStepper({ file }: { file: Pick<FileItem, "status" | "reject_reason"> }) {
  if (file.status === "rejected") return <div role="alert" className="mt-3 rounded-md border border-destructive p-3">
    <p className="text-sm font-medium text-destructive">File rejected</p>
    <p className="mt-1 text-sm text-muted-foreground">{rejectionMessage(file.reject_reason)}</p>
  </div>;
  const current = fileSteps.indexOf(file.status);
  return <ol aria-label="File processing steps" className="mt-4 grid grid-cols-2 gap-2 sm:grid-cols-4">
    {fileSteps.map((step, index) => <li key={step} aria-current={index === current ? "step" : undefined}
      className={cn("flex items-center gap-2 rounded-md px-2 py-2 text-xs",
        index <= current ? "bg-secondary text-secondary-foreground" : "bg-muted text-muted-foreground")}>
      <span aria-hidden="true" className="grid size-6 shrink-0 place-items-center rounded-full border border-current">
        {index < current ? "✓" : index + 1}
      </span>
      <span>{stepLabels[step]}<span className="sr-only">{index < current ? ", complete" : index === current ? ", current" : ", waiting"}</span></span>
    </li>)}
  </ol>;
}
