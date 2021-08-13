.PHONEY: all

all: min

min.o: min.s
	as -o min.o min.s

min: min.o
	gcc -o min min.o

test: min
	./min; test $$? -eq 42 && echo ok

.PHONY: clean
clean:
	rm *.o min
