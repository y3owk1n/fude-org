package test

import (
	"testing"
	"time"

	"github.com/y3owk1n/fude/plugin"
)

// stamp returns the active timestamp of the local date days from today.
func stamp(days int) string {
	return "<" + time.Now().AddDate(0, 0, days).Format("2006-01-02 Mon") + ">"
}

func TestSchedule(t *testing.T) {
	p := load(t, plugin.Grants{})
	text := "* a\nbody"
	set, _ := answer(t, p, "<Leader>os", text, 5, "2026-11-02 10:00 +1w")
	if want := "* a\nSCHEDULED: <2026-11-02 Mon 10:00 +1w>\nbody"; set != want {
		t.Fatalf("<Leader>os left %q, want %q", set, want)
	}
	// It asks from the date it has, and none removes it.
	if got, res := answer(t, p, "<Leader>os", set, 0, ""); res.Pick.Query != "2026-11-02 10:00 +1w" || got != text {
		t.Errorf("<Leader>os asked from %q and left %q", res.Pick.Query, got)
	}
	weekday := (int(time.Friday) - int(time.Now().Weekday()) + 7) % 7
	for typed, want := range map[string]string{
		"+3d":      "* a\nDEADLINE: " + stamp(3) + " SCHEDULED: <2026-11-02 Mon 10:00 +1w>\nbody",
		"tomorrow": "* a\nDEADLINE: " + stamp(1) + " SCHEDULED: <2026-11-02 Mon 10:00 +1w>\nbody",
		"fri":      "* a\nDEADLINE: " + stamp(weekday) + " SCHEDULED: <2026-11-02 Mon 10:00 +1w>\nbody",
	} {
		if got, _ := answer(t, p, "<Leader>od", set, 0, typed); got != want {
			t.Errorf("<Leader>od %s left %q, want %q", typed, got, want)
		}
	}
	if _, res := answer(t, p, "<Leader>od", text, 0, "someday"); res.Error == "" {
		t.Error("<Leader>od someday said nothing")
	}
	if got, _ := answer(t, p, "<Leader>o.", "x ", 1, ""); got != "x"+stamp(0)+" " {
		t.Errorf("<Leader>o. left %q", got)
	}
}

func TestShiftStamp(t *testing.T) {
	line := "SCHEDULED: <2026-10-31 Sat 23:30>"
	p := load(t, plugin.Grants{})
	for _, c := range []struct {
		keys      string
		at, count int
		want      string
	}{
		{"<C-a>", 21, 0, "SCHEDULED: <2026-11-01 Sun 23:30>"},
		{"<C-x>", 21, 3, "SCHEDULED: <2026-10-28 Wed 23:30>"},
		{"<C-a>", 13, 0, "SCHEDULED: <2027-10-31 Sun 23:30>"},
		{"<C-a>", 29, 0, "SCHEDULED: <2026-11-01 Sun 00:30>"},
		{"<S-Up>", 25, 0, "SCHEDULED: <2026-11-01 Sun 23:30>"},
		{"<S-Left>", 14, 0, "SCHEDULED: <2026-10-30 Fri 23:30>"},
	} {
		res := p.Key(c.keys, line, c.at, c.count)
		if got := applied(line, res); got != c.want || res.Jump == nil || res.Jump.Pos != c.at {
			t.Errorf("%d%s at %d left %q with the cursor at %+v, want %q", c.count, c.keys, c.at, got, res.Jump, c.want)
		}
	}
	// Off a timestamp <C-a> asks for nothing, so it adds to a number.
	if res := p.Key("<C-a>", "SCHEDULED: 3", 11, 0); res.Edits != nil || res.Jump != nil {
		t.Errorf("<C-a> off a timestamp asked for %+v", res)
	}
	run(t, []step{{"<S-Right>", "* a", 0, "* TODO a"}})
}
