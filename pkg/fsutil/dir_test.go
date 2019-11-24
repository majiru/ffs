package fsutil

import (
	"fmt"
	"io"
	"os"
	"path"
	"testing"
)

const FilesPerDir = 20

func TestReaddir(t *testing.T) {
	d := CreateDir("/")
	for i := 0; i < FilesPerDir; i++ {
		d.Append(CreateFile([]byte{}, 0644, fmt.Sprintf("test%d", i)).Stats)
	}
	fi, err := d.Readdir(-1)
	if err != nil {
		t.Errorf("Readdir returned error for negative read: %v", err)
	}
	if len(fi) != FilesPerDir {
		t.Errorf("Dropped file info in negative read")
	}
	for i := 0; i < FilesPerDir; i++ {
		fi, err := d.Readdir(1)
		if err != nil {
			t.Errorf("Readdir: %v", err)
		}
		if fi[0].Name() != fmt.Sprintf("test%d", i) {
			t.Errorf("out of order read")
		}
	}
	_, err = d.Readdir(2)
	if err != io.EOF {
		t.Errorf("Expected EOF")
	}
}

func TestFind(t *testing.T) {
	d := CreateDir("/")
	for i := 0; i < FilesPerDir; i++ {
		d.Append(CreateFile([]byte{}, 0644, fmt.Sprintf("test%d", i)).Stats)
	}
	for i := 0; i < FilesPerDir; i++ {
		s := fmt.Sprintf("test%d", i)
		_, err := d.Find(s)
		if err != nil {
			t.Errorf("Find did not find file %s", s)
		}
	}
	s := fmt.Sprintf("test%d", FilesPerDir+1)
	if _, err := d.Find(s); err != os.ErrNotExist {
		t.Errorf("Found non existent file: %s", s)
	}
}

func TestSearch(t *testing.T) {
	var err error
	d := CreateDir("/", CreateFile([]byte{}, 0644, "Test1").Stats,
		CreateDir("Test2", CreateDir("Test3", CreateFile([]byte{}, 0644, "Test4").Stats).Stats).Stats,
		CreateDir("Test5", CreateFile([]byte{}, 0644, "Test6").Stats).Stats)
	for i := 1; i < 7; i++ {
		s := fmt.Sprintf("Test%d", i)
		_, err = d.Search(s)
		if err != nil {
			t.Errorf("Search returned: %v", err)
		}
	}
}

func TestSearchEmpty(t *testing.T) {
	d := CreateDir("/")
	_, err := d.Search("chris")
	if err != os.ErrNotExist {
		t.Fatalf("expected %v got %v", os.ErrNotExist, err)
	}
}

func TestWalk(t *testing.T) {
	d := CreateDir("/", CreateFile([]byte{}, 0644, "Test1").Stats,
		CreateDir("Test2", CreateDir("Test3", CreateFile([]byte{}, 0644, "Test4").Stats).Stats).Stats,
		CreateDir("Test5", CreateFile([]byte{}, 0644, "Test5").Stats).Stats)

	walkErr := func(s string) {
		fi, err := d.Walk(s)
		if err != nil {
			t.Errorf("%v when walking for %s", err, s)
		}
		if fi.Name() != path.Base(s) {
			t.Errorf("Expected %s, got %s for walk", path.Base(s), fi.Name())
		}
	}

	walkErr("/Test1")
	walkErr("/Test2/Test3")
	walkErr("/Test2/Test3/Test4")
	walkErr("/Test5/Test5")
}

func TestWalkErr(t *testing.T) {
	d := CreateDir("/")

	walkErr := func(s string, d *Dir, err error) {
		_, e := d.Walk(s)
		if e != err {
			t.Fatalf("expected %v got %v", err, e)
		}
	}

	walkErr("//", d, os.ErrNotExist)

	d.Append(CreateDir("Test1").Stats)
	walkErr("/chris", d, os.ErrNotExist)

	d = CreateDir("/", CreateFile([]byte{}, 0644, "Test1").Stats,
		CreateDir("Test2", CreateDir("Test3", CreateFile([]byte{}, 0644, "Test4").Stats).Stats).Stats,
		CreateDir("Test5", CreateFile([]byte{}, 0644, "Test5").Stats).Stats)
	walkErr("/Test5/Test5/chris", d, ErrCastDir)
	walkErr("/Test2/Test3/chris", d, os.ErrNotExist)
}

func walkForFile(d *Dir, s string) (interface{ Stat() (os.FileInfo, error) }, error) {
	return d.WalkForFile(s)
}

func walkForDir(d *Dir, s string) (interface{ Stat() (os.FileInfo, error) }, error) {
	return d.WalkForDir(s)
}

func TestWalkCast(t *testing.T) {
	d := CreateDir("/", CreateFile([]byte{}, 0644, "Test1").Stats,
		CreateDir("Test2", CreateDir("Test3", CreateFile([]byte{}, 0644, "Test4").Stats).Stats).Stats,
		CreateDir("Test5", CreateFile([]byte{}, 0644, "Test5").Stats).Stats)

	walkErr := func(s string, fun func(d *Dir, filepath string) (interface{ Stat() (os.FileInfo, error) }, error)) {
		f, err := fun(d, s)
		if err != nil {
			t.Errorf("%v when walking for %s", err, s)
		}
		fi, err := f.Stat()
		if err != nil {
			t.Fatal("error stating file:", err)
		}
		if fi.Name() != path.Base(s) {
			t.Errorf("Expected %s, got %s for walk", path.Base(s), fi.Name())
		}
	}
	walkErr("/Test1", walkForFile)
	walkErr("/Test2", walkForDir)
	walkErr("/Test2/Test3", walkForDir)
	walkErr("/Test2/Test3/Test4", walkForFile)
	walkErr("/Test5/Test5", walkForFile)
}

func TestWalkCastErr(t *testing.T) {
	d := CreateDir("/", CreateFile([]byte{}, 0644, "Test1").Stats,
		CreateDir("Test2", CreateDir("Test3", CreateFile([]byte{}, 0644, "Test4").Stats).Stats).Stats,
		CreateDir("Test5", CreateFile([]byte{}, 0644, "Test5").Stats).Stats)

	walkErr := func(s string, err error, f func(d *Dir, filepath string) (interface{ Stat() (os.FileInfo, error) }, error)) {
		_, e := f(d, s)
		if e != err {
			t.Fatalf("got %v expected %v", e, err)
		}
	}
	walkErr("/Test1", ErrCastDir, walkForDir)
	walkErr("/Test2", ErrCastFile, walkForFile)
	walkErr("/Test2/Test3", ErrCastFile, walkForFile)
	walkErr("/Test2/Test3/Test4", ErrCastDir, walkForDir)
	walkErr("/Test5/Test5", ErrCastDir, walkForDir)
	walkErr("/chris", os.ErrNotExist, walkForDir)
	walkErr("/bliss", os.ErrNotExist, walkForFile)
}

func TestCopy(t *testing.T) {
	names := []string{"chris", "bliss", "danny", "test", "test2"}
	d := CreateDir("/")
	for _, n := range names {
		d.Append(CreateDir(n).Stats)
	}
	fi := d.Copy()
	for i := range names {
		if fi[i].Name() != names[i] {
			t.Fatalf("expected %s got %s", names[i], fi[i].Name())
		}
	}
}