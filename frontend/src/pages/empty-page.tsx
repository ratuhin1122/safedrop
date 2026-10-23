export function EmptyPage({ title }: { title: string }) {
  return <section aria-label={title} className="min-h-[60dvh] rounded-xl border bg-card shadow-[0_8px_40px_-24px_rgb(0_0_0/0.15)]">
    <h1 className="sr-only">{title}</h1>
  </section>;
}
