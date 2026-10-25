package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// running is the clock running: the file and title of its heading and
// when it started, or none.
var running *struct {
	Path, Title string
	Started     time.Time
}

// stampNow returns now as an inactive timestamp with its time.
func stampNow() Stamp {
	return Stamp{Day: today(), Time: sdk.Now().Format("15:04")}
}

// hours writes d as org writes a clocked time, such as 1:05.
func hours(d time.Duration) string {
	m := int(d.Round(time.Minute) / time.Minute)
	return fmt.Sprintf("%d:%02d", m/60, m%60)
}

// clockLine reads a CLOCK: line: when it started, when it ended and
// whether it has. It is false for any other line.
func clockLine(line string) (Stamp, Stamp, bool, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), "CLOCK:")
	if !ok {
		return Stamp{}, Stamp{}, false, false
	}
	start, n, ok := parseStamp(strings.TrimSpace(rest))
	if !ok {
		return Stamp{}, Stamp{}, false, false
	}
	after, closed := strings.CutPrefix(strings.TrimSpace(rest)[n:], "--")
	if !closed {
		return start, Stamp{}, false, true
	}
	end, _, ok := parseStamp(after)
	return start, end, ok, true
}

// clockIn clocks in the heading of the section that holds cursor, after
// clocking out the clock running. It adds CLOCK: and now at the top of the
// heading's LOGBOOK drawer, which it makes under the front when there is
// none.
func clockIn(cursor int) error {
	h, ok := owner(cursor)
	if !ok {
		return errors.New("no heading here")
	}
	if running != nil {
		if err := clockOut(false); err != nil {
			return err
		}
	}
	f, end := front(h)
	line := "CLOCK: " + stampNow().String() + "\n"
	n, _ := sdk.Len()
	l, _ := sdk.LineOf(end)
	if _, next := lineAt(l); end < n && strings.TrimSpace(next) == ":LOGBOOK:" {
		at, _ := sdk.LineStart(l + 1)
		edit(api.Edit{Start: at, End: at, Text: line})
	} else {
		text := ":LOGBOOK:\n" + line + ":END:\n"
		if last, _ := sdk.Text(end-1, end); last != "\n" {
			text = "\n" + strings.TrimSuffix(text, "\n")
		}
		edit(api.Edit{Start: end, End: end, Text: text})
	}
	running = &struct {
		Path, Title string
		Started     time.Time
	}{here(), f.Head.Title, sdk.Now()}
	showClock()
	return sdk.After(60_000, tick)
}

// tick shows the clock running again each minute.
func tick() {
	if running != nil {
		showClock()
		sdk.After(60_000, tick)
	}
}

// showClock shows the clock running on the status line, or nothing.
func showClock() {
	text := ""
	if running != nil {
		text = "⏱ " + hours(sdk.Now().Sub(running.Started)) + " " + running.Title
	}
	sdk.Status(text)
}

// clockOut closes the clock running, in its file or the buffer, with now
// and how long it ran. With cancel it removes the clock's line instead.
// With no clock running, it closes the buffer's open clock.
func clockOut(cancel bool) error {
	if running != nil && running.Path != here() {
		p := running.Path
		text, err := sdk.ReadFile(p)
		if err != nil {
			return fmt.Errorf("read %s: %w", p, err)
		}
		out, err := closeClock(text, cancel)
		if err != nil {
			return err
		}
		running = nil
		showClock()
		return sdk.WriteFile(p, out)
	}
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	out, err := closeClock(text, cancel)
	if err != nil {
		return err
	}
	running = nil
	showClock()
	if e, ok := diffEdit(text, out, 0); ok {
		edit(e)
	}
	return nil
}

// closeClock closes the open CLOCK: line of text with now. With cancel it
// removes the line, and the LOGBOOK drawer when that leaves it empty.
func closeClock(text string, cancel bool) (string, error) {
	lines := strings.Split(text, "\n")
	i := slices.IndexFunc(lines, func(l string) bool {
		_, _, ended, ok := clockLine(l)
		return ok && !ended
	})
	if i < 0 {
		return "", errors.New("no clock is running")
	}
	start, _, _, _ := clockLine(lines[i])
	if cancel {
		lines = slices.Delete(lines, i, i+1)
		if i > 0 && i < len(lines) && strings.TrimSpace(lines[i-1]) == ":LOGBOOK:" && strings.TrimSpace(lines[i]) == ":END:" {
			lines = slices.Delete(lines, i-1, i+1)
		}
		return strings.Join(lines, "\n"), nil
	}
	end := stampNow()
	indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], " \t"))]
	lines[i] = indent + "CLOCK: " + start.String() + "--" + end.String() + " =>  " + hours(end.at().Sub(start.at()))
	return strings.Join(lines, "\n"), nil
}

// clockGoto goes to the heading the clock runs on.
func clockGoto() error {
	if running == nil {
		return errors.New("no clock is running")
	}
	return open("file:" + running.Path + "::*" + running.Title)
}

// clockTable puts a clock table of the buffer at the cursor, or writes the
// one the cursor is in again. It lists the total, then each heading with
// time clocked in it or under it, and how long.
func clockTable(cursor int) {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	lines := strings.Split(text, "\n")
	at, _ := sdk.LineOf(cursor)
	first, last := at, at-1
	for l := at; l >= 0; l-- {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(lines[l])), "#+begin: clocktable") {
			for e := l; e < len(lines); e++ {
				if strings.EqualFold(strings.TrimSpace(lines[e]), "#+end:") {
					first, last = l, e
					break
				}
			}
			break
		}
		if l < at && strings.EqualFold(strings.TrimSpace(lines[l]), "#+end:") {
			break
		}
	}
	block := strings.Join(clockRows(lines), "\n")
	start := lineStart(text, first)
	end := start
	if last >= first {
		end = lineStart(text, last) + len(lines[last])
	} else {
		block += "\n"
	}
	edit(api.Edit{Start: start, End: end, Text: block})
}

// clockRows returns the lines of a clock table of the text lines, from
// #+BEGIN: to #+END:.
func clockRows(lines []string) []string {
	t, d := keywordsIn(strings.Join(lines, "\n"), todo, done)
	keys := slices.Concat(t, d)
	type row struct {
		level int
		title string
		own   time.Duration
	}
	var rows []row
	for _, l := range lines {
		if h, ok := parseHead(l, keys); ok {
			rows = append(rows, row{h.Level, h.Title, 0})
			continue
		}
		if start, end, ended, ok := clockLine(l); ok && ended && len(rows) > 0 {
			rows[len(rows)-1].own += end.at().Sub(start.at())
		}
	}
	// A heading's time is its own and that of the headings under it.
	total := make([]time.Duration, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		total[i] += rows[i].own
		for j := i - 1; j >= 0; j-- {
			if rows[j].level < rows[i].level {
				total[j] += total[i]
				break
			}
		}
	}
	all := time.Duration(0)
	var body []string
	for i, r := range rows {
		if r.level == 1 {
			all += total[i]
		}
		if total[i] > 0 {
			body = append(body, "|"+strings.Repeat(`\_ `, r.level-1)+r.title+"|"+hours(total[i])+"|")
		}
	}
	table := append([]string{"|Heading|Time|", "|-", "|*Total time*|*" + hours(all) + "*|", "|-"}, body...)
	return slices.Concat([]string{"#+BEGIN: clocktable :scope file"}, parseTable(table).Lines(), []string{"#+END:"})
}
