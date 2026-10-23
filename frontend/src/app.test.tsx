import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import { App } from "@/app";
import { ThemeProvider } from "@/components/theme-provider";
import { ToastProvider } from "@/components/toast-provider";
import { useToast } from "@/hooks/use-toast";
import { AuthProvider } from "@/components/auth-provider";

function shell() {
  return render(<ThemeProvider><ToastProvider><AuthProvider><MemoryRouter><App /></MemoryRouter></AuthProvider></ToastProvider></ThemeProvider>);
}

describe("layout", () => {
  it("renders empty content and persists a keyboard-operated theme choice", async () => {
    const user = userEvent.setup();
    shell();
    expect(screen.getByRole("main")).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Main navigation" })).toBeInTheDocument();
    const toggle = screen.getByRole("button", { name: "Switch to dark mode" });
    toggle.focus();
    await user.keyboard("{Enter}");
    expect(document.documentElement).toHaveClass("dark");
    expect(localStorage.getItem("safedrop-theme")).toBe("dark");
    await user.click(screen.getByRole("button", { name: "Switch to light mode" }));
    expect(document.documentElement).not.toHaveClass("dark");
  });

  it("loads dark mode from the saved preference", () => {
    localStorage.setItem("safedrop-theme", "dark");
    shell();
    expect(document.documentElement).toHaveClass("dark");
  });

  it("opens and navigates the user menu with the keyboard", async () => {
    const user = userEvent.setup();
    shell();
    screen.getByRole("button", { name: "User menu" }).focus();
    await user.keyboard("{Enter}");
    expect(await screen.findByRole("menu")).toBeVisible();
    await user.keyboard("{End}{Enter}");
    expect(await screen.findByRole("heading", { name: "Create account" })).toBeInTheDocument();
  });
});

function ToastHarness() {
  const { toast } = useToast();
  return <button onClick={() => toast({ title: "Upload failed", description: "This file exceeds 25 MiB.", variant: "destructive" })}>
    Notify
  </button>;
}

it("shows a specific toast message and lets the user dismiss it", async () => {
  const user = userEvent.setup();
  render(<ToastProvider><ToastHarness /></ToastProvider>);
  await user.click(screen.getByRole("button", { name: "Notify" }));
  expect(screen.getByText("This file exceeds 25 MiB.")).toBeVisible();
  await user.click(screen.getByRole("button", { name: "Dismiss notification" }));
  await waitFor(() => expect(screen.queryByRole("button", { name: "Dismiss notification" })).not.toBeInTheDocument());
});
