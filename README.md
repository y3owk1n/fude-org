# fude-org

Org mode for fude, as a plugin. It has fude build the org grammar, edits
headings, TODOs and checkboxes, and shows an agenda of the org files in one
folder, with a capture into its inbox.

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
capabilities = ["buffer.read", "buffer.write", "ui", "fs.read", "fs.write", "keys", "grammar"]
paths = ["~/org"]
```

- `grammar` lets fude build the org grammar from
  [nvim-orgmode/tree-sitter-org](https://github.com/nvim-orgmode/tree-sitter-org).
  The grammar runs in fude itself, outside the plugin's sandbox. With
  `auto_install` on, it builds when you open an org file. Otherwise
  `:grammar install org` builds it.
- `paths` is the folder of your org files, which the agenda reads and
  capture writes to.
- `keys` lets `<Tab>` and `<S-Tab>` run fude's fold cycling. A guest of a
  shared session may not use a plugin granted `keys`.

Keep the file named `fude-org.wasm`. Its options and the agenda's keys are
named after it.

## Use

In an org file:

| Keys | What they do |
|---|---|
| `<Tab>` | on a heading, steps its section from closed, to open with its subheadings closed, to all open |
| `<S-Tab>` | steps the file from top headings, to every heading, to everything |
| `cit`, `ciT` | puts the next or the previous TODO keyword on the heading |
| `<C-Space>` | ticks the list item's checkbox, or clears it |
| `>>`, `<<` | adds a star to the heading, or takes one off |
| `>s`, `<s` | does the same to the heading and every heading under it |
| `<Leader>oJ`, `<Leader>oK` | moves the heading and its subtree past the next or the previous heading of its level |
| `ah`, `ih` | a heading and its subtree, or the subtree without the heading, for an operator such as `d` or `y` |

Off a heading or a list item, each key does what it does in any other
file, so `<Tab>` still jumps forward and `>>` still indents.

Anywhere:

| Keys | Command | What it does |
|---|---|---|
| `<Leader>oa` | `:OrgAgenda` | shows what is overdue, due today, this week and later, then the TODOs with no date |
| `<Leader>oc` | `:OrgCapture [text]` | adds a TODO to the inbox, the text typed after it or asked for |

In the agenda, `<CR>` opens the heading on the cursor's line, `t` puts the
next TODO keyword on it in its file, and `r` gathers the agenda again.

## Options

Set them in the table's `options`, such as
`options = { directory = "~/notes", inbox = "~/notes/inbox.org" }`, or with
`:set fude-org.directory ~/notes`.

| Option | Default | What it is |
|---|---|---|
| `todo` | `TODO \| DONE` | the TODO keywords, those after `\|` done. The plugin reads it when it loads. |
| `directory` | `~/org` | the folder the agenda reads, which `paths` must hold |
| `inbox` | `~/org/inbox.org` | the file capture adds to |

## Not yet

The agenda lists TODO headings, and headings with a `SCHEDULED:` or
`DEADLINE:` date on the line after them. It has no repeaters, clocking,
tags view or custom views. There are no tables, properties, links, export
or code blocks yet.

"Today" is today in UTC, as a plugin's clock is, so near midnight it may
be a day off from yours.

## Check

```sh
fude plugin build && go test ./test/
```
