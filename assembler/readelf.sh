#!/usr/bin/env bash

file=$1
readelf -W --hex-dump=.text $file
readelf -W --hex-dump=.data $file
readelf -W --relocs $file
readelf -W --hex-dump=.rela.data $file
readelf -W --symbols $file
readelf -W --hex-dump=.symtab $file
readelf -W --hex-dump=.strtab $file
readelf -W --hex-dump=.shstrtab $file
readelf -W --section-headers $file
