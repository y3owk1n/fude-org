package main

import (
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Entry is a heading the agenda lists: a TODO one, or one scheduled or due
// that is not done.
type Entry struct {
	Path, Keyword, Title string
	Tags                 []string
	Line                 int    // from 0
	Date                 string // when it is scheduled or due, as 2006-01-02, or ""
	Deadline             bool
}

// file is what the agenda last read of a file: when it changed then, its
// entries and its headings.
type file struct {
	changed time.Time
	entries []Entry
	heads   []Target
}

// read keeps each file the agenda read, so the agenda reads a file again
// only once it changes.
var read = map[string]file{}

// scan is the agenda being gathered: the folders still to list, the files
// still to read, what runs once all are read, and what each line of the
// agenda shows once shown.
var scan struct {
	dirs, files []string
	then        func()
	shown       []*Entry
}

// slice is how long one step of gathering the agenda runs, well within the
// budget of a callback.
const slice = 40 * time.Millisecond

// agenda gathers the org files under the directory option and then shows
// their entries.
func agenda() { gather(show) }

// gather reads the org files under the directory option, a step in each
// callback, and then runs then.
func gather(then func()) {
	dir, _ := sdk.Option("fude-org.directory")
	scan.dirs, scan.files, scan.then = []string{dir}, nil, then
	report(sdk.After(0, step))
}

// step lists folders and reads files until its slice of time is up, then
// asks for the next step, or runs what gather was given once all are
// read.
func step() {
	until := sdk.Now().Add(slice)
	for sdk.Now().Before(until) {
		switch {
		case len(scan.dirs) > 0:
			dir := scan.dirs[0]
			scan.dirs = scan.dirs[1:]
			entries, err := sdk.List(dir)
			if err != nil {
				sdk.Do(api.Result{Error: err.Error()})
				return
			}
			for _, e := range entries {
				switch p := path.Join(dir, e.Name); {
				case strings.HasPrefix(e.Name, "."):
				case e.Dir:
					scan.dirs = append(scan.dirs, p)
				case strings.HasSuffix(e.Name, ".org"):
					scan.files = append(scan.files, p)
				}
			}
		case len(scan.files) > 0:
			p := scan.files[0]
			scan.files = scan.files[1:]
			if err := readFile(p); err != nil {
				sdk.Do(api.Result{Error: err.Error()})
				return
			}
		default:
			scan.then()
			return
		}
	}
	sdk.After(0, step)
}

// readFile reads the entries of the file at p, unless it has not changed
// since it was last read.
func readFile(p string) error {
	f, ok, err := sdk.Stat(p)
	if err != nil || !ok {
		return err
	}
	if was, ok := read[p]; ok && was.changed.Equal(f.Changed) {
		return nil
	}
	text, err := sdk.ReadFile(p)
	if err != nil {
		return fmt.Errorf("read %s: %w", p, err)
	}
	read[p] = file{f.Changed, entries(p, text), targets(p, text)}
	return nil
}

// entries returns the agenda's entries in text, the file at p, by the
// TODO keywords it sets.
func entries(p, text string) []Entry {
	t, d := keywordsIn(text, todo, done)
	keys := slices.Concat(t, d)
	lines := strings.Split(text, "\n")
	var out []Entry
	for i, line := range lines {
		h, ok := parseHead(line, keys)
		if !ok || slices.Contains(d, h.Keyword) {
			continue
		}
		e := Entry{Path: p, Line: i, Keyword: h.Keyword, Title: h.Title, Tags: h.Tags}
		if i+1 < len(lines) {
			if plan, ok := parsePlan(lines[i+1]); ok {
				switch {
				case plan.Deadline != nil:
					e.Date, e.Deadline = plan.Deadline.Day.Format(time.DateOnly), true
				case plan.Scheduled != nil:
					e.Date = plan.Scheduled.Day.Format(time.DateOnly)
				}
			}
		}
		if e.Keyword != "" || e.Date != "" {
			out = append(out, e)
		}
	}
	return out
}

// show shows the agenda: what is overdue, due today, in the week ahead and
// later, by date, and then the TODOs with no date.
func show() {
	var all []*Entry
	for _, p := range slices.Sorted(maps.Keys(read)) {
		for i := range read[p].entries {
			all = append(all, &read[p].entries[i])
		}
	}
	slices.SortStableFunc(all, func(a, b *Entry) int { return strings.Compare(a.Date, b.Date) })
	now := sdk.Now()
	today, week := now.Format(time.DateOnly), now.AddDate(0, 0, 7).Format(time.DateOnly)
	groups := []struct {
		name string
		in   func(e *Entry) bool
	}{
		{"Overdue", func(e *Entry) bool { return e.Date != "" && e.Date < today }},
		{"Today", func(e *Entry) bool { return e.Date == today }},
		{"This week", func(e *Entry) bool { return e.Date > today && e.Date <= week }},
		{"Later", func(e *Entry) bool { return e.Date > week }},
		{"No date", func(e *Entry) bool { return e.Date == "" }},
	}
	var lines []string
	scan.shown = nil
	for _, g := range groups {
		var in []*Entry
		for _, e := range all {
			if g.in(e) {
				in = append(in, e)
			}
		}
		if len(in) == 0 {
			continue
		}
		if len(lines) > 0 {
			lines, scan.shown = append(lines, ""), append(scan.shown, nil)
		}
		lines, scan.shown = append(lines, g.name), append(scan.shown, nil)
		for _, e := range in {
			lines, scan.shown = append(lines, "  "+describe(e)), append(scan.shown, e)
		}
	}
	if len(lines) == 0 {
		lines = []string{"nothing to do"}
	}
	sdk.Do(api.Result{Show: &api.Panel{Title: "[agenda]", Text: strings.Join(lines, "\n"), Unnumbered: true, Keys: map[string]string{
		"<CR>": "<Plug>(fude-org.agenda_open)",
		"t":    "<Plug>(fude-org.agenda_todo)",
		"r":    "<Plug>(fude-org.agenda)",
	}}})
}

// describe returns e as a line of the agenda.
func describe(e *Entry) string {
	var b strings.Builder
	if e.Date != "" {
		b.WriteString(e.Date + "  ")
		if e.Deadline {
			b.WriteString("due  ")
		}
	}
	if e.Keyword != "" {
		b.WriteString(e.Keyword + " ")
	}
	b.WriteString(e.Title + "  " + path.Base(e.Path))
	return b.String()
}

// shownAt returns the entry the agenda shows on the line that holds pos.
func shownAt(pos int) (*Entry, bool) {
	n, err := sdk.LineOf(pos)
	if err != nil || n >= len(scan.shown) || scan.shown[n] == nil {
		return nil, false
	}
	return scan.shown[n], true
}

// todoIn puts the next TODO keyword on e's heading in its file, as
// setKeyword does, and shows the agenda again.
func todoIn(e *Entry) error {
	text, err := sdk.ReadFile(e.Path)
	if err != nil {
		return fmt.Errorf("read %s: %w", e.Path, err)
	}
	lines := strings.SplitAfter(text, "\n")
	if e.Line >= len(lines) || level(lines[e.Line]) == 0 {
		return fmt.Errorf("%s changed since the agenda read it", e.Path)
	}
	start := len(strings.Join(lines[:e.Line], ""))
	t, d := keywordsIn(text, todo, done)
	f, used := parseFront(text[start:], slices.Concat(t, d))
	setKeyword(&f, nextKeyword(f.Head.Keyword, t, d, false), t, d, sdk.Now(), logDone())
	front := f.String()
	if !strings.HasSuffix(text[start:start+used], "\n") {
		front = strings.TrimSuffix(front, "\n")
	}
	if err := sdk.WriteFile(e.Path, text[:start]+front+text[start+used:]); err != nil {
		return err
	}
	agenda()
	return nil
}
