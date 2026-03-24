package benchmarks

import (
	"testing"
	
	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
)

func TestGEMMOctupleDebug(t *testing.T) {
	prog, err := loader.Load(polybenchELFPath("gemm"))
	if err != nil {
		t.Fatalf("failed to load ELF: %v", err)
	}
	
	regFile := &emu.RegFile{}
	memory := emu.NewMemory()
	
	for _, seg := range prog.Segments {
		for i, b := range seg.Data {
			memory.Write8(seg.VirtAddr+uint64(i), b)
		}
		for i := uint64(len(seg.Data)); i < seg.MemSize; i++ {
			memory.Write8(seg.VirtAddr+i, 0)
		}
	}
	
	regFile.SP = prog.InitialSP
	
	opts := []pipeline.PipelineOption{
		pipeline.WithOctupleIssue(),
	}
	
	pipe := pipeline.NewPipeline(regFile, memory, opts...)
	pipe.SetPC(prog.EntryPoint)
	
	const maxCycles = 200000
	still_running := pipe.RunCycles(maxCycles)
	stats := pipe.Stats()
	
	t.Logf("8-wide GEMM (limit=%d): cycles=%d, insts=%d, still_running=%v", 
		maxCycles, stats.Cycles, stats.Instructions, still_running)
}
