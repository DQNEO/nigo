package runtime

import "syscall"
import "unsafe"

var SYS_BRK int

var heapHead uintptr
var heapCurrent uintptr
var heapTail uintptr
var heapSize uintptr

var s string

func heapInit() {
	SYS_BRK = 12
	heapSize = 40205360
	heapHead = brk(0)
	heapTail = brk(heapHead + heapSize)
	heapCurrent = heapHead
	s = "runtime"
}

func brk(addr0 uintptr) uintptr {
	var ret uintptr = 0
	var arg0 uintptr = uintptr(SYS_BRK)
	var arg1 uintptr = addr0
	var arg2 uintptr = uintptr(0)
	var arg3 uintptr = uintptr(0)
	// @FIXME
	ret = syscall.Syscall(arg0, arg1, arg2, arg3)
	return ret
}

func panic(s string) {
	var buf []uint8 = []uint8(s)
	syscall.Write(2, buf)
	var arg0 uintptr = uintptr(60) // sys exit
	var arg1 uintptr = 1 // status
	var arg2 uintptr = uintptr(0)
	var arg3 uintptr = uintptr(0)
	syscall.Syscall(arg0, arg1, arg2, arg3)
}

func memzeropad(addr1 uintptr, size uintptr) {
	var p *uint8 = (*uint8)(unsafe.Pointer(addr1))
	var isize int = int(size)
	var i int
	var up uintptr
	for i = 0; i < isize; i = i+1 {
		*p = 0
		up = uintptr(unsafe.Pointer(p)) + 1
		p = (*uint8)(unsafe.Pointer(up))
	}
}

func makeSlice(elmSize int, slen int, scap int) (uintptr, int, int) {
	var size uintptr = uintptr(elmSize * scap)
	var addr2 uintptr = malloc(size)
	return addr2, slen, scap
}

func malloc(size uintptr) uintptr {
	if heapCurrent+size > heapTail {
		panic("malloc exceeds heap capacity")
		return 0
	}
	var r uintptr
	r = heapCurrent
	heapCurrent = heapCurrent + size
	memzeropad(r, size)
	return r
}

func catstrings(a string, b string) string {
	var totallen int
	var r []uint8
	totallen = len(a) + len(b)
	r = make([]uint8, totallen, totallen)
	var i int
	for i = 0; i < len(a); i = i + 1 {
		r[i] = a[i]
	}
	var j int
	for j = 0; j < len(b); j = j + 1 {
		r[i+j] = b[j]
	}
	return string(r)
}

func cmpstrings(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var i int
	for i = 0; i < len(a); i = i + 1 {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
