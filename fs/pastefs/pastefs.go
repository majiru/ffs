package pastefs

import (
	"os"
	"path"
	"log"
	"time"
	"strconv"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)


type Pastefs struct {
	newpaste	*fsutil.File
	pastes		*fsutil.Dir
}

func (fs *Pastefs) root() ffs.Dir {
	fi, _ := fs.newpaste.Stat()
	d, _ := fs.pastes.Stat()
	return fsutil.CreateDir("/", d, fi)
}

func (fs Pastefs) Stat(file string) (os.FileInfo, error) {
	if file == "/" {
		return fs.root().Stat()
	}
	if file == "/new" {
		return fs.newpaste.Stat()
	}
	if file == "/pastes" {
		return fs.pastes.Stat()
	}

	return fs.pastes.Find(path.Base(file))
}

func (fs Pastefs) ReadDir(path string) (ffs.Dir, error) {
	if path == "/" {
		return fs.root(), nil
	} else if path == "/pastes" {
		d := *fs.pastes
		return &d, nil
	}

	return nil, os.ErrNotExist
}

func (fs Pastefs) Open(file string, mode int) (interface{}, error) {
	//User is doing a write, we don't care where, this results in a paste
	if file == "/new" {
		name := strconv.FormatInt(time.Now().Unix(), 10)
		log.Println("Doing Write to: ", name)
		f := fsutil.CreateFile([]byte("\n"), 0777, name)
		fi, _ := f.Stat()
		fs.pastes.Append(fi)
		return f, nil
	}

	f, _ := fs.pastes.Find(path.Base(file))
	return f.Sys(), nil
}

func NewPastefs() *Pastefs {
	return &Pastefs{fsutil.CreateFile([]byte("\n"), 0777, "new"), fsutil.CreateDir("pastes")}
}