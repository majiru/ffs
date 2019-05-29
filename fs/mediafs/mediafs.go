package mediafs

import (
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Mediafs struct {
	*sync.RWMutex
	Root             *fsutil.Dir
	DB               *anidb.TitleDB
	dbfile, homepage *fsutil.File
}

func NewMediafs(db io.ReadWriter) (fs *Mediafs, err error) {
	fs = &Mediafs{
		&sync.RWMutex{},
		fsutil.CreateDir("/"),
		&anidb.TitleDB{},
		fsutil.CreateFile([]byte(""), 0644, "db"),
		fsutil.CreateFile([]byte(""), 0644, "index.html"),
	}
	if db != nil {
		_, err = io.Copy(fs.dbfile, db)
		if err != nil {
			return
		}
		fs.dbfile.Seek(0, io.SeekStart)
		err = fs.update()
	}
	return
}

func (fs *Mediafs) updateTree() {
	fs.Lock()
	fs.Root = fsutil.CreateDir("/")
	for _, s := range fs.DB.Anime {
		subdir := fsutil.CreateDir(s.Name)
		for _, p := range s.Path {
			subdir.Append(fsutil.CreateFile([]byte(p), 0644, path.Base(p)).Stats)
		}
		fs.Root.Append(subdir.Stats)
	}
	fs.Unlock()
}

func (fs *Mediafs) updateHomepage() (err error) {
	fs.Lock()
	defer fs.Unlock()
	t := template.New("homepage")
	t.Funcs(template.FuncMap{
		"files": func(name string) []os.FileInfo {
			if dir, err := fs.Root.WalkForDir(name); err == nil {
				return dir.Copy()
			}
			return nil
		},
	})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	fs.homepage.Truncate(1)
	fs.homepage.Seek(0, io.SeekStart)
	err = t.ExecuteTemplate(fs.homepage, "homepage", fs)
	return
}

func (fs *Mediafs) updateDB() (err error) {
	fs.Lock()
	b, err := ioutil.ReadAll(fs.dbfile)
	if err != nil {
		return
	}
	fs.dbfile.Seek(0, io.SeekStart)
	err = json.Unmarshal(b, fs.DB)
	fs.Unlock()
	return
}

func (fs *Mediafs) update() (err error) {
	if err = fs.updateDB(); err != nil {
		return
	}
	fs.updateTree()
	return fs.updateHomepage()
}

func (fs *Mediafs) Check() (err error) {
	if fs.dbfile.Stats.ModTime().After(fs.homepage.Stats.ModTime()) {
		return fs.update()
	}
	return
}

func (fs *Mediafs) Stat(file string) (os.FileInfo, error) {
	if err := fs.Check(); err != nil {
		log.Println(err)
		return nil, err
	}
	switch file {
	case "/index.html":
		return fs.homepage.Stat()
	case "/db":
		return fs.dbfile.Stat()
	case "/":
		return fs.Root.Stat()
	default:
		return fs.Root.Walk(file)
	}
}

func (fs *Mediafs) ReadDir(path string) (ffs.Dir, error) {
	if err := fs.Check(); err != nil {
		log.Println(err)
		return nil, err
	}
	switch path {
	case "/":
		return fs.Root.Dup(), nil
	default:
		return fs.Root.WalkForDir(path)
	}
}

func (fs *Mediafs) Open(file string, mode int) (interface{}, error) {
	if err := fs.Check(); err != nil {
		log.Println(err)
		return nil, err
	}
	switch file {
	case "/index.html":
		return fs.homepage.Dup(), nil
	case "/db":
		return fs.dbfile.Dup(), nil
	default:
		if f, err := fs.Root.WalkForFile(file); err != nil {
			return nil, err
		} else {
			//These files store the absolute path, not the file contents
			f.Seek(0, io.SeekStart)
			if b, err := ioutil.ReadAll(f); err != nil {
				return nil, err
			} else {
				return os.OpenFile(string(b), os.O_RDONLY, 0555)
			}
		}
	}
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
			{{range $ani := .DB.Anime}}
				<div class="card" style="background-color:#FFFFEA">
					<img class="card-img-top" src="https://img7-us.anidb.net/pics/anime/{{$ani.Picture}}" alt="{{$ani.Name}}" style="width:%10">
					<div class="card-body">
						<h5 class="card-title">{{$ani.Name}}</h5>
						<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}">Episodes</a>
						<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}Desc">Synopsis</a>
					</div>
				</div>
				<div class="modal fade" id="Modal{{.ID}}Desc" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Desc" aria-hidden="true">
					<div class="modal-dialog modal-content modal-body">
					<center>
					<p>{{$ani.Description}}</p>
					</center>
					</div>
				</div>
				<div class="modal fade" id="Modal{{.ID}}" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Label" aria-hidden="true">
					<div class="modal-dialog modal-content modal-body">
						<center>
						<div class="list-group">
							{{- with files $ani.Name -}}
							{{- range . -}}
							<a href="/{{$ani.Name}}/{{.Name}}" class="list-group-item list-group-item-action">{{.Name}}</a>
							{{- end -}}
							{{- end -}}
						</div>
						</center>
					</div>
				</div>
			{{end}}
			</div>
	</body>
</html>
`
