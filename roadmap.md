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

### M3: Fix Hardware Baselines + Calibrate Pipeline Parameters ✅ DONE (Cycle 8-13)
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

### M4: MADD Accumulator Chain Optimization ⬜ IN PROGRESS
**Goal:** Reduce average error from 28.9% to <20% by fixing the MADD-accumulator pipeline issue.

**Root Cause Discovered (Athena, Cycle 14):**

Deep pipeline analysis revealed:
- M3 result: avg error 28.9% (max 43.2% for 2mm, already <50%)
- 2mm/3mm have 43%/39% error because of a specific instruction pattern in their compiled inner loops
- 2mm's inner loop is FULLY UNROLLED: 16 MADD instructions all accumulate into the SAME register (w23)
- Current pipeline: WAW (Write-After-Write) hazard detection prevents consecutive MADD instructions from co-issuing when they write to the same register → only 2 instructions per cycle ([madd, ldr] groups)
- GEMM works well (3.5% error) because it uses 16 DIFFERENT accumulator registers (w0, w17, w16, etc.) → no WAW → multiple MADDs co-issue → 3 instructions/cycle
- Additionally, MADD Ra forwarding is MISSING: the accumulator (Ra) register of MADD reads from register file directly without EXMEM forwarding, potentially producing wrong results

**Fix Required:**
1. **MADD Ra forwarding**: In `timing/pipeline/pipeline.go` tickOctupleIssue execute section, for MADD/MSUB, forward Ra (accumulator register, stored in `inst.Rt2`) from EXMEM stages (same-cycle + 1-cycle-old) before passing to execute. Currently executes with `s.regFile.ReadReg(inst.Rt2)` which ignores EXMEM. 
   - Specifically: compute `raValue` from EXMEM forwarding (check p.exmem, p.exmem2, p.exmem3 for Ra=inst.Rt2), then update the MADD result after `ExecuteWithFlags` call.

2. **WAW bypass for MADD chains**: In `canIssueWith` (`timing/pipeline/superscalar.go`), relax the WAW check when:
   - `prev.Inst.Op == OpMADD || prev.Inst.Op == OpMSUB` (previous is MADD)
   - `new.Inst.Op == OpMADD || new.Inst.Op == OpMSUB` (new is also MADD)
   - `prev.Rd == new.Rd` (same destination = WAW)
   - `new.Inst.Rt2 == prev.Rd` (new MADD's Ra IS the previous MADD's result)
   - In this case: ALLOW co-issue (WAW will resolve correctly via Ra forwarding)
   - Still need RAW check for new MADD's Rn/Rm (multiplier operands)

**Expected Impact:**
- 2mm: 2 insts/cycle → 4 insts/cycle for main loop → sim cycles from 12422 to ~8000-9000 → error <15%
- 3mm: similar improvement → error <10%  
- jacobi-1d, atax, mvt, bicg: smaller impact (different loop structures)
- Average error: 28.9% → estimated <15%
- GEMM: NO CHANGE (already uses different accumulators, no WAW to bypass)

**Acceptance Criteria:**
1. `go test ./benchmarks/ -run TestPolybenchM1Accuracy -v -timeout 120s` shows:
   - Average error < 20%  
   - Max error < 50%
   - gemm error remains < 10% (regression check)
2. All 7 benchmarks exit with code 0 (no correctness regressions)
3. `go build ./...` passes
4. `go test ./timing/... -timeout 60s` passes

**Cycles Budget:** 4

**Files to change:**
- `timing/pipeline/superscalar.go`: `canIssueWith()` — relax WAW for consecutive MADD/MSUB with same accumulator
- `timing/pipeline/pipeline.go`: `tickOctupleIssue()` execute section — add MADD Ra forwarding from EXMEM stages
- `timing/pipeline/stages.go`: `ExecuteWithFlags()` for OpMADD/OpMSUB — use idex.RmValue as Ra (after pipeline sets it up via forwarding) OR add inline Ra override

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
- The `LoadLatency` config has NO EFFECT because `latencyTable` is never passed to the pipeline harness
- The `ExecStalls` counter is always 0 for all benchmarks (latency table not used)
- GEMM achieves high IPC (2.85) because compiler uses 16 DIFFERENT accumulator registers → no WAW
- 2mm/3mm/jacobi have single-accumulator MADD chains → WAW limits to 2 insts/cycle (vs GEMM's 3)
- The WAW check correctly prevents same-cycle co-issue but there's NO EXMEM forwarding for MADD's Ra (accumulator)
- MADD Ra is read directly from register file (`s.regFile.ReadReg(inst.Rt2)`) bypassing all forwarding paths
- Fixing MADD Ra forwarding + WAW bypass for consecutive MADD chains is the key to M4

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
| 8-13 | Implementation+Verification | M3 complete: baselines corrected, avg error 28.9% |
| 14 | Planning | Athena: deep pipeline analysis; found MADD WAW+Ra-forwarding issue; defined M4 |
