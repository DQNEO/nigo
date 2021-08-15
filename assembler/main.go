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

const ELFHeaderSize = unsafe.Sizeof(Elf64_Ehdr{})

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
	e_shentsize:uint16(unsafe.Sizeof(*sh0)), // 64
	// e_shnum: 0, // calculated at runtime
	// e_shstrndx: 0, // calculated at runtime
}

// # Body
var _text1 = []byte{
	0x48, 0xc7, 0xc0, 0x2a, 0x00, 0x00,  0x00, 0xc3, // movq $0x2a, %rax
}

var _text2 []byte = []byte{
	// .text section
	// offset: 0x40,
	// size=len(sc1)=>0x12
	0x48, 0xc7, 0xc0, 0x0b, 0x00, 0x00, 0x00, // movq $0xb, %rax
	0x48, 0xc7, 0xc1, 0x1f, 0x00, 0x00, 0x00, // movq $0x1f, %rcx
	0x48, 0x01, 0xc8,                         // addq %rcx, %rax
	0xc3, // retq
}

var sc1 []byte = _text2

var sc4 = []byte{
	//  SHT_SYMTAB (symbol table)
	// size:0x78(=0x18 * 5)
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x01, 0x00, 0x00, 0x00, 0x10, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

var sc5 = []byte{
	// SHT_STRTAB
	// size: 0x06
	0x00,
	0x6d, 0x61, 0x69, 0x6e, // main
	0x00,
}

var sc6 = []byte{
	// SHT_STRTAB (e_shstrndx =  the section names)
	0x00, 0x2e, 0x73, 0x79, 0x6d, 0x74, 0x61, 0x62, // .symtab
	0x00, 0x2e, 0x73, 0x74, 0x72, 0x74, 0x61, 0x62, // .strtab
	0x00, 0x2e, 0x73, 0x68, 0x73, 0x74, 0x72, 0x74, 0x61, 0x62, //.shstrtab
	0x00, 0x2e, 0x74, 0x65, 0x78, 0x74, // .text
	0x00, 0x2e, 0x64, 0x61, 0x74, 0x61, // .data
	0x00, 0x2e, 0x62, 0x73, 0x73, // .bss
	0x00,
}

type sectionHeader struct {
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

var sh0 = &sectionHeader{
}

var sh1 = &sectionHeader{
	sh_name:      0x1b, // ".text"
	sh_type:      0x01, // SHT_PROGBITS
	sh_flag:      0x06, // SHF_ALLOC|SHF_EXECINSTR
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}

var sz1 uintptr

var sh2 = &sectionHeader{
	sh_name:      0x21, // ".data"
	sh_type:      0x01, // SHT_PROGBITS
	sh_flag:      0x03, // SHF_WRITE|SHF_ALLOC
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}
var sz2 uintptr

var sh3 = &sectionHeader{
	sh_name:      0x27, // ".bss"
	sh_type:      0x08, // SHT_NOBITS
	sh_flag:      0x03, // SHF_WRITE|SHF_ALLOC
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}
var sz3 uintptr

var sh4 = &sectionHeader{
	sh_name:      0x01, // ".symtab"
	sh_type:      0x02, // SHT_SYMTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0x05,
	sh_info:      0x04,
	sh_addralign: 0x08,
	sh_entsize:   0x18,
}
var sz4 uintptr

var sh5 = &sectionHeader{
	sh_name:      0x09, // ".strtab"
	sh_type:      0x03,
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}
var sz5 uintptr
//  this is what e_shstrndx points to
var sh6 *sectionHeader = &sectionHeader{
	sh_name:      0x11, // ".shstrtab"
	sh_type:      0x03, // SHT_STRTAB
	sh_flag:      0,
	sh_addr:      0,
	sh_link:      0,
	sh_info:      0,
	sh_addralign: 0x01,
	sh_entsize:   0,
}
var sz6 uintptr

var sectionHeaderTable = []*sectionHeader{
	sh0,sh1, sh2, sh3, sh4, sh5, sh6,
}

func main() {
	sh1.sh_offst = ELFHeaderSize
	sh1.sh_size = uintptr(len(sc1))

	sh2.sh_offst = sh1.sh_offst + sh1.sh_size

	sh3.sh_offst = sh2.sh_offst + sh2.sh_size

	_offset := sh3.sh_offst + sh3.sh_size
	var align  = sh4.sh_addralign
	sz4 = align - (_offset % align)
	sh4.sh_offst = _offset + sz4
	sh4.sh_size = uintptr(len(sc4))
	sh5.sh_offst = sh4.sh_offst + sh4.sh_size
	sh5.sh_size = uintptr(len(sc5))
	sh6.sh_offst = sh5.sh_offst + sh5.sh_size
	sh6.sh_size = uintptr(len(sc6))

	shoff := (sh6.sh_offst + sh6.sh_size)
	// shoff should be bytes of 8 * x
	mod := shoff % 8
	paddingBeforeSectionHeaderTable := 8 - mod
	shoff += paddingBeforeSectionHeaderTable
	elfHeader.e_shoff = shoff
	elfHeader.e_shnum = uint16(len(sectionHeaderTable))
	elfHeader.e_shstrndx = elfHeader.e_shnum - 1

	// Output

	// Write ELF Header
	var buf []byte = ((*[unsafe.Sizeof(elfHeader)]byte)(unsafe.Pointer(&elfHeader)))[:]
	os.Stdout.Write(buf)

	// Write Contents
	os.Stdout.Write(sc1)
	os.Stdout.Write(make([]uint8, sz4))
	os.Stdout.Write(sc4)
	os.Stdout.Write(sc5)
	os.Stdout.Write(sc6)

	// Write Section Header Table
	os.Stdout.Write(make([]uint8, paddingBeforeSectionHeaderTable))
	for _, entryS := range sectionHeaderTable {
		var buf []byte = ((*[unsafe.Sizeof(*entryS)]byte)(unsafe.Pointer(entryS)))[:]
		os.Stdout.Write(buf)
	}
}
