import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { expect, it, vi } from "vitest";
import { App } from "@/app";
import { ThemeProvider } from "@/components/theme-provider";
import { AuthProvider } from "@/components/auth-provider";
import { loadTurnstile, type TurnstileOptions } from "@/lib/turnstile";

vi.mock("@/lib/env", () => ({ getApiBase: () => "/v1", turnstileSiteKey: "public-test-key" }));
vi.mock("@/lib/turnstile", () => ({ loadTurnstile: vi.fn() }));

it("requires a challenge and resets its single-use token after a failed registration", async () => {
  let options!: TurnstileOptions;
  const remove = vi.fn();
  const renderWidget = vi.fn((_element, input: TurnstileOptions) => { options = input; return "widget"; });
  vi.mocked(loadTurnstile).mockResolvedValue({ render: renderWidget, remove });
  const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ code: "conflict", message: "Account already exists." }), { status: 409 }));
  vi.stubGlobal("fetch", fetcher);
  const view = render(<ThemeProvider><AuthProvider><MemoryRouter initialEntries={["/register"]}><App /></MemoryRouter></AuthProvider></ThemeProvider>);
  const user = userEvent.setup();
  await user.type(await screen.findByLabelText("Email address"), "a@example.com");
  await user.type(screen.getByLabelText("Password", { exact: true }), "a long unique password");
  await user.click(screen.getByRole("button", { name: "Create account" }));
  expect(await screen.findByText("Complete the security check before creating an account.")).toBeVisible();
  expect(fetcher).not.toHaveBeenCalled();
  act(() => options.callback("challenge-token"));
  await user.click(screen.getByRole("button", { name: "Create account" }));
  expect(await screen.findByText("Account already exists.")).toBeVisible();
  expect(JSON.parse(fetcher.mock.calls[0][1].body).turnstile_token).toBe("challenge-token");
  await waitFor(() => expect(renderWidget).toHaveBeenCalledTimes(2));
  await user.click(screen.getByRole("button", { name: "Create account" }));
  expect(fetcher).toHaveBeenCalledOnce();
  view.unmount();
  expect(remove).toHaveBeenCalled();
});
