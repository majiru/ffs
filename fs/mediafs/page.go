package mediafs

import (
	"html/template"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/pkg/fsutil"
)

const itemsPerPage = 20

type Page struct {
	Shows            []*anidb.Anime
	Prev, Next       int
	CanPrev, CanNext bool
}

func (fs *Mediafs) readShowDir(name string) []os.FileInfo {
	if dir, err := fs.Root.WalkForDir(path.Join("/shows", name)); err == nil {
		return dir.Copy()
	}
	return nil
}

//Geneate a page with all given shows from the slice
func (fs *Mediafs) genpage(f ffs.Writer, shows []*anidb.Anime) (err error) {
	t := template.New("page")
	t.Funcs(template.FuncMap{"files": fs.readShowDir})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	f.Truncate(1)
	f.Seek(0, io.SeekStart)
	err = t.ExecuteTemplate(f, "page", Page{shows, 0, 0, false, false})
	return
}

//Generate a window into the shows slice, calculated based on pagenum
func (fs *Mediafs) genwindow(f ffs.Writer, shows []*anidb.Anime, pagenum int) (err error) {
	t := template.New("page")
	t.Funcs(template.FuncMap{"files": fs.readShowDir})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	f.Truncate(1)
	f.Seek(0, io.SeekStart)
	page := Page{Prev: pagenum - 1, Next: pagenum + 1}
	if (pagenum+1)*itemsPerPage > len(shows) {
		page.CanNext = false
		page.Shows = shows[pagenum*itemsPerPage:]
	} else {
		page.CanNext = true
		page.Shows = shows[pagenum*itemsPerPage : (pagenum+1)*itemsPerPage]
	}
	if pagenum == 0 {
		page.CanPrev = false
	} else {
		page.CanPrev = true
	}
	err = t.ExecuteTemplate(f, "page", page)
	return
}

func (fs *Mediafs) handlePagination(req string, shows []*anidb.Anime) (ffs.Writer, error) {
	_, file := path.Split(req)
	if !strings.HasSuffix(file, ".html") {
		return nil, os.ErrNotExist
	}
	pagenum, err := strconv.Atoi(strings.Trim(file, ".html"))
	if err != nil {
		return nil, err
	}
	if pagenum < 0 {
		return nil, os.ErrNotExist
	}
	//If our current page doesn't have any items in it, we don't exist
	if pagenum*itemsPerPage > len(shows) {
		return nil, os.ErrNotExist
	}
	content := fsutil.CreateFile([]byte(""), 0644, file)
	err = fs.genwindow(content, shows, pagenum)
	content.Seek(0, io.SeekStart)
	return content.Dup(), err
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
		<div class="container">
			<form action="/search" method="post">
				<div class="form-group">
					<input name="search" class="form-control" id="search" placeholder="keyword">
				</div>
				<button type="submit" class="btn btn-primary">Search!</button>
			</form>
			<div>
			<ul class="pagination justify-content-center">
					{{- if .CanPrev -}}
						<li class="page-item">
					{{- else -}}
						<li class="page-item disabled">
					{{- end -}}
					<a class="page-link" href="/page/{{.Prev}}.html">Previous</a></li>
					{{- if .CanNext -}}
						<li class="page-item">
					{{- else -}}
						<li class="page-item disabled">
					{{- end -}}
					<a class="page-link" href="/page/{{.Next}}.html">Next</a></li>
				</ul>
			</div>
		<div class="card-columns">
		{{range $ani := .Shows}}
			<div class="card" style="background-color:#FFFFEA">
				<img class="card-img-top" src="https://img7-us.anidb.net/pics/anime/{{$ani.Picture}}" alt="{{$ani.Name}}" style="width:%10">
				<div class="card-body">
					<h5 class="card-title">{{$ani.Name}}</h5>
					<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}">Episodes</a>
					<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}Desc">Synopsis</a>
					<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}Staff">Staff</a>
					<a href="#" class="btn btn-primary" data-toggle="modal" data-target="#Modal{{$ani.ID}}Tags">Tags</a>
				</div>
			</div>
			<div class="modal fade" id="Modal{{.ID}}Desc" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Desc" aria-hidden="true">
				<div class="modal-dialog modal-content modal-body">
				<center>
				<p>{{$ani.Description}}</p>
				</center>
				</div>
			</div>
			<div class="modal fade" id="Modal{{.ID}}Staff" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Staff" aria-hidden="true">
				<div class="modal-dialog modal-content modal-body">
				<center>
					<div class="list-group">
						{{- range .Creators -}}
						<a href="/staff/{{.Name}}" class="list-group-item list-group-item-action">{{.Name}}, {{.Role}}</a>
						{{- end -}}
					</div>
				</center>
				</div>
			</div>
			<div class="modal fade" id="Modal{{.ID}}Tags" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Tags" aria-hidden="true">
				<div class="modal-dialog modal-content modal-body">
				<center>
					<div class="list-group">
						{{- range .Tags -}}
						<a href="/tags/{{.Name}}" class="list-group-item list-group-item-action">{{.Name}}</a>
						{{- end -}}
					</div>
				</center>
				</div>
			</div>
			<div class="modal fade" id="Modal{{.ID}}" tabindex="-1" role="dialog" aria-labelledby="Modal{{.ID}}Label" aria-hidden="true">
				<div class="modal-dialog modal-content modal-body">
					<center>
					<div class="list-group">
						{{- with files $ani.Name -}}
						{{- range . -}}
						<a href="/shows/{{$ani.Name}}/{{.Name}}" class="list-group-item list-group-item-action">{{.Name}}</a>
						{{- end -}}
						{{- end -}}
					</div>
					</center>
				</div>
			</div>
		{{end}}
		</div>
		<div>
			<ul class="pagination justify-content-center">
				{{- if .CanPrev -}}
					<li class="page-item">
				{{- else -}}
					<li class="page-item disabled">
				{{- end -}}
				<a class="page-link" href="/page/{{.Prev}}.html">Previous</a></li>
				{{- if .CanNext -}}
					<li class="page-item">
				{{- else -}}
					<li class="page-item disabled">
				{{- end -}}
				<a class="page-link" href="/page/{{.Next}}.html">Next</a></li>
			</ul>
		</div>
		</div>
	</body>
</html>
`
