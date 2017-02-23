package ipfs_objects

import (
	"bytes"
	"io"
	"io/ioutil"

	ma "gx/ipfs/QmSWLfmj5frN9xVLMMN846dMDriy5wN5jeghUm7aTW3DAG/go-multiaddr"
	pstore "gx/ipfs/QmQMQ2RUjnaEEX8ybmrhuFFGhAwPjyL1Eo6ZoJGD7aAccM/go-libp2p-peerstore"
	"gx/ipfs/QmZcUPvPhD1Xvk6mwijYF8AfR3mG31S1YsEfHG4khrFPRr/go-libp2p-peer"

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
	res.Id = ipobj.PeerId(i.ID)
	for _, a := range i.Addrs {
		res.Addrs = append(res.Addrs, a.Bytes())
	}
	return
}
