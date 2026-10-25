package main

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Table is an org table: its rows of cells, a rule being nil, and the
// indent of its lines.
type Table struct {
	Indent string
	Rows   [][]string
}

// isTableLine reports whether line is a row or a rule of a table.
func isTableLine(line string) bool {
	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "|")
}

// parseTable reads the lines of a table.
func parseTable(lines []string) Table {
	t := Table{Indent: lines[0][:len(lines[0])-len(strings.TrimLeft(lines[0], " \t"))]}
	for _, line := range lines {
		rest := strings.TrimSpace(line)
		if strings.HasPrefix(rest, "|-") {
			t.Rows = append(t.Rows, nil)
			continue
		}
		rest = strings.TrimPrefix(rest, "|")
		rest = strings.TrimSuffix(rest, "|")
		var cells []string
		for c := range strings.SplitSeq(rest, "|") {
			cells = append(cells, strings.TrimSpace(c))
		}
		t.Rows = append(t.Rows, cells)
	}
	return t
}

// width returns how many columns t has.
func (t Table) width() int {
	n := 0
	for _, r := range t.Rows {
		n = max(n, len(r))
	}
	return n
}

// Lines returns t aligned: each column as wide as its widest cell, to the
// right when more than half its cells with text are numbers, as org does,
// and else to the left.
func (t Table) Lines() []string {
	cols := t.width()
	widths := make([]int, cols)
	numeric := make([]bool, cols)
	for c := range cols {
		numbers, filled := 0, 0
		for _, r := range t.Rows {
			if c < len(r) {
				widths[c] = max(widths[c], utf8.RuneCountInString(r[c]))
				if r[c] != "" {
					filled++
					if _, err := strconv.ParseFloat(r[c], 64); err == nil {
						numbers++
					}
				}
			}
		}
		numeric[c] = 2*numbers > filled
		widths[c] = max(widths[c], 1)
	}
	out := make([]string, len(t.Rows))
	for i, r := range t.Rows {
		var b strings.Builder
		b.WriteString(t.Indent)
		if r == nil {
			for c, w := range widths {
				if c > 0 {
					b.WriteByte('+')
				} else {
					b.WriteByte('|')
				}
				b.WriteString(strings.Repeat("-", w+2))
			}
			b.WriteByte('|')
			out[i] = b.String()
			continue
		}
		for c, w := range widths {
			cell := ""
			if c < len(r) {
				cell = r[c]
			}
			pad := strings.Repeat(" ", w-utf8.RuneCountInString(cell))
			if numeric[c] {
				cell = pad + cell
			} else {
				cell += pad
			}
			b.WriteString("| " + cell + " ")
		}
		b.WriteByte('|')
		out[i] = b.String()
	}
	return out
}

// cellStart returns where cell c of line, an aligned row, starts its text.
func cellStart(line string, c int) int {
	at := strings.IndexByte(line, '|')
	for range c {
		at += strings.IndexByte(line[at+1:], '|') + 1
	}
	return at + 2
}

// cellAt returns which cell of line holds col: how many bars come before
// it, less one.
func cellAt(line string, col int) int {
	return max(strings.Count(line[:min(col, len(line))], "|")-1, 0)
}

// Formula is one of a #+TBLFM: line's formulas: the field it sets, a whole
// column when Row is 0 and the last row when it is lastRow, and its
// expression.
type Formula struct {
	Row, Col int
	Expr     string
}

// lastRow is the Row of a formula that sets the last row, @>.
const lastRow = -1

// parseFormulas reads a #+TBLFM: line.
func parseFormulas(line string) ([]Formula, error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), "#+TBLFM:")
	if !ok {
		return nil, nil
	}
	var out []Formula
	for f := range strings.SplitSeq(rest, "::") {
		target, expr, ok := strings.Cut(strings.TrimSpace(f), "=")
		if !ok {
			return nil, fmt.Errorf("%q is no formula", f)
		}
		var fm Formula
		switch r, c, field := strings.Cut(strings.TrimSpace(target), "$"); {
		case field && r == "":
			n, err := strconv.Atoi(c)
			if err != nil {
				return nil, fmt.Errorf("%q is no column", target)
			}
			fm.Col = n
		case field && strings.HasPrefix(r, "@"):
			var row int
			var err1 error
			switch r[1:] {
			case "<":
				row = 1
			case ">":
				row = lastRow
			default:
				row, err1 = strconv.Atoi(r[1:])
			}
			col, err2 := strconv.Atoi(c)
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("%q is no field", target)
			}
			fm.Row, fm.Col = row, col
		default:
			return nil, fmt.Errorf("%q is no field or column", target)
		}
		fm.Expr = strings.TrimSpace(expr)
		out = append(out, fm)
	}
	return out, nil
}

// Calc applies fs to t. A column formula sets each row under the first
// rule, or every row when there is none, and a field formula then sets its
// field. Rows count from 1 and skip rules, as org counts them.
func (t *Table) Calc(fs []Formula) error {
	var rows []int // the indexes of t.Rows that are rows, not rules
	firstData := 0
	for i, r := range t.Rows {
		if r == nil {
			if firstData == 0 && len(rows) > 0 {
				firstData = len(rows)
			}
			continue
		}
		rows = append(rows, i)
	}
	cell := func(r, c int) string {
		if r < 1 || r > len(rows) || c < 1 {
			return ""
		}
		row := t.Rows[rows[r-1]]
		if c > len(row) {
			return ""
		}
		return row[c-1]
	}
	set := func(r, c int, v float64) {
		row := t.Rows[rows[r-1]]
		for len(row) < c {
			row = append(row, "")
		}
		row[c-1] = format(v)
		t.Rows[rows[r-1]] = row
	}
	columns := slices.DeleteFunc(slices.Clone(fs), func(f Formula) bool { return f.Row != 0 })
	fields := slices.DeleteFunc(slices.Clone(fs), func(f Formula) bool { return f.Row == 0 })
	for _, f := range slices.Concat(columns, fields) {
		targets := []int{f.Row}
		switch f.Row {
		case lastRow:
			targets = []int{len(rows)}
		case 0:
			targets = nil
			for r := firstData + 1; r <= len(rows); r++ {
				targets = append(targets, r)
			}
		}
		for _, r := range targets {
			if r < 1 || r > len(rows) {
				return fmt.Errorf("the table has no row %d", r)
			}
			v, err := evaluate(f.Expr, r, f.Col, len(rows), cell)
			if err != nil {
				return fmt.Errorf("%s: %w", f.Expr, err)
			}
			set(r, f.Col, v)
		}
	}
	return nil
}

// format writes v as a table shows it: whole, or to six places at most.
func format(v float64) string {
	return strconv.FormatFloat(math.Round(v*1e6)/1e6, 'f', -1, 64)
}

// evaluate works out expr for the field at row and col of a table of rows
// rows, and cell gives the text of a field. expr has numbers, + - * / ^
// and brackets. $c is column c of row, @r is row r of col, and @r$c is a
// field. vsum, vmean, vmin, vmax and vcount take a range such as @2..@4
// or @2$1..@4$3. A row is a number, @< for the first, @> for the last, or
// @-n or @+n from row.
func evaluate(expr string, row, col, rows int, cell func(r, c int) string) (float64, error) {
	p := &calc{s: strings.ReplaceAll(expr, " ", ""), row: row, col: col, rows: rows, cell: cell}
	v, err := p.sum()
	if err == nil && p.i < len(p.s) {
		err = fmt.Errorf("%q is not understood", p.s[p.i:])
	}
	return v, err
}

// calc reads an expression, from s at i.
type calc struct {
	s              string
	i              int
	row, col, rows int
	cell           func(r, c int) string
}

func (p *calc) sum() (float64, error) {
	v, err := p.product()
	for err == nil && p.i < len(p.s) && strings.ContainsRune("+-", rune(p.s[p.i])) {
		op := p.s[p.i]
		p.i++
		var w float64
		if w, err = p.product(); op == '+' {
			v += w
		} else {
			v -= w
		}
	}
	return v, err
}

func (p *calc) product() (float64, error) {
	v, err := p.power()
	for err == nil && p.i < len(p.s) && strings.ContainsRune("*/", rune(p.s[p.i])) {
		op := p.s[p.i]
		p.i++
		var w float64
		if w, err = p.power(); op == '*' {
			v *= w
		} else {
			v /= w
		}
	}
	return v, err
}

func (p *calc) power() (float64, error) {
	v, err := p.unit()
	if err == nil && p.i < len(p.s) && p.s[p.i] == '^' {
		p.i++
		var w float64
		w, err = p.power()
		v = math.Pow(v, w)
	}
	return v, err
}

func (p *calc) unit() (float64, error) {
	switch {
	case p.i >= len(p.s):
		return 0, errors.New("it ends too soon")
	case p.s[p.i] == '-':
		p.i++
		v, err := p.unit()
		return -v, err
	case p.s[p.i] == '(':
		p.i++
		v, err := p.sum()
		if err == nil && (p.i >= len(p.s) || p.s[p.i] != ')') {
			err = errors.New("a ( has no )")
		}
		p.i++
		return v, err
	case p.s[p.i] == '$' || p.s[p.i] == '@':
		r, c, err := p.ref()
		if err != nil {
			return 0, err
		}
		return number(p.cell(r, c)), nil
	case p.s[p.i] >= 'a' && p.s[p.i] <= 'z':
		return p.call()
	}
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= '0' && p.s[p.i] <= '9' || p.s[p.i] == '.') {
		p.i++
	}
	if start == p.i {
		return 0, fmt.Errorf("%q is not understood", p.s[start:])
	}
	return strconv.ParseFloat(p.s[start:p.i], 64)
}

// call reads a function of a range, such as vsum(@2..@4).
func (p *calc) call() (float64, error) {
	start := p.i
	for p.i < len(p.s) && p.s[p.i] >= 'a' && p.s[p.i] <= 'z' {
		p.i++
	}
	name := p.s[start:p.i]
	if p.i >= len(p.s) || p.s[p.i] != '(' {
		return 0, fmt.Errorf("%s needs a range in brackets", name)
	}
	p.i++
	r1, c1, err := p.ref()
	if err != nil {
		return 0, err
	}
	if !strings.HasPrefix(p.s[p.i:], "..") {
		return 0, fmt.Errorf("%s needs a range, such as @2..@4", name)
	}
	p.i += 2
	r2, c2, err := p.ref()
	if err != nil {
		return 0, err
	}
	if p.i >= len(p.s) || p.s[p.i] != ')' {
		return 0, fmt.Errorf("%s has no )", name)
	}
	p.i++
	var vs []float64
	for r := min(r1, r2); r <= max(r1, r2); r++ {
		for c := min(c1, c2); c <= max(c1, c2); c++ {
			if text := p.cell(r, c); text != "" {
				vs = append(vs, number(text))
			}
		}
	}
	switch name {
	case "vsum":
		s := 0.0
		for _, v := range vs {
			s += v
		}
		return s, nil
	case "vmean":
		s := 0.0
		for _, v := range vs {
			s += v
		}
		return s / max(float64(len(vs)), 1), nil
	case "vmin", "vmax":
		if len(vs) == 0 {
			return 0, nil
		}
		if name == "vmin" {
			return slices.Min(vs), nil
		}
		return slices.Max(vs), nil
	case "vcount":
		return float64(len(vs)), nil
	}
	return 0, fmt.Errorf("no function %s", name)
}

// ref reads a reference, @r, $c or @r$c, and returns its row and column,
// the formula's own for a part it leaves out.
func (p *calc) ref() (int, int, error) {
	r, c := p.row, p.col
	if p.i < len(p.s) && p.s[p.i] == '@' {
		p.i++
		switch {
		case strings.HasPrefix(p.s[p.i:], "<"):
			r, p.i = 1, p.i+1
		case strings.HasPrefix(p.s[p.i:], ">"):
			r, p.i = p.rows, p.i+1
		default:
			sign := 0
			if p.i < len(p.s) && (p.s[p.i] == '-' || p.s[p.i] == '+') {
				sign = map[byte]int{'-': -1, '+': 1}[p.s[p.i]]
				p.i++
			}
			n, err := p.int()
			if err != nil {
				return 0, 0, err
			}
			r = n
			if sign != 0 {
				r = p.row + sign*n
			}
		}
	}
	if p.i < len(p.s) && p.s[p.i] == '$' {
		p.i++
		n, err := p.int()
		if err != nil {
			return 0, 0, err
		}
		c = n
	}
	return r, c, nil
}

func (p *calc) int() (int, error) {
	start := p.i
	for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	if start == p.i {
		return 0, fmt.Errorf("a reference needs a number at %q", p.s[start:])
	}
	return strconv.Atoi(p.s[start:p.i])
}

// number reads a field as a number, 0 for one that is not.
func number(text string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(text), 64)
	return v
}
