package benchmarks

import (
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
)

func TestGEMMCycleTrace(t *testing.T) {
	elfPath := "polybench/gemm_m2sim.elf"
	prog, err := loader.Load(elfPath)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	
	memory := emu.NewMemory()
	regFile := &emu.RegFile{}
	
	for _, seg := range prog.Segments {
		for i, b := range seg.Data {
			memory.Write8(seg.VirtAddr+uint64(i), b)
		}
		for i := uint64(len(seg.Data)); i < seg.MemSize; i++ {
			memory.Write8(seg.VirtAddr+i, 0)
		}
	}
	
	regFile.SP = prog.InitialSP
	
	// Single issue 
	pipe := pipeline.NewPipeline(regFile, memory)
	pipe.SetPC(prog.EntryPoint)

	// Run 3000 cycles (enough to get through BSS clear and enter main's prologue)
	pipe.RunCycles(3000)
	stats := pipe.Stats()
	t.Logf("After 3K cycles: insts=%d, PC=0x%x", stats.Instructions, pipe.PC())
	
	// SP should be 0x91000 - 0xa0 = 0x90f60 after the sub sp instruction
	// STP saves X30 at SP+0x48 = 0x90fa8
	// Read actual SP from regFile
	sp := regFile.SP
	t.Logf("regFile.SP = 0x%x", sp)
	savedX29 := memory.Read64(sp + 0x40)
	savedX30 := memory.Read64(sp + 0x48)
	t.Logf("At SP=0x90f60: [sp+0x40]=0x%x (X29) [sp+0x48]=0x%x (X30, expected 0x80034)", savedX29, savedX30)
	
	// Run another 12000 cycles  
	pipe.RunCycles(12000)
	stats = pipe.Stats()
	t.Logf("After 15K cycles: insts=%d, PC=0x%x, halted=%v", stats.Instructions, pipe.PC(), pipe.Halted())
	
	// Run another 15000
	pipe.RunCycles(15000)
	stats = pipe.Stats()
	t.Logf("After 30K cycles: insts=%d, PC=0x%x, halted=%v", stats.Instructions, pipe.PC(), pipe.Halted())
}
