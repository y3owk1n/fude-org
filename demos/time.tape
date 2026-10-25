# Time in fude-org: clocking in shows the task and its time on the status
# line, a clock table sums the time clocked under each heading, and a task
# that repeats moves its date on when it is done.
Output demos/time.gif
Output demos/out/time.mp4
Source demos/style.tape

Hide
Type "source demos/setup.sh time"
Enter
Wait
Show
Sleep 500ms

Type "fude work.org"
Enter
Sleep 2s

# <Leader>oxi clocks in the intro, into its LOGBOOK, and the status line
# shows it.
Type "6G"
Sleep 400ms
Type " oxi"
Sleep 2.5s

# <Leader>oxr puts a clock table of the file under Report.
Type "G"
Sleep 400ms
Type " oxr"
Sleep 3s

# The plants need water every three days. cit marks them done, which
# moves their date on and keeps them TODO.
Type "/Water"
Enter
Sleep 300ms
Type ":noh"
Enter
Sleep 400ms
Type "cit"
Sleep 3s
