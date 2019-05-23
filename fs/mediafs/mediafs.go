package mediafs

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"sync"

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Mediafs struct {
	*sync.RWMutex
	db                            *anidb.TitleDB
	dbfile, homepage, refreshfile *fsutil.File
}

func (fs *Mediafs) root() (d *fsutil.Dir) {
	d = fsutil.CreateDir("/")
	fs.RLock()
	for _, s := range fs.db.Anime {
		fi, _ := fsutil.CreateDir(s.Name).Stat()
		d.Append(fi)
	}
	fs.RUnlock()
	return
}

func (fs *Mediafs) child(root *fsutil.Dir, file string) (*fsutil.Dir, *anidb.Anime, error) {
	parts := strings.Split(file, "/")
	plen := len(parts)
	if plen < 2 || plen > 3 {
		return nil, nil, os.ErrNotExist
	}
	log.Println(parts[1])
	child, err := root.Find(parts[1])
	if err != nil {
		log.Println("find failed")
		return nil, nil, err
	}
	dir, ok := child.Sys().(*fsutil.Dir)
	if !ok {
		log.Println("cast failed")
		return nil, nil, os.ErrNotExist
	}
	dir = dir.Dup()
	var ani *anidb.Anime
	fs.RLock()
	defer fs.RUnlock()
	for i := range fs.db.Anime {
		if fs.db.Anime[i].Name == child.Name() {
			ani = fs.db.Anime[i]
			for _, p := range fs.db.Anime[i].Path {
				fi, err := os.Stat(p)
				if err != nil {
					log.Println("stat failed")
					return nil, nil, err
				}
				if fi.IsDir() {
					files, err := ioutil.ReadDir(p)
					if err != nil {
						log.Println("readdir failed")
						return nil, nil, err
					}
					dir.Append(files...)
				} else {
					dir.Append(fi)
				}
			}
		}
	}
	return dir, ani, nil
}

func FindUnion(name string, paths []string) (filepath string, siblings []os.FileInfo, err error) {
	var (
		fi       os.FileInfo
		contents []os.FileInfo
	)
	for _, p := range paths {
		fi, err = os.Stat(p)
		if err != nil {
			return
		}
		if fi.IsDir() {
			contents, err = ioutil.ReadDir(p)
			if err != nil {
				return
			}
			siblings = append(siblings, contents...)
			for _, fi := range contents {
				if fi.Name() == name {
					filepath = path.Join(p, fi.Name())
				}
			}
		} else {
			siblings = append(siblings, fi)
			if fi.Name() == name {
				filepath = path.Join(p, fi.Name())
			}
		}
	}
	return
}

func (fs *Mediafs) Stat(file string) (os.FileInfo, error) {
	var r *fsutil.Dir
	switch file {
	case "/index.html":
		return fs.homepage.Stat()
	case "/db":
		return fs.dbfile.Stat()
	case "/refresh":
		return fs.refreshfile.Stat()
	case "/":
		r = fs.root()
		return r.Stat()
	default:
		r = fs.root()
		c, _, err := fs.child(r, file)
		if err != nil {
			return nil, err
		}
		if strings.Count(file, "/") < 2 {
			return c.Stat()
		}
		_, file = path.Split(file)
		return c.Find(file)
	}
}

func (fs *Mediafs) ReadDir(path string) (ffs.Dir, error) {
	r := fs.root()
	if path == "/" {
		return r, nil
	}
	c, _, err := fs.child(r, path)
	return c, err
}

func (fs *Mediafs) Open(file string, mode int) (interface{}, error) {
	switch file {
	case "/index.html":
		return fs.homepage.Dup(), nil
	case "/refresh":
		err := fs.Update()
		return fs.refreshfile.Dup(), err
	case "/db":
		return fs.dbfile.Dup(), nil
	default:
		_, ani, err := fs.child(fs.root(), file)
		if err != nil {
			return nil, err
		}
		_, name := path.Split(file)
		if file, _, err := FindUnion(name, ani.Path); err == nil {
			return os.OpenFile(file, os.O_RDONLY, 0755)
		} else {
			return nil, err
		}
	}
}

func (fs *Mediafs) genHomepage() (err error) {
	fs.Lock()
	defer fs.Unlock()
	t := template.New("homepage")
	t.Funcs(template.FuncMap{
		"files": func(paths []string) []os.FileInfo {
			if _, fi, err := FindUnion("", paths); err == nil {
				return fi
			}
			return nil
		}})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	fs.homepage.Truncate(1)
	fs.homepage.Seek(0, io.SeekStart)
	err = t.ExecuteTemplate(fs.homepage, "homepage", fs.db)
	return
}

func (fs *Mediafs) Update() (err error) {
	b, err := ioutil.ReadAll(fs.dbfile)
	if err != nil {
		return
	}
	fs.dbfile.Seek(0, io.SeekStart)
	fs.Lock()
	err = json.Unmarshal(b, fs.db)
	fs.Unlock()
	err = fs.genHomepage()
	log.Println("Done updating")
	return
}

func NewMediafs(db io.ReadWriter) (fs *Mediafs, err error) {
	fs = &Mediafs{
		&sync.RWMutex{},
		&anidb.TitleDB{},
		fsutil.CreateFile([]byte(""), 0644, "db"),
		fsutil.CreateFile([]byte(""), 0644, "index.html"),
		fsutil.CreateFile([]byte("Processed refresh"), 0644, "refresh"),
	}
	if db != nil {
		_, err = io.Copy(fs.dbfile, db)
		if err != nil {
			return
		}
		fs.dbfile.Seek(0, io.SeekStart)
		err = fs.Update()
	}
	return
}

const homepagetemplate = `
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
		<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">
		<script src="https://code.jquery.com/jquery-3.3.1.slim.min.js" integrity="sha384-q8i/X+965DzO0rT7abK41JStQIAqVgRVzpbzo5smXKp4YfRvH+8abtTE1Pi6jizo" crossorigin="anonymous"></script>
		<script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.14.7/umd/popper.min.js" integrity="sha384-UO2eT0CpHqdSJQ6hJty5KVphtPhzWj9WO1clHTMGa3JDZwrnQq4sF86dIHNDz0W1" crossorigin="anonymous"></script>
		<script src="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/js/bootstrap.min.js" integrity="sha384-JjSmVgyd0p3pXB1rRibZUAYoIIy6OrQ6VrjIEaFf/nJGzIxFDsf4x0xIM+B07jRM" crossorigin="anonymous"></script>
		<title>mediafs</title>
	</head>
	<body style="background-color:#777777">
			<div class="container card-columns">
			{{range $ani := .Anime}}
				<div class="card" style="background-color:#FFFFEA">
					<img class="card-img-top" src="https://img7-us.anidb.net/pics/anime/{{$ani.Picture}}" alt="{{$ani.Name}}" style="width:%10">
					<div class="card-body">
						<h5 class="card-title">{{$ani.Name}}</h5>
						<p class="card-text">{{$ani.Description}}</p>
						<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}">Episodes</a>
					</div>
				</div>
				<div class="modal fade" id="Modal{{.ID}}" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Label" aria-hidden="true">
					<div class="modal-dialog">
					<div class="modal-content">
					<div class="modal-body">
					<center>
					<div class="list-group">
					{{with files $ani.Path}}
					{{range .}}
					<a href="/{{$ani.Name}}/{{.Name}}" class="list-group-item list-group-item-action">{{.Name}}</a>
					{{end}}
					{{end}}
					<center>
					</div>
					</div>
					</div>
					</div>
				</div>
			{{end}}
			</div>
	</body>
</html>
`
