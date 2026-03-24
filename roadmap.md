# M1Sim Project Roadmap

## Goal

Cycle-accurate Apple M1 CPU simulator that can simulate the PolyBench/C benchmark suite with <20% average time estimation error and <50% maximum error per benchmark.

Error formula: `error = abs(sim_cycles - hw_cycles) / min(sim_cycles, hw_cycles)`

Hardware baseline: Apple M1 Max (available locally for measurement).

---

## Milestones

### M1: Project Bootstrap ✅ DONE (Cycle 1-2)
**Goal:** Set up Go module, project structure, CLAUDE.md. Adapt m2sim codebase to m1sim (rename module, update M1-specific constants: L2=12MB, branch penalty=14 cycles).  
**Result:** `go build ./...` passes. Module correctly named. M1-specific values in place.

### M2: Fix Pipeline STP Bug + Run PolyBench End-to-End 🚧 IN PROGRESS
**Goal:** Fix the critical STP (store pair) pipeline bug that causes programs using function calls to loop infinitely. Then collect M1 hardware baselines and run all 7 PolyBench benchmarks end-to-end.

**Root Cause Found:** The pipeline's `EXMEMRegister` only has a single `StoreValue` field. For STP (store pair), only the first register (Rt) is stored to memory; the second register (Rt2) is silently dropped. This corrupts function prologues (which use `STP X29, X30, [SP, #offset]` to save callee-saved registers), causing functions to return to wrong addresses and loop infinitely.

**Acceptance Criteria:**
1. `go test ./benchmarks/ -run TestPolybenchGEMM -v -timeout 60s` passes
2. `go test ./benchmarks/ -run TestPolybench2MM -v -timeout 60s` passes
3. `go test ./benchmarks/ -run TestPolybench3MM -v -timeout 60s` passes
4. All 7 PolyBench benchmarks complete with exit code 0 (not timeout)
5. File `benchmarks/native/m1_baseline.json` exists with M1 hardware timing data
6. `go test ./benchmarks/ -run TestPolybenchAccuracy -v` (or similar) shows accuracy report

**Cycles Budget:** 6

### M3: Calibration & Validation ⬜ NOT STARTED
**Goal:** Tune M1 microarch parameters (superscalar width, issue ports, cache latencies, branch predictor) to achieve <20% average error and <50% maximum error across all 7 PolyBench benchmarks.

**Approach:**
- Systematically vary key timing parameters
- Use GEMM, ATAX, 2MM as calibration set, remaining 4 as validation set
- Focus on: issue width, load latency, branch penalty, cache miss penalty

**Cycles Budget:** 8

---

## Technical Notes

### Critical Bug: STP Second Register Not Stored
File: `timing/pipeline/pipeline.go`, `timing/pipeline/stages.go`

In `MemoryStage.Access()`, STP only writes the first register (`StoreValue`). Need to add `StoreValue2` field and write second register for STP.

The fix requires:
1. Add `StoreValue2 uint64` to `EXMEMRegister` (and secondary/tertiary variants)  
2. In each tick function, when `MemWrite && inst.Op == OpSTP`: also read `inst.Rt2` and store as `storeValue2`
3. In `MemoryStage.Access()`: for STP, also write `StoreValue2` at `addr + 4/8`

### Also Fix: SVC in Secondary Pipeline Slots
The 8-wide superscalar pipeline handles SVC in the primary slot but not in secondary slots 2-8. This was identified previously and there are uncommitted fixes on the `ares/fix-pipeline-and-tests` branch.

### M1 Microarch Parameters (M1 Firestorm core)
- Decode/issue width: 8 instructions/cycle
- Instruction queue: ~10 entries per port
- ALU latency: 1 cycle
- Load-to-use latency: 4 cycles (L1D hit)
- L1D cache: 128KB, 8-way
- L2 cache: 12MB
- Branch misprediction penalty: ~14 cycles

---

## Lessons Learned

- m2sim was more complete than expected; nearly all functional emulation was ready
- The functional emulator (`emu.Emulator`) works correctly for all PolyBench benchmarks
- The pipeline timing model has a critical STP bug affecting any program with function calls
- ATAX works in the pipeline (no function calls in inner loops?) but GEMM does not
- PolyBench MINI_DATASET: GEMM has ~11,824 instructions; ATAX ~2,193 instructions
- Tests in `benchmarks/` use M2 baseline data; need to update for M1 hardware

---

## Cycle History

| Cycle | Phase | Action |
|-------|-------|--------|
| 1 | Planning | Initial roadmap; m2sim reference studied |
| 2 | Implementation | Bootstrap: module renamed, go build passes |
| 3 | Verification | Verification cycle |
| 4 | Planning | Deep repo analysis: found critical pipeline STP bug |
| 5 | Planning | Athena deep-dive: confirmed STP bug, planned M2 milestone |
