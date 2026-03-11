# M1Sim Project Roadmap

## Goal
Build a cycle-accurate Apple M1 CPU simulator (M1Sim) on the Akita framework that simulates the PolyBench benchmark suite with <20% average timing error and <50% max error per benchmark.

## Reference
- **sarchlab/m2sim** — Apple M2 CPU simulator with 16.9% average error on 18 benchmarks (7 PolyBench + 11 microbenchmarks). Strategy: adapt m2sim for M1 microarchitecture.
- Hardware: Apple M1 Max machine available for local measurement.

## Success Criteria
- Simulate all PolyBench benchmarks (the standard polybench/C suite)
- Average error ≤ 20% (error = abs(sim-hw)/min(sim,hw))
- Max error per benchmark ≤ 50%

## M1 vs M2 Microarchitecture Differences
- M1 "Firestorm" vs M2 "Avalanche" P-cores — same ARM64 ISA
- L2 cache: M1=12MB (per cluster), M2=24MB
- Branch mispredict penalty: M1~14 cycles vs M2~12 cycles (estimated)
- L1I/L1D sizes: similar (192KB/128KB per P-core)
- Pipeline width: both 8-wide

## Milestones

### M1: Project Bootstrap ✅ DONE (Cycle 1-2)
**Goal:** Set up Go module, project structure, CLAUDE.md, CI workflow. Copy and adapt m2sim codebase to m1sim (rename module, update M1-specific comments).
**Result:** go build ./... passes. L2=12MB, branch penalty=14 cycles.

### M2: Fix Critical Pipeline Bug + Test Infrastructure ⬜ IN PROGRESS
**Goal:** Fix the 8-wide superscalar pipeline bug (SVC not handled in secondary slots causing infinite loop), fix M2-specific test values to M1 values, build PolyBench ELFs, collect M1 hardware baselines.
**Why combined:** These are all small fixes that unblock validation work.
**Cycles budget:** 6

**Acceptance criteria:**
- `go test ./timing/...` all pass (no failures)
- `go test ./benchmarks/ -run TestArithmeticSequential -timeout 5s` passes
- `go test ./benchmarks/ -run TestPolybench` doesn't fail (either passes or skips with ELF found)
- PolyBench ELF binaries exist in benchmarks/polybench/
- M1 hardware baselines exist in benchmarks/native/m1_baseline.json

### M3: PolyBench End-to-End Validation ⬜ NOT STARTED
**Goal:** Run all 7 PolyBench benchmarks through the simulator, compare against M1 hardware measurements. Measure initial accuracy.
**Cycles:** 4

### M4: Calibration & Validation ⬜ NOT STARTED
**Goal:** Tune M1 microarch parameters to achieve <20% average error and <50% max error across all PolyBench benchmarks.
**Cycles:** 8

## Lessons Learned
- m2sim was more complete than expected; nearly everything (emu, timing, benchmarks) was already copied
- Critical bug: 8-wide superscalar pipeline (tickOctupleIssue) doesn't handle SVC in secondary slots 2-8
- Apple clang can build aarch64-unknown-none-elf binaries using lld (installed via homebrew)
- Tests were copied verbatim from m2sim and still assert M2 values, need updating

## Cycle History
| Cycle | Phase | Action |
|-------|-------|--------|
| 1 | Planning | Initial roadmap created, m2sim reference studied |
| 2 | Implementation | Bootstrap: cloned m2sim, built project, go build passes |
| 3 | Verification | Verification cycle |
| 4 | Planning | Deep repo analysis: found critical pipeline bug, updated roadmap |
