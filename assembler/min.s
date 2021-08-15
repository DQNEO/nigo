.text
.global main
main:
  movq $0x2a, %rax
  movq $0xb, %rax
  movq $0x1f, %rcx
  addq %rcx, %rax
  callq myfunc
  retq
  retq
myfunc:
  retq
