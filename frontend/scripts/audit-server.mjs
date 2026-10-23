import { createServer } from "node:http";
import { createReadStream } from "node:fs";
import { readFile, stat } from "node:fs/promises";
import { resolve, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(fileURLToPath(new URL("../dist/", import.meta.url)));
const base = {
  original_name: "Meadow.webp", mime: "image/webp", size: 102400, width: 1200, height: 800,
  created_at: "2026-10-23T00:00:00Z", reject_reason: null,
  variants: { w375: null, w768: null, w1200: null },
};
export function auditServer() {
  const requests = new Set();
  const server = createServer(async (req, res) => {
    const url = new URL(req.url, "http://localhost");
    requests.add(url.pathname);
    const json = (value) => { res.setHeader("Content-Type", "application/json"); res.end(JSON.stringify(value)); };
    if (url.pathname === "/v1/me/usage") return json({ bytes_today: 850000, bytes_limit: 1000000, uploads_this_hour: 7, uploads_limit: 20 });
    if (url.pathname === "/v1/files") return json({ items: ["ready", "scanning", "rejected"].map((status, i) => ({
      ...base, id: "11111111-1111-4111-8111-" + String(i + 1).padStart(12, "0"), status,
      original_name: ["Meadow.webp", "Forest.png", "Rejected.jpg"][i], reject_reason: status === "rejected" ? "Invalid image" : null,
    })), next_cursor: "" });
    const path = resolve(root, "." + decodeURIComponent(url.pathname));
    if (path !== root && !path.startsWith(root + sep)) { res.writeHead(403); res.end(); return; }
    if (url.pathname.startsWith("/assets/")) {
      try {
        await stat(path);
        res.setHeader("Content-Type", extname(path) === ".css" ? "text/css" : "text/javascript");
        createReadStream(path).pipe(res);
      } catch { res.writeHead(404); res.end(); }
      return;
    }
    const html = await readFile(resolve(root, "index.html"), "utf8");
    const theme = url.searchParams.get("theme") === "dark" ? "dark" : "light";
    // Synthetic account and API data are injected only by this isolated audit server.
    const setup = '<script>sessionStorage.setItem("safedrop-token","audit-fixture-only");localStorage.setItem("safedrop-theme","' + theme + '")</script>';
    res.setHeader("Content-Type", "text/html");
    res.end(html.replace("<head>", "<head>" + setup));
  });
  server.fixtureRequests = requests;
  return server;
}
