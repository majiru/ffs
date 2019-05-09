package main

import (
	"aqwari.net/net/styx"
	"errors"
	"github.com/majiru/ffs"
	"github.com/majiru/ffs/fs/domainfs"
	"github.com/majiru/ffs/fs/pastefs"
	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/majiru/ffs/pkg/server"
	"io"
	"log"
	"net/http"
	"os"
)

type chrisfs struct {
	file ffs.Writer
	root *fsutil.Dir
}

func (fs *chrisfs) ReadDir(path string) (ffs.Dir, error) {
	dir := *fs.root
	return &dir, nil
}

func (fs *chrisfs) Open(path string, mode int) (interface{}, error) {
	if mode&os.O_TRUNC != 0 {
		if err := fs.file.Truncate(1); err != nil {
			return nil, errors.New("chrisfs.Open: Truncate Failed")
		}
		fs.file.Seek(0, io.SeekStart)
	}
	return fs.file, nil
}

func (fs *chrisfs) Stat(path string) (os.FileInfo, error) {
	switch path {
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
	m["127.0.0.1"] = &chrisfs{file, dir}
	m["localhost"] = pastefs.NewPastefs()
	dfs := &domainfs.Domainfs{m}

	srv := server.Server{dfs}
	styxServer.Handler = styx.HandlerFunc(srv.Serve9P)
	styxServer.Addr = ":564"
	go http.ListenAndServe(":80", dfs)
	log.Fatal(styxServer.ListenAndServe())
}
