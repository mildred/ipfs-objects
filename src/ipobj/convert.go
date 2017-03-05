package ipobj

import (
	"bytes"
	"io"
	"io/ioutil"
)

type ByteReader interface {
	io.Reader
	Bytes() []byte
}

func ReaderToBytes(r io.Reader) ([]byte, error) {
	if bytesReader, ok := r.(ByteReader); ok {
		return bytesReader.Bytes(), nil
	} else {
		return ioutil.ReadAll(r)
	}
}

func BytesToReader(b []byte) ByteReader {
	return &simpleByteReader{bytes.NewReader(b), b}
}

type simpleByteReader struct {
	*bytes.Reader
	b []byte
}

func (sbr *simpleByteReader) Bytes() []byte {
	pos, _ := sbr.Seek(0, io.SeekCurrent)
	if pos == 0 {
		return sbr.b
	} else {
		return sbr.b[pos:]
	}
}
