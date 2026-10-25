// Package main is an org mode plugin for fude. It has fude build the org
// grammar, edits headings, TODOs and checkboxes, and shows an agenda of the
// files under its directory, with a capture into its inbox.
package main

import (
	"errors"
	"maps"
	"slices"
	"strconv"
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
	sdk.Define("log_done", "true", "a heading that is done gets a CLOSED date")
	todo, done = keywords()
	q := highlights + "(item . (expr) @keyword (#any-of? @keyword " + quoted(todo) + "))\n"
	q += "(item . (expr) @comment (#any-of? @comment " + quoted(done) + "))\n"
	grammar.Queries = map[string]string{"highlights": q, "folds": "(section) @fold"}
	sdk.Register(api.Registration{Kind: api.Language, Name: "org", Help: "the org grammar", Source: &grammar}, func(_, _, _ int) int { return 0 })

	key("cycle", "<Tab>", "on a heading, opens or closes its section as <Plug>(fold.cycle) does, and in a table aligns it and goes to the next cell", func(c sdk.Call) {
		if _, ok := heading(c.Cursor); ok {
			sdk.Do(api.Result{Keys: "<Plug>(fold.cycle)"})
		} else {
			moveCell(c.Cursor, false)
		}
	})
	key("cycle_all", "<S-Tab>", "in a table aligns it and goes to the cell before, and elsewhere shows the outline, then every heading, then everything", func(c sdk.Call) {
		if !moveCell(c.Cursor, true) {
			sdk.Do(api.Result{Keys: "<Plug>(fold.cycle_all)"})
		}
	})
	for _, k := range []string{"<Tab>", "<S-Tab>"} {
		sdk.Register(api.Registration{Kind: api.Typed, Name: k, Help: "in an org table, aligns it and goes to the next cell, or the one before"}, func(cursor, _, _ int) int {
			if sdk.Current().Filetype == "org" {
				moveCell(cursor, k == "<S-Tab>")
			}
			return 0
		})
	}
	key("table", "<Leader>o|", "aligns the table, or makes the line one, its cells separated by commas or tabs", func(c sdk.Call) { alignTable(c.Cursor) })
	key("table_rule", "<Leader>o-", "adds a rule under the table's row", func(c sdk.Call) { addRule(c.Cursor) })
	key("table_calc", "<Leader>o=", "works out the formulas of the #+TBLFM: line under the table", func(c sdk.Call) { report(recalc(c.Cursor)) })
	key("todo_next", "cit", "puts the next TODO keyword on the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			cycleTodo(c.Cursor, h, false)
		}
	})
	key("todo_prev", "ciT", "puts the TODO keyword before on the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); ok {
			cycleTodo(c.Cursor, h, true)
		}
	})
	key("checkbox", "<C-Space>", "ticks the checkbox of the list item, or clears it, and counts the section's checkboxes again", func(c sdk.Call) {
		n, _ := sdk.LineOf(c.Cursor)
		e, ok := checkbox(lineAt(n))
		if !ok {
			return
		}
		h, ok := owner(c.Cursor)
		if !ok {
			edit(e)
			return
		}
		counted(c.Cursor, h, func(sub string) string {
			at := e.Start - h.Start
			return sub[:at] + e.Text + sub[e.End-h.Start:]
		})
	})
	key("count", "<Leader>o#", "counts again the checkboxes and TODOs the section's [/] and [%] count", func(c sdk.Call) {
		if h, ok := owner(c.Cursor); ok {
			counted(c.Cursor, h, func(sub string) string { return sub })
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
	key("next_heading", "]]", "goes to the next heading, or the n-th", func(c sdk.Call) {
		n, _ := sdk.LineOf(c.Cursor)
		goTo(headingFrom(n, max(c.Count, 1), false))
	})
	key("prev_heading", "[[", "goes to the heading above, or the n-th", func(c sdk.Call) {
		n, _ := sdk.LineOf(c.Cursor)
		goTo(headingFrom(n, max(c.Count, 1), true))
	})
	key("next_sibling", "][", "goes to the next heading of the section's level, or the n-th", func(c sdk.Call) {
		goTo(siblings(c, true))
	})
	key("prev_sibling", "[]", "goes to the heading of the section's level above, or the n-th", func(c sdk.Call) {
		goTo(siblings(c, false))
	})
	key("parent", "g{", "goes to the section's heading, or from a heading to the one it is under", func(c sdk.Call) {
		h, ok := owner(c.Cursor)
		if n, _ := sdk.LineOf(c.Cursor); ok && n == h.Line {
			h, ok = parent(h)
		}
		goTo(h, ok)
	})
	key("meta_return", "<Leader><CR>", "adds a heading after the heading's subtree, or an item after the list item", func(c sdk.Call) {
		n, _ := sdk.LineOf(c.Cursor)
		_, text := lineAt(n)
		if it, ok := parseItem(text); ok {
			insertLine(itemEnd(n, it), it.next())
		} else if h, ok := headingAt(n); ok {
			addHeading(c.Cursor, h, h.Level, "")
		}
	})
	key("insert_heading", "<Leader>oih", "adds a heading of the section's level after its subtree", func(c sdk.Call) {
		h, _ := owner(c.Cursor)
		addHeading(c.Cursor, h, max(h.Level, 1), "")
	})
	key("insert_todo", "<Leader>oit", "adds a TODO heading of the section's level after its subtree", func(c sdk.Call) {
		h, _ := owner(c.Cursor)
		t, _ := bufferKeys()
		addHeading(c.Cursor, h, max(h.Level, 1), t[0])
	})
	key("insert_subheading", "<Leader>ois", "adds a heading under the section's heading, after its subtree", func(c sdk.Call) {
		h, _ := owner(c.Cursor)
		addHeading(c.Cursor, h, h.Level+1, "")
	})
	key("shift_up", "<S-Up>", "moves the part of the timestamp under the cursor later, or raises the heading's priority, none, C, B, A", func(c sdk.Call) {
		if _, ok := heading(c.Cursor); !shiftStamp(c.Cursor, max(c.Count, 1), false) && ok {
			changeFront(c.Cursor, func(f *Front) { f.Head.Priority = stepPriority(f.Head.Priority, 1) })
		}
	})
	key("shift_down", "<S-Down>", "moves the part of the timestamp under the cursor earlier, or lowers the heading's priority, A, B, C, none", func(c sdk.Call) {
		if _, ok := heading(c.Cursor); !shiftStamp(c.Cursor, -max(c.Count, 1), false) && ok {
			changeFront(c.Cursor, func(f *Front) { f.Head.Priority = stepPriority(f.Head.Priority, -1) })
		}
	})
	key("shift_right", "<S-Right>", "moves the timestamp under the cursor a day later, or puts the next TODO keyword on the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); !shiftStamp(c.Cursor, max(c.Count, 1), true) && ok {
			cycleTodo(c.Cursor, h, false)
		}
	})
	key("shift_left", "<S-Left>", "moves the timestamp under the cursor a day earlier, or puts the TODO keyword before on the heading", func(c sdk.Call) {
		if h, ok := heading(c.Cursor); !shiftStamp(c.Cursor, -max(c.Count, 1), true) && ok {
			cycleTodo(c.Cursor, h, true)
		}
	})
	key("increment", "<C-a>", "moves the part of the timestamp under the cursor later, n of its unit", func(c sdk.Call) {
		shiftStamp(c.Cursor, max(c.Count, 1), false)
	})
	key("decrement", "<C-x>", "moves the part of the timestamp under the cursor earlier, n of its unit", func(c sdk.Call) {
		shiftStamp(c.Cursor, -max(c.Count, 1), false)
	})
	key("schedule", "<Leader>os", "asks when the section is scheduled, such as fri, +3d or 2026-11-02 10:00 +1w", func(c sdk.Call) { setDate(c.Cursor, 2) })
	key("deadline", "<Leader>od", "asks when the section is due, such as fri, +3d or 2026-11-02", func(c sdk.Call) { setDate(c.Cursor, 1) })
	key("timestamp", "<Leader>o.", "asks for a date and puts it at the cursor as a timestamp, today's for none", func(c sdk.Call) { insertStamp(c.Cursor, true) })
	key("inactive_timestamp", "<Leader>o!", "puts an inactive timestamp, which the agenda leaves out, at the cursor", func(c sdk.Call) { insertStamp(c.Cursor, false) })
	key("priority", "<Leader>o,", "asks for the section's priority, A, B, C or none", func(c sdk.Call) {
		ask("priority (A, B, C or none)", "", func(text string) error {
			p := strings.ToUpper(strings.TrimSpace(text))
			if p != "" && !strings.Contains("ABC", p) || len(p) > 1 {
				return errors.New("a priority is A, B, C or none")
			}
			changeFront(c.Cursor, func(f *Front) { f.Head.Priority = p })
			return nil
		})
	})
	key("tags", "<Leader>ot", "asks for the section's tags, separated by spaces or colons", func(c sdk.Call) {
		h, ok := owner(c.Cursor)
		if !ok {
			return
		}
		f, _ := front(h)
		ask("tags", strings.Join(f.Head.Tags, " "), func(text string) error {
			tags := strings.FieldsFunc(text, func(r rune) bool { return r == ' ' || r == ':' })
			changeFront(c.Cursor, func(f *Front) { f.Head.Tags = tags })
			return nil
		})
	})
	key("set_property", "<Leader>op", "asks for a property of the section, its name and its value", func(c sdk.Call) {
		ask("property (name value)", "", func(text string) error {
			name, value, _ := strings.Cut(strings.TrimSpace(text), " ")
			if name == "" || strings.TrimSpace(value) == "" {
				return errors.New("a property is a name and a value")
			}
			changeFront(c.Cursor, func(f *Front) { f.setProp(strings.ToUpper(name), strings.TrimSpace(value)) })
			return nil
		})
	})
	key("delete_property", "<Leader>oP", "asks for a property of the section by name, and removes it", func(c sdk.Call) {
		ask("remove property", "", func(text string) error {
			changeFront(c.Cursor, func(f *Front) { f.setProp(strings.TrimSpace(text), "") })
			return nil
		})
	})
	sdk.Define("opener", "open", "the program that opens a web link, open on macOS and xdg-open on Linux, which the plugin's programs must hold")
	key("follow", "<CR>", "follows the link under the cursor: a heading, a file, or a web link with the opener, or goes to the footnote's definition", func(c sdk.Call) {
		if !follow(c.Cursor) && !followCite(c.Cursor) {
			footnote(c.Cursor, true)
		}
	})
	key("footnote", "<Leader>of", "goes from a footnote to its definition and back, or adds one at the cursor", func(c sdk.Call) { footnote(c.Cursor, false) })
	key("store_link", "<Leader>ols", "stores a link to the section's heading for <Leader>oli", func(c sdk.Call) { report(store(c.Cursor)) })
	key("set_id", "<Leader>olI", "gives the section's heading an ID, for links that follow it to any file", func(c sdk.Call) { report(setID(c.Cursor)) })
	key("insert_link", "<Leader>oli", "asks for a link, from the one stored, and a description, and puts it at the cursor", func(c sdk.Call) { insertLink(c.Cursor) })
	sdk.On("buffer.opened", func(sdk.Event) { showLinks() })
	sdk.On("buffer.changed", func(sdk.Event) { showLinks() })
	key("refile", "<Leader>or", "asks for a heading of the org files and moves the section's subtree under it", func(c sdk.Call) { refile(c.Cursor) })
	key("archive", "<Leader>o$", "moves the section's subtree to the file's archive, <file>_archive", func(c sdk.Call) { report(archive(c.Cursor)) })
	key("clock_in", "<Leader>oxi", "clocks in the section's heading, and out the clock running", func(c sdk.Call) { report(clockIn(c.Cursor)) })
	key("clock_out", "<Leader>oxo", "clocks out the clock running", func(sdk.Call) { report(clockOut(false)) })
	key("clock_cancel", "<Leader>oxq", "cancels the clock running, taking its line away", func(sdk.Call) { report(clockOut(true)) })
	key("clock_goto", "<Leader>oxj", "goes to the heading the clock runs on", func(sdk.Call) { report(clockGoto()) })
	key("clock_table", "<Leader>oxr", "puts a clock table of the file at the cursor, or writes the one the cursor is in again", func(c sdk.Call) { clockTable(c.Cursor) })
	sdk.On("buffer.shown", func(sdk.Event) { showClock() })
	key("sparse", "<Leader>o/", "asks for text and folds the file so it shows the lines that hold it and the headings over them", func(sdk.Call) {
		ask("show", "", sparse)
	})
	key("narrow", "<Leader>ons", "opens the section's subtree in a buffer of its own, which :w writes back", func(c sdk.Call) { report(narrow(c.Cursor)) })
	for _, kind := range []string{"org-narrow", "org-src"} {
		sdk.Register(api.Registration{Kind: api.Write, Name: kind, Help: "writes a narrowed subtree or a source block back in place of the one it came from"}, func(_, _, _ int) int {
			report(writeBack())
			return 0
		})
	}
	sdk.Define("languages", "sh=sh bash=bash python=python3 js=node ruby=ruby", "the program that runs each language's source blocks, as language=program, which the plugin's programs must hold")
	key("run_block", "<Leader>obe", "runs the source block and puts what it prints under #+RESULTS:", func(c sdk.Call) { report(runBlock(c.Cursor)) })
	key("tangle", "<Leader>obt", "writes each source block with :tangle path to that file", func(sdk.Call) { report(tangle()) })
	key("edit_block", "<Leader>o'", "opens the source block in a buffer of its language, which :w writes back", func(c sdk.Call) { report(editBlock(c.Cursor)) })
	key("columns", "<Leader>oC", "shows the section's headings, or the file's, with the columns of its #+COLUMNS: line", func(c sdk.Call) { columnView(c.Cursor) })
	sdk.Define("bibliography", "", "BibTeX files every org file cites from, separated by spaces, besides those its #+bibliography: lines name")
	key("cite", "<Leader>o@", "asks for an entry of the bibliographies and cites it at the cursor, or adds it to the citation there", func(c sdk.Call) { report(cite(c.Cursor)) })
	key("attachments", "<Leader>oA", "asks whether to attach a file to the section's heading or open one of its attachments", func(c sdk.Call) { attachments(c.Cursor) })
	sdk.Define("pandoc", "pandoc", "the program export runs, which the plugin's programs must hold")
	sdk.Command("OrgExport", "writes the buffer beside its file in a format with pandoc: html, md, pdf, docx, odt, latex, epub or txt", func(c sdk.Call) { report(export(c.Args)) })
	sdk.Key("export", "<Leader>oE", "asks for a format and writes the buffer beside its file in it with pandoc", func(sdk.Call) { pickExport() })
	sdk.Object("ah", "a heading and its subtree", func(start, _ int) (int, int, bool) { return section(start, true) })
	sdk.Object("ih", "a heading's subtree, without the heading", func(start, _ int) (int, int, bool) { return section(start, false) })

	sdk.Define("directory", "~/org", "the folder whose org files the agenda gathers, which the plugin's paths must hold")
	sdk.Define("templates", "t inbox.org * TODO %?", "the capture templates, a line each: a key, a file with ::*heading after it or not, and the text")
	sdk.Define("views", "", "the agenda's own views, a line each: a key and a view, such as w tags work+urgent")
	sdk.Command("OrgAgenda", "shows the agenda view named: week and days, todo and keywords, tags and a match, search and text, or a key of the views option", func(c sdk.Call) { report(showView(c.Args)) })
	sdk.Key("agenda", "<Leader>oa", "asks for an agenda view, the week, every TODO or a match of tags, and shows it", func(sdk.Call) { pickView() })
	sdk.Plug("agenda_again", "gathers the agenda again and shows the view shown last", func(sdk.Call) { agenda() })
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
	sdk.Command("OrgCapture", "captures with the first template the text typed after it, or asks for a template and the text", func(c sdk.Call) {
		if c.Args == "" {
			askCapture(c.Cursor)
			return
		}
		ts, err := templates()
		if err == nil {
			err = capture(ts[0], c.Args, linkHere(c.Cursor))
		}
		report(err)
	})
	sdk.Key("capture", "<Leader>oc", "asks for a capture template and the text, and adds it to the template's file", func(c sdk.Call) { askCapture(c.Cursor) })
}

// builtins are the agenda's own views, and what each asks for.
var builtins = []struct{ name, asks string }{{"week", ""}, {"todo", ""}, {"tags", "match tags"}, {"search", "search for"}}

// customViews reads the views option: a view a line, a key and the view it
// shows, such as w tags work+urgent.
func customViews() map[string]string {
	v, _ := sdk.Option("fude-org.views")
	out := map[string]string{}
	for line := range strings.Lines(v) {
		if key, view, ok := strings.Cut(strings.TrimSpace(line), " "); ok {
			out[key] = strings.TrimSpace(view)
		}
	}
	return out
}

// showView shows the agenda view args names: week and a number of days,
// todo and keywords, tags and a match, search and text, or the key of a
// view of the views option.
func showView(args string) error {
	name, rest, _ := strings.Cut(strings.TrimSpace(args), " ")
	rest = strings.TrimSpace(rest)
	switch name {
	case "", "week":
		days := 7
		if rest != "" {
			n, err := strconv.Atoi(rest)
			if err != nil || n < 1 {
				return errors.New("week takes a number of days")
			}
			days = n
		}
		view = func() { week(days) }
	case "todo":
		keywords := strings.Fields(rest)
		view = func() { todos(keywords) }
	case "tags", "search":
		if rest == "" {
			return errors.New(name + " needs what to look for")
		}
		view = func() { tagged(rest) }
		if name == "search" {
			view = func() { search(rest) }
		}
	default:
		custom, ok := customViews()[name]
		if !ok {
			return errors.New("no agenda view " + name)
		}
		return showView(custom)
	}
	agenda()
	return nil
}

// pickView asks for an agenda view, its own or one of the views option,
// and what a view looks for, and shows it.
func pickView() {
	var items, names []string
	for _, b := range builtins {
		items, names = append(items, b.name), append(names, b.name)
	}
	custom := customViews()
	for _, key := range slices.Sorted(maps.Keys(custom)) {
		items, names = append(items, key+"  "+custom[key]), append(names, key)
	}
	report(sdk.Pick("agenda", items, func(i int, ok bool) {
		if !ok {
			return
		}
		if i < len(builtins) && builtins[i].asks != "" {
			ask(builtins[i].asks, "", func(text string) error { return showView(names[i] + " " + text) })
			return
		}
		report(showView(names[i]))
	}))
}

// ask asks for text, with query typed at the start, and runs answer with
// it once the user answers. answer's error shows as the call's.
func ask(prompt, query string, answer func(text string) error) {
	id := sdk.Callback(func() bool {
		if sdk.About("ok") == "true" {
			report(answer(sdk.About("text")))
		}
		return false
	})
	report(sdk.Do(api.Result{Pick: &api.Pick{ID: id, Prompt: prompt, Query: query}}))
}

// stepPriority returns the priority by steps after p, in the order none,
// C, B, A, and around again.
func stepPriority(p string, by int) string {
	order := []string{"", "C", "B", "A"}
	return order[(slices.Index(order, p)+by+len(order))%len(order)]
}

// report shows err, if any, as the call's error.
func report(err error) {
	if err != nil {
		sdk.Do(api.Result{Error: err.Error()})
	}
}

// goTo moves the cursor to h, or says there is no heading there.
func goTo(h Heading, ok bool) {
	if !ok {
		sdk.Do(api.Result{Error: "no heading there"})
		return
	}
	sdk.Do(api.Result{Jump: &api.Jump{Pos: h.Start}})
}

// siblings returns the n-th heading of the section's level after it, or
// before it.
func siblings(c sdk.Call, below bool) (Heading, bool) {
	h, ok := owner(c.Cursor)
	for range max(c.Count, 1) {
		if !ok {
			break
		}
		h, ok = sibling(h, below)
	}
	return h, ok
}

// addHeading adds a heading of level, with keyword, after h's subtree, or
// after the line that holds cursor when there is no h, and starts insert
// mode at its end.
func addHeading(cursor int, h Heading, level int, keyword string) {
	line := strings.Repeat("*", level) + " "
	if keyword != "" {
		line += keyword + " "
	}
	n, _ := sdk.Len()
	at := n
	switch end := subtree(h); {
	case h.Level == 0:
		l, _ := sdk.LineOf(cursor)
		start, text := lineAt(l)
		at = start + len(text)
	case end < n:
		at = end - 1
	}
	insertLine(at, line)
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
