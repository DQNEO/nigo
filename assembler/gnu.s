.data
myGlobalInt:
  .quad 0x0b

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
  movq $0x2a, %rax
  movq myGlobalInt(%rip), %rax
  movq $0xb, %rax
  movq $0x1f, %rcx
  addq %rcx, %rax
  callq myfunc
  callq myfunc2

  movq $0x3c, %rax
  movq $0x2a, %rdi
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
