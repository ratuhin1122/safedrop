import { useEffect, useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { createShare } from "@/api/shares";
import { shareFormSchema, shareInput, publicShareUrl, type ShareFormValues } from "@/lib/share-form";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ShareResult } from "@/components/share-result";

export function ShareForm({ fileId }: { fileId: string }) {
  const client = useQueryClient();
  const abort = useRef<AbortController | null>(null);
  const [result, setResult] = useState<{ url: string; expires_at: string }>();
  useEffect(() => () => abort.current?.abort(), []);
  const { register, watch, handleSubmit, setError, clearErrors, formState: { errors, isSubmitting } } = useForm<ShareFormValues>({
    resolver: zodResolver(shareFormSchema), defaultValues: { preset: "week", customDate: "", password: "", maxDownloads: "" },
  });
  const preset = watch("preset");
  async function submit(values: ShareFormValues) {
    clearErrors("root");
    // Revalidate against the current time, including time spent on the form.
    const parsed = shareFormSchema.safeParse(values);
    if (!parsed.success) { setError("customDate", { message: "Choose a date and time in the future." }); return; }
    abort.current = new AbortController();
    try {
      const response = await createShare(fileId, shareInput(parsed.data), abort.current.signal);
      if (abort.current.signal.aborted) return;
      setResult({ ...response, url: publicShareUrl(response.url) });
      void client.invalidateQueries({ queryKey: ["shares", fileId] });
    } catch (error) {
      if (!abort.current.signal.aborted) setError("root", { message: error instanceof Error ? error.message : "Could not create a share link." });
    }
  }
  if (result) return <ShareResult url={result.url} expiresAt={result.expires_at} />;
  return <form noValidate onSubmit={handleSubmit(submit)} className="space-y-5" aria-busy={isSubmitting}>
    <fieldset disabled={isSubmitting} className="min-w-0 space-y-5">
      <legend className="sr-only">Share options</legend>
      <fieldset><legend className="mb-2 text-sm font-medium">Link expires after</legend>
        <div className="grid grid-cols-2 gap-2">
          {([["hour", "1 hour"], ["day", "1 day"], ["week", "7 days"], ["custom", "Custom date"]] as const).map(([value, label]) =>
            <label key={value} className="flex min-h-11 cursor-pointer items-center gap-2 rounded-md border p-3 text-sm has-checked:border-primary has-checked:bg-accent">
              <input type="radio" value={value} {...register("preset")} className="accent-primary focus-visible:outline-2 focus-visible:outline-ring" />{label}
            </label>)}
        </div>
      </fieldset>
      {preset === "custom" && <div>
        <label htmlFor="share-date" className="mb-2 block text-sm font-medium">Expiry date and time</label>
        <Input id="share-date" type="datetime-local" {...register("customDate")} aria-invalid={Boolean(errors.customDate)}
          aria-describedby="share-date-help share-date-error" />
        <p id="share-date-help" className="mt-1 text-xs text-muted-foreground">In your local time zone.</p>
        <p id="share-date-error" role={errors.customDate ? "alert" : undefined} className="mt-1 text-sm text-destructive">{errors.customDate?.message}</p>
      </div>}
      <div><label htmlFor="share-password" className="mb-2 block text-sm font-medium">Password (optional)</label>
        <Input id="share-password" type="password" autoComplete="new-password" {...register("password")}
          aria-invalid={Boolean(errors.password)} aria-describedby="share-password-error" />
        <p id="share-password-error" role={errors.password ? "alert" : undefined} className="mt-1 text-sm text-destructive">{errors.password?.message}</p>
      </div>
      <div><label htmlFor="share-downloads" className="mb-2 block text-sm font-medium">Max downloads (optional)</label>
        <Input id="share-downloads" type="number" min={1} step={1} inputMode="numeric" placeholder="Unlimited" {...register("maxDownloads")}
          aria-invalid={Boolean(errors.maxDownloads)} aria-describedby="share-downloads-error" />
        <p id="share-downloads-error" role={errors.maxDownloads ? "alert" : undefined} className="mt-1 text-sm text-destructive">{errors.maxDownloads?.message}</p>
      </div>
    </fieldset>
    {errors.root && <p role="alert" className="text-sm text-destructive">{errors.root.message}</p>}
    <Button className="w-full" type="submit" disabled={isSubmitting}>{isSubmitting ? "Creating link…" : "Create link"}</Button>
  </form>;
}
