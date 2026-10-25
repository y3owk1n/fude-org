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
programs = ["open"]
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
  `xdg-open` on Linux, set as the `opener` option too.
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
| `<Leader>o#` | counts again what the section's `[/]` and `[%]` count |
| `<CR>` | follows the link under the cursor, or goes to the footnote's definition |
| `<Leader>of` | goes from a footnote to its definition and back, or adds a footnote at the cursor, its definition under `* Footnotes` |
| `<Leader>oxi`, `<Leader>oxo` | clocks in the section's heading, clocking out the clock running, or clocks out |
| `<Leader>oxq`, `<Leader>oxj` | cancels the clock running, or goes to its heading |
| `<Leader>oxr` | puts a clock table of the file at the cursor, or writes the one the cursor is in again |
| `<Leader>or` | asks for a heading of the org files, or a file, and moves the section's subtree under it |
| `<Leader>o$` | moves the section's subtree to the end of the file's archive, `<file>_archive` |
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

A table's column is to the right when more than half its cells are
numbers. Its formulas are org's: `$4=$2*$3` sets a column under the first
rule, and `@4$2=vsum(@2..@3)` one field. A formula knows `+ - * / ^`,
brackets, `$c`, `@r`, `@r$c`, `@<` and `@>` for the first and last row,
`@-1` for the row above, and `vsum`, `vmean`, `vmin`, `vmax` and `vcount`
of a range. Rows count past rules, as in org.

A clock goes in the heading's `:LOGBOOK:` drawer as `CLOCK: [start]`, and
clocking out closes it as `CLOCK: [start]--[end] =>  1:05`. The status line
shows the task and how long it has run. A clock table sums the closed
clocks of each heading and the headings under it, between `#+BEGIN:
clocktable` and `#+END:`.

A link is `[[target][description]]` or `[[target]]`. Away from the
cursor's line it shows only its description, or its target when it has
none. A target is `*title` for a heading of the file, `#id` for a
heading's `CUSTOM_ID`, a web address, or a file, with `file:` before it or
not. After a file, `::*title`, `::#id`, a line number or text says where
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
| `<Leader>oa` | `:OrgAgenda [week\|todo\|tags MATCH]` | asks for an agenda view, or shows the one named: the week, every TODO, or the headings whose tags match |
| `<Leader>oc` | `:OrgCapture [text]` | asks for a capture template and the text, or captures the text typed after it with the first template |

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
| `templates` | `t inbox.org * TODO %?` | the capture templates, a line each |

## Not yet

Column view, sparse trees, narrowing, ID
links, custom agenda views, export, code blocks, citations and
attachments.

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
