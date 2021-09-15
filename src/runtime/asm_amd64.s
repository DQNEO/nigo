.data
runtime.main_main:
  .quad 0

.text

runtime.rt0_go:
  # copy arguments
  movq %rdi, %rax # argc
  movq %rsi, %rbx # argv
  subq $32, %rsp
  movq %rax, 16(%rsp) # argc
  movq %rbx, 24(%rsp) # argv

  movq 16(%rsp), %rax  # copy argc
  movq %rax, 0(%rsp)
  movq 24(%rsp), %rax  # copy argv
  movq %rax, 8(%rsp)
  callq runtime.args

  addq $32, %rsp

  movq %rdi, %rax # argc
  imulq $8,  %rax # argc * 8
  addq %rsp, %rax # stack top addr + (argc * 8)
  addq $16,  %rax # + 16 (skip null and go to next) => envp
  movq %rax, runtime.envp+0(%rip) # envp

  # register main_main
  leaq main.main(%rip), %rax
  movq %rax, runtime.main_main(%rip)

  callq runtime.__initGlobals


  callq runtime.schedinit

  callq main.__initGlobals

  callq os.init # set os.Args

  callq runtime.mstart
  ret # not reached

runtime.mstart:
  callq runtime.mstart0
  ret # not reached

runtime.main:
  movq runtime.main_main(%rip), %rax
  callq *%rax
  pushq $0
  callq runtime.exit
  # End of program


runtime.exit:
  movq 8(%rsp), %rdi  # status 0
  movq $60, %rax # sys_exit
  syscall
