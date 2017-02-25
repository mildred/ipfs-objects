package ipfs_objects

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"ipobj"
)

type ByteReader interface {
	io.Reader
	Bytes() []byte
}

func readerToBytes(r io.Reader) ([]byte, error) {
	if bytesReader, ok := r.(ByteReader); ok {
		return bytesReader.Bytes(), nil
	} else {
		return ioutil.ReadAll(r)
	}
}

func bytesToReader(b []byte) ByteReader {
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

func encodePeerInfo(i ipobj.PeerInfo) (res pstore.PeerInfo) {
	res.ID = peer.ID(i.Id)
	for _, a := range i.Addrs {
		multiaddr, err := ma.NewMultiaddrBytes([]byte(a))
		if err != nil {
			panic(err)
		}
		res.Addrs = append(res.Addrs, multiaddr)
	}
	return
}

func decodePeerInfo(i pstore.PeerInfo) (res ipobj.PeerInfo) {
	res.Id = []byte(i.ID)
	for _, a := range i.Addrs {
		res.Addrs = append(res.Addrs, a.Bytes())
	}
	return
}
