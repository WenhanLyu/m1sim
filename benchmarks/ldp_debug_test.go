package benchmarks

import (
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/loader"
	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/insts"
)

func TestLDPDebug(t *testing.T) {
	// Create a simple program that tests LDP
	// ldp x0, x1, [sp]
	// Where sp[0] = 0xDEAD and sp[8] = 0xBEEF

	memory := emu.NewMemory()
	regFile := &emu.RegFile{}
	
	// Put values on "stack"
	memory.Write64(0x10000, 0xDEAD)
	memory.Write64(0x10008, 0xBEEF)
	
	regFile.SP = 0x10000
	
	// Encode LDP x0, x1, [sp] (signed offset = 0)
	// 64-bit LDP: opc=10, L=1, imm7=0000000, Rt2=01, Rn=11111 (sp), Rt=00000
	// = 10 101 0 01 1 0000000 00001 11111 00000
	// binary: 1010100110000000 0000111111 00000
	// opc=10, V=0, L=1 (load), imm7=0000000, Rt2=00001, Rn=11111, Rt=00000
	// Encoding: opc(2) V(1) 10100 L(1) imm7(7) Rt2(5) Rn(5) Rt(5)
	// = 10 0 10100 1 0000000 00001 11111 00000
	// = 1001 0100 1000 0000 0000 1111 1100 0000
	// = 0x94800FC0 -- Let me look up the exact encoding
	
	// Actually, use the AArch64 encoding directly
	// ldp x0, x1, [sp]
	// bits[31:30] = opc = 10 (64-bit)
	// bits[29:27] = 101 (load/store pair, post-index)
	// bit[26] = 0 (not FP/SIMD)
	// bits[25:23] = 010 (signed offset)
	// bit[22] = 1 (load)
	// bits[21:15] = imm7 = 0000000
	// bits[14:10] = Rt2 = 00001 (x1)
	// bits[9:5] = Rn = 11111 (sp)
	// bits[4:0] = Rt = 00000 (x0)
	// = 10 101 0 010 1 0000000 00001 11111 00000
	// = 1010 1001 0100 0000 0000 0111 1110 0000
	// = 0xA9400FE0 -- no wait
	
	// Let me compute more carefully
	// opcode: bits[31:30] = 10 (64-bit)
	// bit[29] = 1 (indexed type bit1)
	// bit[28:27] = 01 (signed offset)
	// bit[26] = 0 (integer)
	// bit[25] = 1 (load/store pair)
	// bit[24] = 0 (post/pre/signed offset bit2)
	// bit[23] = 1 (load, L=1)
	// bit[22:16] = imm7 = 0 
	// bits[14:10] = Rt2 = 1
	// bits[9:5] = Rn = 31 (sp)
	// bits[4:0] = Rt = 0
	// Hmm let me just use a known encoding:
	// ldp x0, x1, [sp] = 10 101 0 01 1 0000000 00001 11111 00000
	//                    [31:30][29][28:27][26][25][24][23][22:16][14:10][9:5][4:0]
	// Wait, standard AArch64 encoding:
	// opc:2 | 10 | 1 | 0 | 1 | 0 | L | imm7:7 | Rt2:5 | Rn:5 | Rt:5
	// but bits[31:28] for LDP(x): 1010 1001 
	// known: ldp x29, x30, [sp, #0x40] = a9447bfd
	//   opc=10 (x64), bit[26]=0 (int), L=1
	//   imm7 = 0x40/8 = 8 = 0001000
	//   Rt2 = x30 = 11110 = 30
	//   Rn = sp = 11111 = 31
	//   Rt = x29 = 11101 = 29
	
	// For ldp x0, x1, [sp] = imm7=0, Rt2=1, Rn=31, Rt=0
	// 10 101 0 011 0000000 00001 11111 00000
	// = 1010 1001 0100 0000 0000 0111 1110 0000
	// = 0xA9400FE0 -- no, let me do this bit by bit
	
	// A9407FE0 -- from 'ldp x0, x1, [sp]'
	// Let me just verify using the actual instruction
	// 0xA9400FE0 would be ldp with some encoding...
	
	// Use decoded test directly
	decoder := insts.NewDecoder()
	ldpInst := decoder.Decode(0xA9400FE0) // guess
	t.Logf("0xA9400FE0: Op=%v, Rd=%d, Rt2=%d, Rn=%d, Imm=%d", ldpInst.Op, ldpInst.Rd, ldpInst.Rt2, ldpInst.Rn, ldpInst.Imm)
	
	// Try other encodings
	ldpInst2 := decoder.Decode(0xA9407BE0) // ldp x0, x30, [sp]
	t.Logf("0xA9407BE0: Op=%v, Rd=%d, Rt2=%d", ldpInst2.Op, ldpInst2.Rd, ldpInst2.Rt2)
	
	// Test with single-issue pipeline
	// Use the known-good instruction: a9447bfd = ldp x29, x30, [sp, #0x40]
	ldpKnown := decoder.Decode(0xa9447bfd)
	t.Logf("0xa9447bfd: Op=%v, Rd=%d, Rt2=%d, Rn=%d, SignedImm=%d, Is64Bit=%v", 
		ldpKnown.Op, ldpKnown.Rd, ldpKnown.Rt2, ldpKnown.Rn, ldpKnown.SignedImm, ldpKnown.Is64Bit)
	
	// Write test program: just ldp x0, x1, [sp] then svc
	// We'll use a9400fe0 (ldp x0, x1, [sp] with Rn=31=sp)
	// Actually write proper test using known instruction
	
	// Program at 0x1000: ldp x0, x1, [sp] ; movz x8, 93 ; movz x0, 0 ; svc #0
	// For now, just decode and check
	
	_ = regFile
	_ = memory
	_ = pipeline.Pipeline{}
	_ = loader.Program{}
}
