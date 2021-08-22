.data
var1:
  .quad 0x10
var2:
  .quad 0x20

.text
.global _start
_start:
  call myfunc1
  call myfunc2
  call myfunc3
  ret

myfunc1:
  ret
myfunc2:
  movq $0x3c, %rax # sys_exit
  movq $0x2a, %rdi # 42
  syscall
  ret
myfunc3:
  ret
