# CLAUDE.md

## Build, Test, Lint

```bash
go build ./...            # Build
go test ./...             # Test
golangci-lint run ./...   # Lint
```

## Conventions
- Reuse Akita components
- Separate functional (emu/) and timing (timing/) logic
- Follow Go best practices

## Project Structure
- `emu/` — functional emulator (ARM64 instruction execution)
- `insts/` — instruction decoder
- `loader/` — ELF binary loader
- `timing/` — cycle-accurate timing model
- `benchmarks/` — benchmark harness and test files
- `driver/` — simulator driver/orchestration
- `cmd/` — command-line interface

## M1 vs M2 Differences
- L2 cache: 12MB (M1) vs 24MB (M2)
- Branch mispredict penalty: 14 cycles (M1 Firestorm) vs 12 cycles (M2 Avalanche)
