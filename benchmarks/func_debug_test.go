package benchmarks

import (
	"testing"

	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
)

// TestCallNoBL tests STP/LDP/RET without BL
// Manually sets X30 = 0x1004 (the "return address")
// Program at 0x1010: STP, NOP*n, LDP, RET
// "return address" at 0x1004: SVC #0
func TestCallNoBL(t *testing.T) {
	for _, nops := range []int{0, 1, 2, 3, 5, 10} {
		for _, label := range []string{"single", "quad"} {
			regFile := &emu.RegFile{}
			regFile.WriteReg(8, 93)
			regFile.WriteReg(0, 42)
			regFile.WriteReg(30, 0x1004) // Manually set X30 = return address
			regFile.SP = 0x10000
			memory := emu.NewMemory()
			
			// At 0x1004: SVC #0
			memory.Write32(0x1004, 0xD4000001)
			
			// callee at 0x1010:
			memory.Write32(0x1010, 0xA9BE7BFD) // STP X29, X30, [SP, #-0x10]!
			addr := uint64(0x1014)
			for i := 0; i < nops; i++ {
				memory.Write32(addr, 0xD503201F) // NOP
				addr += 4
			}
			memory.Write32(addr, 0xA8C27BFD) // LDP X29, X30, [SP], #0x10
			memory.Write32(addr+4, 0xD65F03C0) // RET
			
			var opts []pipeline.PipelineOption
			if label == "quad" {
				opts = []pipeline.PipelineOption{pipeline.WithQuadIssue()}
			}
			
			pipe := pipeline.NewPipeline(regFile, memory, opts...)
			pipe.SetPC(0x1010) // Start at callee directly
			pipe.RunCycles(5000)
			stats := pipe.Stats()
			ok := "OK"; if pipe.ExitCode() != 42 { ok = "FAIL" }
			t.Logf("%s nops=%d: exit=%d, cycles=%d, insts=%d (%s)", label, nops, pipe.ExitCode(), stats.Cycles, stats.Instructions, ok)
		}
	}
}
