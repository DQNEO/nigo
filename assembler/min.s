.text
.global main
main:
  movq $0x2a, %rax
  movq $0xb, %rax
  movq $0x1f, %rcx
  addq %rcx, %rax
  callq myfunc
  callq myfunc2
  retq
  retq
myfunc:
  retq
myfunc2:
  retq
