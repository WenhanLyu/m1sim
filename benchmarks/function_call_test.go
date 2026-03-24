package benchmarks

import (
	"testing"

	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
)

// TestFunctionCallQuad tests that function call/return works correctly in quad-issue
// Tests the exact pattern: STP X29,X30 / ... / LDP X29,X30 / RET
func TestFunctionCallQuad(t *testing.T) {
	regFile := &emu.RegFile{}
	regFile.WriteReg(8, 93) // syscall exit
	regFile.WriteReg(0, 42) // exit code
	regFile.SP = 0x10000
	memory := emu.NewMemory()

	// Main program at 0x1000:
	// BL callee (at 0x1020)
	// SVC #0 (exit)
	//
	// Callee at 0x1020:
	// STP X29, X30, [SP, #-32]! 
	// ADD X1, X1, #1  (some work)
	// ADD X2, X2, #1
	// ADD X3, X3, #1
	// LDP X29, X30, [SP], #32
	// RET

	// Main: BL to 0x1020 (offset = 0x1020 - 0x1000 = 0x20 = 8 instrs)
	// BL encoding: 0x94 << 24 | (offset/4)
	blOffset := int32(0x1020 - 0x1000)
	blImm26 := uint32(blOffset/4) & 0x3FFFFFF
	blInst := uint32(0b100101<<26) | blImm26
	memory.Write32(0x1000, blInst)
	// SVC #0 at 0x1004
	memory.Write32(0x1004, 0xD4000001) // SVC #0

	// Callee at 0x1020:
	// STP X29, X30, [SP, #-32]! (pre-index)
	// Encoding: SF=1, V=0, opc=01, L=0, imm7=-32/8=-4, Rt2=30, Rn=31, Rt=29
	// 1010 1001 1011 1110 0111 1100 0001 1101
	// 10 1001001 opc=01 L=0 imm7=(-32/8) Rt2 Rn Rt
	// A9 BF 7B FD -> STP x29, x30, [sp, #-0x20]!
	memory.Write32(0x1020, 0xA9BF7BFD) // STP X29, X30, [SP, #-0x20]!
	// ADD X1, X1, #1
	memory.Write32(0x1024, 0x91000421) // ADD X1, X1, #1
	// ADD X2, X2, #1
	memory.Write32(0x1028, 0x91000442) // ADD X2, X2, #1
	// ADD X3, X3, #1
	memory.Write32(0x102C, 0x91000463) // ADD X3, X3, #1
	// LDP X29, X30, [SP], #0x20 (post-index)
	memory.Write32(0x1030, 0xA8C17BFD) // LDP X29, X30, [SP], #0x20
	// RET
	memory.Write32(0x1034, 0xD65F03C0) // RET

	opts := []pipeline.PipelineOption{
		pipeline.WithQuadIssue(),
	}
	pipe := pipeline.NewPipeline(regFile, memory, opts...)
	pipe.SetPC(0x1000)

	// Run with limit to detect infinite loop
	still_running := pipe.RunCycles(10000)
	if still_running {
		t.Fatalf("pipeline is stuck in infinite loop after 10000 cycles")
	}
	
	stats := pipe.Stats()
	t.Logf("function call (quad): cycles=%d, insts=%d, exit=%d", stats.Cycles, stats.Instructions, pipe.ExitCode())
	
	if pipe.ExitCode() != 42 {
		t.Errorf("expected exit 42, got %d", pipe.ExitCode())
	}
}
