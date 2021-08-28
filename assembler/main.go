package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unsafe"
)

var debug bool = false

func debugf(s string, a ...interface{}) {
	if !debug {
		return
	}
	fmt.Fprintf(os.Stderr, s, a...)
}

func buildSectionBodies(hasRelaText, hasRelaData, hasSymbols bool) []*section {
	var sections = []*section{
		s_text, // .text
		s_data, // .data
		s_bss,  // .bss (always empty)
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
	index      int
	header     *ElfSectionHeader
	numZeroPad uintptr
	zeros      []uint8
	contents   []uint8
}

// .symtab
var symbolTable = []*ElfSym{
	&ElfSym{}, // NULL entry
}

func prepareSectionHeaderEntries(hasRelaText, hasRelaData, hasSymbols bool) []*section {

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
		s.index = i
	}

	if hasSymbols {
		s_symtab.header.sh_link = uint32(s_strtab.index) // @TODO confirm the reason to do this

		if hasRelaText {
			s_rela_text.header.sh_link = uint32(s_symtab.index)
		}

		if hasRelaData {
			s_rela_data.header.sh_link = uint32(s_symtab.index)
			s_rela_data.header.sh_info = uint32(s_data.index)
		}
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
}

var s_rela_text = &section{
	sh_name: ".rela.text",
	header: &ElfSectionHeader{
		sh_type:      0x04, // SHT_RELA
		sh_flags:     0x40, // * ??
		sh_link:      0x00, // The section header index of the associated symbol table
		sh_info:      0x01,
		sh_addralign: 0x08,
		sh_entsize:   0x18,
	},
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
}

//  ".symtab"
//  SHT_SYMTAB (symbol table)
var s_symtab = &section{
	sh_name: ".symtab",
	header: &ElfSectionHeader{
		sh_type:  0x02, // SHT_SYMTAB
		sh_flags: 0,
		sh_addr:  0,
		//	sh_link:      0x05, // section index of .strtab ?
		sh_addralign: 0x08,
		sh_entsize:   0x18,
	},
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
	debugf("size of section %s is %x\n", s.sh_name, s.header.sh_size)
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
	var r []string

	if hasSymbols {
		r = append(r, ".symtab", ".strtab")
	}

	var dataName string
	var textName string

	if hasRelaData {
		dataName = ".rela.data"
	} else {
		dataName = ".data"
	}

	if hasRelaText {
		textName = ".rela.text"
	} else {
		textName = ".text"
	}

	r = append(r, ".shstrtab", textName, dataName, ".bss")
	return r
}

// Make contents of .shstrtab"
func makeShStrTab(sectionNames []string) {
	buf := []byte{0x00}
	for _, name := range sectionNames {
		buf = append(buf, name...)
		buf = append(buf, 0)
	}
	s_shstrtab.contents = buf
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

var definedSymbols = make(map[string]*symbolStruct)

const STT_SECTION = 0x03

var debugSymbolTable bool = true

func buildSymbolTable(hasRelaData bool, globalSymbols map[string]bool) {
	var index int
	if hasRelaData {
		index++
		symbolTable = append(symbolTable, &ElfSym{
			st_name:  0,
			st_info:  STT_SECTION,
			st_other: 0,
			st_shndx: uint16(s_data.index),
			st_value: 0,
			st_size:  0,
		})
		symbolIndex[".data"] = index

	}
	debugf("symbolsInLexicalOrder[%d]=%v\n", len(symbolsInLexicalOrder), symbolsInLexicalOrder)
	debugf("globalSymbols=%v\n", globalSymbols)

	var localSymbols []string
	var globalDefinedSymbols []string
	var globalUndefinedSymbols []string
	for _, sym := range symbolsInLexicalOrder {
		isGlobal := globalSymbols[sym]
		_, isDefined := definedSymbols[sym]
		if !isDefined {
			isGlobal = true
		}

		if !isGlobal {
			localSymbols = append(localSymbols, sym)
		} else {
			if isDefined {
				globalDefinedSymbols = append(globalDefinedSymbols, sym)
			} else {
				globalUndefinedSymbols = append(globalUndefinedSymbols, sym)
			}
		}
	}

	// local => global defined => global undefined
	allSymbolsForElf := append(localSymbols, globalDefinedSymbols...)
	allSymbolsForElf = append(allSymbolsForElf, globalUndefinedSymbols...)

	//debugf("allSymbolsForElf=%v\n", allSymbolsForElf)
	s_strtab.contents = makeStrTab(allSymbolsForElf)
	//panic(len(allSymbolsForElf))

	// https://reviews.llvm.org/D28950
	// The sh_info field of the SHT_SYMTAB section holds the index for the first non-local symbol.
	var indexOfFirstNonLocalSymbol int

	for _, symname := range allSymbolsForElf {
		isGlobal := globalSymbols[symname]
		sym, isDefined := definedSymbols[symname]
		var addr uintptr
		var shndx int
		if isDefined {
			addr = sym.address
			switch sym.section {
			case ".text":
				shndx = s_text.index
			case ".data":
				shndx = s_data.index
			default:
				panic("TBI")
			}
		} else {
			debugf("undefined symbol  %s\n", symname)
			isGlobal = true
		}

		index++
		name_offset := bytes.Index(s_strtab.contents, []byte(symname))
		if name_offset < 0 {
			panic("name_offset should not be negative")
		}
		var st_info uint8
		if isGlobal {
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
			st_value: addr,
		}
		symbolTable = append(symbolTable, e)
		symbolIndex[symname] = index
		//debugf("[buildSymbolTable] appended. index = %d, name = %s\n", index, symname)
	}

	s_symtab.header.sh_info = uint32(indexOfFirstNonLocalSymbol)

	for _, entry := range symbolTable {
		var buf []byte = ((*[24]byte)(unsafe.Pointer(entry)))[:]
		s_symtab.contents = append(s_symtab.contents, buf...)
	}
}

var symbolIndex = make(map[string]int)

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
	addr   uintptr
	toJump bool
	uses   string
	adjust int64
}

var relaTextUsers []*relaTextUser

func assert(bol bool, errorMsg string) {
	if !bol {
		panic("assert failed: " + errorMsg)
	}
}

func dumpText(code []byte) string {
	var r []string = make([]string, len(code))
	for i, b := range code {
		r[i] = fmt.Sprintf("%02x", b)
	}
	return strings.Join(r, " ")
}

var debugEncoder bool = false

func encodeAllText(ss []*statement) []byte {
	var allText []byte
	var textAddr uintptr
	for _, s := range ss {
		if s.labelSymbol == "" && s.keySymbol == "" {
			continue
		}
		tmpAddr := textAddr
		instr := encode(s, textAddr)
		buf := instr.code
		textAddr += uintptr(len(buf))
		if debugEncoder {
			debugf("[encoder] %04x : %s\t=>\t%s\n", tmpAddr, s.raw, dumpText(buf))
		}
		allText = append(allText, buf...)
	}

	//debugf("iterating unresolvedCodeSymbols...\n")
	for addr, replaceInfo := range unresolvedCodeSymbols {
		sym, ok := definedSymbols[replaceInfo.symbolUsed]
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
			allText[addr] = byte(diff) // @FIXME diff can be larget than a byte
		}
	}
	return allText
}

func encodeAllData(ss []*statement) []byte {
	var dataAddr uintptr
	var allData []byte
	for _, s := range ss {
		buf := encodeData(s, dataAddr)
		dataAddr += uintptr(len(buf))
		allData = append(allData, buf...)
	}
	return allData
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

	var textStmts []*statement
	var dataStmts []*statement

	var globalSymbols = make(map[string]bool)
	var currentSection = ".text"
	for _, s := range stmts {
		if s == emptyStatement {
			continue
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

		if s.labelSymbol != "" {
			definedSymbols[s.labelSymbol] = &symbolStruct{
				name:    s.labelSymbol,
				section: currentSection,
			}
		}

		switch currentSection {
		case ".data":
			dataStmts = append(dataStmts, s)
		case ".text":
			textStmts = append(textStmts, s)
		default:
		}

	}

	code := encodeAllText(textStmts)
	//dumpText(code)
	data := encodeAllData(dataStmts)
	//debugf("mapDataLabelAddr=%v\n", mapDataLabelAddr)
	s_data.contents = data
	s_text.contents = code

	//fmt.Printf("symbols=%+v\n",p.symStruct)
	hasRelaText := len(relaTextUsers) > 0
	//panic(hasRelaText)
	hasRelaData := len(relaDataUsers) > 0
	hasSymbols := len(definedSymbols) > 0
	sectionHeaders := prepareSectionHeaderEntries(hasRelaText, hasRelaData, hasSymbols)
	if len(definedSymbols) > 0 {
		buildSymbolTable(hasRelaData, globalSymbols)
	}

	buildRelaSections(relaTextUsers, relaDataUsers)

	sectionNames := makeSectionNames(hasRelaText, hasRelaData, hasSymbols)
	makeShStrTab(sectionNames)

	sectionBodies := buildSectionBodies(hasRelaText, hasRelaData, hasSymbols)
	resolveShNames(sectionBodies)

	// prepare ELF File format
	elfFile := prepareElfFile(sectionBodies, sectionHeaders)
	elfFile.writeTo(os.Stdout)
}

// build rela text and data contents and headers
func buildRelaSections(relaTextUsers []*relaTextUser, relaDataUsers []*relaDataUser) {

	var rela_data_c []byte
	for _, ru := range relaDataUsers {
		sym, ok := definedSymbols[ru.uses]
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

	if len(relaTextUsers) > 0 {
		var rela_text_c []byte

		for _, ru := range relaTextUsers {
			sym, defined := definedSymbols[ru.uses]
			var addr int64
			if defined {
				if sym.section == ".text" {
					// skip symbols that belong to the same section
					continue
				}
				//	debugf("symbol not found:" + ru.uses)
				addr = int64(sym.address)
			}

			const R_X86_64_PC32 = 2
			const R_X86_64_PLT32 = 4
			var typ uint64
			if ru.toJump {
				typ = R_X86_64_PLT32
			} else {
				typ = R_X86_64_PC32
			}

			var symIdx int
			if defined && sym.section == ".data" {
				symIdx = symbolIndex[".data"]
			} else {
				symIdx = symbolIndex[ru.uses]
			}

			rla := &ElfRela{
				r_offset: ru.addr,                  // 8 bytes
				r_info:   uint64(symIdx)<<32 + typ, // 8 bytes
				r_addend: addr + ru.adjust - 4,     // 8 bytes
			}
			p := (*[24]byte)(unsafe.Pointer(rla))[:]
			debugf("[rela.text] r_offset:%x, r_info=%x, r_addend=%x    (%s)\n", rla.r_offset, rla.r_info, rla.r_addend, ru.uses)
			rela_text_c = append(rela_text_c, p...)
		}
		s_rela_text.contents = rela_text_c
	}
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
