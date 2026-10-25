package test

import (
	"strings"
	"testing"

	"github.com/y3owk1n/fude/plugin"
)

func TestTableMove(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "|name|qty|\n|-\n|apple|3|\n"
	aligned := "| name  | qty |\n|-------+-----|\n| apple | 3   |"
	for _, c := range []struct {
		keys string
		at   int
		want string
		pos  int // the line and column the cursor goes to, as line*100+column
	}{
		{"<Tab>", 1, aligned, 10},                         // name to qty
		{"<Tab>", 7, aligned, 202},                        // qty, past the rule, to apple
		{"<S-Tab>", 18, aligned, 10},                      // apple back to qty
		{"<Tab>", 21, aligned + "\n|       |     |", 302}, // past the last cell, a new row
	} {
		res := p.Key(c.keys, text[:len(text)-1], c.at, 0)
		got := applied(text[:len(text)-1], res)
		if got != c.want || res.Jump == nil {
			t.Errorf("%s at %d left %q, want %q", c.keys, c.at, got, c.want)
			continue
		}
		before := got[:res.Jump.Pos]
		line, col := strings.Count(before, "\n"), len(before)-strings.LastIndexByte(before, '\n')-1
		if line*100+col != c.pos {
			t.Errorf("%s at %d left the cursor at line %d column %d, want %d", c.keys, c.at, line, col, c.pos)
		}
	}
	// Outside a table <Tab> asks for nothing, and <S-Tab> cycles the folds.
	if res := p.Key("<Tab>", "text", 0, 0); res.Edits != nil || res.Keys != "" {
		t.Errorf("<Tab> outside a table asked for %+v", res)
	}
}

func TestTableMake(t *testing.T) {
	run(t, []step{
		{"<Leader>o|", "a,b,c", 0, "| a | b | c |"},
		{"<Leader>o|", "| x |yy|", 0, "| x | yy |"},
		{"<Leader>o-", "|a|b|\n|c|d|", 0, "| a | b |\n|---+---|\n| c | d |"},
	})
}

func TestTableCalc(t *testing.T) {
	text := "| item | qty | price | total |\n|-\n| tea | 2 | 1.5 | |\n| cake | 1 | 4 | |\n|-\n| sum | | | |\n#+TBLFM: $4=$2*$3::@4$4=vsum(@2..@3)::@4$2=vsum(@2..@3)"
	want := "| item | qty | price | total |\n|------+-----+-------+-------|\n| tea  |   2 |   1.5 |     3 |\n| cake |   1 |     4 |     4 |\n|------+-----+-------+-------|\n| sum  |   3 |       |     7 |\n#+TBLFM: $4=$2*$3::@4$4=vsum(@2..@3)::@4$2=vsum(@2..@3)"
	p := load(t, plugin.Grants{})
	res := p.Key("<Leader>o=", text, 0, 0)
	if got := applied(text, res); got != want {
		t.Errorf("<Leader>o= left\n%s\nwant\n%s\n%s", got, want, res.Error)
	}
	if res := p.Key("<Leader>o=", "| a |\n#+TBLFM: $1=(2", 0, 0); res.Error == "" {
		t.Error("a broken formula said nothing")
	}
}
