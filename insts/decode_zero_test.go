package insts

import (
	"fmt"
	"testing"
)

func TestDecodeZero(t *testing.T) {
	d := NewDecoder()
	inst := d.Decode(0x00000000)
	fmt.Printf("Decode 0x00000000: Op=%d (%v), Rd=%d, Rn=%d, Rm=%d, Format=%v\n", 
		inst.Op, inst.Op, inst.Rd, inst.Rn, inst.Rm, inst.Format)
	
	// Also check a few nearby addresses
	inst2 := d.Decode(0xd2800000)
	fmt.Printf("Decode 0xd2800000 (movz x0, #0): Op=%d, Rd=%d, Imm=%d\n", inst2.Op, inst2.Rd, inst2.Imm)
	
	inst3 := d.Decode(0xd2800ba8)
	fmt.Printf("Decode 0xd2800ba8 (movz x8, #93): Op=%d, Rd=%d, Imm=%d\n", inst3.Op, inst3.Rd, inst3.Imm)
}
