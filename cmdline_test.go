package cmdline

import (
	"strings"
	"testing"
)

type Opts struct {
	verbose bool
	help    string
	file    string
	level   int
	cmd     string
	args    []string
}

func parse_cmdline(s string) (Opts, error) {
	var opts Opts

	fields := strings.Fields(s)
	
	ErrorFunc = ReturnError
	Help = `Verwendung: cmdline [OPTS]`
	
	HelpFunc = func(help string) {
		opts.help = help
	}
	
	err := ParseArgs(fields, func(p *Parser) {
		switch {
		case p.IsOpt("verbose", "v"):
			opts.verbose = true
		case p.IsStrOpt("file", "f"):
			opts.file = p.StrVal()
		case p.IsIntOpt("level", "l", 0, 3):
			opts.level = p.IntVal()
		case p.IsArgN(0):
			opts.cmd = p.Arg()
		case p.IsArg() && p.ArgIdx() < 3:
			opts.args = append(opts.args, p.Arg())
		}
	})

	return opts, err
}

func TestLongOpts(t *testing.T) {
	opts, err := parse_cmdline("/usr/bin/cmdline --verbose --file=file.txt --level=2 cmd arg0 arg1")
	assertEqual(t, Program, "cmdline")
	assertSuccess(t, err)
	assertTrue(t, opts.verbose)
	assertEqual(t, opts.help, "")
	assertEqual(t, opts.file, "file.txt")
	assertEqual(t, opts.cmd, "cmd")
	assertEqual(t, opts.level, 2)
	assertEqual(t, len(opts.args), 2)
	assertEqual(t, opts.args[0], "arg0")
	assertEqual(t, opts.args[1], "arg1")
}

func TestShortOpts(t *testing.T) {
	opts, err := parse_cmdline("cmdline -v -f file.txt -l 2 cmd arg0")
	assertEqual(t, Program, "cmdline")
	assertSuccess(t, err)
	assertTrue(t, opts.verbose)
	assertEqual(t, opts.file, "file.txt")
	assertEqual(t, opts.level, 2)
	assertEqual(t, opts.cmd, "cmd")
	assertEqual(t, len(opts.args), 1)
	assertEqual(t, opts.args[0], "arg0")
}

func TestHelp(t *testing.T) {
	opts, err := parse_cmdline("cmdline --help --verbose")
	assertEqual(t, Program, "cmdline")
	assertSuccess(t, err)
	assertFalse(t, opts.verbose)
	assertEqual(t, opts.help, Help)
	assertEqual(t, opts.file, "")
	assertEqual(t, opts.cmd, "")
	assertEqual(t, opts.level, 0)
	assertEqual(t, len(opts.args), 0)
}

func TestOnlyArgs(t *testing.T) {
	opts, err := parse_cmdline("cmdline -v -- cmd --file=file.txt")
	assertSuccess(t, err)
	assertTrue(t, opts.verbose)
	assertEqual(t, opts.file, "")
	assertEqual(t, opts.cmd, "cmd")
	assertEqual(t, opts.level, 0)
	assertEqual(t, len(opts.args), 1)
	assertEqual(t, opts.args[0], "--file=file.txt")
}

func TestTooManyArgs(t *testing.T) {
	_, err := parse_cmdline("cmdline cmd arg0 arg1 arg2")
	assertError(t, err, "Zu viele Argumente!")
}

func TestUnknownOpt(t *testing.T) {
	_, err := parse_cmdline("cmdline --verbose --unknown")
	assertError(t, err, "Unbekannte Option: --unknown")
}

func TestMissingOptVal(t *testing.T) {
	_, err := parse_cmdline("cmdline --file --verbose")
	assertError(t, err, "Option erwartet ein Options-Argument: --file")
}

func TestUnwantedOptVal(t *testing.T) {
	_, err := parse_cmdline("cmdline --verbose=file1")
	assertError(t, err, "Option erlaubt kein Options-Argument: --verbose")
}

func TestInvalidIntVal(t *testing.T) {
	_, err := parse_cmdline("cmdline --level=nonum")
	assertError(t, err, "Ungültige Zahl: nonum (Option --level)")
	
	_, err = parse_cmdline("cmdline --level=-1")
	assertError(t, err, "Zahl muß >= 0 sein: -1 (Option --level)")
	
	_, err = parse_cmdline("cmdline --level=4")
	assertError(t, err, "Zahl muß <= 3 sein: 4 (Option --level)")
	
	_, err = parse_cmdline("cmdline -l 4")
	assertError(t, err, "Zahl muß <= 3 sein: 4 (Option --level)")
}

func TestFormatHelp(t *testing.T) {
	help := `Verwendung: cmd [OPTS]
	
	| Mein Kommando.
	
	Optionen:
	| -v, --verbose
	
	`
	
	help = FormatHelp(help)
	
	exp := "Verwendung: cmd [OPTS]\n" +
	"\n" +
	"  Mein Kommando.\n" +
	"\n" +
	"Optionen:\n" +
	"  -v, --verbose"
	
	assertEqual(t, help, exp)
}

//--------------------------------------------------------------------------------
// Assertions
//--------------------------------------------------------------------------------

func assertError(t *testing.T, err error, msg string) {
	if err == nil {
		t.Errorf("want error %q, got success", msg)
	} else if (err.Error() != msg) {
		t.Errorf("want error %q, got error %q", msg, err.Error())
	}
}

func assertSuccess(t *testing.T, err error) {
	if err != nil {
		t.Errorf("want success, got error: %q", err.Error())
	}
}

func assertEqual[T comparable](t *testing.T, got, want T) {
	if got != want {
		t.Errorf("want: %v, got: %v", want, got)
	}
}

func assertNotEqual(t *testing.T, got, dont_want int) {
	if got == dont_want {
		t.Errorf("don't want: %d, got: %d", dont_want, got)
	}
}

func assertTrue(t *testing.T, got bool) {
	if got != true {
		t.Errorf("want: true, got: false")
	}
}

func assertFalse(t *testing.T, got bool) {
	if got != false {
		t.Errorf("want: false, got: true")
	}
}
