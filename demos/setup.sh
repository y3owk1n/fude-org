# Sourced by each tape in a hidden prelude, from the repo root inside its
# oku environment, as `source demos/setup.sh <clip>`. It builds the plugin
# with the fude on PATH, and gives them a throwaway HOME under
# /tmp/fude-org-demo with the clip's org notes.
# A recording never touches your own config, state or notes.

repo=$PWD
clip=$1
root=/tmp/fude-org-demo
grammars=${XDG_DATA_HOME:-$HOME/.local/share}/fude/grammars
# Grammars for the clips, kept between recordings: a copy of yours and the
# org grammar.
cache=$repo/demos/out/grammars
pin=$(sed -n 's/.*Commit: *"\([0-9a-f]*\)".*/\1/p' $repo/main.go)

chmod -R u+rwX $root 2>/dev/null
rm -rf $root
mkdir -p $root/bin $root/run $root/.config/fude $root/.local/share/fude $root/org/work
fude=$(command -v fude) || { echo "fude is not on PATH" >&2; return 1; }
ln -s $fude $root/bin/fude
$root/bin/fude plugin build $repo >/dev/null || return 1
ln -s $(command -v git) $root/bin/git
ln -s $(command -v tree-sitter) $root/bin/tree-sitter
if [ ! -d $cache ]; then
  mkdir -p $cache
  [ -d $grammars ] && cp -R $grammars/. $cache
fi
cp -R $cache $root/.local/share/fude/grammars

cat > $root/.config/fude/config.toml <<TOML
number = true
cursor_line = true
icons = true

[[plugin]]
path = "$repo/fude-org.wasm"
capabilities = ["buffer.read", "buffer.write", "ui", "fs.read", "fs.write", "process", "keys", "grammar"]
programs = ["sh"]
paths = ["~/org"]
TOML

export HOME=$root
export XDG_CONFIG_HOME=$root/.config
export XDG_STATE_HOME=$root/.local/state
export XDG_DATA_HOME=$root/.local/share
export XDG_CACHE_HOME=$root/.cache
export XDG_RUNTIME_DIR=$root/run

# The org grammar builds once, in a tmux session out of view, so a clip has colours
# from its start. It needs git, tree-sitter and the network.
built() { [ -n "$(find $1/org -name "$pin-*.so" 2>/dev/null)" ]; }
if ! built $cache; then
  touch $root/org/build.org
  tmux -L fude-org-demo new-session -d -x 100 -y 30 "$root/bin/fude $root/org/build.org" || return 1
  for _ in $(seq 600); do
    built $root/.local/share/fude/grammars && break
    sleep 0.5
  done
  # The compiler may still be writing the library.
  sleep 1
  tmux -L fude-org-demo kill-server
  rm $root/org/build.org
  built $root/.local/share/fude/grammars || return 1
  rm -rf $cache/org
  cp -R $root/.local/share/fude/grammars/org $cache/org
fi

# The clip's notes, dated from today so the agenda has something due.
day() { LC_ALL=C date -u -v${1}d "+${2:-%F}" 2>/dev/null || LC_ALL=C date -u -d "$1 days" "+${2:-%F}"; }
case $clip in
org)
  cat > $root/org/todo.org <<ORG
* TODO Plan the week :home:
SCHEDULED: <$(day +0)>
Notes on what matters most.
** TODO Groceries [1/3]
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
  ;;
tables)
  cat > $root/org/trip.org <<'ORG'
#+COLUMNS: %28ITEM %TODO %Effort(Effort){:}
* Budget
| Item | Qty | Price | Total |
|------+-----+-------+-------|
|
#+TBLFM: $4=$2*$3::@>$4=vsum(@2..@-1)
* Packing
** TODO Book the train
:PROPERTIES:
:Effort:   0:30
:END:
** TODO Pack the bags
:PROPERTIES:
:Effort:   1:15
:END:
** DONE Find a hotel
:PROPERTIES:
:Effort:   2:00
:END:
ORG
  ;;
code)
  cat > $root/org/scripts.org <<'ORG'
* What is in my notes
#+begin_src sh
ls
#+end_src

* Greet
#+begin_src sh :tangle greet.sh
echo "hello from org"
#+end_src
ORG
  ;;
time)
  cat > $root/org/work.org <<ORG
* Writing
** TODO Draft the chapter
:LOGBOOK:
CLOCK: [$(day -1 '%F %a') 09:00]--[$(day -1 '%F %a') 10:30] =>  1:30
:END:
** TODO Edit the intro
:LOGBOOK:
CLOCK: [$(day -2 '%F %a') 14:00]--[$(day -2 '%F %a') 14:45] =>  0:45
:END:
* Home
** TODO Water the plants
SCHEDULED: <$(day +0 '%F %a') +3d>
* Report

ORG
  ;;
notes)
  cat > $root/org/notes.org <<'ORG'
* Ideas
See [[file:work/plans.org::*Team offsite][the offsite plan]] and [[*Reading list][the reading list]].
** Plain text
Plain text outlives the programs that write it.
** Agenda for the offsite
- [ ] goals for next year
- [ ] who does what
* Reading list
** TODO The Pragmatic Programmer :books:
** A Philosophy of Software Design :books:
** TODO Designing Data-Intensive Applications :books:
ORG
  cat > $root/org/work/plans.org <<ORG
* TODO Quarterly report
DEADLINE: <$(day -1 '%F %a')>
* Team offsite
SCHEDULED: <$(day +4 '%F %a')>
ORG
  ;;
esac

export PATH=$root/bin:/usr/bin:/bin:/usr/sbin:/sbin
export TERM=xterm-256color COLORTERM=truecolor SHELL=/bin/zsh
PROMPT='%F{blue}%~%f %F{magenta}❯%f '
cd $root/org
printf '\033[2J\033[3J\033[H'
