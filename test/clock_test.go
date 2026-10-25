package test

import (
	"strings"
	"testing"
	"time"

	"github.com/y3owk1n/fude/plugin"
)

func TestClock(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "* Task\nbody"
	res := p.Key("<Leader>oxi", text, 0, 0)
	in := applied(text, res)
	now := "[" + time.Now().Format("2006-01-02 Mon") + " "
	if !strings.HasPrefix(in, "* Task\n:LOGBOOK:\nCLOCK: "+now) || !strings.HasSuffix(in, "]\n:END:\nbody") {
		t.Fatalf("clocking in left %q", in)
	}
	if res.Status == nil || *res.Status != "⏱ 0:00 Task" || len(res.Timers) == 0 {
		t.Errorf("clocking in showed %v and asked for timers %v", res.Status, res.Timers)
	}
	out := applied(in, p.Key("<Leader>oxo", in, 0, 0))
	if !strings.Contains(out, "]--"+now) || !strings.Contains(out, "] =>  0:00\n:END:\nbody") {
		t.Errorf("clocking out left %q", out)
	}
	// A heading with a LOGBOOK gets its clock at the top of it.
	again := applied(out, p.Key("<Leader>oxi", out, 0, 0))
	if lines := strings.Split(again, "\n"); len(lines) != 6 || !strings.HasPrefix(lines[2], "CLOCK: ") || strings.Contains(lines[2], "--") {
		t.Errorf("clocking in again left %q", again)
	}
	if got := applied(again, p.Key("<Leader>oxq", again, 0, 0)); got != out {
		t.Errorf("cancelling left %q, want %q", got, out)
	}
	if res := p.Key("<Leader>oxo", text, 0, 0); res.Error == "" {
		t.Error("clocking out with no clock said nothing")
	}
}

func TestClockTable(t *testing.T) {
	text := "* A\n:LOGBOOK:\nCLOCK: [2026-10-25 Sun 10:00]--[2026-10-25 Sun 11:30] =>  1:30\n:END:\n** A1\nCLOCK: [2026-10-25 Sun 12:00]--[2026-10-25 Sun 12:15] =>  0:15\n* B\n"
	table := "#+BEGIN: clocktable :scope file\n| Heading      | Time   |\n|--------------+--------|\n| *Total time* | *1:45* |\n|--------------+--------|\n| A            | 1:45   |\n| \\_ A1        | 0:15   |\n#+END:\n"
	run(t, []step{{"<Leader>oxr", text, len(text), text + table}})
	// In a table, it writes it again.
	stale := strings.Replace(text+table, "1:45   |", "9:99   |", 1)
	run(t, []step{{"<Leader>oxr", stale, len(text) + 40, text + table}})
}
