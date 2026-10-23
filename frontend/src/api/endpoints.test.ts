import { expect, it, vi } from "vitest";
import { login, register } from "@/api/auth";
import { deleteFile, getFile, listFiles, uploadFile } from "@/api/files";
import { createShare, deleteShare, listShares } from "@/api/shares";
import { getHealth, getUsage } from "@/api/usage";
import { getAccessToken, setAccessToken } from "@/api/session";
import { expires, fileFixture, fileId, shareId } from "@/api/fixtures.test-data";

function reply(payload: unknown, status = 200) {
  vi.stubEnv("VITE_API_URL", "/proxy/v1");
  const fetcher = vi.fn().mockImplementation(async () => new Response(status === 204 ? null : JSON.stringify(payload), { status }));
  vi.stubGlobal("fetch", fetcher);
  return fetcher;
}

it("registers and logs in without an old JWT, then retains the returned token", async () => {
  const fetcher = reply({ token: "new-jwt" });
  setAccessToken("old-jwt");
  await register({ email: "a@example.com", password: "long password", turnstile_token: "challenge" });
  expect(fetcher.mock.calls[0][0]).toBe("/proxy/v1/auth/register");
  expect(new Headers(fetcher.mock.calls[0][1].headers).get("Authorization")).toBeNull();
  expect(JSON.parse(fetcher.mock.calls[0][1].body)).toMatchObject({ turnstile_token: "challenge" });
  await login({ email: "a@example.com", password: "long password" });
  expect(fetcher.mock.calls[1][0]).toBe("/proxy/v1/auth/login");
  expect(getAccessToken()).toBe("new-jwt");
});

it("uploads one multipart file without a manually set content type", async () => {
  const fetcher = reply({ id: fileId, status: "uploaded" }, 201);
  const file = new File(["small fixture"], "photo.png", { type: "image/png" });
  expect(await uploadFile(file)).toEqual({ id: fileId, status: "uploaded" });
  const init = fetcher.mock.calls[0][1];
  expect(init.body.getAll("file")).toHaveLength(1);
  expect(init.body.get("file").name).toBe("photo.png");
  expect(new Headers(init.headers).has("Content-Type")).toBe(false);
});

it("validates and encodes pagination, then reads and deletes owned files", async () => {
  const fetcher = reply({ items: [fileFixture], next_cursor: "" });
  const page = await listFiles({ cursor: "a+b=", limit: 10 });
  expect(page.items[0].variants.w375?.width).toBe(375);
  expect(fetcher.mock.calls[0][0]).toBe("/proxy/v1/files?cursor=a%2Bb%3D&limit=10");
  reply(fileFixture);
  expect((await getFile(fileId)).id).toBe(fileId);
  const remove = reply(undefined, 204);
  expect(await deleteFile(fileId)).toBeUndefined();
  expect(remove.mock.calls[0][1].method).toBe("DELETE");
  expect(() => getFile("bad-id")).toThrow();
});

it("creates, lists, and revokes shares with validated settings", async () => {
  const fetcher = reply({ url: "/s/public-token", expires_at: expires }, 201);
  await createShare(fileId, { expires_at: expires, max_downloads: 1 });
  expect(fetcher.mock.calls[0][0]).toBe(`/proxy/v1/files/${fileId}/share`);
  reply({ items: [{ id: shareId, file_id: fileId, expires_at: expires, max_downloads: 1, download_count: 0, password_protected: false }] });
  expect((await listShares(fileId)).items[0].id).toBe(shareId);
  const remove = reply(undefined, 204);
  await deleteShare(shareId);
  expect(remove.mock.calls[0][0]).toBe(`/proxy/v1/shares/${shareId}`);
  expect(() => createShare(fileId, { expires_at: "2000-01-01T00:00:00Z" })).toThrow();
});

it("loads usage and reaches public health outside the v1 base", async () => {
  reply({ bytes_today: 0, bytes_limit: 1000, uploads_this_hour: 0, uploads_limit: 20 });
  expect((await getUsage()).uploads_limit).toBe(20);
  const fetcher = reply({ status: "ok" });
  await getHealth();
  expect(fetcher.mock.calls[0][0]).toBe("/proxy/healthz");
});
