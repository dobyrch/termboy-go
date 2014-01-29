package cpu

import (
	"errors"
	"fmt"
	"github.com/dobyrch/termboy-go/constants"
	"github.com/dobyrch/termboy-go/mmu"
	"github.com/dobyrch/termboy-go/types"
	"github.com/dobyrch/termboy-go/utils"
	"log"
)

const NAME = "CPU"
const PREFIX = NAME + ":"

//flags
const (
	_ = iota
	C
	H
	N
	Z
)

type CPUFrame struct {
	PC                      types.Word // Program Counter
	SP                      types.Word // Stack Pointer
	R                       Registers
	InterruptsEnabled       bool
	CurrentInstruction      Instruction
	LastInstrCycle          Clock
	PCJumped                bool
	Halted                  bool
	InterruptFlagBeforeHalt byte
}

func (cpu *CPU) GetFrame() *CPUFrame {
	var frame *CPUFrame = new(CPUFrame)
	frame.PC = cpu.PC
	frame.SP = cpu.SP
	frame.R = cpu.R
	frame.InterruptsEnabled = cpu.InterruptsEnabled
	frame.CurrentInstruction = cpu.CurrentInstruction
	frame.LastInstrCycle = cpu.LastInstrCycle
	frame.PCJumped = cpu.PCJumped
	frame.Halted = cpu.Halted
	frame.InterruptFlagBeforeHalt = cpu.InterruptFlagBeforeHalt
	return frame
}

type Registers struct {
	A byte
	B byte
	C byte
	D byte
	E byte
	H byte
	L byte
	F byte // Flags Register
}

func (r Registers) String() string {
	formatByte := func(b byte) string {
		if b < 0x10 {
			return fmt.Sprintf("0x0%X", b)
		}
		return fmt.Sprintf("0x%X", b)
	}
	return fmt.Sprintf("A: %s  B: %s  C: %s  D: %s  E: %s  H: %s  L: %s", formatByte(r.A), formatByte(r.B), formatByte(r.C), formatByte(r.D), formatByte(r.E), formatByte(r.H), formatByte(r.L))
}

//See ZILOG z80 cpu manual p.80  (http://www.zilog.com/docs/z80/um0080.pdf)
type Clock struct {
	M int
	t int
}

func (c *Clock) T() int {
	return c.M * 4
}

func (c *Clock) Reset() {
	c.M, c.t = 0, 0
}

func (c *Clock) String() string {
	return fmt.Sprintf("[M: %X, T: %X]", c.M, c.t)
}

type CPU struct {
	PC                      types.Word // Program Counter
	SP                      types.Word // Stack Pointer
	R                       Registers
	InterruptsEnabled       bool
	CurrentInstruction      Instruction
	LastInstrCycle          Clock
	mmu                     mmu.MemoryMappedUnit
	PCJumped                bool
	Halted                  bool
	InterruptFlagBeforeHalt byte
	Speed                   int
}

func NewCPU() *CPU {
	cpu := new(CPU)
	cpu.Reset()
	return cpu
}

func (cpu *CPU) LinkMMU(m mmu.MemoryMappedUnit) {
	cpu.mmu = m
	log.Println(PREFIX, "Linked CPU to MMU")
}

func (cpu *CPU) Validate() error {
	if cpu.mmu == nil {
		return errors.New("No MMU linked to CPU")
	}
	return nil
}

func (cpu *CPU) Reset() {
	log.Println(PREFIX, "Resetting", NAME)
	cpu.PC = 0
	cpu.SP = 0
	cpu.R.A = 0
	cpu.R.B = 0
	cpu.R.C = 0
	cpu.R.D = 0
	cpu.R.E = 0
	cpu.R.F = 0
	cpu.R.H = 0
	cpu.R.L = 0
	cpu.Speed = 1
	cpu.CurrentInstruction, _ = cpu.Decode(0x00)
	cpu.InterruptsEnabled = true
	cpu.LastInstrCycle.Reset()
	cpu.PCJumped = false
	cpu.Halted = false
}

func (cpu *CPU) FlagsString() string {
	var flags string = ""
	var minus string = "-"

	if cpu.IsFlagSet(Z) {
		flags += "Z"
	} else {
		flags += minus
	}

	if cpu.IsFlagSet(N) {
		flags += "N"
	} else {
		flags += minus
	}

	if cpu.IsFlagSet(H) {
		flags += "H"
	} else {
		flags += minus
	}

	if cpu.IsFlagSet(C) {
		flags += "C"
	} else {
		flags += minus
	}
	return flags
}

func (cpu *CPU) String() string {
	return fmt.Sprint("PC: ", cpu.PC, "  SP: ", cpu.SP, "  ", cpu.R, "  ", cpu.FlagsString(), "  ", cpu.CurrentInstruction)
}

func (cpu *CPU) ResetFlag(flag int) {
	switch flag {
	case Z:
		cpu.R.F = cpu.resetBit(7, cpu.R.F)
	case N:
		cpu.R.F = cpu.resetBit(6, cpu.R.F)
	case H:
		cpu.R.F = cpu.resetBit(5, cpu.R.F)
	case C:
		cpu.R.F = cpu.resetBit(4, cpu.R.F)
	default:
		log.Panicf(PREFIX+" Unknown flag %c", flag)
	}
	cpu.R.F &= 0xF0
}

func (cpu *CPU) SetFlag(flag int) {
	switch flag {
	case Z:
		cpu.R.F = cpu.setBit(7, cpu.R.F)
	case N:
		cpu.R.F = cpu.setBit(6, cpu.R.F)
	case H:
		cpu.R.F = cpu.setBit(5, cpu.R.F)
	case C:
		cpu.R.F = cpu.setBit(4, cpu.R.F)
	default:
		log.Panicf(PREFIX+" Unknown flag %c", flag)
	}
	cpu.R.F &= 0xF0
}

func (cpu *CPU) IsFlagSet(flag int) bool {
	switch flag {
	case Z:
		return cpu.R.F&0x80 == 0x80
	case N:
		return cpu.R.F&0x40 == 0x40
	case H:
		return cpu.R.F&0x20 == 0x20
	case C:
		return cpu.R.F&0x10 == 0x10
	default:
		log.Panicf(PREFIX+" Unknown flag %c", flag)
	}
	return false
}

func (cpu *CPU) IncrementPC(by int) {
	cpu.PC += types.Word(by)
}

func (cpu *CPU) DispatchCB(Opcode byte) {
	switch Opcode {
	case 0x00: //RLC B
		cpu.Rlc_r(&cpu.R.B)
	case 0x01: //RLC C
		cpu.Rlc_r(&cpu.R.C)
	case 0x02: //RLC D
		cpu.Rlc_r(&cpu.R.D)
	case 0x03: //RLC E
		cpu.Rlc_r(&cpu.R.E)
	case 0x04: //RLC H
		cpu.Rlc_r(&cpu.R.H)
	case 0x05: //RLC L
		cpu.Rlc_r(&cpu.R.L)
	case 0x06: //RLC (HL)
		cpu.Rlc_hl()
	case 0x07: //RLC A
		cpu.Rlc_r(&cpu.R.A)
	case 0x08: //RRC B
		cpu.Rrc_r(&cpu.R.B)
	case 0x09: //RRC C
		cpu.Rrc_r(&cpu.R.C)
	case 0x0A: //RRC D
		cpu.Rrc_r(&cpu.R.D)
	case 0x0B: //RRC E
		cpu.Rrc_r(&cpu.R.E)
	case 0x0C: //RRC H
		cpu.Rrc_r(&cpu.R.H)
	case 0x0D: //RRC L
		cpu.Rrc_r(&cpu.R.L)
	case 0x0E: //RRC (HL)
		cpu.Rrc_hl()
	case 0x0F: //RRC A
		cpu.Rrc_r(&cpu.R.A)
	case 0x10: //RL B
		cpu.Rl_r(&cpu.R.B)
	case 0x11: //RL C
		cpu.Rl_r(&cpu.R.C)
	case 0x12: //RL D
		cpu.Rl_r(&cpu.R.D)
	case 0x13: //RL E
		cpu.Rl_r(&cpu.R.E)
	case 0x14: //RL H
		cpu.Rl_r(&cpu.R.H)
	case 0x15: //RL L
		cpu.Rl_r(&cpu.R.L)
	case 0x16: //RL (HL)
		cpu.Rl_hl()
	case 0x17: //RL A
		cpu.Rl_r(&cpu.R.A)
	case 0x18: //RR B
		cpu.Rr_r(&cpu.R.B)
	case 0x19: //RR C
		cpu.Rr_r(&cpu.R.C)
	case 0x1A: //RR D
		cpu.Rr_r(&cpu.R.D)
	case 0x1B: //RR E
		cpu.Rr_r(&cpu.R.E)
	case 0x1C: //RR H
		cpu.Rr_r(&cpu.R.H)
	case 0x1D: //RR L
		cpu.Rr_r(&cpu.R.L)
	case 0x1E: //RR (HL)
		cpu.Rr_hl()
	case 0x1F: //RR A
		cpu.Rr_r(&cpu.R.A)
	case 0x20: //SLA B
		cpu.Sla_r(&cpu.R.B)
	case 0x21: //SLA C
		cpu.Sla_r(&cpu.R.C)
	case 0x22: //SLA D
		cpu.Sla_r(&cpu.R.D)
	case 0x23: //SLA E
		cpu.Sla_r(&cpu.R.E)
	case 0x24: //SLA H
		cpu.Sla_r(&cpu.R.H)
	case 0x25: //SLA L
		cpu.Sla_r(&cpu.R.L)
	case 0x26: //SLA (HL)
		cpu.Sla_hl()
	case 0x27: //SLA A
		cpu.Sla_r(&cpu.R.A)
	case 0x28: //SRA B
		cpu.Sra_r(&cpu.R.B)
	case 0x29: //SRA C
		cpu.Sra_r(&cpu.R.C)
	case 0x2A: //SRA D
		cpu.Sra_r(&cpu.R.D)
	case 0x2B: //SRA E
		cpu.Sra_r(&cpu.R.E)
	case 0x2C: //SRA H
		cpu.Sra_r(&cpu.R.H)
	case 0x2D: //SRA L
		cpu.Sra_r(&cpu.R.L)
	case 0x2E: //SRA (HL)
		cpu.Sra_hl()
	case 0x2F: //SRA A
		cpu.Sra_r(&cpu.R.A)
	case 0x30: //SWAP B
		cpu.Swap_r(&cpu.R.B)
	case 0x31: //SWAP C
		cpu.Swap_r(&cpu.R.C)
	case 0x32: //SWAP D
		cpu.Swap_r(&cpu.R.D)
	case 0x33: //SWAP E
		cpu.Swap_r(&cpu.R.E)
	case 0x34: //SWAP H
		cpu.Swap_r(&cpu.R.H)
	case 0x35: //SWAP L
		cpu.Swap_r(&cpu.R.L)
	case 0x36: //SWAP (HL)
		cpu.Swap_hl()
	case 0x37: //SWAP A
		cpu.Swap_r(&cpu.R.A)
	case 0x38: //SRL B
		cpu.Srl_r(&cpu.R.B)
	case 0x39: //SRL C
		cpu.Srl_r(&cpu.R.C)
	case 0x3A: //SRL D
		cpu.Srl_r(&cpu.R.D)
	case 0x3B: //SRL E
		cpu.Srl_r(&cpu.R.E)
	case 0x3C: //SRL H
		cpu.Srl_r(&cpu.R.H)
	case 0x3D: //SRL L
		cpu.Srl_r(&cpu.R.L)
	case 0x3E: //SRL (HL)
		cpu.Srl_hl()
	case 0x3F: //SRL A
		cpu.Srl_r(&cpu.R.A)
	case 0x40: //BIT 0, B
		cpu.Bitb_r(0x00, &cpu.R.B)
	case 0x41: //BIT 0, C
		cpu.Bitb_r(0x00, &cpu.R.C)
	case 0x42: //BIT 0, D
		cpu.Bitb_r(0x00, &cpu.R.D)
	case 0x43: //BIT 0, E
		cpu.Bitb_r(0x00, &cpu.R.E)
	case 0x44: //BIT 0, H
		cpu.Bitb_r(0x00, &cpu.R.H)
	case 0x45: //BIT 0, L
		cpu.Bitb_r(0x00, &cpu.R.L)
	case 0x46: //BIT 0, (HL)
		cpu.Bitb_hl(0x00)
	case 0x47: //BIT 0, A
		cpu.Bitb_r(0x00, &cpu.R.A)
	case 0x48: //BIT 1, B
		cpu.Bitb_r(0x01, &cpu.R.B)
	case 0x49: //BIT 1, C
		cpu.Bitb_r(0x01, &cpu.R.C)
	case 0x4A: //BIT 1, D
		cpu.Bitb_r(0x01, &cpu.R.D)
	case 0x4B: //BIT 1, E
		cpu.Bitb_r(0x01, &cpu.R.E)
	case 0x4C: //BIT 1, H
		cpu.Bitb_r(0x01, &cpu.R.H)
	case 0x4D: //BIT 1, L
		cpu.Bitb_r(0x01, &cpu.R.L)
	case 0x4E: //BIT 1, (HL)
		cpu.Bitb_hl(0x01)
	case 0x4F: //BIT 1, A
		cpu.Bitb_r(0x01, &cpu.R.A)
	case 0x50: //BIT 2, B
		cpu.Bitb_r(0x02, &cpu.R.B)
	case 0x51: //BIT 2, C
		cpu.Bitb_r(0x02, &cpu.R.C)
	case 0x52: //BIT 2, D
		cpu.Bitb_r(0x02, &cpu.R.D)
	case 0x53: //BIT 2, E
		cpu.Bitb_r(0x02, &cpu.R.E)
	case 0x54: //BIT 2, H
		cpu.Bitb_r(0x02, &cpu.R.H)
	case 0x55: //BIT 2, L
		cpu.Bitb_r(0x02, &cpu.R.L)
	case 0x56: //BIT 2, (HL)
		cpu.Bitb_hl(0x02)
	case 0x57: //BIT 2, A
		cpu.Bitb_r(0x02, &cpu.R.A)
	case 0x58: //BIT 3, B
		cpu.Bitb_r(0x03, &cpu.R.B)
	case 0x59: //BIT 3, C
		cpu.Bitb_r(0x03, &cpu.R.C)
	case 0x5A: //BIT 3, D
		cpu.Bitb_r(0x03, &cpu.R.D)
	case 0x5B: //BIT 3, E
		cpu.Bitb_r(0x03, &cpu.R.E)
	case 0x5C: //BIT 3, H
		cpu.Bitb_r(0x03, &cpu.R.H)
	case 0x5D: //BIT 3, L
		cpu.Bitb_r(0x03, &cpu.R.L)
	case 0x5E: //BIT 3, (HL)
		cpu.Bitb_hl(0x03)
	case 0x5F: //BIT 3, A
		cpu.Bitb_r(0x03, &cpu.R.A)
	case 0x60: //BIT 4, B
		cpu.Bitb_r(0x04, &cpu.R.B)
	case 0x61: //BIT 4, C
		cpu.Bitb_r(0x04, &cpu.R.C)
	case 0x62: //BIT 4, D
		cpu.Bitb_r(0x04, &cpu.R.D)
	case 0x63: //BIT 4, E
		cpu.Bitb_r(0x04, &cpu.R.E)
	case 0x64: //BIT 4, H
		cpu.Bitb_r(0x04, &cpu.R.H)
	case 0x65: //BIT 4, L
		cpu.Bitb_r(0x04, &cpu.R.L)
	case 0x66: //BIT 4, (HL)
		cpu.Bitb_hl(0x04)
	case 0x67: //BIT 4, A
		cpu.Bitb_r(0x04, &cpu.R.A)
	case 0x68: //BIT 5, B
		cpu.Bitb_r(0x05, &cpu.R.B)
	case 0x69: //BIT 5, C
		cpu.Bitb_r(0x05, &cpu.R.C)
	case 0x6A: //BIT 5, D
		cpu.Bitb_r(0x05, &cpu.R.D)
	case 0x6B: //BIT 5, E
		cpu.Bitb_r(0x05, &cpu.R.E)
	case 0x6C: //BIT 5, H
		cpu.Bitb_r(0x05, &cpu.R.H)
	case 0x6D: //BIT 5, L
		cpu.Bitb_r(0x05, &cpu.R.L)
	case 0x6E: //BIT 5, (HL)
		cpu.Bitb_hl(0x05)
	case 0x6F: //BIT 5, A
		cpu.Bitb_r(0x05, &cpu.R.A)
	case 0x70: //BIT 6, B
		cpu.Bitb_r(0x06, &cpu.R.B)
	case 0x71: //BIT 6, C
		cpu.Bitb_r(0x06, &cpu.R.C)
	case 0x72: //BIT 6, D
		cpu.Bitb_r(0x06, &cpu.R.D)
	case 0x73: //BIT 6, E
		cpu.Bitb_r(0x06, &cpu.R.E)
	case 0x74: //BIT 6, H
		cpu.Bitb_r(0x06, &cpu.R.H)
	case 0x75: //BIT 6, L
		cpu.Bitb_r(0x06, &cpu.R.L)
	case 0x76: //BIT 6, (HL)
		cpu.Bitb_hl(0x06)
	case 0x77: //BIT 6, A
		cpu.Bitb_r(0x06, &cpu.R.A)
	case 0x78: //BIT 7, B
		cpu.Bitb_r(0x07, &cpu.R.B)
	case 0x79: //BIT 7, C
		cpu.Bitb_r(0x07, &cpu.R.C)
	case 0x7A: //BIT 7, D
		cpu.Bitb_r(0x07, &cpu.R.D)
	case 0x7B: //BIT 7, E
		cpu.Bitb_r(0x07, &cpu.R.E)
	case 0x7C: //BIT 7, H
		cpu.Bitb_r(0x07, &cpu.R.H)
	case 0x7D: //BIT 7, L
		cpu.Bitb_r(0x07, &cpu.R.L)
	case 0x7E: //BIT 7, (HL)
		cpu.Bitb_hl(0x07)
	case 0x7F: //BIT 7, A
		cpu.Bitb_r(0x07, &cpu.R.A)
	case 0x80: //RES 0, B
		cpu.Resb_r(0x00, &cpu.R.B)
	case 0x81: //RES 0, C
		cpu.Resb_r(0x00, &cpu.R.C)
	case 0x82: //RES 0, D
		cpu.Resb_r(0x00, &cpu.R.D)
	case 0x83: //RES 0, E
		cpu.Resb_r(0x00, &cpu.R.E)
	case 0x84: //RES 0, H
		cpu.Resb_r(0x00, &cpu.R.H)
	case 0x85: //RES 0, L
		cpu.Resb_r(0x00, &cpu.R.L)
	case 0x86: //RES 0,(HL)
		cpu.Resb_hl(0x00)
	case 0x87: //RES 0, A
		cpu.Resb_r(0x00, &cpu.R.A)
	case 0x88: //RES 1, B
		cpu.Resb_r(0x01, &cpu.R.B)
	case 0x89: //RES 1, C
		cpu.Resb_r(0x01, &cpu.R.C)
	case 0x8A: //RES 1, D
		cpu.Resb_r(0x01, &cpu.R.D)
	case 0x8B: //RES 1, E
		cpu.Resb_r(0x01, &cpu.R.E)
	case 0x8C: //RES 1, H
		cpu.Resb_r(0x01, &cpu.R.H)
	case 0x8D: //RES 1, L
		cpu.Resb_r(0x01, &cpu.R.L)
	case 0x8E: //RES 1,(HL)
		cpu.Resb_hl(0x01)
	case 0x8F: //RES 1, A
		cpu.Resb_r(0x01, &cpu.R.A)
	case 0x90: //RES 2, B
		cpu.Resb_r(0x02, &cpu.R.B)
	case 0x91: //RES 2, C
		cpu.Resb_r(0x02, &cpu.R.C)
	case 0x92: //RES 2, D
		cpu.Resb_r(0x02, &cpu.R.D)
	case 0x93: //RES 2, E
		cpu.Resb_r(0x02, &cpu.R.E)
	case 0x94: //RES 2, H
		cpu.Resb_r(0x02, &cpu.R.H)
	case 0x95: //RES 2, L
		cpu.Resb_r(0x02, &cpu.R.L)
	case 0x96: //RES 2,(HL)
		cpu.Resb_hl(0x02)
	case 0x97: //RES 2, A
		cpu.Resb_r(0x02, &cpu.R.A)
	case 0x98: //RES 3, B
		cpu.Resb_r(0x03, &cpu.R.B)
	case 0x99: //RES 3, C
		cpu.Resb_r(0x03, &cpu.R.C)
	case 0x9A: //RES 3, D
		cpu.Resb_r(0x03, &cpu.R.D)
	case 0x9B: //RES 3, E
		cpu.Resb_r(0x03, &cpu.R.E)
	case 0x9C: //RES 3, H
		cpu.Resb_r(0x03, &cpu.R.H)
	case 0x9D: //RES 3, L
		cpu.Resb_r(0x03, &cpu.R.L)
	case 0x9E: //RES 3,(HL)
		cpu.Resb_hl(0x03)
	case 0x9F: //RES 3, A
		cpu.Resb_r(0x03, &cpu.R.A)
	case 0xA0: //RES 4, B
		cpu.Resb_r(0x04, &cpu.R.B)
	case 0xA1: //RES 4, C
		cpu.Resb_r(0x04, &cpu.R.C)
	case 0xA2: //RES 4, D
		cpu.Resb_r(0x04, &cpu.R.D)
	case 0xA3: //RES 4, E
		cpu.Resb_r(0x04, &cpu.R.E)
	case 0xA4: //RES 4, H
		cpu.Resb_r(0x04, &cpu.R.H)
	case 0xA5: //RES 4, L
		cpu.Resb_r(0x04, &cpu.R.L)
	case 0xA6: //RES 4,(HL)
		cpu.Resb_hl(0x04)
	case 0xA7: //RES 4, A
		cpu.Resb_r(0x04, &cpu.R.A)
	case 0xA8: //RES 5, B
		cpu.Resb_r(0x05, &cpu.R.B)
	case 0xA9: //RES 5, C
		cpu.Resb_r(0x05, &cpu.R.C)
	case 0xAA: //RES 5, D
		cpu.Resb_r(0x05, &cpu.R.D)
	case 0xAB: //RES 5, E
		cpu.Resb_r(0x05, &cpu.R.E)
	case 0xAC: //RES 5, H
		cpu.Resb_r(0x05, &cpu.R.H)
	case 0xAD: //RES 5, L
		cpu.Resb_r(0x05, &cpu.R.L)
	case 0xAE: //RES 5,(HL)
		cpu.Resb_hl(0x05)
	case 0xAF: //RES 5, A
		cpu.Resb_r(0x05, &cpu.R.A)
	case 0xB0: //RES 6, B
		cpu.Resb_r(0x06, &cpu.R.B)
	case 0xB1: //RES 6, C
		cpu.Resb_r(0x06, &cpu.R.C)
	case 0xB2: //RES 6, D
		cpu.Resb_r(0x06, &cpu.R.D)
	case 0xB3: //RES 6, E
		cpu.Resb_r(0x06, &cpu.R.E)
	case 0xB4: //RES 6, H
		cpu.Resb_r(0x06, &cpu.R.H)
	case 0xB5: //RES 6, L
		cpu.Resb_r(0x06, &cpu.R.L)
	case 0xB6: //RES 6,(HL)
		cpu.Resb_hl(0x06)
	case 0xB7: //RES 6, A
		cpu.Resb_r(0x06, &cpu.R.A)
	case 0xB8: //RES 7, B
		cpu.Resb_r(0x07, &cpu.R.B)
	case 0xB9: //RES 7, C
		cpu.Resb_r(0x07, &cpu.R.C)
	case 0xBA: //RES 7, D
		cpu.Resb_r(0x07, &cpu.R.D)
	case 0xBB: //RES 7, E
		cpu.Resb_r(0x07, &cpu.R.E)
	case 0xBC: //RES 7, H
		cpu.Resb_r(0x07, &cpu.R.H)
	case 0xBD: //RES 7, L
		cpu.Resb_r(0x07, &cpu.R.L)
	case 0xBE: //RES 7,(HL)
		cpu.Resb_hl(0x07)
	case 0xBF: //RES 7, A
		cpu.Resb_r(0x07, &cpu.R.A)
	case 0xC0: //SET 0, B
		cpu.Setb_r(0x00, &cpu.R.B)
	case 0xC1: //SET 0, C
		cpu.Setb_r(0x00, &cpu.R.C)
	case 0xC2: //SET 0, D
		cpu.Setb_r(0x00, &cpu.R.D)
	case 0xC3: //SET 0, E
		cpu.Setb_r(0x00, &cpu.R.E)
	case 0xC4: //SET 0, H
		cpu.Setb_r(0x00, &cpu.R.H)
	case 0xC5: //SET 0, L
		cpu.Setb_r(0x00, &cpu.R.L)
	case 0xC6: //SET 0, (HL)
		cpu.Setb_hl(0x00)
	case 0xC7: //SET 0, A
		cpu.Setb_r(0x00, &cpu.R.A)
	case 0xC8: //SET 1, B
		cpu.Setb_r(0x01, &cpu.R.B)
	case 0xC9: //SET 1, C
		cpu.Setb_r(0x01, &cpu.R.C)
	case 0xCA: //SET 1, D
		cpu.Setb_r(0x01, &cpu.R.D)
	case 0xCB: //SET 1, E
		cpu.Setb_r(0x01, &cpu.R.E)
	case 0xCC: //SET 1, H
		cpu.Setb_r(0x01, &cpu.R.H)
	case 0xCD: //SET 1, L
		cpu.Setb_r(0x01, &cpu.R.L)
	case 0xCE: //SET 1, (HL)
		cpu.Setb_hl(0x01)
	case 0xCF: //SET 1, A
		cpu.Setb_r(0x01, &cpu.R.A)
	case 0xD0: //SET 2, B
		cpu.Setb_r(0x02, &cpu.R.B)
	case 0xD1: //SET 2, C
		cpu.Setb_r(0x02, &cpu.R.C)
	case 0xD2: //SET 2, D
		cpu.Setb_r(0x02, &cpu.R.D)
	case 0xD3: //SET 2, E
		cpu.Setb_r(0x02, &cpu.R.E)
	case 0xD4: //SET 2, H
		cpu.Setb_r(0x02, &cpu.R.H)
	case 0xD5: //SET 2, L
		cpu.Setb_r(0x02, &cpu.R.L)
	case 0xD6: //SET 2, (HL)
		cpu.Setb_hl(0x02)
	case 0xD7: //SET 2, A
		cpu.Setb_r(0x02, &cpu.R.A)
	case 0xD8: //SET 3, B
		cpu.Setb_r(0x03, &cpu.R.B)
	case 0xD9: //SET 3, C
		cpu.Setb_r(0x03, &cpu.R.C)
	case 0xDA: //SET 3, D
		cpu.Setb_r(0x03, &cpu.R.D)
	case 0xDB: //SET 3, E
		cpu.Setb_r(0x03, &cpu.R.E)
	case 0xDC: //SET 3, H
		cpu.Setb_r(0x03, &cpu.R.H)
	case 0xDD: //SET 3, L
		cpu.Setb_r(0x03, &cpu.R.L)
	case 0xDE: //SET 3, (HL)
		cpu.Setb_hl(0x03)
	case 0xDF: //SET 3, A
		cpu.Setb_r(0x03, &cpu.R.A)
	case 0xE0: //SET 4, B
		cpu.Setb_r(0x04, &cpu.R.B)
	case 0xE1: //SET 4, C
		cpu.Setb_r(0x04, &cpu.R.C)
	case 0xE2: //SET 4, D
		cpu.Setb_r(0x04, &cpu.R.D)
	case 0xE3: //SET 4, E
		cpu.Setb_r(0x04, &cpu.R.E)
	case 0xE4: //SET 4, H
		cpu.Setb_r(0x04, &cpu.R.H)
	case 0xE5: //SET 4, L
		cpu.Setb_r(0x04, &cpu.R.L)
	case 0xE6: //SET 4, (HL)
		cpu.Setb_hl(0x04)
	case 0xE7: //SET 4, A
		cpu.Setb_r(0x04, &cpu.R.A)
	case 0xE8: //SET 5, B
		cpu.Setb_r(0x05, &cpu.R.B)
	case 0xE9: //SET 5, C
		cpu.Setb_r(0x05, &cpu.R.C)
	case 0xEA: //SET 5, D
		cpu.Setb_r(0x05, &cpu.R.D)
	case 0xEB: //SET 5, E
		cpu.Setb_r(0x05, &cpu.R.E)
	case 0xEC: //SET 5, H
		cpu.Setb_r(0x05, &cpu.R.H)
	case 0xED: //SET 5, L
		cpu.Setb_r(0x05, &cpu.R.L)
	case 0xEE: //SET 5, (HL)
		cpu.Setb_hl(0x05)
	case 0xEF: //SET 5, A
		cpu.Setb_r(0x05, &cpu.R.A)
	case 0xF0: //SET 6, B
		cpu.Setb_r(0x06, &cpu.R.B)
	case 0xF1: //SET 6, C
		cpu.Setb_r(0x06, &cpu.R.C)
	case 0xF2: //SET 6, D
		cpu.Setb_r(0x06, &cpu.R.D)
	case 0xF3: //SET 6, E
		cpu.Setb_r(0x06, &cpu.R.E)
	case 0xF4: //SET 6, H
		cpu.Setb_r(0x06, &cpu.R.H)
	case 0xF5: //SET 6, L
		cpu.Setb_r(0x06, &cpu.R.L)
	case 0xF6: //SET 6, (HL)
		cpu.Setb_hl(0x06)
	case 0xF7: //SET 6, A
		cpu.Setb_r(0x06, &cpu.R.A)
	case 0xF8: //SET 7, B
		cpu.Setb_r(0x07, &cpu.R.B)
	case 0xF9: //SET 7, C
		cpu.Setb_r(0x07, &cpu.R.C)
	case 0xFA: //SET 7, D
		cpu.Setb_r(0x07, &cpu.R.D)
	case 0xFB: //SET 7, E
		cpu.Setb_r(0x07, &cpu.R.E)
	case 0xFC: //SET 7, H
		cpu.Setb_r(0x07, &cpu.R.H)
	case 0xFD: //SET 7, L
		cpu.Setb_r(0x07, &cpu.R.L)
	case 0xFE: //SET 7, (HL)
		cpu.Setb_hl(0x07)
	case 0xFF: //SET 7, A
		cpu.Setb_r(0x07, &cpu.R.A)

	default:
		log.Panicf(PREFIX+" Invalid/Unknown instruction %X", Opcode)
	}
}

func (cpu *CPU) Dispatch(Opcode byte) {
	switch Opcode {
	case 0x00: //NOP
		cpu.NOP()
	case 0x01: //LD BC, nn
		cpu.LDn_nn(&cpu.R.B, &cpu.R.C)
	case 0x02: //LD (BC), A
		cpu.LDrr_r(&cpu.R.B, &cpu.R.C, &cpu.R.A)
	case 0x03: //INC BC
		cpu.Inc_rr(&cpu.R.B, &cpu.R.C)
	case 0x04: //INC B
		cpu.Inc_r(&cpu.R.B)
	case 0x05: //DEC B
		cpu.Dec_r(&cpu.R.B)
	case 0x06: //LD B,n
		cpu.LDrn(&cpu.R.B)
	case 0x07: //RLCA
		cpu.RLCA()
	case 0x08: //LD nn, SP
		cpu.LDnn_SP()
	case 0x09: //ADD HL,BC
		cpu.Addhl_rr(&cpu.R.B, &cpu.R.C)
	case 0x0A: //LD A, (BC)
		cpu.LDr_rr(&cpu.R.B, &cpu.R.C, &cpu.R.A)
	case 0x0B: //DEC BC
		cpu.Dec_rr(&cpu.R.B, &cpu.R.C)
	case 0x0C: //INC C
		cpu.Inc_r(&cpu.R.C)
	case 0x0D: //DEC C
		cpu.Dec_r(&cpu.R.C)
	case 0x0E: //LD C,n
		cpu.LDrn(&cpu.R.C)
	case 0x0F: //RRCA
		cpu.RRCA()
	case 0x10: //STOP
		cpu.Stop()
	case 0x11: //LD DE, nn
		cpu.LDn_nn(&cpu.R.D, &cpu.R.E)
	case 0x12: //LD (DE), A
		cpu.LDrr_r(&cpu.R.D, &cpu.R.E, &cpu.R.A)
	case 0x13: //INC DE
		cpu.Inc_rr(&cpu.R.D, &cpu.R.E)
	case 0x14: //INC D
		cpu.Inc_r(&cpu.R.D)
	case 0x15: //DEC D
		cpu.Dec_r(&cpu.R.D)
	case 0x16: //LD D,n
		cpu.LDrn(&cpu.R.D)
	case 0x17: //RLA
		cpu.RLA()
	case 0x18: //JR n
		cpu.JR_n()
	case 0x19: //ADD HL,DE
		cpu.Addhl_rr(&cpu.R.D, &cpu.R.E)
	case 0x1A: //LD A, (DE)
		cpu.LDr_rr(&cpu.R.D, &cpu.R.E, &cpu.R.A)
	case 0x1B: //DEC DE
		cpu.Dec_rr(&cpu.R.D, &cpu.R.E)
	case 0x1C: //INC E
		cpu.Inc_r(&cpu.R.E)
	case 0x1D: //DEC E
		cpu.Dec_r(&cpu.R.E)
	case 0x1E: //LD E,n
		cpu.LDrn(&cpu.R.E)
	case 0x1F: //RRA
		cpu.RRA()
	case 0x20: //JR NZ,n
		cpu.JRcc_nn(Z, false)
	case 0x21: //LD HL, nn
		cpu.LDn_nn(&cpu.R.H, &cpu.R.L)
	case 0x22: //LDI (HL), A
		cpu.LDIhl_r(&cpu.R.A)
	case 0x23: //INC HL
		cpu.Inc_rr(&cpu.R.H, &cpu.R.L)
	case 0x24: //INC H
		cpu.Inc_r(&cpu.R.H)
	case 0x25: //DEC H
		cpu.Dec_r(&cpu.R.H)
	case 0x26: //LD H,n
		cpu.LDrn(&cpu.R.H)
	case 0x27: //DAA
		cpu.Daa()
	case 0x28: //JR Z,n
		cpu.JRcc_nn(Z, true)
	case 0x29: //ADD HL,HL
		cpu.Addhl_rr(&cpu.R.H, &cpu.R.L)
	case 0x2A: //LDI A, (HL)
		cpu.LDIr_hl(&cpu.R.A)
	case 0x2B: //DEC HL
		cpu.Dec_rr(&cpu.R.H, &cpu.R.L)
	case 0x2C: //INC L
		cpu.Inc_r(&cpu.R.L)
	case 0x2D: //DEC L
		cpu.Dec_r(&cpu.R.L)
	case 0x2E: //LD L,n
		cpu.LDrn(&cpu.R.L)
	case 0x2F: //CPL
		cpu.CPL()
	case 0x30: //JR NZ,n
		cpu.JRcc_nn(C, false)
	case 0x31: //LD SP, nn
		cpu.LDSP_nn()
	case 0x32: //LDD (HL), A
		cpu.LDDhl_r(&cpu.R.A)
	case 0x33: //INC SP
		cpu.Inc_sp()
	case 0x34: //INC (HL)
		cpu.Inc_hl()
	case 0x35: //DEC (HL)
		cpu.Dec_hl()
	case 0x36: //LD (HL), n
		cpu.LDhl_n()
	case 0x37: //SCF
		cpu.SCF()
	case 0x38: //JR C,n
		cpu.JRcc_nn(C, true)
	case 0x39: //ADD HL,SP
		cpu.Addhl_sp()
	case 0x3A: //LDD A, (HL)
		cpu.LDDr_hl(&cpu.R.A)
	case 0x3B: //DEC SP
		cpu.Dec_sp()
	case 0x3C: //INC A
		cpu.Inc_r(&cpu.R.A)
	case 0x3D: //DEC A
		cpu.Dec_r(&cpu.R.A)
	case 0x3E: //LD A, n
		cpu.LDrn(&cpu.R.A)
	case 0x3F: //CCF
		cpu.CCF()
	case 0x40: //LD B, B
		cpu.LDrr(&cpu.R.B, &cpu.R.B)
	case 0x41: //LD B, C
		cpu.LDrr(&cpu.R.B, &cpu.R.C)
	case 0x42: //LD B, D
		cpu.LDrr(&cpu.R.B, &cpu.R.D)
	case 0x43: //LD B, E
		cpu.LDrr(&cpu.R.B, &cpu.R.E)
	case 0x44: //LD B, H
		cpu.LDrr(&cpu.R.B, &cpu.R.H)
	case 0x45: //LD B, L
		cpu.LDrr(&cpu.R.B, &cpu.R.L)
	case 0x46: //LD B, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.B)
	case 0x47: //LD B, A
		cpu.LDrr(&cpu.R.B, &cpu.R.A)
	case 0x48: //LD C, B
		cpu.LDrr(&cpu.R.C, &cpu.R.B)
	case 0x49: //LD C, C
		cpu.LDrr(&cpu.R.C, &cpu.R.C)
	case 0x4A: //LD C, D
		cpu.LDrr(&cpu.R.C, &cpu.R.D)
	case 0x4B: //LD C, E
		cpu.LDrr(&cpu.R.C, &cpu.R.E)
	case 0x4C: //LD C, H
		cpu.LDrr(&cpu.R.C, &cpu.R.H)
	case 0x4D: //LD C, L
		cpu.LDrr(&cpu.R.C, &cpu.R.L)
	case 0x4E: //LD C, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.C)
	case 0x4F: //LD C, A
		cpu.LDrr(&cpu.R.C, &cpu.R.A)
	case 0x50: //LD D, B
		cpu.LDrr(&cpu.R.D, &cpu.R.B)
	case 0x51: //LD D, C
		cpu.LDrr(&cpu.R.D, &cpu.R.C)
	case 0x52: //LD D, D
		cpu.LDrr(&cpu.R.D, &cpu.R.D)
	case 0x53: //LD D, E
		cpu.LDrr(&cpu.R.D, &cpu.R.E)
	case 0x54: //LD D, H
		cpu.LDrr(&cpu.R.D, &cpu.R.H)
	case 0x55: //LD D, L
		cpu.LDrr(&cpu.R.D, &cpu.R.L)
	case 0x56: //LD D, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.D)
	case 0x57: //LD D, A
		cpu.LDrr(&cpu.R.D, &cpu.R.A)
	case 0x58: //LD E, B
		cpu.LDrr(&cpu.R.E, &cpu.R.B)
	case 0x59: //LD E, C
		cpu.LDrr(&cpu.R.E, &cpu.R.C)
	case 0x5A: //LD E, D
		cpu.LDrr(&cpu.R.E, &cpu.R.D)
	case 0x5B: //LD E, E
		cpu.LDrr(&cpu.R.E, &cpu.R.E)
	case 0x5C: //LD E, H
		cpu.LDrr(&cpu.R.E, &cpu.R.H)
	case 0x5D: //LD E, L
		cpu.LDrr(&cpu.R.E, &cpu.R.L)
	case 0x5E: //LD E, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.E)
	case 0x5F: //LD E, A
		cpu.LDrr(&cpu.R.E, &cpu.R.A)
	case 0x60: //LD H, B
		cpu.LDrr(&cpu.R.H, &cpu.R.B)
	case 0x61: //LD H, C
		cpu.LDrr(&cpu.R.H, &cpu.R.C)
	case 0x62: //LD H, D
		cpu.LDrr(&cpu.R.H, &cpu.R.D)
	case 0x63: //LD H, E
		cpu.LDrr(&cpu.R.H, &cpu.R.E)
	case 0x64: //LD H, H
		cpu.LDrr(&cpu.R.H, &cpu.R.H)
	case 0x65: //LD H, L
		cpu.LDrr(&cpu.R.H, &cpu.R.L)
	case 0x66: //LD H, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.H)
	case 0x67: //LD H, A
		cpu.LDrr(&cpu.R.H, &cpu.R.A)
	case 0x68: //LD L, B
		cpu.LDrr(&cpu.R.L, &cpu.R.B)
	case 0x69: //LD L, C
		cpu.LDrr(&cpu.R.L, &cpu.R.C)
	case 0x6A: //LD L, D
		cpu.LDrr(&cpu.R.L, &cpu.R.D)
	case 0x6B: //LD L, E
		cpu.LDrr(&cpu.R.L, &cpu.R.E)
	case 0x6C: //LD L, H
		cpu.LDrr(&cpu.R.L, &cpu.R.H)
	case 0x6D: //LD L, L
		cpu.LDrr(&cpu.R.L, &cpu.R.L)
	case 0x6E: //LD L, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.L)
	case 0x6F: //LD L, A
		cpu.LDrr(&cpu.R.L, &cpu.R.A)
	case 0x70: //LD (HL), B
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.B)
	case 0x71: //LD (HL), C
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.C)
	case 0x72: //LD (HL), D
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.D)
	case 0x73: //LD (HL), E
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.E)
	case 0x74: //LD (HL), H
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.H)
	case 0x75: //LD (HL), L
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.L)
	case 0x76: //HALT
		cpu.HALT()
	case 0x77: //LD (HL), A
		cpu.LDrr_r(&cpu.R.H, &cpu.R.L, &cpu.R.A)
	case 0x78: //LD A, B
		cpu.LDrr(&cpu.R.A, &cpu.R.B)
	case 0x79: //LD A, C
		cpu.LDrr(&cpu.R.A, &cpu.R.C)
	case 0x7A: //LD A, D
		cpu.LDrr(&cpu.R.A, &cpu.R.D)
	case 0x7B: //LD A, E
		cpu.LDrr(&cpu.R.A, &cpu.R.E)
	case 0x7C: //LD A, H
		cpu.LDrr(&cpu.R.A, &cpu.R.H)
	case 0x7D: //LD A, L
		cpu.LDrr(&cpu.R.A, &cpu.R.L)
	case 0x7E: //LD A, (HL)
		cpu.LDr_rr(&cpu.R.H, &cpu.R.L, &cpu.R.A)
	case 0x7F: //LD A, A
		cpu.LDrr(&cpu.R.A, &cpu.R.A)
	case 0x80: //ADD A, B
		cpu.AddA_r(&cpu.R.B)
	case 0x81: //ADD A, C
		cpu.AddA_r(&cpu.R.C)
	case 0x82: //ADD A, D
		cpu.AddA_r(&cpu.R.D)
	case 0x83: //ADD A, E
		cpu.AddA_r(&cpu.R.E)
	case 0x84: //ADD A, H
		cpu.AddA_r(&cpu.R.H)
	case 0x85: //ADD A, L
		cpu.AddA_r(&cpu.R.L)
	case 0x86: //ADD A,(HL)
		cpu.AddA_hl()
	case 0x87: //ADD A, A
		cpu.AddA_r(&cpu.R.A)
	case 0x88: //ADC A, B
		cpu.AddCA_r(&cpu.R.B)
	case 0x89: //ADC A, C
		cpu.AddCA_r(&cpu.R.C)
	case 0x8A: //ADC A, D
		cpu.AddCA_r(&cpu.R.D)
	case 0x8B: //ADC A, E
		cpu.AddCA_r(&cpu.R.E)
	case 0x8C: //ADC A, H
		cpu.AddCA_r(&cpu.R.H)
	case 0x8D: //ADC A, L
		cpu.AddCA_r(&cpu.R.L)
	case 0x8E: //ADC A, (HL)
		cpu.AddCA_hl()
	case 0x8F: //ADC A, A
		cpu.AddCA_r(&cpu.R.A)
	case 0x90: // SUB A, B
		cpu.SubA_r(&cpu.R.B)
	case 0x91: // SUB A, C
		cpu.SubA_r(&cpu.R.C)
	case 0x92: // SUB A, D
		cpu.SubA_r(&cpu.R.D)
	case 0x93: // SUB A, E
		cpu.SubA_r(&cpu.R.E)
	case 0x94: // SUB A, H
		cpu.SubA_r(&cpu.R.H)
	case 0x95: // SUB A, L
		cpu.SubA_r(&cpu.R.L)
	case 0x96: //SUB A, (HL)
		cpu.SubA_hl()
	case 0x97: // SUB A, A
		cpu.SubA_r(&cpu.R.A)
	case 0x98: // SBC A, B
		cpu.SubAC_r(&cpu.R.B)
	case 0x99: // SBC A, C
		cpu.SubAC_r(&cpu.R.C)
	case 0x9A: // SBC A, D
		cpu.SubAC_r(&cpu.R.D)
	case 0x9B: // SBC A, E
		cpu.SubAC_r(&cpu.R.E)
	case 0x9C: // SBC A, H
		cpu.SubAC_r(&cpu.R.H)
	case 0x9D: // SBC A, L
		cpu.SubAC_r(&cpu.R.L)
	case 0x9E: //SBC A, (HL)
		cpu.SubAC_hl()
	case 0x9F: // SBC A, A
		cpu.SubAC_r(&cpu.R.A)
	case 0xA0: //AND A, B
		cpu.AndA_r(&cpu.R.B)
	case 0xA1: //AND A, C
		cpu.AndA_r(&cpu.R.C)
	case 0xA2: //AND A, D
		cpu.AndA_r(&cpu.R.D)
	case 0xA3: //AND A, E
		cpu.AndA_r(&cpu.R.E)
	case 0xA4: //AND A, H
		cpu.AndA_r(&cpu.R.H)
	case 0xA5: //AND A, L
		cpu.AndA_r(&cpu.R.L)
	case 0xA6: //AND A, (HL)
		cpu.AndA_hl()
	case 0xA7: //AND A, A
		cpu.AndA_r(&cpu.R.A)
	case 0xA8: //XOR A, B
		cpu.XorA_r(&cpu.R.B)
	case 0xA9: //XOR A, C
		cpu.XorA_r(&cpu.R.C)
	case 0xAA: //XOR A, D
		cpu.XorA_r(&cpu.R.D)
	case 0xAB: //XOR A, E
		cpu.XorA_r(&cpu.R.E)
	case 0xAC: //XOR A, H
		cpu.XorA_r(&cpu.R.H)
	case 0xAD: //XOR A, L
		cpu.XorA_r(&cpu.R.L)
	case 0xAE: //XOR A,(HL)
		cpu.XorA_hl()
	case 0xAF: //XOR A, A
		cpu.XorA_r(&cpu.R.A)
	case 0xB0: //OR A, B
		cpu.OrA_r(&cpu.R.B)
	case 0xB1: //OR A, C
		cpu.OrA_r(&cpu.R.C)
	case 0xB2: //OR A, D
		cpu.OrA_r(&cpu.R.D)
	case 0xB3: //OR A, E
		cpu.OrA_r(&cpu.R.E)
	case 0xB4: //OR A, H
		cpu.OrA_r(&cpu.R.H)
	case 0xB5: //OR A, L
		cpu.OrA_r(&cpu.R.L)
	case 0xB6: //OR A,(HL)
		cpu.OrA_hl()
	case 0xB7: //OR A, A
		cpu.OrA_r(&cpu.R.A)
	case 0xB8: //CP A, B
		cpu.CPA_r(&cpu.R.B)
	case 0xB9: //CP A, C
		cpu.CPA_r(&cpu.R.C)
	case 0xBA: //CP A, D
		cpu.CPA_r(&cpu.R.D)
	case 0xBB: //CP A, E
		cpu.CPA_r(&cpu.R.E)
	case 0xBC: //CP A, H
		cpu.CPA_r(&cpu.R.H)
	case 0xBD: //CP A, L
		cpu.CPA_r(&cpu.R.L)
	case 0xBE: //CP A, (HL)
		cpu.CPA_hl()
	case 0xBF: //CP A, A
		cpu.CPA_r(&cpu.R.A)
	case 0xC0: //RET NZ
		cpu.Retcc(Z, false)
	case 0xC1: //POP BC
		cpu.Pop_nn(&cpu.R.B, &cpu.R.C)
	case 0xC2: //JP NZ,nn
		cpu.JPcc_nn(Z, false)
	case 0xC3: //JP nn
		cpu.JP_nn()
	case 0xC4: //CALL NZ, nn
		cpu.Callcc_nn(Z, false)
	case 0xC5: //PUSH BC
		cpu.Push_nn(&cpu.R.B, &cpu.R.C)
	case 0xC6: //ADD A,#
		cpu.AddA_n()
	case 0xC7: //RST n
		cpu.Rst(0x00)
	case 0xC8: //RET Z
		cpu.Retcc(Z, true)
	case 0xC9: //RET
		cpu.Ret()
	case 0xCA: //JP Z,nn
		cpu.JPcc_nn(Z, true)
	case 0xCC: //CALL Z, nn
		cpu.Callcc_nn(Z, true)
	case 0xCD: //CALL nn
		cpu.Call_nn()
	case 0xCE: //ADC A, n
		cpu.AddCA_n()
	case 0xCF: //RST n
		cpu.Rst(0x08)
	case 0xD0: //RET NC
		cpu.Retcc(C, false)
	case 0xD1: //POP DE
		cpu.Pop_nn(&cpu.R.D, &cpu.R.E)
	case 0xD2: //JP NC,nn
		cpu.JPcc_nn(C, false)
	case 0xD4: //CALL NC, nn
		cpu.Callcc_nn(C, false)
	case 0xD5: //PUSH DE
		cpu.Push_nn(&cpu.R.D, &cpu.R.E)
	case 0xD6: //SUB A, n
		cpu.SubA_n()
	case 0xD7: //RST n
		cpu.Rst(0x10)
	case 0xD8: //RET C
		cpu.Retcc(C, true)
	case 0xD9: //RETI
		cpu.Ret_i()
	case 0xDA: //JP C,nn
		cpu.JPcc_nn(C, true)
	case 0xDC: //CALL C, nn
		cpu.Callcc_nn(C, true)
	case 0xDE: //SBC A, n
		cpu.SubAC_n()
	case 0xDF: //RST n
		cpu.Rst(0x18)
	case 0xE0: //LDH n, A
		cpu.LDHn_r(&cpu.R.A)
	case 0xE1: //POP HL
		cpu.Pop_nn(&cpu.R.H, &cpu.R.L)
	case 0xE2: //LD (C),A
		cpu.LDffplusc_r(&cpu.R.A)
	case 0xE5: //PUSH HL
		cpu.Push_nn(&cpu.R.H, &cpu.R.L)
	case 0xE6: //AND A, n
		cpu.AndA_n()
	case 0xE7: //RST n
		cpu.Rst(0x20)
	case 0xE8: //ADD SP,n
		cpu.Addsp_n()
	case 0xE9: //JP (HL)
		cpu.JP_hl()
	case 0xEA: //LD (nn), A
		cpu.LDnn_r(&cpu.R.A)
	case 0xEE: //XOR A, n
		cpu.XorA_n()
	case 0xEF: //RST n
		cpu.Rst(0x28)
	case 0xF0: //LDH r, n
		cpu.LDHr_n(&cpu.R.A)
	case 0xF1: //POP AF
		cpu.Pop_AF()
	case 0xF2: //LD A,(C)
		cpu.LDr_ffplusc(&cpu.R.A)
	case 0xF3: //DI
		cpu.DI()
	case 0xF5: //PUSH AF
		cpu.Push_nn(&cpu.R.A, &cpu.R.F)
	case 0xF6: //OR A, n
		cpu.OrA_n()
	case 0xF7: //RST n
		cpu.Rst(0x30)
	case 0xF8: //LDHL SP, n
		cpu.LDHLSP_n()
	case 0xF9: //LD SP, HL
		cpu.LDSP_hl()
	case 0xFA: //LD A, (nn)
		cpu.LDr_nn(&cpu.R.A)
	case 0xFB: //EI
		cpu.EI()
	case 0xFE: //CP A, n
		cpu.CPA_n()
	case 0xFF: //RST n
		cpu.Rst(0x38)
	default:
		log.Panicf(PREFIX+" Invalid/Unknown instruction %X", Opcode)
	}
}

func (cpu *CPU) Step() int {
	if err := cpu.Validate(); err != nil {
		log.Panicln(PREFIX, err)
	}
	cpu.LastInstrCycle.Reset()

	if !cpu.Halted {
		cpu.CheckForInterrupts()
		var Opcode byte = cpu.ReadByte(cpu.PC)
		var ok bool = false

		if Opcode == 0xCB {
			cpu.IncrementPC(1)
			Opcode = cpu.ReadByte(cpu.PC)
			cpu.CurrentInstruction, ok = cpu.DecodeCB(Opcode)
			if !ok {
				log.Panicf("No instruction found for opcode: %X\n%s", Opcode, cpu.String())
			}
			cpu.CurrentInstruction = cpu.Compile(cpu.CurrentInstruction)
			cpu.DispatchCB(Opcode)
		} else {
			cpu.CurrentInstruction, ok = cpu.Decode(Opcode)
			if !ok {
				log.Panicf("No instruction found for opcode: %X\n%s", Opcode, cpu.String())
			}
			cpu.CurrentInstruction = cpu.Compile(cpu.CurrentInstruction)
			cpu.Dispatch(Opcode)
		}

		//this is put in place to check whether the PC has been altered by an instruction. If it has then don't
		//do any incrementing
		if cpu.PCJumped == false {
			cpu.IncrementPC(cpu.CurrentInstruction.OperandsSize + 1)
		}

		cpu.PCJumped = false

		//calculate cycles
		cpu.LastInstrCycle.M += cpu.CurrentInstruction.Cycles
	} else {
		iflagnow := cpu.mmu.ReadByte(constants.INTERRUPT_FLAG_ADDR)

		//if IF flag has changed then the cpu should resume...
		if cpu.InterruptFlagBeforeHalt != iflagnow {
			cpu.Halted = false
		}

		//Halt consumes 1 cpu cycle
		cpu.LastInstrCycle.M = 1
	}

	return cpu.LastInstrCycle.M
}

func (cpu *CPU) CheckForInterrupts() bool {
	if cpu.InterruptsEnabled {
		var ie byte = cpu.mmu.ReadByte(constants.INTERRUPT_ENABLED_FLAG_ADDR)
		var iflag byte = cpu.mmu.ReadByte(constants.INTERRUPT_FLAG_ADDR)
		var interrupt byte = iflag & ie
		if interrupt != 0x00 {
			switch {
			case interrupt&constants.V_BLANK_IRQ == constants.V_BLANK_IRQ:
				cpu.mmu.WriteByte(constants.INTERRUPT_FLAG_ADDR, iflag&0xFE)
				cpu.pushWordToStack(cpu.PC)
				cpu.PC = types.Word(constants.V_BLANK_IR_ADDR)
				cpu.InterruptsEnabled = false
				return true
			case interrupt&constants.LCD_IRQ == constants.LCD_IRQ:
				cpu.mmu.WriteByte(constants.INTERRUPT_FLAG_ADDR, iflag&0xFD)
				cpu.pushWordToStack(cpu.PC)
				cpu.PC = types.Word(constants.LCD_IR_ADDR)
				cpu.InterruptsEnabled = false
				return true
			case interrupt&constants.TIMER_OVERFLOW_IRQ == constants.TIMER_OVERFLOW_IRQ:
				cpu.mmu.WriteByte(constants.INTERRUPT_FLAG_ADDR, iflag&0xFB)
				cpu.pushWordToStack(cpu.PC)
				cpu.PC = types.Word(constants.TIMER_OVERFLOW_IR_ADDR)
				cpu.InterruptsEnabled = false
				return true
			case interrupt&constants.JOYP_HILO_IRQ == constants.JOYP_HILO_IRQ:
				log.Println("JOYP!")
				cpu.mmu.WriteByte(constants.INTERRUPT_FLAG_ADDR, iflag&0xEF)
				cpu.pushWordToStack(cpu.PC)
				cpu.PC = types.Word(constants.JOYP_HILO_IR_ADDR)
				cpu.InterruptsEnabled = false
				return true
			default:
				log.Panicf("Unknown interrupt = %d", interrupt)
			}
		}
	}

	return false
}

//Checks to see if the CPU speed should change to double (CGB only)
func (cpu *CPU) SetCPUSpeed() {
	var speedPrepRegister byte = cpu.mmu.ReadByte(mmu.CGB_DOUBLE_SPEED_PREP_REG)
	if speedPrepRegister&0x01 == 0x01 {
		switch cpu.Speed {
		case 2:
			cpu.Speed = 1
			cpu.mmu.WriteByte(mmu.CGB_DOUBLE_SPEED_PREP_REG, 0x00)
		case 1:
			cpu.Speed = 2
			cpu.mmu.WriteByte(mmu.CGB_DOUBLE_SPEED_PREP_REG, 0x80)
		default:
			log.Panicf("Unsupported CPU speed (%d): this should not happen!", cpu.Speed)
		}
		log.Printf("CPU: Setting CPU speed to %dx speed", cpu.Speed)
	}
}

func (cpu *CPU) Compile(instruction Instruction) Instruction {
	switch instruction.OperandsSize {
	case 1:
		instruction.Operands[0] = cpu.mmu.ReadByte(cpu.PC + 1)
	case 2:
		instruction.Operands[0] = cpu.mmu.ReadByte(cpu.PC + 1)
		instruction.Operands[1] = cpu.mmu.ReadByte(cpu.PC + 2)
	}

	return instruction
}

func (cpu *CPU) Decode(instruction byte) (Instruction, bool) {
	ins, ok := Instructions[instruction]
	return ins, ok
}

func (cpu *CPU) DecodeCB(instruction byte) (Instruction, bool) {
	ins, ok := InstructionsCB[instruction]
	return ins, ok
}

func (cpu *CPU) pushByteToStack(b byte) {
	cpu.SP--
	cpu.WriteByte(cpu.SP, b)
}

func (cpu *CPU) pushWordToStack(word types.Word) {
	hs, ls := utils.SplitIntoBytes(uint16(word))
	cpu.pushByteToStack(hs)
	cpu.pushByteToStack(ls)
}

func (cpu *CPU) popByteFromStack() byte {
	var b byte = cpu.ReadByte(cpu.SP)
	cpu.SP++
	return b
}

func (cpu *CPU) popWordFromStack() types.Word {
	ls := cpu.popByteFromStack()
	hs := cpu.popByteFromStack()

	return types.Word(utils.JoinBytes(hs, ls))
}

func (cpu *CPU) ReadByte(addr types.Word) byte {
	if err := cpu.Validate(); err != nil {
		log.Panicln(PREFIX, err)
	}
	return cpu.mmu.ReadByte(addr)
}

func (cpu *CPU) WriteByte(addr types.Word, value byte) {
	if err := cpu.Validate(); err != nil {
		log.Panicln(PREFIX, err)
	}
	cpu.mmu.WriteByte(addr, value)
}

// INSTRUCTION HELPERS
//-----------------------------------------------------------------------

//General add function
//side effect is flags are set on the cpu accordingly
func (cpu *CPU) addBytes(a, b byte) byte {
	var calculation byte = a + b

	cpu.ResetFlag(N)

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if (calculation^b^a)&0x10 == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	if calculation < a {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	return calculation
}

//General add words function
//side effect is flags are set on the cpu accordingly
func (cpu *CPU) addWords(a, b types.Word) types.Word {
	var calculation types.Word = a + b

	if calculation < a {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if (calculation^b^a)&0x1000 == 0x1000 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	cpu.ResetFlag(N)

	return calculation
}

//General subtraction function
//side effect is flags are set on the cpu accordingly
func (cpu *CPU) subBytes(a, b byte) byte {

	cpu.SetFlag(N)

	if (int(a) & 0xF) < (int(b) & 0xF) {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	if (int(a) & 0xFF) < (int(b) & 0xFF) {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	var calculation byte = a - b
	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	return calculation
}

//General and function
//side effect is flags are set on the cpu accordingly
func (cpu *CPU) andBytes(a, b byte) byte {
	var calculation byte = a & b

	cpu.SetFlag(H)
	cpu.ResetFlag(N)
	cpu.ResetFlag(C)

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	return calculation
}

//General or function
//side effect is flags are set on the cpu accordingly
func (cpu *CPU) orBytes(a, b byte) byte {
	var calculation byte = a | b

	cpu.ResetFlag(H)
	cpu.ResetFlag(N)
	cpu.ResetFlag(C)

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	return calculation
}

//General xor function
//side effect is flags are set on the cpu accordingly
func (cpu *CPU) xorBytes(a, b byte) byte {
	var calculation byte = a ^ b

	cpu.ResetFlag(H)
	cpu.ResetFlag(N)
	cpu.ResetFlag(C)

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	return calculation
}

//General increment function
//Side effect is CPU flags get set accordingly
func (cpu *CPU) incByte(value byte) byte {
	var calculation byte = value + 0x01

	//N should be reset
	cpu.ResetFlag(N)

	//set zero flag
	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	//set half carry flag
	if (calculation^0x01^value)&0x10 == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	return calculation
}

//General decrement function
//Side effect is CPU flags get set accordingly
func (cpu *CPU) decByte(a byte) byte {
	var calculation byte = a - 1
	cpu.SetFlag(N)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	//Set half carry flag if needed
	if (calculation^0x01^a)&0x10 == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	return calculation
}

//General swap function
//Side effect is CPU flags get set accordingly
func (cpu *CPU) swapByte(a byte) byte {
	var calculation byte = utils.SwapNibbles(a)
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)
	cpu.ResetFlag(C)

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	return calculation
}

//General bit - test function
//Side effect is CPU flags get set accordingly
func (cpu *CPU) bitTest(bit byte, a byte) {
	if (a >> bit & 1) == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.SetFlag(H)
}

//General set bit function
func (cpu *CPU) setBit(bit byte, a byte) byte {
	return a | (1 << uint(bit))
}

//General reset bit function
func (cpu *CPU) resetBit(bit byte, a byte) byte {
	return a & ^(1 << uint(bit))
}

// INSTRUCTIONS START
//-----------------------------------------------------------------------

//NOP
//No operation
func (cpu *CPU) NOP() {

}

//HALT
//Halt CPU
func (cpu *CPU) HALT() {
	//get and store the state of the IF FLAG now so we know when it changes
	cpu.InterruptFlagBeforeHalt = cpu.mmu.ReadByte(constants.INTERRUPT_FLAG_ADDR)
	cpu.Halted = true
}

//STOP
func (cpu *CPU) Stop() {
	log.Println("CPU: Stopping...")
	//After a stop instruction is executed, CGB hardware should check to see if the CPU speed should change
	cpu.SetCPUSpeed()
}

//DI
//Disable interrupts
func (cpu *CPU) DI() {
	cpu.InterruptsEnabled = false
}

//EI
//Enable interrupts
func (cpu *CPU) EI() {
	cpu.InterruptsEnabled = true
}

//LD r,n
//Load value (n) from memory address in the PC into register (r) and increment PC by 1
func (cpu *CPU) LDrn(r *byte) {
	*r = cpu.CurrentInstruction.Operands[0]
}

//LD r,r
//Load value from register (r2) into register (r1)
func (cpu *CPU) LDrr(r1 *byte, r2 *byte) {
	*r1 = *r2
}

//LD rr, r
//Load value from register (r) into memory address located at register pair (RR)
func (cpu *CPU) LDrr_r(hs *byte, ls *byte, r *byte) {
	var RR types.Word = types.Word(utils.JoinBytes(*hs, *ls))
	cpu.WriteByte(RR, *r)
}

//LD r, rr
//Load value from memory address located in register pair (RR) into register (r)
func (cpu *CPU) LDr_rr(hs *byte, ls *byte, r *byte) {
	var RR types.Word = types.Word(utils.JoinBytes(*hs, *ls))
	*r = cpu.ReadByte(RR)
}

//LD nn,r
//Load value from register (r) and put it in memory address (nn) taken from the next 2 bytes of memory from the PC. Increment the PC by 2
func (cpu *CPU) LDnn_r(r *byte) {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]
	var resultAddr types.Word = types.Word(utils.JoinBytes(hs, ls))
	cpu.WriteByte(resultAddr, *r)
}

//LD r, nn
//Load the value in memory address defined from the next two bytes relative to the PC and store it in register (r). Increment the PC by 2
func (cpu *CPU) LDr_nn(r *byte) {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]
	var nn types.Word = types.Word(utils.JoinBytes(hs, ls))
	*r = cpu.ReadByte(nn)
}

//LD (HL),n
//Load the value (n) from the memory address in the PC and put it in the memory address designated by register pair (HL)
func (cpu *CPU) LDhl_n() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.WriteByte(HL, value)
}

//LD r,(C)
//Load the value from memory addressed 0xFF00 + value in register C. Store it in register (r)
func (cpu *CPU) LDr_ffplusc(r *byte) {
	var valueAddr types.Word = 0xFF00 + types.Word(cpu.R.C)
	*r = cpu.ReadByte(valueAddr)
}

//LD (C),r
//Load the value from register (r) and store it in memory addressed 0xFF00 + value in register C.
func (cpu *CPU) LDffplusc_r(r *byte) {
	var valueAddr types.Word = 0xFF00 + types.Word(cpu.R.C)
	cpu.WriteByte(valueAddr, *r)
}

//LDD r, (HL)
//Load the value from memory addressed in register pair (HL) and store it in register R. Decrement the HL registers
func (cpu *CPU) LDDr_hl(r *byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	*r = cpu.ReadByte(HL)

	HL -= 1

	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(HL))
}

//LDD (HL), r
//Load the value in register (r) and store in memory addressed in register pair (HL). Decrement the HL registers
func (cpu *CPU) LDDhl_r(r *byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	cpu.WriteByte(HL, *r)

	HL -= 1

	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(HL))
}

//LDI r, (HL)
//Load the value from memory addressed in register pair (HL) and store it in register R. Increment the HL registers
func (cpu *CPU) LDIr_hl(r *byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	*r = cpu.ReadByte(HL)

	HL += 1

	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(HL))
}

//LDI (HL), r
//Load the value in register (r) and store in memory addressed in register pair (HL). Increment the HL registers
func (cpu *CPU) LDIhl_r(r *byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	cpu.WriteByte(HL, *r)

	HL += 1

	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(HL))
}

//LDH n, r
func (cpu *CPU) LDHn_r(r *byte) {
	var n byte = cpu.CurrentInstruction.Operands[0]
	cpu.WriteByte(types.Word(0xFF00)+types.Word(n), *r)
}

//LDH r, n
//Load value (n) in register (r) and store it in memory address FF00+PC. Increment PC by 1
func (cpu *CPU) LDHr_n(r *byte) {
	var n byte = cpu.CurrentInstruction.Operands[0]
	*r = cpu.ReadByte((types.Word(0xFF00) + types.Word(n)))
}

//LD n, nn
func (cpu *CPU) LDn_nn(r1, r2 *byte) {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]

	//LS nibble first
	*r1 = hs
	*r2 = ls
}

//LD SP, nn
func (cpu *CPU) LDSP_nn() {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]

	cpu.SP = types.Word(utils.JoinBytes(hs, ls))
}

//LD nn, SP
func (cpu *CPU) LDnn_SP() {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]
	var addr types.Word = types.Word(utils.JoinBytes(hs, ls))

	cpu.WriteByte(addr+1, byte(cpu.SP&0xFF00>>8))
	cpu.WriteByte(addr, byte(cpu.SP&0x00FF))
}

//LD SP, rr
func (cpu *CPU) LDSP_hl() {
	cpu.SP = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
}

//LDHL SP, n
func (cpu *CPU) LDHLSP_n() {
	var n byte = cpu.CurrentInstruction.Operands[0]

	var HL types.Word

	if n > 127 {
		HL = cpu.SP - types.Word(-n)
	} else {
		HL = cpu.SP + types.Word(n)
	}

	var check types.Word = types.Word(cpu.SP ^ types.Word(n) ^ ((cpu.SP + types.Word(n)) & 0xffff))

	//set carry flag
	if (check & 0x100) == 0x100 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if (check & 0x10) == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	//reset flags
	cpu.ResetFlag(Z)
	cpu.ResetFlag(N)

	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(HL))
}

//PUSH nn
//Push register pair nn onto the stack and decrement the SP twice
func (cpu *CPU) Push_nn(r1, r2 *byte) {
	word := types.Word(utils.JoinBytes(*r1, *r2))
	cpu.pushWordToStack(word)
}

//POP nn
//Pop the stack twice onto register pair nn
func (cpu *CPU) Pop_nn(r1, r2 *byte) {
	*r1, *r2 = utils.SplitIntoBytes(uint16(cpu.popWordFromStack()))
}

//POP AF
//Pop the stack twice onto register pair AF
func (cpu *CPU) Pop_AF() {
	cpu.R.A, cpu.R.F = utils.SplitIntoBytes(uint16(cpu.popWordFromStack()))
	cpu.R.F &= 0xF0
}

//ADD A,r
//Add the value in register (r) to register A
func (cpu *CPU) AddA_r(r *byte) {
	cpu.R.A = cpu.addBytes(cpu.R.A, *r)
}

//ADD A,(HL)
//Add the value in memory addressed in register pair (HL) to register A
func (cpu *CPU) AddA_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)

	cpu.R.A = cpu.addBytes(cpu.R.A, value)
}

//ADD A,n
//Add the value in memory addressed PC to register A. Increment the PC by 1
func (cpu *CPU) AddA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.R.A = cpu.addBytes(cpu.R.A, value)
}

//ADDC A,r
func (cpu *CPU) AddCA_r(r *byte) {
	var carry int = 0
	if cpu.IsFlagSet(C) {
		carry = 1
	}

	cpu.ResetFlag(N)

	if ((int(cpu.R.A) & 0xF) + (int(*r) & 0xF) + carry) > 0xF {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	if ((int(cpu.R.A) & 0xFF) + (int(*r) & 0xFF) + carry) > 0xFF {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.R.A += *r + byte(carry)

	if cpu.R.A == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}
}

//ADDC A,(HL)
func (cpu *CPU) AddCA_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.ReadByte(HL)

	var carry int = 0
	if cpu.IsFlagSet(C) {
		carry = 1
	}

	cpu.ResetFlag(N)

	if ((int(cpu.R.A) & 0xF) + (int(hlValue) & 0xF) + carry) > 0xF {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	if ((int(cpu.R.A) & 0xFF) + (int(hlValue) & 0xFF) + carry) > 0xFF {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.R.A += hlValue + byte(carry)

	if cpu.R.A == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}
}

//ADDC A,n
func (cpu *CPU) AddCA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	var carry int = 0
	if cpu.IsFlagSet(C) {
		carry = 1
	}

	cpu.ResetFlag(N)

	if ((int(cpu.R.A) & 0xF) + (int(value) & 0xF) + carry) > 0xF {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	if ((int(cpu.R.A) & 0xFF) + (int(value) & 0xFF) + carry) > 0xFF {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.R.A += value + byte(carry)

	if cpu.R.A == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}
}

//SUB A,r
func (cpu *CPU) SubA_r(r *byte) {
	cpu.R.A = cpu.subBytes(cpu.R.A, *r)
}

//SUB A,hl
func (cpu *CPU) SubA_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)

	cpu.R.A = cpu.subBytes(cpu.R.A, value)
}

//SUB A,n
func (cpu *CPU) SubA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.R.A = cpu.subBytes(cpu.R.A, value)
}

//SBC A,r
func (cpu *CPU) SubAC_r(r *byte) {
	var un int = int(*r) & 0xff
	var tmpa int = int(cpu.R.A) & 0xff
	var ua int = int(cpu.R.A) & 0xff
	ua -= un
	if cpu.IsFlagSet(C) {
		ua -= 1
	}

	if ua < 0 {
		cpu.R.F = 0x50
	} else {
		cpu.R.F = 0x40
	}
	ua &= 0xff
	if ua == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if ((ua ^ un ^ tmpa) & 0x10) == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	cpu.R.A = byte(ua)
}

//SBC A, (HL)
func (cpu *CPU) SubAC_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)

	var un int = int(value) & 0xff
	var tmpa int = int(cpu.R.A) & 0xff
	var ua int = int(cpu.R.A) & 0xff

	ua -= un

	if cpu.IsFlagSet(C) {
		ua -= 1
	}

	if ua < 0 {
		cpu.R.F = 0x50
	} else {
		cpu.R.F = 0x40
	}

	ua &= 0xff
	if ua == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if ((ua ^ un ^ tmpa) & 0x10) == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	cpu.R.A = byte(ua)
}

//SBC A, n
func (cpu *CPU) SubAC_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	var un int = int(value) & 0xff
	var tmpa int = int(cpu.R.A) & 0xff
	var ua int = int(cpu.R.A) & 0xff

	ua -= un

	if cpu.IsFlagSet(C) {
		ua -= 1
	}

	if ua < 0 {
		cpu.R.F = 0x50
	} else {
		cpu.R.F = 0x40
	}

	ua &= 0xff
	if ua == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if ((ua ^ un ^ tmpa) & 0x10) == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	cpu.R.A = byte(ua)
}

//AND A, r
func (cpu *CPU) AndA_r(r *byte) {
	cpu.R.A = cpu.andBytes(cpu.R.A, *r)
}

//AND A, (HL)
func (cpu *CPU) AndA_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	cpu.R.A = cpu.andBytes(cpu.R.A, value)
}

//AND A, n
func (cpu *CPU) AndA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.R.A = cpu.andBytes(cpu.R.A, value)
}

//OR A, r
func (cpu *CPU) OrA_r(r *byte) {
	cpu.R.A = cpu.orBytes(cpu.R.A, *r)
}

//OR A, (HL)
func (cpu *CPU) OrA_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	cpu.R.A = cpu.orBytes(cpu.R.A, value)
}

//OR A, n
func (cpu *CPU) OrA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.R.A = cpu.orBytes(cpu.R.A, value)
}

//XOR A, r
func (cpu *CPU) XorA_r(r *byte) {
	cpu.R.A = cpu.xorBytes(cpu.R.A, *r)
}

//XOR A, (HL)
func (cpu *CPU) XorA_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	cpu.R.A = cpu.xorBytes(cpu.R.A, value)
}

//XOR A, n
func (cpu *CPU) XorA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.R.A = cpu.xorBytes(cpu.R.A, value)
}

//CP A, r
func (cpu *CPU) CPA_r(r *byte) {
	cpu.subBytes(cpu.R.A, *r)
}

//CP A, (HL)
func (cpu *CPU) CPA_hl() {
	var hlAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.ReadByte(hlAddr)
	cpu.subBytes(cpu.R.A, hlValue)
}

//CP A, n
func (cpu *CPU) CPA_n() {
	var value byte = cpu.CurrentInstruction.Operands[0]
	cpu.subBytes(cpu.R.A, value)
}

//INC r
func (cpu *CPU) Inc_r(r *byte) {
	*r = cpu.incByte(*r)
}

//INC (HL)
func (cpu *CPU) Inc_hl() {
	var hlAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.ReadByte(hlAddr)
	var result byte = cpu.incByte(hlValue)
	cpu.WriteByte(hlAddr, result)
}

//DEC r
func (cpu *CPU) Dec_r(r *byte) {
	*r = cpu.decByte(*r)
}

//DEC (HL)
func (cpu *CPU) Dec_hl() {
	var hlAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.ReadByte(hlAddr)
	var result byte = cpu.decByte(hlValue)
	cpu.WriteByte(hlAddr, result)
}

// --------------- 16 bit operations ---------------
//ADD HL,rr
func (cpu *CPU) Addhl_rr(r1, r2 *byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var RR types.Word = types.Word(utils.JoinBytes(*r1, *r2))
	var result types.Word = cpu.addWords(HL, RR)
	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(result))
}

//ADD HL,SP
func (cpu *CPU) Addhl_sp() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var result types.Word = cpu.addWords(HL, cpu.SP)
	cpu.R.H, cpu.R.L = utils.SplitIntoBytes(uint16(result))
}

//ADD SP,n
func (cpu *CPU) Addsp_n() {
	var n byte = cpu.CurrentInstruction.Operands[0]

	var calculation types.Word

	// immediate value is signed
	if n > 127 {
		calculation = cpu.SP - types.Word(-n)
	} else {
		calculation = cpu.SP + types.Word(n)
	}

	var check types.Word = types.Word(cpu.SP ^ types.Word(n) ^ ((cpu.SP + types.Word(n)) & 0xffff))

	//set carry flag
	if (check & 0x100) == 0x100 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if (check & 0x10) == 0x10 {
		cpu.SetFlag(H)
	} else {
		cpu.ResetFlag(H)
	}

	//reset flags
	cpu.ResetFlag(Z)
	cpu.ResetFlag(N)

	cpu.SP = calculation
}

//INC rr
func (cpu *CPU) Inc_rr(r1, r2 *byte) {
	var RR types.Word = types.Word(utils.JoinBytes(*r1, *r2))
	RR += 1
	*r1, *r2 = utils.SplitIntoBytes(uint16(RR))
}

//INC SP
func (cpu *CPU) Inc_sp() {
	cpu.SP = (cpu.SP + 1) & 0xFFFF
}

//DEC rr
func (cpu *CPU) Dec_rr(r1, r2 *byte) {
	var RR types.Word = types.Word(utils.JoinBytes(*r1, *r2))
	RR -= 1
	*r1, *r2 = utils.SplitIntoBytes(uint16(RR))
}

//DEC SP
func (cpu *CPU) Dec_sp() {
	cpu.SP = (cpu.SP - 1) & 0xFFFF
}

//CPL
func (cpu *CPU) CPL() {
	cpu.R.A = ^cpu.R.A
	cpu.SetFlag(N)
	cpu.SetFlag(H)
}

//CCF
func (cpu *CPU) CCF() {
	if cpu.IsFlagSet(C) {
		cpu.ResetFlag(C)
	} else {
		cpu.SetFlag(C)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)
}

//SCF
func (cpu *CPU) SCF() {
	cpu.SetFlag(C)

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)
}

//DAA - this instruction was a complete PITA to implement
//thankfully DParrot from here http://forums.nesdev.com/viewtopic.php?t=9088
//provided a correct solution that passes the blargg tests
func (cpu *CPU) Daa() {
	var a types.Word = types.Word(cpu.R.A)

	if cpu.IsFlagSet(N) == false {
		if cpu.IsFlagSet(H) || a&0x0F > 9 {
			a += 0x06
		}

		if cpu.IsFlagSet(C) || a > 0x9F {
			a += 0x60
		}
	} else {
		if cpu.IsFlagSet(H) {
			a = (a - 6) & 0xFF
		}

		if cpu.IsFlagSet(C) {
			a -= 0x60
		}
	}

	cpu.ResetFlag(H)

	if a&0x100 == 0x100 {
		cpu.SetFlag(C)
	}

	a &= 0xFF

	if a == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.R.A = byte(a)
}

//SWAP r
func (cpu *CPU) Swap_r(r *byte) {
	*r = cpu.swapByte(*r)
}

//SWAP (HL)
func (cpu *CPU) Swap_hl() {
	var hlAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.ReadByte(hlAddr)
	var result = cpu.swapByte(hlValue)
	cpu.WriteByte(hlAddr, result)
}

//RLCA
func (cpu *CPU) RLCA() {
	var bit7 bool = false

	if cpu.R.A&0x80 == 0x80 {
		bit7 = true
	}

	var calculation byte = cpu.R.A << 1

	if bit7 {
		cpu.SetFlag(C)
		calculation ^= 0x01
	} else {
		cpu.ResetFlag(C)
	}

	//reset flags
	cpu.ResetFlag(Z)
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	cpu.R.A = calculation
}

//RLA
func (cpu *CPU) RLA() {
	var bit7 bool = false
	var calculation byte = cpu.R.A

	if calculation&0x80 == 0x80 {
		bit7 = true
	}

	calculation = calculation << 1

	if cpu.IsFlagSet(C) {
		calculation ^= 0x01
	}

	if bit7 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.ResetFlag(Z)
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	cpu.R.A = calculation
}

//RLC r
func (cpu *CPU) Rlc_r(r *byte) {
	var bit7 bool = false

	if *r&0x80 == 0x80 {
		bit7 = true
	}

	var calculation byte = *r << 1

	if bit7 {
		cpu.SetFlag(C)
		calculation ^= 0x01
	} else {
		cpu.ResetFlag(C)
	}

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)

	}

	//reset flags
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	*r = calculation
}

//RLC (HL)
func (cpu *CPU) Rlc_hl() {
	var hlAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.mmu.ReadByte(hlAddr)

	var bit7 bool = false

	if hlValue&0x80 == 0x80 {
		bit7 = true
	}

	var calculation byte = hlValue << 1

	if bit7 {
		cpu.SetFlag(C)
		calculation ^= 0x01
	} else {
		cpu.ResetFlag(C)
	}

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	//reset flags
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	cpu.mmu.WriteByte(hlAddr, calculation)
}

//RL r
func (cpu *CPU) Rl_r(r *byte) {
	var bit7 bool = false
	var calculation byte = *r

	if calculation&0x80 == 0x80 {
		bit7 = true
	}

	calculation = calculation << 1

	if cpu.IsFlagSet(C) {
		calculation ^= 0x01
	}

	if bit7 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	*r = calculation
}

//RL (HL)
func (cpu *CPU) Rl_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)

	var bit7 bool = false
	var calculation byte = value

	if calculation&0x80 == 0x80 {
		bit7 = true
	}

	calculation = calculation << 1

	if cpu.IsFlagSet(C) {
		calculation ^= 0x01
	}

	if bit7 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)
	cpu.WriteByte(HL, calculation)
}

//RRCA
func (cpu *CPU) RRCA() {
	var bit0 bool = false

	if cpu.R.A&0x01 == 0x01 {
		bit0 = true
	}

	var calculation byte = cpu.R.A >> 1

	if bit0 {
		cpu.SetFlag(C)
		calculation ^= 0x80
	} else {
		cpu.ResetFlag(C)
	}

	//reset flags
	cpu.ResetFlag(Z)
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	cpu.R.A = calculation
}

//RRA
func (cpu *CPU) RRA() {
	var bit0 bool = false
	var calculation byte = cpu.R.A

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = calculation >> 1

	if cpu.IsFlagSet(C) {
		calculation ^= 0x80
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.ResetFlag(Z)
	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	cpu.R.A = calculation
}

//RRC r
func (cpu *CPU) Rrc_r(r *byte) {
	var bit0 bool = false

	if *r&0x01 == 0x01 {
		bit0 = true
	}

	var calculation byte = *r >> 1

	if bit0 {
		cpu.SetFlag(C)
		calculation ^= 0x80
	} else {
		cpu.ResetFlag(C)
	}

	//reset flags
	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	*r = calculation
}

//RRC (HL)
func (cpu *CPU) Rrc_hl() {
	var hlAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var hlValue byte = cpu.mmu.ReadByte(hlAddr)
	var bit0 bool = false

	if hlValue&0x01 == 0x01 {
		bit0 = true
	}

	var calculation byte = hlValue >> 1

	if bit0 {
		cpu.SetFlag(C)
		calculation ^= 0x80
	} else {
		cpu.ResetFlag(C)
	}

	//reset flags
	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	cpu.mmu.WriteByte(hlAddr, calculation)
}

//RR r
func (cpu *CPU) Rr_r(r *byte) {
	var bit0 bool = false
	var calculation byte = *r

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = calculation >> 1

	if cpu.IsFlagSet(C) {
		calculation ^= 0x80
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	*r = calculation
}

//RR (HL)
func (cpu *CPU) Rr_hl() {
	var HLAddr types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HLAddr)

	var bit0 bool = false
	var calculation byte = value

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = calculation >> 1

	if cpu.IsFlagSet(C) {
		calculation ^= 0x80
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	if calculation == 0x00 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)
	cpu.WriteByte(HLAddr, calculation)
}

//SLA r
func (cpu *CPU) Sla_r(r *byte) {
	var bit7 bool = false
	var calculation byte = *r

	if calculation&0x80 == 0x80 {
		bit7 = true
	}

	calculation = calculation << 1

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if bit7 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	*r = calculation
}

//SLA (HL)
func (cpu *CPU) Sla_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	var calculation byte = value
	var bit7 bool = false

	if calculation&0x80 == 0x80 {
		bit7 = true
	}

	calculation = calculation << 1

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if bit7 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.WriteByte(HL, calculation)
}

//SRA r
func (cpu *CPU) Sra_r(r *byte) {
	var bit0 bool = false
	var calculation byte = *r

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = (calculation >> 1) | (calculation & 0x80)

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	*r = calculation
}

//SRA (HL)
func (cpu *CPU) Sra_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	var calculation byte = value
	var bit0 bool = false

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = (calculation >> 1) | (calculation & 0x80)

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.WriteByte(HL, calculation)
}

//SRL r
func (cpu *CPU) Srl_r(r *byte) {
	var bit0 bool = false
	var calculation byte = *r

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = calculation >> 1

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	*r = calculation
}

//SRL (HL)
func (cpu *CPU) Srl_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	var calculation byte = value
	var bit0 bool = false

	if calculation&0x01 == 0x01 {
		bit0 = true
	}

	calculation = calculation >> 1

	cpu.ResetFlag(N)
	cpu.ResetFlag(H)

	if calculation == 0 {
		cpu.SetFlag(Z)
	} else {
		cpu.ResetFlag(Z)
	}

	if bit0 {
		cpu.SetFlag(C)
	} else {
		cpu.ResetFlag(C)
	}

	cpu.WriteByte(HL, calculation)
}

//BIT b, r
func (cpu *CPU) Bitb_r(b byte, r *byte) {
	cpu.bitTest(b, *r)
}

//BIT b,(HL)
func (cpu *CPU) Bitb_hl(b byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var value byte = cpu.ReadByte(HL)
	cpu.bitTest(b, value)
}

// SET b, r
func (cpu *CPU) Setb_r(b byte, r *byte) {
	*r = cpu.setBit(b, *r)
}

// SET b, (HL)
func (cpu *CPU) Setb_hl(b byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var HLValue byte = cpu.ReadByte(HL)
	var result byte = cpu.setBit(b, HLValue)

	cpu.WriteByte(HL, result)
}

// RES b, r
func (cpu *CPU) Resb_r(b byte, r *byte) {
	*r = cpu.resetBit(b, *r)
}

// RES b, (HL)
func (cpu *CPU) Resb_hl(b byte) {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	var HLValue byte = cpu.ReadByte(HL)
	var result byte = cpu.resetBit(b, HLValue)

	cpu.WriteByte(HL, result)
}

//JP nn
func (cpu *CPU) JP_nn() {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]
	cpu.PC = types.Word(utils.JoinBytes(hs, ls))
	cpu.PCJumped = true
}

//JP (HL)
func (cpu *CPU) JP_hl() {
	var HL types.Word = types.Word(utils.JoinBytes(cpu.R.H, cpu.R.L))
	cpu.PC = HL
	cpu.PCJumped = true
}

//JP cc, nn
func (cpu *CPU) JPcc_nn(flag int, jumpWhen bool) {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]

	if cpu.IsFlagSet(flag) == jumpWhen {
		cpu.PCJumped = true
		cpu.PC = types.Word(utils.JoinBytes(hs, ls))
		cpu.CurrentInstruction.Cycles = 4
	} else {
		cpu.CurrentInstruction.Cycles = 3
	}
}

//JR n
func (cpu *CPU) JR_n() {
	var n byte = cpu.CurrentInstruction.Operands[0]
	if n != 0x00 {
		cpu.PC += types.Word(cpu.CurrentInstruction.OperandsSize + 1)

		if n > 127 {
			cpu.PC -= types.Word(-n)
		} else {
			cpu.PC += types.Word(n)
		}

		cpu.PCJumped = true
	}
}

//JR cc, nn
func (cpu *CPU) JRcc_nn(flag int, jumpWhen bool) {
	var n byte = cpu.CurrentInstruction.Operands[0]

	if cpu.IsFlagSet(flag) == jumpWhen {
		if n != 0x00 {
			cpu.PC += types.Word(cpu.CurrentInstruction.OperandsSize + 1)

			if n > 127 {
				cpu.PC -= types.Word(-n)
			} else {
				cpu.PC += types.Word(n)
			}

			cpu.PCJumped = true
		}
		cpu.CurrentInstruction.Cycles = 3
	} else {
		cpu.CurrentInstruction.Cycles = 2
	}
}

// CALL nn
//Push address of next instruction onto stack and then jump to address nn
func (cpu *CPU) Call_nn() {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]
	var nextInstr types.Word = cpu.PC + 3
	cpu.pushWordToStack(nextInstr)
	cpu.PC = types.Word(utils.JoinBytes(hs, ls))
	cpu.PCJumped = true
}

// CALL cc,nn
func (cpu *CPU) Callcc_nn(flag int, callWhen bool) {
	var ls byte = cpu.CurrentInstruction.Operands[0]
	var hs byte = cpu.CurrentInstruction.Operands[1]
	var nextInstr types.Word = cpu.PC + 3

	if cpu.IsFlagSet(flag) == callWhen {
		cpu.pushWordToStack(nextInstr)
		cpu.PC = types.Word(utils.JoinBytes(hs, ls))
		cpu.PCJumped = true
		cpu.CurrentInstruction.Cycles = 6
	} else {
		cpu.CurrentInstruction.Cycles = 3
	}
}

// RET
func (cpu *CPU) Ret() {
	cpu.PC = cpu.popWordFromStack()
	cpu.PCJumped = true
}

// RET cc
func (cpu *CPU) Retcc(flag int, returnWhen bool) {
	if cpu.IsFlagSet(flag) == returnWhen {
		cpu.PC = cpu.popWordFromStack()
		cpu.PCJumped = true
		cpu.CurrentInstruction.Cycles = 5
	} else {
		cpu.CurrentInstruction.Cycles = 2
	}
}

// RETI
func (cpu *CPU) Ret_i() {
	cpu.PC = cpu.popWordFromStack()
	cpu.InterruptsEnabled = true
	cpu.PCJumped = true
}

// RST n
func (cpu *CPU) Rst(n byte) {
	cpu.pushWordToStack(cpu.PC + 1)
	cpu.PC = types.Word(n)
	cpu.PCJumped = true
}

//-----------------------------------------------------------------------
//INSTRUCTIONS END
