# Source blocks in fude-org: a block runs and its output goes under
# #+RESULTS:, a block opens in a buffer of its language and writes back,
# and tangling writes blocks out to their files.
Output demos/code.gif
Output demos/out/code.mp4
Source demos/style.tape

Hide
Type "source demos/setup.sh code"
Enter
Wait
Show
Sleep 500ms

Type "fude scripts.org"
Enter
Sleep 2s

# <Leader>obe runs the block with sh and puts what it prints under it.
Type "3G"
Sleep 400ms
Type " obe"
Sleep 2s

# <Leader>o' opens the greeting in a buffer of its own, as a shell script,
# and :w writes it back into the org file.
Type "/hello"
Enter
Sleep 300ms
Type ":noh"
Enter
Sleep 400ms
Type " o'"
Sleep 1.5s
Type@80ms `oecho "and from fude"`
Escape
Sleep 600ms
Type ":w"
Enter
Sleep 800ms
Type ":b 1"
Enter
Sleep 1.2s

# <Leader>obe runs the edited block too.
Type " obe"
Sleep 2s

# <Leader>obt writes the block marked :tangle to greet.sh, and the first
# block lists it.
Type " obt"
Sleep 1.5s
Type "3G"
Sleep 400ms
Type " obe"
Sleep 3s
