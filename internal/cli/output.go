package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	formatJSONL = "jsonl"
	formatView  = "view"
	extJSONL    = "." + formatJSONL
	extView     = "." + formatView
)

type formats struct {
	jsonl bool
	view  bool
}

var allFormats = formats{jsonl: true, view: true}

func parseFormats(list string) (formats, error) {
	var f formats
	for _, name := range strings.Split(list, ",") {
		switch strings.TrimSpace(name) {
		case formatJSONL:
			f.jsonl = true
		case formatView:
			f.view = true
		case "":
		default:
			return formats{}, fmt.Errorf("unknown format %q in -f: use %s, %s", name, formatJSONL, formatView)
		}
	}
	if f == (formats{}) {
		return formats{}, fmt.Errorf("-f needs a list of formats")
	}
	return f, nil
}

func outputBase(path string) string {
	for _, ext := range []string{extJSONL, extView} {
		if strings.HasSuffix(path, ext) {
			return strings.TrimSuffix(path, ext)
		}
	}
	return path
}

func dumpPath(ra runArgs) (string, error) {
	if ra.formats.jsonl {
		return ra.out + extJSONL, nil
	}
	f, err := os.CreateTemp(filepath.Dir(ra.out), filepath.Base(ra.out)+".*"+extJSONL)
	if err != nil {
		return "", fmt.Errorf("create output: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("create output: %w", err)
	}
	return f.Name(), nil
}
