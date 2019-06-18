package mediafs

import (
	"html/template"
	"os"
	"io"
	"path"

	anidb "github.com/majiru/anidb2json"
	"github.com/majiru/ffs"
)

func (fs *Mediafs) genpage(f ffs.Writer, shows []*anidb.Anime) (err error) {
	t := template.New("page")
	t.Funcs(template.FuncMap{
		"files": func(name string) []os.FileInfo {
			if dir, err := fs.Root.WalkForDir(path.Join("/shows", name)); err == nil {
				return dir.Copy()
			}
			return nil
		},
	})
	t, err = t.Parse(homepagetemplate)
	if err != nil {
		return
	}
	f.Truncate(1)
	f.Seek(0, io.SeekStart)
	err = t.ExecuteTemplate(fs.homepage, "page", shows)
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
			{{range $ani := .}}
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
							<a href="/shows/{{$ani.Name}}/{{.Name}}" class="list-group-item list-group-item-action">{{.Name}}</a>
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
