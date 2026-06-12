// Headless Node harness that loads the compiled main.wasm, calls the exported
// globalThis.compileFE8(...), and asserts the FE8 output looks correct.
//
// Run from the repo root after building the wasm:
//   GOOS=js GOARCH=wasm go build -o playground/site/main.wasm ./playground/wasm
//   cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" playground/site/   # or lib/wasm
//   node playground/test_wasm.mjs

import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const here = dirname(fileURLToPath(import.meta.url));
const siteDir = join(here, "site");
const repoRoot = join(here, "..");

function fail(msg) {
    console.error("FAIL:", msg);
    process.exit(1);
}

// Load the Go wasm runtime glue. It assigns globalThis.Go.
await import(join(siteDir, "wasm_exec.js"));
if (typeof globalThis.Go !== "function") {
    fail("wasm_exec.js did not define globalThis.Go");
}

const go = new globalThis.Go();
const wasmBytes = await readFile(join(siteDir, "main.wasm"));
const { instance } = await WebAssembly.instantiate(wasmBytes, go.importObject);

// go.run resolves when the wasm program exits. Our wasm blocks forever
// (select {}), so we must NOT await it; the registered function is available
// synchronously once run() has set it up.
go.run(instance);

if (typeof globalThis.compileFE8 !== "function") {
    fail("globalThis.compileFE8 was not registered by the wasm module");
}

const sampleSource = `
script EventScr_Sample_BeginningScene {
    call(EventScr_Sample_Intro)
    playbgm(SONG_RAID)
    loadunits(1, UnitDef_Event_SampleAlly)
    wait
    move(0x18, CHARACTER_SETH, 4, 4)
    flashcursor(CHARACTER_SETH, 60)
    text(0x90E)
    if (flag(EVFLAG_TMP(8))) {
        asmcall(BmGuideTextSetAllGreen)
        text(0x90F)
    } else {
        text(0x910)
    }
    setflag(0x8)
    nofade
    end
}

script EventScr_Sample_Intro {
    fadein(16)
    text(0x903)
    end
}`;

const cmdCfgJSON = await readFile(join(repoRoot, "command_config.fe8.json"), "utf8");
const fontCfgJSON = await readFile(join(repoRoot, "font_config.json"), "utf8");

const result = globalThis.compileFE8(sampleSource, cmdCfgJSON, fontCfgJSON);

if (!result || typeof result !== "object") {
    fail("compileFE8 did not return an object");
}
if (result.error) {
    fail("compileFE8 returned an error: " + result.error);
}
const output = result.output || "";

const expectations = [
    "CONST_DATA EventListScr",
    "EventScr_Sample_BeginningScene",
    "MUSC(SONG_RAID)",
    "MOVE(0x18, CHARACTER_SETH, 4, 4)",
    "ENDA",
];
for (const want of expectations) {
    if (!output.includes(want)) {
        fail(`output missing expected substring ${JSON.stringify(want)}\n--- output ---\n${output}`);
    }
}

console.log("PASS: compileFE8 produced valid FE8 EventListScr output.");
console.log("--- output (first 400 chars) ---");
console.log(output.slice(0, 400));
process.exit(0);
