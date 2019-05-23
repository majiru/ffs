package domainfs

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/majiru/ffs/pkg/server"
)

type Domainfs struct {
	*sync.RWMutex
	sub, dns []string
	domains  map[string]ffs.Fs
}

func NewDomainfs() *Domainfs {
	return &Domainfs{
		&sync.RWMutex{},
		[]string{},
		[]string{},
		make(map[string]ffs.Fs),
	}
}

func (fs *Domainfs) Add(newfs ffs.Fs, names ...string) {
	fs.Lock()
	for _, n := range names {
		fs.domains[n] = newfs
	}
	fs.Unlock()
}

func (fs *Domainfs) AddDNS(newfs ffs.Fs, names ...string) {
	fs.Lock()
	fs.dns = append(fs.dns, names...)
	for _, n := range names {
		fs.domains[n] = newfs
		for _, s := range fs.sub {
			fs.domains[s+"."+n] = newfs
		}
	}
	fs.Unlock()
}

func (fs *Domainfs) AddSub(newfs ffs.Fs, names ...string) {
	fs.Lock()
	fs.sub = append(fs.sub, names...)
	for _, n := range names {
		for _, d := range fs.dns {
			fs.domains[n+"."+d] = newfs
		}
	}
	fs.Unlock()
}

func (fs *Domainfs) map2dir() ffs.Dir {
	fs.RLock()
	root := fsutil.CreateDir("/")

	for k, _ := range fs.domains {
		fi, _ := fsutil.CreateDir(k).Stat()
		root.Append(fi)
	}

	fs.RUnlock()
	return root
}

func (fs *Domainfs) path2fs(path string) (ffs.Fs, string, error) {
	fs.RLock()
	defer fs.RUnlock()
	paths := strings.Split(path, "/")
	if len(paths) < 2 {
		return nil, "", os.ErrNotExist
	}

	child := fs.domains[paths[1]]
	if child == nil {
		return nil, "", os.ErrNotExist
	}

	file := "/" + strings.Join(paths[2:], "/")
	return child, file, nil
}

func (fs *Domainfs) Stat(path string) (os.FileInfo, error) {
	if path == "/" {
		return fs.map2dir().Stat()
	}
	child, file, err := fs.path2fs(path)
	if err != nil {
		return nil, err
	}
	return child.Stat(file)
}

func (fs *Domainfs) ReadDir(path string) (ffs.Dir, error) {
	if path == "/" {
		return fs.map2dir(), nil
	}
	child, file, err := fs.path2fs(path)
	if err != nil {
		return nil, err
	}
	return child.ReadDir(file)
}

func (fs *Domainfs) Open(path string, mode int) (interface{}, error) {
	child, file, err := fs.path2fs(path)
	if err != nil {
		return nil, err
	}
	return child.Open(file, mode)
}

func (fs *Domainfs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	strip := regexp.MustCompile(`:[0-9]+`)
	name := strip.ReplaceAllString(r.Host, "")
	child, _, err := fs.path2fs("/" + name)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	server.Server{child}.ServeHTTP(w, r)
	return
}
