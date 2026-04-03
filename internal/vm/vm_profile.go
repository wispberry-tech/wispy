//go:build groveprofile

package vm

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"grove/internal/compiler"
)

// Opcode categories for profiling.
const (
	CatVarLookup = iota // OP_LOAD, OP_GET_ATTR, OP_GET_INDEX
	CatOutput           // OP_OUTPUT, OP_OUTPUT_RAW
	CatFilter           // OP_FILTER
	CatLoop             // OP_FOR_INIT, OP_FOR_BIND_1, OP_FOR_BIND_KV, OP_FOR_STEP, OP_CALL_RANGE
	CatArith            // OP_ADD, OP_SUB, OP_MUL, OP_DIV, OP_MOD, OP_CONCAT, OP_NEGATE
	CatCompare          // OP_EQ, OP_NEQ, OP_LT, OP_LTE, OP_GT, OP_GTE
	CatControl          // OP_JUMP, OP_JUMP_FALSE, OP_AND, OP_OR, OP_NOT
	CatScope            // OP_STORE_VAR
	CatStack            // OP_PUSH_CONST, OP_PUSH_NIL, OP_HALT
	CatOther            // everything else
	CatCount
)

// CatNames maps category index to human-readable name.
var CatNames = [CatCount]string{
	"Var Lookup", "Output", "Filter", "Loop",
	"Arithmetic", "Compare", "Control", "Scope",
	"Stack", "Other",
}

// OpcodeStats holds cumulative profiling data across all VM executions.
type OpcodeStats struct {
	Count    [CatCount]int64
	Duration [CatCount]time.Duration
}

var (
	globalStats OpcodeStats
	statsMu     sync.Mutex
)

// ResetOpcodeStats clears all accumulated profiling data.
func ResetOpcodeStats() {
	statsMu.Lock()
	globalStats = OpcodeStats{}
	statsMu.Unlock()
}

// GetOpcodeStats returns a snapshot of accumulated profiling data.
func GetOpcodeStats() OpcodeStats {
	statsMu.Lock()
	s := globalStats
	statsMu.Unlock()
	return s
}

// String returns a formatted table of opcode stats.
func (s OpcodeStats) String() string {
	var total time.Duration
	var totalCount int64
	for i := 0; i < CatCount; i++ {
		total += s.Duration[i]
		totalCount += s.Count[i]
	}
	if total == 0 {
		return "  (no opcode data collected)\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  %-18s %12s %12s %8s\n", "Category", "Count", "Time", "%")
	fmt.Fprintf(&b, "  %-18s %12s %12s %8s\n", "──────────────────", "────────────", "────────────", "────────")
	for i := 0; i < CatCount; i++ {
		if s.Count[i] == 0 {
			continue
		}
		pct := float64(s.Duration[i]) / float64(total) * 100
		fmt.Fprintf(&b, "  %-18s %12d %12s %7.1f%%\n",
			CatNames[i], s.Count[i], s.Duration[i].Round(time.Microsecond), pct)
	}
	fmt.Fprintf(&b, "  %-18s %12d %12s\n", "TOTAL", totalCount, total.Round(time.Microsecond))
	return b.String()
}

func opcodeCategory(op compiler.Opcode) int {
	switch op {
	case compiler.OP_LOAD, compiler.OP_GET_ATTR, compiler.OP_GET_INDEX:
		return CatVarLookup
	case compiler.OP_OUTPUT, compiler.OP_OUTPUT_RAW:
		return CatOutput
	case compiler.OP_FILTER:
		return CatFilter
	case compiler.OP_FOR_INIT, compiler.OP_FOR_BIND_1, compiler.OP_FOR_BIND_KV, compiler.OP_FOR_STEP, compiler.OP_CALL_RANGE:
		return CatLoop
	case compiler.OP_ADD, compiler.OP_SUB, compiler.OP_MUL, compiler.OP_DIV, compiler.OP_MOD, compiler.OP_CONCAT, compiler.OP_NEGATE:
		return CatArith
	case compiler.OP_EQ, compiler.OP_NEQ, compiler.OP_LT, compiler.OP_LTE, compiler.OP_GT, compiler.OP_GTE:
		return CatCompare
	case compiler.OP_JUMP, compiler.OP_JUMP_FALSE, compiler.OP_AND, compiler.OP_OR, compiler.OP_NOT:
		return CatControl
	case compiler.OP_STORE_VAR:
		return CatScope
	case compiler.OP_PUSH_CONST, compiler.OP_PUSH_NIL, compiler.OP_HALT:
		return CatStack
	default:
		return CatOther
	}
}

type profileState struct {
	prevOp  compiler.Opcode
	started bool
	start   time.Time
}

func profileInit() profileState {
	return profileState{}
}

func profileRecord(ps *profileState, op compiler.Opcode) {
	now := time.Now()
	if ps.started {
		cat := opcodeCategory(ps.prevOp)
		elapsed := now.Sub(ps.start)
		statsMu.Lock()
		globalStats.Duration[cat] += elapsed
		globalStats.Count[cat]++
		statsMu.Unlock()
	}
	ps.prevOp = op
	ps.start = now
	ps.started = true
}

func profileFlush(ps *profileState) {
	if ps.started {
		cat := opcodeCategory(ps.prevOp)
		elapsed := time.Since(ps.start)
		statsMu.Lock()
		globalStats.Duration[cat] += elapsed
		globalStats.Count[cat]++
		statsMu.Unlock()
	}
}
