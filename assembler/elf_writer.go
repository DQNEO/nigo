package main

import (
	"io"
	"unsafe"
)

// ELF format
// Spec: https://refspecs.linuxfoundation.org/elf/elf.pdf
//
// see https://man7.org/linux/man-pages/man5/elf.5.html
// see https://sourceware.org/git/?p=glibc.git;a=blob;f=elf/elf.h;h=4738dfa28f6549fc11654996a15659dc8007e686;hb=HEAD
// see https://en.wikipedia.org/wiki/Executable_and_Linkable_Format#File_layout
type ElfFile struct {
	header         *Elf64_Ehdr
	sections       []*ElfSectionBodies
	zerosBeforeSHT []uint8
	sht            []*ElfSectionHeader
}

// Part1: ELF Header

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
	e_ident     [16]uint8
	e_type      uint16
	e_machine   uint16  // 20
	e_version   uint32  // 24
	e_entry     uintptr // 32
	e_phoff     uintptr // 40
	e_shoff     uintptr // 48
	e_flags     uint32  // 52
	e_ehsize    uint16
	e_phentsize uint16
	e_phnum     uint16
	e_shentsize uint16
	e_shnum     uint16
	e_shstrndx  uint16 // 64
}

const ELFHeaderSize = unsafe.Sizeof(Elf64_Ehdr{})

// Fixed data for ELF header.
// e_shoff, e_shentsize, and e_shstrndx should be set dynamically
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
	e_type:      1,    // ET_REL
	e_machine:   0x3e, // AMD x86-64
	e_version:   1,
	e_entry:     0,
	e_phoff:     0,
	e_flags:     0,
	e_ehsize:    uint16(ELFHeaderSize),
	e_phentsize: 0,
	e_phnum:     0,
	e_shentsize: uint16(SectionHeaderEntrySize), // 64
}

// Part2: Section Bodies
type ElfSectionBodies struct {
	zeros []uint8
	body  []uint8
}

// Relocation entries (Rel & Rela)
// Relocation is the process of connecting symbolic references with
// symbolic definitions.  Relocatable files must have information
// that describes how to modify their section contents, thus
// allowing executable and shared object files to hold the right
// information for a process's program image.  Relocation entries
// are these data.
//
// Relocation structures that need an addend:
//     typedef struct {
//               Elf64_Addr r_offset;
//               uint64_t   r_info;
//               int64_t    r_addend;
//           } Elf64_Rela;
//
//       r_offset
//              This member gives the location at which to apply the
//              relocation action.  For a relocatable file, the value is
//              the byte offset from the beginning of the section to the
//              storage unit affected by the relocation.  For an
//              executable file or shared object, the value is the virtual
//              address of the storage unit affected by the relocation.
//
//       r_info This member gives both the symbol table index with respect
//              to which the relocation must be made and the type of
//              relocation to apply.  Relocation types are processor-
//              specific.  When the text refers to a relocation entry's
//              relocation type or symbol table index, it means the result
//              of applying ELF[32|64]_R_TYPE or ELF[32|64]_R_SYM,
//              respectively, to the entry's r_info member.
//
//       r_addend
//              This member specifies a constant addend used to compute
//              the value to be stored into the relocatable field.
type ElfRela struct {
	r_offset uintptr
	r_info   uint64
	r_addend int64
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

type ElfSym struct {
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
	st_info  uint8
	st_other uint8
	//  Every symbol table entry is "defined" in relation to some
	//  section.  This member holds the relevant section header
	//  table index.
	st_shndx uint16
	// This member gives the value of the associated symbol.
	st_value uintptr
	st_size  uint64
}

// # Part3: Section Header Table

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
	sh_entsize   uintptr // 64
}

const SectionHeaderEntrySize = unsafe.Sizeof(ElfSectionHeader{})

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
