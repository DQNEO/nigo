package main

import (
	"io"
	"unsafe"
)

// ELF format
// see https://en.wikipedia.org/wiki/Executable_and_Linkable_Format#File_layout
// see https://man7.org/linux/man-pages/man5/elf.5.html
// see https://sourceware.org/git/?p=glibc.git;a=blob;f=elf/elf.h;h=4738dfa28f6549fc11654996a15659dc8007e686;hb=HEAD

type ElfFile struct {
	header         *Elf64_Ehdr
	sections       []*ElfSectionContents
	zerosBeforeSHT []uint8
	sht            []*ElfSectionHeader
}

type ElfSectionContents struct {
	zeros []uint8
	body  []uint8
}


// # Part1: ELF Header

//  #define EI_NIDENT (16)
//
//  typedef struct
//  {
//  	unsigned char	e_ident[EI_NIDENT];	/* Magic number and other info */
//  	Elf64_Half	e_type;			/* Object file type */
//  	Elf64_Half	e_machine;		/* Architecture */
//  	Elf64_Word	e_version;		/* Object file version */
//  	Elf64_Addr	e_entry;		/* Entry point virtual address */
//  	Elf64_Off	e_phoff;		/* Program header table file offset */
//  	Elf64_Off	e_shoff;		/* Section header table file offset */
//  	Elf64_Word	e_flags;		/* Processor-specific flags */
//  	Elf64_Half	e_ehsize;		/* ELF header size in bytes */
//  	Elf64_Half	e_phentsize;		/* Program header table entry size */
//  	Elf64_Half	e_phnum;		/* Program header table entry count */
//  	Elf64_Half	e_shentsize;		/* Section header table entry size */
//  	Elf64_Half	e_shnum;		/* Section header table entry count */
//  	Elf64_Half	e_shstrndx;		/* Section header string table index */
//  } Elf64_Ehdr;
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

var elfHeader = &Elf64_Ehdr{
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

// # Part3: Section Header Table
type sectionHeaderTable struct {
	padding  uintptr
	sections []*section
}

// https://man7.org/linux/man-pages/man5/elf.5.html
//   typedef struct { //               uint32_t   sh_name;
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

type ElfSectionHeader struct {
	// This member specifies the name of the section.
	// Its value is an index into the section header string table section,
	// giving the location of a null-terminated string.
	sh_name   uint32  // 4
	sh_type   uint32  // 8
	sh_flags  uintptr // 16
	sh_addr   uintptr // 24
	sh_offset uintptr // 32
	sh_size   uintptr // 40

	// This member holds a section header table index link,
	// whose interpretation depends on the section type.
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

func (elfFile *ElfFile) writeTo(w io.Writer) {
	// Part 1: Write ELF Header
	h := elfFile.header
	buf := ((*[unsafe.Sizeof(*h)]byte)(unsafe.Pointer(h)))[:]
	w.Write(buf)

	// Part 2: Write section contents
	for _, s := range elfFile.sections {
		w.Write(s.zeros)
		w.Write(s.body)
	}

	w.Write(elfFile.zerosBeforeSHT)

	// Part 3: Write Section Header Table
	for _, sh := range elfFile.sht {
		buf := ((*[unsafe.Sizeof(*sh)]byte)(unsafe.Pointer(sh)))[:]
		w.Write(buf)
	}
}
