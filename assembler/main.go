package main

import (
	"fmt"
	"os"
	"unsafe"
)

type token struct {
	typ string
	raw string
}

func isIdent(ch byte) bool{
	switch  {
	case ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z'):
		return true
	case '0' <= ch && ch <= '9':
		return true
	case ch == '_':
		return true
	}
	return false
}

func peekCh() byte {
	if byteIndex == len(source) {
		return 255
	}
	return source[byteIndex]
}

func readCh() byte {
	if byteIndex == len(source) {
		return 255
	}
	ch := source[byteIndex]
	byteIndex++
	return ch
}

func readRegi() string {
	var buf []byte
	for {
		ch := peekCh()
		if isIdent(ch) {
			buf = append(buf, ch)
			byteIndex++
		} else {
			return string(buf)
		}
	}
}

func readIdent(first byte) string {
	var buf []byte  = []byte{first}
	for {
		ch := peekCh()
		if isIdent(ch) {
			buf = append(buf, ch)
			byteIndex++
		} else {
			return string(buf)
		}
	}
}

func readNumber(first byte) string {
	var buf []byte  = []byte{first}
	for {
		ch := peekCh()
		if ('0' <= ch && ch <= '9') || ch == 'x' {
			buf = append(buf, ch)
			byteIndex++
		} else {
			return string(buf)
		}
	}
}

var source []byte
var byteIndex int

func tokenize() []*token {
	var tokens []*token
	for  {
		ch := readCh()
		fmt.Println("byte:", ch)
		var tok *token
		switch  {
		case ch == 255:
			fmt.Println("EOF")
			return tokens
		case ch == ' ':
			continue
		case ch == '\n':
			tok = &token{
			typ: "newline",
			raw: "",
			}
		case ch == '%':
			regi := readRegi()
			tok = &token{
				typ: "regi",
				raw: regi,
			}
		case ch == ':':
			tok = &token{
				typ: "punct",
				raw: ":",
			}
/*
		case ch == '.':
			// keyword (".data" or ".text) or punct
			dotstring := readIdent(ch)
			tok = &token{
				typ: "dotstring",
				raw: dotstring,
			}

 */
		case ch == '.', ch == ',', ch == '(', ch ==')':
			tok = &token{
				typ: "punct",
				raw: ".",
			}
		case ch == '$':
			n := readNumber(ch)
			tok = &token{
				typ: "dolnum",
				raw: n,
			}
		case '0' <= ch && ch <= '9':
			n := readNumber(ch)
			tok = &token{
				typ: "number",
				raw: n,
			}
		case isIdent(ch):
			ident := readIdent(ch)
			tok = &token{
				typ: "ident",
				raw: ident,
			}
		default:
			panic(ch)
		}
		if tok != nil {
			tokens = append(tokens, tok)
		}
	}

	return tokens
}


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
	sh_name uint32 // 4
	sh_type uint32  // 8
	sh_flag uintptr // 16
	sh_addr uintptr // 24
	sh_offst uintptr // 32
	sh_size uintptr // 40
	sh_link uint32 // 44
	sh_info uint32 // 48

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
var sectionsOrderByContents = []*section{
	s_text,      // .text
	s_data,      // .data
	s_bss,       // .bss (no contents)
	s_symtab,    // .symtab
	s_strtab,    // .strtab
	s_rela_text, // .rela.text
	s_rela_data, // .rela.data
	s_shstrtab,  // .shstrtab
}

// ## .text (machine code)
var text []byte = []byte{
	// .text section
	// offset: 0x40,
	// size=len(sc1)=>0x12

	// _start:
	0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, 0x90, // nop * 8
	0xe8, 0x23, 0x00, 0x00, 0x00, // call myfunc
	0xe8, 0x1f, 0x00, 0x00, 0x00, // call myfunc2
	0x48, 0x8b, 0x05, 0x00, 0x00, 0x00, 0x00, // movq myGlobalInt(%rip), %rax
	0x48, 0x8b, 0x00, // movq (%rax),%rax
	0x48, 0xc7, 0xc7, 0x20, 0x00, 0x00, 0x00, // movq $0x20, %rdi
	0x48, 0x01, 0xc7, // addq %rax, %rdi
	0x48, 0xc7, 0xc0, 0x3c, 0x00, 0x00, 0x00, // movq $0x3c, %rax

	0x0f, 0x05, // syscall

	0xc3, // retq
	// myfunc:
	0xc3, // retq
	// myfunc2:
	0xc3, // retq
	0xc3, // retq
	0xc3, // retq
	0xc3, // retq
	0xc3, // retq
	0xc3, // retq
}

// .data
var data = []byte{
	0x0a,0,0,0,0,0,0,0, // .quad 0x0a (8 bytes)
	0,0,0,0,0,0,0,0, // zero ?
}

// .symtab
var symbolTable = []*symbolTableEntry{
	&symbolTableEntry{ // NULL entry
	},
	&symbolTableEntry{
		st_info:  0x03, // STT_SECTION
		st_shndx: 0x03, // section ".data"
	},
	&symbolTableEntry{
		st_name:  0x01, // "myGlobalInt"
		st_info:  0,
		st_shndx: 0x03, // section ".data"
		st_value: 0x0, // ?
	},
	&symbolTableEntry{
		st_name:  0x0d, // "pGlobalInt"
		st_info:  0,
		st_shndx: 0x03, // section ".data"
		st_value: 0x08, // ?
	},
	&symbolTableEntry{
		st_name:  0x18, // "myfunc"
		st_info:  0,
		st_shndx: 0x01, // section 1 ".txt"
		st_value: 0x30, // address of myfunc label
	},
	&symbolTableEntry{
		st_name:  0x1f, // "myfunc2"
		st_info:  0,
		st_shndx: 0x01, // section 1 ".txt"
		st_value: 0x31, // address of myfunc2 label
	},
	&symbolTableEntry{
		st_name:  0x27, // "_start"
		st_info:  0x10, // ?
		st_shndx: 0x01, // section 1 ".txt"
		st_value: 0,
	},
}

// contents of .strtab
var symbolNames = []string{
	"myGlobalInt",
	"pGlobalInt",
	"myfunc",
	"myfunc2",
	"_start",
}

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

// contents of .shstrtab"
var sectionNames = []string{
	".symtab",
	".strtab",
	".shstrtab",
	".rela.text",
	".rela.data",
	".bss",
}

type sectionHeaderTable struct {
	padding uintptr
	entries []*sectionHeader
}
// # Part3: Section Header Table

var sht = &sectionHeaderTable{
	entries: []*sectionHeader{
		sh_null,      // NULL
		sh_text,      // .text
		sh_rela_text, // .rela.text
		sh_data,      // .data
		sh_rela_data, // .rela.data
		sh_bss,       // .bss
		sh_symtab,    // .symtab
		sh_strtab,    // .strtab
		sh_shstrtab,  // .shstrtab
	},
}

var sh_null = &sectionHeader{}

var sh_text = &sectionHeader{
	sh_name:      0x20, // ".text"
	sh_type:      0x01, // SHT_PROGBITS
	sh_flag:      0x06, // SHF_ALLOC|SHF_EXECINSTR
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var sh_rela_text = &sectionHeader{
	sh_name:      0x1b, // ".rela.text"
	sh_type:      0x04, // SHT_RELA ?
	sh_flag:      0x40, // * ??
	sh_link:      0x06,
	sh_info:      0x01,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}

var sh_data = &sectionHeader{
	sh_name:      0x2b, // ".data"
	sh_type:      0x01, // SHT_PROGBITS
	sh_flag:      0x03, // SHF_WRITE|SHF_ALLOC
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var sh_rela_data = &sectionHeader{
	sh_name:      0x26, // ".rela.data"
	sh_type:      0x04, // SHT_RELA ?
	sh_flag:      0x40, // I ??
	sh_link:      0x06,
	sh_info:      0x03,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}

var sh_bss = &sectionHeader{
	sh_name:      0x31, // ".bss"
	sh_type:      0x08, // SHT_NOBITS
	sh_flag:      0x03, // SHF_WRITE|SHF_ALLOC
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}


var sh_symtab = &sectionHeader{
	sh_name:      0x01, // ".symtab"
	sh_type:      0x02, // SHT_SYMTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0x07,
	sh_info:      0x06,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}


//  .strtab
//   This section holds strings, most commonly the strings that
//              represent the names associated with symbol table entries.
//              If the file has a loadable segment that includes the
//              symbol string table, the section's attributes will include
//              the SHF_ALLOC bit.  Otherwise, the bit will be off.  This
//              section is of type SHT_STRTAB.

var sh_strtab = &sectionHeader{
	sh_name:      0x09, // ".strtab"
	sh_type:      0x03, // SHT_STRTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

//  this is what e_shstrndx points to
var sh_shstrtab *sectionHeader = &sectionHeader{
	sh_name:      0x11, // ".shstrtab"
	sh_type:      0x03, // SHT_STRTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var s_text = &section{
	header:   sh_text,
	contents: text,
}

var s_rela_text = &section{
	header:   sh_rela_text,
	contents: rela_text,
}

var s_rela_data = &section{
	header:   sh_rela_data,
	contents: rela_data,
}

var s_data = &section{
	header:   sh_data,
	contents: nil,
}

var s_bss = &section{
	header:   sh_bss,
	contents: nil,
}
//  SHT_SYMTAB (symbol table)
var s_symtab = &section{
	header:   sh_symtab,
	contents: nil,
}

var s_shstrtab = &section{
	header: sh_shstrtab,
}

var s_strtab = &section{
	header:   sh_strtab,
	contents: nil,
}

func calcOffsetOfSection(s *section, prev *section) {
	tentative_offset := prev.header.sh_offst + prev.header.sh_size
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
	s.header.sh_offst = tentative_offset + s.numZeroPad
	s.header.sh_size = uintptr(len(s.contents))
}

func makeDataSection() {
	s_data.contents = data
}

func makeSymbolTable() {
	for _, entry := range symbolTable {
		var buf []byte = ((*[24]byte)(unsafe.Pointer(entry)))[:]
		s_symtab.contents = append(s_symtab.contents, buf...)
	}
}

func makeStrTab() {
	var data []byte = []byte{0x00}
	for _, name := range symbolNames {
		buf := append([]byte(name), 0x00)
		data = append(data, buf...)
	}
	s_strtab.contents = data
}

func makeShStrTab() {
	var data []byte = []byte{0x00}
	for _, name := range sectionNames {
		buf := []byte(name)
		buf = append(buf, 0x00)
		data = append(data, buf...)
	}
	s_shstrtab.contents = data

}

func expect(bol bool) {
	if !bol {
		panic("expect failure")
	}
}

type AstDataBody struct {
	kind *token // ".quad"
	val *token  // "0x123"
}

func parseDataContents(tokens []*token) (*AstDataBody, []*token) {
	expect(tokens[0].raw == ".")
	expect(tokens[1].typ == "ident") // "quad" or something

	return &AstDataBody{
		kind: tokens[1],
		val:  tokens[2],
	}, tokens[3:]

}

func parseDataSection(tokens []*token) {
	fmt.Printf("token=%+v\n", tokens[0])
	tok := tokens[0]
	switch {
	case tok.typ == "ident":
		expect(tokens[1].raw == ":")
		expect(tokens[2].typ == "newline")
		astDataBody , tks := parseDataContents(tokens[3:])
		tokens = tks
		fmt.Printf("astDataBody=%+v\n", astDataBody)
	default:
		panic("STOP")
	}
}


func parseTextBody(tokens []*token) {

}

func parse(tokens []*token) {
	tok := tokens[0]
	switch {
	case tok.raw == ".": // directive or label
		switch {
		case tokens[1].raw == "text": // .text
			parseTextBody(tokens[3:])
		case tokens[1].raw == "data": // .data
			parseDataSection(tokens[3:])
			//dataBody := parseDataBody()
		default:
		}
	}
/*
	 */
	return
	for _, tok := range tokens {
		if tok == nil {
			panic("nil token")
		}

		fmt.Printf("%s \"%s\"\n", tok.typ, tok.raw)
	}

}

func main() {
	src, err := os.ReadFile("/dev/stdin")
	if err != nil {
		panic(err)
	}
	source = src
	tokens := tokenize()
	parse(tokens)
	return
	makeDataSection()
	makeSymbolTable()
	makeStrTab()
	makeShStrTab()

	// Calculates offset and zero padding
	sh_text.sh_offst = ELFHeaderSize
	sh_text.sh_size = uintptr(len(s_text.contents))

	for i := 1; i<len(sectionsOrderByContents);i++ {
		calcOffsetOfSection(
			sectionsOrderByContents[i], sectionsOrderByContents[i-1])
	}

	shoff := (sh_shstrtab.sh_offst + sh_shstrtab.sh_size)
	// align shoff so that e_shoff % 8 be zero. (This is not required actually. Just following gcc's practice)
	mod := shoff % 8
	if mod != 0 {
		sht.padding = 8 - mod
	}
	e_shoff := shoff + sht.padding
	elfHeader.e_shoff = e_shoff
	elfHeader.e_shnum = uint16(len(sht.entries))
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
	for _, entry := range sht.entries {
		var buf []byte = ((*[unsafe.Sizeof(*entry)]byte)(unsafe.Pointer(entry)))[:]
		os.Stdout.Write(buf)
	}
}
