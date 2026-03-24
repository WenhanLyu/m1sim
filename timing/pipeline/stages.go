package pipeline

import (
	"github.com/WenhanLyu/m1sim/emu"
	"github.com/WenhanLyu/m1sim/insts"
)

// FetchStage reads instructions from memory.
type FetchStage struct {
	memory *emu.Memory
}

// NewFetchStage creates a new fetch stage.
func NewFetchStage(memory *emu.Memory) *FetchStage {
	return &FetchStage{memory: memory}
}

// Fetch fetches an instruction word from memory at the given PC.
// Returns the instruction word and whether the fetch was successful.
func (s *FetchStage) Fetch(pc uint64) (uint32, bool) {
	word := s.memory.Read32(pc)
	return word, true
}

// DecodeStage decodes instructions and reads register values.
type DecodeStage struct {
	regFile *emu.RegFile
	decoder *insts.Decoder
	// Pool of pre-allocated instructions to avoid heap allocations during decode
	// Supports up to 8 concurrent decode operations (for 8-wide superscalar pipelines)
	instPool  [64]insts.Instruction // Must be >= pipeline_width * pipeline_stages to avoid pool slot reuse corruption
	poolIndex int
}

// NewDecodeStage creates a new decode stage.
func NewDecodeStage(regFile *emu.RegFile) *DecodeStage {
	return &DecodeStage{
		regFile: regFile,
		decoder: insts.NewDecoder(),
	}
}

// DecodeResult contains the output of the decode stage.
type DecodeResult struct {
	Inst      *insts.Instruction
	RnValue   uint64
	RmValue   uint64
	Rd        uint8
	Rn        uint8
	Rm        uint8
	MemRead   bool
	MemWrite  bool
	RegWrite  bool
	MemToReg  bool
	IsBranch  bool
	IsSyscall bool
}

// Decode decodes an instruction word and reads register values.
func (s *DecodeStage) Decode(word uint32, pc uint64) DecodeResult {
	// Get next available pre-allocated instruction from pool
	inst := &s.instPool[s.poolIndex]
	s.poolIndex = (s.poolIndex + 1) % len(s.instPool)

	// Use DecodeInto with pre-allocated instruction to eliminate heap allocation
	s.decoder.DecodeInto(word, inst)

	result := DecodeResult{
		Inst: inst,
		Rd:   inst.Rd,
		Rn:   inst.Rn,
		Rm:   inst.Rm,
	}

	// For BL/BLR, the destination is always X30 (link register)
	if inst.Op == insts.OpBL || inst.Op == insts.OpBLR {
		result.Rd = 30
	}

	// Read register values
	result.RnValue = s.regFile.ReadReg(inst.Rn)
	result.RmValue = s.regFile.ReadReg(inst.Rm)

	// Determine control signals based on instruction type
	result.RegWrite = s.isRegWriteInst(inst)
	result.MemRead = s.isLoadOp(inst.Op)
	result.MemWrite = s.isStoreOp(inst.Op)
	result.MemToReg = s.isLoadOp(inst.Op)
	result.IsBranch = s.isBranchInst(inst)
	result.IsSyscall = inst.Op == insts.OpSVC

	return result
}

// isLoadOp returns true if the opcode is a load operation.
func (s *DecodeStage) isLoadOp(op insts.Op) bool {
	switch op {
	case insts.OpLDR, insts.OpLDP, insts.OpLDRB, insts.OpLDRSB,
		insts.OpLDRH, insts.OpLDRSH, insts.OpLDRLit, insts.OpLDRQ:
		return true
	default:
		return false
	}
}

// isStoreOp returns true if the opcode is a store operation.
func (s *DecodeStage) isStoreOp(op insts.Op) bool {
	switch op {
	case insts.OpSTR, insts.OpSTP, insts.OpSTRB, insts.OpSTRH, insts.OpSTRQ:
		return true
	default:
		return false
	}
}

// isRegWriteInst determines if the instruction writes to a register.
func (s *DecodeStage) isRegWriteInst(inst *insts.Instruction) bool {
	// Don't write if destination is XZR (register 31)
	if inst.Rd == 31 && inst.Op != insts.OpBL && inst.Op != insts.OpBLR {
		return false
	}

	switch inst.Op {
	case insts.OpADD, insts.OpSUB, insts.OpAND, insts.OpORR, insts.OpEOR,
		insts.OpBIC, insts.OpORN, insts.OpEON:
		return true
	case insts.OpLDR, insts.OpLDP, insts.OpLDRB, insts.OpLDRSB,
		insts.OpLDRH, insts.OpLDRSH, insts.OpLDRLit, insts.OpLDRQ:
		return true
	case insts.OpBL, insts.OpBLR:
		return true // BL/BLR write to X30
	case insts.OpMOVZ, insts.OpMOVN, insts.OpMOVK:
		return true // Wide immediate moves write to destination register
	case insts.OpADR, insts.OpADRP:
		return true // PC-relative address computation writes to destination
	case insts.OpMADD, insts.OpMSUB:
		return true // Multiply-add/sub write to destination
	case insts.OpUBFM, insts.OpSBFM, insts.OpBFM:
		return true // Bitfield moves write to destination
	default:
		return false
	}
}

// isBranchInst determines if the instruction is a branch.
func (s *DecodeStage) isBranchInst(inst *insts.Instruction) bool {
	switch inst.Op {
	case insts.OpB, insts.OpBL, insts.OpBCond, insts.OpBR, insts.OpBLR, insts.OpRET:
		return true
	default:
		return false
	}
}

// ExecuteStage performs ALU operations.
type ExecuteStage struct {
	regFile *emu.RegFile
}

// NewExecuteStage creates a new execute stage.
func NewExecuteStage(regFile *emu.RegFile) *ExecuteStage {
	return &ExecuteStage{regFile: regFile}
}

// ExecuteResult contains the output of the execute stage.
type ExecuteResult struct {
	ALUResult    uint64
	StoreValue   uint64
	BranchTaken  bool
	BranchTarget uint64

	// Flag output for flag-setting instructions (CMP, SUBS, ADDS).
	// These are stored in EXMEM for forwarding to dependent B.cond instructions.
	SetsFlags bool
	FlagN     bool
	FlagZ     bool
	FlagC     bool
	FlagV     bool
}

// Execute performs the ALU operation for the instruction.
// rnValue and rmValue are the (possibly forwarded) operand values.
func (s *ExecuteStage) Execute(idex *IDEXRegister, rnValue, rmValue uint64) ExecuteResult {
	// Call ExecuteWithFlags with no forwarded flags (reads from PSTATE)
	return s.ExecuteWithFlags(idex, rnValue, rmValue, false, false, false, false, false)
}

// ExecuteWithFlags performs the ALU operation with optional flag forwarding.
// When forwardFlags is true, the provided n, z, c, v flags are used instead of reading PSTATE.
// This fixes the pipeline timing hazard where CMP sets PSTATE at cycle END but B.cond reads
// at cycle START, causing stale flag reads.
func (s *ExecuteStage) ExecuteWithFlags(idex *IDEXRegister, rnValue, rmValue uint64,
	forwardFlags bool, fwdN, fwdZ, fwdC, fwdV bool) ExecuteResult {
	result := ExecuteResult{}

	if !idex.Valid || idex.Inst == nil {
		return result
	}

	inst := idex.Inst

	// Apply shift to Rm for data-processing register instructions.
	// This mirrors the emulator's applyShift64/applyShift32 in executeDPReg.
	if inst.Format == insts.FormatDPReg && inst.ShiftAmount > 0 {
		if inst.Is64Bit {
			switch inst.ShiftType {
			case insts.ShiftLSL:
				rmValue = rmValue << inst.ShiftAmount
			case insts.ShiftLSR:
				rmValue = rmValue >> inst.ShiftAmount
			case insts.ShiftASR:
				rmValue = uint64(int64(rmValue) >> inst.ShiftAmount)
			case insts.ShiftROR:
				rmValue = (rmValue >> inst.ShiftAmount) | (rmValue << (64 - inst.ShiftAmount))
			}
		} else {
			rm32 := uint32(rmValue)
			switch inst.ShiftType {
			case insts.ShiftLSL:
				rm32 = rm32 << inst.ShiftAmount
			case insts.ShiftLSR:
				rm32 = rm32 >> inst.ShiftAmount
			case insts.ShiftASR:
				rm32 = uint32(int32(rm32) >> inst.ShiftAmount)
			case insts.ShiftROR:
				rm32 = (rm32 >> inst.ShiftAmount) | (rm32 << (32 - inst.ShiftAmount))
			}
			rmValue = uint64(rm32)
		}
	}

	switch inst.Op {
	case insts.OpADD:
		result.ALUResult = s.executeADD(inst, rnValue, rmValue)
		if inst.SetFlags {
			n, z, c, v := s.computeAddFlags(inst, rnValue, rmValue, result.ALUResult)
			s.regFile.PSTATE.N = n
			s.regFile.PSTATE.Z = z
			s.regFile.PSTATE.C = c
			s.regFile.PSTATE.V = v
			result.SetsFlags = true
			result.FlagN = n
			result.FlagZ = z
			result.FlagC = c
			result.FlagV = v
		}
	case insts.OpSUB:
		result.ALUResult = s.executeSUB(inst, rnValue, rmValue)
		if inst.SetFlags {
			n, z, c, v := s.computeSubFlags(inst, rnValue, rmValue, result.ALUResult)
			s.regFile.PSTATE.N = n
			s.regFile.PSTATE.Z = z
			s.regFile.PSTATE.C = c
			s.regFile.PSTATE.V = v
			result.SetsFlags = true
			result.FlagN = n
			result.FlagZ = z
			result.FlagC = c
			result.FlagV = v
		}
	case insts.OpAND:
		result.ALUResult = s.executeAND(inst, rnValue, rmValue)
	case insts.OpORR:
		result.ALUResult = s.executeORR(inst, rnValue, rmValue)
	case insts.OpEOR:
		result.ALUResult = s.executeEOR(inst, rnValue, rmValue)
	case insts.OpBIC:
		result.ALUResult = s.executeBIC(inst, rnValue, rmValue)
	case insts.OpORN:
		result.ALUResult = s.executeORN(inst, rnValue, rmValue)
	case insts.OpEON:
		result.ALUResult = s.executeEON(inst, rnValue, rmValue)
	case insts.OpLDR, insts.OpSTR, insts.OpLDP, insts.OpSTP,
		insts.OpLDRB, insts.OpSTRB, insts.OpLDRSB,
		insts.OpLDRH, insts.OpSTRH, insts.OpLDRSH:
		// Address calculation: base + offset
		// If base register is 31, use SP instead
		baseAddr := rnValue
		if inst.Rn == 31 {
			baseAddr = s.regFile.SP
		}
		// Handle indexed addressing modes
		switch inst.IndexMode {
		case insts.IndexPre:
			// Pre-index: address = base + signed offset
			result.ALUResult = uint64(int64(baseAddr) + inst.SignedImm)
		case insts.IndexPost:
			// Post-index: address = base (writeback happens later)
			result.ALUResult = baseAddr
		default:
			// Unsigned offset or signed offset for LDP/STP
			if inst.Format == insts.FormatLoadStorePair {
				result.ALUResult = uint64(int64(baseAddr) + inst.SignedImm)
			} else {
				result.ALUResult = baseAddr + inst.Imm
			}
		}
		result.StoreValue = rmValue // For STR, the value to store
	case insts.OpB:
		// Unconditional branch
		result.BranchTaken = true
		result.BranchTarget = uint64(int64(idex.PC) + inst.BranchOffset)
	case insts.OpBL:
		// Branch with link
		result.BranchTaken = true
		result.BranchTarget = uint64(int64(idex.PC) + inst.BranchOffset)
		result.ALUResult = idex.PC + 4 // Return address
	case insts.OpBCond:
		// Conditional branch
		var conditionMet bool
		if idex.IsFused {
			// Fused CMP+B.cond: compute flags from fused operands
			var op2 uint64
			if idex.FusedIsImm {
				op2 = idex.FusedImmVal
			} else {
				op2 = idex.FusedRmVal
			}
			n, z, c, v := ComputeSubFlags(idex.FusedRnVal, op2, idex.FusedIs64)
			conditionMet = EvaluateConditionWithFlags(inst.Cond, n, z, c, v)
		} else if forwardFlags {
			// Non-fused with flag forwarding: use forwarded flags from previous
			// flag-setting instruction (e.g., CMP in EXMEM stage).
			// This fixes the pipeline timing hazard where CMP sets PSTATE at cycle
			// END but B.cond reads at cycle START.
			conditionMet = EvaluateConditionWithFlags(inst.Cond, fwdN, fwdZ, fwdC, fwdV)
		} else {
			// Non-fused without forwarding: read condition from PSTATE
			conditionMet = s.checkCondition(inst.Cond)
		}
		if conditionMet {
			result.BranchTaken = true
			result.BranchTarget = uint64(int64(idex.PC) + inst.BranchOffset)
		} else {
			result.BranchTaken = false
		}
	case insts.OpBR:
		// Branch to register
		result.BranchTaken = true
		result.BranchTarget = rnValue
	case insts.OpBLR:
		// Branch with link to register
		result.BranchTaken = true
		result.BranchTarget = rnValue
		result.ALUResult = idex.PC + 4 // Return address
	case insts.OpRET:
		// Return (branch to Rn, typically X30)
		result.BranchTaken = true
		result.BranchTarget = rnValue
	case insts.OpMOVZ:
		// MOVZ: Rd = (Imm << Shift)
		result.ALUResult = inst.Imm << inst.Shift
	case insts.OpMOVN:
		// MOVN: Rd = ~(Imm << Shift)
		result.ALUResult = ^(inst.Imm << inst.Shift)
		if !inst.Is64Bit {
			result.ALUResult = uint64(uint32(result.ALUResult))
		}
	case insts.OpMOVK:
		// MOVK: Rd = (Rd & ~mask) | (Imm << Shift)
		// The value being kept is the current Rd value (rnValue used as current dest)
		mask := uint64(0xFFFF) << inst.Shift
		result.ALUResult = (rnValue &^ mask) | ((inst.Imm << inst.Shift) & mask)
	case insts.OpADR:
		// ADR: Rd = PC + offset
		result.ALUResult = uint64(int64(idex.PC) + inst.BranchOffset)
	case insts.OpADRP:
		// ADRP: Rd = (PC & ~0xFFF) + (offset << 12)
		// BranchOffset is already shifted by 12 in the decoder
		pageBase := idex.PC & ^uint64(0xFFF)
		result.ALUResult = uint64(int64(pageBase) + inst.BranchOffset)
	case insts.OpMADD:
		// MADD: Rd = Ra + (Rn * Rm)
		// Ra is stored in Rt2 field, read as RmValue for 3-source instructions
		raValue := idex.RmValue // Ra operand (3rd source, stored as Rm in pipeline)
		if inst.Rt2 != 31 {
			raValue = s.regFile.ReadReg(inst.Rt2)
		}
		if inst.Is64Bit {
			result.ALUResult = raValue + (rnValue * rmValue)
		} else {
			result.ALUResult = uint64(uint32(raValue) + uint32(rnValue)*uint32(rmValue))
		}
	case insts.OpMSUB:
		// MSUB: Rd = Ra - (Rn * Rm)
		raValue := idex.RmValue
		if inst.Rt2 != 31 {
			raValue = s.regFile.ReadReg(inst.Rt2)
		}
		if inst.Is64Bit {
			result.ALUResult = raValue - (rnValue * rmValue)
		} else {
			result.ALUResult = uint64(uint32(raValue) - uint32(rnValue)*uint32(rmValue))
		}
	case insts.OpUBFM:
		// UBFM: Unsigned bitfield move
		// ALUResult = ubfm(Rn, immr, imms)
		result.ALUResult = executeBitfield(inst, rnValue, false)
	case insts.OpSBFM:
		// SBFM: Signed bitfield move
		result.ALUResult = executeBitfield(inst, rnValue, true)
	case insts.OpBFM:
		// BFM: Bitfield move (with existing destination bits)
		result.ALUResult = executeBFM(inst, rnValue, rmValue)
	}

	return result
}

// executeBitfield implements UBFM and SBFM instruction semantics.
// These implement logical/arithmetic shifts and sign/zero extension operations.
func executeBitfield(inst *insts.Instruction, rnValue uint64, signed bool) uint64 {
	immr := inst.Imm         // rotate amount
	imms := inst.Imm2        // width-1 (or shift amount for aliases)
	bits := uint64(64)
	if !inst.Is64Bit {
		bits = 32
		rnValue = uint64(uint32(rnValue))
	}

	// Handle the common aliases efficiently
	if imms < immr {
		// Extract and optionally sign-extend a field
		width := imms + 1
		var field uint64
		if inst.Is64Bit {
			field = rnValue & ((1 << width) - 1)
		} else {
			field = rnValue & uint64((uint32(1) << width) - 1)
		}
		shift := bits - immr
		result := field << shift >> shift // zero-extend first
		if signed && (field>>(width-1))&1 == 1 {
			// Sign extend
			if inst.Is64Bit {
				mask := ^uint64(0) << width
				result = field | mask
			} else {
				mask := uint32(^uint32(0)) << width
				result = uint64(uint32(field) | mask)
			}
		}
		return result
	}

	// Rotate right by immr and mask
	if immr > 0 {
		if inst.Is64Bit {
			rnValue = (rnValue >> immr) | (rnValue << (bits - immr))
		} else {
			rn32 := uint32(rnValue)
			rnValue = uint64((rn32 >> immr) | (rn32 << (bits - immr)))
		}
	}

	// Mask to width = imms - immr + 1
	width := imms - immr + 1
	var mask uint64
	if width >= bits {
		mask = ^uint64(0)
	} else {
		mask = (1 << width) - 1
	}
	result := rnValue & mask

	if signed && result>>(width-1)&1 == 1 {
		if inst.Is64Bit {
			result |= ^uint64(0) << width
		} else {
			result = uint64(uint32(result) | (^uint32(0) << width))
		}
	}

	if !inst.Is64Bit {
		result = uint64(uint32(result))
	}
	return result
}

// executeBFM implements BFM (Bitfield Move with existing destination bits).
func executeBFM(inst *insts.Instruction, rnValue, rdValue uint64) uint64 {
	// For simplicity, use UBFM logic then merge with destination
	result := executeBitfield(inst, rnValue, false)
	immr := inst.Imm
	imms := inst.Imm2
	if imms >= immr {
		width := imms - immr + 1
		var mask uint64
		if width >= 64 {
			mask = ^uint64(0)
		} else {
			mask = (1 << width) - 1
		}
		return (rdValue &^ mask) | (result & mask)
	}
	return result
}

// checkCondition evaluates a branch condition based on PSTATE flags.
func (s *ExecuteStage) checkCondition(cond insts.Cond) bool {
	pstate := s.regFile.PSTATE

	switch cond {
	case insts.CondEQ:
		return pstate.Z
	case insts.CondNE:
		return !pstate.Z
	case insts.CondCS:
		return pstate.C
	case insts.CondCC:
		return !pstate.C
	case insts.CondMI:
		return pstate.N
	case insts.CondPL:
		return !pstate.N
	case insts.CondVS:
		return pstate.V
	case insts.CondVC:
		return !pstate.V
	case insts.CondHI:
		return pstate.C && !pstate.Z
	case insts.CondLS:
		return !pstate.C || pstate.Z
	case insts.CondGE:
		return pstate.N == pstate.V
	case insts.CondLT:
		return pstate.N != pstate.V
	case insts.CondGT:
		return !pstate.Z && (pstate.N == pstate.V)
	case insts.CondLE:
		return pstate.Z || (pstate.N != pstate.V)
	case insts.CondAL, insts.CondNV:
		return true
	default:
		return false
	}
}

func (s *ExecuteStage) executeADD(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Format == insts.FormatDPImm {
		imm := inst.Imm
		if inst.Shift > 0 {
			imm <<= inst.Shift
		}
		if inst.Is64Bit {
			return rnValue + imm
		}
		return uint64(uint32(rnValue) + uint32(imm))
	}
	// Register format
	if inst.Is64Bit {
		return rnValue + rmValue
	}
	return uint64(uint32(rnValue) + uint32(rmValue))
}

func (s *ExecuteStage) executeSUB(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Format == insts.FormatDPImm {
		imm := inst.Imm
		if inst.Shift > 0 {
			imm <<= inst.Shift
		}
		if inst.Is64Bit {
			return rnValue - imm
		}
		return uint64(uint32(rnValue) - uint32(imm))
	}
	// Register format
	if inst.Is64Bit {
		return rnValue - rmValue
	}
	return uint64(uint32(rnValue) - uint32(rmValue))
}

func (s *ExecuteStage) executeAND(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Is64Bit {
		return rnValue & rmValue
	}
	return uint64(uint32(rnValue) & uint32(rmValue))
}

func (s *ExecuteStage) executeORR(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Is64Bit {
		return rnValue | rmValue
	}
	return uint64(uint32(rnValue) | uint32(rmValue))
}

func (s *ExecuteStage) executeEOR(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Is64Bit {
		return rnValue ^ rmValue
	}
	return uint64(uint32(rnValue) ^ uint32(rmValue))
}

func (s *ExecuteStage) executeBIC(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Is64Bit {
		return rnValue & ^rmValue
	}
	return uint64(uint32(rnValue) & ^uint32(rmValue))
}

func (s *ExecuteStage) executeORN(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Is64Bit {
		return rnValue | ^rmValue
	}
	return uint64(uint32(rnValue) | ^uint32(rmValue))
}

func (s *ExecuteStage) executeEON(inst *insts.Instruction, rnValue, rmValue uint64) uint64 {
	if inst.Is64Bit {
		return rnValue ^ ^rmValue
	}
	return uint64(uint32(rnValue) ^ ^uint32(rmValue))
}

// computeAddFlags computes PSTATE flags for an ADD/ADDS operation without setting them.
// Returns n, z, c, v flag values.
func (s *ExecuteStage) computeAddFlags(inst *insts.Instruction, op1, op2, result uint64) (n, z, c, v bool) {
	if inst.Is64Bit {
		// 64-bit flags
		n = (result >> 63) == 1
		z = result == 0
		c = result < op1 // unsigned overflow (carry out)
		// V: signed overflow - adding same signs gives different sign
		op1Sign := op1 >> 63
		op2Sign := op2 >> 63
		resultSign := result >> 63
		v = (op1Sign == op2Sign) && (op1Sign != resultSign)
	} else {
		// 32-bit flags
		r32 := uint32(result)
		o1 := uint32(op1)
		o2 := uint32(op2)
		n = (r32 >> 31) == 1
		z = r32 == 0
		c = r32 < o1
		op1Sign := o1 >> 31
		op2Sign := o2 >> 31
		resultSign := r32 >> 31
		v = (op1Sign == op2Sign) && (op1Sign != resultSign)
	}
	return
}

// computeSubFlags computes PSTATE flags for a SUB/SUBS/CMP operation without setting them.
// Returns n, z, c, v flag values.
func (s *ExecuteStage) computeSubFlags(inst *insts.Instruction, op1, op2, result uint64) (n, z, c, v bool) {
	if inst.Is64Bit {
		// 64-bit flags
		n = (result >> 63) == 1
		z = result == 0
		c = op1 >= op2 // no borrow
		// V: signed overflow - subtracting different signs gives wrong sign
		op1Sign := op1 >> 63
		op2Sign := op2 >> 63
		resultSign := result >> 63
		v = (op1Sign != op2Sign) && (op2Sign == resultSign)
	} else {
		// 32-bit flags
		r32 := uint32(result)
		o1 := uint32(op1)
		o2 := uint32(op2)
		n = (r32 >> 31) == 1
		z = r32 == 0
		c = o1 >= o2
		op1Sign := o1 >> 31
		op2Sign := o2 >> 31
		resultSign := r32 >> 31
		v = (op1Sign != op2Sign) && (op2Sign == resultSign)
	}
	return
}

// MemoryStage handles memory reads and writes.
type MemoryStage struct {
	memory *emu.Memory
}

// NewMemoryStage creates a new memory stage.
func NewMemoryStage(memory *emu.Memory) *MemoryStage {
	return &MemoryStage{memory: memory}
}

// MemoryResult contains the output of the memory stage.
type MemoryResult struct {
	MemData  uint64
	MemData2 uint64 // Second loaded value for LDP (load pair) instructions
}

// Access performs memory read or write operations.
func (s *MemoryStage) Access(exmem *EXMEMRegister) MemoryResult {
	result := MemoryResult{}

	if !exmem.Valid {
		return result
	}

	addr := exmem.ALUResult

	if exmem.MemRead {
		// Load: read from memory
		if exmem.Inst != nil && exmem.Inst.Is64Bit {
			result.MemData = s.memory.Read64(addr)
			// LDP (load pair) reads a second consecutive value for Rt2
			if exmem.Inst.Op == insts.OpLDP {
				result.MemData2 = s.memory.Read64(addr + 8)
			}
		} else {
			result.MemData = uint64(s.memory.Read32(addr))
			// LDP (load pair) reads a second consecutive value for Rt2
			if exmem.Inst != nil && exmem.Inst.Op == insts.OpLDP {
				result.MemData2 = uint64(s.memory.Read32(addr + 4))
			}
		}
	}

	if exmem.MemWrite {
		// Store: write to memory
		if exmem.Inst != nil && exmem.Inst.Is64Bit {
			s.memory.Write64(addr, exmem.StoreValue)
		} else {
			s.memory.Write32(addr, uint32(exmem.StoreValue))
		}
	}

	return result
}

// MemorySlot interface for memory stage processing.
// Implemented by all EXMEM register types.
type MemorySlot interface {
	IsValid() bool
	GetPC() uint64
	GetMemRead() bool
	GetMemWrite() bool
	GetInst() *insts.Instruction
	GetALUResult() uint64
	GetStoreValue() uint64
}

// MemorySlot performs memory access for any EXMEM slot.
// Returns the memory result.
func (s *MemoryStage) MemorySlot(slot MemorySlot) MemoryResult {
	result := MemoryResult{}

	if !slot.IsValid() {
		return result
	}

	addr := slot.GetALUResult()
	inst := slot.GetInst()

	if slot.GetMemRead() {
		// Load: read from memory
		if inst != nil && inst.Is64Bit {
			result.MemData = s.memory.Read64(addr)
			// LDP (load pair) reads a second consecutive value for Rt2
			if inst.Op == insts.OpLDP {
				result.MemData2 = s.memory.Read64(addr + 8)
			}
		} else {
			result.MemData = uint64(s.memory.Read32(addr))
			// LDP (load pair) reads a second consecutive value for Rt2
			if inst != nil && inst.Op == insts.OpLDP {
				result.MemData2 = uint64(s.memory.Read32(addr + 4))
			}
		}
	}

	if slot.GetMemWrite() {
		// Store: write to memory
		if inst != nil && inst.Is64Bit {
			s.memory.Write64(addr, slot.GetStoreValue())
		} else {
			s.memory.Write32(addr, uint32(slot.GetStoreValue()))
		}
	}

	return result
}

// WritebackStage writes results back to the register file.
type WritebackStage struct {
	regFile *emu.RegFile
}

// NewWritebackStage creates a new writeback stage.
func NewWritebackStage(regFile *emu.RegFile) *WritebackStage {
	return &WritebackStage{regFile: regFile}
}

// Writeback writes the result to the destination register.
func (s *WritebackStage) Writeback(memwb *MEMWBRegister) {
	if !memwb.Valid || !memwb.RegWrite {
		return
	}

	// Don't write to XZR
	if memwb.Rd == 31 {
		return
	}

	var value uint64
	if memwb.MemToReg {
		value = memwb.MemData
	} else {
		value = memwb.ALUResult
	}

	s.regFile.WriteReg(memwb.Rd, value)
}

// WritebackSlot interface for writeback stage processing.
// Implemented by all MEMWB register types.
type WritebackSlot interface {
	IsValid() bool
	GetRegWrite() bool
	GetRd() uint8
	GetMemToReg() bool
	GetALUResult() uint64
	GetMemData() uint64
	GetMemData2() uint64 // Second loaded value for LDP (load pair) instructions
	GetIsFused() bool
	GetInst() *insts.Instruction
}

// writebackSlot performs writeback for any MEMWB slot.
// Returns true if an instruction was retired.
func (s *WritebackStage) WritebackSlot(slot WritebackSlot) bool {
	if !slot.IsValid() {
		return false // Not retired
	}

	// Handle base register writeback for pre/post-indexed addressing.
	// This updates the base register (Rn) or SP after indexed load/store.
	s.performBaseWriteback(slot)

	if !slot.GetRegWrite() {
		return true // Valid but no regwrite still counts as retired
	}

	// Don't write to XZR
	if slot.GetRd() == 31 {
		return true // Instruction retired
	}

	var value uint64
	if slot.GetMemToReg() {
		value = slot.GetMemData()
	} else {
		value = slot.GetALUResult()
	}

	s.regFile.WriteReg(slot.GetRd(), value)

	// Handle LDP (load pair): write the second register (Rt2) with the second loaded value.
	// LDP loads two consecutive values; the pipeline stores the second in MemData2.
	inst := slot.GetInst()
	if inst != nil && inst.Op == insts.OpLDP && slot.GetMemToReg() {
		if inst.Rt2 != 31 {
			s.regFile.WriteReg(inst.Rt2, slot.GetMemData2())
		}
	}

	return true
}

// performBaseWriteback handles base register writeback for pre/post-indexed
// load/store instructions (e.g., str x2, [x0], #8 or ldr x0, [sp, #-16]!).
func (s *WritebackStage) performBaseWriteback(slot WritebackSlot) {
	inst := slot.GetInst()
	if inst == nil {
		return
	}
	if inst.IndexMode != insts.IndexPre && inst.IndexMode != insts.IndexPost {
		return
	}

	// Compute the writeback address:
	// - Pre-indexed:  address = base + offset (already in ALUResult), writeback = ALUResult
	// - Post-indexed: address = base (already in ALUResult), writeback = base + SignedImm
	var writebackVal uint64
	if inst.IndexMode == insts.IndexPost {
		writebackVal = uint64(int64(slot.GetALUResult()) + inst.SignedImm)
	} else {
		writebackVal = slot.GetALUResult()
	}

	if inst.Rn == 31 {
		s.regFile.SP = writebackVal
	} else {
		s.regFile.WriteReg(inst.Rn, writebackVal)
	}
}

// WritebackSlots performs batched writeback for multiple MEMWB slots.
// Returns the total number of instructions retired.
// This optimization reduces function call overhead in tickOctupleIssue.
func (s *WritebackStage) WritebackSlots(slots []WritebackSlot) uint64 {
	retired := uint64(0)

	// Batch process all slots to reduce function call overhead
	for _, slot := range slots {
		if !slot.IsValid() {
			continue
		}

		retired++

		// Handle base register writeback for pre/post-indexed addressing
		s.performBaseWriteback(slot)

		// Skip register write operations
		if !slot.GetRegWrite() || slot.GetRd() == 31 {
			continue
		}

		// Select value source
		var value uint64
		if slot.GetMemToReg() {
			value = slot.GetMemData()
		} else {
			value = slot.GetALUResult()
		}

		// Write to register file
		s.regFile.WriteReg(slot.GetRd(), value)

		// Handle LDP (load pair): write the second register (Rt2) with the second loaded value.
		inst := slot.GetInst()
		if inst != nil && inst.Op == insts.OpLDP && slot.GetMemToReg() {
			if inst.Rt2 != 31 {
				s.regFile.WriteReg(inst.Rt2, slot.GetMemData2())
			}
		}
	}

	return retired
}

// IsCMP returns true if the instruction is a CMP (compare) operation.
// CMP is encoded as SUB/SUBS with Rd=31 (XZR) and SetFlags=true.
func IsCMP(inst *insts.Instruction) bool {
	if inst == nil {
		return false
	}
	return inst.Op == insts.OpSUB && inst.SetFlags && inst.Rd == 31
}

// IsBCond returns true if the instruction is a conditional branch (B.cond).
func IsBCond(inst *insts.Instruction) bool {
	if inst == nil {
		return false
	}
	return inst.Op == insts.OpBCond
}

// ComputeSubFlags computes PSTATE flags from a SUB/CMP operation.
// Returns N, Z, C, V flags.
func ComputeSubFlags(op1, op2 uint64, is64Bit bool) (n, z, c, v bool) {
	if is64Bit {
		result := op1 - op2
		n = (result >> 63) == 1
		z = result == 0
		c = op1 >= op2 // no borrow
		// V: signed overflow - subtracting different signs gives wrong sign
		op1Sign := op1 >> 63
		op2Sign := op2 >> 63
		resultSign := result >> 63
		v = (op1Sign != op2Sign) && (op2Sign == resultSign)
	} else {
		o1 := uint32(op1)
		o2 := uint32(op2)
		r32 := o1 - o2
		n = (r32 >> 31) == 1
		z = r32 == 0
		c = o1 >= o2
		op1Sign := o1 >> 31
		op2Sign := o2 >> 31
		resultSign := r32 >> 31
		v = (op1Sign != op2Sign) && (op2Sign == resultSign)
	}
	return
}

// EvaluateConditionWithFlags evaluates a branch condition with given PSTATE flags.
func EvaluateConditionWithFlags(cond insts.Cond, n, z, c, v bool) bool {
	switch cond {
	case insts.CondEQ:
		return z
	case insts.CondNE:
		return !z
	case insts.CondCS:
		return c
	case insts.CondCC:
		return !c
	case insts.CondMI:
		return n
	case insts.CondPL:
		return !n
	case insts.CondVS:
		return v
	case insts.CondVC:
		return !v
	case insts.CondHI:
		return c && !z
	case insts.CondLS:
		return !c || z
	case insts.CondGE:
		return n == v
	case insts.CondLT:
		return n != v
	case insts.CondGT:
		return !z && (n == v)
	case insts.CondLE:
		return z || (n != v)
	case insts.CondAL, insts.CondNV:
		return true
	default:
		return false
	}
}
