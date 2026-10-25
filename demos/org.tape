# fude-org in half a minute: the outline folds and cycles, a TODO and a
# checkbox change, a subtree moves, and capture and the agenda work across
# the files in ~/org.
Output demos/org.gif
Output demos/out/org.mp4
Source demos/style.tape

Hide
Type "source demos/setup.sh"
Enter
Wait
# The org grammar builds out of view, so the clip has colours from its start.
Type "fude todo.org"
Enter
Wait+Screen@200s /built the org grammar/
Type ":q"
Enter
Wait
Type "clear"
Enter
Wait
Show
Sleep 500ms

Type "fude todo.org"
Enter
Sleep 2s

# <S-Tab> shows the top headings, then every heading, then everything.
Shift+Tab
Sleep 1.5s
Shift+Tab
Sleep 1.5s
Shift+Tab
Sleep 1.5s

# <Tab> on a heading closes it, opens it with its subheadings closed, and
# opens it all.
Tab
Sleep 1.2s
Tab
Sleep 1.2s
Tab
Sleep 1.5s

# cit marks the groceries done, and <C-Space> ticks the milk.
Type "4G"
Sleep 400ms
Type "cit"
Sleep 1.2s
Type "j"
Sleep 400ms
Ctrl+Space
Sleep 1.5s

# <Leader>oJ moves Projects and its subtree below Reading.
Type "9G"
Sleep 400ms
Type " oJ"
Sleep 2s

# Capture adds a TODO to ~/org/inbox.org from anywhere.
Type " oc"
Sleep 800ms
Type@80ms "Call the dentist"
Sleep 400ms
Enter
Sleep 1.5s
Type ":w"
Enter
Sleep 800ms

# The agenda gathers every file in ~/org: overdue, today, this week, and
# the TODOs with no date. t marks one done in its file, and <CR> opens one.
Type " oa"
Sleep 2.5s
Type "jjjj"
Sleep 600ms
Type "t"
Sleep 2s
Type "G"
Sleep 600ms
Enter
Sleep 2.5s
