import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, vi } from "vitest";
import { setAccessToken } from "@/api/session";

// jsdom does not implement the browser pointer-capture API used by Radix swipe.
Object.assign(HTMLElement.prototype, {
  hasPointerCapture: () => false,
  setPointerCapture: () => {},
  releasePointerCapture: () => {},
});

Object.defineProperty(window, "matchMedia", { writable: true, value: vi.fn().mockImplementation((query: string) => ({
  matches: false, media: query, onchange: null,
  addEventListener: vi.fn(), removeEventListener: vi.fn(), dispatchEvent: vi.fn(),
})) });

afterEach(() => {
  cleanup();
  localStorage.clear();
  setAccessToken(null);
  document.documentElement.classList.remove("dark");
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});
