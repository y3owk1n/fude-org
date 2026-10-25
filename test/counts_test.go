package test

import (
	"testing"

	"github.com/y3owk1n/fude/plugin"
)

func TestCounts(t *testing.T) {
	run(t, []step{
		{"<C-Space>", "* Groceries [/]\n- [ ] milk\n- [X] bread", 16, "* Groceries [2/2]\n- [X] milk\n- [X] bread"},
		{"<C-Space>", "* G [%]\n- [ ] a\n- [ ] b\n- c", 8, "* G [50%]\n- [X] a\n- [ ] b\n- c"},
		{"<C-Space>", "* G\n- shop [/]\n  - [ ] a\n  - [X] b\n- [ ] c", 17, "* G\n- shop [2/2]\n  - [X] a\n  - [X] b\n- [ ] c"},
		{"cit", "* P [/]\n** TODO a\n** TODO b\n*** TODO deep", 8, "* P [1/2]\n** DONE a\n** TODO b\n*** TODO deep"},
		{"<Leader>o#", "* P [%]\n** DONE a\n** TODO b\n** c", 0, "* P [50%]\n** DONE a\n** TODO b\n** c"},
	})
	// The cursor stays on the item it ticked, though the count before it
	// grew.
	p := load(t, plugin.Grants{})
	res := p.Key("<C-Space>", "* G [/]\n- [ ] a", 10, 0)
	if res.Jump == nil || res.Jump.Pos != 12 {
		t.Errorf("<C-Space> left the cursor at %+v, want 12", res.Jump)
	}
}
