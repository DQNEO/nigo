package main

import (
	"github.com/DQNEO/babygo/lib/strconv"
	"os"
	"fmt"
)

func isDirective(symbol string) bool {
	return len(symbol) > 0 && symbol[0] == '.'
}

func parseArgs(keySymbol string) string {
	//if keySymbol[0] == '.' {
	//	// directive
	//} else {
	//	// instruction
	//}

	var buf []byte
	for ; !atEOL();{
		ch := source[idx]
		if ch == '\n' {
			panic("SHOULD NOT REACH HERE")
		}
		buf = append(buf, ch)
		idx++
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
var lineno int = 1

type statement struct {
	labelSymbol string
	keySymbol string
	args string
}

var emptyStatement = &statement{}

func atEOL() bool {
	return source[idx] == '\n' || source[idx] == '#'
}

func parseFail(msg string) {
	panic(msg + " at line " + strconv.Itoa(lineno))
}

func assert(bol bool) {
	if !bol {
		parseFail("assert failed")
	}
}

func consumeEOL() {
	if source[idx] == '#' {
		skipLineComment()
	}
	assert(source[idx] == '\n')
	idx++
	lineno++
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
	skipWhitespaces()
	if atEOL() {
		consumeEOL()
		return emptyStatement
	}
	var stmt = &statement{}
	symbol := trySymbol()
	if symbol == "" {
		assert(atEOL())
		consumeEOL()
		return emptyStatement
	}

	var keySymbol string

	if source[idx] == ':' {
		// this symbol is a label
		stmt.labelSymbol = symbol
		skipWhitespaces()
		if atEOL() {
			consumeEOL()
			return stmt
		}
		keySymbol = trySymbol()
		if len(keySymbol) == 0 {
			skipWhitespaces()
			assert(atEOL())
			consumeEOL()
			return stmt
		}
	} else {
		// this symbol is the key symbol in this statement
		keySymbol = symbol
	}
	stmt.keySymbol = keySymbol

	skipWhitespaces()
	if atEOL() {
		consumeEOL()
		return stmt
	}

	var args string
	//if keySymbol != "" {
		args = parseArgs(keySymbol)
	//}
	stmt.args = args
	consumeEOL()
	return stmt
}

// GAS Manual: https://sourceware.org/binutils/docs-2.37/as.html
func parse() []*statement {
	var stmts []*statement
	for idx < len(source) {
		s := parseStmt()
		stmts = append(stmts, s)
	}
	return stmts
}

func dumpStmts(stmts []*statement) {
	fmt.Printf("%3s|%30s:|%30s|%30s\n", "NO", "LABEL", "DIRECTIVE", "ARGS")
	for i, stmt := range stmts {
		if stmt == emptyStatement {
			continue
		}
		fmt.Printf("%03d|%29s: |%30s|%30s\n", i+1, stmt.labelSymbol, stmt.keySymbol, stmt.args)
	}
}

type none string

func debugParser() {
	var err error
	source, err = os.ReadFile("/dev/stdin")
	if err != nil {
		panic(err)
	}
	stmts := parse()
	insts := make(map[string]none)
	dircs := make(map[string]none)
	labels := make(map[string]none)

	for _, s := range stmts {
		if isDirective(s.keySymbol) {
			dircs[s.keySymbol] = ""
		} else {
			insts[s.keySymbol] = ""
		}
		labels[s.labelSymbol] = ""
	}

	for k, _ := range labels {
		fmt.Printf("%v\n", k)
	}
	fmt.Println("------------------")
	for k, _ := range dircs {
		fmt.Printf("%v\n", k)
	}
	fmt.Println("------------------")
	for k, _ := range insts {
		fmt.Printf("%v\n", k)
	}
	//dumpStmts(stmts)
}

