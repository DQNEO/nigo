.data
myGlobalInt:
  .quad 0x0a

.text
.global _start
_start:
  nop
  nop
  nop
  nop
  nop
  nop
  nop
  nop
  callq myfunc
  callq myfunc2

  movq myGlobalInt(%rip), %rax
  movq $0x20, %rdi
  addq %rax, %rdi
  movq $0x3c, %rax
  syscall

  retq
myfunc:
  retq
myfunc2:
  retq
  retq
  retq
  retq
  retq
  retq
