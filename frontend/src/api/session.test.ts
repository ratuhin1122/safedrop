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
