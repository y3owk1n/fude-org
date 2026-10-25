# fude-org in under a minute: the outline folds and cycles, a TODO, a
# checkbox and a date change, a subtree moves, and capture and the agenda's
# week work across the files in ~/org.
Output demos/org.gif
Output demos/out/org.mp4
Source demos/style.tape

Hide
Type "source demos/setup.sh org"
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

# cit marks the groceries done with the time, and <C-Space> ticks the
# milk and counts again.
Type "4G"
Sleep 400ms
Type "cit"
Sleep 1.2s
Type "jj"
Sleep 400ms
Ctrl+Space
Sleep 1.5s

# <Leader>os schedules Errands, from a date typed as org reads one.
Type "/Errands"
Enter
Sleep 400ms
Type " os"
Wait+Screen@10s /scheduled/
Type@80ms "fri 10:00"
Enter
Sleep 600ms
Type ":noh"
Enter
Sleep 1s

# <Leader>oJ moves Projects and its subtree below Reading.
Type "/^\\* Projects"
Enter
Sleep 400ms
Type ":noh"
Enter
Sleep 400ms
Type " oJ"
Sleep 2s

# Capture adds a TODO to ~/org/inbox.org from anywhere.
Type " oc"
Wait+Screen@10s /capture/
Type@80ms "Call the dentist"
Sleep 400ms
Enter
Sleep 1.5s
Type ":w"
Enter
Sleep 800ms

# The agenda's week gathers every file in ~/org: what is overdue, then
# each day. t marks one done in its file, and <CR> opens one.
Type " oa"
Wait+Screen@10s /agenda/
Enter
Sleep 2.5s
Type "jjj"
Sleep 600ms
Type "t"
Sleep 2s
Type "ggj"
Sleep 600ms
Enter
Sleep 2.5s
