# FE8 Poryscript Playground

An in-browser playground for the FE8 (fireemblem8u) target of
[poryscript-fe8](https://github.com/laqieer/poryscript-fe8). Type a `.pory`
script and see the compiled FE8 `EventListScr[]` event-script C header live, with
all compilation running locally in your browser via WebAssembly.

Live demo: https://laqieer.github.io/poryscript-fe8/

## Attribution

This playground is adapted from the
[Poryscript Playground](https://github.com/huderlem/poryscript-playground) by
[huderlem](https://github.com/huderlem) (MIT). The editor shell, WASM bridge
pattern, CSS layout, and the `?code=` URL-sharing approach come from that
project. Thanks to huderlem for the original Poryscript and its playground.

This repository (poryscript-fe8) remains MIT-licensed; see the root
[LICENSE.md](../LICENSE.md).

## Layout

```
playground/
  wasm/main.go        WASM entrypoint; exposes globalThis.compileFE8(...)
  site/               Static site served by GitHub Pages
    index.html        UI (CodeMirror input + read-only output, Compile button)
    styles.css        Layout
    command_config.fe8.json   Default FE8 command config (DSL -> EAstdlib macros)
    font_config.json          Default font config
    lib/              Vendored CodeMirror + lz-string (MIT, with LICENSE files)
    main.wasm         (generated) compiled compiler
    wasm_exec.js      (generated) Go WASM runtime glue
    preview.png       Screenshot used for documentation
  test_wasm.mjs       Headless Node harness asserting compileFE8 output
  screenshot.mjs      Playwright headless visual check + screenshot
```

`main.wasm` and `wasm_exec.js` are build artifacts (git-ignored); the GitHub
Pages workflow regenerates them on every deploy.

## Building locally

From the repository root:

```sh
# 1. Build the WASM compiler.
GOOS=js GOARCH=wasm go build -o playground/site/main.wasm ./playground/wasm

# 2. Copy the Go WASM runtime glue.
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" playground/site/ \
  || cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" playground/site/

# 3. Serve the site.
python3 -m http.server -d playground/site 8099
# Visit http://localhost:8099/
```

## Tests

```sh
# Go unit tests (includes compile.CompileFE8).
go test ./...

# Headless Node WASM harness (objective, no browser).
node playground/test_wasm.mjs

# Playwright visual check + screenshot (needs a server on :8099).
npx playwright install chromium
python3 -m http.server -d playground/site 8099 &
node playground/screenshot.mjs
```

## How it works

The compile pipeline is shared with the CLI via the `compile` package:
`compile.CompileFE8(source, commandConfigJSON, fe8CommandConfigJSON,
fontConfigJSON, defaultFontID, maxLineLength, optimize, switches)` runs
lexer -> parser -> `EmitFE8` using only in-memory JSON config (no file I/O), so
it works under `GOOS=js GOARCH=wasm`. The WASM entrypoint registers a single
`globalThis.compileFE8(source, fe8CommandConfigJSON, fontConfigJSON)` function
that returns `{output, error}`.
