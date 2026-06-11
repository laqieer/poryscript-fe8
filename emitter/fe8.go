package emitter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/huderlem/poryscript/ast"
	"github.com/huderlem/poryscript/token"
)

// FE8CommandConfig describes how to translate a Poryscript program into a
// fireemblem8u (FE8: The Sacred Stones) event-script C header. It is loaded from
// a JSON file (see command_config.fe8.json) and maps Poryscript DSL command names
// onto the EAstdlib.h event macros used by the decomp.
type FE8CommandConfig struct {
	// HeaderIncludes are emitted as `#include "..."` lines at the top of the
	// generated header so that the FE8 event macros resolve.
	HeaderIncludes []string `json:"header_includes"`
	// ArrayType is the C type of the emitted event-list array (e.g. "EventListScr").
	ArrayType string `json:"array_type"`
	// Commands maps a Poryscript command name onto an FE8 macro definition.
	Commands map[string]FE8Command `json:"commands"`
	// CallCommand is the DSL command used to invoke another event script.
	// Its single argument is the target script's symbol name. Defaults to "call".
	CallCommand string `json:"call_command"`
	// CallMacro is the FE8 macro used to render a call (e.g. "CALL").
	CallMacro string `json:"call_macro"`
}

// FE8Command maps a single Poryscript command onto an FE8 macro.
type FE8Command struct {
	// Macro is the EAstdlib.h macro name to emit (e.g. "MUSC", "MOVE").
	Macro string `json:"macro"`
	// Args is the order in which DSL arguments are passed to the macro. Each
	// entry is the zero-based index of the DSL argument. When omitted, the DSL
	// arguments are passed straight through in order.
	Args []int `json:"args"`
}

// EmitFE8 transforms the parsed Poryscript program into a fireemblem8u event-script
// C header of the form:
//
//	CONST_DATA EventListScr EventScr_Name[] = {
//	    ...macros...
//	    ENDA
//	};
//
// Only a working subset of Poryscript is supported (see README). Text is msgId-based:
// `text(0x90E)` passes the message id straight through; inline strings are not
// supported because auto-wrapping is FE8's text system's job.
func (e *Emitter) EmitFE8(config FE8CommandConfig) (string, error) {
	if config.ArrayType == "" {
		config.ArrayType = "EventListScr"
	}
	if config.CallCommand == "" {
		config.CallCommand = "call"
	}
	if config.CallMacro == "" {
		config.CallMacro = "CALL"
	}

	var sb strings.Builder
	sb.WriteString("#pragma once\n\n")
	for _, inc := range config.HeaderIncludes {
		sb.WriteString(fmt.Sprintf("#include \"%s\"\n", inc))
	}
	if len(config.HeaderIncludes) > 0 {
		sb.WriteString("\n")
	}

	// Inline text statements are unsupported in the FE8 PoC; text is msgId-based.
	for _, text := range e.program.Texts {
		return "", fmt.Errorf("inline text '%s' is not supported by the FE8 target; pass a message id directly, e.g. text(0x90E)", text.Name)
	}

	first := true
	for _, stmt := range e.program.TopLevelStatements {
		scriptStmt, ok := stmt.(*ast.ScriptStatement)
		if !ok {
			return "", fmt.Errorf("the FE8 target only supports 'script' statements, but got '%s'", stmt.TokenLiteral())
		}
		if !first {
			sb.WriteString("\n")
		}
		first = false
		out, err := e.emitFE8Script(scriptStmt, config)
		if err != nil {
			return "", err
		}
		sb.WriteString(out)
	}
	return sb.String(), nil
}

func (e *Emitter) emitFE8Script(scriptStmt *ast.ScriptStatement, config FE8CommandConfig) (string, error) {
	chunks, err := buildScriptChunks(scriptStmt)
	if err != nil {
		return "", err
	}

	// Order the chunks so that fall-throughs are exploited where possible, exactly
	// like the pokeemerald renderer.
	var chunkIDs []int
	if e.optimize {
		chunkIDs = optimizeChunkOrder(chunks)
	} else {
		chunkIDs = make([]int, 0, len(chunks))
		for k := range chunks {
			chunkIDs = append(chunkIDs, k)
		}
		sort.Ints(chunkIDs)
	}

	// Render each chunk body, tracking which chunks are actually jumped to so we
	// only emit the LABEL() markers that are needed.
	chunkBodies := make(map[int]*strings.Builder)
	jumpChunks := make(map[int]bool)
	registerJumpChunk := func(chunkID int) { jumpChunks[chunkID] = true }

	for i, chunkID := range chunkIDs {
		var body strings.Builder
		chunkBodies[chunkID] = &body
		nextChunkID := -1
		if i < len(chunkIDs)-1 {
			nextChunkID = chunkIDs[i+1]
		}
		c := chunks[chunkID]
		if err := renderFE8Statements(&body, c, config); err != nil {
			return "", err
		}
		if err := renderFE8Branching(&body, c, nextChunkID, registerJumpChunk); err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CONST_DATA %s %s[] = {\n", config.ArrayType, scriptStmt.Name.Value))
	for _, chunkID := range chunkIDs {
		// Chunk 0 is the array entry point and never needs an explicit label.
		if chunkID != 0 && jumpChunks[chunkID] {
			sb.WriteString(fmt.Sprintf("LABEL(0x%X)\n", chunkID))
		}
		sb.WriteString(chunkBodies[chunkID].String())
	}
	sb.WriteString("};\n")
	return sb.String(), nil
}

func renderFE8Statements(sb *strings.Builder, c *chunk, config FE8CommandConfig) error {
	for _, stmt := range c.statements {
		switch s := stmt.(type) {
		case *ast.CommandStatement:
			line, err := renderFE8Command(s, config)
			if err != nil {
				return err
			}
			sb.WriteString(line)
		case *ast.LabelStatement:
			// User-defined labels are not addressable in the numeric FE8 label space.
			return fmt.Errorf("user-defined label '%s' is not supported by the FE8 target; use if/elif/else for control flow", s.Name.Value)
		default:
			return fmt.Errorf("could not render statement '%s' for the FE8 target", stmt.TokenLiteral())
		}
	}
	return nil
}

// renderFE8Command translates a single DSL command into its FE8 macro form.
// "end"/"return" are control-flow terminators handled by the branch renderer, so
// they should never reach here as standalone non-terminal commands; if they do
// (e.g. mid-chunk), they are rendered as ENDA.
func renderFE8Command(s *ast.CommandStatement, config FE8CommandConfig) (string, error) {
	name := s.Name.Value
	if name == "end" || name == "return" {
		return "\tENDA\n", nil
	}
	if name == config.CallCommand {
		if len(s.Args) != 1 {
			return "", fmt.Errorf("'%s' expects exactly 1 argument (the target script), but got %d", name, len(s.Args))
		}
		return fmt.Sprintf("\t%s(%s)\n", config.CallMacro, strings.TrimSpace(s.Args[0])), nil
	}

	cmd, ok := config.Commands[name]
	if !ok {
		return "", fmt.Errorf("unknown command '%s' for the FE8 target; add it to command_config.fe8.json", name)
	}

	args, err := reorderFE8Args(name, s.Args, cmd.Args)
	if err != nil {
		return "", err
	}
	if len(args) == 0 {
		return fmt.Sprintf("\t%s\n", cmd.Macro), nil
	}
	return fmt.Sprintf("\t%s(%s)\n", cmd.Macro, strings.Join(args, ", ")), nil
}

// reorderFE8Args maps the DSL arguments into the macro argument order declared in
// the command config. When no order is declared, arguments pass through unchanged.
func reorderFE8Args(name string, dslArgs []string, order []int) ([]string, error) {
	trimmed := make([]string, len(dslArgs))
	for i, a := range dslArgs {
		trimmed[i] = strings.TrimSpace(a)
	}
	if len(order) == 0 {
		return trimmed, nil
	}
	out := make([]string, 0, len(order))
	for _, idx := range order {
		if idx < 0 || idx >= len(trimmed) {
			return nil, fmt.Errorf("command '%s' references DSL argument %d, but only %d argument(s) were provided", name, idx, len(trimmed))
		}
		out = append(out, trimmed[idx])
	}
	return out, nil
}

func renderFE8Branching(sb *strings.Builder, c *chunk, nextChunkID int, registerJumpChunk func(int)) error {
	if c.branchBehavior != nil {
		switch b := c.branchBehavior.(type) {
		case *leafExpressionBranch:
			return renderFE8LeafBranch(sb, b, nextChunkID, registerJumpChunk)
		case *jump:
			if b.destChunkID != nextChunkID {
				registerJumpChunk(b.destChunkID)
				sb.WriteString(fmt.Sprintf("\tGOTO(0x%X)\n", b.destChunkID))
			}
			return nil
		case *breakContext:
			if b.destChunkID == -1 {
				sb.WriteString("\tENDA\n")
			} else if b.destChunkID != nextChunkID {
				registerJumpChunk(b.destChunkID)
				sb.WriteString(fmt.Sprintf("\tGOTO(0x%X)\n", b.destChunkID))
			}
			return nil
		default:
			// switchBranch and any other unsupported brancher. Failing loudly is
			// safer than emitting a comment that compiles to a broken script.
			return fmt.Errorf("control flow construct '%T' is not supported by the FE8 target; supported control flow is if/elif/else, while, and do-while on flag() conditions", b)
		}
	}

	// No branch behavior: handle natural return / goto.
	if c.returnID == -1 {
		sb.WriteString("\tENDA\n")
	} else if c.returnID != nextChunkID {
		registerJumpChunk(c.returnID)
		sb.WriteString(fmt.Sprintf("\tGOTO(0x%X)\n", c.returnID))
	}
	return nil
}

// renderFE8LeafBranch emits an FE8 flag check + conditional branch. FE8 has no
// inline boolean comparison: CHECK_EVENTID(flag) loads the flag's value (0 or 1)
// into EVT_SLOT_C, then BEQ/BNE compares it against EVT_SLOT_0 (zero) and jumps to
// a numeric label.
func renderFE8LeafBranch(sb *strings.Builder, b *leafExpressionBranch, nextChunkID int, registerJumpChunk func(int)) error {
	if b.preambleStatement != nil {
		// Autovar preamble commands are a pokeemerald concept; if one reaches here
		// the condition can't be lowered to an FE8 flag check.
		return fmt.Errorf("autovar conditions are not supported by the FE8 target; only flag() conditions are supported")
	}

	op := b.truthyDest.operatorExpression
	if op.Type != token.FLAG {
		return fmt.Errorf("condition '%s(...)' is not supported by the FE8 target; only flag() conditions are supported", strings.ToLower(string(op.Type)))
	}

	registerJumpChunk(b.truthyDest.id)

	// Determine whether truth means "flag is set".
	truthMeansSet := (op.Operator == token.EQ && op.ComparisonValue == token.TRUE) ||
		(op.Operator == token.NEQ && op.ComparisonValue == token.FALSE)

	sb.WriteString(fmt.Sprintf("\tCHECK_EVENTID(%s)\n", op.Operand.Literal))
	if truthMeansSet {
		// Jump to truthy when slot C != 0 (flag set).
		sb.WriteString(fmt.Sprintf("\tBNE(0x%X, EVT_SLOT_C, EVT_SLOT_0)\n", b.truthyDest.id))
	} else {
		// Jump to truthy when slot C == 0 (flag clear).
		sb.WriteString(fmt.Sprintf("\tBEQ(0x%X, EVT_SLOT_C, EVT_SLOT_0)\n", b.truthyDest.id))
	}

	if b.falseyReturnID == -1 {
		sb.WriteString("\tENDA\n")
	} else if b.falseyReturnID != nextChunkID {
		registerJumpChunk(b.falseyReturnID)
		sb.WriteString(fmt.Sprintf("\tGOTO(0x%X)\n", b.falseyReturnID))
	}
	return nil
}
