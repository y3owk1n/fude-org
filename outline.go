package main

import (
	"errors"
	"slices"
	"strconv"
	"strings"

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
func subtree(h Heading) int {
	n, _ := sdk.Len()
	rest, _ := sdk.Text(h.Start, n)
	at := strings.IndexByte(rest, '\n')
	for at >= 0 {
		line, _, _ := strings.Cut(rest[at+1:], "\n")
		if l := level(line); l > 0 && l <= h.Level {
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

// cycleTodo puts the next of the keywords todo and done on the heading h,
// or the one before with back, in their order and then none.
func cycleTodo(h Heading, todo, done []string, back bool) api.Edit {
	keys := slices.Concat(todo, done)
	head, _ := parseHead(h.Text, keys)
	at := slices.Index(keys, head.Keyword)
	if at < 0 {
		at = len(keys)
	}
	step := 1
	if back {
		step = len(keys)
	}
	head.Keyword = ""
	if next := (at + step) % (len(keys) + 1); next < len(keys) {
		head.Keyword = keys[next]
	}
	return api.Edit{Start: h.Start, End: h.Start + len(h.Text), Text: head.String()}
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
