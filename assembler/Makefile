.PHONEY: all

all: min my.o

my.o: main.go
	go run main.go > my.o

.PHONY: dump
dump:
	xxd my.o

# test by GNU tools
min.o: min.s
	as -o min.o min.s

min: min.o
	ld -o min min.o

test: min
	./min; test $$? -eq 42 && echo ok
	make compare

my.o.xxd: my.o
	xxd -g 1 $< > $@

min.o.xxd: min.o
	xxd -g 1 $< > $@

.PHONY: compare
compare: min.o.xxd my.o.xxd
	 diff --color -u min.o.xxd my.o.xxd

.PHONY: clean
clean:
	rm *.o min
