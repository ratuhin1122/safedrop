import { Suspense } from "react";
import { RouteLoading } from "@/components/route-loading";
import { Outlet } from "react-router";
import { Header } from "@/components/header";

export function Layout() {
  return <div className="min-h-dvh font-sans antialiased">
    <a href="#main-content" className="sr-only rounded-full bg-primary px-5 py-3 text-primary-foreground focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50">
      Skip to main content
    </a>
    <div className="mx-auto max-w-7xl">
      <Header />
      <main id="main-content" tabIndex={-1} className="px-page pb-section pt-4 focus:outline-none sm:pt-8">
        <Suspense fallback={<RouteLoading />}><Outlet /></Suspense>
      </main>
    </div>
  </div>;
}
