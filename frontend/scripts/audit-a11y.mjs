import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import lighthouse from "lighthouse";
import { launch } from "chrome-launcher";
import { mkdir, writeFile } from "node:fs/promises";
import { auditServer } from "./audit-server.mjs";

const build = spawnSync(process.execPath, [fileURLToPath(new URL("../node_modules/vite/bin/vite.js", import.meta.url)), "build"], {
  cwd: fileURLToPath(new URL("../", import.meta.url)),
  env: { ...process.env, VITE_API_URL: "/v1", VITE_TURNSTILE_SITE_KEY: "" }, stdio: "inherit",
});
if (build.status !== 0) throw new Error("The fixture build failed.");
const server = auditServer();
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const base = "http://127.0.0.1:" + server.address().port;
let chrome;
const results = [];
try {
  chrome = await launch({ chromeFlags: ["--headless=new", "--no-sandbox", "--disable-gpu"] });
  await mkdir(new URL("../.lighthouse/", import.meta.url), { recursive: true });
  for (const route of ["/", "/files"]) {
    for (const theme of ["light", "dark"]) {
      const name = (route === "/" ? "upload" : "files") + "-" + theme;
      server.fixtureRequests.clear();
      const result = await lighthouse(base + route + "?theme=" + theme, {
        port: chrome.port, onlyCategories: ["accessibility"], output: "html", logLevel: "error",
        formFactor: "mobile", disableStorageReset: false,
      });
      const required = route === "/" ? "/v1/me/usage" : "/v1/files";
      if (!server.fixtureRequests.has(required)) throw new Error("Audit did not load fixture API data: " + required);
      const score = Math.round(result.lhr.categories.accessibility.score * 100);
      const failed = Object.values(result.lhr.audits).filter((audit) => audit.scoreDisplayMode === "binary" && audit.score === 0)
        .map((audit) => ({ id: audit.id, title: audit.title, details: audit.details }));
      results.push({ route, theme, score, failed });
      await writeFile(new URL("../.lighthouse/" + name + ".html", import.meta.url), result.report);
      console.log(name + ": " + score, JSON.stringify(failed));
    }
  }
  await writeFile(new URL("../.lighthouse/results.json", import.meta.url), JSON.stringify(results, null, 2));
  if (results.some((result) => result.score < 95)) process.exitCode = 1;
} finally {
  await chrome?.kill();
  await new Promise((resolve) => server.close(resolve));
}
