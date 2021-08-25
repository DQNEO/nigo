// Encode assembly statements into machine code
package main

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

const REX_W byte = 0x48

const ModRegiToRegi uint8 = 0b11
const ModIndirectionWithNoDisplacement uint8 = 0b00
const ModIndirectionWithDisplacement8 uint8 = 0b01
const ModIndirectionWithDisplacement32 uint8 = 0b10

const RM_SPECIAL_101 uint8 = 0b101 // none? rip?

// ModR/M
// https://wiki.osdev.org/X86-64_Instruction_Encoding#ModR.2FM
// The ModR/M byte encodes a register or an opcode extension, and a register or a memory address. It has the following fields:
//
//    7   6   5   4   3   2   1   0
//  +---+---+---+---+---+---+---+---+
//  |  mod  |    reg    |     rm    |
//  +---+---+---+---+---+---+---+---+
func composeModRM(mod byte, reg byte, rm byte) byte {
	return mod * 64 + reg * 8 + rm
}

// The registers are encoded using the 4-bit values in the X.Reg column of the following table.
// X.Reg is in binary.
func regBits(reg string) uint8 {
	var x_reg uint8
	switch reg {
	case "ax": x_reg = 0b0000
	case "cx": x_reg = 0b0001
	case "dx": x_reg = 0b0010
	case "bx": x_reg = 0b0011
	case "sp": x_reg = 0b0100
	case "bp": x_reg = 0b0101 // NONE
	case "si": x_reg = 0b0110
	case "di": x_reg = 0b0111
	default:
		panic("TBI: unexpected register " + reg)
	}
	return x_reg
}

// SIB
// https://wiki.osdev.org/X86-64_Instruction_Encoding#SIB

const SibIndexNone uint8 = 0b100
const SibBaseRSP uint8 = 0b100

//     7                           0
//  +---+---+---+---+---+---+---+---+
//  | scale |   index   |    base   |
//  +---+---+---+---+---+---+---+---+
func composeSIB(scale byte, index byte, base byte) byte {
	return scale* 32 + index* 8 + base
}


type Instruction struct {
	startAddr uintptr
	raw       *statement
	code      []byte
}

func encode(s *statement) *Instruction {
	//debugf("stmt=%#v\n", s)
	var r []byte
	var instr = &Instruction{
		startAddr:   currentTextAddr,
		raw:         s,
	}
	if s.labelSymbol != "" {
		allSymbols[s.labelSymbol].address = currentTextAddr
	}
	//fmt.Printf("[translator] %s (%d ops) => ", s.keySymbol, len(s.operands))
	switch s.keySymbol {
	case "nop":
		r = []byte{0x90}
	case "jmp":
		// EB cb
		r = []byte{0xeb, 0}
		target_symbol := s.operands[0].ifc.(*symbolExpr).name
		unresolvedCodeSymbols[currentTextAddr+1] = &addrToReplace{
			nextInstrAddr: currentTextAddr + uintptr(len(r)),
			symbolUsed:    target_symbol,
		}
	case "callq","call":
		r =  []byte{0xe8, 0, 0, 0, 0}
		target_symbol := s.operands[0].ifc.(*symbolExpr).name
		unresolvedCodeSymbols[currentTextAddr+1] = &addrToReplace{
			nextInstrAddr: currentTextAddr + uintptr(len(r)),
			symbolUsed:    target_symbol,
		}
	case "leaq":
		op1, op2 := s.operands[0], s.operands[1]
		switch op1dtype := op1.ifc.(type) {
		case *indirection: // leaq foo(%regi), %regi
			regi := op1dtype.regi
			op2regi := op2.ifc.(*register)
			if regi.name == "rip" {
				// RIP relative addressing
				panic(fmt.Sprintf("TBI:%v", op1))
			} else if regi.name == "rsp" {
				var opcode uint8 = 0x8d
				var mod uint8 = 0b01 // indirection with 8bit displacement
				rm := regBits("sp")
				reg := op2regi.toBits()
				modRM := composeModRM(mod, reg, rm)
				sib := composeSIB(0b00, SibIndexNone, SibBaseRSP)
				num := op1dtype.expr.(*numberExpr).val
				displacement, err := strconv.ParseInt(num, 0, 8)
				if err != nil {
					panic(err)
				}
				r = []byte{REX_W, opcode, modRM, sib, uint8(displacement)}
			}
		default:
			panic(fmt.Sprintf("TBI:%v", op1))
		}
	//case "movl":
	//	op1, op2 := s.operands[0], s.operands[1]
	//	assert(op1.typ == "$number", "op1 type should be $number")
	//	//op1Regi, IsOp1Regi := op1.ifc.(*register)
	//	op2Regi := op2.ifc.(*register)
	//
	//	//debugf("op1,op2=%s,%s  ", op1, op2)
	//	intNum, err := strconv.ParseInt(op1.string, 0, 32)
	//	if err != nil {
	//		panic(err)
	//	}
	//	var num int32 = int32(intNum)
	//	bytesNum := (*[4]byte)(unsafe.Pointer(&num))
	//	var opcode byte
	//	regFieldN := op2Regi.toBits()
	//	opcode = 0xb8 + regFieldN
	//	tmp := []byte{opcode}
	//	r = append(tmp, (bytesNum[:])...)
	case "movq":
		op2 := s.operands[1]
		//		assert(op1.typ == "$number", "op1 type should be $number")
		//assert(op2.typ == "register", "op2 type should be register")
		switch op1 := s.operands[0].ifc.(type) {
		case *immediate: // movq $123, %regi
			intNum, err := strconv.ParseInt(op1.expr, 0, 32)
			if err != nil {
				panic(err)
			}
			var num int32 = int32(intNum)
			bytesNum := (*[4]byte)(unsafe.Pointer(&num))
			var opcode uint8 = 0xc7
			var modRM uint8 = 0b11000000+ op2.ifc.(*register).toBits()
			r = []byte{REX_W, opcode, modRM}
			r = append(r, bytesNum[:]...)
		case *register:
			var opcode uint8 = 0x89
			switch op2dtype := op2.ifc.(type) {
			case *register:
				mod := ModRegiToRegi
				reg := op1.toBits() // src
				op2Regi := op2.ifc.(*register)
				rm :=  op2Regi.toBits()  // dst
				modRM := composeModRM(mod, reg, rm)
				r = []byte{REX_W,opcode,modRM}
			case *indirection:
				if op2dtype.isRipRelative() {
					switch expr := op2dtype.expr.(type) {
					case *binaryExpr:
						// REX.W 89 /r (MOV r/m64 r64, MR)
						// "movq %rbx, runtime.__argv__+8(%rip)"
						mod := ModIndirectionWithNoDisplacement
						reg := op1.toBits() // src
						rm := RM_SPECIAL_101 // RIP
						modRM := composeModRM(mod, reg, rm)
						r = []byte{REX_W,opcode,modRM}

						symbol := expr.left.(*symbolExpr).name

						// @TODO shouud use expr.right.(*numberExpr).val as an offset
						ru := &relaTextUser{
							addr: currentTextAddr + uintptr(len(r)),
							uses: symbol,
						}

						r = append(r,  0,0,0,0)

						relaTextUsers = append(relaTextUsers, ru)
					default:
						panic("TBI:" + string(s.raw))
					}
				} else {
					// movq %rax, 32(%rsp)
					mod := ModIndirectionWithDisplacement8
					reg := op1.toBits() // src
					rm := regBits("sp")
					modRM := composeModRM(mod, reg, rm)

					sib := composeSIB(0b00, SibIndexNone, SibBaseRSP)
					num := op2dtype.expr.(*numberExpr).val
					displacement, err := strconv.ParseInt(num, 0, 8)
					if err != nil {
						panic(err)
					}
					r = []byte{REX_W, opcode, modRM, sib, uint8(displacement)}
				}
			default:
				panic("unexpected op2.typ:")
			}
		case *indirection: // "movq foo(%regi), X", "movq (%regi), X"
			op1regi := op1.regi
			op2regi := op2.ifc.(*register)
			if op1regi.name == "rip" {
				// RIP relative addressing
				var opcode uint8 = 0x8b
				reg := op2regi.toBits()
				mod := ModIndirectionWithNoDisplacement
				modRM := composeModRM(mod, reg, 0b101)
				r = []byte{REX_W, opcode, modRM}

				symbol := op1.expr.(*symbolExpr).name
				ru := &relaTextUser{
					addr: currentTextAddr + uintptr(len(r)),
					uses: symbol,
				}

				r = append(r,  0,0,0,0)

				relaTextUsers = append(relaTextUsers, ru)
			} else if op1regi.name  == "rsp" {
				var opcode uint8 = 0x8b
				var mod uint8 = 0b000 // indirection
				var rm = regBits("sp")
				reg := op2regi.toBits()
				modRM := composeModRM(mod, reg, rm)
				sib := composeSIB(0b00, SibIndexNone, SibBaseRSP)
				r = []byte{REX_W, opcode, modRM, sib}
			} else {
				var opcode uint8 = 0x8b
				reg := op2regi.toBits()
				mod := ModIndirectionWithNoDisplacement
				rm := op1regi.toBits()
				modRM := composeModRM(mod, reg, rm)
				r = []byte{REX_W, opcode, modRM}
			}
		default:
			panic(fmt.Sprintf("TBI:%v", op1))
		}
	case "addl":
		_, op2 := s.operands[0], s.operands[1]
		var opcode uint8 = 0x01
		regFieldN := op2.ifc.(*register).toBits()
		var modRM uint8 = 0b11000000+ regFieldN
		r = []byte{opcode, modRM}
	case "addq":
		r = []byte{REX_W, 0x01, 0xc7} // REX.W, ADD, ModR/M
	case "ret", "retq":
		r = []byte{0xc3}
	case "syscall":
		r = []byte{0x0f, 0x05}
	case ".text":
		//fmt.Printf(" skip\n")
	case "imulq":
	case "subq":
	case "pushq":
	case "popq":
	default:
		if strings.HasPrefix(s.keySymbol , ".") {
			//fmt.Printf(" (directive)\n")
		} else {
			if s.labelSymbol != "" && s.keySymbol == "" {
				//fmt.Printf(" (label)\n")
				return instr
			} else {
				panic("Unexpected key symbols:" + s.keySymbol)
			}
		}
		//return nil
	}

	//fmt.Printf("=>  %#x\n", r)
	instr.code = r
	return instr
}

func encodeData(s *statement) []byte {
	if s.labelSymbol != "" {
		allSymbols[s.labelSymbol].address = currentDataAddr
	}

	switch s.keySymbol {
	case ".quad":
		op := s.operands[0]
		//debugf(".quad type=%T\n", op.ifc)
		switch opDtype := op.ifc.(type) {
		case *numberExpr:
			rawVal := opDtype.val
			var i int64
			if strings.HasPrefix(rawVal, "0x") {
				var err error
				i, err = strconv.ParseInt(rawVal, 0, 0)
				if err != nil {
					panic(err)
				}
			} else {
				// TBI
			}
			buf := (*[8]byte)(unsafe.Pointer(&i))
			return buf[:]
		case *symbolExpr:
			ru := &relaDataUser{
				addr: currentDataAddr,
				uses: opDtype.name,
			}
			relaDataUsers = append(relaDataUsers, ru)
			return make([]byte, 8)
		default:
			panic("Unexpected op.typ:")
		}
	case "": // label
		//panic("empty keySymbol:" + s.labelSymbol)
	default:
		panic("TBI:"+ s.keySymbol)
	}
	return nil
}
