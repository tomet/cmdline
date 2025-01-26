// Zum Parsen von Kommandozeilen-Argumenten und -Optionen.
//
//	cmd := ""
//	verbose := false
//	file := ""
//	args := []string{}
//	level := 0
//
//	cmdline.Help = `Verwendung: mycommand [OPTIONEN] [CMD [ARGS]]
//
//	 | Das ist mein eigenes Kommando
//
//	 Optionen:
//	 | -v, --verbose    Verbose Meldungen
//	 | -l, --level=NUM  Verbose-Level (0 bis 3)
//	 | -f, --file=FILE  Datei anzeigen
//	 |     --help       diese Hilfe
//	 `
//
//	cmdline.Parse(func(p *cmdline.Parser) {
//	    case p.IsOpt("verbose", "v"):
//	        verbose = true
//	    case p.IsStrOpt("file", "f"):
//	        file = p.StrVal()
//	     case p.IsIntOpt("level", "l", 0, 3):
//	         level = p.IntVal()
//	    case p.IsArgN(0):
//	        cmd = p.Arg()
//	    case p.IsArg():
//	        args = append(args, p.Arg())
//	    }
//	})
//
//	if cmd == "" {
//	    cmdline.SyntaxError("Fehlendes Kommando!")
//	}
//
// Kann auch gut mit github.com/tomet/ansi verwendet werden um z.B. Fehlermeldungen
// rot auszugeben:
//
//   cmdline.FormatWarningFunc = ansi.Red.Bright().WrapFunc()
package cmdline

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	// der Name des Programms für die Ausgabe von Warn(), RuntimeError() und SyntaxError()
	Program string
	// der Hilfe-Text, der von PrintHelp() ausgegeben wird
	Help string
	// kann verwendet werden um Zeilen von Fehlermeldungen zu formatieren (z.B. in rot auszugeben).
	FormatWarningFunc func(line string) string = DontFormat
	// kann verwendet werden um Zeilen von Infomeldungen zu formatieren (z.B. in weiß ausgeben).
	FormatInfoFunc func(line string) string = DontFormat
	// diese Funktion wird bei einem Fehler beim Parsen (also durch
	// (*Parser).Errorf()) aufgerufen. Normalerweise wird einfach SyntaxError aufgerufen.
	// Siehe auch ReturnError().
	ErrorFunc func(format string, args ...any) = SyntaxError
	// die Funktion, die für die Option --help verwendet werden soll
	HelpFunc func(help string) = PrintHelp
)

//--------------------------------------------------------------------------------
// Funktionen
//--------------------------------------------------------------------------------

// Kann als [ErrorFunc] verwendet werden, falls Syntax-Fehler von [Parse] oder
// [ParseArgs] zurückgegeben werden und nicht zum Programmabbruch führen sollen.
func ReturnError(format string, args ...any) {
}

// Ist der Default-Wert für [FormatWarningFunc] und [FormatInfoFunc] und
// liefert den übergebenen String unformatiert zurück.
func DontFormat(s string) string {
	return s
}

// Parst einen mehrzeiligen Help-String und trimmt führende Spaces.
// Falls eine Zeile mit einem "|"-Zeichen beginnt, wird dieses durch ein Space ersetzt.
// Leerzeilen am Ende werden entfernt.
func FormatHelp(help string) string {
	lines := strings.Split(help, "\n")

	for i, line := range lines {
		line := strings.TrimSpace(line)
		after, found := strings.CutPrefix(line, "|")
		if found {
			line = " " + after
		}
		lines[i] = line
	}

	n_lines := len(lines)

	for n_lines > 0 {
		if lines[n_lines-1] != "" {
			break
		}
		n_lines--
		lines = lines[:n_lines]
	}

	return strings.Join(lines, "\n")
}

// Parst [Help] mit [FormatHelp], gibt das Ergebnis auf Stdout aus und beendet mit os.Exit(0).
func PrintHelp(help string) {
	fmt.Println(FormatHelp(help))
	os.Exit(0)
}

// Gibt eine Fehlermeldung mit "Verwenden Sie --help ..." auf Stderr aus und
// beendet mit os.Exit(1)
func SyntaxError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	Warn("%s\n\nVerwenden Sie --help für weitere Hilfe!", msg)
	os.Exit(1)
}

// Gibt eine Fehlermeldung auf [os.Stderr] aus und beendet mit os.Exit(1).
// Siehe [ProgramMessage].
func RuntimeError(format string, args ...any) {
	Warn(format, args...)
	os.Exit(1)
}

// Gibt eine Fehlermeldung im Format "Program: Fehler" auf [os.Stderr] aus.
// Siehe [ProgramMessage].
func Warn(format string, args ...any) {
	ProgramMessage(os.Stderr, FormatWarningFunc, format, args...)
}

// Git eine Meldung im Format "Program: Meldung" auf [os.Stdout] aus.
// Siehe [ProgramMessage].
func Info(format string, args ...any) {
	ProgramMessage(os.Stdout, FormatInfoFunc, format, args...)
}

// Gibt eine Meldung im Format "Program: Message" auf `fd` aus.
// Die Meldung kann auch mehrzeilig sein. In diesem Fall wird nur in
// der ersten Zeile der Programm-Name ausgegeben. Alle weiteren Zeilen
// werden entsprechend eingerückt.
func ProgramMessage(fd *os.File, formatLineFn func(string) string, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	lines := strings.Split(msg, "\n")
	fmt.Fprintln(fd, formatLineFn(fmt.Sprintf("%s: %s\n", Program, lines[0])))
	if len(lines) > 1 {
		indentLen := len([]rune(Program)) + 2
		indent := strings.Repeat(" ", indentLen)
		for _, line := range lines[1:] {
			fmt.Fprintln(fd, formatLineFn(indent + line))
		}
	}
}

//--------------------------------------------------------------------------------
// Parser
//--------------------------------------------------------------------------------

// Wird von [Parse] bzw. [ParseArgs] an die Funktion übergeben.
type Parser struct {
	rest     []string
	argIdx   int
	onlyArgs bool
	opt      string
	strVal   string
	intVal   int
	grabbed  bool
	err      error
}

// Parst die Kommandozeilen-Argument ([os.Args]) mittels [ParseArgs].
func Parse(fn func(*Parser)) error {
	return ParseArgs(os.Args, fn)
}

// Parst die übergebenen Argumente.
// Das erste Argument muß der Pfad des Executables sein (wird ggf. verwendet um [Program] zu setzen).
// Mit Hilfe des übergebenen Parsers können dann die Argumente und Optionen ausgewertet werden.
func ParseArgs(args []string, fn func(*Parser)) error {
	if len(args) > 0 {
		if Program == "" {
			Program = path.Base(args[0])
		}
		args = args[1:]
	}

	parser := &Parser{rest: args}

	for len(parser.rest) > 0 {
		arg := parser.popNextArg()

		if parser.onlyArgs == true {
			parser.opt = ""
			parser.strVal = arg
		} else if arg == "--help" {
			HelpFunc(Help)
			return nil
		} else {
			if arg == "--" {
				if len(parser.rest) == 0 {
					return parser.err
				}
				parser.onlyArgs = true
				arg = parser.popNextArg()
			}

			parser.opt, parser.strVal = parser.parseArg(arg)
		}

		parser.grabbed = false

		fn(parser)

		if parser.err != nil {
			return parser.err
		}

		if !parser.grabbed {
			if parser.opt != "" {
				return parser.Errorf("Unbekannte Option: --%s", parser.opt)
			} else {
				return parser.Errorf("Zu viele Argumente!")
			}
		}
	}
	return nil
}

func (parser *Parser) popNextArg() string {
	if len(parser.rest) == 0 {
		return ""
	}
	arg := parser.rest[0]
	parser.rest = parser.rest[1:]
	return arg
}

func (parser *Parser) parseArg(arg string) (opt, strVal string) {
	if strings.HasPrefix(arg, "-") {
		parts := strings.SplitN(strings.TrimLeft(arg, "-"), "=", 2)
		opt = parts[0]
		if opt == "" {
			opt = "???"
		}
		if len(parts) > 1 {
			strVal = parts[1]
		}
	} else {
		strVal = arg
	}

	return opt, strVal
}

// Gibt entweder eine [SyntaxError]-Meldung auf Stderr aus oder setzt die
// Fehlermeldung, welche von [Parse] bzw. [ParseArgs] zurückgegeben werden soll.
// Welche Aktion ausgeführt werden soll bestimmt [ErrorFunc].
func (parser *Parser) Errorf(format string, args ...any) error {
	ErrorFunc(format, args...)
	parser.err = fmt.Errorf(format, args...)
	return parser.err
}

//--------------------------------------------------------------------------------
// Argumente/Optionen prüfen
//--------------------------------------------------------------------------------

// Prüft auf Optionen ohne Argumente.
func (parser *Parser) IsOpt(long, short string) bool {
	if parser.opt != long && parser.opt != short {
		return false
	}

	parser.opt = long

	if parser.strVal != "" {
		parser.Errorf("Option erlaubt kein Options-Argument: --%s", parser.opt)
		return false
	}

	parser.grabbed = true
	return true
}

// Prüft auf Optionen mit einem Argument.
func (parser *Parser) IsStrOpt(long, short string) bool {
	if parser.opt != long && parser.opt != short {
		return false
	}

	parser.opt = long

	if parser.strVal != "" {
		parser.grabbed = true
		return true
	}

	if len(parser.rest) > 0 {
		opt, strVal := parser.parseArg(parser.popNextArg())
		if opt == "" && strVal != "" {
			parser.strVal = strVal
			parser.grabbed = true
			return true
		}
	}

	parser.Errorf("Option erwartet ein Options-Argument: --%s", parser.opt)
	return false
}

// Liefert das Options-Argument der letzten Option.
func (parser *Parser) StrVal() string {
	return parser.strVal
}

// Prüft auf Optionen mit einer Integer-Zahl als Options-Argument.
// min und max bestimmen den Gültigkeitsbereich.
func (parser *Parser) IsIntOpt(long, short string, min, max int) bool {
	if !parser.IsStrOpt(long, short) {
		return false
	}

	parsedVal, err := strconv.ParseInt(parser.strVal, 10, 64)
	intVal := int(parsedVal)

	if err != nil {
		parser.Errorf("Ungültige Zahl: %s (Option --%s)", parser.strVal, parser.opt)
		return false
	}
	if intVal < min {
		parser.Errorf("Zahl muß >= %d sein: %d (Option --%s)", min, intVal, parser.opt)
		return false
	}
	if intVal > max {
		parser.Errorf("Zahl muß <= %d sein: %d (Option --%s)", max, intVal, parser.opt)
		return false
	}

	parser.intVal = intVal
	return true
}

// Liefert die Zahl der letzen Integer-Option.
func (parser *Parser) IntVal() int {
	return parser.intVal
}

// Prüft auf ein beliebiges Argument ohne bestimmten Index.
func (parser *Parser) IsArg() bool {
	if parser.opt == "" && parser.strVal != "" {
		return true
	}
	return false
}

// Liefert den Index des aktuellen Arguments
func (parser *Parser) ArgIdx() int {
	return parser.argIdx
}

// Prüft auf ein Argument mit einem bestimmten Index.
// Das erste Argument hat den Index 0.
func (parser *Parser) IsArgN(idx int) bool {
	if parser.argIdx != idx {
		return false
	}
	return parser.IsArg()
}

// Liefert das letzte Argument.
func (parser *Parser) Arg() string {
	if !parser.grabbed {
		parser.grabbed = true
		parser.argIdx++
	}
	return parser.strVal
}
