package main

import (
	"errors"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/y3owk1n/fude/sdk"
)

// formats are what export writes, by name: pandoc's writer and the file's
// extension.
var formats = map[string][2]string{
	"html":  {"html", "html"},
	"md":    {"gfm", "md"},
	"pdf":   {"pdf", "pdf"},
	"docx":  {"docx", "docx"},
	"odt":   {"odt", "odt"},
	"latex": {"latex", "tex"},
	"epub":  {"epub", "epub"},
	"txt":   {"plain", "txt"},
}

// export writes the buffer, as its text is now, in format beside its file
// with pandoc, html for none.
func export(format string) error {
	format = strings.TrimSpace(format)
	if format == "" {
		format = "html"
	}
	f, ok := formats[format]
	if !ok {
		return errors.New("no format " + format + ": " + strings.Join(slices.Sorted(maps.Keys(formats)), ", "))
	}
	here := sdk.Current().Path
	if here == "" {
		return errors.New("the buffer has no file to export beside")
	}
	to := strings.TrimSuffix(here, path.Ext(here)) + "." + f[1]
	n, _ := sdk.Len()
	text, _ := sdk.Text(0, n)
	args := []string{"-f", "org", "-t", f[0], "--standalone", "-o", to}
	if f[0] == "pdf" {
		args = slices.Delete(args, 2, 4)
	}
	// Citations need pandoc's citeproc and the bibliographies.
	if bibs := bibliographies(); len(bibs) > 0 {
		args = append(args, "--citeproc")
		for _, b := range bibs {
			args = append(args, "--bibliography", b)
		}
	}
	pandoc, _ := sdk.Option("fude-org.pandoc")
	return sdk.Pipe(text, func(_ string, err error) {
		if err != nil {
			report(err)
			return
		}
		sdk.Message("exported to " + path.Base(to))
	}, pandoc, args...)
}

// pickExport asks for a format and exports the buffer in it.
func pickExport() {
	names := slices.Sorted(maps.Keys(formats))
	report(sdk.Pick("export to", names, func(i int, ok bool) {
		if ok {
			report(export(names[i]))
		}
	}))
}
