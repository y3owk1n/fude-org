package main

import (
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/y3owk1n/fude/api"
	"github.com/y3owk1n/fude/sdk"
)

// attachDir returns the folder of the attachments of the heading whose ID
// is id, data/ and the ID's first two characters and the rest beside the
// buffer's file, as org keeps them.
func attachDir(id string) string {
	return path.Join(path.Dir(sdk.Current().Path), "data", id[:min(2, len(id))], id[min(2, len(id)):])
}

// attachments asks what to do with the attachments of the section that
// holds cursor: attach a file, or open one.
func attachments(cursor int) {
	report(sdk.Pick("attachments", []string{"attach a file", "open an attachment"}, func(i int, ok bool) {
		switch {
		case !ok:
		case i == 0:
			ask("attach file", "", func(file string) error { return attach(cursor, strings.TrimSpace(file)) })
		default:
			report(pickAttachment(cursor))
		}
	}))
}

// attach copies file to the attachments of the section's heading, giving
// it an ID when it has none and the ATTACH tag.
func attach(cursor int, file string) error {
	h, ok := owner(cursor)
	if !ok {
		return errors.New("no heading here")
	}
	data, err := sdk.ReadFile(file)
	if err != nil {
		return errors.New("read " + file + ": " + err.Error())
	}
	f, end := front(h)
	id := f.prop("ID")
	if id == "" {
		id = newID()
		f.setProp("ID", id)
	}
	dir := attachDir(id)
	if err := sdk.MakeDir(dir); err != nil {
		return err
	}
	if err := sdk.WriteFile(path.Join(dir, path.Base(file)), data); err != nil {
		return err
	}
	if !slices.Contains(f.Head.Tags, "ATTACH") {
		f.Head.Tags = append(f.Head.Tags, "ATTACH")
	}
	putFront(h, f, end)
	return sdk.Message("attached " + path.Base(file))
}

// pickAttachment lists the attachments of the section's heading and opens
// the one picked.
func pickAttachment(cursor int) error {
	dir, err := attachedDir(cursor)
	if err != nil {
		return err
	}
	entries, err := sdk.List(dir)
	if err != nil {
		return errors.New("the heading has no attachments")
	}
	var names []string
	for _, e := range entries {
		if !e.Dir {
			names = append(names, e.Name)
		}
	}
	return sdk.Pick("open attachment", names, func(i int, ok bool) {
		if ok {
			report(openFile(path.Join(dir, names[i])))
		}
	})
}

// attachedDir returns the attachment folder of the heading of the section
// that holds cursor, which needs an ID.
func attachedDir(cursor int) (string, error) {
	h, ok := owner(cursor)
	if !ok {
		return "", errors.New("no heading here")
	}
	f, _ := front(h)
	if f.prop("ID") == "" {
		return "", errors.New("the heading has no ID, so no attachments")
	}
	return attachDir(f.prop("ID")), nil
}

// openFile opens an org or text file in fude, and any other with the
// opener.
func openFile(p string) error {
	switch path.Ext(p) {
	case ".org", ".txt", ".md", "":
		return sdk.Do(api.Result{Jump: &api.Jump{Path: p}})
	}
	opener, _ := sdk.Option("fude-org.opener")
	return sdk.Start(func(_ string, err error) { report(err) }, opener, p)
}

// openAttachment opens the attachment name of the section that holds
// cursor, as an attachment: link names it.
func openAttachment(cursor int, name string) error {
	dir, err := attachedDir(cursor)
	if err != nil {
		return err
	}
	return openFile(path.Join(dir, name))
}
