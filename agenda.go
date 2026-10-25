package main

import (
	"cmp"
	"fmt"
	"strconv"
	"maps"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Entry is a heading the agenda may list: one with a TODO keyword, a date
// it is scheduled or due, or tags.
type Entry struct {
	Path, Keyword, Title, Priority string
	Tags                           []string // its own and those of the headings it is under
	Line                           int      // from 0
	Date                           *Stamp   // when it is due, or else scheduled, or nil
	Deadline, Done                 bool
}

// file is what the agenda last read of a file: when it changed then, its
// entries and its headings.
type file struct {
	changed time.Time
	entries []Entry
	heads   []Target
	ids     map[string]int // by ID, the line of the heading that has it
	text    string
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

// view is the agenda view shown last, which r and t show again.
var view = func() { week(7) }

// agenda gathers the org files under the directory option and shows the
// view shown last, the week at first.
func agenda() { gather(func() { view() }) }

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
	read[p] = file{f.Changed, entries(p, text), targets(p, text), ids(text), text}
	return nil
}

// entries returns the agenda's entries in text, the file at p, by the
// TODO keywords it sets. A heading has the tags of those it is under too.
func entries(p, text string) []Entry {
	t, d := keywordsIn(text, todo, done)
	keys := slices.Concat(t, d)
	lines := strings.Split(text, "\n")
	var tags [][]string // by level, the tags of the headings over the line
	var out []Entry
	for i, line := range lines {
		h, ok := parseHead(line, keys)
		if !ok {
			continue
		}
		tags = append(tags[:min(len(tags), h.Level-1)], h.Tags)
		e := Entry{Path: p, Line: i, Keyword: h.Keyword, Title: h.Title, Priority: h.Priority, Tags: slices.Concat(tags...), Done: slices.Contains(d, h.Keyword)}
		if i+1 < len(lines) {
			if plan, ok := parsePlan(lines[i+1]); ok {
				e.Date, e.Deadline = plan.Deadline, plan.Deadline != nil
				if e.Date == nil {
					e.Date = plan.Scheduled
				}
			}
		}
		if e.Keyword != "" || e.Date != nil || len(e.Tags) > 0 {
			out = append(out, e)
		}
	}
	return out
}

// all returns the entries of every file read, by file and line.
func all() []*Entry {
	var out []*Entry
	for _, p := range slices.Sorted(maps.Keys(read)) {
		for i := range read[p].entries {
			out = append(out, &read[p].entries[i])
		}
	}
	return out
}

// week shows what is overdue and not done, then each of the next days
// days with what is scheduled or due on it, a date that repeats on each
// day it repeats to.
func week(days int) {
	day := today()
	var lines []string
	var shown []*Entry
	add := func(title string, es []*Entry, timed bool) {
		lines, shown = append(lines, title), append(shown, nil)
		for _, e := range es {
			lines, shown = append(lines, "  "+describe(e, timed)), append(shown, e)
		}
	}
	var overdue []*Entry
	for _, e := range all() {
		if !e.Done && e.Date != nil && e.Date.Day.Before(day) && !repeatsTo(*e.Date, day) {
			overdue = append(overdue, e)
		}
	}
	if len(overdue) > 0 {
		add("Overdue", overdue, false)
	}
	for n := range days {
		d := day.AddDate(0, 0, n)
		var on []*Entry
		for _, e := range all() {
			if !e.Done && e.Date != nil && (e.Date.Day.Equal(d) || repeatsTo(*e.Date, d)) {
				on = append(on, e)
			}
		}
		slices.SortStableFunc(on, func(a, b *Entry) int { return strings.Compare(a.Date.Time, b.Date.Time) })
		add(d.Format("Mon 2006-01-02"), on, true)
	}
	panel("[agenda: "+strconv.Itoa(days)+" days]", lines, shown)
}

// repeatsTo reports whether st repeats to the day d, after its own.
func repeatsTo(st Stamp, d time.Time) bool {
	n, unit, ok := interval(st.Repeat)
	if !ok || n <= 0 {
		return false
	}
	for st = st.add(n, unit); !st.Day.After(d); st = st.add(n, unit) {
		if st.Day.Equal(d) {
			return true
		}
	}
	return false
}

// todos shows every heading with a TODO keyword that is not done, or with
// one of keywords when it names any, by priority.
func todos(keywords []string) {
	var es []*Entry
	for _, e := range all() {
		if e.Keyword != "" && !e.Done && (len(keywords) == 0 || slices.Contains(keywords, e.Keyword)) {
			es = append(es, e)
		}
	}
	slices.SortStableFunc(es, func(a, b *Entry) int { return strings.Compare(cmp.Or(a.Priority, "Z"), cmp.Or(b.Priority, "Z")) })
	var lines []string
	for _, e := range es {
		lines = append(lines, describe(e, false))
	}
	panel("[agenda: TODOs]", lines, es)
}

// search shows each line of the org files that holds text, any case, by
// file and line.
func search(text string) {
	needle := strings.ToLower(text)
	var es []*Entry
	var lines []string
	for _, p := range slices.Sorted(maps.Keys(read)) {
		for i, line := range strings.Split(read[p].text, "\n") {
			if strings.Contains(strings.ToLower(line), needle) {
				es = append(es, &Entry{Path: p, Line: i})
				lines = append(lines, path.Base(p)+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
		}
	}
	panel("[agenda: "+text+"]", lines, es)
}

// tagged shows each heading whose tags match expr, as matches reads it.
func tagged(expr string) {
	var es []*Entry
	var lines []string
	for _, e := range all() {
		if matches(e.Tags, expr) {
			es, lines = append(es, e), append(lines, describe(e, false)+"  :"+strings.Join(e.Tags, ":")+":")
		}
	}
	panel("[agenda: "+expr+"]", lines, es)
}

// matches reports whether tags match expr, as org matches tags. expr is
// groups separated by |, and one group must match. A group is tags, each
// with + before it to need it or - to forbid it, + when it has neither.
func matches(tags []string, expr string) bool {
	for _, group := range strings.Split(expr, "|") {
		terms, ok := 0, true
		for i := 0; i < len(group); {
			sign := byte('+')
			if group[i] == '+' || group[i] == '-' {
				sign, i = group[i], i+1
			}
			j := i + strings.IndexAny(group[i:]+"+", "+-")
			if name := strings.TrimSpace(group[i:j]); name != "" {
				terms++
				has := slices.ContainsFunc(tags, func(t string) bool { return strings.EqualFold(t, name) })
				ok = ok && has == (sign == '+')
			}
			i = j
		}
		if ok && terms > 0 {
			return true
		}
	}
	return false
}

// panel shows lines in the agenda's window under title, shown being the
// entry each line shows, nil for none.
func panel(title string, lines []string, shown []*Entry) {
	scan.shown = shown
	if len(lines) == 0 {
		lines = []string{"nothing to show"}
	}
	report(sdk.Do(api.Result{Show: &api.Panel{Title: title, Text: strings.Join(lines, "\n"), Unnumbered: true, Keys: map[string]string{
		"<CR>": "<Plug>(fude-org.agenda_open)",
		"t":    "<Plug>(fude-org.agenda_todo)",
		"r":    "<Plug>(fude-org.agenda_again)",
	}}}))
}

// describe returns e as a line of the agenda, with its time when timed or
// else its date, and its file.
func describe(e *Entry, timed bool) string {
	var parts []string
	switch {
	case e.Date != nil && timed && e.Date.Time != "":
		parts = append(parts, e.Date.Time)
	case e.Date != nil && !timed:
		parts = append(parts, e.Date.Day.Format(time.DateOnly))
	}
	if e.Deadline {
		parts = append(parts, "due")
	}
	if e.Keyword != "" {
		parts = append(parts, e.Keyword)
	}
	if e.Priority != "" {
		parts = append(parts, "[#"+e.Priority+"]")
	}
	return strings.Join(append(parts, e.Title, " "+path.Base(e.Path)), " ")
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
