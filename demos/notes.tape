# Notes in fude-org: links show their descriptions and <CR> follows them,
# into another file or to a heading, a footnote goes under Footnotes, a
# sparse tree shows only what matches, and refile moves a subtree into
# another file.
Output demos/notes.gif
Output demos/out/notes.mp4
Source demos/style.tape

Hide
Type "source demos/setup.sh notes"
Enter
Wait
Show
Sleep 500ms

Type "fude notes.org"
Enter
Sleep 2s

# A link shows its text away from the cursor. <CR> on one follows it to a
# heading of another file, and <C-o> comes back.
Type "2Gw"
Sleep 1.2s
Enter
Sleep 2s
Ctrl+O
Sleep 1.2s

# A link to a heading of the file goes there.
Type "$F["
Sleep 600ms
Enter
Sleep 1.5s

# <Leader>of adds a footnote under Footnotes, and goes back to it.
Type "4G$"
Sleep 400ms
Type " of"
Sleep 800ms
Type@70ms "Org files from decades ago still open."
Escape
Sleep 1.2s
Type " of"
Sleep 1.5s

# <Leader>o/ folds the file to what mentions the offsite.
Type " o/"
Wait+Screen@10s /show/
Type@80ms "offsite"
Enter
Sleep 2.5s
Type "zR"
Sleep 800ms

# <Leader>or moves the offsite agenda under the offsite, in plans.org.
Type "/Agenda for"
Enter
Sleep 300ms
Type ":noh"
Enter
Sleep 400ms
Type " or"
Wait+Screen@10s /refile to/
Sleep 600ms
Type@80ms "Team offsite"
Sleep 800ms
Enter
Sleep 1.5s
Type ":e work/plans.org"
Enter
Sleep 3s
