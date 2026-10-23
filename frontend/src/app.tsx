import { PublicSharePage } from "@/pages/public-share-page";
import { SharesPage } from "@/pages/shares-page";
import { Route, Routes } from "react-router";
import { Layout } from "@/components/layout";
import { FilesPage } from "@/pages/files-page";
import { NotFound } from "@/pages/not-found";
import { AuthPage } from "@/pages/auth-page";
import { ProtectedRoute } from "@/components/protected-route";
import { UploadPage } from "@/pages/upload-page";

export function App() {
  return <Routes>
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
  </Routes>;
}
