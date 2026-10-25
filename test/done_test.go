package test

import (
	"strings"
	"testing"
	"time"

	"github.com/y3owk1n/fude/plugin"
)

func TestDone(t *testing.T) {
	p := load(t, plugin.Grants{})
	now := time.Now()
	day := func(days int) string { return now.AddDate(0, 0, days).Format("2006-01-02 Mon") }
	// Done gets the time it closed, and leaving done takes it away.
	closed := applied("* TODO a\nbody", p.Key("cit", "* TODO a\nbody", 0, 0))
	if !strings.HasPrefix(closed, "* DONE a\nCLOSED: ["+day(0)+" ") || !strings.HasSuffix(closed, "]\nbody") {
		t.Fatalf("cit to DONE left %q", closed)
	}
	if got := applied(closed, p.Key("cit", closed, 0, 0)); got != "* a\nbody" {
		t.Errorf("cit from DONE left %q", got)
	}
	// A date that repeats moves on, and the heading stays to do.
	for _, c := range []struct{ plan, want string }{
		{"SCHEDULED: <2026-10-01 Thu +1w>", "SCHEDULED: <2026-10-08 Thu +1w>"},
		{"DEADLINE: <" + day(-10) + " ++1w>", "DEADLINE: <" + day(4) + " ++1w>"},
		{"SCHEDULED: <2026-01-01 Thu 09:00 .+2d>", "SCHEDULED: <" + day(2) + " 09:00 .+2d>"},
	} {
		text := "** TODO a\n" + c.plan + "\nbody"
		got := applied(text, p.Key("cit", text, 0, 0))
		want := "** TODO a\n" + c.want + "\n:PROPERTIES:\n:LAST_REPEAT: [" + day(0) + " "
		if !strings.HasPrefix(got, want) || !strings.HasSuffix(got, "]\n:END:\nbody") {
			t.Errorf("cit on %q left %q, want it to start %q", c.plan, got, want)
		}
	}
}
