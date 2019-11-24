package fsutil

import (
	"bytes"
	"io"
	"os"
	"sync"
	"testing"
)

func eRead(t *testing.T, f *File, b []byte, expect []byte) {
	n, err := f.Read(b)
	if err != nil {
		t.Fatal("err during read:", err)
	}
	if n != len(b) {
		t.Fatal("short read")
	}
	if len(expect) == 0 {
		return
	}
	if bytes.Compare(b, expect) != 0 {
		t.Fatalf("expected %s got %s", string(b), string(expect))
	}
}

func eWrite(t *testing.T, f *File, b []byte) {
	n, err := f.Write(b)
	if err != nil {
		t.Fatal("err during write:", err)
	}
	if n != len(b) {
		t.Fatal("short write")
	}
}

func eReadAt(t *testing.T, f *File, b []byte, whence int64, expect []byte) {
	n, err := f.ReadAt(b, whence)
	if err != nil {
		t.Fatal("err during read:", err)
	}
	if n != len(b) {
		t.Fatal("short read")
	}
	if len(expect) == 0 {
		return
	}
	if bytes.Compare(b, expect) != 0 {
		t.Fatalf("expected %s got %s", string(b), string(expect))
	}
}

func eWriteAt(t *testing.T, f *File, b []byte, whence int64) {
	n, err := f.WriteAt(b, whence)
	if err != nil {
		t.Fatal("err during write:", err)
	}
	if n != len(b) {
		t.Fatal("short write")
	}
}

func TestRead(t *testing.T) {
	buf := make([]byte, 1)
	testStr := []byte("Testing")
	f := CreateFile(testStr, 0644, "test")
	for i := range testStr {
		eRead(t, f, buf, []byte{testStr[i]})
	}
}

func TestReadConcur(t *testing.T) {
	buf := make([]byte, 1)
	testStr := "Testing"
	f := CreateFile([]byte(testStr), 0644, "test")
	for range testStr {
		go eRead(t, f, buf, []byte{})
	}
}

func TestReadDup(t *testing.T) {
	buf := make([]byte, 1)
	testStr := []byte("Testing")
	f := CreateFile(testStr, 0644, "test")
	for range testStr {
		go func() {
			eRead(t, f.Dup(), buf, []byte{testStr[0]})
		}()
	}
}

func TestWrite(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte{}, 0644, "test")
	eWrite(t, f, testStr)
	if f.Size() != int64(len(testStr)) {
		t.Errorf("Size does not properly reflect buffer")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Errorf("Got err %v when seeking to start", err)
	}

	buf := make([]byte, 1)
	for i := range testStr {
		eRead(t, f, buf, []byte{testStr[i]})
	}
}

func TestWriteConcur(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(""), 0644, "test")
	wg := sync.WaitGroup{}
	wg.Add(len(testStr))
	for i := range testStr {
		go func() {
			eWrite(t, f, []byte{testStr[i]})
			wg.Done()
		}()
	}
	wg.Wait()
	if f.Size() != int64(len(testStr)) {
		t.Errorf("Dropped writes, only %d bytes were written, expected %d", f.Size(), len(testStr))
	}
}

func TestReadAt(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile(testStr, 0644, "test")
	wg := sync.WaitGroup{}
	wg.Add(20)
	for i := 0; i < 10; i++ {
		go func() {
			buf := make([]byte, 1)
			eReadAt(t, f, buf, 0, []byte{testStr[0]})
			wg.Done()
		}()
		go func() {
			buf := make([]byte, 1)
			eReadAt(t, f, buf, 1, []byte{testStr[1]})
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestWriteAt(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile(testStr, 0644, "test")
	wg := sync.WaitGroup{}
	wg.Add(len(testStr))
	for i := range testStr {
		go func() {
			eWriteAt(t, f, []byte{testStr[i]}, 1)
			eReadAt(t, f, make([]byte, 1), int64(i), []byte{testStr[i]})
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestTruncate(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(testStr), 0644, "test")

	err := f.Truncate(4)
	if err != nil {
		t.Fatal("Truncate returned error:", err)
	}
	eReadAt(t, f, make([]byte, 4), 0, testStr[:4])
	err = f.Truncate(10)
	if err != nil {
		t.Fatal("Truncate returned error:", err)
	}
	if f.Size() != 10 {
		t.Fatalf("expected %d got %d for Size after Truncate", 10, f.Size())
	}
}

func TestSeekPos(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(testStr), 0644, "test")
	if f.SeekPos() != 0 {
		t.Fatal("SeekPos is not 0 at creation of file")
	}

	seekErr := func(f *File, expect int) {
		if f.SeekPos() != int64(expect) {
			t.Fatalf("expected %d got %d for SeekPos after read", expect, f.SeekPos())
		}
	}

	readSeek := func(f *File, b []byte, expect int) {
		eRead(t, f, b, []byte{})
		seekErr(f, expect)
	}
	b := make([]byte, 3)
	readSeek(f, b, 3)
	readSeek(f.Dup(), b, 6)
	seekErr(f, 3)
}

func TestClose(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(testStr), 0644, "test")
	if f.SeekPos() != 0 {
		t.Fatal("SeekPos is not 0 at creation of file")
	}
	eRead(t, f, make([]byte, 3), testStr[:3])
	f2 := f.Dup()
	err := f.Close()
	if err != nil {
		t.Fatal("Error returned err:", err)
	}
	if f.SeekPos() != 0 {
		t.Fatalf("expected %d got %d for SeekPos after close", 0, f.SeekPos())
	}
	if f2.SeekPos() != 3 {
		t.Fatalf("expected %d got %d for SeekPos after dup", 3, f.SeekPos())
	}
}

func TestStats(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(testStr), 0644, "test")
	s, err := f.Stat()
	if err != nil {
		t.Fatal("Stat returned err:", err)
	}
	if s.Mode() != 0644 {
		t.Fatalf("expected %v got %v for Mode()", os.FileMode(0644), s.Mode())
	}
	if s.IsDir() != false {
		t.Fatalf("expected %t got %t for IsDir()", false, true)
	}
	if s.Size() != int64(len(testStr)) {
		t.Fatalf("expected %d got %d for Size()", s.Size(), len(testStr))
	}
}

func TestSeek(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile(testStr, 0644, "test")
	_, err := f.Seek(-1, io.SeekStart)
	if err != ErrNeg {
		t.Fatalf("expected %v got %v for negative Seek()", ErrNeg, err)
	}
	_, err = f.Seek(0, 100)
	if err != ErrInvalWhence {
		t.Fatalf("expected %v got %v for bad whence in Seek()", ErrInvalWhence, err)
	}
	n, err := f.Seek(1, io.SeekCurrent)
	if n != 1 {
		t.Fatalf("moved %d bytes expected %d", n, 1)
	}
	if err != nil {
		t.Fatal("got error from Seek:", err)
	}
	n, err = f.Seek(-1, io.SeekEnd)
	if n != int64(len(testStr)-1) {
		t.Fatalf("moved %d bytes expected %d", n, len(testStr)-1)
	}
	if err != nil {
		t.Fatal("got error from Seek:", err)
	}
}

func TestReadWriteEof(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile(testStr, 0644, "test")
	ioErrAt := func(expect error, len int, b []byte, whence int64, fun func([]byte, int64) (int, error)) {
		n, err := fun(b, whence)
		if err != expect {
			t.Fatalf("expected %v got %v", expect, err)
		}
		if n != len {
			t.Fatalf("expected %d got %d", len, n)
		}
	}
	ioErr := func(expect error, len int, b []byte, fun func([]byte) (int, error)) {
		n, err := fun(b)
		if err != expect {
			t.Fatalf("expected %v got %v", expect, err)
		}
		if n != len {
			t.Fatalf("expected %d got %d", len, n)
		}
	}
	ioErrAt(ErrNeg, 0, []byte{}, -1, f.WriteAt)
	ioErrAt(ErrNeg, 0, []byte{}, -1, f.ReadAt)

	b := make([]byte, len(testStr)+1)
	ioErrAt(io.EOF, len(testStr), b, 0, f.ReadAt)

	f.Read(make([]byte, len(testStr)))
	ioErr(io.EOF, 0, make([]byte, len(testStr)), f.Read)

	ioErrAt(io.EOF, 0, b, int64(len(testStr)+1), f.ReadAt)
}
