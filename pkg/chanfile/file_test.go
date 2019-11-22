package chanfile

import (
	"io"
	"testing"
)

func basicfileproc(f *File, n int) {
	for i := 0; i < n; i++ {
		m := <-f.Req
		switch m.Type {
		case Read:
			f.Recv <- RecvMsg{Commit, nil}
		case Write:
			f.Recv <- RecvMsg{Commit, nil}
		case Trunc:
			f.Recv <- RecvMsg{Commit, nil}
		case Close:
			f.Recv <- RecvMsg{Commit, nil}
		}
	}
}

var m1 = []byte("Hello World")
var m2 = []byte("Hello")
var m3 = []byte("World")

func TestRead(t *testing.T) {
	f := CreateFile(m1, 0644, "test")
	b := make([]byte, len(m1))
	go basicfileproc(f, 1)
	n, err := f.Read(b)
	if err != nil {
		t.Fatal("error reading chanfile:", err)
	}
	if n != len(m1) {
		t.Fatal("short read")
	}
	if string(m1) != string(b) {
		t.Fatal("content mismatch")
	}
}

func TestReadSeek(t *testing.T) {
	f := CreateFile(append(m2, m3...), 0644, "test")
	go basicfileproc(f, 2)
	b := make([]byte, len(m2))
	n, err := f.Read(b)
	if err != nil {
		t.Fatal("error reading chanfile:", err)
	}
	if n != len(m2) {
		t.Fatal("short read")
	}
	if string(m2) != string(b) {
		t.Fatal("content mismatch")
	}
	b = make([]byte, len(m3))
	n, err = f.Read(b)
	if err != nil {
		t.Fatal("error reading chanfile:", err)
	}
	if n != len(m3) {
		t.Fatal("short read")
	}
	if string(m3) != string(b) {
		t.Fatal("content mismatch")
	}
}

func TestWrite(t *testing.T) {
	f := CreateFile([]byte{}, 0644, "test")
	go basicfileproc(f, 1)
	n, err := f.Write(m1)
	if err != nil {
		t.Fatal("error reading chanfile:", err)
	}
	if n != len(m1) {
		t.Fatal("short read")
	}
}

func TestReadWrite(t *testing.T) {
	f := CreateFile([]byte{}, 0644, "test")
	go basicfileproc(f, 2)
	n, err := f.Write(m1)
	if err != nil {
		t.Fatal("error reading chanfile:", err)
	}
	if n != len(m1) {
		t.Fatal("short read")
	}
	f.Seek(0, io.SeekStart)
	b := make([]byte, len(m1))
	n, err = f.Read(b)
	if err != nil {
		t.Fatal("error reading chanfile:", err)
	}
	if n != len(m1) {
		t.Fatal("short read")
	}
	if string(m1) != string(b) {
		t.Fatal("content mismatch")
	}
}