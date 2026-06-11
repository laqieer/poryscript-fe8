# poryscript-fe8

**Proof-of-concept** fork of [huderlem/poryscript](https://github.com/huderlem/poryscript)
that retargets its emitter to produce **[fireemblem8u](https://github.com/FireEmblemUniverse/fireemblem8u)**
(FE8: *The Sacred Stones*) decomp event scripts instead of pokeemerald assembly.

A Poryscript-style `.pory` file compiles to a C header that the decomp's event
system consumes directly:

```c
CONST_DATA EventListScr EventScr_Sample_Intro[] = {
    FADI(16)
    FlashCursor(CHARACTER_EIRIKA, 60)
    Text(0x903)
    ENDA
};
```

This is a **decomp-native alternative** for authoring events. The wider Fire Emblem
romhacking community already has mature tooling for event scripting — most notably
[Event Assembler](https://feuniverse.us/t/event-assembler/1749) and
[ColorzCore](https://github.com/FireEmblemUniverse/ColorzCore). poryscript-fe8 is
*not* a replacement for those; it is an experiment in driving the `fireemblem8u`
decomp's own `EAstdlib.h` event macros from a high-level, C-like control-flow DSL.

## What it is (and is not)

- It is the upstream Poryscript front-end (lexer → parser → game-agnostic AST)
  with a **new FE8 emitter** (`emitter/fe8.go`) bolted on. The pokeemerald emitter
  is untouched and still works (`-fe8=false`).
- It is a **proof of concept** scoped to a working subset of constructs (below).
  It is not a complete event language.

## How it works

Poryscript's pipeline is `lexer/` → `parser/` → `ast/` (game-agnostic) →
`emitter/`. Upstream the emitter produces pokeemerald `.inc` assembly. This fork
adds `emitter/fe8.go`, which reuses the shared, target-agnostic chunk/branch
control-flow engine (`buildScriptChunks`) and renders each chunk as FE8 event
macros from `EAstdlib.h`:

- **Script** `script EventScr_X { ... }` → `CONST_DATA EventListScr EventScr_X[] = { ... };`
- **Commands** are mapped to macros via `command_config.fe8.json`
  (e.g. `playbgm(SONG_RAID)` → `MUSC(SONG_RAID)`).
- **`call(EventScr_Y)`** → `CALL(EventScr_Y)`.
- **`end` / `return`** → `ENDA`.
- **`if`/`elif`/`else` on `flag(...)`** → `CHECK_EVENTID(flag)` (loads the flag's
  value into `EVT_SLOT_C`) + `BNE`/`BEQ(label, EVT_SLOT_C, EVT_SLOT_0)` + numeric
  `LABEL(n)` / `GOTO(n)`. This matches how the hand-written decomp scripts branch
  (see `src/events/prologue-eventscript.h`).
- **Text is message-id based**: `text(0x90E)` passes the message id straight
  through to `Text(0x90E)`.

### Control-flow example

```pory
script EventScr_Sample_BeginningScene {
    call(EventScr_Sample_Intro)
    playbgm(SONG_RAID)
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
```

compiles to:

```c
CONST_DATA EventListScr EventScr_Sample_BeginningScene[] = {
    CALL(EventScr_Sample_Intro)
    MUSC(SONG_RAID)
    CHECK_EVENTID(EVFLAG_TMP(8))
    BNE(0x2, EVT_SLOT_C, EVT_SLOT_0)
    Text(0x910)
LABEL(0x1)
    ENUT(0x8)
    NoFade
    ENDA
LABEL(0x2)
    ASMC(BmGuideTextSetAllGreen)
    Text(0x90F)
    GOTO(0x1)
};
```

(See `examples/sample.pory` and the generated `examples/sample.h`.)

## Supported subset

- `script` blocks → `CONST_DATA EventListScr Name[] = { ... };`
- Plain commands mapped through `command_config.fe8.json`
- `call(...)` → `CALL(...)`
- `end` / `return` → `ENDA`
- `if` / `elif` / `else` whose conditions are `flag(...)` checks
- `while` / `do...while` loops whose conditions are `flag(...)` checks
- `&&` / `||` compound flag conditions (handled by the shared branch engine)
- Message-id text: `text(0x90E)` → `Text(0x90E)`
- Auto-generated numeric `LABEL(n)` / `GOTO(n)` for control flow

## Limitations

- **Text is msgId-only.** Inline string literals (`text MyText { "Hello" }`) are
  **rejected** with an error. Authoring/wrapping/encoding actual message strings
  is the job of FE8's text/message system, not this PoC.
- **Conditions are flag-only.** `var(...) == n` and `defeated(...)` conditions
  (pokeemerald concepts) are not translated to FE8 and raise a hard error.
  FE8 slot comparisons (`SVAL`/`SADD`/`BEQ` on arbitrary slots) are not exposed
  as a DSL operator yet.
- **No `switch` lowering for FE8.** The parser accepts `switch` (shared
  front-end) but the FE8 emitter raises a hard error rather than emitting a
  broken script. `if`/`elif`/`else` and `while`/`do...while` on `flag()`
  conditions are fully lowered.
- **User-defined labels inside a script are not supported** (FE8 labels are a
  numeric, compiler-managed space). Use `if`/`elif`/`else` for control flow.
- **Command coverage is partial.** Only the commands in
  `command_config.fe8.json` are known; add more as needed. Each maps to a macro
  in the decomp's `include/EAstdlib.h`.
- Only `script` top-level statements are supported (no `mapscripts`, `movement`,
  `mart`, `raw`, etc.).

## Command config (`command_config.fe8.json`)

Maps DSL command names to `EAstdlib.h` macros and argument order:

```json
{
  "array_type": "EventListScr",
  "call_command": "call",
  "call_macro": "CALL",
  "header_includes": ["global.h", "EAstdlib.h", "..."],
  "commands": {
    "playbgm":   { "macro": "MUSC" },
    "loadunits": { "macro": "LOAD1", "args": [0, 1] },
    "move":      { "macro": "MOVE",  "args": [0, 1, 2, 3] },
    "wait":      { "macro": "ENUN" },
    "setflag":   { "macro": "ENUT" },
    "text":      { "macro": "Text" }
  }
}
```

`args` reorders the DSL arguments into macro-argument order (each entry is a
zero-based DSL argument index). Omit `args` to pass arguments through unchanged.
`header_includes` are emitted as `#include "..."` lines so the macros resolve.

## Build & run

```sh
go build -o poryscript-fe8 .

# Compile a script (FE8 target is the default for this fork):
./poryscript-fe8 -i examples/sample.pory -o examples/sample.h

# Or with explicit config:
./poryscript-fe8 -i examples/sample.pory -fcc command_config.fe8.json > examples/sample.h
```

Useful flags: `-fe8=false` falls back to the original pokeemerald emitter;
`-fcc` selects the FE8 command config; `-optimize=false` disables fall-through
reordering. Run `./poryscript-fe8 -h` for the full list.

`make sample` regenerates `examples/sample.h`.

## Validation

The generated header is validated against a **read-only** `fireemblem8u`
checkout (the decomp is never mutated):

```sh
FE8_DIR=/path/to/fireemblem8u ./check.sh
# or: make check FE8_DIR=/path/to/fireemblem8u
```

`check.sh`:

1. builds the compiler and compiles `examples/sample.pory`,
2. runs a **cpp macro-resolution check** — every emitted macro/symbol must
   resolve against the decomp's real headers:

   ```sh
   cpp -iquote $FE8_DIR/include -iquote $FE8_DIR/src \
       -I $FE8_DIR/tools/agbcc/include -nostdinc -undef check/check.c
   ```

3. if `agbcc` + `iconv` are available, runs a **full agbcc compile** exactly like
   the decomp's C build rule, producing real ARM assembly:

   ```sh
   cpp ... check/check.c | iconv -f UTF-8 -t CP932 \
     | $FE8_DIR/tools/agbcc/bin/agbcc -mthumb-interwork -Wimplicit \
       -Wparentheses -O2 -fhex-asm -o check.s
   ```

`check/check.c` `#include`s the generated `sample.h` and forward-declares the
project-specific symbols the standalone sample references (another event script,
a unit definition, an ASM-called function) that would otherwise live elsewhere in
a real chapter.

## License & attribution

This is a fork of **[huderlem/poryscript](https://github.com/huderlem/poryscript)**
(© 2019 huderlem, MIT). The upstream MIT `LICENSE.md` is retained unchanged and
the original project README is preserved as `README.upstream.md`. Huge thanks to
huderlem and the Poryscript contributors for the front-end and the chunk-based
control-flow engine this fork reuses.

FE8 event macros and semantics come from the
[fireemblem8u](https://github.com/FireEmblemUniverse/fireemblem8u) decompilation
project (`include/EAstdlib.h`, `include/eventscript.h`).
