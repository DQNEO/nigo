.PHONEY: all

all: gnu gnu.o.xxd my.o.xxd

my.o: main.go
	go run $< > $@

# test by GNU tools
gnu.o: gnu.s
	as -o $@ $<

gnu: gnu.o
	ld -o $@ $<

test: gnu my.o.xxd gnu.o.xxd
	./gnu; test $$? -eq 42 && echo ok
	make diff

my.o.xxd: my.o
	xxd -g 1 -c 8 -a $< > $@

gnu.o.xxd: gnu.o
	xxd -g 1 -c 8 -a $< > $@

.PHONY: diff
diff: gnu.o.xxd my.o.xxd
	 diff --color -u gnu.o.xxd my.o.xxd

.PHONY: clean
clean:
	rm -f gnu *.o *.xxd
