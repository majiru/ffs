package server

import (
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/majiru/ffs"
)

func (srv Server) ReadHTTP(w http.ResponseWriter, r *http.Request, path string) (file ffs.File, err error) {
	file, err = srv.Fs.Open(path, os.O_RDONLY)
	if err != nil {
		if err == os.ErrNotExist {
			log.Printf("fs stat returned %s exists but Open does not\n", path)
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		log.Println("Error: " + err.Error() + " for request " + r.URL.Path)
		http.Error(w, "Internal server error", 500)
		return
	}
	return
}

func (srv Server) WriteHTTP(w http.ResponseWriter, r *http.Request, path string) (content ffs.Writer, err error) {
	file, err := srv.Fs.Open(path, os.O_RDWR|os.O_TRUNC)
	content, ok := file.(ffs.Writer)
	if !ok || err != nil {
		if err == os.ErrNotExist {
			log.Printf("fs stat returned %s exists but Open does not\n", path)
			http.NotFoundHandler().ServeHTTP(w, r)
			return
		}
		log.Println("Error: " + err.Error() + " for request " + r.URL.Path)
		http.Error(w, "Internal server error", 500)
		return
	}
	//As a special case, POST requests that upload
	//a file, instead write the first uploaded file
	//BUG: This drops other form information.
	if mr, err := r.MultipartReader(); err == nil {
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return nil, nil
			}
			if err != nil {
				log.Fatal(err)
			}
			_, err = io.Copy(content, p)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	return
}

func (srv Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedFile := r.URL.Path
	requestedFile = filepath.Join("/", filepath.FromSlash(path.Clean("/"+requestedFile)))
	if requestedFile == "/" {
		requestedFile = "/index.html"
	}
	fi, err := srv.Fs.Stat(requestedFile)
	if err != nil {
		http.NotFoundHandler().ServeHTTP(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		content, err := srv.ReadHTTP(w, r, requestedFile)
		if err == nil && content != nil {
			http.ServeContent(w, r, requestedFile, fi.ModTime(), content)
			content.Close()
		}
	//Post expects the result to depend on what the user writes,
	//so the input is sent to the content file before sending the file to the client
	case http.MethodPost:
		content, err := srv.WriteHTTP(w, r, requestedFile)
		if err == nil && content != nil {
			content.Seek(0, io.SeekStart)
			io.Copy(content, r.Body)
			content.Seek(0, io.SeekStart)
			if err = content.Close(); err != nil {
				log.Fatal(err)
			}
			http.ServeContent(w, r, requestedFile, fi.ModTime(), content)
		}
	//Put expects the input to be reflected in the desired file.
	//In this case the contents of the file are sent to the client before
	//being overwritten.
	case http.MethodPut:
		content, err := srv.WriteHTTP(w, r, requestedFile)
		if err == nil && content != nil {
			http.ServeContent(w, r, requestedFile, fi.ModTime(), content)
			content.Seek(0, io.SeekStart)
			n, _ := io.Copy(content, r.Body)
			content.Truncate(n)
			content.Seek(0, io.SeekStart)
			if err = content.Close(); err != nil {
				log.Fatal(err)
			}
		}
	}
	return
}
