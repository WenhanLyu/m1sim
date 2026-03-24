package benchmarks

import (
	"bytes"
	"testing"
)

func TestGEMMTestELF(t *testing.T) {
	config := DefaultConfig()
	config.Output = &bytes.Buffer{}
	config.EnableICache = false
	config.EnableDCache = false
	
	harness := NewHarness(config)
	harness.AddBenchmark(BenchmarkFromELF("gemm_test", "gemm_test", "polybench/gemm_test.elf"))
	results := harness.RunAll()
	r := results[0]
	t.Logf("gemm_test.elf: exit=%d, cycles=%d, insts=%d, CPI=%.3f", r.ExitCode, r.SimulatedCycles, r.InstructionsRetired, r.CPI)
}
