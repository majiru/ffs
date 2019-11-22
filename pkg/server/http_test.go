package server

import (
	"bytes"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/majiru/ffs/fs/ramfs"
	"github.com/majiru/ffs/pkg/fsutil"
)

const m1 = "Hello World"
const m2 = "World Hello"

func testServer() *httptest.Server {
	fs := &ramfs.Ramfs{Root: fsutil.CreateDir("/")}
	fs.Root.Append(fsutil.CreateFile([]byte(m1), 0644, "index.html").Stats)
	return httptest.NewUnstartedServer(Server{fs})
}

func TestGET(t *testing.T) {
	srv := testServer()
	srv.Start()
	c := srv.Client()
	resp, err := c.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatal("error opening index.html:", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("could not read response:", err)
	}
	if string(b) != m1 {
		t.Fatal("Content mismatch")
	}
}

func TestDefaultGET(t *testing.T) {
	srv := testServer()
	srv.Start()
	c := srv.Client()
	resp, err := c.Get(srv.URL + "/")
	if err != nil {
		t.Fatal("error opening index.html:", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("could not read response:", err)
	}
	if string(b) != m1 {
		t.Fatal("Content mismatch")
	}
}

func TestPOST(t *testing.T) {
	srv := testServer()
	srv.Start()
	c := srv.Client()
	submit := strings.NewReader(m2)
	resp, err := c.Post(srv.URL+"/index.html", "text/html; charset=UTF-8", submit)
	if err != nil {
		t.Fatal("Error performing post:", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("error reading post:", err)
	}
	if string(b) != m2 {
		t.Fatalf("content mismatch: saw %s, expected %s", string(b), m2)
	}
}

func TestPut(t *testing.T) {
	srv := testServer()
	srv.Start()
	c := srv.Client()
	c.Timeout = 5 * time.Second
	submit := strings.NewReader(m2)
	req, err := http.NewRequest("PUT", srv.URL+"/index.html", submit)
	if err != nil {
		t.Fatal("could not create req:", err)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal("could not perfrom http request:", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("could not read resp:", err)
	}
	//First request should return original content
	if string(b) != m1 {
		t.Fatal("content mismatch")
	}

	//Second request should return new content
	resp, err = c.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatal("could not perfrom second http request:", err)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("could not read second resp:", err)
	}
	if string(b) != m2 {
		t.Fatal("content mismatch")
	}
}

func TestMultipart(t *testing.T) {
	srv := testServer()
	srv.Start()
	c := srv.Client()
	c.Timeout = 5 * time.Second
	buf := &bytes.Buffer{}
	mp := multipart.NewWriter(buf)

	//BUG: These are currently meaningless
	part, err := mp.CreateFormFile("danny", "bliss")
	if err != nil {
		t.Fatal("creating file:", err)
	}
	part.Write([]byte(m2))
	mp.Close()

	req, err := http.NewRequest("POST", srv.URL+"/index.html", buf)
	if err != nil {
		t.Fatal("creating req:", err)
	}
	req.Header.Set("Content-Type", mp.FormDataContentType())
	_, err = c.Do(req)
	if err != nil {
		t.Fatal("doing req:", err)
	}
	resp, err := c.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatal("doing second req:", err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("reading resp:", err)
	}
	if string(b) != m2 {
		t.Fatal("content mismatch")
	}

	//To test seeking we request the file again
	resp, err = c.Get(srv.URL + "/index.html")
	if err != nil {
		t.Fatal("doing second req:", err)
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal("reading resp:", err)
	}
	if string(b) != m2 {
		t.Fatal("content mismatch")
	}
}
