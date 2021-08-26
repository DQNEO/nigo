#!/usr/bin/env bash
testnum=$1
basename=test${testnum}
my=${basename}.my.o
gnu=${basename}.gnu.o

diff -u --color <(xxd -g 1 -s 64 -o -64 $my) <(xxd -g 1 -s 64 -o -64 $gnu) | head




