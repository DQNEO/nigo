GOSOURCES = $(wildcard *.go)
SOURCES = $(wildcard *.s)
GNU_OBJS = $(SOURCES:%.s=%.gnu.o)
MY_OBJS = $(SOURCES:%.s=%.my.o)

.PHONY: diff
diff: objs
	diff test0.gnu.o test0.my.o
	diff test1.gnu.o test1.my.o
	diff test2.gnu.o test2.my.o
	diff test3.gnu.o test3.my.o
	diff test4.gnu.o test4.my.o

.PHONY: objs
objs: $(GNU_OBJS) $(MY_OBJS)

%.gnu.o: %.s
	as -o $@ $<

%.my.o: %.s gas
	./gas < $< > $@

gas: $(GOSOURCES)
	go build -o gas $(GOSOURCES)

clean:
	rm -f gas *.o

