package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readRepoFile reads a file from the repository root (one directory up from this
// package). It fails the test if the file cannot be read.
func readRepoFile(t *testing.T, name string) string {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", name))
	if err != nil {
		t.Fatalf("failed to read %s: %v", name, err)
	}
	return string(bytes)
}

const sampleSource = `
script EventScr_Sample_BeginningScene {
    call(EventScr_Sample_Intro)
    playbgm(SONG_RAID)
    loadunits(1, UnitDef_Event_SampleAlly)
    wait
    move(0x18, CHARACTER_SETH, 4, 4)
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
}`

func TestCompileFE8WithRepoConfigs(t *testing.T) {
	fe8CommandConfigJSON := readRepoFile(t, "command_config.fe8.json")
	fontConfigJSON := readRepoFile(t, "font_config.json")

	out, err := CompileFE8(sampleSource, "", fe8CommandConfigJSON, fontConfigJSON, "", FE8DefaultMaxLineLength, true, nil)
	if err != nil {
		t.Fatalf("CompileFE8 returned error: %v", err)
	}

	for _, want := range []string{
		"CONST_DATA EventListScr EventScr_Sample_BeginningScene[]",
		"MUSC(SONG_RAID)",
		"LOAD1(1, UnitDef_Event_SampleAlly)",
		"MOVE(0x18, CHARACTER_SETH, 4, 4)",
		"CALL(EventScr_Sample_Intro)",
		"ENDA",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

// TestCompileFE8EmptyConfigs verifies the function is robust to empty config
// inputs (it should fall back to FE8 defaults and not panic or perform I/O).
func TestCompileFE8EmptyConfigs(t *testing.T) {
	out, err := CompileFE8(`script EventScr_Empty { end }`, "", "", "", "", 0, true, nil)
	if err != nil {
		t.Fatalf("CompileFE8 returned error: %v", err)
	}
	if !strings.Contains(out, "CONST_DATA EventListScr EventScr_Empty[]") {
		t.Errorf("output missing array declaration\n%s", out)
	}
	if !strings.Contains(out, "ENDA") {
		t.Errorf("output missing ENDA\n%s", out)
	}
}

// TestCompileFE8ParseError verifies that parse errors are surfaced.
func TestCompileFE8ParseError(t *testing.T) {
	_, err := CompileFE8(`script {`, "", "", "", "", 0, true, nil)
	if err == nil {
		t.Fatal("expected a parse error, got nil")
	}
}
