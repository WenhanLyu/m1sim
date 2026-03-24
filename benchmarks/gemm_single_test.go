package benchmarks

import (
	"bytes"
	"testing"
)

func TestPolybenchGEMMSingle(t *testing.T) {
	config := DefaultConfig()
	config.EnableOctupleIssue = false
	config.EnableSextupleIssue = false
	config.EnableQuadIssue = false
	config.EnableDualIssue = false
	config.Output = &bytes.Buffer{}
	config.EnableICache = false
	config.EnableDCache = false
	
	harness := NewHarness(config)
	harness.AddBenchmark(BenchmarkFromELF("polybench_gemm", "GEMM", polybenchELFPath("gemm")))
	
	results := harness.RunAll()
	r := results[0]
	t.Logf("single-issue gemm: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
	if r.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", r.ExitCode)
	}
}
