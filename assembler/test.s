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
myGlobalInt:
  .quad 0x0a
pGlobalInt:
  .quad myGlobalInt

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
start2:
  nop
  callq myfunc4
  callq myfunc5

  movq pGlobalInt(%rip), %rax
  movq (%rax), %rax
  movq $0x20, %rdi
  addq %rax, %rdi
  movq $0x3c, %rax
  syscall

  retq
myfunc4:
  retq
myfunc5:
  retq
