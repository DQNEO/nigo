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
	gcc -o min min.o

test: min
	./min; test $$? -eq 42 && echo ok

.PHONY: compare
compare: min.o my.o
	diff min.o my.o

.PHONY: clean
clean:
	rm *.o min
