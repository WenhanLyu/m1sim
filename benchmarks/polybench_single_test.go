package benchmarks

import (
	"bytes"
	"testing"
)

func TestPolybenchATAXSingle(t *testing.T) {
	config := DefaultConfig()
	config.EnableOctupleIssue = false
	config.EnableSextupleIssue = false
	config.EnableQuadIssue = false
	config.EnableDualIssue = false
	config.Output = &bytes.Buffer{}
	config.EnableICache = false
	config.EnableDCache = false
	
	harness := NewHarness(config)
	harness.AddBenchmark(BenchmarkFromELF("polybench_atax", "ATAX", polybenchELFPath("atax")))
	
	results := harness.RunAll()
	r := results[0]
	t.Logf("single-issue atax: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
}
