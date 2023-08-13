package fmt

import (
	"io"
	"os"
	"reflect"

	"github.com/DQNEO/babygo/lib/strconv"
)

type buffer []byte

type pp struct {
	buf buffer
}

func newPrinter() *pp {
	return &pp{}
}

func (p *pp) doPrintf(format string, a ...interface{}) {
	var r []uint8
	var inPercent bool
	var argIndex int

	for _, c := range []uint8(format) {
		if inPercent {
			if c == '%' { // "%%"
				r = append(r, '%')
			} else {
				arg := a[argIndex]
				var sign uint8 = c
				var str string
				switch sign {
				case '#':
					// skip for now
				case 's': // %s
					switch _arg := arg.(type) {
					case string: // ("%s", "xyz")
						str = _arg
					case int: // ("%s", 123)
						strNumber := strconv.Itoa(_arg)
						str = "%!s(int=" + strNumber + ")" // %!s(int=123)
					default:
						str = "unknown type"
					}
					for _, _c := range []uint8(str) {
						r = append(r, _c)
					}
				case 'd', 'p': // %d
					switch _arg := arg.(type) {
					case string: // ("%d", "xyz")
						str = "%!d(string=" + _arg + ")" // %!d(string=xyz)
					case int: // ("%d", 123)
						str = strconv.Itoa(_arg)
					case uintptr: // ("%d", 123)
						intVal := int(_arg)
						str = strconv.Itoa(intVal)
					default:
						str = "unknown type"
					}
					for _, _c := range []uint8(str) {
						r = append(r, _c)
					}
				case 'T':
					t := reflect.TypeOf(arg)
					if t == nil {
						// ?
					} else {
						str = t.String()
					}
					for _, _c := range []uint8(str) {
						r = append(r, _c)
					}
				default:
					panic("Sprintf: Unknown format:" + string([]uint8{uint8(sign)}))
				}
				argIndex++
			}
			inPercent = false
		} else {
			if c == '%' {
				inPercent = true
			} else {
				r = append(r, c)
			}
		}
	}

	p.buf = r
}

func (p *pp) doPrint(a []interface{}) {
	for _, i := range a {
		s, ok := i.(string)
		if !ok {
			panic("only string is supported")
		}
		bytes := []byte(s)
		for _, b := range bytes {
			p.buf = append(p.buf, b)
		}
	}
}

func (p *pp) doPrintln(a []interface{}) {
	for _, i := range a {
		s, ok := i.(string)
		if !ok {
			panic("only string is supported")
		}
		bytes := []byte(s)
		for _, b := range bytes {
			p.buf = append(p.buf, b)
		}
	}
	p.buf = append(p.buf, '\n')
}

func (p *pp) free() {
	p.buf = nil
}

func Fprintf(w io.Writer, format string, a ...interface{}) (int, error) {
	p := newPrinter()
	p.doPrintf(format, a...)
	n, err := w.Write(p.buf)
	p.free()
	return n, err
}

func Printf(format string, a ...interface{}) (int, error) {
	n, err := Fprintf(os.Stdout, format, a...)
	return n, err
}

func Sprintf(format string, a ...interface{}) string {
	p := newPrinter()
	p.doPrintf(format, a...)
	s := string(p.buf)
	p.free()
	return s
}

func Fprint(w io.Writer, a ...interface{}) (int, error) {
	p := newPrinter()
	p.doPrint(a)
	n, err := w.Write(p.buf)
	p.free()
	return n, err
}

func Print(a ...interface{}) (int, error) {
	n, err := Fprint(os.Stdout, a...)
	return n, err
}

func Fprintln(w io.Writer, a ...interface{}) (int, error) {
	p := newPrinter()
	p.doPrintln(a)
	n, err := w.Write(p.buf)
	p.free()
	return n, err
}

func Println(a ...interface{}) (int, error) {
	n, err := Fprintln(os.Stdout, a...)
	return n, err
}
