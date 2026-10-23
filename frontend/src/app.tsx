import { lazy, Suspense } from "react";
import { Route, Routes, useLocation } from "react-router";
import { Layout } from "@/components/layout";
import { ProtectedRoute } from "@/components/protected-route";
import { ErrorBoundary } from "@/components/error-boundary";
import { OfflineBanner } from "@/components/offline-banner";
import { RouteLoading } from "@/components/route-loading";

const UploadPage = lazy(() => import("@/pages/upload-page").then((m) => ({ default: m.UploadPage })));
const FilesPage = lazy(() => import("@/pages/files-page").then((m) => ({ default: m.FilesPage })));
const SharesPage = lazy(() => import("@/pages/shares-page").then((m) => ({ default: m.SharesPage })));
const AuthPage = lazy(() => import("@/pages/auth-page").then((m) => ({ default: m.AuthPage })));
const NotFound = lazy(() => import("@/pages/not-found").then((m) => ({ default: m.NotFound })));
const PublicSharePage = lazy(() => import("@/pages/public-share-page").then((m) => ({ default: m.PublicSharePage })));

export function App() {
  const location = useLocation();
  return <ErrorBoundary key={location.pathname}>
    <OfflineBanner />
    <Suspense fallback={<main className="p-page"><RouteLoading /></main>}>
      <Routes>
        <Route path="s/:token" element={<PublicSharePage />} />
        <Route element={<Layout />}>
          <Route index element={<UploadPage />} />
          <Route element={<ProtectedRoute />}>
            <Route path="files" element={<FilesPage />} />
            <Route path="shares" element={<SharesPage />} />
          </Route>
          <Route path="login" element={<AuthPage key="login" />} />
          <Route path="register" element={<AuthPage key="register" creating />} />
          <Route path="*" element={<NotFound />} />
        </Route>
      </Routes>
    </Suspense>
  </ErrorBoundary>;
}
