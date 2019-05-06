package main

import (
	"log"
	"os"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/majiru/ffs/pkg/server"
	"github.com/majiru/ffs/fs/domainfs"
	"aqwari.net/net/styx"
	"github.com/majiru/ffs"
	"net/http"
)


type chrisfs struct {
	file ffs.Writer
	root *fsutil.Dir
}

func (fs chrisfs) ReadDir(path string) (ffs.Dir, error) {
	dir := *fs.root
	return &dir, nil
}

func (fs chrisfs) Read(path string) (interface{}, error) {
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

	m := make(map[string]ffs.Fs)
	m["192.168.0.20"] = &chrisfs{file, dir}
	dfs := &domainfs.Domainfs{m}

	srv := server.Server{ dfs }
	styxServer.Handler = styx.HandlerFunc(srv.Serve9P)
	styxServer.Addr = ":564"
	go http.ListenAndServe(":80", dfs)
	log.Fatal(styxServer.ListenAndServe())
}