import { Route, Routes } from "react-router";
import { Layout } from "@/components/layout";
import { EmptyPage } from "@/pages/empty-page";
import { NotFound } from "@/pages/not-found";
import { AuthPage } from "@/pages/auth-page";
import { ProtectedRoute } from "@/components/protected-route";

export function App() {
  return <Routes>
    <Route element={<Layout />}>
      <Route index element={<EmptyPage title="Upload" />} />
      <Route element={<ProtectedRoute />}>
        <Route path="files" element={<EmptyPage title="My files" />} />
      </Route>
      <Route path="login" element={<AuthPage key="login" />} />
      <Route path="register" element={<AuthPage key="register" creating />} />
      <Route path="*" element={<NotFound />} />
    </Route>
  </Routes>;
}
