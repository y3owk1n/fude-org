package main

import (
	"errors"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// narrowed is the subtree the narrow buffer holds: the buffer it came from
// and its text there, as it was last written.
var narrowed struct {
	from int
	text string
}

// narrow opens the subtree of the section that holds cursor in a buffer of
// its own, in the window, which :w writes back in its place.
func narrow(cursor int) error {
	h, ok := owner(cursor)
	if !ok {
		return errors.New("no heading here")
	}
	narrowed.from = sdk.Current().ID
	narrowed.text, _ = sdk.Text(h.Start, subtree(h))
	return sdk.Do(api.Result{Scratch: &api.Scratch{Text: strings.TrimSuffix(narrowed.text, "\n"), Filetype: "org", Where: "here", Kind: "org-narrow"}})
}

// writeBack writes the narrow buffer's text in place of the subtree it
// came from, which must be as it was last written.
func writeBack() error {
	total, err := sdk.LenOf(narrowed.from)
	if err != nil {
		return errors.New("the file the subtree came from is closed")
	}
	src, _ := sdk.TextOf(narrowed.from, 0, total)
	at := strings.Index(src, narrowed.text)
	if at < 0 {
		return errors.New("the subtree changed in its file since it was narrowed")
	}
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	if strings.HasSuffix(narrowed.text, "\n") && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	edit := api.Edit{Start: at, End: at + len(narrowed.text), Text: text, Buffer: narrowed.from}
	narrowed.text = text
	// A writer's message says its write failed, so it shows none.
	return sdk.Do(api.Result{Edits: []api.Edit{edit}})
}
