import { expect, it, vi } from "vitest";
import { getAccessToken, handleUnauthorized, setAccessToken } from "@/api/session";

it("clears the session and redirects to login without a redirect loop", () => {
  const assign = vi.fn();
  const location = { pathname: "/files", assign };
  vi.stubGlobal("window", { location });
  setAccessToken("expired-jwt");
  handleUnauthorized();
  expect(getAccessToken()).toBeNull();
  expect(assign).toHaveBeenCalledWith("/login");
  location.pathname = "/login";
  handleUnauthorized();
  expect(assign).toHaveBeenCalledTimes(1);
});

it("restores the session from sessionStorage after a module reload", async () => {
  sessionStorage.setItem("safedrop-token", "restored-jwt");
  vi.resetModules();
  const restored = await import("@/api/session");
  expect(restored.getAccessToken()).toBe("restored-jwt");
  restored.setAccessToken(null);
  expect(sessionStorage.getItem("safedrop-token")).toBeNull();
});

it("keeps a memory session when storage is unavailable", () => {
  const write = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("Storage blocked"); });
  try {
    setAccessToken("memory-session");
    expect(getAccessToken()).toBe("memory-session");
  } finally { write.mockRestore(); }
});
