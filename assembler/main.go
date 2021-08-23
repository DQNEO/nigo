package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

type section struct {
	sh_name    string
	shndx      int
	header     *ElfSectionHeader
	numZeroPad uintptr
	zeros      []uint8
	contents   []uint8
}

func buildSectionBodies() []*section {
	var sections = []*section{
		s_text, // .text
		s_data, // .data
		s_bss,  // .bss (no contents)
	}

	if len(allSymbolNames) > 0 {
		sections = append(sections, s_symtab, s_strtab)
	}

	if len(relaTextUsers) > 0 {
		sections = append(sections, s_rela_text)
	}

	if len(relaDataUsers) > 0 {
		sections = append(sections, s_rela_data)
	}

	sections = append(sections,s_shstrtab)
	return sections
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

func makeSectionNames() []string {
	var sectionNames []string

	if len(allSymbolNames) > 0 {
		sectionNames = append(sectionNames, ".symtab",".strtab")
	}

	var dataName string = ".data"
	if len(relaDataUsers)> 0 {
		dataName = ".rela.data"
	}

	var textName = ".text"
	if len(relaTextUsers) > 0 {
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
			fmt.Fprintf(os.Stderr, "idx of sh %s = %d\n", s.sh_name, idx)
			panic(  s.sh_name + " is not found in .strtab contents")
		}
		s.header.sh_name = uint32(idx)
	}
}

type symbolTableStruct struct {
	dataSymbols []string
	localfuncSymbols []string
	globalfuncSymbols []string
}

type symbolStruct struct {
	name       string
	nameOffset uint32
	section    string
	address uintptr
}

var textStmts []*statement
var dataStmts []*statement
var symStruct symbolTableStruct
var allSymbolNames map[string]*symbolStruct
var orderedSymbolNames []string
var globalSymbols = make(map[string]bool)

const STT_SECTION = 0x03

func buildSymbolTable() {
	var index int
	if len(relaDataUsers)> 0 {
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
	fmt.Fprintf(os.Stderr, "globalSymbols=%v\n", globalSymbols)
	for _, sym := range orderedSymbolNames {
		if globalSymbols[sym] {
			ordererGlobalSymbols = append(ordererGlobalSymbols, sym)
		} else {
			orderedNonGlobalSymbols = append(orderedNonGlobalSymbols, sym)
		}
	}
	orderedAllsymbols := append(orderedNonGlobalSymbols,ordererGlobalSymbols...)
	fmt.Fprintf(os.Stderr, "orderedAllsymbols=%v\n", orderedAllsymbols)
	s_strtab.contents = makeStrTab(orderedAllsymbols)

	for _, symname := range orderedAllsymbols {
		sym := allSymbolNames[symname]
		index++
		var shndx int
		switch sym.section {
		case ".text":
			shndx = s_text.shndx
			addr, ok := mapTextLabelAddr[sym.name]
			if !ok {
				panic("symbol not found from mapTextLabelAddr: " + sym.name)
			}
			//panic("sym.addreess should not be zero")
			sym.address = uintptr(addr)
		case ".data":
			addr, ok := mapDataLabelAddr[sym.name]
			if !ok {
				panic("symbol not found from mapDataLabelAddr: " + sym.name)
			}
			fmt.Fprintf(os.Stderr, "@@@ data symbol %s has addr %x\n", sym.name, addr)
			sym.address = uintptr(addr)
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
				fmt.Fprintf(os.Stderr, "indexOfFirstNonLocalSymbol=%d\n", indexOfFirstNonLocalSymbol)
			}
		}
		fmt.Fprintf(os.Stderr, "symbol %s shndx = %d\n", sym.name, shndx)
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
		mapDataLabelAddr[s.labelSymbol] = currentDataAddr
	}

	switch s.keySymbol {
	case ".quad":
		op := s.operands[0]
		fmt.Fprintf(os.Stderr, ".quad type=%s\n", op.typ)
		switch op.typ {
		case "number":
			rawVal := op.string
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
		case "symbol":
			ru := &relaDataUser{
				addr: currentDataAddr,
				uses: op.string,
			}
			relaDataUsers = append(relaDataUsers, ru)
			return make([]byte, 8)
		default:
			panic("Unexpected op.typ:" + op.typ)
		}
	case "": // label
		//panic("empty keySymbol:" + s.labelSymbol)
	default:
		panic("TBI:"+ s.keySymbol)
	}
	return nil
}

const REX_W byte = 0x48

// The registers are encoded using the 4-bit values in the X.Reg column of the following table.
// X.Reg is in binary.
func regField(reg string) uint8 {
	var x_reg uint8
	switch reg {
	case "ax": x_reg = 0b0000
	case "cx": x_reg = 0b0001
	case "dx": x_reg = 0b0010
	case "bx": x_reg = 0b0011
	case "sp": x_reg = 0b0100
	case "bp": x_reg = 0b0101
	case "si": x_reg = 0b0110
	case "di": x_reg = 0b0111
	default:
		panic("TBI: unexpected register " + reg)
	}
	return x_reg
}

var mapTextLabelAddr = make(map[string]uintptr)
var mapDataLabelAddr = make(map[string]uintptr)

var unresolvedCodeSymbols = make(map[uintptr]string)

// ModR/M
// The ModR/M byte encodes a register or an opcode extension, and a register or a memory address. It has the following fields:
//
//   7   6   5   4   3   2   1   0
// +---+---+---+---+---+---+---+---+
// |  mod  |    reg    |     rm    |
// +---+---+---+---+---+---+---+---+
func composeModRM(mod byte, reg byte, rm byte) byte {
	return mod * 32 + reg * 8 + rm
}

type relaTextUser struct {
	addr uintptr
	uses string
}
var relaTextUsers []*relaTextUser
func translateCode(s *statement) []byte {
	var r []byte

	if s.labelSymbol != "" {
		mapTextLabelAddr[s.labelSymbol] = currentTextAddr
	}
	//fmt.Printf("[translator] %s (%d ops) => ", s.keySymbol, len(s.operands))
	switch s.keySymbol {
	case "nop":
		r = []byte{0x90}
	case "callq","call":
		target_symbol := s.operands[0].string
		_ = target_symbol
		r =  []byte{0xe8, 0, 0, 0, 0}
		unresolvedCodeSymbols[currentTextAddr+1] = target_symbol
	case "movl":
		op1, op2 := s.operands[0], s.operands[1]
		assert(op1.typ == "$number", "op1 type should be $number")
		assert(op2.typ == "register", "op2 type should be register")
		//fmt.Fprintf(os.Stderr, "op1,op2=%s,%s  ", op1, op2)
		intNum, err := strconv.ParseInt(op1.string, 0, 32)
		if err != nil {
			panic(err)
		}
		var num int32 = int32(intNum)
		bytesNum := (*[4]byte)(unsafe.Pointer(&num))
		var opcode byte
		regFieldN := regField(op2.string[1:])
		opcode = 0xb8 + regFieldN
		tmp := []byte{opcode}
		r = append(tmp, (bytesNum[:])...)
	case "movq":
		op1, op2 := s.operands[0], s.operands[1]
//		assert(op1.typ == "$number", "op1 type should be $number")
		assert(op2.typ == "register", "op2 type should be register")
		switch {
		case op1.typ == "$number": // movq $123, %regi
			intNum, err := strconv.ParseInt(op1.string, 0, 32)
			if err != nil {
				panic(err)
			}
			var num int32 = int32(intNum)
			bytesNum := (*[4]byte)(unsafe.Pointer(&num))
			var opcode uint8 = 0xc7
			regFieldN := regField(op2.string[1:])
			var modRM uint8 = 0b11000000+ regFieldN
			r = []byte{REX_W, opcode, modRM}
			r = append(r, bytesNum[:]...)
		case op1.typ == "indirection": // movq foo(%regi), %regi
			splitted := strings.Split(op1.string, ",")
			if splitted[1] == "rip" {
				// RIP relative addressing
				var opcode uint8 = 0x8b

				reg := regField(op2.string[1:])
				modRM := composeModRM(0b000, reg, 0b101)
				r = []byte{REX_W, opcode, modRM}

				symbol := splitted[0]
				ru := &relaTextUser{
					addr: currentTextAddr + uintptr(len(r)),
					uses: symbol,
				}
				r = append(r,  0,0,0,0)
				relaTextUsers = append(relaTextUsers, ru)
			} else {
				panic("TBI")
			}
		default:
			panic("TBI:"+ op1.string)
		}
	case "addl":
		op1, op2 := s.operands[0], s.operands[1]
		assert(op1.typ == "register", "op1 type should be register")
		assert(op2.typ == "register", "op2 type should be register")
		var opcode uint8 = 0x01
		regFieldN := regField(op2.string[1:])
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
		return nil
	default:
		if strings.HasPrefix(s.keySymbol , ".") {
			//fmt.Printf(" (directive)\n")
			return nil
		} else {
			if s.labelSymbol != "" && s.keySymbol == "" {
				//fmt.Printf(" (label)\n")
				return nil
			} else {
				panic("Unexpected key symbols:" + s.keySymbol)
			}
		}
		//return nil
	}

	//fmt.Printf("=>  %#x\n", r)
	currentTextAddr += uintptr(len(r))
	return r
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

func assembleCode(ss []*statement) []byte {
	var code []byte
	for _, s := range ss {
		if s.labelSymbol == "" && s.keySymbol == "" {
			continue
		}
		buf := translateCode(s)
		code = append(code, buf...)
	}

	for addr, symbol := range unresolvedCodeSymbols {
		fmt.Fprintf(os.Stderr, "unresolvedCodeSymbols: addr %x, symbol %s\n", addr, symbol)
		target_addr, ok := mapTextLabelAddr[symbol]
		if !ok {
			panic("symbol not found from mapTextLabelAddr: " + symbol)
		}
		diff := target_addr - (addr + 4)
		code[addr] = byte(diff) // @FIXME diff can be larget than a byte
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

func dumpCode(code []byte) {
	fmt.Printf("[dumping code]\n")
	for _, c := range code {
		fmt.Printf("%x ", c)
	}
	fmt.Println()
}

func main() {
	//debugParser()
	var err error
	source, err = os.ReadFile("/dev/stdin")
	if err != nil {
		panic(err)
	}
	stmts := parse()
	dumpStmts(stmts)

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
			globalSymbols[s.operands[0].string] = true
		}

		switch currentSection {
		case ".data":
			dataStmts = append(dataStmts, s)
			if s.labelSymbol != "" {
				symStruct.dataSymbols = append(symStruct.dataSymbols, s.labelSymbol)
			}
		case ".text":
			textStmts = append(textStmts, s)
			if s.keySymbol == "call" ||  s.keySymbol == "callq" {
				sym := s.operands[0].string
				if !seenSymbols[sym] {
					orderedSymbolNames = append(orderedSymbolNames, sym)
					seenSymbols[sym] = true
				}
			}

			if s.labelSymbol != "" {
				if globalSymbols[s.labelSymbol] {
					symStruct.globalfuncSymbols = append(symStruct.globalfuncSymbols, s.labelSymbol)
				} else {
					symStruct.localfuncSymbols = append(symStruct.localfuncSymbols, s.labelSymbol)
				}
			}
		default:
		}

	}

	var allSymbols = make(map[string]*symbolStruct)

	for _, sym := range symStruct.dataSymbols {
		//		addr, ok := addresses[sym]
		//		if !ok {
		////			panic("address not found")
		//		}
		allSymbols[sym] = &symbolStruct{
			name:    sym,
			section: ".data",
			address: 0,
		}
	}
	for _, sym := range symStruct.localfuncSymbols {
		//		addr, ok := addresses[sym]
		//		if !ok {
		////			panic("address not found")
		//		}
		allSymbols[sym] = &symbolStruct{
			name:    sym,
			section: ".text",
			address: 0,
		}
	}
	for _, sym := range symStruct.globalfuncSymbols {
		//		addr, ok := addresses[sym]
		//		if !ok {
		////			panic("address not found")
		//		}
		allSymbols[sym] = &symbolStruct{
			name:    sym,
			section: ".text",
			address: 0,
			nameOffset: 1,
		}
	}
	allSymbolNames = allSymbols

	//dumpProgram(p)
	code := assembleCode(stmts)
	//dumpCode(code)
	//return
	s_text.contents = code

	data := assembleData(dataStmts)
	//fmt.Fprintf(os.Stderr, "mapDataLabelAddr=%v\n", mapDataLabelAddr)
	s_data.contents = data

	//fmt.Printf("symbols=%+v\n",p.symStruct)
	sectionHeaders := prepareSHTEntries(len(relaTextUsers) > 0,len(relaDataUsers) > 0, len(allSymbolNames) > 0)
	if len(allSymbolNames) > 0 {
		buildSymbolTable()
	}

	// build rela_data contents
	var rela_data_c []byte
	for _ , ru := range relaDataUsers {
		addend, ok := mapDataLabelAddr[ru.uses]
		if !ok {
			panic("label not found")
		}

		rla := &ElfRela{
			r_offset: ru.addr,
			r_info:   0x0100000001,
			r_addend: int64(addend),
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
			fmt.Fprintf(os.Stderr, "re.uses:%s\n", ru.uses)
			addend, ok := mapTextLabelAddr[ru.uses]
			if !ok {
				panic("label not found")
			}
			if addend == 0 {
				addend, ok = mapDataLabelAddr[ru.uses]
				if !ok {
					panic("label not found")
				}
				if ok {

				}
			}
			//msg := fmt.Sprintf("mapDataLabelAddr=%v\n", mapDataLabelAddr)

			rla := &ElfRela{
				r_offset: ru.addr,
				r_info:   0x0100000002,
				r_addend: int64(addend) - 4, // -4 ????
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

	sectionNames := makeSectionNames()
	makeShStrTab(sectionNames)

	sectionBodies := buildSectionBodies()
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
