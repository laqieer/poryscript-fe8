// Package compile provides a string-based, filesystem-free compilation API for
// Poryscript. It runs the full lexer -> parser -> emitter pipeline using only
// in-memory inputs, so it can be used both by the CLI (main.go) and in
// environments without a filesystem (e.g. WebAssembly via the playground).
package compile

import (
	"encoding/json"
	"fmt"

	"github.com/huderlem/poryscript/emitter"
	"github.com/huderlem/poryscript/lexer"
	"github.com/huderlem/poryscript/parser"
)

// FE8DefaultMaxLineLength is the default line length (in pixels) used for text
// auto-formatting when none is supplied. It matches the historical default used
// by the playground.
const FE8DefaultMaxLineLength = 208

// CompileFE8 compiles Poryscript source into a fireemblem8u (FE8) event-script C
// header (a CONST_DATA EventListScr[] array). All configuration is supplied as
// in-memory JSON strings, so this function performs no file I/O.
//
//   - source is the Poryscript source code.
//   - commandConfigJSON is the standard Poryscript command config (the contents
//     of command_config.json). May be empty.
//   - fe8CommandConfigJSON is the FE8 command config that maps DSL commands onto
//     EAstdlib.h macros (the contents of command_config.fe8.json). May be empty,
//     in which case sensible FE8 defaults are used.
//   - fontConfigJSON is the font width config (the contents of font_config.json).
//     May be empty.
//   - defaultFontID overrides the default font id (empty uses the config default).
//   - maxLineLength overrides the max text line length in pixels (0 uses the
//     config default).
//   - optimize toggles compiled-script size optimization.
//   - compileSwitches sets compile-time switches (may be nil).
func CompileFE8(source, commandConfigJSON, fe8CommandConfigJSON, fontConfigJSON, defaultFontID string, maxLineLength int, optimize bool, compileSwitches map[string]string) (string, error) {
	var commandConfig parser.CommandConfig
	if len(commandConfigJSON) > 0 {
		if err := json.Unmarshal([]byte(commandConfigJSON), &commandConfig); err != nil {
			return "", fmt.Errorf("failed to parse command config: %w", err)
		}
	}

	var fe8Config emitter.FE8CommandConfig
	if len(fe8CommandConfigJSON) > 0 {
		if err := json.Unmarshal([]byte(fe8CommandConfigJSON), &fe8Config); err != nil {
			return "", fmt.Errorf("failed to parse FE8 command config: %w", err)
		}
	}

	fontConfig, err := parser.ParseFontConfig(fontConfigJSON)
	if err != nil {
		return "", fmt.Errorf("failed to parse font config: %w", err)
	}

	p := parser.New(lexer.New(source), commandConfig, "", defaultFontID, maxLineLength, compileSwitches)
	if len(fontConfigJSON) > 0 {
		p.SetFontConfig(&fontConfig)
	}

	program, err := p.ParseProgram()
	if err != nil {
		return "", err
	}

	em := emitter.New(program, optimize, false, "")
	return em.EmitFE8(fe8Config)
}
