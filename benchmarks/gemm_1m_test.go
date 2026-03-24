package benchmarks

import (
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
)

func TestGEMM1M(t *testing.T) {
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

	// Run 1M cycles
	done := pipe.RunCycles(1000000)
	stats := pipe.Stats()
	t.Logf("After 1M cycles: halted=%v, insts=%d", pipe.Halted(), stats.Instructions)
	if done {
		t.Logf("Not halted (done=true means still running)")
	} else {
		t.Logf("Pipeline halted! exit=%d", pipe.ExitCode())
	}
}
