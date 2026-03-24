#!/bin/bash
# Build script for PolyBench M1Sim bare-metal benchmarks
#
# Usage: ./build.sh [benchmark]
#   benchmark: gemm (default: all)

set -e

# Cross-compiler: use Apple clang with aarch64-unknown-none-elf target + lld linker
CC="clang --target=aarch64-unknown-none-elf -fuse-ld=lld"
OBJDUMP=llvm-objdump

# Script directory
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Compiler flags
# -fno-tree-vectorize: Disable auto-vectorization (M1Sim doesn't support NEON yet)
CFLAGS="-O2 -ffreestanding -nostdlib"
CFLAGS+=" -march=armv8-a+nofp+nosimd"  # Disable SIMD/FP registers for simulator compatibility
CFLAGS+=" -fno-tree-vectorize"
CFLAGS+=" -I$SCRIPT_DIR/common"
CFLAGS+=" -DPOLYBENCH_USE_RESTRICT"
CFLAGS+=" -DMINI_DATASET"

# Available benchmarks
BENCHMARKS="gemm atax 2mm mvt jacobi-1d 3mm bicg"

# Build function
build_benchmark() {
    local name=$1
    local src_dir="$SCRIPT_DIR/$name"
    
    if [ ! -d "$src_dir" ]; then
        echo "Error: Benchmark directory $src_dir not found"
        return 1
    fi
    
    echo "Building $name for M1Sim..."
    
    # Compile benchmark source
    $CC $CFLAGS -c "$src_dir/$name.c" -o "$SCRIPT_DIR/$name.o"
    
    # Compile startup code
    $CC $CFLAGS -c "$SCRIPT_DIR/common/startup.S" -o "$SCRIPT_DIR/startup.o"
    
    # Link
    $CC $CFLAGS -T "$SCRIPT_DIR/linker.ld" \
        "$SCRIPT_DIR/startup.o" \
        "$SCRIPT_DIR/$name.o" \
        -o "$SCRIPT_DIR/${name}_m2sim.elf"
    
    # Generate disassembly (if llvm-objdump available)
    if command -v llvm-objdump &>/dev/null; then
        llvm-objdump -d "$SCRIPT_DIR/${name}_m2sim.elf" > "$SCRIPT_DIR/${name}_m2sim.dis"
    fi
    
    echo "Build complete: ${name}_m2sim.elf"
    ls -la "$SCRIPT_DIR/${name}_m2sim.elf"
}

# Clean function
clean() {
    echo "Cleaning build artifacts..."
    rm -f "$SCRIPT_DIR"/*.o
    rm -f "$SCRIPT_DIR"/*_m2sim.elf
    rm -f "$SCRIPT_DIR"/*_m2sim.dis
}

# Main
case "${1:-all}" in
    clean)
        clean
        ;;
    all)
        for bench in $BENCHMARKS; do
            build_benchmark "$bench"
        done
        ;;
    *)
        build_benchmark "$1"
        ;;
esac

echo "Done."
