package main

import (
	"errors"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Heading is a heading's line: its number, where it starts, how many stars
// it has, and its text.
type Heading struct {
	Line, Start, Level int
	Text               string
}

// lineAt returns where line n starts and its text, without its newline.
func lineAt(n int) (int, string) {
	start, _ := sdk.LineStart(n)
	end, _ := sdk.Len()
	if lines, _ := sdk.Lines(); n+1 < lines {
		next, _ := sdk.LineStart(n + 1)
		end = next - 1
	}
	text, _ := sdk.Text(start, end)
	return start, text
}

// level returns how many stars start a heading's line, or 0 for any other
// line. The stars are followed by a blank or end the line.
func level(text string) int {
	n := len(text) - len(strings.TrimLeft(text, "*"))
	if n == 0 || n < len(text) && text[n] != ' ' && text[n] != '\t' {
		return 0
	}
	return n
}

// heading returns the heading on the line that holds pos.
func heading(pos int) (Heading, bool) {
	n, err := sdk.LineOf(pos)
	if err != nil {
		return Heading{}, false
	}
	return headingAt(n)
}

// headingAt returns the heading on line n.
func headingAt(n int) (Heading, bool) {
	start, text := lineAt(n)
	l := level(text)
	return Heading{n, start, l, text}, l > 0
}

// subtree returns where h's subtree ends: at the next heading of h's
// level or above, or at the end of the text.
func subtree(h Heading) int { return until(h, h.Level) }

// sectionEnd returns where h's own text ends: at the next heading, or at
// the end of the text.
func sectionEnd(h Heading) int { return until(h, math.MaxInt) }

// until returns where the next heading after h of level top or above
// starts, or the end of the text.
func until(h Heading, top int) int {
	n, _ := sdk.Len()
	rest, _ := sdk.Text(h.Start, n)
	at := strings.IndexByte(rest, '\n')
	for at >= 0 {
		line, _, _ := strings.Cut(rest[at+1:], "\n")
		if l := level(line); l > 0 && l <= top {
			return h.Start + at + 1
		}
		next := strings.IndexByte(rest[at+1:], '\n')
		if next < 0 {
			break
		}
		at += next + 1
	}
	return n
}

// front reads the front of h's section, and returns it and where it ends.
func front(h Heading) (Front, int) {
	text, _ := sdk.Text(h.Start, sectionEnd(h))
	t, d := bufferKeys()
	f, used := parseFront(text, slices.Concat(t, d))
	return f, h.Start + used
}

// putFront asks for f in place of h's front, which ends at end.
func putFront(h Heading, f Front, end int) {
	edit(api.Edit{Start: h.Start, End: end, Text: frontText(f, end)})
}

// frontText returns f as the text of a front that ends at end. A front at
// the end of the text may end with no newline.
func frontText(f Front, end int) string {
	if last, _ := sdk.Text(end-1, end); last != "\n" {
		return strings.TrimSuffix(f.String(), "\n")
	}
	return f.String()
}

// counted asks for h's subtree as change makes it, with its counts
// written again, and keeps the cursor where it was.
func counted(cursor int, h Heading, change func(sub string) string) {
	end := subtree(h)
	old, _ := sdk.Text(h.Start, end)
	t, d := bufferKeys()
	text := recount(change(old), h.Level, t, d)
	e, ok := diffEdit(old, text, h.Start)
	if !ok {
		return
	}
	// The cursor keeps its line and column, as a count changes lines in
	// place and a done keyword adds them after the heading.
	before := old[:cursor-h.Start]
	line, col := strings.Count(before, "\n"), len(before)-strings.LastIndexByte(before, '\n')-1
	at := 0
	for range line {
		at += strings.IndexByte(text[at:], '\n') + 1
	}
	rest, _, _ := strings.Cut(text[at:], "\n")
	sdk.Do(api.Result{Edits: []api.Edit{e}, Jump: &api.Jump{Pos: h.Start + at + min(col, len(rest))}})
}

// diffEdit returns the edit that makes old new, which starts at base: of
// what lies between what they share at their starts and at their ends. It
// is false when they are the same.
func diffEdit(old, new string, base int) (api.Edit, bool) {
	if old == new {
		return api.Edit{}, false
	}
	p := 0
	for p < len(old) && p < len(new) && old[p] == new[p] {
		p++
	}
	s := 0
	for s < len(old)-p && s < len(new)-p && old[len(old)-1-s] == new[len(new)-1-s] {
		s++
	}
	return api.Edit{Start: base + p, End: base + len(old) - s, Text: new[p : len(new)-s]}, true
}

// changeFront runs change on the front of the section that holds pos, and
// asks for what it made of it.
func changeFront(pos int, change func(f *Front)) {
	h, ok := owner(pos)
	if !ok {
		sdk.Do(api.Result{Error: "no heading here"})
		return
	}
	f, end := front(h)
	change(&f)
	putFront(h, f, end)
}

// owner returns the heading whose section holds pos: the one on its line,
// or the nearest above it.
func owner(pos int) (Heading, bool) {
	n, err := sdk.LineOf(pos)
	if err != nil {
		return Heading{}, false
	}
	for ; n >= 0; n-- {
		if h, ok := headingAt(n); ok {
			return h, true
		}
	}
	return Heading{}, false
}

// sibling returns the heading of h's level next to h's subtree, above it
// or below, within the same parent.
func sibling(h Heading, below bool) (Heading, bool) {
	if below {
		end := subtree(h)
		n, _ := sdk.Len()
		if end >= n {
			return Heading{}, false
		}
		line, _ := sdk.LineOf(end)
		s, ok := headingAt(line)
		return s, ok && s.Level == h.Level
	}
	for n := h.Line - 1; n >= 0; n-- {
		if s, ok := headingAt(n); ok {
			if s.Level == h.Level {
				return s, true
			}
			if s.Level < h.Level {
				break
			}
		}
	}
	return Heading{}, false
}

// cycleTodo puts the next keyword on the heading h, or the one before with
// back, as setKeyword does, and counts its parent's children again.
func cycleTodo(cursor int, h Heading, back bool) {
	t, d := bufferKeys()
	f, end := front(h)
	setKeyword(&f, nextKeyword(f.Head.Keyword, t, d, back), t, d, sdk.Now(), logDone())
	p, ok := parent(h)
	if !ok {
		putFront(h, f, end)
		return
	}
	text := frontText(f, end)
	counted(cursor, p, func(sub string) string {
		return sub[:h.Start-p.Start] + text + sub[end-p.Start:]
	})
}

// logDone reports whether a heading that is done gets a CLOSED date.
func logDone() bool {
	v, _ := sdk.Option("fude-org.log_done")
	return v != "false"
}

// bufferKeys returns the TODO keywords of the buffer the call is about, as
// its #+TODO lines or the todo option set them.
func bufferKeys() ([]string, []string) {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, min(n, 8192))
	return keywordsIn(text, todo, done)
}

// shift adds a star to each heading from start to end, or takes one away
// with up. A heading of one star cannot go up.
func shift(start, end int, up bool) ([]api.Edit, error) {
	text, _ := sdk.Text(start, end)
	var edits []api.Edit
	at := start
	for line := range strings.Lines(text) {
		l := level(line)
		switch {
		case l == 0:
		case up && l == 1:
			return nil, errors.New("a heading of one star has no level above it")
		case up:
			edits = append(edits, api.Edit{Start: at, End: at + 1})
		default:
			edits = append(edits, api.Edit{Start: at, End: at, Text: "*"})
		}
		at += len(line)
	}
	return edits, nil
}

// checkbox returns the edit that ticks the checkbox of the list item on the
// line that starts at start, or clears a ticked one.
func checkbox(start int, text string) (api.Edit, bool) {
	rest := strings.TrimLeft(text, " \t")
	bullet, after, ok := strings.Cut(rest, " ")
	if !ok || !isBullet(bullet) || !strings.HasPrefix(after, "[") || len(after) < 3 || after[2] != ']' {
		return api.Edit{}, false
	}
	at := start + len(text) - len(after) + 1
	mark := "X"
	if after[1] == 'X' || after[1] == 'x' {
		mark = " "
	}
	return api.Edit{Start: at, End: at + 1, Text: mark}, true
}

// isBullet reports whether s starts a list item: -, + or a number with .
// or ) after it.
func isBullet(s string) bool {
	if s == "-" || s == "+" {
		return true
	}
	digits := strings.TrimRight(s, ".)")
	return len(digits) == len(s)-1 && digits != "" && strings.Trim(digits, "0123456789") == ""
}

// headingFrom returns the n-th heading below line, or above it with up.
func headingFrom(line, n int, up bool) (Heading, bool) {
	lines, _ := sdk.Lines()
	step := 1
	if up {
		step = -1
	}
	for l := line + step; l >= 0 && l < lines; l += step {
		if h, ok := headingAt(l); ok {
			if n--; n == 0 {
				return h, true
			}
		}
	}
	return Heading{}, false
}

// parent returns the heading h is under.
func parent(h Heading) (Heading, bool) {
	for n := h.Line - 1; n >= 0; n-- {
		if p, ok := headingAt(n); ok && p.Level < h.Level {
			return p, true
		}
	}
	return Heading{}, false
}

// Item is a list item's line taken apart.
type Item struct {
	Indent, Bullet string
	Box            bool // it has a checkbox
}

// parseItem takes a list item's line apart. It is false for any other line.
func parseItem(line string) (Item, bool) {
	rest := strings.TrimLeft(line, " \t")
	bullet, after, ok := strings.Cut(rest, " ")
	if !ok || !isBullet(bullet) {
		return Item{}, false
	}
	return Item{line[:len(line)-len(rest)], bullet, strings.HasPrefix(after, "[") && len(after) > 2 && after[2] == ']'}, true
}

// next returns the line that starts the item after it: its bullet, a
// number one more, and an empty checkbox when it has one.
func (it Item) next() string {
	bullet := it.Bullet
	if digits := strings.TrimRight(bullet, ".)"); digits != bullet {
		n, _ := strconv.Atoi(digits)
		bullet = strconv.Itoa(n+1) + bullet[len(digits):]
	}
	line := it.Indent + bullet + " "
	if it.Box {
		line += "[ ] "
	}
	return line
}

// itemEnd returns where the item that starts on line n ends: after the
// lines under it indented more than it.
func itemEnd(n int, it Item) int {
	lines, _ := sdk.Lines()
	last := n
	for l := n + 1; l < lines; l++ {
		_, text := lineAt(l)
		rest := strings.TrimLeft(text, " \t")
		if rest == "" || len(text)-len(rest) <= len(it.Indent) {
			break
		}
		last = l
	}
	start, text := lineAt(last)
	return start + len(text)
}

// insertLine asks for line on a line of its own after the end of a line at
// at, the cursor at its end in insert mode.
func insertLine(at int, line string) {
	sdk.Do(api.Result{
		Edits: []api.Edit{{Start: at, End: at, Text: "\n" + line}},
		Jump:  &api.Jump{Pos: at + 1 + len(line)},
		Keys:  "a",
	})
}

// shiftStamp moves the timestamp under the cursor by n of the unit of the
// part under it, or by n days with days, and keeps the cursor where it
// was. It is false off a timestamp.
func shiftStamp(cursor, n int, days bool) bool {
	l, _ := sdk.LineOf(cursor)
	start, text := lineAt(l)
	st, from, to, ok := stampAt(text, cursor-start)
	if !ok {
		return false
	}
	unit := byte('d')
	if !days {
		unit = st.unitAt(cursor - start - from)
	}
	sdk.Do(api.Result{Edits: []api.Edit{{Start: start + from, End: start + to, Text: st.add(n, unit).String()}}, Jump: &api.Jump{Pos: cursor}})
	return true
}

// today returns the local date, at midnight UTC as a Stamp's day is.
func today() time.Time {
	now := sdk.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// setDate asks for the date of the section's planning keyword at i of
// planWords, starting from the one it has, and sets it, or removes it for
// none.
func setDate(cursor, i int) {
	h, ok := owner(cursor)
	if !ok {
		sdk.Do(api.Result{Error: "no heading here"})
		return
	}
	f, _ := front(h)
	query := ""
	if st := *f.Plan.at(i); st != nil {
		query = st.typed()
	}
	ask(strings.ToLower(strings.TrimSuffix(planWords[i], ":")), query, func(text string) error {
		var st *Stamp
		if strings.TrimSpace(text) != "" {
			s, err := parseDate(text, today())
			if err != nil {
				return err
			}
			st = &s
		}
		changeFront(cursor, func(f *Front) { *f.Plan.at(i) = st })
		return nil
	})
}

// insertStamp asks for a date and puts it at the cursor as a timestamp,
// today's for none.
func insertStamp(cursor int, active bool) {
	ask("timestamp", "", func(text string) error {
		st := Stamp{Day: today()}
		if strings.TrimSpace(text) != "" {
			s, err := parseDate(text, today())
			if err != nil {
				return err
			}
			st = s
		}
		st.Active = active
		edit(api.Edit{Start: cursor, End: cursor, Text: st.String()})
		return nil
	})
}

// sparse closes every fold and opens those that hide a line holding text,
// as fude's zM and zv do, so the outline shows those lines and the
// headings over them. Case does not matter.
func sparse(text string) error {
	n, _ := sdk.Len()
	all, _ := sdk.Text(0, n)
	needle := strings.ToLower(strings.TrimSpace(text))
	keys := "zM"
	count, first := 0, 0
	for i, line := range strings.Split(all, "\n") {
		if needle != "" && strings.Contains(strings.ToLower(line), needle) {
			if count == 0 {
				first = i
			}
			count++
			keys += ":" + strconv.Itoa(i+1) + "<CR>zv"
		}
	}
	if count == 0 {
		return errNotFound("line that holds " + text)
	}
	keys += ":" + strconv.Itoa(first+1) + "<CR>"
	return sdk.Do(api.Result{Keys: keys, Message: strconv.Itoa(count) + " lines hold " + text})
}

// here returns the absolute path of the buffer the call is about, or ""
// for none. fude may give the path relative to its working folder.
func here() string {
	p := sdk.Current().Path
	if p == "" || isAbs(p) {
		return p
	}
	return path.Join(sdk.About("cwd"), p)
}

// isAbs reports whether p needs no folder before it. That is a path
// absolute on Unix or Windows, such as C:\notes, or one starting with ~.
func isAbs(p string) bool {
	return path.IsAbs(p) || strings.HasPrefix(p, "~") || strings.HasPrefix(p, `\\`) || len(p) > 2 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}
