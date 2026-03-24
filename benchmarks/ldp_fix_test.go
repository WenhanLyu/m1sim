package benchmarks

import (
	"testing"
	"github.com/WenhanLyu/m1sim/timing/pipeline"
	"github.com/WenhanLyu/m1sim/emu"
)

func TestLDPFix(t *testing.T) {
	memory := emu.NewMemory()
	regFile := &emu.RegFile{}
	
	// Program:
	// 1. stp x29, x30, [sp, #0x40]  → save x29, x30 on stack (at sp+0x40 and sp+0x48)
	// 2. [modify x30 somehow - no, let's just test directly]
	// 
	// Actually, put known values in memory and use ldp to load them
	
	// Memory layout:
	// 0x2000: 0xDEAD (x29 value = first register Rd)
	// 0x2008: 0xBEEF (x30 value = second register Rt2)
	
	memory.Write64(0x2000, 0xDEAD)
	memory.Write64(0x2008, 0xBEEF)
	
	regFile.SP = 0x2000
	
	// Program: ldp x29, x30, [sp] then svc
	// ldp x29, x30, [sp] = 0xA9407BFD
	// Let's compute: Rt=x29=11101, Rt2=x30=11110, Rn=sp=11111, imm7=0, opc=10, L=1
	// 64-bit signed offset: 10 101 0 011 0000000 11110 11111 11101
	// = 1010 1001 0100 0000 0111 1011 1111 1101
	// = 0xA9407BFD
	
	// movz x8, #93
	// d2800ba8
	
	// movz x0, #0
	// d2800000
	
	// svc #0
	// d4000001
	
	program := []uint32{
		0xA9407BFD,  // ldp x29, x30, [sp]  (imm7=0 means offset=0)
		0xd2800ba8,  // movz x8, #93
		0xd2800000,  // movz x0, #0
		0xd4000001,  // svc #0
	}
	
	for i, w := range program {
		memory.Write8(0x1000+uint64(i*4)+0, uint8(w))
		memory.Write8(0x1000+uint64(i*4)+1, uint8(w>>8))
		memory.Write8(0x1000+uint64(i*4)+2, uint8(w>>16))
		memory.Write8(0x1000+uint64(i*4)+3, uint8(w>>24))
	}
	
	pipe := pipeline.NewPipeline(regFile, memory)
	pipe.SetPC(0x1000)
	
	for i := 0; i < 100; i++ {
		pipe.Tick()
		if pipe.Halted() {
			t.Logf("Halted at cycle %d", i)
			break
		}
	}
	
	t.Logf("x29=%x (want 0xDEAD)", regFile.ReadReg(29))
	t.Logf("x30=%x (want 0xBEEF)", regFile.ReadReg(30))
	
	if regFile.ReadReg(29) != 0xDEAD {
		t.Errorf("x29 wrong: got 0x%x, want 0xDEAD", regFile.ReadReg(29))
	}
	if regFile.ReadReg(30) != 0xBEEF {
		t.Errorf("x30 wrong: got 0x%x, want 0xBEEF - LDP Rt2 not written!", regFile.ReadReg(30))
	}
}
