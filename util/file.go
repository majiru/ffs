package fsutil

import (
	"os"
	"io"
	"time"
	"github.com/majiru/ffs"
	"bytes"
)


type filedata interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type File struct{
	filedata
	Perm os.FileMode
	Time time.Time
}

func (f *File) Close() error { return nil }

func (f *File) Stat(name string) (os.FileInfo){
	return &Stat{f, name}
}

func createFile(content []byte, mode os.FileMode) ffs.File{
	return &File{bytes.NewReader(content), mode, time.Now()}
}

type Stat struct {
	File *File
	name string
}

func (s Stat) Name() string     { return s.name }
func (s Stat) Sys() interface{} { return s.File }

func (s Stat) ModTime() time.Time {
	return s.File.Time
}

func (s Stat) Mode() os.FileMode {
	return s.File.Perm
}

func (s Stat) IsDir() bool {
	return s.File.Perm.IsDir()
}

func (s Stat) Size() (int64) {
	size, err := s.File.Seek(0, io.SeekEnd)
	if err != nil {
		return 0
	}

	_, err = s.File.Seek(0, io.SeekStart)
	if err != nil {
		return 0
	}
	return size
}

type Dir struct {
	c chan os.FileInfo
}

func (d *Dir) Readdir(n int) ([]os.FileInfo, error) {
	var err error
	fi := make([]os.FileInfo, 0, 256)
	for i := 0; i < n; i++ {
		s, ok := <-d.c
		if !ok {
			err = io.EOF
			break
		}
		fi = append(fi, s)
	}
	return fi, err
}

func createDir(files ...os.FileInfo) *Dir {
	c := make(chan os.FileInfo, 256)
	go func() {
		for _, i := range files {
			c <- i
		}
		close(c)
	}()
	return &Dir{c}
}