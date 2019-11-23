package ramfs

import (
	"os"
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

func TestStat(t *testing.T) {
	ramfs := Ramfs{Root: fsutil.CreateDir("/")}
	_, err := ramfs.Open("afile", 0644)
	if err != nil {
		t.Fatal("Error opening file:", err)
	}
	fi, err := ramfs.Stat("afile")
	if err != nil {
		t.Fatal("Error stating file:", err)
	}
	if fi.Name() != "afile" || fi.Mode() != 0644 {
		t.Fatal("content mismatch for existing file")
	}
	fi, err = ramfs.Stat("doesnotexist")
	if err != os.ErrNotExist || fi != nil {
		t.Fatal("error opening non existing file:", err)
	}
}

func TestFindOrCreate(t *testing.T) {
	ramfs := Ramfs{Root: fsutil.CreateDir("/")}
	_, _, err := ramfs.FindOrCreate("adir/adir2/adir3", true)
	if err != nil {
		t.Fatal("error when creating dir:", err)
	}
	_, _, err = ramfs.FindOrCreate("adir", true)
	if err != nil {
		t.Fatal("error when creating dir:", err)
	}
	_, d, err := ramfs.FindOrCreate("//adir", true)
	if err != nil {
		t.Fatal("error when creating dir:", err)
	}
	fi, err := d.Stat()
	if err != nil {
		t.Fatal("dir stats returned err:", err)
	}
	if fi.Name() != "adir" {
		t.Fatal("content mismatch")
	}
}

func TestFindOrCreateErr(t *testing.T) {
	ramfs := Ramfs{Root: fsutil.CreateDir("/")}
	_, _, err := ramfs.FindOrCreate("adir", true)
	if err != nil {
		t.Fatal("error when creating dir:", err)
	}
	_, _, err = ramfs.FindOrCreate("adir", false)
	if err != DirExists {
		t.Fatalf("expected %v got %v for file alread existing as dir", DirExists, err)
	}
	_, _, err = ramfs.FindOrCreate("/adir/adir2", true)
	if err != nil {
		t.Fatal("error when creating dir:", err)
	}
	_, _, err = ramfs.FindOrCreate("/adir/adir2/adir3/adir4", true)
	if err != nil {
		t.Fatal("error when creating dir:", err)
	}
	_, _, err = ramfs.FindOrCreate("/adir/adir2", false)
	if err != DirExists {
		t.Fatalf("expected %v got %v for file alread existing as dir", DirExists, err)
	}
	_, _, err = ramfs.FindOrCreate("afile", false)
	if err != nil {
		t.Fatal("error when creating file:", err)
	}
	_, _, err = ramfs.FindOrCreate("afile", true)
	if err != FileExists {
		t.Fatalf("expected %v got %v for dir alread existing as file", FileExists, err)
	}
	_, _, err = ramfs.FindOrCreate("/afile/adir2", false)
	if err != DirExists {
		t.Fatalf("expected %v got %v for file alread existing as dir", DirExists, err)
	}
}