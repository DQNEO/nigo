.data
.text
.global main
main:
  movl $0x11, %eax
  movl $0x11, %ecx
  movl $0x11, %edx
  movl $0x11, %ebx
  movl $0x11, %ebx
  movl $0x11, %esp
  movl $0x11, %ebp
  movl $0x11, %esi
  movl $0x11, %edi
  addl %eax, %eax
  addl %eax, %ecx
  addl %eax, %edi
  movl $0x2a, %eax
  ret
