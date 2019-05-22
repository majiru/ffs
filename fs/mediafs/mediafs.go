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

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

type Mediafs struct {
	db *anidb.TitleDB
}

func (fs *Mediafs) root() (d *fsutil.Dir) {
	d = fsutil.CreateDir("/")
	for _, s := range fs.db.Anime {
		fi, _ := fsutil.CreateDir(s.Name).Stat()
		d.Append(fi)
	}
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

func (fs *Mediafs) Stat(file string) (os.FileInfo, error) {
	if strings.HasSuffix(file, "index.html") {
		f := fsutil.CreateFile([]byte(""), 0644, "/index.html")
		return f.Stat()
	}
	r := fs.root()
	if file == "/" {
		return r.Stat()
	}
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

func (fs *Mediafs) ReadDir(path string) (ffs.Dir, error) {
	r := fs.root()
	if path == "/" {
		return r, nil
	}
	c, _, err := fs.child(r, path)
	return c, err
}

func findUnion(name string, paths []string) (*os.File, error) {
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		if fi.IsDir() {
			contents, err := ioutil.ReadDir(p)
			if err != nil {
				return nil, err
			}
			for _, fi := range contents {
				if fi.Name() == name {
					return os.Open(path.Join(p, fi.Name()))
				}
			}
		} else {
			if fi.Name() == name {
				return os.Open(p)
			}
		}
	}
	return nil, os.ErrNotExist
}

func (fs *Mediafs) homepage(w io.Writer) (err error) {
	t := template.New("homepage")
	t.Funcs(template.FuncMap{"start": func(i, j int) bool { return i%j == 0 }})
	t.Funcs(template.FuncMap{"stop": func(i, j int) bool { return i%j == (j - 1) }})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	err = t.ExecuteTemplate(w, "homepage", fs.db)
	return
}

func childpage(ani *anidb.Anime, fi []os.FileInfo, w io.Writer) (err error) {
	content := struct {
		Ani   *anidb.Anime
		Files []os.FileInfo
	}{
		ani,
		fi,
	}
	t := template.New("childpage")
	t, err = t.Parse(seriespagetemplate)
	if err != nil {
		return
	}
	err = t.ExecuteTemplate(w, "childpage", &content)
	return
}

func (fs *Mediafs) Open(file string, mode int) (interface{}, error) {
	if file == "/index.html" {
		f := fsutil.CreateFile([]byte(""), 0644, "/index.html")
		err := fs.homepage(f)
		return f, err
	}
	child, ani, err := fs.child(fs.root(), file)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(file, "index.html") {
		fi, err := child.Readdir(0)
		if err != nil {
			return nil, err
		}
		f := fsutil.CreateFile([]byte(""), 0644, "/index.html")
		err = childpage(ani, fi, f)
		return f, err
	}
	_, name := path.Split(file)
	return findUnion(name, ani.Path)
}

func NewMediafs(db io.Reader) (fs *Mediafs, err error) {
	fs = &Mediafs{&anidb.TitleDB{}}
	b, err := ioutil.ReadAll(db)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, fs.db)
	return
}

const homepagetemplate = `
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
		<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">
		<title>mediafs</title>
	</head>
	<body style="background-color:#FFFFEA">
		{{range $index, $s := .Anime}}
		{{if start $index 3}}<div class="row">{{end}}
			<div class="col">
				<div class="thumbnail">
      				<a href="/{{$s.Name}}/index.html">
						<img src="https://img7-us.anidb.net/pics/anime/{{$s.Picture}}">
						<div class="caption" >
							<p>{{$s.Name}}</p>
						</div>
					</a>
				</div>
			</div>
		{{if stop $index 3}}</div>{{end}}
		{{end}}
	</body>
</html>
`

const seriespagetemplate = `
<!doctype html>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
		<link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">
		<title>{{.Ani.Name}}</title>
	</head>
	<body style="background-color:#FFFFEA; color:black">
		<div class="d-flex">
			<div class="p-2 flex-fill"><img src="https://img7-us.anidb.net/pics/anime/{{.Ani.Picture}}"></div>
			<div class="p-2 flex-fill">{{.Ani.Description}}</div>
		</div>
		<div class="d-flex flex-column mb-3">
  			{{range .Files}}
				<a href="{{.Name}}">{{.Name}}</a>
			{{end}}
		</div>
	</body>
</html>
`
