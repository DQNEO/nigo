.data
var1:
  .quad 0x10
var2:
  .quad var1
var3:
  .quad 0
  .quad 0
  .quad 0
var4:
  .quad var2
var5:
  .quad var4

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
