package benchmarks

import (
    "bytes"
    "os"
    "testing"
)

func TestArithDualIssue(t *testing.T) {
    config := DefaultConfig()
    config.EnableOctupleIssue = false
    config.EnableSextupleIssue = false
    config.EnableQuadIssue = false
    config.EnableDualIssue = true
    config.Output = &bytes.Buffer{}
    config.EnableICache = false
    config.EnableDCache = false
    
    harness := NewHarness(config)
    harness.AddBenchmark(arithmeticSequential())
    
    results := harness.RunAll()
    if len(results) != 1 {
        t.Fatal("no results")
    }
    r := results[0]
    t.Logf("dual-issue: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
}

func TestArithSingleIssue(t *testing.T) {
    config := DefaultConfig()
    config.EnableOctupleIssue = false
    config.EnableSextupleIssue = false
    config.EnableQuadIssue = false
    config.EnableDualIssue = false
    config.Output = &bytes.Buffer{}
    config.EnableICache = false
    config.EnableDCache = false
    
    harness := NewHarness(config)
    harness.AddBenchmark(arithmeticSequential())
    
    results := harness.RunAll()
    if len(results) != 1 {
        t.Fatal("no results")
    }
    r := results[0]
    t.Logf("single-issue: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
}

func TestArithQuadIssue(t *testing.T) {
    config := DefaultConfig()
    config.EnableOctupleIssue = false
    config.EnableSextupleIssue = false
    config.EnableQuadIssue = true
    config.EnableDualIssue = false
    config.Output = &bytes.Buffer{}
    config.EnableICache = false
    config.EnableDCache = false
    
    harness := NewHarness(config)
    harness.AddBenchmark(arithmeticSequential())
    
    results := harness.RunAll()
    r := results[0]
    t.Logf("quad-issue: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
}

func TestArithSextupleIssue(t *testing.T) {
    config := DefaultConfig()
    config.EnableOctupleIssue = false
    config.EnableSextupleIssue = true
    config.EnableQuadIssue = false
    config.EnableDualIssue = false
    config.Output = &bytes.Buffer{}
    config.EnableICache = false
    config.EnableDCache = false
    
    harness := NewHarness(config)
    harness.AddBenchmark(arithmeticSequential())
    
    results := harness.RunAll()
    r := results[0]
    t.Logf("sextuple-issue: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
}

func TestPolybenchATAXDual(t *testing.T) {
    if testing.Short() {
        t.Skip("short mode")
    }
    config := DefaultConfig()
    config.EnableOctupleIssue = false
    config.EnableDualIssue = true
    config.Output = &bytes.Buffer{}
    config.EnableICache = false
    config.EnableDCache = false
    
    elfPath := polybenchELFPath("atax")
    if _, err := os.Stat(elfPath); os.IsNotExist(err) {
        t.Skipf("ELF not found: %s", elfPath)
    }
    
    harness := NewHarness(config)
    harness.AddBenchmark(BenchmarkFromELF("polybench_atax", "ATAX", polybenchELFPath("atax")))
    
    results := harness.RunAll()
    r := results[0]
    t.Logf("dual-issue atax: cycles=%d, insts=%d, CPI=%.3f, exit=%d", r.SimulatedCycles, r.InstructionsRetired, r.CPI, r.ExitCode)
}
