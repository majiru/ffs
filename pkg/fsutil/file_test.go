package fsutil

import (
	"io"
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