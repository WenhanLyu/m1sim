package benchmarks

import (
	"testing"

	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
)

// TestBLSTPQuad tests BL followed immediately by STP X29,X30 in quad-issue
func TestBLSTPQuad(t *testing.T) {
	regFile := &emu.RegFile{}
	regFile.WriteReg(8, 93)
	regFile.WriteReg(0, 42)
	regFile.SP = 0x10000
	memory := emu.NewMemory()

	// 0x1000: BL 0x1010
	blOffset := int32(0x1010 - 0x1000)
	blImm26 := uint32(blOffset/4) & 0x3FFFFFF
	blInst := uint32(0b100101<<26) | blImm26
	memory.Write32(0x1000, blInst)
	memory.Write32(0x1004, 0xD4000001) // SVC #0

	// callee at 0x1010: just STP, LDP, RET (no NOPs)
	memory.Write32(0x1010, 0xA9BE7BFD) // STP X29, X30, [SP, #-0x10]!
	memory.Write32(0x1014, 0xA8C27BFD) // LDP X29, X30, [SP], #0x10
	memory.Write32(0x1018, 0xD65F03C0) // RET

	for _, width := range []string{"single", "dual", "quad", "octuple"} {
		var opts []pipeline.PipelineOption
		switch width {
		case "dual": opts = []pipeline.PipelineOption{pipeline.WithDualIssue()}
		case "quad": opts = []pipeline.PipelineOption{pipeline.WithQuadIssue()}
		case "octuple": opts = []pipeline.PipelineOption{pipeline.WithOctupleIssue()}
		}
		
		r := &emu.RegFile{}
		r.WriteReg(8, 93); r.WriteReg(0, 42); r.SP = 0x10000
		m := emu.NewMemory()
		m.Write32(0x1000, blInst); m.Write32(0x1004, 0xD4000001)
		m.Write32(0x1010, 0xA9BE7BFD); m.Write32(0x1014, 0xA8C27BFD); m.Write32(0x1018, 0xD65F03C0)
		
		pipe := pipeline.NewPipeline(r, m, opts...)
		pipe.SetPC(0x1000)
		pipe.RunCycles(1000)
		stats := pipe.Stats()
		ok := "OK"; if pipe.ExitCode() != 42 { ok = "FAIL" }
		t.Logf("%s: exit=%d, cycles=%d, insts=%d (%s)", width, pipe.ExitCode(), stats.Cycles, stats.Instructions, ok)
	}
}
