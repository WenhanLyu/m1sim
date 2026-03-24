package benchmarks

import (
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
)

func TestGEMMTrace(t *testing.T) {
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
	
	pipe := pipeline.NewPipeline(regFile, memory)
	pipe.SetPC(prog.EntryPoint)

	// Run 200K cycles
	done := pipe.RunCycles(200000)
	stats := pipe.Stats()
	if !done {
		t.Logf("Still running at 200K cycles: insts=%d", stats.Instructions)
	} else {
		t.Logf("Halted at 200K cycles: insts=%d, exit=%d", stats.Instructions, pipe.ExitCode())
	}
}
