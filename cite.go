package main

import (
	"errors"
	"path"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Ref is an entry of a BibTeX file: its key, the line it starts on, and
// what a list of them shows of it.
type Ref struct {
	Key, Author, Year, Title, Path string
	Line                           int
}

// bibliographies returns the BibTeX files of the buffer: those its
// #+bibliography: lines name, beside its file, and the bibliography
// option's.
func bibliographies() []string {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	dir := path.Dir(here())
	var out []string
	for line := range strings.Lines(text) {
		if rest, ok := cutFold(strings.TrimSpace(line), "#+bibliography:"); ok {
			for _, p := range strings.Fields(rest) {
				if !isAbs(p) {
					p = path.Join(dir, p)
				}
				out = append(out, p)
			}
		}
	}
	v, _ := sdk.Option("fude-org.bibliography")
	return append(out, strings.Fields(v)...)
}

// cutFold is strings.CutPrefix with any case.
func cutFold(s, prefix string) (string, bool) {
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return s, false
	}
	return s[len(prefix):], true
}

// refs reads the entries of the BibTeX files.
func refs(files []string) ([]Ref, error) {
	var out []Ref
	for _, f := range files {
		text, err := sdk.ReadFile(f)
		if err != nil {
			return nil, errors.New("read " + f + ": " + err.Error())
		}
		for i, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "@") {
				if _, key, ok := strings.Cut(line, "{"); ok {
					out = append(out, Ref{Key: strings.TrimSpace(strings.TrimSuffix(key, ",")), Path: f, Line: i})
				}
				continue
			}
			name, value, ok := strings.Cut(line, "=")
			if !ok || len(out) == 0 {
				continue
			}
			value = strings.Trim(strings.TrimSpace(value), "{}\",")
			switch r := &out[len(out)-1]; strings.ToLower(strings.TrimSpace(name)) {
			case "author":
				r.Author, _, _ = strings.Cut(value, " and ")
			case "year":
				r.Year = value
			case "title":
				r.Title = value
			}
		}
	}
	return out, nil
}

// citeAt returns the key of the citation key [cite:...@key...] in line that
// holds col, and where the citation ends.
func citeAt(line string, col int) (string, int, bool) {
	start := strings.LastIndex(line[:min(col+1, len(line))], "[cite")
	if start < 0 {
		return "", 0, false
	}
	end := strings.IndexByte(line[start:], ']')
	if end < 0 || start+end < col {
		return "", 0, false
	}
	end += start
	at := strings.LastIndexByte(line[:min(col+1, end)], '@')
	if at < start {
		return "", end, true
	}
	key := line[at+1 : end]
	if i := strings.IndexAny(key, " ;]"); i >= 0 {
		key = key[:i]
	}
	return key, end, true
}

// cite asks for an entry of the bibliographies and puts a citation of it
// at the cursor, or adds it to the citation the cursor is in.
func cite(cursor int) error {
	all, err := refs(bibliographies())
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return errors.New("no #+bibliography: or bibliography option names a file with entries")
	}
	items := make([]string, len(all))
	for i, r := range all {
		items[i] = r.Key + "  " + r.Author + " (" + r.Year + ") " + r.Title
	}
	l, _ := sdk.LineOf(cursor)
	start, line := lineAt(l)
	_, end, in := citeAt(line, cursor-start)
	return sdk.Pick("cite", items, func(i int, ok bool) {
		if !ok {
			return
		}
		if in {
			edit(api.Edit{Start: start + end, End: start + end, Text: ";@" + all[i].Key})
			return
		}
		edit(api.Edit{Start: cursor, End: cursor, Text: "[cite:@" + all[i].Key + "]"})
	})
}

// followCite opens the entry of the citation key under the cursor in its
// BibTeX file. It is false off a citation key.
func followCite(cursor int) bool {
	l, _ := sdk.LineOf(cursor)
	start, line := lineAt(l)
	key, _, ok := citeAt(line, cursor-start)
	if !ok || key == "" {
		return false
	}
	all, err := refs(bibliographies())
	if err != nil {
		report(err)
		return true
	}
	for _, r := range all {
		if r.Key == key {
			report(sdk.Do(api.Result{Jump: &api.Jump{Path: r.Path, Line: r.Line + 1}}))
			return true
		}
	}
	report(errNotFound("entry " + key))
	return true
}
