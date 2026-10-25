package main

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Target is a heading refile may move a subtree under: its file, line,
// level and the titles of the headings it is under and its own, or a file
// itself, of level 0.
type Target struct {
	Path        string
	Line, Level int
	Outline     []string
}

// String is t as refile lists it: its file and its outline.
func (t Target) String() string {
	return strings.Join(append([]string{path.Base(t.Path)}, t.Outline...), " › ")
}

// targets returns the file at p, whose text is text, and its headings, as
// refile may move a subtree under them.
func targets(p, text string) []Target {
	t, d := keywordsIn(text, todo, done)
	keys := slices.Concat(t, d)
	out := []Target{{Path: p}}
	var outline []string
	for i, line := range strings.Split(text, "\n") {
		h, ok := parseHead(line, keys)
		if !ok {
			continue
		}
		outline = append(outline[:min(len(outline), h.Level-1)], h.Title)
		out = append(out, Target{p, i, h.Level, slices.Clone(outline)})
	}
	return out
}

// refile asks where to move the subtree of the section that holds cursor,
// among the headings of the org files and the buffer, and moves it there.
func refile(cursor int) {
	h, ok := owner(cursor)
	if !ok {
		sdk.Do(api.Result{Error: "no heading here"})
		return
	}
	gather(func() {
		n, _ := sdk.Len()
		text, _ := sdk.Text(0, n)
		// The buffer's own headings come first, as it has them now.
		file := here()
		all := targets(file, text)
		for _, p := range slices.Sorted(maps.Keys(read)) {
			if p != file {
				all = append(all, read[p].heads...)
			}
		}
		items := make([]string, len(all))
		for i, t := range all {
			items[i] = t.String()
		}
		report(sdk.Pick("refile to", items, func(i int, ok bool) {
			if ok {
				report(moveTo(h, all[i]))
			}
		}))
	})
}

// moveTo moves h's subtree under t, at the end of t's subtree, its levels
// one below t's.
func moveTo(h Heading, t Target) error {
	end := subtree(h)
	sub, _ := sdk.Text(h.Start, end)
	if !strings.HasSuffix(sub, "\n") {
		sub += "\n"
	}
	if now, ok := headingAt(h.Line); !ok || now.Text != h.Text {
		return errors.New("the heading moved while refile asked where")
	}
	sub = relevel(sub, h.Level, t.Level+1)
	if t.Path != here() {
		text, err := sdk.ReadFile(t.Path)
		if err != nil {
			return fmt.Errorf("read %s: %w", t.Path, err)
		}
		if err := sdk.WriteFile(t.Path, under(text, t, sub)); err != nil {
			return err
		}
		return sdk.Do(api.Result{Edits: []api.Edit{{Start: h.Start, End: end}}, Message: "refiled to " + t.String()})
	}
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	at := n
	if t.Level > 0 {
		if start := lineStart(text, t.Line); start >= h.Start && start < end {
			return errors.New("a heading cannot go under itself")
		}
		at = subtreeEnd(text, t)
	}
	if at == n && !strings.HasSuffix(text, "\n") {
		sub = "\n" + strings.TrimSuffix(sub, "\n")
	}
	edits := []api.Edit{{Start: h.Start, End: end}, {Start: at, End: at, Text: sub}}
	// A place that touches the subtree's own takes it where it is.
	if at >= h.Start && at <= end {
		edits = []api.Edit{{Start: h.Start, End: end, Text: sub}}
	}
	return sdk.Do(api.Result{Edits: edits, Message: "refiled to " + t.String()})
}

// under returns text with sub at the end of t's subtree, or of the text
// for a file.
func under(text string, t Target, sub string) string {
	at := len(text)
	if t.Level > 0 {
		at = subtreeEnd(text, t)
	}
	if at == len(text) && text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
		at++
	}
	return text[:at] + sub + text[at:]
}

// subtreeEnd returns where the subtree of the heading t names ends in text.
func subtreeEnd(text string, t Target) int {
	at := lineStart(text, t.Line+1)
	for at < len(text) {
		line, _, _ := strings.Cut(text[at:], "\n")
		if l := level(line); l > 0 && l <= t.Level {
			return at
		}
		at += len(line) + 1
	}
	return len(text)
}

// lineStart returns where line n of text starts, or its end.
func lineStart(text string, n int) int {
	at := 0
	for range n {
		i := strings.IndexByte(text[at:], '\n')
		if i < 0 {
			return len(text)
		}
		at += i + 1
	}
	return at
}

// relevel moves each heading of sub, a subtree whose top is of level from,
// so its top is of level to.
func relevel(sub string, from, to int) string {
	var b strings.Builder
	for line := range strings.Lines(sub) {
		if l := level(line); l > 0 {
			line = strings.Repeat("*", max(1, l-from+to)) + line[l:]
		}
		b.WriteString(line)
	}
	return b.String()
}

// archive moves the subtree of the section that holds cursor to the end of
// the file's archive, <file>_archive, as a heading of one star with where
// it came from in its drawer.
func archive(cursor int) error {
	h, ok := owner(cursor)
	if !ok {
		return errors.New("no heading here")
	}
	f, frontEnd := front(h)
	var outline []string
	for p, ok := parent(h); ok; p, ok = parent(p) {
		t, d := bufferKeys()
		head, _ := parseHead(p.Text, slices.Concat(t, d))
		outline = append([]string{head.Title}, outline...)
	}
	f.setProp("ARCHIVE_TIME", Stamp{Day: today(), Time: sdk.Now().Format("15:04")}.String())
	file := here()
	f.setProp("ARCHIVE_FILE", file)
	if len(outline) > 0 {
		f.setProp("ARCHIVE_OLPATH", strings.Join(outline, "/"))
	}
	f.setProp("ARCHIVE_CATEGORY", strings.TrimSuffix(path.Base(file), path.Ext(file)))
	if f.Head.Keyword != "" {
		f.setProp("ARCHIVE_TODO", f.Head.Keyword)
	}
	end := subtree(h)
	rest, _ := sdk.Text(frontEnd, end)
	sub := relevel(f.String()+rest, h.Level, 1)
	if !strings.HasSuffix(sub, "\n") {
		sub += "\n"
	}
	to := file + "_archive"
	_, exists, err := sdk.Stat(to)
	text := ""
	switch {
	case err != nil:
		return err
	case exists:
		if text, err = sdk.ReadFile(to); err != nil {
			return fmt.Errorf("read %s: %w", to, err)
		}
	}
	if err := sdk.WriteFile(to, under(text, Target{Path: to}, sub)); err != nil {
		return err
	}
	return sdk.Do(api.Result{Edits: []api.Edit{{Start: h.Start, End: end}}, Message: "archived to " + path.Base(to)})
}
