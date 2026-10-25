package test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/plugin"
)

func TestCite(t *testing.T) {
	dir := t.TempDir()
	bib := "@book{knuth84,\n  author = {Donald Knuth and Someone},\n  title = {Literate Programming},\n  year = {1984},\n}\n\n@article{dijkstra68,\n  author = \"Edsger Dijkstra\",\n  year = 1968,\n  title = {Go To Statement Considered Harmful},\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "refs.bib"), []byte(bib), 0o644); err != nil {
		t.Fatal(err)
	}
	p := load(t, plugin.Grants{Paths: []string{dir}, Programs: []string{"pandoc"}})
	p.Path = filepath.Join(dir, "a.org")
	text := "#+bibliography: refs.bib\nAs shown "
	res := p.Key("<Leader>o@", text, len(text), 0)
	if res.Pick == nil || !slices.Equal(res.Pick.Items, []string{"knuth84  Donald Knuth (1984) Literate Programming", "dijkstra68  Edsger Dijkstra (1968) Go To Statement Considered Harmful"}) {
		t.Fatalf("<Leader>o@ offered %+v", res.Pick)
	}
	got := applied(text, p.Callback(res.Pick.ID, text, api.Context{"picked": "1", "ok": "true"}))
	if got != text+"[cite:@dijkstra68]" {
		t.Fatalf("citing left %q", got)
	}
	// In a citation it adds to it.
	res = p.Key("<Leader>o@", got, len(got)-3, 0)
	if got := applied(got, p.Callback(res.Pick.ID, got, api.Context{"picked": "0", "ok": "true"})); got != text+"[cite:@dijkstra68;@knuth84]" {
		t.Errorf("citing in a citation left %q", got)
	}
	// <CR> on a key opens its entry.
	if res := p.Key("<CR>", got, len(text)+10, 0); res.Jump == nil || res.Jump.Path != filepath.Join(dir, "refs.bib") || res.Jump.Line != 7 {
		t.Errorf("<CR> on the key went to %+v", res.Jump)
	}
	// Export cites with pandoc's citeproc.
	res = p.Command("OrgExport", "", got, 0)
	if argv := res.Jobs[0].Argv; !slices.Contains(argv, "--citeproc") || argv[len(argv)-1] != filepath.Join(dir, "refs.bib") {
		t.Errorf("export ran %q", argv)
	}
	// fude joins a list in config.toml with commas.
	other := filepath.Join(dir, "other.bib")
	if err := os.WriteFile(other, []byte("@book{lamport94,\n  author = {Leslie Lamport},\n  title = {LaTeX},\n  year = {1994},\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p = load(t, plugin.Grants{Paths: []string{dir}, Programs: []string{"pandoc"}, Options: map[string]string{"bibliography": filepath.Join(dir, "refs.bib") + "," + other}})
	p.Path = filepath.Join(dir, "a.org")
	if res := p.Key("<Leader>o@", "As shown ", 9, 0); res.Pick == nil || len(res.Pick.Items) != 3 {
		t.Errorf("<Leader>o@ with a list of bibliographies offered %+v", res.Pick)
	}
}
