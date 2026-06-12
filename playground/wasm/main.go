//go:build js && wasm

// Command wasm exposes the FE8 Poryscript compiler to JavaScript via a single
// global function, compileFE8(source, fe8CommandConfigJSON, fontConfigJSON).
// It is built with `GOOS=js GOARCH=wasm` and loaded by the playground site.
package main

import (
	"syscall/js"

	"github.com/huderlem/poryscript/compile"
)

// compileFE8 is the JS-facing entrypoint. It accepts three string arguments:
//
//	args[0]: the Poryscript source
//	args[1]: the FE8 command config JSON (command_config.fe8.json contents)
//	args[2]: the font config JSON (font_config.json contents)
//
// It returns a JS object of the form {output: string, error: string}. On
// success, error is the empty string; on failure, output is empty and error
// holds the message.
func compileFE8(this js.Value, args []js.Value) any {
	result := map[string]any{"output": "", "error": ""}

	if len(args) < 3 {
		result["error"] = "compileFE8 expects 3 string arguments: (source, fe8CommandConfigJSON, fontConfigJSON)"
		return js.ValueOf(result)
	}

	source := args[0].String()
	fe8CommandConfigJSON := args[1].String()
	fontConfigJSON := args[2].String()

	out, err := compile.CompileFE8(source, "", fe8CommandConfigJSON, fontConfigJSON, "", compile.FE8DefaultMaxLineLength, true, nil)
	if err != nil {
		result["error"] = err.Error()
		return js.ValueOf(result)
	}

	result["output"] = out
	return js.ValueOf(result)
}

func main() {
	js.Global().Set("compileFE8", js.FuncOf(compileFE8))
	// Block forever so the registered function stays callable.
	select {}
}
