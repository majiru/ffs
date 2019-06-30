package pastefs

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/majiru/ffs/pkg/fsutil"
	"github.com/majiru/ffs/pkg/server"
)

func fetchHomepage(t *testing.T, URL string) []byte {
	homepage := fmt.Sprintf("%s/index.html", URL)
	r, err := http.Get(homepage)
	if err != nil {
		t.Errorf("%q", err)
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("%q", err)
	}
	return b
}

func testHomepage(t *testing.T, files []os.FileInfo, result []byte) {
	var err error
	content := struct{ Files []os.FileInfo }{files}
	template := template.New("homepage")
	if template, err = template.Parse(homepage); err != nil {
		t.Errorf("%q", err)
	}
	outFile := fsutil.CreateFile([]byte{}, 0644, "temp")
	if err = template.ExecuteTemplate(outFile, "homepage", content); err != nil {
		t.Errorf("%q", err)
	}
	outFile.Seek(0, io.SeekStart)
	b, err := ioutil.ReadAll(outFile)
	if err != nil {
		t.Errorf("%q", err)
	}
	if bytes.Compare(b, result) != 0 {
		t.Errorf("mismatch in homepage")
	}
}

//TestStat tests for special files to exist on fs creation.
func TestStat(t *testing.T) {
	fs := NewPastefs()
	if _, err := fs.Stat("/"); err != nil {
		t.Errorf("Could not stat /: %q", err)
	}
	if _, err := fs.Stat("/index.html"); err != nil {
		t.Errorf("Could not stat /index.html: %q", err)
	}
}

//ensurePaste posts a new paste and checks that the tree and content is correctly set
func ensurePaste(t *testing.T, fs *Pastefs, URL string, content string) (pasteName string) {
	new := fmt.Sprintf("%s/new", URL)
	r, err := http.Post(new, "text/plain", strings.NewReader(content))
	if err != nil {
		t.Errorf("Error performing /new POST: %q", err)
	}
	pasteNameByte, err := ioutil.ReadAll(r.Body)
	pasteName = string(pasteNameByte)
	if err != nil {
		t.Errorf("Error reading /new response body: %q", err)
	}
	func() {
		for _, fi := range fs.pastes.Copy() {
			if fi.Name() == pasteName {
				return
			}
		}
		t.Errorf("Did not find new pastefile name %s in fs root", pasteName)
	}()
	reqStr := fmt.Sprintf("%s/pastes/%s", URL, pasteName)
	t.Logf("Requesting: %s", reqStr)
	r, err = http.Get(fmt.Sprintf("%s/pastes/%s", URL, pasteName))
	if err != nil {
		t.Errorf("Error in GETing new paste: %q", err)
	}
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Errorf("Error in reading back new paste: %q", err)
	}
	if content != string(b) {
		t.Errorf("Paste readback differed from write. Got %s, expected %s", string(b), content)
	}
	return
}

func TestPastefs(t *testing.T) {
	fs := NewPastefs()
	ts := httptest.NewServer(server.Server{fs})
	defer ts.Close()

	testHomepage(t, []os.FileInfo{}, fetchHomepage(t, ts.URL))
	paste1 := fsutil.CreateFile([]byte{}, 0644, ensurePaste(t, fs, ts.URL, "hello world")).Stats
	//Pastes are stored based on UNIX time, give it a second to change
	<-time.After(time.Second)
	paste2 := fsutil.CreateFile([]byte{}, 0644, ensurePaste(t, fs, ts.URL, "world hello")).Stats
	testHomepage(t, []os.FileInfo{paste1, paste2}, fetchHomepage(t, ts.URL))
}
