#!/bin/bash
#
# wrapper to my assembler which behaves like `as`
#

if [[ "$1" -eq "" ]]; then
  file="a.out"
else
  file=$1
fi

go run *.go > $file
