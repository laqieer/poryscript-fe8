// Visual verification: loads the playground in headless Chromium, waits for the
// wasm compiler to initialize, triggers a compile of the default sample, asserts
// the output pane contains FE8 EventListScr output, and writes a screenshot.
//
// Requires a static server running at http://localhost:8099 serving
// playground/site, and `npx playwright install chromium`.

import { chromium } from "playwright";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const screenshotPath = join(here, "site", "preview.png");
const url = process.env.PLAYGROUND_URL || "http://localhost:8099/";

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1280, height: 720 } });

page.on("console", (msg) => console.log("[page]", msg.text()));
page.on("pageerror", (err) => console.log("[pageerror]", err.message));

await page.goto(url, { waitUntil: "load" });

// Wait for the wasm compiler to register and the Compile button to enable.
await page.waitForFunction(() => typeof globalThis.compileFE8 === "function", null, { timeout: 30000 });
await page.waitForSelector("#compile-button:not([disabled])", { timeout: 30000 });

// Trigger an explicit compile (live compile also runs, but be deterministic).
await page.click("#compile-button");

// The output editor is a CodeMirror instance; read its value.
await page.waitForFunction(() => {
    const cm = document.querySelectorAll(".CodeMirror")[1];
    return cm && cm.CodeMirror && cm.CodeMirror.getValue().includes("EventListScr");
}, null, { timeout: 30000 });

const outputText = await page.evaluate(() => {
    const cm = document.querySelectorAll(".CodeMirror")[1];
    return cm.CodeMirror.getValue();
});

if (!outputText.includes("EventListScr")) {
    console.error("FAIL: output pane does not contain EventListScr");
    await browser.close();
    process.exit(1);
}

await page.screenshot({ path: screenshotPath, fullPage: false });
console.log("PASS: output pane contains EventListScr. Screenshot saved to", screenshotPath);

await browser.close();
process.exit(0);
