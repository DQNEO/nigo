.PHONEY: test
test: my.o.xxd gnu.o.xxd gnu.readelf my.readelf diff

my.o: test0.s main.go parser.go
	go run main.go parser.go < $< > $@

gnu.o: test0.s
	as -o $@ $<

my.o.xxd: my.o
	xxd -g 1 -c 8 $< > $@

gnu.o.xxd: gnu.o
	xxd -g 1 -c 8 $< > $@

gnu.readelf: gnu.o
	readelf -a -W $< > $@

my.readelf: my.o
	readelf -a -W $< > $@

.PHONY: diff
diff: gnu.o.xxd my.o.xxd
	diff --color -u my.o.xxd gnu.o.xxd
	@echo ok

test.bin: my.o
	gcc -o $@ $<

test-binary: test.bin
	./test.bin; test $$? -eq 42 && echo ok

.PHONY: clean
clean:
	rm -f *.o *.bin *.readelf *.xxd
