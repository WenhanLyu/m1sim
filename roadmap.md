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

### M1: Project Bootstrap ⬜ NOT STARTED
**Goal:** Set up Go module, project structure, CLAUDE.md, CI workflow. Copy and adapt m2sim codebase to m1sim (rename module, update M1-specific comments).
**Cycles:** 5

### M2: Functional Emulator ⬜ NOT STARTED
**Goal:** ARM64 instruction decode and functional emulation — ALU, load/store, branches, SIMD, syscalls, ELF loading. Must correctly execute ARM64 user-space programs.
**Cycles:** 6

### M3: Timing Model ⬜ NOT STARTED
**Goal:** Cycle-accurate pipeline simulation — fetch/decode/execute/memory/writeback stages, L1I/L1D/L2 cache hierarchy, branch predictor, superscalar issue. Parameterized for M1.
**Cycles:** 6

### M4: PolyBench Integration ⬜ NOT STARTED
**Goal:** Compile PolyBench/C benchmarks for ARM64 (cross-compiled or native), integrate into test framework, run them through simulator, collect M1 hardware baselines.
**Cycles:** 5

### M5: Calibration & Validation ⬜ NOT STARTED
**Goal:** Tune M1 microarch parameters (cache latencies, pipeline widths, branch penalty) to achieve <20% average error and <50% max error across all PolyBench benchmarks.
**Cycles:** 8

## Lessons Learned
*(Updated each cycle)*

## Cycle History
| Cycle | Phase | Action |
|-------|-------|--------|
| 1 | Planning | Initial roadmap created, m2sim reference studied |
