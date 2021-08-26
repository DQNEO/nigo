GOSOURCES = $(wildcard *.go)
SOURCES = $(wildcard *.s)
GNU_OBJS = $(SOURCES:%.s=%.gnu.o)
MY_OBJS = $(SOURCES:%.s=%.my.o)

.PHONY: diff
diff: objs
	diff 0.gnu.o 0.my.o
	diff 1.gnu.o 1.my.o
	diff 2.gnu.o 2.my.o
	diff 3.gnu.o 3.my.o
	diff 4.gnu.o 4.my.o
	diff 5.gnu.o 5.my.o
	@echo ok

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

