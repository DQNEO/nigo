.text
.global _start
_start:
  movq $0x2a, %rax
  movq $0xb, %rax
  movq $0x1f, %rcx
  addq %rcx, %rax
  callq myfunc
  callq myfunc2

  movq $0x3c, %rax
  movq $0x2a, %rdi
  syscall

  retq
  retq
myfunc:
  retq
myfunc2:
  retq
