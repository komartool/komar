package compact

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/txtar"
)

var update = flag.Bool("update", false, "rewrite want.view in testdata from the current output")

const (
	fileProbes = "probes"
	fileLayout = "layout"
	fileDump   = "dump.jsonl"
	fileWant   = "want.view"
)

func TestRender(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "*.txtar"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("no golden files in testdata: %v", err)
	}
	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".txtar"), func(t *testing.T) {
			golden(t, path)
		})
	}
}

func golden(t *testing.T, path string) {
	t.Helper()
	ar, err := txtar.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	for _, f := range ar.Files {
		files[f.Name] = f.Data
	}
	lay, err := parseLayout(files[fileLayout])
	if err != nil {
		t.Fatal(err)
	}
	got := render(files[fileDump], strings.Fields(string(files[fileProbes])), lay)
	if *update {
		writeWant(t, path, ar, got)
		return
	}
	if want := string(files[fileWant]); got != want {
		t.Errorf("view differs from %s (go test -update rewrites it)\n--- got\n%s\n--- want\n%s", path, got, want)
	}
}

func parseLayout(data []byte) (layout, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return defaultLayout, nil
	}
	lay := defaultLayout
	if _, err := fmt.Sscanf(string(data), "%d %d", &lay.contentsLines, &lay.contentsEntries); err != nil {
		return layout{}, fmt.Errorf("layout must be \"<contents lines> <contents entries>\": %w", err)
	}
	return lay, nil
}

func writeWant(t *testing.T, path string, ar *txtar.Archive, got string) {
	t.Helper()
	replaced := false
	for i := range ar.Files {
		if ar.Files[i].Name == fileWant {
			ar.Files[i].Data = []byte(got)
			replaced = true
		}
	}
	if !replaced {
		ar.Files = append(ar.Files, txtar.File{Name: fileWant, Data: []byte(got)})
	}
	if err := os.WriteFile(path, txtar.Format(ar), 0o600); err != nil {
		t.Fatal(err)
	}
}
