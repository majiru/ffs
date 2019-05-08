package fsutil

import (
	"errors"
	"io"
	"log"
	"os"
	"time"
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

func (s Stat) Size() int64 {
	if f, ok := s.file.(interface{ Size() int64 }); ok {
		return f.Size()
	} else {
		log.Println("fsutil.Stat.Size: Warning: Falling back to initial(probably wrong) size")
	}
	return s.size
}

type File struct {
	s     []byte
	i     int64
	Stats *Stat
}

func (f File) Size() int64 { return int64(len(f.s)) }

func (f *File) Grow(n int64) {
	if int64(cap(f.s)) >= n {
		return
	}
	new := make([]byte, n)
	copy(new, f.s)
	f.s = new
	return
}

func (f *File) Write(b []byte) (n int, err error) {
	f.Grow(int64(len(b)) + f.i)
	n = copy(f.s[f.i:], b)
	if n < len(b) {
		return 0, errors.New("fsutil.File.Write: Bad Copy")
	}
	f.i += int64(n)
	return
}

func (f *File) WriteAt(b []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, errors.New("fsutil.File.WriteAt: negative offset")
	}
	f.Grow(int64(len(b)) + off)
	n = copy(f.s[off:], b)
	if n < len(b) {
		return 0, errors.New("fsutil.File.WriteAt: Bad Copy")
	}
	return
}

func (f *File) Read(b []byte) (n int, err error) {
	if f.i >= int64(len(f.s)) {
		return 0, io.EOF
	}
	n = copy(b, f.s[f.i:])
	f.i += int64(n)
	return
}

func (f *File) ReadAt(b []byte, off int64) (n int, err error) {
	// cannot modify state - see io.ReaderAt
	if off < 0 {
		return 0, errors.New("fsutil.File.ReadAt: negative offset")
	}
	if off >= int64(len(f.s)) {
		return 0, io.EOF
	}
	n = copy(b, f.s[off:])
	if n < len(b) {
		err = io.EOF
	}
	return
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = f.i + offset
	case io.SeekEnd:
		abs = int64(len(f.s)) + offset
	default:
		return 0, errors.New("fsutil.File.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("fsutil.File.Seek: negative position")
	}
	f.i = abs
	return abs, nil
}

func (f *File) Close() error {
	f.i = 0
	return nil
}

func (f File) Stat() (os.FileInfo, error) {
	return f.Stats, nil
}

func (f *File) Truncate(size int64) error {
	if size > int64(cap(f.s)) {
		f.Grow(size)
		return nil
	}
	new := make([]byte, size)
	n := copy(new, f.s[:size])
	if int64(n) != size {
		return errors.New("fsutil.File.Truncate: Bad Copy")
	}
	f.s = new
	return nil
}

func CreateFile(content []byte, mode os.FileMode, name string) *File {
	f := File{content, int64(len(content)), nil}
	f.Stats = &Stat{mode, name, time.Now(), int64(len(content)), &f}
	return &f
}

type Dir struct {
	files []os.FileInfo
	i     int
	Stats *Stat
}

func CreateDir(name string, files ...os.FileInfo) *Dir {
	d := Dir{files, 0, nil}
	d.Stats = &Stat{os.ModeDir | 0777, name, time.Now(), 0, nil}
	return &d
}

func (d *Dir) Readdir(n int) ([]os.FileInfo, error) {
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

func (d Dir) Stat() (os.FileInfo, error) {
	return d.Stats, nil
}

func (d *Dir) Append(files ...os.FileInfo) {
	d.files = append(d.files, files...)
}
