package record

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseMultiline(t *testing.T) {
	r, err := Parse("1\n/tmp\nfor i in 1 2; do\n  false\ndone\n")
	if err != nil {
		t.Fatal(err)
	}
	if r.Code != 1 || r.Cwd != "/tmp" || r.Cmd != "for i in 1 2; do\n  false\ndone" {
		t.Fatalf("%+v", r)
	}
}

func TestLoadFallsBackToNewest(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "last-111"), []byte("2\n/a\nfoo\n"), 0o644)
	r, err := Load(dir, 999)
	if err != nil || r.Cmd != "foo" {
		t.Fatalf("%+v %v", r, err)
	}
	os.WriteFile(filepath.Join(dir, "last-999"), []byte("3\n/b\nbar\n"), 0o644)
	if r, _ := Load(dir, 999); r.Cmd != "bar" {
		t.Fatalf("親シェルの記録が優先されない: %+v", r)
	}
}
