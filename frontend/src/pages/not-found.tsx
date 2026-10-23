import { Link } from "react-router";
import { Button } from "@/components/ui/button";

export function NotFound() {
  return <section className="rounded-xl border bg-card px-6 py-section text-center">
    <h1 className="text-2xl font-semibold">Page not found</h1>
    <p className="mb-8 mt-3 text-muted-foreground">This address does not match a SafeDrop page.</p>
    <Button asChild><Link to="/">Back to upload</Link></Button>
  </section>;
}
