package fsutil

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"
)

//Dir represents an in memory Directory.
type Dir struct {
	files []os.FileInfo
	i     int
	Stats *Stat
}

//CreateDir creates a new Dir struct.
//The underlying Stats.Sys() points to the new Dir.
func CreateDir(name string, files ...os.FileInfo) *Dir {
	d := Dir{files, 0, nil}
	d.Stats = &Stat{os.ModeDir | 0777, name, time.Now(), 0, &d}
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

//Find performs a 1 level deep search to find a file specified by name
func (d *Dir) Find(name string) (os.FileInfo, error) {
	for _, dir := range d.files {
		if dir.Name() == name {
			return dir, nil
		}
	}
	return nil, os.ErrNotExist
}

func search(target string, files []os.FileInfo) (os.FileInfo, error) {
	if len(files) == 0 {
		return nil, os.ErrNotExist
	}
	fi := files[0]
	if fi.Name() == target {
		return fi, nil
	}
	if fi.IsDir() {
		if dir, ok := fi.Sys().(*Dir); ok {
			if match, err := search(target, dir.files); err == nil {
				return match, nil
			}
		}
	}
	return search(target, files[1:])
}

//Search performs a recursive search into all subdirs looking for name.
//recusrive descent is only possible if subdir is of type *Dir
func (d *Dir) Search(name string) (os.FileInfo, error) {
	return search(name, d.files)
}

func split(fpath string) (clean []string) {
	//Using strings.Split has some drawbacks,
	//but for the case in which we are using this for a service
	//everthing is / delimited anyway
	for _, parts := range strings.Split(fpath, "/") {
		if parts != "" {
			clean = append(clean, parts)
		}
	}
	return
}

func (d *Dir) Walk(fpath string) (fi os.FileInfo, err error) {
	parts := split(fpath)
	if len(parts) == 0 {
		return nil, os.ErrNotExist
	}
	if fi, err = d.Find(parts[0]); err != nil {
		return
	}
	for _, part := range parts[1:] {
		if subdir, ok := fi.Sys().(*Dir); !ok {
			return nil, errors.New("fsutil.Dir.Walk: cast to Dir failed")
		} else {
			if fi, err = subdir.Find(part); err != nil {
				return nil, os.ErrNotExist
			}
		}
	}
	return
}

func (d *Dir) WalkForFile(fpath string) (*File, error) {
	if fi, err := d.Walk(fpath); err != nil {
		return nil, err
	} else {
		if f, ok := fi.Sys().(*File); ok {
			return f.Dup(), nil
		} else {
			return nil, errors.New("fsutil.Dir.WalkForFile: cast to File Failed")
		}
	}
}

func (d *Dir) WalkForDir(fpath string) (*Dir, error) {
	if fi, err := d.Walk(fpath); err != nil {
		return nil, err
	} else {
		if d, ok := fi.Sys().(*Dir); ok {
			return d.Dup(), nil
		} else {
			return nil, errors.New("fsutil.Dir.WalkForFile: cast to Dir Failed")
		}
	}
}

//Copy duplicates the held file info slice to the caller.
func (d *Dir) Copy() (out []os.FileInfo) {
	out = make([]os.FileInfo, len(d.files))
	copy(out, d.files)
	return
}

//Dup creates a new Dir, duplicatin all contained data.
func (d *Dir) Dup() *Dir {
	new := *d
	return &new
}
