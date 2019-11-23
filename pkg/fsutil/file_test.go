package fsutil

import (
	"io"
	"os"
	"sync"
	"testing"
)

func TestRead(t *testing.T) {
	buf := make([]byte, 1)
	testStr := "Testing"
	f := CreateFile([]byte(testStr), 0644, "test")
	for i := range testStr {
		n, err := f.Read(buf)
		if err != nil {
			t.Errorf("Read returned err: %v", err)
		}
		if n != 1 {
			t.Errorf("Read returned %d for 1 byte read", n)
		}
		if buf[0] != testStr[i] {
			t.Errorf("Read retuned %v, expected %v", buf[0], testStr[i])
		}
	}
}

func TestReadConcur(t *testing.T) {
	buf := make([]byte, 1)
	testStr := "Testing"
	f := CreateFile([]byte(testStr), 0644, "test")
	for range testStr {
		go func() {
			n, err := f.Read(buf)
			if err != nil {
				t.Errorf("Read returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("Read returned %d for 1 byte read", n)
			}
		}()
	}
}

func TestReadDup(t *testing.T) {
	buf := make([]byte, 1)
	testStr := "Testing"
	f := CreateFile([]byte(testStr), 0644, "test")
	for range testStr {
		go func() {
			n, err := f.Dup().Read(buf)
			if err != nil {
				t.Errorf("Read returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("Read returned %d for 1 byte read", n)
			}
			if buf[0] != testStr[0] {
				t.Errorf("Read retuned %v, expected %v", buf[0], testStr[0])
			}
		}()
	}
}

func TestWrite(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(""), 0644, "test")
	n, err := f.Write(testStr)
	if err != nil {
		t.Errorf("Write returned err: %v", err)
	}
	if n != len(testStr) {
		t.Errorf("Did write all of buffer")
	}
	if f.Size() != int64(len(testStr)) {
		t.Errorf("Size does not properly reflect buffer")
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		t.Errorf("Got err %v when seeking to start", err)
	}

	buf := make([]byte, 1)
	for i := range testStr {
		n, err := f.Read(buf)
		if err != nil {
			t.Errorf("Read returned err: %v", err)
		}
		if n != 1 {
			t.Errorf("Read returned %d for 1 byte read", n)
		}
		if buf[0] != testStr[i] {
			t.Errorf("Read retuned %v, expected %v", buf[0], testStr[i])
		}
	}
}

func TestWriteConcur(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(""), 0644, "test")
	wg := sync.WaitGroup{}
	wg.Add(len(testStr))
	for i := range testStr {
		go func() {
			n, err := f.Write([]byte{testStr[i]})
			if err != nil {
				t.Errorf("Write returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("Write returned %d written for 1 byte write", n)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	if f.Size() != int64(len(testStr)) {
		t.Errorf("Dropped writes, only %d bytes were written, expected %d", f.Size(), len(testStr))
	}
}

func TestReadAt(t *testing.T) {
	testStr := "Testing"
	f := CreateFile([]byte(testStr), 0644, "test")
	wg := sync.WaitGroup{}
	wg.Add(20)
	for i := 0; i < 10; i++ {
		go func() {
			buf := make([]byte, 1)
			n, err := f.ReadAt(buf, 0)
			if err != nil {
				t.Errorf("ReadAt returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("ReadAt returned %d for 1 byte read", n)
			}
			if string(buf)[0] != testStr[0] {
				t.Errorf("ReadAt did not grab the right content. Expected %c, got %c", string(buf)[0], testStr[0])
			}
			wg.Done()
		}()
		go func() {
			buf := make([]byte, 1)
			df := f.Dup()
			if _, err := df.Seek(1, io.SeekStart); err != nil {
				t.Errorf("Seek returned err: %v", err)
			}
			n, err := f.ReadAt(buf, 1)
			if err != nil {
				t.Errorf("ReadAt returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("ReadAt returned %d for 1 byte read", n)
			}
			if string(buf)[0] != testStr[1] {
				t.Errorf("ReadAt did not grab the right content. Expected %v, got %v", string(buf)[0], testStr[1])
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestWriteAt(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(testStr), 0644, "test")
	wg := sync.WaitGroup{}
	wg.Add(len(testStr))
	for i := range testStr {
		go func() {
			n, err := f.WriteAt([]byte{testStr[i]}, 1)
			if err != nil {
				t.Errorf("Write returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("Write returned %d written for 1 byte write", n)
			}
			buf := make([]byte, 1)
			n, err = f.ReadAt(buf, int64(i))
			if err != nil {
				t.Errorf("ReadAt returned err: %v", err)
			}
			if n != 1 {
				t.Errorf("ReadAt returned %d for 1 byte read", n)
			}
			if string(buf)[0] != testStr[i] {
				t.Errorf("ReadAt did not grab the right content. Expected %v, got %v", string(buf)[0], testStr[i])
			}
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
	b := make([]byte, 4)
	n, err := f.ReadAt(b, 0)
	if n != 4 {
		t.Fatal("Truncate: short read")
	}
	if string(b) != "Testing"[:4] {
		t.Fatal("Truncate: content mismatch")
	}
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
	b := make([]byte, 3)
	n, err := f.Read(b)
	if err != nil {
		t.Fatal("Read returned err:", err)
	}
	if n != 3 {
		t.Fatal("short read")
	}
	if f.SeekPos() != 3 {
		t.Fatalf("expected %d got %d for SeekPos after read", 3, f.SeekPos())
	}
	f2 := f.Dup()
	n, err = f2.Read(b)
	if err != nil {
		t.Fatal("Read returned err:", err)
	}
	if n != 3 {
		t.Fatal("short read")
	}
	if f2.SeekPos() != 6 {
		t.Fatalf("expected %d got %d for SeekPos after read", 6, f.SeekPos())
	}
	if f.SeekPos() != 3 {
		t.Fatalf("expected %d got %d for SeekPos after read", 3, f.SeekPos())
	}
}

func TestClose(t *testing.T) {
	testStr := []byte("Testing")
	f := CreateFile([]byte(testStr), 0644, "test")
	if f.SeekPos() != 0 {
		t.Fatal("SeekPos is not 0 at creation of file")
	}
	b := make([]byte, 3)
	n, err := f.Read(b)
	if err != nil {
		t.Fatal("Read returned err:", err)
	}
	if n != 3 {
		t.Fatal("short read")
	}
	f2 := f.Dup()
	err = f.Close()
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
	n, err := f.WriteAt([]byte{}, -1)
	if err != ErrNeg {
		t.Fatalf("expected %v got %v for negative WriteAt()", ErrNeg, err)
	}
	if n != 0 {
		t.Fatalf("expected %d got %d for negative WriteAt()", 0, n)
	}
	n, err = f.ReadAt([]byte{}, -1)
	if err != ErrNeg {
		t.Fatalf("expected %v got %v for negative ReadAt()", ErrNeg, err)
	}
	if n != 0 {
		t.Fatalf("expected %d got %d for negative ReadAt()", 0, n)
	}
	b := make([]byte, len(testStr)+1)
	n, err = f.ReadAt(b, 0)
	if err != io.EOF {
		t.Fatalf("expected %v got %v for EOF ReadAt()", io.EOF, err)
	}
	if n != len(testStr) {
		t.Fatalf("expected %d got %d for EOF ReadAt()", len(testStr), n)
	}
	f.Read(make([]byte, len(testStr)))
	n, err = f.Read(b)
	if err != io.EOF {
		t.Fatalf("expected %v got %v for EOF Read()", io.EOF, err)
	}
	if n != 0 {
		t.Fatalf("expected %d got %d for EOF Read()", 0, n)
	}
	n, err = f.ReadAt(b, int64(len(testStr)+1))
	if err != io.EOF {
		t.Fatalf("expected %v got %v for EOF ReadAt()", io.EOF, err)
	}
	if n != 0 {
		t.Fatalf("expected %d got %d for EOF ReadAt()", 0, n)
	}
}
