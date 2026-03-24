package benchmarks

import (
	"bytes"
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
)

func TestGEMMDebug(t *testing.T) {
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
	
	opts := []pipeline.PipelineOption{pipeline.WithOctupleIssue()}
	pipe := pipeline.NewPipeline(regFile, memory, opts...)
	pipe.SetPC(prog.EntryPoint)

	// Run max 5M cycles
	done := pipe.RunCycles(5000000)
	if !done {
		t.Logf("Still running after 5M cycles: insts=%d, halted=%v", 
			pipe.Stats().Instructions, pipe.Halted())
	} else {
		t.Logf("Halted! cycles=%d, insts=%d, exit=%d",
			pipe.Stats().Cycles, pipe.Stats().Instructions, pipe.ExitCode())
	}
}

func TestGEMMDebugSingle(t *testing.T) {
	_ = bytes.Buffer{}
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
	
	// No octuple issue
	pipe := pipeline.NewPipeline(regFile, memory)
	pipe.SetPC(prog.EntryPoint)

	// Run max 5M cycles
	done := pipe.RunCycles(5000000)
	if !done {
		t.Logf("Single: Still running after 5M cycles: insts=%d, halted=%v", 
			pipe.Stats().Instructions, pipe.Halted())
	} else {
		t.Logf("Single: Halted! cycles=%d, insts=%d, exit=%d",
			pipe.Stats().Cycles, pipe.Stats().Instructions, pipe.ExitCode())
	}
}
