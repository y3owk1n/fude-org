package main

import (
	"slices"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// tableAt returns the first and last lines of the table that holds line n,
// and their text.
func tableAt(n int) (int, int, []string, bool) {
	if _, text := lineAt(n); !isTableLine(text) {
		return 0, 0, nil, false
	}
	lines, _ := sdk.Lines()
	first, last := n, n
	for ; first > 0; first-- {
		if _, text := lineAt(first - 1); !isTableLine(text) {
			break
		}
	}
	for ; last+1 < lines; last++ {
		if _, text := lineAt(last + 1); !isTableLine(text) {
			break
		}
	}
	var out []string
	for l := first; l <= last; l++ {
		_, text := lineAt(l)
		out = append(out, text)
	}
	return first, last, out, true
}

// putTable asks for t, aligned, in place of lines first to last, and the
// cursor at the start of cell c of row r.
func putTable(first, last int, t Table, r, c int) {
	start, _ := lineAt(first)
	lastStart, lastText := lineAt(last)
	lines := t.Lines()
	pos := start
	for _, l := range lines[:r] {
		pos += len(l) + 1
	}
	if t.Rows[r] != nil {
		pos += cellStart(lines[r], c)
	}
	sdk.Do(api.Result{Edits: []api.Edit{{Start: start, End: lastStart + len(lastText), Text: strings.Join(lines, "\n")}}, Jump: &api.Jump{Pos: pos}})
}

// moveCell aligns the table that holds cursor and moves to its next cell,
// or the one before with back, past rules. Past the last cell it adds a
// row. It is false outside a table.
func moveCell(cursor int, back bool) bool {
	n, _ := sdk.LineOf(cursor)
	first, last, lines, ok := tableAt(n)
	if !ok {
		return false
	}
	t := parseTable(lines)
	start, _ := lineAt(n)
	r, c, cols := n-first, cellAt(lines[n-first], cursor-start), t.width()
	// The cells in reading order, past rules.
	type cell struct{ r, c int }
	var cells []cell
	for i, row := range t.Rows {
		for j := range cols {
			if row != nil {
				cells = append(cells, cell{i, j})
			}
		}
	}
	at := slices.IndexFunc(cells, func(x cell) bool { return x.r > r || x.r == r && x.c >= c })
	on := at >= 0 && cells[at] == cell{r, c}
	switch {
	case at < 0 && back:
		at = len(cells) - 1
	case at < 0:
		at = len(cells)
	case back:
		at = max(at-1, 0)
	case on:
		at++
	}
	if at >= len(cells) {
		t.Rows = append(t.Rows, make([]string, cols))
		cells = append(cells, cell{len(t.Rows) - 1, 0})
	}
	r, c = cells[at].r, cells[at].c
	putTable(first, last, t, r, c)
	return true
}

// alignTable aligns the table that holds cursor, or makes the cursor's
// line one, its cells apart by commas or tabs.
func alignTable(cursor int) {
	n, _ := sdk.LineOf(cursor)
	first, last, lines, ok := tableAt(n)
	if !ok {
		_, text := lineAt(n)
		sep := ","
		if !strings.Contains(text, ",") {
			sep = "\t"
		}
		first, last, lines = n, n, []string{"|" + strings.ReplaceAll(text, sep, "|")}
	}
	putTable(first, last, parseTable(lines), n-first, 0)
}

// addRule adds a rule under the row of the table that holds cursor.
func addRule(cursor int) {
	n, _ := sdk.LineOf(cursor)
	first, last, lines, ok := tableAt(n)
	if !ok {
		return
	}
	t := parseTable(lines)
	r := n - first
	t.Rows = append(t.Rows[:r+1], append([][]string{nil}, t.Rows[r+1:]...)...)
	putTable(first, last, t, r, 0)
}

// recalc works out the formulas of the #+TBLFM: line under the table that
// holds cursor, and aligns it.
func recalc(cursor int) error {
	n, _ := sdk.LineOf(cursor)
	first, last, lines, ok := tableAt(n)
	if !ok {
		return errNotFound("table")
	}
	_, below := lineAt(last + 1)
	fs, err := parseFormulas(below)
	if err != nil {
		return err
	}
	if fs == nil {
		return errNotFound("#+TBLFM: line under the table")
	}
	t := parseTable(lines)
	if err := t.Calc(fs); err != nil {
		return err
	}
	start, _ := lineAt(n)
	putTable(first, last, t, n-first, max(cellAt(lines[n-first], cursor-start), 0))
	return nil
}
