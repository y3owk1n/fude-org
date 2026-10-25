package main

import (
	"errors"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// Block is a source block: its language, its header arguments by name,
// such as :tangle, its text, the lines of its #+BEGIN_SRC and #+END_SRC,
// and its #+BEGIN_SRC line, which finds it again.
type Block struct {
	Lang       string
	Args       map[string]string
	Text       string
	Begin, End int
	BeginLine  string
}

// blockAt returns the source block that holds line n of lines.
func blockAt(lines []string, n int) (Block, bool) {
	for b := n; b >= 0; b-- {
		upper := strings.ToUpper(strings.TrimSpace(lines[b]))
		if b < n && strings.HasPrefix(upper, "#+END_SRC") {
			return Block{}, false
		}
		if !strings.HasPrefix(upper, "#+BEGIN_SRC") {
			continue
		}
		for e := b + 1; e < len(lines); e++ {
			if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(lines[e])), "#+END_SRC") {
				if e < n {
					return Block{}, false
				}
				return parseBlock(lines, b, e), true
			}
		}
		return Block{}, false
	}
	return Block{}, false
}

// parseBlock reads the block from line b to line e of lines.
func parseBlock(lines []string, b, e int) Block {
	words := strings.Fields(strings.TrimSpace(lines[b])[len("#+BEGIN_SRC"):])
	blk := Block{Args: map[string]string{}, Text: strings.Join(lines[b+1:e], "\n"), Begin: b, End: e, BeginLine: lines[b]}
	if len(words) > 0 {
		blk.Lang = words[0]
	}
	for i := 1; i < len(words); i++ {
		if name := words[i]; strings.HasPrefix(name, ":") {
			value := ""
			if i+1 < len(words) && !strings.HasPrefix(words[i+1], ":") {
				value, i = words[i+1], i+1
			}
			blk.Args[name] = value
		}
	}
	return blk
}

// program returns the program the languages option runs lang with.
func program(lang string) (string, bool) {
	v, _ := sdk.Option("fude-org.languages")
	for _, pair := range strings.Fields(v) {
		if name, prog, ok := strings.Cut(pair, "="); ok && name == lang {
			return prog, true
		}
	}
	return "", false
}

// runBlock runs the source block under the cursor with its language's
// program, with the block's text as input. What the program prints
// replaces the results under the block.
func runBlock(cursor int) error {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	l, _ := sdk.LineOf(cursor)
	blk, ok := blockAt(strings.Split(text, "\n"), l)
	if !ok {
		return errors.New("no source block here")
	}
	prog, ok := program(blk.Lang)
	if !ok {
		return errors.New("the languages option runs no " + blk.Lang)
	}
	return sdk.Pipe(blk.Text+"\n", func(out string, err error) {
		if err != nil {
			out = err.Error()
		}
		report(putResults(blk, out))
	}, prog)
}

// putResults puts out under #+RESULTS: after blk, which it finds again by
// its lines, in place of the results it had.
func putResults(blk Block, out string) error {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	lines := strings.Split(text, "\n")
	at := strings.Index(text, blk.BeginLine+"\n"+blk.Text+"\n")
	if at < 0 {
		return errors.New("the block changed while it ran")
	}
	end := strings.Count(text[:at], "\n") + strings.Count(blk.Text, "\n") + 2 // the line of #+END_SRC
	from, to := lineStart(text, end+1), lineStart(text, end+1)
	if next := end + 1; next < len(lines) && strings.EqualFold(strings.TrimSpace(lines[next]), "#+RESULTS:") {
		k := next + 1
		for k < len(lines) && strings.HasPrefix(lines[k], ":") {
			k++
		}
		to = lineStart(text, k)
	}
	block := "#+RESULTS:"
	for line := range strings.Lines(strings.TrimRight(out, "\n")) {
		block += "\n: " + strings.TrimRight(line, "\n")
	}
	switch {
	case from == len(text) && !strings.HasSuffix(text, "\n"):
		block = "\n" + block
	case to < len(text) || strings.HasSuffix(text, "\n"):
		block += "\n"
	}
	edit(api.Edit{Start: from, End: to, Text: block})
	return nil
}

// tangle writes the text of each source block with :tangle path to that
// file, beside the buffer's file, the blocks of one file in their order.
func tangle() error {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	lines := strings.Split(text, "\n")
	files := map[string][]string{}
	blocks := 0
	for i := 0; i < len(lines); i++ {
		blk, ok := blockAt(lines, i)
		if !ok || blk.Begin != i {
			continue
		}
		if to := blk.Args[":tangle"]; to != "" && to != "no" {
			if !path.IsAbs(to) && !strings.HasPrefix(to, "~") {
				to = path.Join(path.Dir(sdk.Current().Path), to)
			}
			files[to] = append(files[to], blk.Text)
			blocks++
		}
		i = blk.End
	}
	if len(files) == 0 {
		return errors.New("no source block has :tangle")
	}
	for _, to := range slices.Sorted(maps.Keys(files)) {
		if err := sdk.WriteFile(to, strings.Join(files[to], "\n")+"\n"); err != nil {
			return err
		}
	}
	return sdk.Message("tangled " + strconv.Itoa(blocks) + " blocks into " + strconv.Itoa(len(files)) + " files")
}

// filetypes are the filetypes of the languages a block may name that differ
// from their names.
var filetypes = map[string]string{"sh": "bash", "shell": "bash", "js": "javascript", "py": "python", "elisp": "lisp", "emacs-lisp": "lisp"}

// editBlock opens the text of the source block under the cursor in a
// buffer of its language, which :w writes back.
func editBlock(cursor int) error {
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	l, _ := sdk.LineOf(cursor)
	blk, ok := blockAt(strings.Split(text, "\n"), l)
	if !ok {
		return errors.New("no source block here")
	}
	ft := blk.Lang
	if f, ok := filetypes[ft]; ok {
		ft = f
	}
	narrowed.from, narrowed.before, narrowed.text = sdk.Current().ID, blk.BeginLine+"\n", blk.Text+"\n"
	return sdk.Do(api.Result{Scratch: &api.Scratch{Text: blk.Text, Filetype: ft, Where: "here", Kind: "org-src"}})
}
