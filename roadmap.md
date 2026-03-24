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

### M2: Fix Pipeline STP Bug + Run PolyBench End-to-End ✅ DONE (Cycle 5-6)
**Goal:** Fix the critical STP (store pair) pipeline bug and run all 7 PolyBench benchmarks.

**Result:**
- STP bug fixed (Leo: add StoreValue2 to EXMEMRegister)
- LDP Rt2 forwarding fixed (Ares)
- All 7 benchmarks run and exit with code 0
- M1 hardware baselines collected in `benchmarks/native/m1_baseline.json`
- Accuracy report: 123.6% average error, 306.3% max (jacobi-1d)

**Current accuracy (Cycle 6):**
```
benchmark    sim_cycles  hw_cycles   error%  pass?
gemm              4603       4355     5.7%   PASS ✅
atax              1222        494   147.4%   FAIL ❌
2mm              12476       8154    53.0%   FAIL ❌
3mm              18696      12625    48.1%   PASS ✅
mvt               1308        546   139.6%   FAIL ❌
bicg              1805        681   165.1%   FAIL ❌
jacobi-1d         2438        600   306.3%   FAIL ❌
Average error: 123.6% (target: <20%)
```

### M3: Fix Hardware Baselines + Calibrate Pipeline Parameters ⬜ IN PROGRESS
**Goal:** Two-pronged approach to reduce error:

1. **Fix hardware baseline methodology**: The current baselines were measured with native macOS binaries that use SIMD/vector instructions for memory copy loops, while the simulator ELFs are compiled with `-march=armv8-a+nofp+nosimd` (scalar only). This creates an unfair comparison — hardware appears faster because it executes fewer (vectorized) instructions. Remeasure baselines using scalar-constrained native binaries.

2. **Tune pipeline timing parameters**: Reduce `LoadLatency` from 4 cycles to 2 cycles to better model M1 Firestorm's effective load-to-use latency (M1's out-of-order execution hides most load latency; 2-cycle effective load latency is a reasonable approximation for in-order modeling).

**Root cause of accuracy gap:**
- Native binaries use SIMD (ldp q0, q1 = 128-bit loads) for copy loops
- ELF uses scalar loads (32-bit, one at a time)
- This makes jacobi-1d native much faster than ELF for the array swap
- The OoO execution gap: M1 Firestorm is deeply out-of-order; our model is in-order
  - Hardware achieves ~2-8x better throughput for memory-bound loops
  - Effective fix: reduce LoadLatency (2 cycles models OoO effective latency)

**Acceptance Criteria:**
1. `benchmarks/native/m1_baseline.json` updated with scalar-correct hardware measurements
2. `LoadLatency` in `timing/latency/config.go` reduced to 2 cycles (from 4)
3. `go test ./benchmarks/ -run TestPolybenchM1Accuracy -v -timeout 120s` shows:
   - Average error < 60%
   - At least 4/7 benchmarks have error < 50%
   - GEMM remains < 15% error
4. All benchmarks still exit with code 0 (no regressions)

**Cycles Budget:** 5

### M4: OoO Execution / Advanced Calibration ⬜ NOT STARTED
**Goal:** If M3 doesn't achieve <20% avg/<50% max, implement non-blocking load execution to model M1's out-of-order memory handling. Alternatively, further calibrate parameters.

**Key insight**: The M1 Firestorm's 600+ ROB entries allow overlapping many loop iterations in flight. For memory-bound loops, this is the primary advantage. Non-blocking loads (allow slots 2-8 to execute while slot 1's load is in-flight) would capture ~2x of this benefit.

**Cycles Budget:** 8

---

## Technical Notes

### Hardware Baseline Methodology Issue
The original baselines (`benchmarks/native/m1_baseline.json`) were collected with native macOS binaries compiled with `-O2 -mcpu=apple-m2 -fno-vectorize -fno-slp-vectorize`. Despite `-fno-vectorize`, the compiler still generates NEON SIMD instructions for memory copy loops (ldp q0/q1 = 128-bit pair loads for the B→A array swap). This results in 28-50% fewer cycles than scalar equivalent for benchmarks with copy loops.

**Verified**: jacobi-1d with scalar volatile: 828 cycles; with SIMD: 600 cycles (28% faster).

### Effective Load Latency
M1 Firestorm's OoO execution effectively reduces load-to-use latency for loops:
- Physical load latency: 4 cycles
- Effective latency (with OoO prefetching): ~1-2 cycles for sequential access patterns
- Setting `LoadLatency=2` models this without implementing full OoO

### Simulator CPI vs Hardware
- GEMM: sim CPI=0.335, hw CPI=0.37 → close match (compute-bound, OoO helps less)
- jacobi-1d: sim CPI=0.450, hw CPI=0.114 → 4x gap (memory-bound, OoO dominates)
- The CPI gap correlates with memory-boundedness (load density in inner loop)

### Key Files
- `timing/latency/config.go`: `LoadLatency` (default 4 → tune to 2)
- `benchmarks/native/m1_baseline.json`: Hardware baselines (needs update)
- `benchmarks/polybench/build_native.sh`: Native build script (uses SMALL_DATASET + SIMD → needs fix)
- `benchmarks/accuracy_test.go`: Accuracy test (assertions currently disabled → enable after M3)

---

## Lessons Learned

- m2sim was more complete than expected; nearly all functional emulation was ready
- The functional emulator (`emu.Emulator`) works correctly for all PolyBench benchmarks
- The pipeline timing model had a critical STP bug affecting any program with function calls
- ATAX works in the pipeline (simple kernel) but jacobi-1d and MVT are memory-bound
- The hardware baseline methodology matters: SIMD vs scalar creates unfair comparison
- M1 Firestorm's deep OoO execution is the primary challenge for in-order modeling
- In-order simulation achieves CPI 0.33-0.46 for PolyBench; hardware achieves 0.11-0.37
- Fix baseline methodology FIRST before implementing complex OoO features

---

## Cycle History

| Cycle | Phase | Action |
|-------|-------|--------|
| 1 | Planning | Initial roadmap; m2sim reference studied |
| 2 | Implementation | Bootstrap: module renamed, go build passes |
| 3 | Verification | Verification cycle |
| 4 | Planning | Deep repo analysis: found critical pipeline STP bug |
| 5 | Planning | Athena deep-dive: confirmed STP bug, planned M2 milestone |
| 6 | Implementation | Leo+Ares: fixed STP, LDP Rt2; Maya: collected M1 baselines; all 7 polybench run |
| 7 | Planning | Athena: deep accuracy analysis; found SIMD baseline issue; defined M3 |
