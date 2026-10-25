package test

import (
	"strings"
	"testing"

	"github.com/y3owk1n/fude/plugin"
)

func TestColumns(t *testing.T) {
	p := load(t, plugin.Grants{})
	p.Path = "plan.org"
	text := "#+COLUMNS: %25ITEM %TODO %Effort(Time){:} %COST{+}\n* Trip :fun:\n** TODO Book\n:PROPERTIES:\n:Effort: 0:30\n:COST: 300\n:END:\n** DONE Pack\n:PROPERTIES:\n:Effort: 1:15\n:COST: 20.5\n:END:\n* Other"
	res := p.Key("<Leader>oC", text, 0, 0)
	if res.Show == nil {
		t.Fatalf("<Leader>oC showed nothing: %+v", res)
	}
	want := strings.Join([]string{
		"| ITEM    | TODO | Time |  COST |",
		"|---------+------+------+-------|",
		"| * Trip  |      | 1:45 | 320.5 |",
		"| ** Book | TODO | 0:30 |   300 |",
		"| ** Pack | DONE | 1:15 |  20.5 |",
		"| * Other |      |      |       |",
	}, "\n")
	if res.Show.Text != want {
		t.Errorf("column view shows\n%s\nwant\n%s", res.Show.Text, want)
	}
	// In a section it shows that section's headings.
	if res := p.Key("<Leader>oC", text, 100, 0); res.Show == nil || strings.Contains(res.Show.Text, "Other") {
		t.Errorf("column view in Trip showed %+v", res.Show)
	}
}
