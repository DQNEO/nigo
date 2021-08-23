.data
var1:
  .quad 0x10
var2:
  .quad 0x20
  .quad 0x30
var3:
  .quad var4
var4:
  .quad 0x40
  .quad 0
var5:
  .quad var2

.text
.global _start
_start:
  call myfunc1
  call myfunc2
  call myfunc3
  ret

myfunc1:
  movq var1(%rip), %rcx
  movq $0x1a, %rcx
  ret
myfunc2:
  movq $0x3c, %rax # sys_exit
  movq $0x2a, %rdi # status code
  syscall
  ret
myfunc3:
  ret
