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
		s := fmt.Sprintf("Test%d", 1)
		_, err = d.Search(s)
		if err != nil {
			t.Errorf("Search returned: %v", err)
		}
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