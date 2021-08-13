package main

import (
	"os"
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

var _eh Elf64_Ehdr
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
	e_ehsize: uint16(unsafe.Sizeof(_eh)),
	e_phentsize:0,
	e_phnum:0,
	e_shentsize:0x40,
	// e_shnum: 0, // calculated at runtime
	// e_shstrndx: 0, // calculated at runtime
}

const shentsize = 0x40

// # Body
var text []byte = []byte{
	// .text section
	// offset: 0x40, size=len(text)=>0x12
	0x48, 0xc7, 0xc0, 0x0b, 0x00, 0x00, 0x00, // movq $0xb, %rax
	0x48, 0xc7, 0xc1, 0x1f, 0x00, 0x00, 0x00, // movq $0x1f, %rcx
	0x48, 0x01, 0xc8,                         // addq %rcx, %rax
	0xc3, // retq
}

var symtab = []byte{
	//  SHT_SYMTAB (symbol table)
	// size:0x78(=0x18 * 5)
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

var strtab1 = []byte{
	// SHT_STRTAB
	// size: 0x06
	0x00,
	0x6d, 0x61, 0x69, 0x6e, // main
	0x00,
}

var strtabSectionNames = []byte{
	// SHT_STRTAB (e_shstrndx =  the section names)
	0x00, 0x2e, 0x73, 0x79, 0x6d, 0x74, 0x61, 0x62, // .symtab
	0x00, 0x2e, 0x73, 0x74, 0x72, 0x74, 0x61, 0x62, // .strtab
	0x00, 0x2e, 0x73, 0x68, 0x73, 0x74, 0x72, 0x74, 0x61, 0x62, //.shstrtab
	0x00, 0x2e, 0x74, 0x65, 0x78, 0x74, // .text
	0x00, 0x2e, 0x64, 0x61, 0x74, 0x61, // .data
	0x00, 0x2e, 0x62, 0x73, 0x73, // .bss
	0x00,
}

var zeropad = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

type SectionHeaderTableEntry struct {
	sh_name uint32 // 4
	sh_type uint32  // 8
	sh_flag uintptr // 16
	sh_addr uintptr // 24
	sh_offst uintptr // 32
	sh_size uintptr // 40
	sh_link uint32 // 44
	sh_info uint32 // 48
	sh_addralign uintptr // 56
	sh_entsize uintptr // 64
}

var hts0 = &SectionHeaderTableEntry{
}

var hts1 = &SectionHeaderTableEntry{
	sh_name:      0x1b, // ".text"
	sh_type:      0x01, // SHT_PROGBITS
	sh_flag:      0x06, // SHF_ALLOC|SHF_EXECINSTR
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var hts2 = &SectionHeaderTableEntry{
	sh_name:      0x21, // ".data"
	sh_type:      0x01, // SHT_PROGBITS
	sh_flag:      0x03, // SHF_WRITE|SHF_ALLOC
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 01,
	sh_entsize:   0,
}

var hts3 = &SectionHeaderTableEntry{
	sh_name:      0x27, // ".bss"
	sh_type:      0x08, // SHT_NOBITS
	sh_flag:      0x03, // SHF_WRITE|SHF_ALLOC
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var hts4 = &SectionHeaderTableEntry{
	sh_name:      0x01, // ".symtab"
	sh_type:      0x02, // SHT_SYMTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0x05,
	sh_info:      0x04,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}

var hts5 = &SectionHeaderTableEntry{
	sh_name:      0x09, // ".strtab"
	sh_type:      0x03,
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

//  this is what e_shstrndx points to
var hts6 *SectionHeaderTableEntry = &SectionHeaderTableEntry{
	sh_name:      0x11, // ".shstrtab"
	sh_type:      0x03, // SHT_STRTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var body [][]byte = [][]byte{
	text,
	[]byte{0,0,0,0,0,0},
	symtab,
	strtab1,
	strtabSectionNames,
	zeropad,
}

var sectionHeaderTable = []*SectionHeaderTableEntry{
	hts0,hts1,hts2,hts3,hts4,hts5,hts6,
}

const offset_offset = 24
const size_offset = offset_offset + 8
func setOffsetsOfSectionHeaderTable() {

	hts1.sh_offst = 0x40
	hts1.sh_size = uintptr(len(text))
	hts2.sh_offst = hts1.sh_offst + hts1.sh_size

	hts3.sh_offst = hts2.sh_offst
	hts4.sh_offst = hts3.sh_offst + 6 // 6 is what ?
	hts4.sh_size = uintptr(len(symtab))
	hts5.sh_offst = hts4.sh_offst + hts4.sh_size
	hts5.sh_size = uintptr(len(strtab1))
	hts6.sh_offst = hts5.sh_offst + hts5.sh_size
	hts6.sh_size = uintptr(len(strtabSectionNames))
}

func main() {
	setOffsetsOfSectionHeaderTable()
	var shoff = 0x108 // 264
	elfHeader.e_shoff = uintptr(shoff)
	elfHeader.e_shnum = uint16(len(sectionHeaderTable))
	elfHeader.e_shstrndx = elfHeader.e_shnum - 1

	var buf []byte = ((*[unsafe.Sizeof(elfHeader)]byte)(unsafe.Pointer(&elfHeader)))[:]
	os.Stdout.Write(buf)

	for _, buf := range body {
		os.Stdout.Write(buf)
	}

	for _, entryS := range sectionHeaderTable {
		var buf []byte = ((*[unsafe.Sizeof(*entryS)]byte)(unsafe.Pointer(entryS)))[:]
		os.Stdout.Write(buf)
	}
}
