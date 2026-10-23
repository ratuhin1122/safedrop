import { Component, type ReactNode } from "react";
import { Button } from "@/components/ui/button";

export class ErrorBoundary extends Component<{ children: ReactNode }, { failed: boolean }> {
  state = { failed: false };
  static getDerivedStateFromError() { return { failed: true }; }
  render() {
    if (!this.state.failed) return this.props.children;
    return <main className="mx-auto max-w-lg px-6 py-section">
      <section role="alert" className="rounded-xl border bg-card p-8">
        <h1 className="text-2xl font-semibold">Something went wrong</h1>
        <p className="my-5 text-sm text-muted-foreground">We couldn’t display this page. Reload to try again. Files you haven’t uploaded will need to be selected again.</p>
        <Button onClick={() => window.location.reload()}>Reload page</Button>
        <a href="/" className="ml-4 rounded-sm text-sm text-primary underline">Back to upload</a>
      </section>
    </main>;
  }
}
