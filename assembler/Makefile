test_input = test.s
#test_input = ../src/runtime/runtime.s

.PHONY: run
run: my.o

.PHONY: test
test: gnu.o.xxd gnu.readelf my.o.xxd my.readelf diff

my.o:  $(test_input) main.go parser.go elf_writer.go
	go run main.go parser.go elf_writer.go < $< > $@

gnu.o: $(test_input)
	as -o $@ $<

my.o.xxd: my.o
	xxd -g 1 -c 8 $< > $@

gnu.o.xxd: gnu.o
	xxd -g 1 -c 8 $< > $@

gnu.readelf: gnu.o
	./readelf.sh $< > $@

my.readelf: my.o
	./readelf.sh $< > $@

.PHONY: diff
diff: gnu.o.xxd my.o.xxd my.readelf gnu.readelf
	#diff --color -u my.o.xxd gnu.o.xxd
	diff --color -u my.readelf gnu.readelf
	@echo ok

test.bin: my.o
	gcc -o $@ $<

test-binary: test.bin
	./test.bin; test $$? -eq 42 && echo ok

.PHONY: clean
clean:
	rm -f *.o *.bin *.readelf *.xxd
