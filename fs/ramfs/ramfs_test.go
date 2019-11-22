package ramfs

import (
	"testing"

	"github.com/majiru/ffs/pkg/fsutil"
)

func TestOpen(t *testing.T) {
	ramfs := Ramfs{Root: fsutil.CreateDir("/")}
	_, err := ramfs.Open("/afile", 0644)
	if err != nil {
		t.Fatal("Error opening file:", err)
	}
	fi := ramfs.Root.Copy()
	if len(fi) != 1 || fi[0].Name() != "afile" || fi[0].IsDir() {
		t.Fatal("File not in root folder")
	}
}

func TestReadDir(t *testing.T) {
	ramfs := Ramfs{Root: fsutil.CreateDir("/")}
	_, err := ramfs.ReadDir("adir")
	if err != nil {
		t.Fatal("Error opening dir:", err)
	}
	fi := ramfs.Root.Copy()
	if len(fi) != 1 || fi[0].Name() != "adir" || !fi[0].IsDir() {
		t.Fatal("File not in root folder")
	}
}

const m1 = "Hello World"

func TestOpenExisting(t *testing.T) {
	ramfs := Ramfs{Root: fsutil.CreateDir("/")}
	f, err := ramfs.Open("afile", 0644)
	if err != nil {
		t.Fatal("Error opening file:", err)
	}
	w, ok := f.(*fsutil.File)
	if !ok {
		t.Fatal("Failed cast")
	}
	w.Write([]byte(m1))
	f.Close()
	f, err = ramfs.Open("afile", 0644)
	if err != nil {
		t.Fatal("Error opening file:", err)
	}
	b := make([]byte, len(m1))
	f.Read(b)
	if string(b) != m1 {
		t.Fatal("Could not read what was written")
	}
}