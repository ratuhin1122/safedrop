import { act, render, screen } from "@testing-library/react";
import { expect, it, vi } from "vitest";
import { ThemeProvider } from "@/components/theme-provider";
import { Turnstile } from "@/components/turnstile";
import type { TurnstileOptions } from "@/lib/turnstile";

it("renders explicitly, invalidates expired tokens, and removes widgets on unmount", async () => {
  let options!: TurnstileOptions;
  const remove = vi.fn();
  window.turnstile = { render: vi.fn((_element, input) => { options = input; return "widget-1"; }), remove };
  const token = vi.fn();
  const view = render(<ThemeProvider><Turnstile siteKey="public-test-key" onToken={token} resetKey={0} /></ThemeProvider>);
  await act(async () => {});
  expect(options.sitekey).toBe("public-test-key");
  act(() => options.callback("verified-token"));
  expect(token).toHaveBeenLastCalledWith("verified-token");
  act(() => options["expired-callback"]());
  expect(token).toHaveBeenLastCalledWith("");
  expect(screen.getByText("Security check expired. Please verify again.")).toBeVisible();
  act(() => options["error-callback"]());
  expect(screen.getByText("Security check failed. Please try again.")).toBeVisible();
  view.unmount();
  expect(remove).toHaveBeenCalledWith("widget-1");
  delete window.turnstile;
});
