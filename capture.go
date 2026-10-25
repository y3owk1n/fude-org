package main

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/y3owk1n/fude/sdk"
)

// Template is a capture template: the key that picks it, the file it adds
// to, the heading it goes under there or none, and its text.
type Template struct {
	Key, File, Heading, Text string
}

// templates reads the templates option, a template a line: its key, its
// file with ::*heading after it or not, and its text, separated by spaces.
func templates() ([]Template, error) {
	v, _ := sdk.Option("fude-org.templates")
	var out []Template
	for line := range strings.Lines(v) {
		key, rest, _ := strings.Cut(strings.TrimSpace(line), " ")
		target, text, _ := strings.Cut(strings.TrimSpace(rest), " ")
		if key == "" {
			continue
		}
		if target == "" || text == "" {
			return nil, fmt.Errorf("template %q has no file or no text", key)
		}
		file, heading, _ := strings.Cut(target, "::")
		out = append(out, Template{key, file, strings.TrimPrefix(heading, "*"), strings.TrimSpace(text)})
	}
	if len(out) == 0 {
		return nil, errors.New("no capture templates")
	}
	return out, nil
}

// askCapture asks for a template, when there is more than one, and for the
// text, and captures it with a link to cursor.
func askCapture(cursor int) {
	ts, err := templates()
	if err != nil {
		report(err)
		return
	}
	if len(ts) == 1 {
		askText(ts[0], cursor)
		return
	}
	items := make([]string, len(ts))
	for i, t := range ts {
		items[i] = t.Key + "  " + t.Text + "  → " + t.File
	}
	report(sdk.Pick("capture with", items, func(i int, ok bool) {
		if ok {
			askText(ts[i], cursor)
		}
	}))
}

// askText asks for the text t's %? stands for, and captures it with a
// link to cursor.
func askText(t Template, cursor int) {
	link := linkHere(cursor)
	ask("capture", "", func(text string) error {
		if text == "" {
			return nil
		}
		return capture(t, text, link)
	})
}

// linkHere returns a link to where the capture began: the heading of the
// section that holds cursor, or the file.
func linkHere(cursor int) string {
	here := sdk.Current().Path
	if here == "" {
		return ""
	}
	if h, ok := owner(cursor); ok && sdk.Current().Filetype == "org" {
		t, d := bufferKeys()
		head, _ := parseHead(h.Text, append(t, d...))
		return "[[file:" + here + "::*" + head.Title + "][" + head.Title + "]]"
	}
	return "[[file:" + here + "]]"
}

// capture adds t's text, with text for %? and link for %a, to the end of
// t's file, or of its heading's subtree there, at the level below it.
func capture(t Template, text, link string) error {
	dir, _ := sdk.Option("fude-org.directory")
	file := t.File
	if !path.IsAbs(file) && !strings.HasPrefix(file, "~") {
		file = path.Join(dir, file)
	}
	now := sdk.Now()
	day := Stamp{Active: true, Day: today()}
	at := day
	at.Time = now.Format("15:04")
	entry := strings.NewReplacer(
		"%?", text, "%a", link, "\\n", "\n",
		"%T", at.String(), "%t", day.String(),
		"%U", inactive(at).String(), "%u", inactive(day).String(),
	).Replace(t.Text) + "\n"
	was := ""
	if _, ok, err := sdk.Stat(file); err != nil {
		return err
	} else if ok {
		if was, err = sdk.ReadFile(file); err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}
	} else if err := sdk.MakeDir(path.Dir(file)); err != nil {
		return err
	}
	target := Target{Path: file}
	if t.Heading != "" {
		line, ok := find(was, "*"+t.Heading)
		if !ok {
			return errNotFound(t.Heading + " in " + path.Base(file))
		}
		target.Line, target.Level = line, level(strings.Split(was, "\n")[line])
	}
	if l := level(entry); l > 0 {
		entry = relevel(entry, l, max(target.Level+1, l))
	}
	if err := sdk.WriteFile(file, under(was, target, entry)); err != nil {
		return err
	}
	return sdk.Message("captured into " + path.Base(file))
}

// inactive returns st as an inactive timestamp.
func inactive(st Stamp) Stamp {
	st.Active = false
	return st
}
