.PHONEY: all

all: gnu gnu.o.xxd my.o.xxd gnu.readelf

my.o: test.s main.go parser.go
	go run main.go parser.go < $< > $@

# test by GNU tools
gnu.o: test.s
	as -o $@ $<

gnu: gnu.o
	ld -o $@ $<

test-gnu: gnu
	./gnu; test $$? -eq 42 && echo ok

test: gnu my.o.xxd gnu.o.xxd
	make test-gnu
	make diff

my.o.xxd: my.o
	xxd -g 1 -c 8 $< > $@

gnu.o.xxd: gnu.o
	xxd -g 1 -c 8 $< > $@

gnu.readelf: gnu.o
	readelf -a -W $< > $@

.PHONY: diff
diff: gnu.o.xxd my.o.xxd
	diff --color -u my.o.xxd gnu.o.xxd
	@echo ok

.PHONY: clean
clean:
	rm -f gnu *.o *.xxd
