package main

import (
	"path"
	"strconv"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Link is an org link, [[target][description]] or [[target]], and where it
// starts and ends in its line.
type Link struct {
	Target, Desc string
	Start, End   int
}

// links returns the links in line.
func links(line string) []Link {
	var out []Link
	for at := 0; ; {
		i := strings.Index(line[at:], "[[")
		if i < 0 {
			return out
		}
		start := at + i
		j := strings.Index(line[start:], "]]")
		if j < 0 {
			return out
		}
		inside := line[start+2 : start+j]
		target, desc, _ := strings.Cut(inside, "][")
		if target != "" && !strings.Contains(target, "[") {
			out = append(out, Link{target, desc, start, start + j + 2})
		}
		at = start + j + 2
	}
}

// stored is the link <Leader>ols stored last, as file:path::*title.
var stored string

// store keeps a link to the heading of the section that holds pos, for
// insertLink.
func store(pos int) error {
	h, ok := owner(pos)
	if !ok {
		return errNotFound("heading")
	}
	t, d := bufferKeys()
	head, _ := parseHead(h.Text, append(t, d...))
	stored = "file:" + sdk.Current().Path + "::*" + head.Title
	return sdk.Message("stored a link to " + head.Title)
}

// follow follows the link on the line that holds cursor and under it. It
// is false off a link.
func follow(cursor int) bool {
	n, _ := sdk.LineOf(cursor)
	start, text := lineAt(n)
	for _, l := range links(text) {
		if cursor-start >= l.Start && cursor-start < l.End {
			report(open(l.Target))
			return true
		}
	}
	return false
}

// open goes to target. *title is a heading of the buffer, #id the heading
// whose CUSTOM_ID is id, and a URL goes to the opener. Anything else is a
// file, with file: before it or not, and what follows :: is where in it.
func open(target string) error {
	switch {
	case strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:"):
		opener, _ := sdk.Option("fude-org.opener")
		return sdk.Start(func(_ string, err error) { report(err) }, opener, target)
	case strings.HasPrefix(target, "*") || strings.HasPrefix(target, "#"):
		n, _ := sdk.Len()
		text, _ := sdk.Text(0, n)
		if line, ok := find(text, target); ok {
			start, _ := sdk.LineStart(line)
			return sdk.Do(api.Result{Jump: &api.Jump{Pos: start}})
		}
		return errNotFound(target)
	}
	file, search, _ := strings.Cut(strings.TrimPrefix(target, "file:"), "::")
	if !path.IsAbs(file) && !strings.HasPrefix(file, "~") {
		file = path.Join(path.Dir(sdk.Current().Path), file)
	}
	j := &api.Jump{Path: file}
	if search != "" {
		text, err := sdk.ReadFile(file)
		if err != nil {
			return err
		}
		line, ok := find(text, search)
		if !ok {
			return errNotFound(search)
		}
		j.Line = line + 1
	}
	return sdk.Do(api.Result{Jump: j})
}

// find returns the line of text a link's search names. *title is a
// heading, #id the heading whose CUSTOM_ID is id, and a number a line. Any
// other text is the first line that holds it.
func find(text, search string) (int, bool) {
	if n, err := strconv.Atoi(search); err == nil {
		return n - 1, n > 0
	}
	t, d := keywordsIn(text, todo, done)
	keys := append(t, d...)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(search, "*"):
			if h, ok := parseHead(line, keys); ok && strings.EqualFold(h.Title, strings.TrimSpace(search[1:])) {
				return i, true
			}
		case strings.HasPrefix(search, "#"):
			name, value, _ := strings.Cut(strings.TrimPrefix(strings.TrimSpace(line), ":"), ":")
			if strings.EqualFold(name, "CUSTOM_ID") && strings.TrimSpace(value) == search[1:] {
				for h := i; h >= 0; h-- {
					if level(lines[h]) > 0 {
						return h, true
					}
				}
			}
		case strings.Contains(line, search):
			return i, true
		}
	}
	return 0, false
}

// errNotFound is the error of a link that leads nowhere.
type errNotFound string

func (e errNotFound) Error() string { return "no " + string(e) + " to go to" }

// insertLink asks for a link, from the one stored, and a description, and
// puts the link at the cursor.
func insertLink(cursor int) {
	ask("link", stored, func(target string) error {
		target = strings.TrimSpace(target)
		if target == "" {
			return nil
		}
		// A link to a heading of this file needs no file.
		if title, ok := strings.CutPrefix(target, "file:"+sdk.Current().Path+"::"); ok {
			target = title
		}
		ask("description", "", func(desc string) error {
			link := "[[" + target + "]]"
			if desc = strings.TrimSpace(desc); desc != "" {
				link = "[[" + target + "][" + desc + "]]"
			}
			edit(api.Edit{Start: cursor, End: cursor, Text: link})
			return nil
		})
		return nil
	})
}

// shown are the decorations that show the links of each buffer by their
// description, by buffer id.
var shown = map[string][]int{}

// showLinks shows each link of an org buffer by its description, or its
// target when it has none, and hides the rest of it. The cursor's line
// shows its links as they are.
func showLinks() {
	buf := sdk.Current()
	id := strconv.Itoa(buf.ID)
	res := api.Result{Undecorate: shown[id]}
	shown[id] = nil
	if buf.Filetype == "org" {
		n, _ := sdk.Len()
		text, _ := sdk.Text(0, n)
		at := 0
		for line := range strings.Lines(text) {
			for _, l := range links(line) {
				shownStart, shownEnd := at+l.Start+2, at+l.Start+2+len(l.Target)
				if l.Desc != "" {
					shownStart, shownEnd = shownEnd+2, at+l.End-2
				}
				for _, d := range []api.Decoration{
					{Start: at + l.Start, End: shownStart, Replace: true},
					{Start: shownStart, End: shownEnd, Style: "markup.link"},
					{Start: shownEnd, End: at + l.End, Replace: true},
				} {
					d.ID = sdk.ID()
					res.Decorate = append(res.Decorate, d)
					shown[id] = append(shown[id], d.ID)
				}
			}
			at += len(line)
		}
	}
	report(sdk.Do(res))
}
