package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Head is a heading line taken apart.
type Head struct {
	Level    int
	Keyword  string // the TODO keyword, or ""
	Priority string // A, B or C, or ""
	Title    string
	Tags     []string
}

// parseHead takes the heading line text apart, with keys the TODO keywords
// it may start with. It is false for a line that is no heading.
func parseHead(text string, keys []string) (Head, bool) {
	l := level(text)
	if l == 0 {
		return Head{}, false
	}
	h, rest := Head{Level: l}, strings.TrimSpace(text[l:])
	if word, after, _ := strings.Cut(rest, " "); slices.Contains(keys, word) {
		h.Keyword, rest = word, strings.TrimSpace(after)
	}
	if len(rest) >= 4 && rest[:2] == "[#" && rest[3] == ']' {
		h.Priority, rest = rest[2:3], strings.TrimSpace(rest[4:])
	}
	i := strings.LastIndexAny(rest, " \t")
	if last := rest[i+1:]; len(last) > 2 && last[0] == ':' && last[len(last)-1] == ':' {
		h.Tags, rest = strings.Split(last[1:len(last)-1], ":"), strings.TrimSpace(rest[:i+1])
	}
	h.Title = rest
	return h, true
}

// String is the heading line h is.
func (h Head) String() string {
	parts := []string{strings.Repeat("*", h.Level)}
	if h.Keyword != "" {
		parts = append(parts, h.Keyword)
	}
	if h.Priority != "" {
		parts = append(parts, "[#"+h.Priority+"]")
	}
	if h.Title != "" {
		parts = append(parts, h.Title)
	}
	if len(h.Tags) > 0 {
		parts = append(parts, ":"+strings.Join(h.Tags, ":")+":")
	}
	return strings.Join(parts, " ")
}

// Stamp is an org timestamp, such as <2026-10-25 Sun 10:00 +1w>.
type Stamp struct {
	Active bool
	Day    time.Time // the date, at midnight UTC
	Time   string    // such as 10:00 or 10:00-11:00, or ""
	Repeat string    // such as +1w, ++1d or .+2m, or ""
	Warn   string    // such as -3d, or ""
}

// parseStamp reads the timestamp s starts with, and returns how much of s
// it took. It is false when s starts with none.
func parseStamp(s string) (Stamp, int, bool) {
	if len(s) < 12 || s[0] != '<' && s[0] != '[' {
		return Stamp{}, 0, false
	}
	closing := map[byte]byte{'<': '>', '[': ']'}[s[0]]
	end := strings.IndexByte(s, closing)
	if end < 0 {
		return Stamp{}, 0, false
	}
	day, err := time.Parse(time.DateOnly, s[1:11])
	if err != nil {
		return Stamp{}, 0, false
	}
	st := Stamp{Active: s[0] == '<', Day: day}
	for _, word := range strings.Fields(s[11:end]) {
		switch {
		case word[0] >= '0' && word[0] <= '9':
			st.Time = word
		case strings.HasPrefix(word, "-"):
			st.Warn = word
		case strings.ContainsAny(word[:1], "+."):
			st.Repeat = word
		}
	}
	return st, end + 1, true
}

// String is the timestamp st is, with its day's name.
func (st Stamp) String() string {
	parts := []string{st.Day.Format("2006-01-02 Mon")}
	for _, p := range []string{st.Time, st.Repeat, st.Warn} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	if st.Active {
		return "<" + strings.Join(parts, " ") + ">"
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// add returns st n units later, a unit being h, d, w, m or y. An hour moves
// its time, and the day with it past midnight.
func (st Stamp) add(n int, unit byte) Stamp {
	switch unit {
	case 'h':
		if st.Time == "" {
			return st
		}
		at := st.at().Add(time.Duration(n) * time.Hour)
		st.Day = time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
		st.Time = at.Format("15:04") + st.Time[min(5, len(st.Time)):]
	case 'd':
		st.Day = st.Day.AddDate(0, 0, n)
	case 'w':
		st.Day = st.Day.AddDate(0, 0, 7*n)
	case 'm':
		st.Day = st.Day.AddDate(0, n, 0)
	case 'y':
		st.Day = st.Day.AddDate(n, 0, 0)
	}
	return st
}

// at returns the time st stands for, its day at its time.
func (st Stamp) at() time.Time {
	t, err := time.Parse("15:04", st.Time[:min(5, len(st.Time))])
	if err != nil {
		return st.Day
	}
	return st.Day.Add(time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute)
}

// interval reads an interval such as 2w, or the 1w of a repeater such as
// +1w, as how many of which unit.
func interval(s string) (int, byte, bool) {
	s = strings.TrimLeft(s, "+.-")
	if len(s) < 2 || !strings.ContainsRune("hdwmy", rune(s[len(s)-1])) {
		return 0, 0, false
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	return n, s[len(s)-1], err == nil
}

// Plan is the planning line under a heading: when it closed, is due and is
// scheduled.
type Plan struct {
	Closed, Deadline, Scheduled *Stamp
}

// planWords are the planning line's keywords, in the order org writes them.
var planWords = []string{"CLOSED:", "DEADLINE:", "SCHEDULED:"}

// parsePlan reads a planning line, which holds nothing but its keywords
// and their timestamps. It is false for any other line.
func parsePlan(line string) (Plan, bool) {
	var p Plan
	rest := strings.TrimSpace(line)
	if rest == "" {
		return p, false
	}
	for rest != "" {
		i := slices.IndexFunc(planWords, func(w string) bool { return strings.HasPrefix(rest, w) })
		if i < 0 {
			return Plan{}, false
		}
		st, n, ok := parseStamp(strings.TrimSpace(rest[len(planWords[i]):]))
		if !ok {
			return Plan{}, false
		}
		*p.at(i) = &st
		after := strings.TrimSpace(rest[len(planWords[i]):])
		rest = strings.TrimSpace(after[n:])
	}
	return p, true
}

// at returns where p keeps the stamp of planWords[i].
func (p *Plan) at(i int) **Stamp {
	return [...]**Stamp{&p.Closed, &p.Deadline, &p.Scheduled}[i]
}

// String is the planning line p is, or "" when p holds nothing.
func (p Plan) String() string {
	var parts []string
	for i, w := range planWords {
		if st := *p.at(i); st != nil {
			parts = append(parts, w+" "+st.String())
		}
	}
	return strings.Join(parts, " ")
}

// Front is the top of a section, which org edits as one: the heading line,
// its planning line and its property drawer.
type Front struct {
	Head  Head
	Plan  Plan
	Props [][2]string // by name, in their order
}

// parseFront reads the front of the section text starts, with keys the
// TODO keywords, and returns how much of text it took, its newline too.
func parseFront(text string, keys []string) (Front, int) {
	lines := strings.SplitAfter(text, "\n")
	f := Front{}
	f.Head, _ = parseHead(strings.TrimRight(lines[0], "\n"), keys)
	n, i := len(lines[0]), 1
	if i < len(lines) {
		if p, ok := parsePlan(strings.TrimRight(lines[i], "\n")); ok {
			f.Plan, n, i = p, n+len(lines[i]), i+1
		}
	}
	if i < len(lines) && strings.TrimSpace(lines[i]) == ":PROPERTIES:" {
		for j := i + 1; j < len(lines); j++ {
			line := strings.TrimSpace(lines[j])
			if line == ":END:" {
				for _, l := range lines[i : j+1] {
					n += len(l)
				}
				return f, n
			}
			name, value, _ := strings.Cut(strings.TrimPrefix(line, ":"), ":")
			f.Props = append(f.Props, [2]string{name, strings.TrimSpace(value)})
		}
		// A drawer with no :END: is no drawer.
		f.Props = nil
	}
	return f, n
}

// String is the front f is, a newline ending each line.
func (f Front) String() string {
	var b strings.Builder
	b.WriteString(f.Head.String() + "\n")
	if p := f.Plan.String(); p != "" {
		b.WriteString(p + "\n")
	}
	if len(f.Props) > 0 {
		b.WriteString(":PROPERTIES:\n")
		for _, p := range f.Props {
			fmt.Fprintf(&b, ":%s: %s\n", p[0], p[1])
		}
		b.WriteString(":END:\n")
	}
	return b.String()
}

// prop returns the value of the property name, or "".
func (f Front) prop(name string) string {
	for _, p := range f.Props {
		if strings.EqualFold(p[0], name) {
			return p[1]
		}
	}
	return ""
}

// setProp sets the property name to value, or removes it for "".
func (f *Front) setProp(name, value string) {
	i := slices.IndexFunc(f.Props, func(p [2]string) bool { return strings.EqualFold(p[0], name) })
	switch {
	case i >= 0 && value == "":
		f.Props = slices.Delete(f.Props, i, i+1)
	case i >= 0:
		f.Props[i][1] = value
	case value != "":
		f.Props = append(f.Props, [2]string{name, value})
	}
}

// keywordsIn returns the TODO keywords a file's #+TODO lines set before its
// first heading, or todo and done when it sets none.
func keywordsIn(text string, todo, done []string) ([]string, []string) {
	var t, d []string
	for line := range strings.Lines(text) {
		if level(line) > 0 {
			break
		}
		upper := strings.ToUpper(line)
		for _, key := range []string{"#+TODO:", "#+SEQ_TODO:", "#+TYP_TODO:"} {
			if strings.HasPrefix(upper, key) {
				before, after, found := strings.Cut(line[len(key):], "|")
				words, doneWords := strings.Fields(before), strings.Fields(after)
				if !found && len(words) > 1 {
					words, doneWords = words[:len(words)-1], words[len(words)-1:]
				}
				t, d = append(t, keyNames(words)...), append(d, keyNames(doneWords)...)
			}
		}
	}
	if len(t) == 0 && len(d) == 0 {
		return todo, done
	}
	return t, d
}

// keyNames drops the fast-access letters a keyword may carry, as in
// TODO(t) or DONE(d!).
func keyNames(words []string) []string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i], _, _ = strings.Cut(w, "(")
	}
	return out
}

// weekdays are the names a date may be typed as, by their first three
// letters.
var weekdays = []string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// parseDate reads a date typed as org reads one, counting from today. The
// date is 2026-10-25, +3d, -1w, today, tomorrow, or a weekday, the nearest
// one from today. A time such as 10:00, a repeater such as +1w and a
// warning such as -2d may follow it.
func parseDate(text string, today time.Time) (Stamp, error) {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return Stamp{}, errors.New("no date")
	}
	st := Stamp{Active: true}
	switch w := words[0]; {
	case w == "today" || w == ".":
		st.Day = today
	case w == "tomorrow":
		st.Day = today.AddDate(0, 0, 1)
	case w[0] == '+' || w[0] == '-':
		if strings.Trim(w[1:], "0123456789") == "" {
			w += "d"
		}
		n, unit, ok := interval(w)
		if !ok {
			return Stamp{}, fmt.Errorf("%s is no date", words[0])
		}
		if w[0] == '-' {
			n = -n
		}
		st = Stamp{Active: true, Day: today}.add(n, unit)
	case slices.Contains(weekdays, w[:min(3, len(w))]):
		want := time.Weekday(slices.Index(weekdays, w[:3]))
		st.Day = today.AddDate(0, 0, (int(want)-int(today.Weekday())+7)%7)
	default:
		day, err := time.Parse(time.DateOnly, w)
		if err != nil {
			return Stamp{}, fmt.Errorf("%s is no date", words[0])
		}
		st.Day = day
	}
	for _, w := range words[1:] {
		switch {
		case strings.Contains(w, ":"):
			st.Time = w
		case w[0] == '-':
			st.Warn = w
		case w[0] == '+' || w[0] == '.':
			st.Repeat = w
		default:
			return Stamp{}, fmt.Errorf("%s is no time, repeater or warning", w)
		}
	}
	return st, nil
}

// typed returns st as parseDate reads it back.
func (st Stamp) typed() string {
	parts := []string{st.Day.Format(time.DateOnly)}
	for _, p := range []string{st.Time, st.Repeat, st.Warn} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " ")
}

// stampAt returns the timestamp in line that holds col, and where it
// starts and ends.
func stampAt(line string, col int) (Stamp, int, int, bool) {
	for i := 0; i < len(line); i++ {
		if line[i] != '<' && line[i] != '[' {
			continue
		}
		if st, n, ok := parseStamp(line[i:]); ok {
			if col >= i && col < i+n {
				return st, i, i + n, true
			}
			i += n - 1
		}
	}
	return Stamp{}, 0, 0, false
}

// unitAt returns the unit of the part of the timestamp's text at rel, as
// String writes it: y, m or d in its date, h in its time, and d elsewhere.
func (st Stamp) unitAt(rel int) byte {
	switch {
	case rel <= 4:
		return 'y'
	case rel <= 7:
		return 'm'
	case st.Time != "" && rel > 15 && rel <= 15+len(st.Time):
		return 'h'
	}
	return 'd'
}

// nextKeyword returns the keyword after cur in todo and done, or the one
// before with back, in their order and then none.
func nextKeyword(cur string, todo, done []string, back bool) string {
	keys := slices.Concat(todo, done)
	at := slices.Index(keys, cur)
	if at < 0 {
		at = len(keys)
	}
	step := 1
	if back {
		step = len(keys)
	}
	if next := (at + step) % (len(keys) + 1); next < len(keys) {
		return keys[next]
	}
	return ""
}

// setKeyword puts keyword on f as org does, at now, the local time. A done
// keyword closes f when log is set. When a date of f's planning line
// repeats, setKeyword moves that date on instead, keeps f's TODO keyword
// and notes the time in LAST_REPEAT. A keyword that is not done opens f
// again.
func setKeyword(f *Front, keyword string, todo, done []string, now time.Time, log bool) {
	was := f.Head.Keyword
	f.Head.Keyword = keyword
	at := Stamp{Day: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), Time: now.Format("15:04")}
	switch {
	case !slices.Contains(done, keyword):
		f.Plan.Closed = nil
	case slices.Contains(done, was):
	case repeat(&f.Plan, at.Day):
		f.Head.Keyword = was
		if !slices.Contains(todo, was) && len(todo) > 0 {
			f.Head.Keyword = todo[0]
		}
		f.setProp("LAST_REPEAT", at.String())
	case log:
		f.Plan.Closed = &at
	}
}

// repeat moves on each date of p that repeats, from today, and reports
// whether one did. +1w moves a date a week, ++1w as many weeks as take it
// past today, and .+1w to a week from today.
func repeat(p *Plan, today time.Time) bool {
	moved := false
	for _, st := range []*Stamp{p.Deadline, p.Scheduled} {
		if st == nil || st.Repeat == "" {
			continue
		}
		n, unit, ok := interval(st.Repeat)
		if !ok || n <= 0 {
			continue
		}
		switch {
		case strings.HasPrefix(st.Repeat, "++"):
			for *st = st.add(n, unit); !st.Day.After(today); *st = st.add(n, unit) {
			}
		case strings.HasPrefix(st.Repeat, ".+"):
			st.Day = today
			*st = st.add(n, unit)
		default:
			*st = st.add(n, unit)
		}
		moved = true
	}
	return moved
}

// cookie returns where the statistics cookie in s, such as [2/3], [/],
// [66%] or [%], starts and ends, and whether it counts in percent.
func cookie(s string) (int, int, bool, bool) {
	for i := strings.IndexByte(s, '['); i >= 0; {
		end := strings.IndexByte(s[i:], ']')
		if end < 0 {
			break
		}
		inside := s[i+1 : i+end]
		digits := strings.Trim(inside, "0123456789")
		if digits == "/" || digits == "%" && strings.HasSuffix(inside, "%") {
			return i, i + end + 1, digits == "%", true
		}
		next := strings.IndexByte(s[i+1:], '[')
		if next < 0 {
			break
		}
		i += next + 1
	}
	return 0, 0, false, false
}

// recount writes the counts of the cookies in the subtree text of a
// heading of level. A heading's cookie counts the checkboxes in its own
// text, or else its children's TODO keywords. A list item's counts the
// checkboxes of the items under it.
func recount(text string, lvl int, todo, done []string) string {
	lines := strings.Split(text, "\n")
	own := len(lines)
	for i, l := range lines[1:] {
		if level(l) > 0 {
			own = i + 1
			break
		}
	}
	keys := slices.Concat(todo, done)
	count := func(i int) (int, int) {
		if i == 0 {
			if n, total := boxes(lines[1:own], -1); total > 0 {
				return n, total
			}
			n, total := 0, 0
			for _, l := range lines[own:] {
				if h, ok := parseHead(l, keys); ok && h.Level == lvl+1 && h.Keyword != "" {
					total++
					if slices.Contains(done, h.Keyword) {
						n++
					}
				}
			}
			return n, total
		}
		it, _ := parseItem(lines[i])
		return boxes(lines[i+1:own], len(it.Indent))
	}
	for i := range own {
		start, end, percent, ok := cookie(lines[i])
		if _, item := parseItem(lines[i]); !ok || i > 0 && !item {
			continue
		}
		n, total := count(i)
		c := fmt.Sprintf("[%d/%d]", n, total)
		if percent {
			c = fmt.Sprintf("[%d%%]", n*100/max(total, 1))
		}
		lines[i] = lines[i][:start] + c + lines[i][end:]
	}
	return strings.Join(lines, "\n")
}

// boxes counts the ticked checkboxes and all checkboxes among the items in
// lines. With indent -1 it counts every item. Otherwise it counts the
// items nearest under an item indented by indent, up to the first line
// indented no more than it.
func boxes(lines []string, indent int) (int, int) {
	n, total, under := 0, 0, -1
	for _, l := range lines {
		rest := strings.TrimLeft(l, " \t")
		if rest == "" {
			continue
		}
		depth := len(l) - len(rest)
		if indent >= 0 && depth <= indent {
			break
		}
		it, ok := parseItem(l)
		if !ok || !it.Box || indent >= 0 && under >= 0 && depth != under {
			continue
		}
		under = depth
		total++
		if strings.HasPrefix(rest[len(it.Bullet)+1:], "[X]") || strings.HasPrefix(rest[len(it.Bullet)+1:], "[x]") {
			n++
		}
	}
	return n, total
}
