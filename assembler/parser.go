package main

import (
	"os"
	"fmt"
)

func parseInstruction(keySymbol string) string {
	if keySymbol[0] == '.' {
		// directive
	} else {
		// instruction
	}

	var buf []byte
	for ; source[idx] != '\n'; idx++ {
		if source[idx] == '#' {
			idx++
			// Anything from a line comment character up to the next newline is considered a comment and is ignored.
			// The line comment character is target specific, and some targets multiple comment characters.
			skipLineComment()
			break
		}
		buf = append(buf, source[idx])
	}
	return string(buf)
}

func trySymbol() string {
	first := source[idx]
	idx++
	if isSymbolBeginning(first) {
		return readSymbol(first)
	} else {
		return ""
	}
}

var source []byte
var idx int
type statement struct {
	labelSymbol string
	keySymbol string
	args string
}

// https://sourceware.org/binutils/docs-2.37/as.html#Statements
//
// A statement ends at a newline character (‘\n’) or a line separator character.
// The newline or line separator character is considered to be part of the preceding statement.
// Newlines and separators within character constants are an exception: they do not end statements.
//
// It is an error to end any statement with end-of-file: the last character of any input file should be a newline.
//
// An empty statement is allowed, and may include whitespace. It is ignored.
//
// A statement begins with zero or more labels, optionally followed by a key symbol
// which determines what kind of statement it is.
// The key symbol determines the syntax of the rest of the statement.
// If the symbol begins with a dot ‘.’ then the statement is an assembler directive: typically valid for any computer. If the symbol begins with a letter the statement is an assembly language instruction: it assembles into a machine language instruction. Different versions of as for different computers recognize different instructions. In fact, the same symbol may represent a different instruction in a different computer’s assembly language.
//
// A label is a symbol immediately followed by a colon (:).
// Whitespace before a label or after a colon is permitted, but you may not have whitespace between a label’s symbol and its colon. See Labels.
func parseStmt() *statement {
	if idx == len(source) {
		return nil // EOF
	}
	skipWhitespaces()
	if source[idx] == '\n' {
		idx++
		// an empty statement
		return &statement{}
	}
	var stmt = &statement{}
	symbol := trySymbol()
	if symbol == "" {
		return stmt
	}
	var keySymbol string
	if source[idx] == ':' {
		// this symbol is a label
		stmt.labelSymbol = symbol
		idx++
		skipWhitespaces()
		if source[idx] == '\n' {
			idx++
			// an empty statement
			return stmt
		}
		keySymbol =  trySymbol()
	} else {
		// this symbol is the key symbol in this statement
		keySymbol = symbol
	}
	stmt.keySymbol = keySymbol
	var args string
	//if keySymbol != "" {
		args = parseInstruction(keySymbol)
	//}
	stmt.args = args
	idx++
	return stmt
}

// GAS Manual: https://sourceware.org/binutils/docs-2.37/as.html
func parse() []*statement {
	var stmts []*statement
	for {
		stmt := parseStmt()
		if stmt == nil {
			return stmts
		}
		//println(stmt)
		stmts = append(stmts, stmt)
	}
}

func debugParser() {
	var err error
	source, err = os.ReadFile("/dev/stdin")
	if err != nil {
		panic(err)
	}
	stmts := parse()
	fmt.Printf("%16s:|%16s|%16s\n", "LABEL", "DIRECTIVE", "ARGS")
	for _, stmt := range stmts {
		fmt.Printf("%16s:|%16s|%16s\n", stmt.labelSymbol, stmt.keySymbol, stmt.args)
	}

}

