# fude-org

Org mode for fude, as a plugin. It has fude build the org grammar, edits
headings, TODOs and checkboxes, and shows an agenda of the org files in one
folder, with a capture into its inbox.

![fude-org cycling folds, marking a TODO done, moving a subtree, capturing and showing the agenda](demos/org.gif)

## Build

```sh
oku sync && oku allow   # Go, TinyGo and binaryen, as fude pins them
fude plugin build       # writes fude-org.wasm
```

It needs fude's API after v0.3.0, so `go.mod` points at a fude checkout
beside this one, `../fude`, until the next release.

## Install

Add this table to fude's `config.toml`:

```toml
[[plugin]]
path = "/path/to/fude-org/fude-org.wasm"
capabilities = ["buffer.read", "buffer.write", "ui", "fs.read", "fs.write", "process", "keys", "grammar"]
programs = ["open", "pandoc", "sh", "python3", "node"]
paths = ["~/org"]
```

- `grammar` lets fude build the org grammar from
  [nvim-orgmode/tree-sitter-org](https://github.com/nvim-orgmode/tree-sitter-org).
  The grammar runs in fude itself, outside the plugin's sandbox. With
  `auto_install` on, it builds when you open an org file. Otherwise
  `:grammar install org` builds it.
- `paths` is the folder of your org files, which the agenda reads and
  capture writes to.
- `process` and `programs` let a web link open with `open`, or with
  `xdg-open` on Linux, set as the `opener` option too, export run
  pandoc, and source blocks run their languages. Leave out what you do
  not use.
- `keys` lets `<Tab>` and `<S-Tab>` run fude's fold cycling. A guest of a
  shared session may not use a plugin granted `keys`.

Keep the file named `fude-org.wasm`. Its options and the agenda's keys are
named after it.

## Use

In an org file:

| Keys | What they do |
|---|---|
| `<Tab>` | on a heading, steps its section from closed, to open with its subheadings closed, to all open. In a table, in insert mode too, aligns it and goes to the next cell, adding a row past the last |
| `<S-Tab>` | steps the file from top headings, to every heading, to everything. In a table it goes to the cell before |
| `<Leader>o\|` | aligns the table, or makes the line one, its cells separated by commas or tabs |
| `<Leader>o-` | adds a rule under the table's row |
| `<Leader>o=` | works out the formulas of the `#+TBLFM:` line under the table |
| `cit`, `ciT` | puts the next or the previous TODO keyword on the heading |
| `<C-Space>` | ticks the list item's checkbox, or clears it |
| `>>`, `<<` | adds a star to the heading, or takes one off |
| `>s`, `<s` | does the same to the heading and every heading under it |
| `<Leader>oJ`, `<Leader>oK` | moves the heading and its subtree past the next or the previous heading of its level |
| `ah`, `ih` | a heading and its subtree, or the subtree without the heading, for an operator such as `d` or `y` |
| `]]`, `[[` | goes to the next or the previous heading |
| `][`, `[]` | goes to the next or the previous heading of the section's level |
| `g{` | goes to the section's heading, or from a heading to the one it is under |
| `<Leader><CR>` | adds a heading after the heading's subtree, or an item after the list item |
| `<Leader>oih`, `<Leader>oit`, `<Leader>ois` | adds a heading, a TODO heading or a subheading after the section's subtree |
| `<S-Up>`, `<S-Down>` | raises or lowers the heading's priority, or moves the part of a timestamp under the cursor |
| `<S-Right>`, `<S-Left>` | puts the next or the previous TODO keyword on the heading, or moves a timestamp a day |
| `<C-a>`, `<C-x>` | moves the part of a timestamp under the cursor, n of its unit |
| `<Leader>o,` | asks for the section's priority |
| `<Leader>ot` | asks for the section's tags, from those it has |
| `<Leader>op`, `<Leader>oP` | sets a property in the section's drawer, or removes one |
| `<Leader>os`, `<Leader>od` | asks when the section is scheduled or due |
| `<Leader>o.`, `<Leader>o!` | puts an active or an inactive timestamp at the cursor |
| `<Leader>o/` | asks for text and folds the file to show the lines that hold it and the headings over them |
| `<Leader>ons` | opens the section's subtree alone in a buffer of its own, which `:w` writes back in its place |
| `<Leader>oC` | shows the section's headings, or the file's, as a table with the columns of the file's `#+COLUMNS:` line |
| `<Leader>obe` | runs the source block and puts what it prints under `#+RESULTS:` |
| `<Leader>obt` | writes each source block with `:tangle path` to that file |
| `<Leader>o'` | opens the source block in a buffer of its language, which `:w` writes back |
| `<Leader>o#` | counts again what the section's `[/]` and `[%]` count |
| `<CR>` | follows the link under the cursor, opens the entry of the citation key, or goes to the footnote's definition |
| `<Leader>o@` | asks for an entry of the bibliographies and cites it at the cursor, or adds it to the citation there |
| `<Leader>of` | goes from a footnote to its definition and back, or adds a footnote at the cursor, its definition under `* Footnotes` |
| `<Leader>oxi`, `<Leader>oxo` | clocks in the section's heading, clocking out the clock running, or clocks out |
| `<Leader>oxq`, `<Leader>oxj` | cancels the clock running, or goes to its heading |
| `<Leader>oxr` | puts a clock table of the file at the cursor, or writes the one the cursor is in again |
| `<Leader>or` | asks for a heading of the org files, or a file, and moves the section's subtree under it |
| `<Leader>o$` | moves the section's subtree to the end of the file's archive, `<file>_archive` |
| `<Leader>olI` | gives the section's heading an `:ID:`, for links that find it in any org file |
| `<Leader>ols`, `<Leader>oli` | stores a link to the section's heading, or asks for a link, from the one stored, and a description, and puts it at the cursor |

Off a heading, a list item or a timestamp, each key does what it does in
any other file, so `<Tab>` still jumps forward, `>>` still indents and
`<C-a>` still adds to a number.

Type a date as in org: `2026-11-02`, `+3d`, `-1w`, `today`, `tomorrow` or a
weekday such as `fri`. A time such as `10:00`, a repeater such as `+1w` and
a warning such as `-2d` may follow it. An empty answer removes the date.

A heading put to a done keyword gets `CLOSED:` with the time. One whose
`SCHEDULED:` or `DEADLINE:` date repeats moves that date on instead and
keeps its TODO keyword: `+1w` a week, `++1w` past today, `.+1w` a week from
today. `LAST_REPEAT` notes when. A file's own `#+TODO:` line sets its
keywords.

A `[/]` or `[%]` in a heading counts the checkboxes in its text, or with
none, its children with a TODO keyword. In a list item it counts the
checkboxes of the items under it. `<C-Space>` and `cit` count again as they
change one.

A sparse tree closes every fold with `zM`, then opens with `zv` each fold
that hides a matching line, so `zR` shows everything again. A section
opened to reach a match also shows its own text, which org would hide,
because a fude fold holds a whole section.

Column view's columns come from `#+COLUMNS:`, such as
`%25ITEM %TODO %Effort(Time){:} %COST{+}`, or else org's default,
`%25ITEM %TODO %3PRIORITY %TAGS`. A column is `ITEM`, `TODO`, `PRIORITY`,
`TAGS` or a property, with its title in brackets. `{+}` sums the numbers
of a heading's children and `{:}` their times. `<CR>` on a row goes to its
heading.

A citation is org's, `[cite:@key]` or `[cite/style:@a;@b]`. Its entries
come from the BibTeX files of the file's `#+bibliography:` lines, beside
it, and of the `bibliography` option. Export renders citations with
pandoc's `--citeproc`.

A source block runs with the program the `languages` option gives its
language, its text the program's input. What it prints replaces the
block's `#+RESULTS:`, a line of `: ` each. Tangling writes the blocks of
one file in their order, beside the org file.

A table's column is to the right when more than half its cells are
numbers. Its formulas work as in org. `$4=$2*$3` sets a column under the
first rule, and `@4$2=vsum(@2..@3)` sets one field. A formula has
`+ - * / ^`, brackets, `$c`, `@r`, `@r$c`, `@<` and `@>` for the first and
last row, `@-1` for the row above, and `vsum`, `vmean`, `vmin`, `vmax` and
`vcount` of a range. Row numbers skip rules, as in org.

Clocking in adds `CLOCK: [start]` to the heading's `:LOGBOOK:` drawer, and
clocking out closes it as `CLOCK: [start]--[end] =>  1:05`. The status line
shows the task and how long it has run. A clock table sums the closed
clocks of each heading and the headings under it, between `#+BEGIN:
clocktable` and `#+END:`.

A link is `[[target][description]]` or `[[target]]`. Away from the
cursor's line it shows only its description, or its target when it has
none. A target is `*title` for a heading of the file, `#id` for a
heading's `CUSTOM_ID`, `id:` and a heading's `:ID:` in any org file, a
web address, or a file, with `file:` before it or not. A stored link to a
heading with an `:ID:` is by its ID. After a file, `::*title`, `::#id`, a line number or text says where
to go in it.

Refile lists the headings of the buffer as it is and of the files the
agenda reads. A subtree takes the level under the heading it goes to. A
subtree moved to another file goes through `fs.write`, which refuses a
file open with changes not written. Archive keeps where the subtree came from
in its drawer: `ARCHIVE_TIME`, `ARCHIVE_FILE`, `ARCHIVE_OLPATH`,
`ARCHIVE_CATEGORY` and `ARCHIVE_TODO`.

Anywhere:

| Keys | Command | What it does |
|---|---|---|
| `<Leader>oa` | `:OrgAgenda [VIEW]` | asks for an agenda view, or shows the one named |
| `<Leader>oE` | `:OrgExport [FORMAT]` | asks for a format, or takes the one named, and writes the buffer beside its file in it with pandoc: `html`, the default, `md`, `pdf`, `docx`, `odt`, `latex`, `epub` or `txt` |
| `<Leader>oc` | `:OrgCapture [text]` | asks for a capture template and the text, or captures the text typed after it with the first template |

An agenda view is `week` and a number of days, 7 by default, `todo` and
keywords to show only those, `tags` and a match, `search` and text, which
lists each line of the org files that holds it, or a key of the `views`
option. Each line of `views` is a key and a view:

```toml
options = { views = """
n todo NEXT WAIT
f week 14
w tags work-meeting
""" }
```

The week shows what is overdue and not done, then each of the next seven
days with what is scheduled or due on it, by time. A date that repeats
shows on each day it repeats to. Tag matches work as in org.
`work+urgent-home` needs work and urgent and not home, and `work|home`
either. A heading has the tags of the headings it is under too.

In the agenda, `<CR>` opens the heading on the cursor's line, `t` puts the
next TODO keyword on it in its file, and `r` gathers the view again.

A capture template is a line of the `templates` option: a key, a file in
the directory, with `::*heading` after it to go under that heading or not,
and the text. In the text, `%?` stands for what you type, `%t` and `%T`
for today's date and the time, `%u` and `%U` for them inactive, `%a` for a
link to where the capture began, and `\n` for a new line. An entry under a
heading takes the level below it.

```toml
options = { templates = """
t inbox.org * TODO %?
j journal.org::*Journal * %U %?\n%a
""" }
```

## Options

Set them in the table's `options`, such as
`options = { directory = "~/notes" }`, or with
`:set fude-org.directory ~/notes`.

| Option | Default | What it is |
|---|---|---|
| `todo` | `TODO \| DONE` | the TODO keywords, those after `\|` done, for files with no `#+TODO:` line. The plugin reads it when it loads. |
| `log_done` | `true` | a heading put to a done keyword gets `CLOSED:` with the time |
| `opener` | `open` | the program that opens a web link, which `programs` must hold |
| `directory` | `~/org` | the folder the agenda reads, which `paths` must hold |
| `views` | none | the agenda's own views, a line each: a key and a view |
| `pandoc` | `pandoc` | the program export runs, which `programs` must hold |
| `bibliography` | none | BibTeX files every org file cites from, separated by spaces |
| `languages` | `sh=sh bash=bash python=python3 js=node ruby=ruby` | the program that runs each language's source blocks, which `programs` must hold |
| `templates` | `t inbox.org * TODO %?` | the capture templates, a line each |

## Not yet

Attachments.

## Check

```sh
fude plugin build && go test ./test/
```

## Demo

`demos/org.tape` records the clip above with
[VHS](https://github.com/charmbracelet/vhs): `vhs demos/org.tape` from the
repo root, with VHS, ttyd and ffmpeg on PATH. `demos/setup.sh` runs hidden
first. It builds fude from `../fude` and this plugin, and gives them a
throwaway HOME under `/tmp/fude-org-demo` with notes dated from today, so
a recording never touches your config or your notes. The tape builds the
org grammar out of view, which needs git, tree-sitter and the network.
