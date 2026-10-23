export const fileId = "11111111-1111-4111-8111-111111111111";
export const shareId = "22222222-2222-4222-8222-222222222222";
export const expires = "2099-01-01T00:00:00Z";
export const fileFixture = {
  id: fileId, original_name: "image.jpg", mime: "image/webp", size: 100,
  width: 1200, height: 800, status: "ready", created_at: "2026-10-23T00:00:00Z", reject_reason: null,
  variants: { w375: { width: 375, height: 250, size: 50, mime: "image/webp", url: `/v1/files/${fileId}/variants/375` },
    w768: null, w1200: null },
};
