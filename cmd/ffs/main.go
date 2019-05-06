package main

import (
	"log"
	"os"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/majiru/ffs/pkg/server"
	"aqwari.net/net/styx"
	"github.com/majiru/ffs"
	"net/http"
)


type chrisfs struct {
	file *fsutil.File
	root *fsutil.Dir
}

func (fs chrisfs) ReadDir(path string) (ffs.Dir, error) {
	dir := *fs.root
	return &dir, nil
}

func (fs chrisfs) Read(path string) (ffs.File, error) {
	return fs.file, nil
}

func (fs chrisfs) Stat(path string) (os.FileInfo, error) {
	switch(path){
	case "/":
		return fs.root.Stat()
	case "/index.html":
		return fs.file.Stat()
	}

	log.Println(path, " not found")
	return nil, os.ErrNotExist
}

func main() {
	var styxServer styx.Server
	styxServer.TraceLog = log.New(os.Stderr, "", 0)
	styxServer.ErrorLog = log.New(os.Stderr, "", 0)
	file := fsutil.CreateFile([]byte("Hello World!\n"), 0777, "index.html")
	fi, _ := file.Stat()
	dir := fsutil.CreateDir("/", fi)
	srv := server.Server{ &chrisfs{file, dir} }
	styxServer.Handler = styx.HandlerFunc(srv.Serve9P)
	styxServer.Addr = ":564"
	go http.ListenAndServe(":80", srv)
	log.Fatal(styxServer.ListenAndServe())
}