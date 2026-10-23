import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
vi.mock("@/api/files", () => ({ listFiles: vi.fn(async () => ({ items: [], next_cursor: "" })) }));
import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "vitest";
import { App } from "@/app";
import { AuthProvider } from "@/components/auth-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { getAccessToken, setAccessToken, sessionKey } from "@/api/session";

vi.mock("@/lib/env", () => ({ getApiBase: () => "/v1", turnstileSiteKey: "" }));

function page(path: string) {
  return render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })}><ThemeProvider><AuthProvider><MemoryRouter initialEntries={[path]}><App /></MemoryRouter></AuthProvider></ThemeProvider></QueryClientProvider>);
}

async function fill(password = "a long unique passphrase") {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText("Email address"), "person@example.com");
  await user.type(screen.getByLabelText("Password", { exact: true }), password);
  return user;
}

it("redirects an unauthenticated /files visit to the login form", async () => {
  page("/files");
  expect(await screen.findByRole("heading", { name: "Welcome back" })).toBeVisible();
  expect(screen.queryByRole("heading", { name: "My files" })).not.toBeInTheDocument();
});

it("validates inline, logs in once, persists the token, and returns to files", async () => {
  let complete!: (response: Response) => void;
  const fetcher = vi.fn().mockImplementation(() => new Promise<Response>((resolve) => { complete = resolve; }));
  vi.stubGlobal("fetch", fetcher);
  page("/files");
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "Sign in" }));
  expect(await screen.findByText("Enter a valid email address.")).toBeVisible();
  expect(screen.getByText("Enter your password.")).toBeVisible();
  expect(fetcher).not.toHaveBeenCalled();
  await fill();
  await user.click(screen.getByRole("button", { name: "Sign in" }));
  expect(screen.getByRole("button", { name: "Signing in…" })).toBeDisabled();
  await user.click(screen.getByRole("button", { name: "Signing in…" }));
  expect(fetcher).toHaveBeenCalledOnce();
  await act(async () => complete(new Response(JSON.stringify({ token: "session-jwt" }), { status: 200 })));
  expect(await screen.findByRole("heading", { name: "My files" })).toBeInTheDocument();
  expect(getAccessToken()).toBe("session-jwt");
  expect(sessionStorage.getItem(sessionKey)).toBe("session-jwt");
});

it("shows the API's specific login error without leaving the form", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "unauthorized", message: "Email or password is incorrect." }), { status: 401 })));
  page("/login");
  const user = await fill();
  await user.click(screen.getByRole("button", { name: "Sign in" }));
  expect(await screen.findByText("Email or password is incorrect.")).toBeVisible();
  expect(screen.getByRole("button", { name: "Sign in" })).toBeEnabled();
  expect(getAccessToken()).toBeNull();
});

it("shows password guidance and registers without loading Turnstile when unset", async () => {
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ token: "registered" }), { status: 201 }));
  vi.stubGlobal("fetch", fetcher);
  page("/register");
  const user = await fill("short");
  expect(screen.getByText(/Too short/)).toBeVisible();
  await user.click(screen.getByRole("button", { name: "Create account" }));
  expect(await screen.findByText("Password must contain 12–1024 UTF-8 bytes.")).toBeVisible();
  expect(fetcher).not.toHaveBeenCalled();
  await user.clear(screen.getByLabelText("Password", { exact: true }));
  await user.type(screen.getByLabelText("Password", { exact: true }), "a long unique passphrase");
  await user.click(screen.getByRole("button", { name: "Show password" }));
  expect(screen.getByLabelText("Password", { exact: true })).toHaveAttribute("type", "text");
  expect(screen.queryByRole("group", { name: "Security verification" })).not.toBeInTheDocument();
  await user.click(screen.getByRole("button", { name: "Create account" }));
  await waitFor(() => expect(fetcher).toHaveBeenCalledOnce());
  expect(JSON.parse(fetcher.mock.calls[0][1].body)).not.toHaveProperty("turnstile_token");
  expect(await screen.findByRole("heading", { name: "My files" })).toBeInTheDocument();
});

it("reacts to sign-out and removes persisted credentials", async () => {
  setAccessToken("existing");
  page("/files");
  const user = userEvent.setup();
  await user.click(screen.getByRole("button", { name: "User menu" }));
  await user.click(await screen.findByRole("menuitem", { name: "Sign out" }));
  expect(await screen.findByRole("heading", { name: "Welcome back" })).toBeVisible();
  expect(sessionStorage.getItem(sessionKey)).toBeNull();
});
