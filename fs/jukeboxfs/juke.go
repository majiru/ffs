package jukeboxfs

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dhowden/tag"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Jukefs struct {
	*sync.RWMutex
	path string
	root *fsutil.Dir
	info map[string]tag.Metadata
}

func NewJukefs(root string) (*Jukefs, error) {
	fs := &Jukefs{
		&sync.RWMutex{},
		root,
		fsutil.CreateDir("/"),
		make(map[string]tag.Metadata),
	}
	if err := fs.updateMetadata(); err != nil {
		return nil, err
	}
	fs.updateTree()
	return fs, nil
}

func (fs *Jukefs) updateMetadata() error {
	fs.Lock()
	l := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	err := filepath.Walk(fs.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Println("Could not open file:", err)
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(info.Name(), "mp3") || strings.HasSuffix(info.Name(), "flac")) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				f, openerr := os.Open(path)
				if openerr != nil {
					return
				}
				defer f.Close()
				m, parseerr := tag.ReadFrom(f)
				if parseerr != nil {
					log.Println("File:", path, "Could not be parsed")
					return
				}
				l.Lock()
				fs.info[path] = m
				l.Unlock()
			}()
		}
		return nil
	})
	wg.Wait()
	fs.Unlock()
	return err
}

func (fs *Jukefs) updateTree() {
	fs.Lock()
	albums := make(map[string]*fsutil.Dir)
	for k, v := range fs.info {
		a := v.Album()
		if d, ok := albums[a]; ok {
			d.Append(fsutil.CreateFile([]byte(k), 0644, v.Title()).Stats)
		} else {
			newd := fsutil.CreateDir(a)
			albums[a] = newd
			fs.root.Append(newd.Stats)
		}
	}
	fs.Unlock()
}

func (fs *Jukefs) Stat(fpath string) (os.FileInfo, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch fpath {
	case "/":
		return fs.root.Stat()
	default:
		return fs.root.Walk(fpath)
	}
}

func (fs *Jukefs) ReadDir(fpath string) (ffs.Dir, error) {
	fs.RLock()
	defer fs.RUnlock()
	switch fpath {
	case "/":
		return fs.root.Dup(), nil
	default:
		return fs.root.WalkForDir(fpath)
	}
}

func (fs *Jukefs) Open(fpath string, mode int) (interface{}, error) {
	fs.RLock()
	defer fs.RUnlock()
	f, err := fs.root.WalkForFile(fpath)
	if err != nil {
		return nil, err
	}
	//These files store the path, not the file contents
	f.Seek(0, io.SeekStart)
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(string(b), os.O_RDONLY, 0555)
}
