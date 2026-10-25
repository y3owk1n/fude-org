package test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
)

func TestRunBlock(t *testing.T) {
	p := load(t, plugin.Grants{Programs: []string{"sh", "python3"}})
	for _, c := range []struct{ text, output, want string }{
		{"#+BEGIN_SRC sh\necho hi\n#+END_SRC\nafter", "hi\nthere\n", "#+BEGIN_SRC sh\necho hi\n#+END_SRC\n#+RESULTS:\n: hi\n: there\nafter"},
		{"#+BEGIN_SRC sh\necho hi\n#+END_SRC\n#+RESULTS:\n: old\n: older\nafter", "new\n", "#+BEGIN_SRC sh\necho hi\n#+END_SRC\n#+RESULTS:\n: new\nafter"},
		{"#+begin_src python :results output\nprint(1)\n#+end_src", "1\n", "#+begin_src python :results output\nprint(1)\n#+end_src\n#+RESULTS:\n: 1"},
	} {
		res := p.Key("<Leader>obe", c.text, 16, 0)
		if len(res.Jobs) != 1 {
			t.Fatalf("<Leader>obe in %q ran %+v, %s", c.text, res.Jobs, res.Error)
		}
		res = p.Callback(res.Jobs[0].ID, c.text, api.Context{"output": c.output})
		if got := applied(c.text, res); got != c.want {
			t.Errorf("running the block in %q left %q, want %q", c.text, got, c.want)
		}
	}
	if res := p.Key("<Leader>obe", "#+BEGIN_SRC sh\necho hi\n#+END_SRC", 16, 0); !slices.Equal(res.Jobs[0].Argv, []string{"sh"}) || res.Jobs[0].Input != "echo hi\n" {
		t.Errorf("<Leader>obe ran %+v", res.Jobs[0])
	}
	for _, text := range []string{"#+BEGIN_SRC cobol\nx\n#+END_SRC", "no block"} {
		if res := p.Key("<Leader>obe", text, 2, 0); res.Error == "" {
			t.Errorf("<Leader>obe in %q said nothing", text)
		}
	}
}

func TestTangleAndEdit(t *testing.T) {
	dir := t.TempDir()
	p := load(t, plugin.Grants{Paths: []string{dir}})
	p.Path = filepath.Join(dir, "a.org")
	text := "#+BEGIN_SRC sh :tangle out.sh\necho a\n#+END_SRC\nmid\n#+BEGIN_SRC sh :tangle no\necho skip\n#+END_SRC\n#+BEGIN_SRC sh :tangle out.sh\necho b\n#+END_SRC"
	if res := p.Key("<Leader>obt", text, 0, 0); res.Error != "" {
		t.Fatal(res.Error)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "out.sh")); err != nil || string(b) != "echo a\necho b\n" {
		t.Errorf("out.sh holds %q, %v", b, err)
	}
	res := p.Key("<Leader>o'", text, 33, 0)
	if s := res.Scratch; s == nil || s.Text != "echo a" || s.Filetype != "bash" || s.Kind != "org-src" {
		t.Errorf("<Leader>o' opened %+v", res.Scratch)
	}
}
