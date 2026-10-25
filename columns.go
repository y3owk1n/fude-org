package main

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/y3owk1n/fude/sdk"
)

// column is a column of column view: the property it shows, its title
// and how a heading sums its children's values, + for numbers, : for
// times such as 1:30, or none.
type column struct {
	name, title, summary string
}

// columnsOf reads the #+COLUMNS: line of text, or org's default columns.
func columnsOf(text string) []column {
	spec := "%25ITEM %TODO %3PRIORITY %TAGS"
	for line := range strings.Lines(text) {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "#+COLUMNS:"); ok {
			spec = rest
			break
		}
	}
	var out []column
	for _, word := range strings.Fields(spec) {
		word = strings.TrimLeft(strings.TrimPrefix(word, "%"), "0123456789")
		c := column{}
		if i := strings.IndexByte(word, '{'); i >= 0 {
			c.summary, word = strings.TrimSuffix(word[i+1:], "}"), word[:i]
		}
		c.name, c.title = word, word
		if i := strings.IndexByte(word, '('); i >= 0 {
			c.name, c.title = word[:i], strings.TrimSuffix(word[i+1:], ")")
		}
		out = append(out, c)
	}
	return out
}

// columnView shows the headings of the section that holds cursor, or of
// the file before the first heading, with the columns of its #+COLUMNS:
// line, a heading's summed column summing its children's values.
func columnView(cursor int) {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	cols := columnsOf(text)
	first, last := 0, n
	if h, ok := owner(cursor); ok {
		first, last = h.Start, subtree(h)
	}
	t, d := bufferKeys()
	keys := slices.Concat(t, d)
	type row struct {
		level  int
		values []string
		entry  *Entry
	}
	var rows []row
	at := first
	for line := range strings.Lines(text[first:last]) {
		if level(line) > 0 {
			f, _ := parseFront(text[at:last], keys)
			values := make([]string, len(cols))
			for i, c := range cols {
				switch c.name {
				case "ITEM":
					values[i] = strings.Repeat("*", f.Head.Level) + " " + f.Head.Title
				case "TODO":
					values[i] = f.Head.Keyword
				case "PRIORITY":
					values[i] = f.Head.Priority
				case "TAGS":
					if len(f.Head.Tags) > 0 {
						values[i] = ":" + strings.Join(f.Head.Tags, ":") + ":"
					}
				default:
					values[i] = f.prop(c.name)
				}
			}
			line, _ := sdk.LineOf(at)
			rows = append(rows, row{f.Head.Level, values, &Entry{Path: sdk.Current().Path, Line: line, Title: f.Head.Title}})
		}
		at += len(line)
	}
	// A heading sums the values of its children, from the deepest up.
	for i, c := range cols {
		if c.summary != "+" && c.summary != ":" {
			continue
		}
		for r := len(rows) - 1; r >= 0; r-- {
			var sum float64
			some := false
			for k := r + 1; k < len(rows) && rows[k].level > rows[r].level; k++ {
				if rows[k].level == rows[r].level+1 && rows[k].values[i] != "" {
					sum += amount(rows[k].values[i], c.summary)
					some = true
				}
			}
			if some {
				rows[r].values[i] = sumText(sum, c.summary)
			}
		}
	}
	titles := make([]string, len(cols))
	for i, c := range cols {
		titles[i] = c.title
	}
	table := Table{Rows: [][]string{titles, nil}}
	shownRows := []*Entry{nil, nil}
	for _, r := range rows {
		table.Rows = append(table.Rows, r.values)
		shownRows = append(shownRows, r.entry)
	}
	panel("[columns]", table.Lines(), shownRows)
}

// amount reads a value of a column that sums, a number for + and a time
// such as 1:30 for :, as minutes.
func amount(v, summary string) float64 {
	if summary == "+" {
		return number(v)
	}
	h, m, _ := strings.Cut(v, ":")
	hh, _ := strconv.Atoi(h)
	mm, _ := strconv.Atoi(m)
	return float64(hh*60 + mm)
}

// sumText writes a sum of a column as it reads.
func sumText(sum float64, summary string) string {
	if summary == "+" {
		return format(sum)
	}
	return hours(time.Duration(sum) * time.Minute)
}
