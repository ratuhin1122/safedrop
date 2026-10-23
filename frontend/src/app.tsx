import { Route, Routes } from "react-router";
import { Layout } from "@/components/layout";
import { EmptyPage } from "@/pages/empty-page";
import { NotFound } from "@/pages/not-found";

export function App() {
  return <Routes>
    <Route element={<Layout />}>
      <Route index element={<EmptyPage title="Upload" />} />
      <Route path="files" element={<EmptyPage title="My files" />} />
      <Route path="login" element={<EmptyPage title="Sign in" />} />
      <Route path="register" element={<EmptyPage title="Create account" />} />
      <Route path="*" element={<NotFound />} />
    </Route>
  </Routes>;
}
