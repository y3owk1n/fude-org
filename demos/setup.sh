# Sourced by org.tape in a hidden prelude, from the repo root inside its oku
# environment. It builds fude from the checkout beside this repo and the
# plugin from this one, and gives them a throwaway HOME under
# /tmp/fude-org-demo with a folder of org notes. A recording never touches
# your own config, state or notes.

repo=$PWD
root=/tmp/fude-org-demo
grammars=${XDG_DATA_HOME:-$HOME/.local/share}/fude/grammars

chmod -R u+rwX $root 2>/dev/null
rm -rf $root
mkdir -p $root/bin $root/run $root/.config/fude $root/org/work
(cd $repo/../fude && go build -o $root/bin/fude ./cmd/fude) || return 1
$root/bin/fude plugin build $repo >/dev/null || return 1
ln -s $(command -v git) $root/bin/git
ln -s $(command -v tree-sitter) $root/bin/tree-sitter
# A copy of your built grammars, so the org grammar the tape builds goes to
# the copy.
mkdir -p $root/.local/share/fude
[ -d $grammars ] && cp -R $grammars $root/.local/share/fude/grammars

cat > $root/.config/fude/config.toml <<TOML
number = true
cursor_line = true
icons = true

[[plugin]]
path = "$repo/fude-org.wasm"
capabilities = ["buffer.read", "buffer.write", "ui", "fs.read", "fs.write", "keys", "grammar"]
paths = ["~/org"]
TOML

# The notes, dated from today so the agenda has something due.
day() { date -u -v${1}d +%F 2>/dev/null || date -u -d "$1 days" +%F; }
cat > $root/org/todo.org <<ORG
* TODO Plan the week :home:
SCHEDULED: <$(day +0)>
Notes on what matters most.
** TODO Groceries
- [ ] milk
- [ ] bread
- [X] coffee
** Errands
* Projects
** TODO Write the fude-org README
** TODO Record a demo
SCHEDULED: <$(day +2)>
* Reading
** The Pragmatic Programmer
ORG
cat > $root/org/work/plans.org <<ORG
* TODO Quarterly report
DEADLINE: <$(day -1)>
* Team offsite
SCHEDULED: <$(day +4)>
ORG

export HOME=$root
export XDG_CONFIG_HOME=$root/.config
export XDG_STATE_HOME=$root/.local/state
export XDG_DATA_HOME=$root/.local/share
export XDG_CACHE_HOME=$root/.cache
export XDG_RUNTIME_DIR=$root/run
export PATH=$root/bin:/usr/bin:/bin:/usr/sbin:/sbin
export TERM=xterm-256color COLORTERM=truecolor SHELL=/bin/zsh
PROMPT='%F{blue}%~%f %F{magenta}❯%f '
cd $root/org
printf '\033[2J\033[3J\033[H'
