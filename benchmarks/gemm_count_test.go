package benchmarks

import (
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
)

func TestGEMMCount(t *testing.T) {
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
	pSingle := pipeline.NewPipeline(regFile, memory)
	pSingle.SetPC(prog.EntryPoint)
	
	// Run 500K cycles max
	for i := 0; i < 500000; i++ {
		pSingle.Tick()
		if pSingle.Halted() {
			t.Logf("Single halted at cycle %d, insts=%d, exit=%d", i, pSingle.Stats().Instructions, pSingle.ExitCode())
			return
		}
	}
	
	stats := pSingle.Stats()
	t.Logf("Single at 500K cycles: insts=%d, NOT HALTED", stats.Instructions)
}
