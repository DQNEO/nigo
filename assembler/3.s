# has .rela.data
.data
var1:
  .quad 0xaa
var2:
  .quad var1

.text
.global main
main:
  nop
  ret
