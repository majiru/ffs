package main

import (
	"os"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/majiru/ffs/pkg/server"
	"aqwari.net/net/styx"
	"github.com/majiru/ffs"
	"net/http"
)


type chrisfs struct {
	root fsutil.File
}

func (fs chrisfs) ReadDir(path string) ([]os.FileInfo, error) {
	return []os.FileInfo{fs.root.Stat("index.html")}, nil
}

func (fs chrisfs) Read(path string) (ffs.File, error) {
	return fs.root, nil
}

func (fs chrisfs) Stat(path string) (os.FileInfo, error) {
	return fs.root.Stat("index.html"), nil
}

func main() {
	fs := chrisfs{fsutil.CreateFile([]byte("Hello World!\n"), 0777)}
	srv := server.Server{ fs }
	go http.ListenAndServe(":80", srv)
	styx.ListenAndServe(":564", styx.HandlerFunc(srv.Serve9P))
}