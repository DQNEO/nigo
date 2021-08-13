.text
.global main
main:
  movq $11, %rax
  movq $31, %rcx
  addq %rcx, %rax
  retq
