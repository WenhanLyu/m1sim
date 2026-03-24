package benchmarks

import (
	"fmt"
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/insts"
)

func TestGEMMPC(t *testing.T) {
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
	
	pSingle := pipeline.NewPipeline(regFile, memory)
	pSingle.SetPC(prog.EntryPoint)
	
	// Track PC changes at various points
	lastPCs := make(map[uint64]int)
	decoder := insts.NewDecoder()
	
	for i := 0; i < 2000000; i++ {
		if pSingle.Halted() {
			t.Logf("Halted at cycle %d, insts=%d, exit=%d", i, pSingle.Stats().Instructions, pSingle.ExitCode())
			return
		}
		
		// Check memwb state
		pc := pSingle.PC()
		if i > 1990000 && i < 2000000 {
			// Log last 10K PCs
			lastPCs[pc]++
		}
		pSingle.Tick()
	}
	
	// Print PC histogram for last 10K
	_ = decoder
	t.Logf("After 2M cycles, insts=%d, PC histogram (last 10K cycles):", pSingle.Stats().Instructions)
	topPCs := make([][2]uint64, 0)
	for pc, count := range lastPCs {
		topPCs = append(topPCs, [2]uint64{pc, uint64(count)})
	}
	// Sort by count
	for i := 0; i < len(topPCs)-1; i++ {
		for j := i+1; j < len(topPCs); j++ {
			if topPCs[j][1] > topPCs[i][1] {
				topPCs[i], topPCs[j] = topPCs[j], topPCs[i]
			}
		}
	}
	
	for _, entry := range topPCs[:min(10, len(topPCs))] {
		word := memory.Read32(entry[0])
		inst := decoder.Decode(word)
		t.Logf("  PC=0x%x: count=%d, inst=%s", entry[0], entry[1], fmt.Sprintf("%v", inst.Op))
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}
