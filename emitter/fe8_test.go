package emitter

import (
	"strings"
	"testing"

	"github.com/huderlem/poryscript/lexer"
	"github.com/huderlem/poryscript/parser"
)

// fe8TestConfig mirrors the shape of command_config.fe8.json for the subset of
// commands exercised by the tests.
func fe8TestConfig() FE8CommandConfig {
	return FE8CommandConfig{
		ArrayType:      "EventListScr",
		CallCommand:    "call",
		CallMacro:      "CALL",
		HeaderIncludes: []string{"EAstdlib.h"},
		Commands: map[string]FE8Command{
			"playbgm":     {Macro: "MUSC"},
			"loadunits":   {Macro: "LOAD1", Args: []int{0, 1}},
			"wait":        {Macro: "ENUN"},
			"move":        {Macro: "MOVE", Args: []int{0, 1, 2, 3}},
			"text":        {Macro: "Text"},
			"setflag":     {Macro: "ENUT"},
			"asmcall":     {Macro: "ASMC"},
			"flashcursor": {Macro: "FlashCursor", Args: []int{0, 1}},
		},
	}
}

func compileFE8(t *testing.T, input string) string {
	t.Helper()
	p := parser.New(lexer.New(input), parser.CommandConfig{}, "", "", 0, nil)
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	e := New(program, true, false, "")
	out, err := e.EmitFE8(fe8TestConfig())
	if err != nil {
		t.Fatalf("emit error: %v", err)
	}
	return out
}

func TestFE8BasicCommands(t *testing.T) {
	out := compileFE8(t, `
script EventScr_Test {
    playbgm(SONG_RAID)
    loadunits(1, UnitDef_Foo)
    wait
    move(0x18, CHARACTER_SETH, 4, 4)
    call(EventScr_Other)
    end
}`)

	for _, want := range []string{
		"CONST_DATA EventListScr EventScr_Test[] = {",
		"\tMUSC(SONG_RAID)\n",
		"\tLOAD1(1, UnitDef_Foo)\n",
		"\tENUN\n",
		"\tMOVE(0x18, CHARACTER_SETH, 4, 4)\n",
		"\tCALL(EventScr_Other)\n",
		"\tENDA\n",
		"};\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestFE8IfElseFlag(t *testing.T) {
	out := compileFE8(t, `
script EventScr_Test {
    if (flag(EVFLAG_TMP(8))) {
        asmcall(SomeFunc)
    } else {
        text(0x910)
    }
    setflag(0x8)
    end
}`)

	// The flag check loads the flag into slot C and branches.
	if !strings.Contains(out, "CHECK_EVENTID(EVFLAG_TMP(8))") {
		t.Errorf("output missing flag check\n%s", out)
	}
	// "flag is set" means jump-to-truthy when slot C != 0.
	if !strings.Contains(out, "BNE(") {
		t.Errorf("output missing BNE branch\n%s", out)
	}
	if !strings.Contains(out, "LABEL(") {
		t.Errorf("output missing LABEL marker\n%s", out)
	}
	if !strings.Contains(out, "GOTO(") {
		t.Errorf("output missing GOTO rejoin\n%s", out)
	}
	if !strings.Contains(out, "ASMC(SomeFunc)") {
		t.Errorf("output missing if-body command\n%s", out)
	}
	if !strings.Contains(out, "Text(0x910)") {
		t.Errorf("output missing else-body command\n%s", out)
	}
}

func TestFE8InlineTextRejected(t *testing.T) {
	input := `
text MyText {
    "Hello"
}
script EventScr_Test {
    end
}`
	p := parser.New(lexer.New(input), parser.CommandConfig{}, "", "", 0, nil)
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	e := New(program, true, false, "")
	if _, err := e.EmitFE8(fe8TestConfig()); err == nil {
		t.Error("expected an error for inline text in the FE8 target, but got none")
	}
}

func TestFE8UnknownCommandRejected(t *testing.T) {
	input := `
script EventScr_Test {
    notacommand(1)
    end
}`
	p := parser.New(lexer.New(input), parser.CommandConfig{}, "", "", 0, nil)
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	e := New(program, true, false, "")
	if _, err := e.EmitFE8(fe8TestConfig()); err == nil {
		t.Error("expected an error for an unknown command, but got none")
	}
}

// emitFE8Err is a helper that parses and emits, returning the emit error (if any).
func emitFE8Err(t *testing.T, input string) error {
	t.Helper()
	p := parser.New(lexer.New(input), parser.CommandConfig{}, "", "", 0, nil)
	program, err := p.ParseProgram()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	e := New(program, true, false, "")
	_, err = e.EmitFE8(fe8TestConfig())
	return err
}

func TestFE8VarConditionRejected(t *testing.T) {
	err := emitFE8Err(t, `
script EventScr_Test {
    if (var(VAR_X) == 5) {
        asmcall(Foo)
    }
    end
}`)
	if err == nil {
		t.Error("expected an error for a var() condition, but got none")
	}
}

func TestFE8SwitchRejected(t *testing.T) {
	err := emitFE8Err(t, `
script EventScr_Test {
    switch (var(VAR_X)) {
        case 0: asmcall(A)
        case 1: asmcall(B)
    }
    end
}`)
	if err == nil {
		t.Error("expected an error for a switch statement, but got none")
	}
}

func TestFE8WhileFlagLoop(t *testing.T) {
	out := compileFE8(t, `
script EventScr_Test {
    while (flag(FLAG_A)) {
        asmcall(Foo)
    }
    end
}`)
	// The loop should re-check the flag and branch with a back-edge GOTO.
	if !strings.Contains(out, "CHECK_EVENTID(FLAG_A)") {
		t.Errorf("output missing loop condition check\n%s", out)
	}
	if !strings.Contains(out, "GOTO(") {
		t.Errorf("output missing loop back-edge\n%s", out)
	}
	if !strings.Contains(out, "ASMC(Foo)") {
		t.Errorf("output missing loop body\n%s", out)
	}
}
