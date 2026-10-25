package main

import (
	"strconv"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// footnoteAt returns the label of the footnote [fn:label] in line that
// holds col, and whether the line defines it, starting with it.
func footnoteAt(line string, col int) (string, bool, bool) {
	for at := 0; ; {
		i := strings.Index(line[at:], "[fn:")
		if i < 0 {
			return "", false, false
		}
		start := at + i
		end := strings.IndexByte(line[start:], ']')
		if end < 0 {
			return "", false, false
		}
		if col >= start && col <= start+end {
			label, _, _ := strings.Cut(line[start+4:start+end], ":")
			return label, start == 0, label != ""
		}
		at = start + end + 1
	}
}

// footnote goes from the footnote reference under the cursor to its
// definition, or from a definition to its first reference. Elsewhere it
// adds a footnote: a reference at the cursor and its definition under
// the Footnotes heading at the end, made when there is none, and starts
// insert mode there. It is false off a footnote when only, as <CR> asks.
func footnote(cursor int, only bool) bool {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	lines := strings.Split(text, "\n")
	l, _ := sdk.LineOf(cursor)
	start, line := lineAt(l)
	label, def, ok := footnoteAt(line, cursor-start)
	switch {
	case ok && def:
		at := 0
		for i, other := range lines {
			if j := strings.Index(other, "[fn:"+label+"]"); j >= 0 && i != l {
				report(sdk.Do(api.Result{Jump: &api.Jump{Pos: at + j}}))
				return true
			}
			at += len(other) + 1
		}
		report(errNotFound("reference to fn:" + label))
		return true
	case ok:
		at := 0
		for _, other := range lines {
			if strings.HasPrefix(other, "[fn:"+label+"]") {
				report(sdk.Do(api.Result{Jump: &api.Jump{Pos: at}}))
				return true
			}
			at += len(other) + 1
		}
		report(errNotFound("definition of fn:" + label))
		return true
	case only:
		return false
	}
	// A new footnote is numbered one past the highest.
	next := 1
	for _, other := range lines {
		for at := 0; ; {
			i := strings.Index(other[at:], "[fn:")
			if i < 0 {
				break
			}
			at += i + 4
			end := strings.IndexAny(other[at:], ":]")
			if num, err := strconv.Atoi(other[at : at+max(end, 0)]); end >= 0 && err == nil {
				next = max(next, num+1)
			}
		}
	}
	ref := "[fn:" + strconv.Itoa(next) + "]"
	// The definition goes at the end of the Footnotes section, or of the
	// text under a new one.
	at, add := n, "\n* Footnotes\n"+ref+" "
	for i, other := range lines {
		if h, ok := parseHead(other, nil); ok && h.Level == 1 && h.Title == "Footnotes" {
			add = "\n" + ref + " "
			if end := subtree(Heading{i, lineStart(text, i), 1, other}); end < n {
				at = end - 1
			}
			break
		}
	}
	if at == n && strings.HasSuffix(text, "\n") {
		add = strings.TrimPrefix(add, "\n")
	}
	pos := at + len(add)
	if cursor <= at {
		pos += len(ref)
	}
	edits := []api.Edit{{Start: cursor, End: cursor, Text: ref}, {Start: at, End: at, Text: add}}
	if cursor == at {
		edits = []api.Edit{{Start: at, End: at, Text: ref + add}}
	}
	sdk.Do(api.Result{Edits: edits, Jump: &api.Jump{Pos: pos}, Keys: "a"})
	return true
}
