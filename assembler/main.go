package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unsafe"
)

// ELF format
// see https://en.wikipedia.org/wiki/Executable_and_Linkable_Format#File_layout
// see https://man7.org/linux/man-pages/man5/elf.5.html
// see https://sourceware.org/git/?p=glibc.git;a=blob;f=elf/elf.h;h=4738dfa28f6549fc11654996a15659dc8007e686;hb=HEAD

// copied from libc's elf/elf.h
//#define EI_NIDENT (16)
//
//typedef struct
//{
//	unsigned char	e_ident[EI_NIDENT];	/* Magic number and other info */
//	Elf64_Half	e_type;			/* Object file type */
//	Elf64_Half	e_machine;		/* Architecture */
//	Elf64_Word	e_version;		/* Object file version */
//	Elf64_Addr	e_entry;		/* Entry point virtual address */
//	Elf64_Off	e_phoff;		/* Program header table file offset */
//	Elf64_Off	e_shoff;		/* Section header table file offset */
//	Elf64_Word	e_flags;		/* Processor-specific flags */
//	Elf64_Half	e_ehsize;		/* ELF header size in bytes */
//	Elf64_Half	e_phentsize;		/* Program header table entry size */
//	Elf64_Half	e_phnum;		/* Program header table entry count */
//	Elf64_Half	e_shentsize;		/* Section header table entry size */
//	Elf64_Half	e_shnum;		/* Section header table entry count */
//	Elf64_Half	e_shstrndx;		/* Section header string table index */
//} Elf64_Ehdr;

type Elf64_Ehdr struct {
	e_ident [16]uint8
	e_type uint16
	e_machine uint16 // 20
	e_version uint32 // 24
	e_entry uintptr // 32
	e_phoff uintptr // 40
	e_shoff uintptr // 48
	e_flags uint32 // 52
	e_ehsize uint16
	e_phentsize uint16
	e_phnum uint16
	e_shentsize uint16
	e_shnum uint16
	e_shstrndx uint16 // 64
}

//  An object file's symbol table holds information needed to locate
//       and relocate a program's symbolic definitions and references.  A
//       symbol table index is a subscript into this array.
//
//   typedef struct {
//               uint32_t      st_name;
//               unsigned char st_info;
//               unsigned char st_other;
//               uint16_t      st_shndx;
//               Elf64_Addr    st_value;
//               uint64_t      st_size;
//           } Elf64_Sym;

type symbolTableEntry struct {
	// This member holds an index into the object file's symbol
	//              string table, which holds character representations of the
	//              symbol names.  If the value is nonzero, it represents a
	//              string table index that gives the symbol name.  Otherwise,
	//              the symbol has no name.
	st_name uint32

	/* Legal values for ST_TYPE subfield of st_info (symbol type).  */
	// #define STT_NOTYPE      0               /* Symbol type is unspecified */
	// #define STT_OBJECT      1               /* Symbol is a data object */
	// #define STT_FUNC        2               /* Symbol is a code object */
	// #define STT_SECTION     3               /* Symbol associated with a section */
	// #define STT_FILE        4               /* Symbol's name is file name */
	// #define STT_COMMON      5               /* Symbol is a common data object */
	// #define STT_TLS         6               /* Symbol is thread-local data object*/
	// #define STT_NUM         7               /* Number of defined types.  */
	// #define STT_LOOS        10              /* Start of OS-specific */
	// #define STT_GNU_IFUNC   10              /* Symbol is indirect code object */
	// #define STT_HIOS        12              /* End of OS-specific */
	// #define STT_LOPROC      13              /* Start of processor-specific */
	// #define STT_HIPROC      15              /* End of processor-specific */
	st_info uint8
	st_other uint8
	//  Every symbol table entry is "defined" in relation to some
	//  section.  This member holds the relevant section header
	//  table index.
	st_shndx uint16
	// This member gives the value of the associated symbol.
	st_value uintptr
	st_size uint64
}

// https://man7.org/linux/man-pages/man5/elf.5.html
//   typedef struct {
//               uint32_t   sh_name;
//               uint32_t   sh_type;
//               uint64_t   sh_flags;
//               Elf64_Addr sh_addr;
//               Elf64_Off  sh_offset;
//               uint64_t   sh_size;
//               uint32_t   sh_link;
//               uint32_t   sh_info;
//               uint64_t   sh_addralign;
//               uint64_t   sh_entsize;
//           } Elf64_Shdr;

type sectionHeader struct {
	// This member specifies the name of the section.
	// Its value is an index into the section header string table section,
	// giving the location of a null-terminated string.
	sh_name   uint32  // 4
	sh_type   uint32  // 8
	sh_flags  uintptr // 16
	sh_addr   uintptr // 24
	sh_offset uintptr // 32
	sh_size   uintptr // 40
	sh_link   uint32  // 44
	sh_info   uint32  // 48

	// Some sections have address alignment constraints.  If a
	// section holds a doubleword, the system must ensure
	// doubleword alignment for the entire section.  That is, the
	// value of sh_addr must be congruent to zero, modulo the
	// value of sh_addralign.  Only zero and positive integral
	// powers of two are allowed.  The value 0 or 1 means that
	// the section has no alignment constraints.
	sh_addralign uintptr // 56
	sh_entsize uintptr // 64
}

type section struct {
	sh_name string
	header *sectionHeader
	numZeroPad uintptr
	contents []uint8
}


const ELFHeaderSize = unsafe.Sizeof(Elf64_Ehdr{})
const SectionHeaderEntrySize = unsafe.Sizeof(sectionHeader{})

// # Part1: ELF Header
var elfHeader = Elf64_Ehdr{
	e_ident: [16]uint8{
		0x7f, 0x45, 0x4c, 0x46, // 0x7F followed by "ELF"(45 4c 46) in ASCII;
		0x02,                                     // EI_CLASS:2=64-bit
		0x01,                                     // EI_DATA:1=little endian
		0x01,                                     // EI_VERSION:1=the original and current version of ELF.
		0x00,                                     // EI_OSABI: 0=System V
		0x00,                                     // EI_ABIVERSION:
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // EI_PAD: always zero.
	},
	e_type: 1, // ET_REL
	e_machine: 0x3e, // AMD x86-64
	e_version: 1,
	e_entry:0,
	e_phoff: 0,
	//e_shoff: 0, // calculated at runtime
	e_flags:0,
	e_ehsize: uint16(ELFHeaderSize),
	e_phentsize:0,
	e_phnum:0,
	e_shentsize:uint16(SectionHeaderEntrySize), // 64
	// e_shnum: 0, // calculated at runtime
	// e_shstrndx: 0, // calculated at runtime
}

// # Part2: Contents of sections
func makeSectionContentsOrder() []*section {
	var sections = []*section{
		s_text, // .text
		s_data, // .data
		s_bss,  // .bss (no contents)
	}
	if len(p.allSymbolNames) > 0 {
		sections = append(sections, s_symtab, s_strtab)
	}

	//append s_rela_text,
	//append s_rela_data,

	sections = append(sections,s_shstrtab)
	return sections
}

// .symtab
var symbolTable = []*symbolTableEntry{
	&symbolTableEntry{ // NULL entry
	},
/*
	&symbolTableEntry{
		st_info:  0x03, // STT_SECTION
		st_shndx: 0x03, // section ".data"
	},

 */
}

/*
// contents of .rela.text
var rela_text = []byte{
	0x15 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00,
	0x02 ,0x00 ,0x00 ,0x00 ,0x01 ,0x00 ,0x00 ,0x00,
	0x04 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00,
}

var rela_data = []byte{
	0x08 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00,
	0x01 ,0x00 ,0x00 ,0x00 ,0x01 ,0x00 ,0x00 ,0x00,
	0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00 ,0x00,
}
*/

type sectionHeaderTable struct {
	padding  uintptr
	sections []*section
}
// # Part3: Section Header Table
func prepareSHTEntries() []*section {
	r := []*section{
		s_null,      // NULL
		s_text,      // .text
		//		sh_rela_text, // .rela.text
		s_data,      // .data
		//		sh_rela_data, // .rela.data
		s_bss,       // .bss
	}

	if len(p.allSymbolNames) > 0 {
		r = append(r, s_symtab, s_strtab)
	}
	r = append(r, s_shstrtab)
	return r
}
var sht = &sectionHeaderTable{
	sections: nil,
}

var s_null = &section{
	header: &sectionHeader{},
}

var s_text = &section{
	sh_name: ".text",
	header: &sectionHeader{
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
/*
var s_rela_text = &section{
	header:   sh_rela_text,
	contents: rela_text,
}

var s_rela_data = &section{
	header:   sh_rela_data,
	contents: rela_data,
}
*/

var s_data = &section{
	sh_name: ".data",
	header: &sectionHeader{
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
	header: &sectionHeader{
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
	header: &sectionHeader{
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
	header: &sectionHeader{
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


/*
// ".rela.text"
var sh_rela_text = &sectionHeader{
	sh_name:      0x00,
	sh_type:      0x04, // SHT_RELA
	sh_flags:     0x40, // * ??
	sh_link:      0x06,
	sh_info:      0x01,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}
*/

/*
// ".rela.data"
var sh_rela_data = &sectionHeader{
	sh_name:      0x00,
	sh_type:      0x04, // SHT_RELA
	sh_flags:     0x40, // I ??
	sh_link:      0x06,
	sh_info:      0x03,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}
*/

// ".symtab"
var sh_symtab = &sectionHeader{
	sh_type:      0x02, // SHT_SYMTAB
	sh_flags:     0,
	sh_addr:      0,
	sh_link:      0x05, // @TODO calculate dynamically
	// https://reviews.llvm.org/D28950
	// The sh_info field of the SHT_SYMTAB section holds the index for the first non-local symbol.
	sh_info:      0x01, // @TODO calculate dynamically
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

func makeDataSection() {
}

func makeSymbolTable() {
	for _, entry := range symbolTable {
		var buf []byte = ((*[24]byte)(unsafe.Pointer(entry)))[:]
		s_symtab.contents = append(s_symtab.contents, buf...)
	}
}

var allSymbolNames = []string{
	"main",
}

func makeStrTab() []byte {
	var nameOffset uint32
	var data []byte = []byte{0x00}
	nameOffset++
	for _, sym := range allSymbolNames {
		//sym.nameOffset = nameOffset
		buf := append([]byte(sym), 0x00)
		data = append(data, buf...)
		nameOffset += uint32(len(buf))
	}

	return data
}

func makeSectionNames() []string {
	var sectionNames []string

	if len(p.allSymbolNames) > 0 {
		sectionNames = append(sectionNames, ".symtab",".strtab")
	}

	var names = []string{
		".shstrtab",
		".text", // or ".rela.text",
		".data", // or ".rela.data",
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
			panic("invalid idx")
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

type programStruct struct {
	textStmts []*statement
	dataStmts []*statement
	symStruct symbolTableStruct
	allSymbolNames []*symbolStruct // => contents of .strtab
}

var p = &programStruct{}
var symbols []string

var globalSymbols = make(map[string]bool)

func analyze(stmts []*statement) {
	var currentSection string
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
			globalSymbols[s.operands[0].string] = true
		}

		switch currentSection {
		case ".data":
			p.dataStmts = append(p.dataStmts, s)
			if s.labelSymbol != "" {
				symbols = append(symbols, s.labelSymbol)
				p.symStruct.dataSymbols = append(p.symStruct.dataSymbols, s.labelSymbol)
			}
		case ".text":
			p.textStmts = append(p.textStmts, s)
			if s.labelSymbol != "" {
				symbols = append(symbols, s.labelSymbol)
				if globalSymbols[s.labelSymbol] {
					p.symStruct.globalfuncSymbols = append(p.symStruct.globalfuncSymbols, s.labelSymbol)
				} else {
					p.symStruct.localfuncSymbols = append(p.symStruct.localfuncSymbols, s.labelSymbol)
				}
			}
		default:
		}

	}

	var allSymbols []*symbolStruct
	for _, sym := range p.symStruct.dataSymbols {
//		addr, ok := addresses[sym]
//		if !ok {
////			panic("address not found")
//		}
		allSymbols = append(allSymbols, &symbolStruct{
			name:    sym,
			section: ".data",
			address: 0,
		})
	}
	for _, sym := range p.symStruct.localfuncSymbols {
//		addr, ok := addresses[sym]
//		if !ok {
////			panic("address not found")
//		}
		allSymbols = append(allSymbols, &symbolStruct{
			name:    sym,
			section: ".text",
			address: 0,
		})
	}
	for _, sym := range p.symStruct.globalfuncSymbols {
//		addr, ok := addresses[sym]
//		if !ok {
////			panic("address not found")
//		}
		allSymbols = append(allSymbols, &symbolStruct{
			name:    sym,
			section: ".text",
			address: 0,
			nameOffset: 1,
		})
	}
	p.allSymbolNames = allSymbols
//	fmt.Printf("%#v\n", allSymbols)
//	panic("STOP")

	if len(p.allSymbolNames) == 0 {
		return
	}

	s_strtab.contents = makeStrTab()
	for _, sym := range allSymbols {
		var shndx uint16
		switch sym.section {
		case ".text":
			shndx = 0x01
		case ".data":
			shndx = 0x03
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
		}
		e := &symbolTableEntry{
			st_name:  uint32(name_offset),
			st_info:  st_info,
			st_other: 0,
			st_shndx: shndx,
			st_value: sym.address,
		}
		symbolTable = append(symbolTable, e)
	}
}

func translateData(s *statement) []byte {
	switch s.keySymbol {
	case ".quad":
		rawVal := s.operands[0].string
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
	case "": // label
		//panic("empty keySymbol:" + s.labelSymbol)
	default:
		panic("TBI:"+ s.keySymbol)
	}
	return nil
}

var movqIdx int
const REX_W byte = 0x48

var movq0 = []byte{REX_W, 0x8b, 0x05, 0x00, 0x00, 0x00, 0x00,}
var movq1 = []byte{REX_W, 0x8b, 0x00,}
var movq2 = []byte{REX_W, 0xc7, 0xc7,    0x20, 0x00, 0x00, 0x00,}
var movq3 = []byte{REX_W, 0xc7, 0xc0,    0x3c, 0x00, 0x00, 0x00,}
var insts = [][]byte{
	movq0, movq1, movq2, movq3,
}

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

func translateCode(s *statement) []byte {
	var r []byte
	//fmt.Printf("[translator] %s (%d ops) => ", s.keySymbol, len(s.operands))
	switch s.keySymbol {
	case "nop":
		r = []byte{0x90}
	case "callq":
		var dst byte
		switch s.operands[0].string {
		case "myfunc":
			dst = 0x23
		case "myfunc2":
			dst = 0x1f
		default:
			panic("ERROR")
		}
		r =  []byte{0xe8, dst, 0, 0, 0}
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
		assert(op1.typ == "$number", "op1 type should be $number")
		assert(op2.typ == "register", "op2 type should be register")
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
func assembleCode(ss []*statement) []byte {
	var code []byte
	for _, s := range ss {
		buf := translateCode(s)
		code = append(code, buf...)
	}

	return code
}

func assembleData(ss []*statement) []byte {
	var data []byte
	for _, s := range ss {
		buf := translateData(s)
		data = append(data, buf...)
	}
	return data
}


func dumpProgram(p *programStruct) {
	fmt.Printf("%4s|%29s: |%30s | %s\n", "Line", "Label", "Instruction", "Operands")
	for _, stmt := range p.dataStmts {
		dumpStmt(0, stmt)
	}
	for _, stmt := range p.textStmts {
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
	analyze(stmts)

	if len(p.allSymbolNames) > 0 {

	}

	//dumpProgram(p)
	code := assembleCode(stmts)
	//dumpCode(code)
	//return
	s_text.contents = code
	//fmt.Printf("symbols=%+v\n",p.symStruct)

	data := assembleData(p.dataStmts)
	s_data.contents = data

	if len(symbols) > 0 {
		makeSymbolTable()
	}
	sectionNames := makeSectionNames()
	makeShStrTab(sectionNames)

	sectionsOrderByContents := makeSectionContentsOrder()
	resolveShNames(sectionsOrderByContents)
	// Calculates offset and zero padding
	s_text.header.sh_offset = ELFHeaderSize
	s_text.header.sh_size = uintptr(len(s_text.contents))

	for i := 1; i<len(sectionsOrderByContents);i++ {
		calcOffsetOfSection(
			sectionsOrderByContents[i], sectionsOrderByContents[i-1])
	}

	shoff := (s_shstrtab.header.sh_offset + s_shstrtab.header.sh_size)
	// align shoff so that e_shoff % 8 be zero. (This is not required actually. Just following gcc's practice)
	mod := shoff % 8
	if mod != 0 {
		sht.padding = 8 - mod
	}
	e_shoff := shoff + sht.padding
	elfHeader.e_shoff = e_shoff

	sht.sections = prepareSHTEntries()

	elfHeader.e_shnum = uint16(len(sht.sections))
	elfHeader.e_shstrndx = elfHeader.e_shnum - 1

	// Output
	output(&elfHeader, sectionsOrderByContents, sht)
}

func output(elfHeader *Elf64_Ehdr, sections []*section, sht *sectionHeaderTable) {
	// Part 1: Write ELF Header
	var buf []byte = ((*[unsafe.Sizeof(*elfHeader)]byte)(unsafe.Pointer(elfHeader)))[:]
	os.Stdout.Write(buf)

	// Part 2: Write Contents
	for _, sect := range sections {
		// Some sections do not have any contents
		if sect.contents != nil {
			// pad zeros when required
			if sect.numZeroPad > 0 {
				os.Stdout.Write(make([]uint8, sect.numZeroPad))
			}
			os.Stdout.Write(sect.contents)
		}
	}

	// Part 3: Write Section Header Table
	os.Stdout.Write(make([]uint8, sht.padding))
	for _, sec := range sht.sections {
		var buf []byte = ((*[unsafe.Sizeof(*sec.header)]byte)(unsafe.Pointer(sec.header)))[:]
		os.Stdout.Write(buf)
	}
}
