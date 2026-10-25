# Tables in fude-org: <Tab> aligns a table as you type and adds rows, the
# #+TBLFM: line works out totals again after a change, and column view sums
# the efforts of a subtree.
Output demos/tables.gif
Output demos/out/tables.mp4
Source demos/style.tape

Hide
Type "source demos/setup.sh tables"
Enter
Wait
Show
Sleep 500ms

Type "fude trip.org"
Enter
Sleep 2s

# <Tab> aligns the table and goes to the next cell, and past the last cell
# it adds a row.
Type "5GA"
Sleep 400ms
Type@90ms "Train"
Tab
Type@90ms "2"
Tab
Type@90ms "45"
Tab 2
Type@90ms "Hotel"
Tab
Type@90ms "3"
Tab
Type@90ms "120"
Tab 2
Type@90ms "Food"
Tab
Type@90ms "3"
Tab
Type@90ms "40"
Tab 2
Type@90ms "Total"
Escape
Sleep 1s

# <Leader>o= works out each row's total and the sum.
Type " o="
Sleep 2s

# One more night at the hotel, and <Leader>o= works out the totals again.
Type "/Hotel"
Enter
Sleep 300ms
Type ":noh"
Enter
Sleep 400ms
Type "f3r4"
Sleep 600ms
Type " o="
Sleep 2s

# <Leader>oC shows the packing list as columns, its effort the sum of its
# tasks'.
Type "/Packing"
Enter
Sleep 300ms
Type ":noh"
Enter
Sleep 400ms
Type " oC"
Sleep 3.5s
