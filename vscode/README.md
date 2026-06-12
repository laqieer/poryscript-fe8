# VS Code Poryscript support for FE8

This directory makes the [Poryscript VS Code extension](https://marketplace.visualstudio.com/items?itemName=karathan.poryscript)
(`karathan.poryscript`, backed by the [poryscript-pls](https://github.com/huderlem/poryscript-pls)
language server) FE8-aware, so editing `.pory` files for the
[FE8 decomp](https://github.com/FireEmblemUniverse/fireemblem8u) gives
autocomplete for the FE8 command set.

## What works / what doesn't

- ✅ **Syntax highlighting** — works out of the box (the `.pory` grammar is
  shared and game-agnostic).
- ✅ **FE8 command completion** — provided by the generated
  [`fe8_event_macros.inc`](fe8_event_macros.inc), which lists the FE8
  poryscript commands (from `command_config.fe8.json`) in the `.macro` format
  the language server parses.
- ✅ **Symbol highlighting / completion** — `#define` constants (songs, items,
  characters, chapters, …) from your FE8 decomp headers, via `symbolIncludes`.
- ⚠️ **Inline diagnostics are lint-only, not a full compile.** With the
  template applied, poryscript-pls's diagnostics *do* read your FE8 config:
  command argument-count checks use `commandIncludes` (the generated
  `fe8_event_macros.inc`) + `commandConfigFilepath` (`command_config.fe8.json`),
  and line-length warnings use `fontConfigFilepath` — so the squiggles are
  FE8-aware. **But** the server only *lints* (parses + checks arg counts and
  line lengths); it never runs codegen, so it will not catch everything that
  actually compiling would (e.g. emitter-level errors, label/symbol resolution).
  For a real build, use the
  [`poryscript-fe8`](https://github.com/laqieer/poryscript-fe8) CLI
  (`poryscript -cc command_config.fe8.json ...`) or the Playground.

## Setup

1. **Install the extension.** In VS Code, install
   [`karathan.poryscript`](https://marketplace.visualstudio.com/items?itemName=karathan.poryscript).
   It bundles the `poryscript-pls` language server.

2. **Copy the settings template.** Copy the body of
   [`settings.template.json`](settings.template.json) into your FE8 hack /
   decomp workspace's `.vscode/settings.json` (create it if needed). The
   template is **not** an active `.vscode/` config in this repo on purpose — it
   lives here only as something you copy out.

3. **Adjust the paths** in those settings to match your project layout. The
   template's example paths assume you keep a checkout of `poryscript-fe8` at
   `tools/poryscript-fe8/` inside your FE8 workspace; change every path below to
   wherever those files actually live:
   - `languageServerPoryscript.commandConfigFilepath` → your copy of
     `command_config.fe8.json` (from `poryscript-fe8`; template:
     `tools/poryscript-fe8/command_config.fe8.json`).
   - `languageServerPoryscript.fontConfigFilepath` → your copy of
     `font_config.json` (template: `tools/poryscript-fe8/font_config.json`).
   - `languageServerPoryscript.commandIncludes` → the generated
     `fe8_event_macros.inc` from this `vscode/` directory (template:
     `tools/poryscript-fe8/vscode/fe8_event_macros.inc`).
   - `languageServerPoryscript.symbolIncludes` → your FE8 decomp constant
     headers (e.g. `include/constants/*.h`). Add one entry per header you want
     completions for.

   Paths are resolved relative to the `.pory` file's workspace folder unless
   absolute.

4. **Reload** the VS Code window so the language server picks up the settings.

## Regenerating the includes

`fe8_event_macros.inc` is generated from the repo-root `command_config.fe8.json`.
Whenever that config changes (commands added/removed, arg counts changed),
regenerate it:

```sh
python3 vscode/generate_includes.py
```

The generator is deterministic — re-running with an unchanged config produces a
byte-identical file. It emits one `.macro <command> param0:req, … .endm` block
per poryscript command (parameter count matching each command's `args`), plus
the `call` command.

## Credits

- The Poryscript language and compiler, the
  [`poryscript-pls`](https://github.com/huderlem/poryscript-pls) language
  server, and the VS Code extension are by
  [huderlem](https://github.com/huderlem) and
  [karathan](https://marketplace.visualstudio.com/publishers/karathan).
- This directory only adds an FE8 command-includes generator and a settings
  template on top of that work.
