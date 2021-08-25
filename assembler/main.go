package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

func debugf(s string, a ...interface{}) {
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

	sections = append(sections, s_shstrtab)
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
		s_null, // NULL
		s_text, // .text
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
	sh_name:  ".symtab",
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
	sh_type:  0x02, // SHT_SYMTAB
	sh_flags: 0,
	sh_addr:  0,
	//	sh_link:      0x05, // section index of .strtab ?
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}

func calcOffsetOfSection(s *section, prev *section) {
	tentative_offset := prev.header.sh_offset + prev.header.sh_size
	var align = s.header.sh_addralign
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
		sectionNames = append(sectionNames, ".symtab", ".strtab")
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
			panic(s.sh_name + " is not found in .strtab contents")
		}
		s.header.sh_name = uint32(idx)
	}
}

type symbolStruct struct {
	name    string
	section string
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
	orderedAllsymbols := append(orderedNonGlobalSymbols, ordererGlobalSymbols...)
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

type addrToReplace struct {
	nextInstrAddr uintptr
	symbolUsed    string
}

var unresolvedCodeSymbols = make(map[uintptr]*addrToReplace)

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
			debugf("[encoder] %04x : %s\t=>\t%s\n", codeAddr, s.raw, dumpCode(buf))
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
		buf := encodeData(s)
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
	currentSection = ".text"
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
			if s.keySymbol == "call" || s.keySymbol == "callq" {
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
	data := assembleData(dataStmts)
	//debugf("mapDataLabelAddr=%v\n", mapDataLabelAddr)
	s_data.contents = data
	s_text.contents = code

	//fmt.Printf("symbols=%+v\n",p.symStruct)
	hasRelaText := len(relaTextUsers) > 0
	hasRelaData := len(relaDataUsers) > 0
	hasSymbols := len(allSymbols) > 0
	sectionHeaders := prepareSHTEntries(hasRelaText, hasRelaData, hasSymbols)
	if len(allSymbols) > 0 {
		buildSymbolTable(hasRelaData)
	}

	// build rela_data contents
	var rela_data_c []byte
	for _, ru := range relaDataUsers {
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

		for _, ru := range relaTextUsers {
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

	sectionNames := makeSectionNames(hasRelaText, hasRelaData, hasSymbols)
	makeShStrTab(sectionNames)

	sectionBodies := buildSectionBodies(hasRelaText, hasRelaData, hasSymbols)
	resolveShNames(sectionBodies)

	// prepare ELF File format
	elfFile := prepareElfFile(sectionBodies, sectionHeaders)
	elfFile.writeTo(os.Stdout)
}

func determineSectionOffsets(sectionBodies []*section) {
	firtSectionInBodies := sectionBodies[0]
	firtSectionInBodies.header.sh_offset = ELFHeaderSize
	firtSectionInBodies.header.sh_size = uintptr(len(firtSectionInBodies.contents))
	for i := 1; i < len(sectionBodies); i++ {
		calcOffsetOfSection(
			sectionBodies[i], sectionBodies[i-1])
	}
}

func calcEShoff(last *ElfSectionHeader) (uintptr, uintptr) {

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

	lastSectionHeader := sectionHeaders[len(sectionHeaders)-1].header
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
		header:         elfHeader,
		sections:       sections,
		zerosBeforeSHT: make([]uint8, paddingBeforeSHT),
		sht:            sht,
	}
}
