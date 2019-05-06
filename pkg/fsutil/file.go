package fsutil

import (
	"bytes"
	"io"
	"os"
	"time"
	"log"
)

type Stat struct {
	perm os.FileMode
	name string
	time time.Time
	size int64
	file interface{}
}

func (s Stat) Name() string     { return s.name }
func (s Stat) Sys() interface{} { return s.file }

func (s Stat) ModTime() time.Time { return s.time }

func (s Stat) Mode() os.FileMode { return s.perm }

func (s Stat) IsDir() bool { return s.perm.IsDir() }

func (s Stat) Size() int64 { return s.size }

type filedata interface {
	io.Reader
	io.ReaderAt
	io.Seeker
}

type File struct {
	filedata
	Stats *Stat
}

func (f File) Close() error { return nil }

func (f File) Stat() (os.FileInfo, error) {
	return f.Stats, nil
}

func CreateFile(content []byte, mode os.FileMode, name string) *File {
	f := File{bytes.NewReader(content), nil}
	f.Stats = &Stat{mode, name, time.Now(), 0, f}
	return &f
}

type Dir struct {
	files []os.FileInfo
	i     int
	Stats *Stat
}

func CreateDir(name string, files ...os.FileInfo) *Dir {
	d := Dir{files, 0, nil}
	d.Stats = &Stat{os.ModeDir, name, time.Now(), 0, nil}
	return &d
}

func (d *Dir) Readdir(n int) ([]os.FileInfo, error) {
	log.Println(n, " ", d.i)
	if n <= 0 {
		return d.files, nil
	}
	if d.i >= len(d.files) {
		return nil, io.EOF
	}
	start := d.i
	if len(d.files) > d.i+n {
		d.i += n
	} else {
		d.i = len(d.files)
	}
	return d.files[start:d.i], nil
}

func (d *Dir) Reset() error {
	d.i = 0
	return nil
}

func (d Dir) Stat() (os.FileInfo, error) {
	return d.Stats, nil
}

func(d *Dir) Close() error { return nil }