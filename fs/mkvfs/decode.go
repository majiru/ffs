package mkvfs

import (
	"encoding/binary"
	"io"
	"fmt"
	"math"
	"os"
	"strings"
	"strconv"

	"github.com/majiru/ffs/pkg/chanfile"
)

type Decoder struct {
	EBML *chanfile.File
	Block *chanfile.File
	f *os.File
}

func NewDecoder() *Decoder {
	d := &Decoder{
		chanfile.CreateFile([]byte{}, 0644, "EBML"),
		chanfile.CreateFile([]byte{}, 0644, "Block"),
		nil,
	}
	go d.blockproc()
	return d
}

func (d *Decoder) decodeBlock(req string) error {
	var (
		tmp []byte
		err error
	)
	parts := strings.Split(req, " ")
	if len(parts) != 2 {
		fmt.Errorf("Usage:offset length")
	}
	offset, err := strconv.Atoi(parts[0])
	if err != nil {
		return err
	}
	length, err := strconv.Atoi(parts[1])
	if err != nil {
		return err
	}
	
	if _, err = d.f.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	if err = d.Block.Content.Truncate(1); err != nil {
		return err
	}
	if _, err = d.Block.Seek(0, io.SeekStart); err != nil {
		return err
	}
	trackNum, n, err := decodeEBML(d.f)
	length = length - n
	if err != nil {
		return err
	}
	if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("Block Header:\n\tTrack number: %d\n", trackNum))); err != nil {
		return err
	}
	tmp = make([]byte, 2)
	if _, err = d.f.Read(tmp); err != nil {
		return err
	}
	length = length - 2
	relativeOffset := binary.BigEndian.Uint16(tmp)
	if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\tRelative Offset: %d\n", relativeOffset))); err != nil {
		return err
	}
	if _, err = d.Block.Content.Write([]byte("\tFlags:\n")); err != nil {
		return err
	}
	var (
		invis bool
		lacing string
	)
	tmp = make([]byte, 1)
	if _, err = d.f.Read(tmp); err != nil {
		return err
	}
	length = length - 1
	if tmp[0] & 0x10 == 0x10 {
		invis = true
	}
	switch f := tmp[0] & 0x6; f {
	case 0:
		lacing = "No Lacing"
	case 0x2:
		lacing = "XIPH Lacing"
	case 0x6:
		lacing = "EBML Lacing"
	case 0x4:
		lacing = "Fixed-size Lacing"
	}
	if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\t\tInvisable: %t\n\t\tLacing: %s\n", invis, lacing))); err != nil {
		return err
	}
	if _, err = d.f.Read(tmp); err != nil {
		return err
	}
	length = length - 1
	numFrames := uint8(tmp[0])+1
	if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\tNumber of frames: %d, Length: %d\n", numFrames, length))); err != nil {
		return err
	}
	if lacing == "EBML Lacing" {
		frameSize, _, err := decodeEBML(d.f)
		if err != nil {
			return err
		}
		if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\tFrames:\n\t\tFrame 1: %d Bytes\n", frameSize))); err != nil {
			return err
		}
		length = int(int64(length) - frameSize)
		var i uint8
		for i = uint8(2); i < numFrames; i++ {
			diff, _, err := decodeEBMLSigned(d.f)
			if err != nil {
				return err
			}
			frameSize = frameSize + diff
			length = int(int64(length) - frameSize)
			if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\t\tFrame %d: %d Bytes\n", i, frameSize))); err != nil {
				return err
			}
		}
		if _, err = d.Block.Content.Write([]byte(fmt.Sprintf("\t\tFrame %d: %d Bytes\n", i, length))); err != nil {
				return err
		}
	}
	return nil
}

func decodeEBMLSigned(f io.ReadSeeker) (x int64, n int, err error) {
	ux, n, err := decodeEBML(f)
	if err != nil {
		return
	}
	switch n {
	case 1:
		x = ux - int64(math.Pow(2, 6)) + 1
	case 2:
		x = ux - int64(math.Pow(2, 13)) + 1
	case 3:
		x = ux - int64(math.Pow(2, 20)) + 1
	case 4:
		x = ux - int64(math.Pow(2, 27)) + 1
	case 5:
		x = ux - int64(math.Pow(2, 34)) + 1
	case 6:
		x = ux - int64(math.Pow(2, 41)) + 1
	case 7:
		x = ux - int64(math.Pow(2, 48)) + 1
	case 8:
		x = ux - int64(math.Pow(2, 56)) + 1
	}
	return
}

func decodeEBML(f io.ReadSeeker) (int64, int, error) {
	tmp := make([]byte, 1)
	if _, err := f.Read(tmp); err != nil {
		return 0, 0, err
	}
	var size int
	var mask byte
	switch {
	case tmp[0] & 0x80 == 0x80:
		size = 1
		mask = 0x7f
	case tmp[0] & 0x40 == 0x40:
		size = 2
		mask = 0x3f
	case tmp[0] & 0x20 == 0x20:
		size = 3
		mask = 0x1f
	case tmp[0] & 0x10 == 0x10:
		size = 4
		mask = 0xf
	case tmp[0] & 0x8 == 0x8:
		size = 5
		mask = 0x7
	case tmp[0] & 0x4 == 0x4:
		size = 6
		mask = 0x3
	case tmp[0] & 0x2 == 0x2:
		size = 7
		mask = 0x1
	case tmp[0] & 0x1 == 0x1:
		size = 8
		mask = 0
	default:
		return 0, 0, fmt.Errorf("Unexpected first byte %q", tmp)
	}
	result := make([]byte, 8)
	result[8-size] = tmp[0] & mask
	if _, err := f.Read(result[8-size+1:]); err != nil {
		return 0, 0, err
	}
	return int64(binary.BigEndian.Uint64(result)), size, nil
}

func (d *Decoder) blockproc() {
	for {
		switch m := <- d.Block.Req; m.Type {
		case chanfile.Read:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Write:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, d.decodeBlock(string(m.Content))}
		case chanfile.Trunc:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		case chanfile.Close:
			d.Block.Recv <- chanfile.RecvMsg{chanfile.Commit, nil}
		}
	}
}