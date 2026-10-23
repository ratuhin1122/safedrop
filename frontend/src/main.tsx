import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App } from "@/app";
import { ThemeProvider } from "@/components/theme-provider";
import { ToastProvider } from "@/components/toast-provider";
import { AuthProvider } from "@/components/auth-provider";
import "@/styles.css";

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: 1, refetchOnWindowFocus: false } } });
const clearSessionQueries = () => queryClient.clear();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <ThemeProvider><ToastProvider><AuthProvider onSessionChange={clearSessionQueries}>
        <BrowserRouter><App /></BrowserRouter>
      </AuthProvider></ToastProvider></ThemeProvider>
    </QueryClientProvider>
  </StrictMode>,
);
