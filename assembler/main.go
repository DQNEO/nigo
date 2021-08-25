package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

func debugf(s string, a...interface{}) {
	fmt.Fprintf(os.Stderr, s, a...)
}

func buildSectionBodies(hasRelaText, hasRelaData, hasSymbols bool) []*section {
	var sections = []*section{
		s_text, // .text
		s_data, // .data
		s_bss,  // .bss (no contents)
	}

	if hasSymbols {
		sections = append(sections, s_symtab, s_strtab)
	}

	if hasRelaText {
		sections = append(sections, s_rela_text)
	}

	if hasRelaData {
		sections = append(sections, s_rela_data)
	}

	sections = append(sections,s_shstrtab)
	return sections
}

type section struct {
	sh_name    string
	shndx      int
	header     *ElfSectionHeader
	numZeroPad uintptr
	zeros      []uint8
	contents   []uint8
}

// .symtab
var symbolTable = []*ElfSym{
	&ElfSym{}, // NULL entry
}

func prepareSHTEntries(hasRelaText, hasRelaData, hasSymbols bool) []*section {

	r := []*section{
		s_null,      // NULL
		s_text,      // .text
	}

	if hasRelaText {
		r = append(r, s_rela_text)
	}

	r = append(r, s_data)

	if hasRelaData {
		r = append(r, s_rela_data)
	}
	r = append(r, s_bss)

	if hasSymbols {
		r = append(r, s_symtab, s_strtab)
	}
	r = append(r, s_shstrtab)
	for i, s := range r {
		s.shndx = i
	}
	return r
}

var s_null = &section{
	header: &ElfSectionHeader{},
}

var s_text = &section{
	sh_name: ".text",
	header: &ElfSectionHeader{
		sh_type:      0x01, // SHT_PROGBITS
		sh_flags:     0x06, // SHF_ALLOC|SHF_EXECINSTR
		sh_addr:      0,
		sh_link:      0,
		sh_info:      0,
		sh_addralign: 0x01,
		sh_entsize:   0,
	},
	contents: nil,
}

var s_rela_text = &section{
	sh_name: ".rela.text",
	header: &ElfSectionHeader{
		sh_type:      0x04, // SHT_RELA
		sh_flags:     0x40, // * ??
		sh_link:      0x06,
		sh_info:      0x01,
		sh_addralign: 0x08,
		sh_entsize:   0x18,
	},
	contents: nil,
}

// ".rela.data"
var s_rela_data = &section{
	sh_name: ".rela.data",
	header: &ElfSectionHeader{
		sh_type:      0x04, // SHT_RELA
		sh_flags:     0x40, // I ??
		sh_info:      0x02, // section idx of .data
		sh_addralign: 0x08,
		sh_entsize:   0x18,
	},
	contents: nil,
}

var s_data = &section{
	sh_name: ".data",
	header: &ElfSectionHeader{
		sh_type:      0x01, // SHT_PROGBITS
		sh_flags:     0x03, // SHF_WRITE|SHF_ALLOC
		sh_addr:      0,
		sh_link:      0,
		sh_info:      0,
		sh_addralign: 0x01,
		sh_entsize:   0,
	},
	contents: nil,
}

var s_bss = &section{
	sh_name: ".bss",
	header: &ElfSectionHeader{
		sh_type:      0x08, // SHT_NOBITS
		sh_flags:     0x03, // SHF_WRITE|SHF_ALLOC
		sh_addr:      0,
		sh_link:      0,
		sh_info:      0,
		sh_addralign: 0x01,
		sh_entsize:   0,
	},
	contents: nil,
}

//  ".symtab"
//  SHT_SYMTAB (symbol table)
var s_symtab = &section{
	sh_name: ".symtab",
	header:   sh_symtab,
	contents: nil,
}

var s_shstrtab = &section{
	sh_name: ".shstrtab",
	header: &ElfSectionHeader{
		sh_type:      0x03, // SHT_STRTAB
		sh_flags:     0,
		sh_addr:      0,
		sh_link:      0,
		sh_info:      0,
		sh_addralign: 0x01,
		sh_entsize:   0,
	},
}

// ".strtab"
//
//   This section holds strings, most commonly the strings that
//              represent the names associated with symbol table entries.
//              If the file has a loadable segment that includes the
//              symbol string table, the section's attributes will include
//              the SHF_ALLOC bit.  Otherwise, the bit will be off.  This
//              section is of type SHT_STRTAB.
var s_strtab = &section{
	sh_name: ".strtab",
	header: &ElfSectionHeader{
		sh_type:      0x03, // SHT_STRTAB
		sh_flags:     0,
		sh_addr:      0,
		sh_link:      0,
		sh_info:      0,
		sh_addralign: 0x01,
		sh_entsize:   0,
	},
	contents: nil,
}

// https://reviews.llvm.org/D28950
// The sh_info field of the SHT_SYMTAB section holds the index for the first non-local symbol.

var indexOfFirstNonLocalSymbol int

// ".symtab"
var sh_symtab = &ElfSectionHeader{
	sh_type:      0x02, // SHT_SYMTAB
	sh_flags:     0,
	sh_addr:      0,
//	sh_link:      0x05, // section index of .strtab ?
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}

func calcOffsetOfSection(s *section, prev *section) {
	tentative_offset := prev.header.sh_offset + prev.header.sh_size
	var align  = s.header.sh_addralign
	if align == 0 || align == 1 {
		s.numZeroPad = 0
	} else {
		mod := tentative_offset % align
		if mod == 0 {
			s.numZeroPad = 0
		} else {
			s.numZeroPad = align - mod
		}
	}
	s.header.sh_offset = tentative_offset + s.numZeroPad
	s.header.sh_size = uintptr(len(s.contents))
}

func makeStrTab(symbols []string) []byte {
	var nameOffset uint32
	var data []byte = []byte{0x00}
	nameOffset++
	for _, sym := range symbols {
		//sym.nameOffset = nameOffset
		buf := append([]byte(sym), 0x00)
		data = append(data, buf...)
		nameOffset += uint32(len(buf))
	}

	return data
}

func makeSectionNames(hasRelaText, hasRelaData, hasSymbols bool) []string {
	var sectionNames []string

	if hasSymbols {
		sectionNames = append(sectionNames, ".symtab",".strtab")
	}

	var dataName string = ".data"
	if hasRelaData {
		dataName = ".rela.data"
	}

	var textName = ".text"
	if hasRelaText {
		textName = ".rela.text"
	}

	var names = []string{
		".shstrtab",
		textName,
		dataName,
		".bss",
	}
	sectionNames = append(sectionNames, names...)
	return sectionNames
}


// Make contents of .shstrtab"
func makeShStrTab(sectionNames []string) {
	var data []byte = []byte{0x00}
	for _, name := range sectionNames {
		buf := []byte(name)
		buf = append(buf, 0x00)
		data = append(data, buf...)
	}
	s_shstrtab.contents = data
}

func resolveShNames(ss []*section) {
	for _, s := range ss {
		sh_name := s.sh_name
		idx := bytes.Index(s_shstrtab.contents, []byte(sh_name))
		if idx <= 0 {
			debugf("idx of sh %s = %d\n", s.sh_name, idx)
			panic(  s.sh_name + " is not found in .strtab contents")
		}
		s.header.sh_name = uint32(idx)
	}
}

type symbolStruct struct {
	name       string
	section    string
	address uintptr
}

var textStmts []*statement
var dataStmts []*statement

var allSymbols = make(map[string]*symbolStruct)
var orderedSymbolNames []string
var globalSymbols = make(map[string]bool)

const STT_SECTION = 0x03

func buildSymbolTable(hasRelaData bool) {
	var index int
	if hasRelaData {
		symbolTable = append(symbolTable, &ElfSym{
			st_name:  0,
			st_info:  STT_SECTION,
			st_other: 0,
			st_shndx: uint16(s_data.shndx),
			st_value: 0,
			st_size:  0,
		})
		index++
	}
	var orderedNonGlobalSymbols []string
	var ordererGlobalSymbols []string
	//debugf("globalSymbols=%v\n", globalSymbols)
	for _, sym := range orderedSymbolNames {
		if globalSymbols[sym] {
			ordererGlobalSymbols = append(ordererGlobalSymbols, sym)
		} else {
			orderedNonGlobalSymbols = append(orderedNonGlobalSymbols, sym)
		}
	}
	orderedAllsymbols := append(orderedNonGlobalSymbols,ordererGlobalSymbols...)
	//debugf("orderedAllsymbols=%v\n", orderedAllsymbols)
	s_strtab.contents = makeStrTab(orderedAllsymbols)

	for _, symname := range orderedAllsymbols {
		sym, ok := allSymbols[symname]
		if !ok {
			//debugf("symbol not found :" + symname)
			continue
		}
		index++
		var shndx int
		switch sym.section {
		case ".text":
			shndx = s_text.shndx
		case ".data":
			shndx = s_data.shndx
		default:
			panic("TBI")
		}
		name_offset := bytes.Index(s_strtab.contents, []byte(sym.name))
		if name_offset < 0 {
			panic("name_offset should not be negative")
		}
		var st_info uint8
		if globalSymbols[sym.name] {
			st_info = 0x10 // GLOBAL ?
			if indexOfFirstNonLocalSymbol == 0 {
				indexOfFirstNonLocalSymbol = index
				//debugf("indexOfFirstNonLocalSymbol=%d\n", indexOfFirstNonLocalSymbol)
			}
		}
		//debugf("symbol %s shndx = %d\n", sym.name, shndx)
		e := &ElfSym{
			st_name:  uint32(name_offset),
			st_info:  st_info,
			st_other: 0,
			st_shndx: uint16(shndx),
			st_value: sym.address,
		}
		symbolTable = append(symbolTable, e)
	}

	for _, entry := range symbolTable {
		var buf []byte = ((*[24]byte)(unsafe.Pointer(entry)))[:]
		s_symtab.contents = append(s_symtab.contents, buf...)
	}
}

type relaDataUser struct {
	addr uintptr
	uses string
}

var relaDataUsers []*relaDataUser

func translateData(s *statement) []byte {
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

type addrToReplace struct {
	nextInstrAddr uintptr
	symbolUsed string
}
var unresolvedCodeSymbols = make(map[uintptr]*addrToReplace)

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

type relaTextUser struct {
	addr uintptr
	uses string
}
var relaTextUsers []*relaTextUser

func assert(bol bool, errorMsg string) {
	if !bol {
		panic("assert failed: " + errorMsg)
	}
}

const REX_W byte = 0x48

const ModRegiToRegi uint8 = 0b11
const ModIndirectionWithNoDisplacement uint8 = 0b00
const ModIndirectionWithDisplacement8 uint8 = 0b01
const ModIndirectionWithDisplacement32 uint8 = 0b10

const RM_SPECIAL_101 uint8 = 0b101 // none? rip?

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

/*
var addresses = map[string]uintptr{
	"myGlobalInt": 0x0,
	"pGlobalInt": 0x08,
	"myfunc": 0x30,
	"myfunc2": 0x31,
	"_start": 0,
}
*/
var currentTextAddr uintptr

func dumpCode(code []byte) string {
	var r []string = make([]string, len(code))
	for i, b := range code {
		r[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(r, " ")
}

var debugEncoder bool

func assembleCode(ss []*statement) []byte {
	var code []byte
	for _, s := range ss {
		if s.labelSymbol == "" && s.keySymbol == "" {
			continue
		}
		codeAddr := currentTextAddr
		instr := encode(s)
		buf := instr.code
		currentTextAddr += uintptr(len(buf))
		if debugEncoder {
			debugf("[encoder] %04x : %s\t=>\t%s\n", codeAddr, s.raw,  dumpCode(buf))
		}
		code = append(code, buf...)
	}

	//debugf("iterating unresolvedCodeSymbols...\n")
	for codeAddr, replaceInfo := range unresolvedCodeSymbols {
		sym, ok := allSymbols[replaceInfo.symbolUsed]
		if !ok {
			//debugf("  symbol not found: %s\n" , replaceInfo.symbolUsed)
		} else {
			//debugf("  found symbol:%v\n", sym.name)
			diff := sym.address - replaceInfo.nextInstrAddr
			if diff > 255 {
				panic("diff is too large")
			}
			//debugf("  patching symol addr into code : %s=%02x => %02x (%02x - %02x)\n",
			//	sym.name, codeAddr, diff, sym.address , replaceInfo.nextInstrAddr)
			code[codeAddr] = byte(diff) // @FIXME diff can be larget than a byte
		}
	}
	return code
}

var currentDataAddr uintptr

func assembleData(ss []*statement) []byte {
	var data []byte
	for _, s := range ss {
		buf := translateData(s)
		currentDataAddr += uintptr(len(buf))
		data = append(data, buf...)
	}
	return data
}


func dumpProgram() {
	fmt.Printf("%4s|%29s: |%30s | %s\n", "Line", "Label", "Instruction", "Operands")
	for _, stmt := range dataStmts {
		dumpStmt(0, stmt)
	}
	for _, stmt := range textStmts {
		dumpStmt(0, stmt)
	}
}

func main() {
	//debugParser()
	var err error
	source, err = os.ReadFile("/dev/stdin")
	if err != nil {
		panic(err)
	}

	stmts := parse()
	//dumpStmts(stmts)

	var seenSymbols = make(map[string]bool)
	var currentSection string
	for _, s := range stmts {
		if s == emptyStatement {
			continue
		}
		if s.labelSymbol != "" {
			if !seenSymbols[s.labelSymbol] {
				orderedSymbolNames = append(orderedSymbolNames, s.labelSymbol)
				seenSymbols[s.labelSymbol] = true
			}
		}
		switch s.keySymbol {
		case ".data":
			currentSection = ".data"
			continue
		case ".text":
			currentSection = ".text"
			continue
		case ".global":
			globalSymbols[s.operands[0].ifc.(*symbolExpr).name] = true
		}

		switch currentSection {
		case ".data":
			dataStmts = append(dataStmts, s)
			if s.labelSymbol != "" {
				allSymbols[s.labelSymbol] = &symbolStruct{
					name:    s.labelSymbol,
					section: ".data",
				}
			}
		case ".text":
			textStmts = append(textStmts, s)
			if s.keySymbol == "call" ||  s.keySymbol == "callq" {
				sym := s.operands[0].ifc.(*symbolExpr).name
				if !seenSymbols[sym] {
					orderedSymbolNames = append(orderedSymbolNames, sym)
					seenSymbols[sym] = true
				}
			}
			if s.labelSymbol != "" {
				allSymbols[s.labelSymbol] = &symbolStruct{
					name:    s.labelSymbol,
					section: ".text",
				}
			}
		default:
		}

	}

	//dumpProgram(p)
	code := assembleCode(textStmts)
	//dumpCode(code)
	//return
	s_text.contents = code

	data := assembleData(dataStmts)
	//debugf("mapDataLabelAddr=%v\n", mapDataLabelAddr)
	s_data.contents = data

	//fmt.Printf("symbols=%+v\n",p.symStruct)
	hasRelaText := len(relaTextUsers) > 0
	hasRelaData := len(relaDataUsers) > 0
	hasSymbols := len(allSymbols) > 0
	sectionHeaders := prepareSHTEntries(hasRelaText,hasRelaData, hasSymbols)
	if len(allSymbols) > 0 {
		buildSymbolTable(hasRelaData)
	}

	// build rela_data contents
	var rela_data_c []byte
	for _ , ru := range relaDataUsers {
		sym, ok := allSymbols[ru.uses]
		if !ok {
			panic("label not found")
		}

		rla := &ElfRela{
			r_offset: ru.addr,
			r_info:   0x0100000001,
			r_addend: int64(sym.address),
		}
		p := (*[24]byte)(unsafe.Pointer(rla))[:]
		rela_data_c = append(rela_data_c, p...)
	}
	s_rela_data.contents = rela_data_c

	s_symtab.header.sh_link = uint32(s_strtab.shndx) // @TODO confirm the reason to do this
	sh_symtab.sh_info = uint32(indexOfFirstNonLocalSymbol)
	if len(relaDataUsers) > 0 {
		s_rela_data.header.sh_link = uint32(s_symtab.shndx)
		s_rela_data.header.sh_info = uint32(s_data.shndx)
	}

	// build rela_text contents
	if len(relaTextUsers) > 0 {
		var rela_text_c []byte

		for _ , ru := range relaTextUsers {
			//debugf("re.uses:%s\n", ru.uses)
			sym, ok := allSymbols[ru.uses]
			if !ok {
			//	debugf("symbol not found:" + ru.uses)
				continue
			}

			rla := &ElfRela{
				r_offset: ru.addr,
				r_info:   0x0100000002,
				r_addend: int64(sym.address) - 4, // -4 ????
			}
			p := (*[24]byte)(unsafe.Pointer(rla))[:]
			rela_text_c = append(rela_text_c, p...)
		}
		s_rela_text.contents = rela_text_c

//		s_symtab.header.sh_link = uint32(s_strtab.shndx) // @TODO confirm the reason to do this
//		sh_symtab.sh_info = uint32(indexOfFirstNonLocalSymbol)
//		if needRelaData {
//			s_rela_data.header.sh_link = uint32(s_symtab.shndx)
//			s_rela_data.header.sh_info = uint32(s_data.shndx)
//		}
	}

	sectionNames := makeSectionNames(hasRelaText,hasRelaData, hasSymbols)
	makeShStrTab(sectionNames)

	sectionBodies := buildSectionBodies(hasRelaText,hasRelaData, hasSymbols)
	resolveShNames(sectionBodies)

	// prepare ELF File format
	elfFile := prepareElfFile(sectionBodies, sectionHeaders)
	elfFile.writeTo(os.Stdout)
}

func determineSectionOffsets(sectionBodies []*section) {
	firtSectionInBodies := sectionBodies[0]
	firtSectionInBodies.header.sh_offset = ELFHeaderSize
	firtSectionInBodies.header.sh_size = uintptr(len(firtSectionInBodies.contents))
	for i := 1; i<len(sectionBodies);i++ {
		calcOffsetOfSection(
			sectionBodies[i], sectionBodies[i-1])
	}
}


func calcEShoff(last *ElfSectionHeader) (uintptr,uintptr) {

	endOfLastSection := last.sh_offset + last.sh_size

	var paddingBeforeSHT uintptr
	// align shoff so that e_shoff % 8 be zero. (This is not required actually. Just following gcc's practice)
	mod := endOfLastSection % 8
	if mod != 0 {
		paddingBeforeSHT = 8 - mod
	}
	eshoff := endOfLastSection + paddingBeforeSHT
	return paddingBeforeSHT, eshoff
}

func prepareElfFile(sectionBodies []*section, sectionHeaders []*section) *ElfFile {

	// Calculates offset and zero padding
	determineSectionOffsets(sectionBodies)

	lastSectionHeader := sectionHeaders[len(sectionHeaders) -1].header
	paddingBeforeSHT, eshoff := calcEShoff(lastSectionHeader)

	elfHeader.e_shoff = eshoff
	elfHeader.e_shnum = uint16(len(sectionHeaders))
	elfHeader.e_shstrndx = elfHeader.e_shnum - 1

	// adjust zero padding before each section
	var sections []*ElfSectionBodies
	for _, sect := range sectionBodies {
		// Some sections may not have any contents
		if sect.contents != nil {
			sc := &ElfSectionBodies{
				body: sect.contents,
			}
			if sect.numZeroPad > 0 {
				// pad zeros when required
				sc.zeros = make([]uint8, sect.numZeroPad)
			}
			sections = append(sections, sc)
		}
	}

	var sht []*ElfSectionHeader
	for _, s := range sectionHeaders {
		sht = append(sht, s.header)
	}

	return &ElfFile{
		header:          elfHeader,
		sections :       sections,
		zerosBeforeSHT : make([]uint8, paddingBeforeSHT),
		sht :            sht,
	}
}
