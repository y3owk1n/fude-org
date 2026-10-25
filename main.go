// Package main is an org mode plugin for fude. It has fude build the org
// grammar, edits headings, TODOs and checkboxes, and shows an agenda of the
// files under its directory, with a capture into its inbox.
package main

import (
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// grammar is the org grammar fude builds, at the commit the plugin pins.
var grammar = api.GrammarSource{
	Repo:   "https://github.com/nvim-orgmode/tree-sitter-org",
	Commit: "219c0b27fdb2c0aeb43841f23f03d6f54657f288",
	Dir:    ".",
}

// highlights colours org in the names fude's themes have. The TODO and
// done keywords come from the todo option.
const highlights = `
(headline (stars) @punctuation)
(headline (item) @markup.heading)
(tag_list (tag) @tag)
[(plan) (property_drawer) (drawer) (comment)] @comment
(timestamp) @string
[(bullet) (checkbox)] @punctuation
(block) @markup.raw
(directive) @property
`

// todo and done are the TODO keywords the todo option lists, as init
// reads it.
var todo, done []string

func init() {
	sdk.Define("todo", "TODO | DONE", "the TODO keywords, those after | done, as #+TODO lists them")
	todo, done = keywords()
	q := highlights + "(item . (expr) @keyword (#any-of? @keyword " + quoted(todo) + "))\n"
	q += "(item . (expr) @comment (#any-of? @comment " + quoted(done) + "))\n"
	grammar.Queries = map[string]string{"highlights": q, "folds": "(section) @fold"}
	sdk.Register(api.Registration{Kind: api.Language, Name: "org", Help: "the org grammar", Source: &grammar}, func(_, _, _ int) int { return 0 })

	key("cycle", "<Tab>", "on a heading, opens or closes its section as <Plug>(fold.cycle) does", func(c sdk.Call) {
		if _, ok := heading(c.Cursor); ok {
			sdk.Do(api.Result{Keys: "<Plug>(fold.cycle)"})
		}
	})
	key("cycle_all", "<S-Tab>", "shows the outline, then every heading, then everything, as <Plug>(fold.cycle_all) does", func(sdk.Call) {
		sdk.Do(api.Result{Keys: "<Plug>(fold.cycle_all)"})
	})
	key("todo_next", "cit", "puts the next TODO keyword on the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			t, d := bufferKeys()
			edit(cycleTodo(h, t, d, false))
		}
	})
	key("todo_prev", "ciT", "puts the TODO keyword before on the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			t, d := bufferKeys()
			edit(cycleTodo(h, t, d, true))
		}
	})
	key("checkbox", "<C-Space>", "ticks the checkbox of the list item, or clears it", func(c sdk.Call) {
		n, _ := sdk.LineOf(c.Cursor)
		if e, ok := checkbox(lineAt(n)); ok {
			edit(e)
		}
	})
	key("promote", "<<", "takes a star off the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			shifted(h.Start, h.Start+len(h.Text), true)
		}
	})
	key("demote", ">>", "adds a star to the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			shifted(h.Start, h.Start+len(h.Text), false)
		}
	})
	key("promote_subtree", "<s", "takes a star off the heading and each heading under it", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			shifted(h.Start, subtree(h), true)
		}
	})
	key("demote_subtree", ">s", "adds a star to the heading and each heading under it", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			shifted(h.Start, subtree(h), false)
		}
	})
	key("move_up", "<Leader>oK", "moves the heading and its subtree above the heading of its level before it", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			move(h, false)
		}
	})
	key("move_down", "<Leader>oJ", "moves the heading and its subtree below the heading of its level after it", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			move(h, true)
		}
	})
	sdk.Object("ah", "a heading and its subtree", func(start, _ int) (int, int, bool) { return section(start, true) })
	sdk.Object("ih", "a heading's subtree, without the heading", func(start, _ int) (int, int, bool) { return section(start, false) })

	sdk.Define("directory", "~/org", "the folder whose org files the agenda gathers, which the plugin's paths must hold")
	sdk.Define("inbox", "~/org/inbox.org", "the file capture adds a TODO to")
	sdk.Command("OrgAgenda", "shows the TODOs and the dated headings of the org files", func(sdk.Call) { agenda() })
	sdk.Key("agenda", "<Leader>oa", "shows the agenda, as :OrgAgenda does", func(sdk.Call) { agenda() })
	sdk.Plug("agenda_open", "opens the heading the agenda shows on the cursor's line", func(c sdk.Call) {
		if e, ok := shownAt(c.Cursor); ok {
			sdk.Do(api.Result{Jump: &api.Jump{Path: e.Path, Line: e.Line + 1}})
		}
	})
	sdk.Plug("agenda_todo", "puts the next TODO keyword on the heading the agenda shows on the cursor's line", func(c sdk.Call) {
		if e, ok := shownAt(c.Cursor); ok {
			report(todoIn(e))
		}
	})
	sdk.Command("OrgCapture", "adds a TODO to the inbox file, the text typed after it or asked for", func(c sdk.Call) {
		if c.Args != "" {
			report(capture(c.Args))
			return
		}
		ask()
	})
	sdk.Key("capture", "<Leader>oc", "asks for a TODO and adds it to the inbox file", func(sdk.Call) { ask() })
}

// ask asks for a TODO and adds it to the inbox file.
func ask() {
	sdk.Ask("capture", func(text string, ok bool) {
		if ok && text != "" {
			report(capture(text))
		}
	})
}

// report shows err, if any, as the call's error.
func report(err error) {
	if err != nil {
		sdk.Do(api.Result{Error: err.Error()})
	}
}

// edit asks for e.
func edit(e api.Edit) { sdk.Do(api.Result{Edits: []api.Edit{e}}) }

// shifted asks for the stars shift adds or takes away from start to end, or
// says why it cannot.
func shifted(start, end int, up bool) {
	edits, err := shift(start, end, up)
	if err != nil {
		sdk.Do(api.Result{Error: err.Error()})
		return
	}
	sdk.Do(api.Result{Edits: edits})
}

// move swaps h's subtree with that of the heading of its level above it,
// or below it, and the cursor goes with h.
func move(h Heading, below bool) {
	s, ok := sibling(h, below)
	if !ok {
		sdk.Do(api.Result{Error: "no heading of its level to move past"})
		return
	}
	first, second := s, h
	if below {
		first, second = h, s
	}
	end := subtree(second)
	a, _ := sdk.Text(first.Start, second.Start)
	b, _ := sdk.Text(second.Start, end)
	// The last subtree of the text may end without a newline.
	if !strings.HasSuffix(b, "\n") {
		b, a = b+"\n", strings.TrimSuffix(a, "\n")
	}
	pos := first.Start
	if below {
		pos += len(b)
	}
	sdk.Do(api.Result{Edits: []api.Edit{{Start: first.Start, End: end, Text: b + a}}, Jump: &api.Jump{Pos: pos}})
}

// section returns the subtree of the heading whose section holds pos, with
// the heading's line when around. It is false outside an org file.
func section(pos int, around bool) (int, int, bool) {
	h, ok := owner(pos)
	if sdk.Current().Filetype != "org" || !ok {
		return 0, 0, false
	}
	end := subtree(h)
	if around {
		// A subtree at the end of the text takes the newline before it, so
		// deleting it leaves no blank line.
		if n, _ := sdk.Len(); end == n && h.Start > 0 {
			return h.Start - 1, end, true
		}
		return h.Start, end, true
	}
	body := min(h.Start+len(h.Text)+1, end)
	return body, end, body < end
}

// key maps keys in org files to the plug name, which runs f. When f asks
// for nothing, the keys do what they do in any other file.
func key(name, keys, help string, f func(sdk.Call)) {
	sdk.Register(api.Registration{Kind: api.Plug, Name: name, Keys: keys, Filetype: "org", Help: help}, func(cursor, count, _ int) int {
		f(sdk.Call{Cursor: cursor, Count: count})
		return 0
	})
}

// keywords returns the TODO keywords the todo option lists, as a #+TODO
// line would.
func keywords() ([]string, []string) {
	v, _ := sdk.Option("fude-org.todo")
	return keywordsIn("#+TODO: "+v, nil, nil)
}

// quoted returns words as a query's strings, each quoted.
func quoted(words []string) string {
	q := make([]string, len(words))
	for i, w := range words {
		q[i] = `"` + w + `"`
	}
	return strings.Join(q, " ")
}

func main() {}
