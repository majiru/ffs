package mkvfs

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/remko/go-mkvparse"
)

type MKVfs struct {
	*sync.RWMutex
	root *fsutil.Dir
	rawpath *fsutil.File
	p *TreeParser
	lastupdate time.Time
}

func NewMKVfs() *MKVfs {
	m := MKVfs{
		&sync.RWMutex{},
		nil,
		fsutil.CreateFile([]byte(""), 0644, "mkv"),
		nil,
		time.Time{},
	}
	m.lastupdate = m.rawpath.Stats.ModTime()
	m.root = fsutil.CreateDir("/", m.rawpath.Stats)
	m.p = NewTreeParser(m.root)
	return &m
}

func (fs *MKVfs) decode() (err error) {
	var b []byte
	fs.rawpath.Seek(0, io.SeekStart)
	b, err = ioutil.ReadAll(fs.rawpath)
	if err != nil {
		return
	}
	err = mkvparse.ParsePath(strings.TrimSuffix(string(b), "\n"), fs.p)
	fs.lastupdate = fs.rawpath.Stats.ModTime()
	return
}

func (fs *MKVfs) check() (err error){
	fs.Lock()
	if fs.rawpath.Stats.ModTime().After(fs.lastupdate) {
		log.Println("Doing update")
		fs.Unlock()
		return fs.decode()
	}
	fs.Unlock()
	return
}

func (fs *MKVfs) Stat(fpath string) (os.FileInfo, error) {
	if err := fs.check(); err != nil {
		return nil, err
	}
	switch fpath {
	case "/":
		return fs.root.Stat()
	default:
		return fs.root.Walk(fpath)
	}
}

func (fs *MKVfs) ReadDir(fpath string) (ffs.Dir, error) {
	if err := fs.check(); err != nil {
		return nil, err
	}
	switch fpath {
	case "/":
		return fs.root.Dup(), nil
	default:
		return fs.root.WalkForDir(fpath)
	}
}

func (fs *MKVfs) Open(fpath string, mode int) (interface{}, error) {
	if err := fs.check(); err != nil {
		return nil, err
	}
	return fs.root.WalkForFile(fpath)
}